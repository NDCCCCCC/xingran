/**
 * 仪表盘系统类型定义
 * Dashboard System Type Definitions
 *
 * 定义仪表盘、Widget、数据源等相关类型
 */

// ============= 仪表盘可见范围 =============

/**
 * 仪表盘可见范围
 */
export type DashboardScope =
	| "private"   // 私有：仅创建者可见
	| "dept"      // 部门：本部门可见
	| "global";   // 全局：全员可见（仅管理员可设置）

// ============= Widget 数据过滤配置 =============

/**
 * Widget 数据过滤配置
 */
export interface WidgetDataFilter {
	/** 是否启用数据过滤 */
	enabled: boolean;
	/** 过滤类型：dept, user, custom */
	filterType: "dept" | "user" | "custom";
	/** 过滤配置 */
	filterConfig: Record<string, string>;
}

// ============= Widget 类型定义 =============

/**
 * Widget 类型枚举
 */
export type WidgetType =
	| "stat-card"      // 统计卡片
	| "chart"          // 图表（折线/柱状/饼图等）
	| "table"          // 表格
	| "list"           // 列表
	| "progress"       // 进度条
	| "metric";        // 指标（圆形进度）

/**
 * 数据源类型枚举
 */
export type DataSourceType =
	| "api"            // REST API
	| "websocket"      // WebSocket 实时推送
	| "static";        // 静态数据

/**
 * 图表类型枚举
 */
export type ChartType =
	| "line"           // 折线图
	| "bar"            // 柱状图
	| "pie"            // 饼图
	| "area"           // 面积图
	| "gauge";         // 仪表盘

/**
 * 数据转换配置（使用 JSONata 表达式）
 */
export interface DataTransformConfig {
	/** JSONata 表达式，用于数据提取和转换 */
	expression?: string;

	/** 聚合函数（sum, avg, count, min, max） */
	aggregate?: "sum" | "avg" | "count" | "min" | "max" | null;

	/** 分组字段 */
	groupBy?: string;

	/** 排序字段和方向 */
	orderBy?: {
		field: string;
		direction: "asc" | "desc";
	};

	/** 限制返回数量 */
	limit?: number;
}

/**
 * API 数据源配置
 */
export interface ApiDataSourceConfig {
	type: "api";

	/** API 端点路径 */
	endpoint: string;

	/** 请求方法 */
	method: "GET" | "POST";

	/** 请求参数（支持变量替换，如 ${userId}） */
	params?: Record<string, unknown>;

	/** 请求体 */
	body?: Record<string, unknown>;

	/** 数据转换配置 */
	transform?: DataTransformConfig;
}

/**
 * WebSocket 数据源配置
 */
export interface WebSocketDataSourceConfig {
	type: "websocket";

	/** WebSocket 频道/主题 */
	channel: string;

	/** 数据转换配置 */
	transform?: DataTransformConfig;
}

/**
 * 静态数据源配置
 */
export interface StaticDataSourceConfig {
	type: "static";

	/** 静态数据 */
	data: unknown;
}

/**
 * 数据源配置（联合类型）
 * 支持直接的数据源配置或包装格式（后端期望的格式）
 */
export type DataSourceConfig =
	| ApiDataSourceConfig
	| WebSocketDataSourceConfig
	| StaticDataSourceConfig
	| { api: ApiDataSourceConfig }
	| { websocket: WebSocketDataSourceConfig }
	| { static: StaticDataSourceConfig };

/**
 * 显示配置基础接口
 */
export interface BaseDisplayConfig {
	/** 主题颜色 */
	color?: string;

	/** 背景色 */
	backgroundColor?: string;

	/** 是否显示边框 */
	showBorder?: boolean;

	/** 是否透明背景 */
	transparent?: boolean;

	/** 自定义 CSS 类名 */
	className?: string;

	/** 自定义样式 */
	style?: Record<string, string>;
}

/**
 * 统计卡片显示配置
 */
export interface StatCardDisplayConfig extends BaseDisplayConfig {
	type: "stat-card";

	/** 数值前缀 */
	prefix?: string;

