/**
 * StatCardWidget - 统计卡片 Widget
 *
 * 显示单个关键指标的卡片组件
 */

import { useMemo } from "react";
import { ArrowUpOutlined, ArrowDownOutlined, ReloadOutlined } from "@ant-design/icons";
import type { StatCardDisplayConfig } from "@/types/dashboard";
import { useWidgetData } from "@/hooks/useWidgetData";
import type { WidgetConfig } from "@/types/dashboard";

interface StatCardWidgetProps {
  widget: WidgetConfig;
  display: StatCardDisplayConfig;
  onEdit?: () => void;
  onDelete?: () => void;
}

export const StatCardWidget: React.FC<StatCardWidgetProps> = ({
  widget,
  display,
  onEdit: _onEdit,
  onDelete: _onDelete,
}) => {
  // 使用useWidgetData直接获取数据
  const { data, loading, error, refresh: _refresh } = useWidgetData(widget);

  // 提取数值
  // eslint-disable-next-line react-hooks/preserve-manual-memoization
  const { value, label, trend, icon, color } = useMemo(() => {
    if (typeof data !== "object" || !data) {
      return { value: "-", label: widget.title, trend: null, icon: null, color: display.iconColor };
    }

    const d = data as Record<string, unknown>;

    // 尝试多种字段名来获取数值
    let numericValue: unknown = d.value ?? d.count ?? d.total;

    // 如果还是没有，尝试常见的设备统计字段名
    if (numericValue === undefined || numericValue === null) {
      const possibleFields = [
        "totalDevices",
        "onlineDevices",
        "offlineDevices",
        "unknownDevices",
        "deviceCount",
        "serverCount",
        "userCount",
        "orderCount",
        "amount",
        "quantity",
        "number",
      ];
      for (const field of possibleFields) {
        if (typeof d[field] === "number") {
          numericValue = d[field];
          break;
        }
      }
    }

    // 最后尝试：查找第一个数值类型的字段
    if (numericValue === undefined || numericValue === null) {
      for (const key of Object.keys(d)) {
        const val = d[key];
        if (typeof val === "number" && !isNaN(val)) {
          numericValue = val;
          break;
        }
      }
    }

    return {
      value: formatValue(numericValue ?? 0, display),
      label: d.label ?? widget.title,
      trend: d.trend as { value: number; direction: "up" | "down" } | null,
      icon: display.icon,
      color: display.iconColor,
    };
  }, [data, display, widget.title]);

  // 格式化数值
  function formatValue(val: unknown, config: StatCardDisplayConfig): string {
    let num = Number(val);
    if (isNaN(num)) return String(val);

    const decimals = config.decimals ?? 0;
    num = Number(num.toFixed(decimals));

    let result = "";
    if (config.prefix) result += config.prefix;
    result += num.toLocaleString();
    if (config.suffix) result += config.suffix;
    if (config.percentage) result += "%";

    return result;
  }

  // 获取趋势图标
  const trendIcon = useMemo(() => {
    if (!trend) return null;
    if (trend.direction === "up") {
      return <ArrowUpOutlined style={{ color: "var(--theme-success, #2d8949)" }} />;
    }
    return <ArrowDownOutlined style={{ color: "var(--theme-error, #ba3630)" }} />;
  }, [trend]);

  return (
    <div className="stat-card-widget" style={{ color }}>
      {loading && <ReloadOutlined spin />}
      {error && (
        <div className="base-widget__error">
          <span>{error}</span>
        </div>
      )}
      {!loading && !error && (
        <>
          {icon && (
            <div
              className="stat-card-widget__icon"
              style={{ backgroundColor: color ? `${color}20` : undefined }}
            >
              <span style={{ fontSize: 24 }}>{icon}</span>
            </div>
          )}
          <div className="stat-card-widget__content">
            <div className="stat-card-widget__value">{value}</div>
            <div className="stat-card-widget__label">{String(label ?? "")}</div>
            {trend && (
              <div className="stat-card-widget__trend">
                {trendIcon}
                <span>{trend.value}%</span>
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
};
