/**
 * 仪表盘状态管理
 * Dashboard State Management
 *
 * 管理仪表盘列表、当前仪表盘、编辑状态等
 */

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type {
	Dashboard,
	WidgetConfig,
	LayoutConfig,
	CreateDashboardRequest,
	UpdateDashboardRequest,
	DashboardListParams,
} from '@/types/dashboard';
import { dashboardService } from '@/services/dashboardService';

// ============= 编辑模式 =============

/**
 * 仪表盘视图模式
 */
export type DashboardViewMode = 'view' | 'edit';

// ============= 状态定义 =============

interface DashboardState {
	// ========== 仪表盘列表 ==========
	/** 仪表盘列表 */
	dashboards: Dashboard[];

	/** 列表加载状态 */
	listLoading: boolean;

	/** 列表错误 */
	listError: string | null;

	/** 列表分页信息 */
	listPagination: {
		total: number;
		current: number;
		pageSize: number;
	};

	// ========== 当前仪表盘 ==========
	/** 当前查看/编辑的仪表盘 */
	currentDashboard: Dashboard | null;

	/** 当前仪表盘加载状态 */
	currentLoading: boolean;

	/** 当前仪表盘错误 */
	currentError: string | null;

	// ========== 编辑状态 ==========
	/** 视图模式 */
	viewMode: DashboardViewMode;

	/** 布局是否已修改（未保存） */
	hasUnsavedChanges: boolean;

	/** 选中的 Widget ID（用于编辑） */
	selectedWidgetId: string | null;

	/** 正在拖拽的 Widget ID */
	draggingWidgetId: string | null;

	// ========== 缓存 ==========
	/** Widget 数据缓存 */
	widgetDataCache: Map<string, {
		data: unknown;
		timestamp: number;
	}>;

	// ========== 全局设置 ==========
	/** 是否显示网格线 */
	showGridLines: boolean;

	/** 是否显示 Widget 边框 */
	showWidgetBorders: boolean;
}

// ============= 操作定义 =============

interface DashboardActions {
	// ========== 仪表盘列表操作 ==========
	/**
	 * 获取仪表盘列表
	 */
	fetchDashboards: (params?: DashboardListParams) => Promise<void>;

	/**
	 * 创建仪表盘
	 */
	createDashboard: (data: CreateDashboardRequest) => Promise<Dashboard>;

	/**
	 * 更新仪表盘
	 */
	updateDashboard: (id: string, data: UpdateDashboardRequest) => Promise<void>;

	/**
	 * 删除仪表盘
	 */
	deleteDashboard: (id: string) => Promise<void>;

	/**
	 * 复制仪表盘
	 */
	duplicateDashboard: (id: string) => Promise<Dashboard>;

	/**
	 * 设置默认仪表盘
	 */
	setDefaultDashboard: (id: string) => Promise<void>;

	// ========== 当前仪表盘操作 ==========
	/**
	 * 获取仪表盘详情
	 */
	fetchDashboard: (id: string) => Promise<void>;

	/**
	 * 设置当前仪表盘（本地状态，不从服务器加载）
	 */
	setCurrentDashboard: (dashboard: Dashboard | null) => void;

	/**
	 * 清除当前仪表盘
	 */
	clearCurrentDashboard: () => void;

	// ========== 编辑操作 ==========
	/**
	 * 设置视图模式
	 */
	setViewMode: (mode: DashboardViewMode) => void;

	/**
	 * 添加 Widget
	 */
	addWidget: (widget: WidgetConfig) => void;

	/**
	 * 更新 Widget
	 */
	updateWidget: (widgetId: string, updates: Partial<WidgetConfig>) => void;

	/**
	 * 删除 Widget
	 */
	removeWidget: (widgetId: string) => void;

	/**
	 * 批量更新 Widget 布局
	 */
	updateWidgetLayouts: (layouts: Array<{ id: string; position: WidgetConfig['position'] }>) => void;

	/**
	 * 更新布局配置
	 */
	updateLayout: (layout: Partial<LayoutConfig>) => void;

	/**
	 * 保存当前仪表盘
	 */
	saveCurrentDashboard: () => Promise<void>;

	/**
	 * 重置当前仪表盘（撤销未保存的更改）
	 */
	resetCurrentDashboard: () => Promise<void>;

	// ========== 选择操作 ==========
	/**
	 * 选中 Widget
	 */
	selectWidget: (widgetId: string | null) => void;

	/**
	 * 开始拖拽 Widget
	 */
	startDragging: (widgetId: string) => void;

	/**
	 * 停止拖拽 Widget
	 */
	stopDragging: () => void;

	// ========== 缓存操作 ==========
	/**
	 * 缓存 Widget 数据
	 */
	cacheWidgetData: (widgetId: string, data: unknown) => void;

	/**
	 * 获取缓存的 Widget 数据
	 */
	getCachedWidgetData: (widgetId: string) => unknown | null;

	/**
	 * 清除 Widget 数据缓存
	 */
	clearWidgetCache: (widgetId?: string) => void;

