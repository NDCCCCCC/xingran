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

export type DashboardViewMode = 'view' | 'edit';

interface DashboardState {
	dashboards: Dashboard[];
	listLoading: boolean;
	listError: string | null;
	listPagination: {
		total: number;
		current: number;
		pageSize: number;
	};
	currentDashboard: Dashboard | null;
	currentLoading: boolean;
	currentError: string | null;
	viewMode: DashboardViewMode;
	hasUnsavedChanges: boolean;
	selectedWidgetId: string | null;
	draggingWidgetId: string | null;
	widgetDataCache: Map<string, {
		data: unknown;
		timestamp: number;
	}>;
	showGridLines: boolean;
	showWidgetBorders: boolean;
}

interface DashboardActions {
	fetchDashboards: (params?: DashboardListParams) => Promise<void>;
	createDashboard: (data: CreateDashboardRequest) => Promise<Dashboard>;
	updateDashboard: (id: string, data: UpdateDashboardRequest) => Promise<void>;
	deleteDashboard: (id: string) => Promise<void>;
	duplicateDashboard: (id: string) => Promise<Dashboard>;
	setDefaultDashboard: (id: string) => Promise<void>;
	fetchDashboard: (id: string) => Promise<void>;
	setCurrentDashboard: (dashboard: Dashboard | null) => void;
	clearCurrentDashboard: () => void;
	setViewMode: (mode: DashboardViewMode) => void;
	addWidget: (widget: WidgetConfig) => void;
	updateWidget: (widgetId: string, updates: Partial<WidgetConfig>) => void;
	removeWidget: (widgetId: string) => void;
	updateWidgetLayouts: (layouts: Array<{ id: string; position: WidgetConfig['position'] }>) => void;
	updateLayout: (layout: Partial<LayoutConfig>) => void;
	saveCurrentDashboard: () => Promise<void>;
	resetCurrentDashboard: () => Promise<void>;
	selectWidget: (widgetId: string | null) => void;
	startDragging: (widgetId: string) => void;
	stopDragging: () => void;
	cacheWidgetData: (widgetId: string, data: unknown) => void;
	getCachedWidgetData: (widgetId: string) => unknown | null;
	clearWidgetCache: (widgetId?: string) => void;
	toggleGridLines: () => void;
	toggleWidgetBorders: () => void;
	reset: () => void;
}

type DashboardStore = DashboardState & DashboardActions;

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

export const useDashboardStore = create<DashboardStore>()(
	persist(
		(set, get) => ({
			...initialState,

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

				set((state) => ({
					dashboards: state.dashboards.map((d) =>
						d.id === id ? { ...d, ...data } : d
					),
				}));

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

				set((state) => ({
					dashboards: state.dashboards.map((d) => ({
						...d,
						isDefault: d.id === id,
					})),
				}));
			},

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

			selectWidget: (widgetId) => {
				set({ selectedWidgetId: widgetId });
			},

			startDragging: (widgetId) => {
				set({ draggingWidgetId: widgetId });
			},

			stopDragging: () => {
				set({ draggingWidgetId: null });
			},

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

			toggleGridLines: () => {
				set((state) => ({ showGridLines: !state.showGridLines }));
			},

			toggleWidgetBorders: () => {
				set((state) => ({ showWidgetBorders: !state.showWidgetBorders }));
			},

			reset: () => {
				set(initialState);
			},
		}),
		{
			name: 'dashboard-storage',
			partialize: (state) => ({
				viewMode: state.viewMode,
				showGridLines: state.showGridLines,
				showWidgetBorders: state.showWidgetBorders,
			}),
		}
	)
);

export const selectCurrentDashboard = (state: DashboardStore) => state.currentDashboard;
export const selectViewMode = (state: DashboardStore) => state.viewMode;
export const selectHasUnsavedChanges = (state: DashboardStore) => state.hasUnsavedChanges;
export const selectSelectedWidgetId = (state: DashboardStore) => state.selectedWidgetId;
export const selectCurrentWidgets = (state: DashboardStore) =>
	state.currentDashboard?.layout.widgets ?? [];
export const selectIsDragging = (state: DashboardStore) => state.draggingWidgetId !== null;
