/**
 * MAC 端口使用热力图页面
 *
 * Phase 15 PERF-04:
 *   - 默认 7 天范围 (后端兜底)
 *   - 时间预设: 近 1h / 24h / 7d / 30d / 90d / 自定义 (沿用 Phase 14 D-07)
 *   - 移动端 (< sm) 降级为 Top-20 端口列表
 *   - 错误状态: 复用 ErrorAlertWithRetry
 *   - 空状态: 复用 EmptyStateWithAction
 */

import React, { Suspense, lazy, useCallback, useState } from "react";
import {
  Card, DatePicker, Space, Button, Form, Alert, Grid, Spin, Tag,
} from "antd";
import { useQuery } from "@tanstack/react-query";
import { useLocation } from "react-router-dom";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import dayjs, { type Dayjs } from "dayjs";

type RangePickerOnChange = React.ComponentProps<typeof DatePicker.RangePicker>["onChange"];
import { queryMACHeatmap, type HeatmapResult } from "@/lib/api/macHeatmapApi";
import { ErrorAlertWithRetry, EmptyStateWithAction } from "@/components/shared";

const { RangePicker } = DatePicker;
const MACHeatmapChart = lazy(() => import("@/components/network/MACHeatmapChart"));

const { useBreakpoint } = Grid;

// 时间预设 (沿用 Phase 14 D-07)
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

const HeatmapPage: React.FC = () => {
  const [form] = Form.useForm();
  const screens = useBreakpoint();
  const isMobile = !screens.sm;

  const [queryParams, setQueryParams] = useState<{ startTime: string; endTime: string } | null>(null);
  const location = useLocation();
  const [activePreset, setActivePreset] = usePersistedStateController<PresetKey>({
    keyPrefix: location.pathname,
    keySuffix: "activePreset",
    defaultValue: "7d",
  });
  const [customRange, setCustomRange] = useState<[Dayjs, Dayjs] | null>(null);

  // 拉数据
  const {
    data: heatmapData,
    isLoading,
    error,
    refetch,
  } = useQuery<HeatmapResult>({
    queryKey: ["macHeatmap", queryParams],
    queryFn: () => queryMACHeatmap(queryParams!),
    enabled: !!queryParams,
    staleTime: 5 * 60 * 1000, // 5分钟 (匹配后端 TTL)
  });

  // 预设点击
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

  // 自定义范围
  const handleCustomRange = useCallback<NonNullable<RangePickerOnChange>>((dates) => {
    setCustomRange(dates as [Dayjs, Dayjs] | null);
    if (dates && dates[0] && dates[1]) {
      setActivePreset("custom");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- setActivePreset from usePersistedStateController is stable
  }, []);

  // 应用查询
  const handleApply = useCallback(() => {
    let start: Dayjs;
    let end: Dayjs;
    if (customRange) {
      start = customRange[0];
      end = customRange[1];
    } else {
      const preset = PRESETS.find((p) => p.key === activePreset);
      if (!preset || preset.key === "custom") {
        start = dayjs().subtract(7, "day");
        end = dayjs();
      } else {
        start = dayjs().subtract(preset.amount, preset.unit);
        end = dayjs();
      }
    }
    setQueryParams({
      startTime: start.toISOString(),
      endTime: end.toISOString(),
    });
  }, [customRange, activePreset]);

  // 错误状态
  if (error) {
    return (
      <ErrorAlertWithRetry
        error={error}
        onRetry={() => refetch()}
        description="热力图数据加载失败"
      />
    );
  }

  // 空状态
  if (!isLoading && heatmapData && heatmapData.cells.length === 0) {
    return (
      <EmptyStateWithAction
        title="暂无热力图数据"
        description="所选时间范围内没有端口变更记录"
        actionLabel="刷新"
        onAction={() => refetch()}
      />
    );
  }

  return (
    <div style={{ padding: 24 }}>
      <Card
        title={
          <Space>
            <span>MAC 端口使用热力图</span>
            <Tag color="blue">PERF-04</Tag>
          </Space>
        }
        extra={
          <Space wrap>
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
        }
      >
        <Form form={form} layout="inline" style={{ marginBottom: 16 }}>
          <Form.Item label="时间范围" name="dateRange">
            <RangePicker
              showTime
              value={customRange}
              onChange={handleCustomRange}
              disabled={activePreset !== "custom"}
            />
          </Form.Item>
          <Form.Item>
            <Button type="primary" onClick={handleApply}>
              查询
            </Button>
          </Form.Item>
        </Form>

        {!queryParams && !isLoading ? (
          <Alert
            type="info"
            showIcon
            message="请选择时间范围后点击查询"
            description="默认展示近 7 天数据,数据源: MV-04 (mv_mac_port_daily_count)"
          />
        ) : (
          <Suspense
            fallback={
              <div style={{ textAlign: "center", padding: 60 }}>
                <Spin tip="加载图表中..." />
              </div>
            }
          >
            <MACHeatmapChart
              data={heatmapData ?? { cells: [], topN: 0, start: "", end: "", total: 0, snapshot: "" }}
              loading={isLoading}
              isMobile={isMobile}
            />
          </Suspense>
        )}

        {heatmapData && heatmapData.cells.length > 0 && (
          <div style={{ marginTop: 16, fontSize: 12, color: "var(--theme-text-tertiary, #999)" }}>
            数据快照: {heatmapData.snapshot} | TopN: {heatmapData.topN} | 总计:{" "}
            {heatmapData.total} 条
          </div>
        )}
      </Card>
    </div>
  );
};

export default HeatmapPage;
