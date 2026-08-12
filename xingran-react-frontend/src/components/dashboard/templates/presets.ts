/**
 * 预设仪表盘模板
 *
 * 提供开箱即用的仪表盘模板
 */

import type { PresetDashboardTemplate, Dashboard, WidgetType, PresetTemplateType, ApiDataSourceConfig, StatCardDisplayConfig, ChartDisplayConfig, TableDisplayConfig, ListDisplayConfig, ProgressDisplayConfig } from "@/types/dashboard";
import { defaultLayoutConfig, widgetDefaultSizes, DashboardStatus } from "@/types/dashboard";

/**
 * 创建Widget配置的辅助函数
 */
function createWidget(
	type: WidgetType,
	title: string,
	endpoint: string,
	x: number,
	y: number,
	extras?: Partial<Dashboard["layout"]["widgets"][0]>
): Dashboard["layout"]["widgets"][0] {
	const size = widgetDefaultSizes[type];

	// 创建API数据源配置
	const dataSource: ApiDataSourceConfig = {
		type: "api",
		endpoint,
		method: "GET",
	};

	// 根据Widget类型创建显示配置
	const createDisplayConfig = (widgetType: WidgetType): StatCardDisplayConfig | ChartDisplayConfig | TableDisplayConfig | ListDisplayConfig | ProgressDisplayConfig => {
		switch (widgetType) {
			case "stat-card":
				return {
					type: "stat-card",
				};
			case "chart":
				return {
					type: "chart",
					chartType: "line",
				};
			case "table":
				return {
					type: "table",
					columns: [],
				};
			case "list":
				return {
					type: "list",
					titleField: "title",
				};
			case "progress":
				return {
					type: "progress",
					progressType: "line",
				};
			default:
				return {
					type: "stat-card",
				};
		}
	};

	return {
		id: `widget-${Date.now()}-${Math.random().toString(36).substring(2, 11)}`,
		type,
		title,
		position: { x, y, ...size },
		dataSource: dataSource,
		display: createDisplayConfig(type),
		enabled: true,
		...extras,
	};
}

/**
 * 运维总览仪表盘模板
 */
export const operationsOverviewTemplate: PresetDashboardTemplate = {
	type: "operations-overview",
	displayName: "运维总览",
	description: "展示系统整体运行状态，包括工单、设备、值班等关键指标",
	dashboard: {
		name: "运维总览",
		description: "展示系统整体运行状态",
		isDefault: false,
		isTemplate: true,
		templateScope: "global",
		layout: {
			...defaultLayoutConfig,
			widgets: [
				// 第一行：统计卡片
				createWidget("stat-card", "待处理工单", "/workorder/statistics", 0, 0),
				createWidget("stat-card", "在线设备数", "/network/devices/list", 6, 0),
				createWidget("stat-card", "今日值班", "/duty/my-duty/stats", 12, 0),
				createWidget("stat-card", "系统通知", "/system/notices", 18, 0),
				// 第二行：图表
				createWidget("chart", "工单趋势", "/workorder/statistics", 0, 3, {
					position: { x: 0, y: 3, w: 12, h: 6 },
				}),
				createWidget("progress", "设备在线率", "/network/devices/list", 12, 3, {
					position: { x: 12, y: 3, w: 6, h: 6 },
				}),
				createWidget("list", "最近工单", "/workorder/orders/my-pending", 18, 3, {
					position: { x: 18, y: 3, w: 6, h: 6 },
				}),
			],
		},
		refreshInterval: 60,
		status: DashboardStatus.Normal,
	},
};

/**
 * 工单管理仪表盘模板
 */
