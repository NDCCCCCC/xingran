import React, { useState, useEffect, useCallback, useMemo } from "react";
import {
  Card,
  DatePicker,
  Input,
  Button,
  Form,
  Space,
  Alert,
  App,
  Row,
  Col,
  Statistic,
  Spin,
} from "antd";
import { useQuery } from "@tanstack/react-query";
import { useLocation } from "react-router-dom";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import dayjs from "dayjs";
import type { Dayjs } from "dayjs";
import MACTrajectoryChart from "@/components/network/MACTrajectoryChart";
import MACEventsTimeline from "@/components/network/MACEventsTimeline";
import { queryMACTrajectory, queryMACVendor } from "@/lib/api/networkApi";
import { EmptyStateWithAction, ErrorAlertWithRetry } from "@/components/shared";
import { formatDurationSeconds } from "@/utils/duration";
import { formatDateTime } from "@/utils/datetime";

type RangePickerOnChange = React.ComponentProps<typeof DatePicker.RangePicker>["onChange"];

const { RangePicker } = DatePicker;

// 时间预设(与 14-01 列表页一致,D-07 锁定)
type PresetKey = "1h" | "24h" | "7d" | "30d" | "90d" | "custom";
interface Preset {
  key: PresetKey;
  label: string;
  amount: number;
  unit: "hour" | "day";
}
const PRESETS: Preset[] = [
  { key: "1h", label: "近 1h", amount: 1, unit: "hour" },
  { key: "24h", label: "近 24h", amount: 24, unit: "hour" },
  { key: "7d", label: "近 7d", amount: 7, unit: "day" },
  { key: "30d", label: "近 30d", amount: 30, unit: "day" },
  { key: "90d", label: "近 90d", amount: 90, unit: "day" },
  { key: "custom", label: "自定义", amount: 0, unit: "day" },
];

interface TrajectoryQueryParams {
  macAddress: string;
  startTime: string; // ISO timestamp
  endTime: string; // ISO timestamp
}

// 本地聚合统计结果(从 trajectoryData reduce 而来)
interface TrajectoryStats {
  deviceCount: number; // 涉及设备数(去重 deviceName)
  portCount: number; // 涉及端口数(去重 deviceId+interface)
  segmentCount: number; // 停留区间数(= nodes.length)
  totalDurationSec: number; // 总停留时长(秒)
  firstSeen: string | null; // 最早 startTime(ISO)
  lastSeen: string | null; // 最晚 endTime(ISO)
  vendor: string; // 厂商
}