	/** 数值后缀 */
	suffix?: string;

	/** 小数位数 */
	decimals?: number;

	/** 是否显示百分比 */
	percentage?: boolean;

	/** 趋势方向（基于与前值对比） */
	showTrend?: boolean;

	/** 图标 */
	icon?: string;

	/** 图标颜色 */
	iconColor?: string;
}

/**
 * 图表显示配置
 */
export interface ChartDisplayConfig extends BaseDisplayConfig {
	type: "chart";

	/** 图表类型 */
	chartType: ChartType;

	/** X 轴字段 */
	xField?: string;

	/** Y 轴字段 */
	yField?: string;

	/** 系列字段（用于多系列图表） */
	seriesField?: string;

	/** 颜色配置 */
	colors?: string[];

	/** 是否显示图例 */
	showLegend?: boolean;

	/** 是否显示数据标签 */
	showLabels?: boolean;

	/** 是否平滑曲线（折线图） */
	smooth?: boolean;

	/** 是否显示面积（面积图） */
	showArea?: boolean;

	/** 图表标题 */
	title?: string;
}

/**
 * 表格显示配置
 */
export interface TableDisplayConfig extends BaseDisplayConfig {
	type: "table";

	/** 列配置 */
	columns: Array<{
		/** 数据字段 */
		dataIndex: string;

		/** 列标题 */
		title: string;

		/** 列宽度 */
		width?: number;

		/** 对齐方式 */
		align?: "left" | "center" | "right";

		/** 自定义渲染器 */
		render?: string; // 表达式或函数名

		/** 是否可排序 */
		sortable?: boolean;
	}>;

	/** 是否显示边框 */
	bordered?: boolean;

	/** 表格大小 */
	size?: "small" | "middle" | "large";

	/** 分页配置 */
	pagination?: {
		enabled: boolean;
		pageSize: number;
	};

	/** 行高 */
	rowHeight?: number;
}

/**
 * 列表显示配置
 */
export interface ListDisplayConfig extends BaseDisplayConfig {
	type: "list";

	/** 标题字段 */
	titleField: string;

	/** 描述字段 */
	descriptionField?: string;

	/** 时间字段 */
	timeField?: string;

	/** 图标字段 */
	iconField?: string;

	/** 最大显示数量 */
	maxItems?: number;

	/** 是否显示序号 */
	showIndex?: boolean;
}

/**
 * 进度条显示配置
 */
export interface ProgressDisplayConfig extends BaseDisplayConfig {
	type: "progress";

	/** 进度条类型 */
	progressType: "line" | "circle" | "dashboard";

	/** 目标值（用于计算百分比） */
	target?: number;

	/** 颜色阈值 */
	colorThresholds?: Array<{
		value: number;
		color: string;
	}>;
}

/**
 * 显示配置（联合类型）
 */
export type DisplayConfig =
	| StatCardDisplayConfig
	| ChartDisplayConfig
	| TableDisplayConfig
	| ListDisplayConfig
	| ProgressDisplayConfig;

// ============= Widget 配置 =============

/**
 * Widget 位置配置（用于 react-grid-layout）
 */
export interface WidgetPosition {
	/** 网格 X 坐标 */
	x: number;

	/** 网格 Y 坐标 */
	y: number;

	/** 宽度（以网格列数为单位） */
	w: number;

	/** 高度（以网格行数为单位） */
	h: number;

	/** 最小宽度 */
	minW?: number;

	/** 最小高度 */
	minH?: number;

	/** 最大宽度 */
	maxW?: number;

	/** 最大高度 */
	maxH?: number;
}

/**
 * Widget 配置
 */
export interface WidgetConfig {
	/** Widget 唯一标识 */
	id: string;

	/** Widget 类型 */
	type: WidgetType;

	/** Widget 标题 */
	title: string;

	/** Widget 描述 */
	description?: string;

	/** 位置和大小 */
	position: WidgetPosition;

	/** 数据源配置 */
	dataSource: DataSourceConfig;

	/** 显示配置 */
	display: DisplayConfig;