	// ========== 全局设置操作 ==========
	/**
	 * 切换网格线显示
	 */
	toggleGridLines: () => void;

	/**
	 * 切换 Widget 边框显示
	 */
	toggleWidgetBorders: () => void;

	/**
	 * 重置所有状态
	 */
	reset: () => void;
}

// ============= Store 类型 =============

type DashboardStore = DashboardState & DashboardActions;

// ============= 初始状态 =============

const initialState: DashboardState = {
	dashboards: [],
	listLoading: false,
	listError: null,
	listPagination: {
		total: 0,
		current: 1,
		pageSize: 10,
	},
	currentDashboard: null,
	currentLoading: false,
	currentError: null,
	viewMode: 'view',
	hasUnsavedChanges: false,
	selectedWidgetId: null,
	draggingWidgetId: null,
	widgetDataCache: new Map(),
	showGridLines: false,
	showWidgetBorders: false,
};

// ============= Store 创建 =============

export const useDashboardStore = create<DashboardStore>()(
	persist(
		(set, get) => ({
			// ========== 初始状态 ==========
			...initialState,

			// ========== 仪表盘列表操作 ==========
			fetchDashboards: async (params) => {
				set({ listLoading: true, listError: null });

				try {
					const response = await dashboardService.getDashboards(
						params || {
							current: get().listPagination.current,
							pageSize: get().listPagination.pageSize,
						}
					);

					set({
						dashboards: response.list,
						listPagination: {
							total: response.total,
							current: response.current,
							pageSize: response.pageSize,
						},
						listLoading: false,
					});
				} catch (error) {
					set({
						listError: (error as Error).message,
						listLoading: false,
					});
					throw error;
				}
			},

			createDashboard: async (data) => {
				const dashboard = await dashboardService.createDashboard(data);

				// 添加到列表
				set((state) => ({
					dashboards: [dashboard, ...state.dashboards],
					listPagination: {
						...state.listPagination,
						total: state.listPagination.total + 1,
					},
				}));

				return dashboard;
			},

			updateDashboard: async (id, data) => {
				await dashboardService.updateDashboard(id, data);

				// 更新列表中的仪表盘
				set((state) => ({
					dashboards: state.dashboards.map((d) =>
						d.id === id ? { ...d, ...data } : d
					),
				}));

				// 如果是当前仪表盘，也更新它
				const { currentDashboard } = get();
				if (currentDashboard?.id === id) {
					set({
						currentDashboard: {
							...currentDashboard,
							...data,
						},
						hasUnsavedChanges: false,
					});
				}
			},

			deleteDashboard: async (id) => {
				await dashboardService.deleteDashboard(id);

				// 从列表中移除
				set((state) => ({
					dashboards: state.dashboards.filter((d) => d.id !== id),
					listPagination: {
						...state.listPagination,
						total: state.listPagination.total - 1,
					},
					currentDashboard: state.currentDashboard?.id === id ? null : state.currentDashboard,
				}));
			},

			duplicateDashboard: async (id) => {
				const dashboard = await dashboardService.duplicateDashboard(id);

				// 添加到列表
				set((state) => ({
					dashboards: [dashboard, ...state.dashboards],
					listPagination: {
						...state.listPagination,
						total: state.listPagination.total + 1,
					},
				}));

				return dashboard;
			},

			setDefaultDashboard: async (id) => {
				await dashboardService.setDefaultDashboard(id);

				// 更新列表
				set((state) => ({
					dashboards: state.dashboards.map((d) => ({
						...d,
						isDefault: d.id === id,
					})),
				}));
			},

			// ========== 当前仪表盘操作 ==========
			fetchDashboard: async (id) => {
				set({ currentLoading: true, currentError: null });

				try {
					const dashboard = await dashboardService.getDashboard(id);
					set({
						currentDashboard: dashboard,
						currentLoading: false,
						hasUnsavedChanges: false,
					});
				} catch (error) {
					set({
						currentError: (error as Error).message,
						currentLoading: false,
					});
					throw error;
				}
			},

			setCurrentDashboard: (dashboard) => {
				set({
					currentDashboard: dashboard,
					hasUnsavedChanges: false,
				});
			},

			clearCurrentDashboard: () => {
				set({
					currentDashboard: null,
					currentError: null,
					hasUnsavedChanges: false,
					selectedWidgetId: null,
				});
			},

			// ========== 编辑操作 ==========
			setViewMode: (mode) => {
				set({ viewMode: mode });
			},

			addWidget: (widget) => {
				set((state) => {
					if (!state.currentDashboard) return state;

					const updatedDashboard = {
						...state.currentDashboard,
						layout: {
							...state.currentDashboard.layout,
							widgets: [...state.currentDashboard.layout.widgets, widget],
						},
					};

					return {
						currentDashboard: updatedDashboard,
						hasUnsavedChanges: true,
					};
				});
			},

			updateWidget: (widgetId, updates) => {
				set((state) => {
					if (!state.currentDashboard) return state;

					const updatedDashboard = {
						...state.currentDashboard,
						layout: {
							...state.currentDashboard.layout,
							widgets: state.currentDashboard.layout.widgets.map((w) =>
								w.id === widgetId ? { ...w, ...updates } : w
							),
						},
					};

					return {
						currentDashboard: updatedDashboard,
						hasUnsavedChanges: true,
					};
				});
			},

			removeWidget: (widgetId) => {
				set((state) => {
					if (!state.currentDashboard) return state;

					const updatedDashboard = {
						...state.currentDashboard,
						layout: {
							...state.currentDashboard.layout,
							widgets: state.currentDashboard.layout.widgets.filter(
								(w) => w.id !== widgetId
							),
						},
					};

					return {
						currentDashboard: updatedDashboard,
						hasUnsavedChanges: true,
						selectedWidgetId:
							state.selectedWidgetId === widgetId ? null : state.selectedWidgetId,
					};
				});
			},

			updateWidgetLayouts: (layouts) => {
				set((state) => {
					if (!state.currentDashboard) return state;

					const layoutMap = new Map(layouts.map((l) => [l.id, l.position]));

					const updatedDashboard = {
						...state.currentDashboard,
						layout: {
							...state.currentDashboard.layout,
							widgets: state.currentDashboard.layout.widgets.map((w) =>
								layoutMap.has(w.id)
									? { ...w, position: layoutMap.get(w.id)! }
									: w
							),
						},
					};

					return {
						currentDashboard: updatedDashboard,
						hasUnsavedChanges: true,
					};
				});
			},

			updateLayout: (layoutUpdates) => {
				set((state) => {
					if (!state.currentDashboard) return state;

					const updatedDashboard = {
						...state.currentDashboard,
						layout: {
							...state.currentDashboard.layout,
							...layoutUpdates,
						},
					};

					return {
						currentDashboard: updatedDashboard,
						hasUnsavedChanges: true,
					};
				});
			},

			saveCurrentDashboard: async () => {
				const { currentDashboard } = get();
				if (!currentDashboard) return;

				await get().updateDashboard(currentDashboard.id, {
					name: currentDashboard.name,
					description: currentDashboard.description,
					layout: currentDashboard.layout,
					refreshInterval: currentDashboard.refreshInterval,
				});
			},

			resetCurrentDashboard: async () => {
				const { currentDashboard } = get();
				if (!currentDashboard) return;

				await get().fetchDashboard(currentDashboard.id);
			},

			// ========== 选择操作 ==========
			selectWidget: (widgetId) => {
				set({ selectedWidgetId: widgetId });
			},

			startDragging: (widgetId) => {
				set({ draggingWidgetId: widgetId });
			},

			stopDragging: () => {
				set({ draggingWidgetId: null });
			},

			// ========== 缓存操作 ==========
			cacheWidgetData: (widgetId, data) => {
				set((state) => {
					const newCache = new Map(state.widgetDataCache);
					newCache.set(widgetId, {
						data,
						timestamp: Date.now(),
					});
					return { widgetDataCache: newCache };
				});
			},

			getCachedWidgetData: (widgetId) => {
				const cached = get().widgetDataCache.get(widgetId);
				if (!cached) return null;

				const cacheExpiry = 5 * 60 * 1000;
				if (Date.now() - cached.timestamp > cacheExpiry) {
					get().clearWidgetCache(widgetId);
					return null;
				}

				return cached.data;
			},

			clearWidgetCache: (widgetId) => {
				set((state) => {
					const newCache = new Map(state.widgetDataCache);
					if (widgetId) {
						newCache.delete(widgetId);
					} else {
						newCache.clear();
					}
					return { widgetDataCache: newCache };
				});
			},

			// ========== 全局设置操作 ==========
			toggleGridLines: () => {
				set((state) => ({ showGridLines: !state.showGridLines }));
			},

			toggleWidgetBorders: () => {
				set((state) => ({ showWidgetBorders: !state.showWidgetBorders }));
			},

			// ========== 重置 ==========
			reset: () => {
				set(initialState);
			},
		}),
		{
			name: 'dashboard-storage',
			// 只持久化部分状态
			partialize: (state) => ({
				viewMode: state.viewMode,
				showGridLines: state.showGridLines,
				showWidgetBorders: state.showWidgetBorders,
			}),
		}
	)
);

// ============= 选择器（用于避免不必要的重渲染） =============

export const selectCurrentDashboard = (state: DashboardStore) => state.currentDashboard;
export const selectViewMode = (state: DashboardStore) => state.viewMode;
export const selectHasUnsavedChanges = (state: DashboardStore) => state.hasUnsavedChanges;
export const selectSelectedWidgetId = (state: DashboardStore) => state.selectedWidgetId;
export const selectCurrentWidgets = (state: DashboardStore) =>
	state.currentDashboard?.layout.widgets ?? [];
export const selectIsDragging = (state: DashboardStore) => state.draggingWidgetId !== null;