const TrajectoryPage: React.FC = () => {
  const { message } = App.useApp();
  const [form] = Form.useForm();
  const location = useLocation();
  const [queryParams, setQueryParams] = useState<TrajectoryQueryParams | null>(null);
  const [activePreset, setActivePreset] = usePersistedStateController<PresetKey>({
    keyPrefix: location.pathname,
    keySuffix: "activePreset",
    defaultValue: "7d",
  });
  const [customRange, setCustomRange] = useState<[Dayjs, Dayjs] | null>(null);

  // 查询轨迹数据
  const {
    data: trajectoryData,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: ["macTrajectory", queryParams],
    queryFn: () => queryMACTrajectory(queryParams!),
    enabled: !!queryParams, // 有参数时才查询
    staleTime: 5 * 60 * 1000, // 5分钟缓存
  });

  // Phase 13 W4:并行查询 MAC 厂商(OUI),用于统计卡片 + tooltip 展示
  // - 24h staleTime 对齐后端 OUI Redis 缓存 TTL(13-08 设计决策 T-13W8-03)
  // - enabled 依赖 queryParams:无参数时不触发,避免空查询
  const { data: vendorName } = useQuery({
    queryKey: ["macVendor", queryParams?.macAddress],
    queryFn: () => queryMACVendor(queryParams!.macAddress),
    enabled: !!queryParams,
    staleTime: 24 * 60 * 60 * 1000, // 24h
  });

  // 合并 trajectoryData + vendorName 为 chartData,每个 node 携带 vendor 字段
  // - useMemo deps 锁定 trajectoryData + vendorName,避免重建数组
  // - 后端 TrajectoryNode 不含 vendor,前端在 page 层合并(网络契约最小化)
  // - chartData 作为 Gantt 图数据源
  const chartData = useMemo(() => {
    if (!trajectoryData) return [];
    return trajectoryData.map((node) => ({ ...node, vendor: vendorName }));
  }, [trajectoryData, vendorName]);

  // 本地聚合统计(Phase 14 重构)
  // 安全性:queryMACTrajectory 返回单 MAC 全量已聚合节点(trajectory 端点非分页,
  // 无 MaxPageSize=100 截断),与 stat-cards-from-list-length-capped-at-100 反模式
  // (分页 list 端点 list.length)本质不同,本地 reduce 即统计事实源。
  const stats: TrajectoryStats | null = useMemo(() => {
    if (!trajectoryData || trajectoryData.length === 0) return null;
    const deviceSet = new Set<string>();
    const portSet = new Set<string>();
    let totalDuration = 0;
    let minStart = Infinity;
    let maxEnd = -Infinity;
    for (const n of trajectoryData) {
      deviceSet.add(n.deviceName || n.deviceId);
      portSet.add(n.deviceId + "|" + n.interface);
      totalDuration += n.duration || 0;
      const s = new Date(n.startTime).getTime();
      const e = new Date(n.endTime).getTime();
      if (!Number.isNaN(s) && s < minStart) minStart = s;
      if (!Number.isNaN(e) && e > maxEnd) maxEnd = e;
    }
    return {
      deviceCount: deviceSet.size,
      portCount: portSet.size,
      segmentCount: trajectoryData.length,
      totalDurationSec: totalDuration,
      firstSeen: minStart === Infinity ? null : new Date(minStart).toISOString(),
      lastSeen: maxEnd === -Infinity ? null : new Date(maxEnd).toISOString(),
      vendor: vendorName ?? "Unknown Vendor",
    };
  }, [trajectoryData, vendorName]);

  // MAC地址规范化函数(复用后端逻辑)
  const normalizeMACAddress = (mac: string): string => {
    const cleaned = mac.replace(/[^a-fA-F0-9]/g, "");
    if (cleaned.length !== 12) return "";
    return cleaned.match(/.{2}/g)?.join(":").toUpperCase() || "";
  };

  // 预设 → 设置表单 dateRange + activePreset
  const handlePresetClick = useCallback(
    (preset: Preset) => {
      if (preset.key === "custom") {
        setActivePreset("custom");
        setCustomRange(null);
        return;
      }
      setActivePreset(preset.key);
      setCustomRange(null);
      const start = dayjs().subtract(preset.amount, preset.unit);
      const end = dayjs();
      form.setFieldsValue({ dateRange: [start, end] });
    },
    [form, setActivePreset]
  );

  const handleCustomRangeChange = useCallback<NonNullable<RangePickerOnChange>>(
    (values) => {
      const arr = values as [Dayjs, Dayjs] | null;
      setCustomRange(arr);
      if (arr && arr[0] && arr[1]) {
        setActivePreset("custom");
        form.setFieldsValue({ dateRange: arr });
      }
    },
    [form, setActivePreset]
  );

  // URL 参数预填(D-17 锁定)— 进入页面时读取 ?mac=...&startTime=...&endTime=...&deviceId=...
  useEffect(() => {
    const sp = new URLSearchParams(location.search);
    const urlMac = sp.get("mac");
    const startTime = sp.get("startTime");
    const endTime = sp.get("endTime");
    // deviceId 仅供未来扩展;当前不参与查询
    // const deviceId = sp.get('deviceId');

    const initial: Record<string, unknown> = {};
    if (urlMac) initial.mac = normalizeMACAddress(urlMac) || urlMac;
    if (startTime && endTime) {
      initial.dateRange = [dayjs(startTime), dayjs(endTime)];
      setActivePreset("custom");
      setCustomRange([dayjs(startTime), dayjs(endTime)]);
    }
    if (Object.keys(initial).length > 0) {
      form.setFieldsValue(initial);
    }

    // URL 包含完整查询条件时,自动触发查询(14-01 跳转场景)
    if (urlMac && startTime && endTime) {
      const normalizedMAC = normalizeMACAddress(urlMac);
      if (normalizedMAC) {
        setQueryParams({
          macAddress: normalizedMAC,
          startTime: dayjs(startTime).toISOString(),
          endTime: dayjs(endTime).toISOString(),
        });
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleSearch = async () => {
    try {
      const values = await form.validateFields();
      const { mac, dateRange } = values;

      // 格式化MAC地址(自动规范化)
      const normalizedMAC = normalizeMACAddress(mac);
      if (!normalizedMAC) {
        message.error("请输入有效的MAC地址");
        return;
      }

      setQueryParams({
        macAddress: normalizedMAC,
        startTime: dateRange[0].toISOString(),
        endTime: dateRange[1].toISOString(),
      });
    } catch (err) {
      console.error("Form validation failed:", err);
    }
  };

  const handleReset = () => {
    form.resetFields();
    setQueryParams(null);
    setActivePreset("7d");
    setCustomRange(null);
    const preset = PRESETS.find((p) => p.key === "7d")!;
    form.setFieldsValue({
      dateRange: [dayjs().subtract(preset.amount, preset.unit), dayjs()],
    });
  };

  return (
    <div style={{ padding: "24px" }}>
      {/* 段1:查询区 */}
      <Card title="MAC地址轨迹查询" bordered={false}>
        {/* 时间预设按钮组(D-07) */}
        <Space wrap style={{ marginBottom: 12 }}>
          {PRESETS.map((p) => (
            <Button
              key={p.key}
              type={activePreset === p.key ? "primary" : "default"}
              size="small"
              onClick={() => handlePresetClick(p)}
            >
              {p.label}
            </Button>
          ))}
        </Space>
        {activePreset === "custom" && (
          <div style={{ marginBottom: 12 }}>
            <RangePicker
              showTime
              format="YYYY-MM-DD HH:mm"
              value={customRange}
              onChange={handleCustomRangeChange}
              placeholder={["开始时间", "结束时间"]}
            />
          </div>
        )}
        <Form form={form} layout="inline">
          <Form.Item
            name="mac"
            label="MAC地址"
            rules={[
              { required: true, message: "请输入MAC地址" },
              {
                pattern: /^[A-Fa-f0-9]{2}[:-][A-Fa-f0-9]{2}[:-][A-Fa-f0-9]{2}[:-][A-Fa-f0-9]{2}[:-][A-Fa-f0-9]{2}[:-][A-Fa-f0-9]{2}$/,
                message: "MAC地址格式不正确",
              },
            ]}
          >
            <Input
              placeholder="AA:BB:CC:DD:EE:FF"
              style={{ width: 200 }}
              onChange={(e) => {
                // 实时格式化输入(Phase 13 CR-02:走 AntD Form 单一源,删除 e.target.value = formatted 反模式)
                const formatted = normalizeMACAddress(e.target.value);
                if (formatted) {
                  form.setFieldValue("mac", formatted);
                }
              }}
              onBlur={(e) => {
                // 失焦自动规范化(增强 UX,与 onChange 形成双重保险)
                const formatted = normalizeMACAddress(e.target.value);
                if (formatted) {
                  form.setFieldValue("mac", formatted);
                }
              }}
            />
          </Form.Item>

          <Form.Item
            name="dateRange"
            label="时间范围"
            initialValue={[dayjs().subtract(7, "days"), dayjs()]}
            rules={[{ required: true, message: "请选择时间范围" }]}
          >
            <RangePicker showTime format="YYYY-MM-DD HH:mm" style={{ width: 400 }} />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" onClick={handleSearch}>
                查询
              </Button>
              <Button onClick={handleReset}>重置</Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>

      {/* 14-05b:错误态替换为 ErrorAlertWithRetry(D-20 错误码 1006/1007/500 分级文案) */}
      {error && (
        <div style={{ marginTop: "16px" }}>
          <ErrorAlertWithRetry error={error} onRetry={() => refetch()} />
        </div>
      )}

      {/* 14-05b:数据为空时 EmptyStateWithAction 提示(D-18 引导) */}
      {!error && queryParams && !isLoading && (!trajectoryData || trajectoryData.length === 0) && (
        <div style={{ marginTop: "16px" }}>
          <EmptyStateWithAction description="该 MAC 在此时间范围内无轨迹数据" />
        </div>
      )}

      {/* 段2:统计卡片行(本地聚合,见 stats 安全性注释) */}
      {stats && (
        <Spin spinning={isLoading}>
          <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
            <Col xs={24} sm={12} md={8} lg={6}>
              <Card size="small">
                <Statistic title="涉及设备" value={stats.deviceCount} />
              </Card>
            </Col>
            <Col xs={24} sm={12} md={8} lg={6}>
              <Card size="small">
                <Statistic title="涉及端口" value={stats.portCount} />
              </Card>
            </Col>
            <Col xs={24} sm={12} md={8} lg={6}>
              <Card size="small">
                <Statistic title="停留区间" value={stats.segmentCount} />
              </Card>
            </Col>
            <Col xs={24} sm={12} md={8} lg={6}>
              <Card size="small">
                <Statistic title="总停留时长" value={formatDurationSeconds(stats.totalDurationSec)} />
              </Card>
            </Col>
            <Col xs={24} sm={12} md={8} lg={6}>
              <Card size="small">
                <Statistic title="首次出现" value={formatDateTime(stats.firstSeen)} />
              </Card>
            </Col>
            <Col xs={24} sm={12} md={8} lg={6}>
              <Card size="small">
                <Statistic title="最后出现" value={formatDateTime(stats.lastSeen)} />
              </Card>
            </Col>
            <Col xs={24} sm={12} md={8} lg={6}>
              <Card size="small">
                <Statistic title="厂商" value={stats.vendor} />
              </Card>
            </Col>
          </Row>
        </Spin>
      )}

      {/* 段3:轨迹 Gantt 图(左) + 事件时间线(右) 左右布局(Phase 14 重构:更紧凑) */}
      <Row gutter={16} style={{ marginTop: 16 }}>
        <Col xs={24} lg={14}>
          <Card
            title={queryParams ? `轨迹时间线 — ${queryParams.macAddress}` : "轨迹时间线"}
            bordered={false}
          >
            <MACTrajectoryChart
              data={chartData}
              loading={isLoading}
              onError={(err) => {
                console.error("Trajectory chart error:", err);
              }}
            />
          </Card>
        </Col>
        <Col xs={24} lg={10}>
          {queryParams ? (
            <MACEventsTimeline
              mac={queryParams.macAddress}
              startTime={queryParams.startTime}
              endTime={queryParams.endTime}
            />
          ) : (
            <Card title="事件时间线" bordered={false}>
              <Alert
                type="info"
                showIcon
                message="请先执行查询"
                description="查询后将在此显示 MAC 在该时间范围内的事件时间线"
              />
            </Card>
          )}
        </Col>
      </Row>
    </div>
  );
};

export default TrajectoryPage;