	/** 数据权限过滤（新增） */
	dataFilter?: WidgetDataFilter;

	/** 刷新间隔（秒），0 表示不自动刷新 */
	refreshInterval?: number;

	/** 是否启用 */
	enabled?: boolean;

	/** Widget 创建时间 */
	createdAt?: string;

	/** Widget 更新时间 */
	updatedAt?: string;
}

// ============= 布局配置 =============

/**
 * 布局配置
 */
export interface LayoutConfig {
	/** Widget 列表 */
	widgets: WidgetConfig[];

	/** 列数（响应式布局） */
	columns: {
		/** 桌面端列数 */
		desktop: number;

		/** 平板端列数 */
		tablet: number;

		/** 移动端列数 */
		mobile: number;
	};

	/** 行高（像素） */
	rowHeight: number;

	/** Widget 间距（像素） */
	margin: [number, number];

	/** 是否允许拖拽 */
	draggable: boolean;

	/** 是否允许调整大小 */
	resizable: boolean;
}

// ============= 仪表盘配置 =============

/**
 * 仪表盘模板作用域
 */
export type TemplateScope =
	| "global"         // 全局模板（所有用户可见）
	| "dept"           // 部门模板
	| "personal";      // 个人模板

/**
 * 仪表盘状态枚举
 */
export enum DashboardStatus {
	/** 正常 */
	Normal = 0,
	/** 停用 */
	Stopped = 1,
}

/**
 * 仪表盘配置
 */
export interface Dashboard {
	/** 仪表盘唯一标识 */
	id: string;

	/** 仪表盘名称 */
	name: string;

	/** 仪表盘描述 */
	description?: string;

	/** 所有者 ID */
	ownerId?: string;

	/** 所有者名称 */
	ownerName?: string;

	/** 创建者部门 ID */
	ownerDeptId?: string;

	/** 是否为默认仪表盘 */
	isDefault: boolean;

	/** 是否为模板 */
	isTemplate: boolean;

	/** 是否为系统仪表盘 */
	isSystem?: boolean;

	/** 仪表盘可见范围（新增） */
	scope?: DashboardScope;

	/** 关联部门 ID（scope=dept 时使用） */
	deptId?: string;

	/** 模板作用域（仅当 isTemplate=true 时有效） */
	templateScope?: TemplateScope;

	/** 布局配置 */
	layout: LayoutConfig;

	/** 刷新间隔（秒） */
	refreshInterval: number;

	/** 状态（0=正常, 1=停用） */
	status: DashboardStatus;

	/** 创建时间 */
	createdAt: string;

	/** 更新时间 */
	updatedAt: string;

	/** 创建者 */
	createdBy?: string;

	/** 更新者 */
	updatedBy?: string;
}

// ============= 预设仪表盘模板 =============

/**
 * 预设仪表盘模板类型
 */
export type PresetTemplateType =
	| "operations-overview"      // 运维总览
	| "workorder-management"     // 工单管理
	| "duty-management"          // 值班管理
	| "system-monitor";          // 系统监控

/**
 * 预设仪表盘模板配置
 */
export interface PresetDashboardTemplate {
	/** 模板类型 */
	type: PresetTemplateType;

	/** 仪表盘配置 */
	dashboard: Omit<Dashboard, "id" | "ownerId" | "createdAt" | "updatedAt">;

	/** 模板显示名称 */
	displayName: string;

	/** 模板描述 */
	description: string;

	/** 模板预览图（可选） */
	previewImage?: string;
}

// ============= API 请求/响应类型 =============

/**
 * 仪表盘列表请求参数
 */
export interface DashboardListParams {
	/** 页码 */
	current: number;

	/** 每页数量 */
	pageSize: number;

	/** 搜索关键词 */
	keyword?: string;

	/** 是否只显示模板 */
	isTemplate?: boolean;

	/** 状态筛选 */
	status?: DashboardStatus;
}

/**
 * 仪表盘列表响应
 */
export interface DashboardListResponse {
	/** 仪表盘列表 */
	list: Dashboard[];

	/** 总数 */
	total: number;

	/** 当前页 */
	current: number;

