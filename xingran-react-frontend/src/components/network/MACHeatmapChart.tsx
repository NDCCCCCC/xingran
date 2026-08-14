/**
 * MAC 端口使用热力图组件
 *
 * Phase 15 PERF-04:
 *   - 桌面端渲染 ECharts heatmap (X 轴: 端口, Y 轴: 设备, 值: change_count)
 *   - 移动端 (< sm) 降级为 Top-20 端口列表 + 颜色卡片
 *   - 颜色映射: 低=蓝绿, 中=黄, 高=红
 */

import React, { useMemo } from "react";
// 改用 EChartsWrapper（lazy 包装），让 echarts-for-react 从 vendor-react 移到 lazy chunk
import ReactECharts from "@/components/charts/EChartsWrapper";
import type { EChartsOption } from "echarts";
import { Card, Empty, Spin, Tag } from "antd";
import type { HeatmapCell, HeatmapResult } from "@/lib/api/macHeatmapApi";

interface MACHeatmapChartProps {
  data: HeatmapResult;
  loading?: boolean;
  isMobile?: boolean;
}

// change_count → 颜色 (D-18 锁定色阶)
function getHeatColor(value: number, max: number): string {
  if (max <= 0) return "#bfbfbf";
  const ratio = value / max;
  if (ratio < 0.25) return "#50a3ba";
  if (ratio < 0.5) return "#fad252";
  if (ratio < 0.75) return "#eac736";
  return "#d94e5d";
}

const MACHeatmapChart: React.FC<MACHeatmapChartProps> = ({
  data,
  loading = false,
  isMobile = false,
}) => {
  // 桌面端: ECharts heatmap (hooks 必须无条件调用,放在 early return 之前)
  const desktopOption: EChartsOption = useMemo(() => {
    const cells = data?.cells ?? [];
    const maxCount = Math.max(...cells.map((c) => c.changeCount), 1);

    // 提取 x (端口) + y (设备) 维度
    const ports = Array.from(new Set(cells.map((c) => c.interfaceName)));
    const devices = Array.from(new Set(cells.map((c) => c.deviceNameSnapshot)));
    const deviceIndex = new Map(devices.map((d, i) => [d, i]));
    const portIndex = new Map(ports.map((p, i) => [p, i]));

    const points: [number, number, number][] = cells.map((c) => [
      portIndex.get(c.interfaceName) ?? 0,
      deviceIndex.get(c.deviceNameSnapshot) ?? 0,
      c.changeCount,
    ]);

    return {
      tooltip: {
        position: "top",
        formatter: (params: unknown) => {
          const p = params as { data: [number, number, number] };
          const port = ports[p.data[0]];
          const device = devices[p.data[1]];
          const count = p.data[2];
          return `<b>${device}</b><br/>端口: ${port}<br/>变更次数: ${count}`;
        },
      },
      grid: { left: 120, right: 20, top: 40, bottom: 80 },
      xAxis: {
        type: "category",
        data: ports,
        axisLabel: { rotate: 45, fontSize: 10 },
        splitArea: { show: true },
      },
      yAxis: {
        type: "category",
        data: devices,
        axisLabel: { fontSize: 10 },
        splitArea: { show: true },
      },
      visualMap: {
        min: 0,
        max: maxCount,
        calculable: true,
        orient: "horizontal",
        left: "center",
        bottom: 10,
        inRange: { color: ["#50a3ba", "#eac736", "#d94e5d"] },
      },
      series: [
        {
          name: "端口变更次数",
          type: "heatmap",
          data: points,
          label: { show: false },
          emphasis: {
            itemStyle: { borderColor: "var(--theme-neutral-900, #000)", borderWidth: 1 },
          },
        },
      ],
    };
  }, [data]);

  if (loading) {
    return (
      <div style={{ textAlign: "center", padding: 60 }}>
        <Spin tip="加载中..." />
      </div>
    );
  }

  if (!data?.cells || data.cells.length === 0) {
    return <Empty description="暂无热力图数据" />;
  }

  if (isMobile) {
    // 移动端: Top-20 端口列表 + 颜色卡片
    const topPorts = [...data.cells].sort((a, b) => b.changeCount - a.changeCount).slice(0, 20);
    const maxCount = Math.max(...topPorts.map((c) => c.changeCount), 1);

    return (
      <div data-testid="mac-heatmap-mobile">
        <h3 style={{ marginBottom: 16 }}>Top-20 高频变更端口</h3>
        {topPorts.map((cell: HeatmapCell, idx: number) => (
          <Card
            key={`${cell.deviceId}-${cell.interfaceName}-${idx}`}
            size="small"
            style={{ marginBottom: 8 }}
            styles={{ body: { padding: 12 } }}
          >
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
              <div>
                <div style={{ fontWeight: 500 }}>{cell.deviceNameSnapshot}</div>
                <div style={{ fontSize: 12, color: "var(--theme-text-tertiary, #999)" }}>
                  {cell.interfaceName}
                </div>
              </div>
              <Tag color={getHeatColor(cell.changeCount, maxCount)}>{cell.changeCount} 次</Tag>
            </div>
          </Card>
        ))}
      </div>
    );
  }

  return (
    <div data-testid="mac-heatmap-desktop">
      <ReactECharts
        option={desktopOption}
        style={{ height: 600, width: "100%" }}
        notMerge
        lazyUpdate
        theme="light"
      />
    </div>
  );
};

export default React.memo(MACHeatmapChart);
