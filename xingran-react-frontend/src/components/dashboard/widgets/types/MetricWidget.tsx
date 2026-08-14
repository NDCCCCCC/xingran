import { useMemo, memo } from "react";
import { Progress } from "antd";
import type { ProgressDisplayConfig, WidgetConfig } from "@/types/dashboard";
import { BaseWidget } from "../base/BaseWidget";
import { useWidgetData } from "@/hooks/useWidgetData";

interface MetricWidgetProps {
  widget: WidgetConfig;
  display: ProgressDisplayConfig;
  onEdit?: () => void;
  onDelete?: () => void;
}

export const MetricWidget = memo(({ widget, display, onEdit, onDelete }: MetricWidgetProps) => {
  const { data, loading, error, refresh } = useWidgetData(widget);

  const { percent, color } = useMemo(() => {
    if (!data || typeof data !== "object") {
      return { percent: 0, color: undefined };
    }

    const d = data as Record<string, unknown>;
    const value = Number(d.value ?? d.percent ?? 0);
    const target = display.target ?? 100;
    const calculatedPercent = Math.min(Math.round((value / target) * 100), 100);

    let strokeColor = undefined;
    if (display.colorThresholds) {
      const sorted = [...display.colorThresholds].sort((a, b) => b.value - a.value);
      for (const threshold of sorted) {
        if (calculatedPercent >= threshold.value) {
          strokeColor = threshold.color;
          break;
        }
      }
    }

    return { percent: calculatedPercent, color: strokeColor };
  }, [data, display.target, display.colorThresholds]);

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
      <div className="metric-widget">
        <Progress
          type="circle"
          percent={percent}
          strokeColor={color}
          width={100}
          format={(p) => <span className="metric-widget-value">{p}%</span>}
        />
      </div>
    </BaseWidget>
  );
});