	/** 每页数量 */
	pageSize: number;
}

/**
 * 创建仪表盘请求
 */
export interface CreateDashboardRequest {
	/** 仪表盘名称 */
	name: string;

	/** 仪表盘描述 */
	description?: string;

	/** 布局配置 */
	layout: LayoutConfig;

	/** 刷新间隔 */
	refreshInterval?: number;

	/** 是否为模板 */
	isTemplate?: boolean;

	/** 模板作用域 */
	templateScope?: TemplateScope;

	/** 仪表盘可见范围（新增） */
	scope?: DashboardScope;

	/** 关联部门 ID（scope=dept 时使用） */
	deptId?: string;

	/** 是否为系统仪表盘 */
	isSystem?: boolean;
}

/**
 * 更新仪表盘请求
 */
export interface UpdateDashboardRequest {
	/** 仪表盘名称 */
	name?: string;

	/** 仪表盘描述 */
	description?: string;

	/** 布局配置 */
	layout?: LayoutConfig;

	/** 刷新间隔 */
	refreshInterval?: number;

	/** 状态 */
	status?: DashboardStatus;
}

/**
 * 仪表盘版本记录
 */
export interface DashboardVersion {
	/** 版本 ID */
	id: string;

	/** 仪表盘 ID */
	dashboardId: string;

	/** 布局配置快照 */
	layout: LayoutConfig;

	/** 版本备注 */
	comment?: string;

	/** 创建时间 */
	createdAt: string;

	/** 创建者 */
	createdBy: string;
}

// ============= Widget 数据类型 =============

/**
 * Widget 数据响应
 */
export interface WidgetDataResponse<T = unknown> {
	/** Widget ID */
	widgetId: string;

	/** 数据 */
	data: T;

	/** 更新时间戳 */
	timestamp: number;

	/** 是否有错误 */
	error?: string;
}

// ============= 默认配置 =============

/**
 * 默认布局配置
 */
export const defaultLayoutConfig: LayoutConfig = {
	widgets: [],
	columns: {
		desktop: 24,
		tablet: 12,
		mobile: 6,
	},
	rowHeight: 60,
	margin: [16, 16],
	draggable: true,
	resizable: true,
};

/**
 * 默认 Widget 位置
 */
export const defaultWidgetPosition: WidgetPosition = {
	x: 0,
	y: 0,
	w: 6,
	h: 4,
	minW: 3,
	minH: 3,
};

/**
 * Widget 默认尺寸映射
 */
export const widgetDefaultSizes: Record<WidgetType, Pick<WidgetPosition, "w" | "h">> = {
	"stat-card": { w: 6, h: 3 },
	"chart": { w: 12, h: 6 },
	"table": { w: 24, h: 8 },
	"list": { w: 8, h: 6 },
	"progress": { w: 6, h: 4 },
	"metric": { w: 6, h: 4 },
};

// ============= API 端点元数据类型 =============

/**
 * 端点详情
 */
export interface EndpointDetail {
	/** 端点路径 */
	route: string;

	/** 请求方法 */
	method: string;

	/** 显示名称 */
	displayName: string;

	/** 描述 */
	description: string;

	/** 所属模块 */
	module: string;

	/** 分类名称 */
	category: string;

	/** 图标 */
	icon: string;

	/** 数据类型（paginated/single） */
	dataType: "paginated" | "single";

	/** 数据路径（JSONata表达式） */
	dataPath: string;

	/** 支持的Widget类型 */
	supportedWidgets: WidgetType[];

	/** 示例参数 */
	exampleParams?: Record<string, string>;

	/** 所需权限 */
	requiredPerms: string[];
}

/**
 * 分类端点列表
 */
export interface EndpointCategory {
	/** 模块标识 */
	module: string;

	/** 分类名称 */
	category: string;

	/** 图标 */
	icon: string;

	/** 端点列表 */
	endpoints: EndpointDetail[];
}

/**
 * 端点列表响应
 */
export interface EndpointsResponse {
	/** 分类端点列表 */
	categories: EndpointCategory[];

	/** 总数 */
	total: number;
}
