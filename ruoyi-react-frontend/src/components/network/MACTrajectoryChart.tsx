import React, { useMemo } from "react";
// 改用 EChartsWrapper（lazy 包装），让 echarts-for-react 从 vendor-react 移到 lazy chunk
import ReactECharts from "@/components/charts/EChartsWrapper";
import type { EChartsOption } from "echarts";
import { Empty, Spin } from "antd";
import { EVENT_COLORS, type MACEventType } from "./macEventMeta";

export interface TrajectoryNode {
  mac: string;
  deviceId: string;
  deviceName: string;
  interface: string;
  vlanId?: number;
  startTime: string; // ISO timestamp
  endTime: string; // ISO timestamp
  duration: number;
  eventType: "appeared" | "disappeared" | "moved" | "vlan_changed";
  /** 厂商名(由 TrajectoryPage 调用 queryMACVendor 合并填充) */
  vendor?: string;
}

interface MACTrajectoryChartProps {
  data: TrajectoryNode[];
  loading?: boolean;
  onError?: (error: unknown) => void;
  /** dataZoom slider 起始百分比 (0-100, 默认 66 表示聚焦最近 1/3) */
  dataZoomStart?: number;
  /** dataZoom slider 结束百分比 (0-100, 默认 100) */
  dataZoomEnd?: number;
}

// 停留时长热力阈值(秒)— Phase 14-02 锁定
const DWELL_THRESHOLDS = {
  short: 3600, // 1h
  medium: 86400, // 24h
  long: 604800, // 7d
} as const;

function getDwellHeatmap(durationSeconds: number): { label: string; color: string } {
  if (durationSeconds < DWELL_THRESHOLDS.short) return { label: "短", color: "var(--theme-text-tertiary, #bfbfbf)" };
  if (durationSeconds < DWELL_THRESHOLDS.medium) return { label: "中", color: "var(--theme-info, #1890ff)" };
  if (durationSeconds < DWELL_THRESHOLDS.long) return { label: "长", color: "var(--theme-warning, #faad14)" };
  return { label: "超长", color: "var(--theme-error, #ff4d4f)" };
}

