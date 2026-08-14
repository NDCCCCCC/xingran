/**
 * ChartWidget - 图表 Widget
 *
 * 基于 ECharts 的图表组件，支持折线图、柱状图、饼图等
 */

import { useMemo } from "react";
import ReactECharts from "@/components/charts/EChartsWrapper";
import type { EChartsOption, PieSeriesOption } from "echarts";
import type { ChartDisplayConfig, WidgetConfig } from "@/types/dashboard";
import { BaseWidget } from "../base/BaseWidget";
import type { BaseWidgetProps } from "../base/BaseWidget";
import { useWidgetData } from "@/hooks/useWidgetData";

interface ChartWidgetProps {
  widget: WidgetConfig;
  display: ChartDisplayConfig;
  onEdit?: () => void;
  onDelete?: () => void;
}

export const ChartWidget: React.FC<ChartWidgetProps> = ({ widget, display, onEdit, onDelete }) => {
  // 使用useWidgetData直接获取数据
  const { data, loading, error, refresh } = useWidgetData(widget);
  // 构建ECharts配置
  const option = useMemo<EChartsOption>(() => {
    if (!data || typeof data !== "object") {
      return getEmptyOption();
    }

    const d = data as Record<string, unknown>;
    const { chartType } = display;

    switch (chartType) {
      case "line":
        return getLineOption(d, display);
      case "bar":
        return getBarOption(d, display);
      case "pie":
        return getPieOption(d, display);
      case "area":
        return getAreaOption(d, display);
      default:
        return getEmptyOption();
    }
  }, [data, display]);

  return (
    <BaseWidget
      widget={widget}
      data={data}
      loading={loading}
      error={error}
      onEdit={onEdit}
      onDelete={onDelete}
      onRefresh={refresh}
    >
      <div className="chart-widget">
        <ReactECharts
          option={option}
          style={{ height: "100%", width: "100%" }}
          opts={{ renderer: "svg" }}
        />
      </div>
    </BaseWidget>
  );
};

// 折线图配置
function getLineOption(data: Record<string, unknown>, display: ChartDisplayConfig): EChartsOption {
  const xData = extractArray(data[display.xField ?? "x"]);
  const yData = extractArray(data[display.yField ?? "y"]);

  return {
    grid: { top: 10, right: 10, bottom: 20, left: 40 },
    xAxis: {
      type: "category",
      data: xData,
      axisLine: { lineStyle: { color: "var(--theme-border-primary, #d9d9d9)" } },
    },
    yAxis: {
      type: "value",
      axisLine: { show: false },
      splitLine: { lineStyle: { color: "var(--theme-border-secondary, #f0f0f0)" } },
    },
    series: [
      {
        data: yData,
        type: "line",
        smooth: display.smooth ?? false,
        lineStyle: { width: 2 },
        itemStyle: { color: display.colors?.[0] ?? "var(--theme-info, #1890ff)" },
      },
    ],
    tooltip: {
      trigger: "axis",
    },
  };
}

// 柱状图配置
function getBarOption(data: Record<string, unknown>, display: ChartDisplayConfig): EChartsOption {
  const xData = extractArray(data[display.xField ?? "x"]);
  const yData = extractArray(data[display.yField ?? "y"]);

  return {
    grid: { top: 10, right: 10, bottom: 20, left: 40 },
    xAxis: {
      type: "category",
      data: xData,
      axisLine: { lineStyle: { color: "var(--theme-border-primary, #d9d9d9)" } },
    },
    yAxis: {
      type: "value",
      axisLine: { show: false },
      splitLine: { lineStyle: { color: "var(--theme-border-secondary, #f0f0f0)" } },
    },
    series: [
      {
        data: yData,
        type: "bar",
        itemStyle: {
          color: display.colors?.[0] ?? "var(--theme-info, #1890ff)",
          borderRadius: [4, 4, 0, 0],
        },
      },
    ],
    tooltip: {
      trigger: "axis",
    },
  };
}

// 饼图配置
function getPieOption(data: Record<string, unknown>, _display: ChartDisplayConfig): EChartsOption {
  const rawData = data.data ?? data.values ?? data;
  const items = extractArray(rawData);

  const pieData = items.map((item, index) => {
    // Try to parse as JSON for structured data
    try {
      const parsed = typeof item === "string" ? JSON.parse(item) : item;
      if (typeof parsed === "object" && parsed !== null) {
        const rawValue = (parsed as Record<string, unknown>).value ?? 0;
        return {
          name: String((parsed as Record<string, unknown>).name ?? `项${index + 1}`),
          value: typeof rawValue === "number" ? rawValue : Number(rawValue) || 0,
        };
      }
    } catch {
      // Use string as both name and value
    }
    return {
      name: `项${index + 1}`,
      value: typeof item === "number" ? item : Number(item) || 0,
    };
  });

  return {
    series: [
      {
        type: "pie",
        data: pieData,
        radius: ["40%", "70%"],
        itemStyle: {
          borderRadius: 4,
          borderColor: "var(--theme-neutral-white, #fff)",
          borderWidth: 2,
        },
      },
    ],
    tooltip: {
      trigger: "item",
      formatter: "{b}: {c} ({d}%)",
    },
  };
}

// 面积图配置
function getAreaOption(data: Record<string, unknown>, display: ChartDisplayConfig): EChartsOption {
  const xData = extractArray(data[display.xField ?? "x"]);
  const yData = extractArray(data[display.yField ?? "y"]);

  return {
    grid: { top: 10, right: 10, bottom: 20, left: 40 },
    xAxis: {
      type: "category",
      data: xData,
      axisLine: { lineStyle: { color: "var(--theme-border-primary, #d9d9d9)" } },
    },
    yAxis: {
      type: "value",
      axisLine: { show: false },
      splitLine: { lineStyle: { color: "var(--theme-border-secondary, #f0f0f0)" } },
    },
    series: [
      {
        data: yData,
        type: "line",
        smooth: display.smooth ?? true,
        areaStyle: {
          color: {
            type: "linear",
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: display.colors?.[0] ?? "var(--theme-info, #1890ff)" },
              { offset: 1, color: "rgba(24, 144, 255, 0.1)" },
            ],
          },
        },
        lineStyle: { width: 2 },
      },
    ],
    tooltip: {
      trigger: "axis",
    },
  };
}

// 空配置
function getEmptyOption(): EChartsOption {
  return {
    title: {
      text: "暂无数据",
      left: "center",
      top: "center",
      textStyle: { color: "var(--theme-text-tertiary, #bfbfbf)" },
    },
  };
}

// 提取数组数据
function extractArray(val: unknown): string[] {
  if (Array.isArray(val)) return val.map(String);
  if (val !== null && val !== undefined) return [String(val)];
  return [];
}
