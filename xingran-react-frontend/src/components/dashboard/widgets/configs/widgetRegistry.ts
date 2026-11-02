/**
 * Widget 注册表
 *
 * 管理 Widget 类型和对应组件的映射关系
 */

import { lazy } from "react";
import type { WidgetType } from "@/types/dashboard";
import {
  asWidgetComponent,
  type WidgetBaseProps,
  type WidgetComponentType,
} from "@/types/widgets/helpers";

// 重新导出类型以保持向后兼容
export type { WidgetBaseProps as BaseWidgetProps, WidgetComponentType };

/**
 * Widget 配置接口
 */
export interface WidgetConfig {
  /** Widget 类型 */
  type: WidgetType;

  /** Widget 显示名称 */
  displayName: string;

  /** Widget 描述 */
  description: string;

  /** Widget 图标 */
  icon: string;

  /** 默认尺寸 */
  defaultSize: {
    w: number;
    h: number;
  };

  /** 组件（懒加载） */
  component: React.LazyExoticComponent<WidgetComponentType>;

  /** 是否支持数据源配置 */
  supportsDataSource?: boolean;

  /** 支持的数据源类型 */
  supportedDataSources?: ("api" | "websocket" | "static")[];
}

/**
 * Widget 注册表
 */
export const widgetRegistry: Record<WidgetType, WidgetConfig> = {
  "stat-card": {
    type: "stat-card",
    displayName: "统计卡片",
    description: "显示单个关键指标的数值卡片",
    icon: "🔢",
    defaultSize: { w: 6, h: 3 },
    component: lazy(() =>
      import("../types/StatCardWidget").then((m) => ({
        default: asWidgetComponent(m.StatCardWidget),
      }))
    ),
    supportsDataSource: true,
    supportedDataSources: ["api", "websocket", "static"],
  },
  chart: {
    type: "chart",
    displayName: "图表",
    description: "支持折线图、柱状图、饼图等多种图表类型",
    icon: "📊",
    defaultSize: { w: 12, h: 6 },
    component: lazy(() =>
      import("../types/ChartWidget").then((m) => ({
        default: asWidgetComponent(m.ChartWidget),
      }))
    ),
    supportsDataSource: true,
    supportedDataSources: ["api", "websocket", "static"],
  },
  table: {
    type: "table",
    displayName: "表格",
    description: "以表格形式展示数据列表",
    icon: "📋",
    defaultSize: { w: 24, h: 8 },
    component: lazy(() =>
      import("../types/TableWidget").then((m) => ({
        default: asWidgetComponent(m.TableWidget),
      }))
    ),
    supportsDataSource: true,
    supportedDataSources: ["api", "websocket", "static"],
  },
  list: {
    type: "list",
    displayName: "列表",
    description: "以简洁的列表形式展示数据",
    icon: "📝",
    defaultSize: { w: 8, h: 6 },
    component: lazy(() =>
      import("../types/ListWidget").then((m) => ({
        default: asWidgetComponent(m.ListWidget),
      }))
    ),
    supportsDataSource: true,
    supportedDataSources: ["api", "websocket", "static"],
  },
  progress: {
    type: "progress",
    displayName: "进度条",
    description: "以进度条形式展示百分比或完成度",
    icon: "📊",
    defaultSize: { w: 6, h: 4 },
    component: lazy(() =>
      import("../types/ProgressWidget").then((m) => ({
        default: asWidgetComponent(m.ProgressWidget),
      }))
    ),
    supportsDataSource: true,
    supportedDataSources: ["api", "websocket", "static"],
  },
  metric: {
    type: "metric",
    displayName: "指标",
    description: "圆形指标仪表盘",
    icon: "🎯",
    defaultSize: { w: 6, h: 4 },
    component: lazy(() =>
      import("../types/MetricWidget").then((m) => ({
        default: asWidgetComponent(m.MetricWidget),
      }))
    ),
    supportsDataSource: true,
    supportedDataSources: ["api", "websocket", "static"],
  },
};

/**
 * 获取所有 Widget 类型列表
 */
export function getWidgetTypes(): WidgetType[] {
  return Object.keys(widgetRegistry) as WidgetType[];
}

/**
 * 获取 Widget 配置
 */
export function getWidgetConfig(type: WidgetType): WidgetConfig | undefined {
  return widgetRegistry[type];
}

/**
 * 获取 Widget 组件
 */
export function getWidgetComponent(
  type: WidgetType
): React.LazyExoticComponent<WidgetComponentType> | undefined {
  return widgetRegistry[type]?.component;
}

/**
 * 注册新的 Widget 类型
 */
export function registerWidget(config: WidgetConfig): void {
  widgetRegistry[config.type] = config;
}

/**
 * 批量注册 Widget
 */
export function registerWidgets(configs: WidgetConfig[]): void {
  configs.forEach((config) => {
    registerWidget(config);
  });
}

/**
 * 默认导出注册表
 */
export default widgetRegistry;