const MACTrajectoryChart: React.FC<MACTrajectoryChartProps> = ({
  data,
  loading = false,
  onError,
  dataZoomStart = 66,
  dataZoomEnd = 100,
}) => {
  // ECharts 配置 (hooks 必须无条件调用,放在 early return 之前)
  const option: EChartsOption = useMemo(() => {
    if (!data || data.length === 0) return {};

    // 按设备分组数据
    const deviceGroups = data.reduce((groups, node) => {
      const device = node.deviceName || "Unknown Device";
      if (!groups[device]) {
        groups[device] = [];
      }
      groups[device].push(node);
      return groups;
    }, {} as Record<string, TrajectoryNode[]>);

    // 构建Gantt图数据
    // value 维度约定(与下方 encode 一一对应,改顺序必须同步改 encode):
    //   [0]=设备索引(Y类目) [1]=startTime毫秒(X起点) [2]=endTime毫秒(X终点)
    //   [3]=duration秒(tooltip) [4]=eventType [5]=mac [6]=deviceName
    const devices = Object.keys(deviceGroups);
    const seriesData = data.map((node) => ({
      name: `${node.interface} (VLAN ${node.vlanId})`,
      value: [
        devices.indexOf(node.deviceName), // [0] Y轴：设备索引
        new Date(node.startTime).getTime(), // [1] X轴：开始时间
        new Date(node.endTime).getTime(), // [2] X轴：结束时间
        node.duration, // [3] 停留时长（用于tooltip）
        node.eventType, // [4] 事件类型（用于颜色）
        node.mac, // [5] MAC地址（用于tooltip）
        node.deviceName, // [6] 设备名（用于tooltip）
      ],
      itemStyle: {
        color: EVENT_COLORS[node.eventType as MACEventType],
      },
    }));

    return {
      // 标题由外层 Card 提供(Phase 14 重构:移除内部 title 避免重复)
      tooltip: {
        trigger: "item",
        formatter: (params: any) => {
          // Phase 13 CR + W5 修复:不再读 value 数组下标(value 数组服务于 ECharts
          // custom series renderItem 的 position 索引,index 错位是脆弱的契约)。
          // 改用 data[params.dataIndex] 节点对象访问字段,稳定且与字段增减解耦。
          const node = data[params.dataIndex];
          const durationHours = node.duration > 0 ? (node.duration / 3600).toFixed(1) : "瞬时";
          const dwell = getDwellHeatmap(node.duration);
          return `
            <div>
              <span style="display:inline-block;padding:0 6px;background:${dwell.color};color:#fff;border-radius:2px;font-size:12px;">停留${dwell.label}</span><br/>
              <strong>MAC: ${node.mac}</strong><br/>
              厂商: ${node.vendor ?? "Unknown"}<br/>
              设备: ${node.deviceName}<br/>
              端口: ${params.name}<br/>
              停留: ${durationHours}${node.duration > 0 ? "小时" : ""}<br/>
              事件: ${node.eventType}<br/>
              VLAN: ${node.vlanId ?? "-"}
            </div>
          `;
        },
      },
      grid: {
        left: "15%",
        right: "5%",
        bottom: "15%",
        top: "8%",
      },
      xAxis: {
        type: "time",
        axisLabel: {
          formatter: "{yyyy}-{MM}-{dd} {HH}:{mm}",
        },
      },
      yAxis: {
        type: "category",
        data: devices,
        axisLabel: {
          formatter: (value: string) => {
            // 设备名太长时截断
            return value.length > 15 ? value.substring(0, 12) + "..." : value;
          },
        },
      },
      series: [
        {
          type: "custom",
          // 关键修复(Phase 14):显式 encode 锁定维度 -> 轴。
          // 缺少 encode 时 ECharts 默认用 value[0](设备索引 0/1/2 整数) 作为
          // xAxis(time) 数据源算轴范围,整数被当毫秒时间戳 -> 轴范围坍缩到
          // epoch 0(1970),时间轴显示 1969/1970,且 renderItem 用真实 2026
          // 时间戳算出的坐标落到画布外 -> Gantt 条不可见。
          // encode.x 为数组时 ECharts 取所列维度的 min/max 作轴范围 ->
          // xAxis 范围 = min(startTime) ~ max(endTime) 的真实区间。
          encode: {
            x: [1, 2],
            y: 0,
            tooltip: [3, 4, 5, 6],
          },
          // 零宽兜底条不被 dataZoom 边缘裁掉
          clip: false,
          renderItem: (params: any, api: any) => {
            const categoryIndex = api.value(0);
            const timeStart = api.value(1);
            const timeEnd = api.value(2);

            const coordStart = api.coord([timeStart, categoryIndex]);
            const coordEnd = api.coord([timeEnd, categoryIndex]);

            const barHeight = 20;
            // 零宽区间兜底:startTime≈endTime(duration=0 瞬时事件)时 width=0 不可见,
            // 用 Math.max 保证最小可见宽度(2px),不破坏时间语义(时长列仍显示 0s/瞬时)。
            const barWidth = Math.max(coordEnd[0] - coordStart[0], 2);

            return {
              type: "rect",
              shape: {
                x: coordStart[0],
                y: coordStart[1] - barHeight / 2,
                width: barWidth,
                height: barHeight,
              },
              style: {
                fill: api.visual("color"),
              },
            };
          },
          data: seriesData,
        },
      ],
      dataZoom: [
        {
          type: "slider",
          xAxisIndex: 0,
          bottom: "5%",
          start: dataZoomStart,
          end: dataZoomEnd,
        },
      ],
    };
  }, [data, dataZoomStart, dataZoomEnd]);

  // 高度根据设备数自适应(单设备紧凑,多设备扩展),避免稀疏数据时固定高度留大片空白
  const chartHeight = useMemo(() => {
    if (!data || data.length === 0) return 240;
    const deviceCount = new Set(data.map((n) => n.deviceName || n.deviceId)).size;
    return Math.max(240, deviceCount * 50 + 130);
  }, [data]);

  // 处理空数据
  if (!loading && (!data || data.length === 0)) {
    return <Empty description="暂无轨迹数据" />;
  }

  if (loading) {
    return (
      <div style={{ textAlign: "center", padding: "50px" }}>
        <Spin>
          <div style={{ minHeight: 80 }} />
        </Spin>
        <div style={{ marginTop: 8, color: "rgba(0, 0, 0, 0.45)" }}>加载轨迹数据...</div>
      </div>
    );
  }

  return (
    <div style={{ width: "100%", height: chartHeight }}>
      <ReactECharts
        option={option}
        style={{ height: "100%", width: "100%" }}
        opts={{ renderer: "canvas" }}
        onError={(error) => {
          console.error("ECharts error:", error);
          onError?.(error);
        }}
      />
    </div>
  );
};

export default MACTrajectoryChart;