export const workorderManagementTemplate: PresetDashboardTemplate = {
	type: "workorder-management",
	displayName: "工单管理",
	description: "专注于工单管理的仪表盘，展示工单统计、趋势和待办事项",
	dashboard: {
		name: "工单管理",
		description: "工单管理专用仪表盘",
		isDefault: false,
		isTemplate: true,
		templateScope: "global",
		layout: {
			...defaultLayoutConfig,
			widgets: [
				createWidget("stat-card", "今日新增工单", "/workorder/statistics", 0, 0),
				createWidget("stat-card", "待处理工单", "/workorder/statistics", 6, 0),
				createWidget("stat-card", "已完成工单", "/workorder/statistics", 12, 0),
				createWidget("stat-card", "平均处理时长", "/workorder/statistics", 18, 0),
				createWidget("chart", "工单优先级分布", "/workorder/statistics", 0, 3, {
					position: { x: 0, y: 3, w: 8, h: 6 },
				}),
				createWidget("chart", "工单分类统计", "/workorder/statistics", 8, 3, {
					position: { x: 8, y: 3, w: 8, h: 6 },
				}),
				createWidget("chart", "工单趋势图", "/workorder/statistics", 16, 3, {
					position: { x: 16, y: 3, w: 8, h: 6 },
				}),
				createWidget("table", "我的待办工单", "/workorder/orders/my-pending", 0, 9, {
					position: { x: 0, y: 9, w: 24, h: 8 },
				}),
			],
		},
		refreshInterval: 60,
		status: DashboardStatus.Normal,
	},
};

/**
 * 值班管理仪表盘模板
 */
export const dutyManagementTemplate: PresetDashboardTemplate = {
	type: "duty-management",
	displayName: "值班管理",
	description: "展示值班安排、值班统计和值班日历",
	dashboard: {
		name: "值班管理",
		description: "值班管理专用仪表盘",
		isDefault: false,
		isTemplate: true,
		templateScope: "global",
		layout: {
			...defaultLayoutConfig,
			widgets: [
				createWidget("stat-card", "今日值班状态", "/duty/my-duty/stats", 0, 0),
				createWidget("stat-card", "本月值班次数", "/duty/my-duty/stats", 6, 0),
				createWidget("stat-card", "下次值班时间", "/duty/my-duty/stats", 12, 0),
				createWidget("stat-card", "值班人员总数", "/duty/pools/stats", 18, 0),
				createWidget("list", "本周值班安排", "/duty/schedules/week", 0, 3, {
					position: { x: 0, y: 3, w: 12, h: 8 },
				}),
				createWidget("table", "值班统计", "/duty/my-duty/stats", 12, 3, {
					position: { x: 12, y: 3, w: 12, h: 8 },
				}),
			],
		},
		refreshInterval: 60,
		status: DashboardStatus.Normal,
	},
};

/**
 * 系统监控仪表盘模板
 */
export const systemMonitorTemplate: PresetDashboardTemplate = {
	type: "system-monitor",
	displayName: "系统监控",
	description: "实时监控系统性能指标，包括CPU、内存、磁盘等",
	dashboard: {
		name: "系统监控",
		description: "系统监控专用仪表盘",
		isDefault: false,
		isTemplate: true,
		templateScope: "global",
		layout: {
			...defaultLayoutConfig,
			widgets: [
				createWidget("progress", "CPU使用率", "/monitor/server-metrics/current", 0, 0, {
					position: { x: 0, y: 0, w: 6, h: 4 },
				}),
				createWidget("progress", "内存使用率", "/monitor/server-metrics/current", 6, 0, {
					position: { x: 6, y: 0, w: 6, h: 4 },
				}),
				createWidget("progress", "磁盘使用率", "/monitor/server-metrics/current", 12, 0, {
					position: { x: 12, y: 0, w: 6, h: 4 },
				}),
				createWidget("stat-card", "缓存命中率", "/monitor/cache/monitor", 18, 0),
				createWidget("chart", "系统负载历史", "/monitor/server-metrics/history", 0, 4, {
					position: { x: 0, y: 4, w: 24, h: 6 },
				}),
				createWidget("table", "最近登录日志", "/monitor/login-logs", 0, 10, {
					position: { x: 0, y: 10, w: 24, h: 8 },
				}),
			],
		},
		refreshInterval: 30,
		status: DashboardStatus.Normal,
	},
};

/**
 * 所有预设模板列表
 */
export const presetDashboardTemplates: PresetDashboardTemplate[] = [
	operationsOverviewTemplate,
	workorderManagementTemplate,
	dutyManagementTemplate,
	systemMonitorTemplate,
];

/**
 * 根据类型获取模板
 */
export function getPresetTemplate(type: PresetTemplateType): PresetDashboardTemplate | undefined {
	return presetDashboardTemplates.find(t => t.type === type);
}
