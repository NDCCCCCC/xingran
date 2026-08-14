import { create } from "zustand";
import { persist } from "zustand/middleware";
import type {
  Dashboard,
  WidgetConfig,
  LayoutConfig,
  CreateDashboardRequest,
  UpdateDashboardRequest,
  DashboardListParams,
} from "@/types/dashboard";
import { dashboardService } from "@/services/dashboardService";

export type DashboardViewMode = "view" | "edit";

// 页面模式类型
export type PageMode = "home" | "list" | "view" | "edit";

// ==================== 辅助函数 ====================

// 创建更新后的仪表盘对象
const createUpdatedDashboard = (dashboard: Dashboard, updates: Partial<Dashboard>): Dashboard => ({
  ...dashboard,
  ...updates,
});

// 更新仪表盘列表中的项
const updateDashboardInList = (
  dashboards: Dashboard[],
  id: string,
  updates: Partial<Dashboard>
): Dashboard[] => dashboards.map((d) => (d.id === id ? createUpdatedDashboard(d, updates) : d));

// 更新当前仪表盘的widgets
const updateWidgets = (
  layout: LayoutConfig,
  updater: (widgets: WidgetConfig[]) => WidgetConfig[]
): LayoutConfig => ({
  ...layout,
  widgets: updater(layout.widgets),
});

// 更新单个widget
const updateSingleWidget = (
  widgets: WidgetConfig[],
  widgetId: string,
  updates: Partial<WidgetConfig>
): WidgetConfig[] => widgets.map((w) => (w.id === widgetId ? { ...w, ...updates } : w));

// WebSocket 连接状态类型
export type WebSocketStatus = "connecting" | "connected" | "disconnected" | "error";

// ==================== 类型定义 ====================

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
  widgetDataCache: Map<
    string,
    {
      data: unknown;
      timestamp: number;
    }
  >;
  showGridLines: boolean;
  showWidgetBorders: boolean;
  // 新增状态
  pageMode: PageMode;
  defaultDashboard: Dashboard | null;
  defaultDashboardLoading: boolean;
  // WebSocket 和实时刷新状态
  wsStatus: WebSocketStatus;
  wsLastConnected: Date | null;
  isRefreshing: boolean;
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
  updateWidgetLayouts: (layouts: Array<{ id: string; position: WidgetConfig["position"] }>) => void;
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
  // 新增 actions
  fetchDefaultDashboard: () => Promise<void>;
  setPageMode: (mode: PageMode) => void;
  // WebSocket 和实时刷新 actions
  setWsStatus: (status: WebSocketStatus) => void;
  setIsRefreshing: (refreshing: boolean) => void;
  updateWidgetData: (widgetId: string, data: unknown) => void;
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
  viewMode: "view",
  hasUnsavedChanges: false,
  selectedWidgetId: null,
  draggingWidgetId: null,
  widgetDataCache: new Map(),
  showGridLines: false,
  showWidgetBorders: false,
  // 新增状态初始值
  pageMode: "home",
  defaultDashboard: null,
  defaultDashboardLoading: false,
  // WebSocket 和实时刷新状态初始值
  wsStatus: "disconnected",
  wsLastConnected: null,
  isRefreshing: false,
};

export const useDashboardStore = create<DashboardStore>()(
  persist(
    (set, get) => ({
      ...initialState,

      fetchDashboards: async (params) => {
        set({ listLoading: true, listError: null });

        try {
          const defaultParams = {
            current: get().listPagination.current,
            pageSize: get().listPagination.pageSize,
          };
          const response = await dashboardService.getDashboards(params ?? defaultParams);

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
          dashboards: updateDashboardInList(state.dashboards, id, data),
          currentDashboard:
            state.currentDashboard?.id === id
              ? createUpdatedDashboard(state.currentDashboard, data)
              : state.currentDashboard,
          hasUnsavedChanges: false,
        }));
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
          dashboards: state.dashboards.map((d) =>
            createUpdatedDashboard(d, { isDefault: d.id === id })
          ),
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

          return {
            currentDashboard: createUpdatedDashboard(state.currentDashboard, {
              layout: updateWidgets(state.currentDashboard.layout, (widgets) => [
                ...widgets,
                widget,
              ]),
            }),
            hasUnsavedChanges: true,
          };
        });
      },

      updateWidget: (widgetId, updates) => {
        set((state) => {
          if (!state.currentDashboard) return state;

          return {
            currentDashboard: createUpdatedDashboard(state.currentDashboard, {
              layout: updateWidgets(state.currentDashboard.layout, (widgets) =>
                updateSingleWidget(widgets, widgetId, updates)
              ),
            }),
            hasUnsavedChanges: true,
          };
        });
      },

      removeWidget: (widgetId) => {
        set((state) => {
          if (!state.currentDashboard) return state;

          return {
            currentDashboard: createUpdatedDashboard(state.currentDashboard, {
              layout: updateWidgets(state.currentDashboard.layout, (widgets) =>
                widgets.filter((w) => w.id !== widgetId)
              ),
            }),
            hasUnsavedChanges: true,
            selectedWidgetId: state.selectedWidgetId === widgetId ? null : state.selectedWidgetId,
          };
        });
      },

      updateWidgetLayouts: (layouts) => {
        set((state) => {
          if (!state.currentDashboard) return state;

          const layoutMap = new Map(layouts.map((l) => [l.id, l.position]));

          return {
            currentDashboard: createUpdatedDashboard(state.currentDashboard, {
              layout: updateWidgets(state.currentDashboard.layout, (widgets) =>
                widgets.map((w) =>
                  layoutMap.has(w.id) ? { ...w, position: layoutMap.get(w.id)! } : w
                )
              ),
            }),
            hasUnsavedChanges: true,
          };
        });
      },

      updateLayout: (layoutUpdates) => {
        set((state) => {
          if (!state.currentDashboard) return state;

          return {
            currentDashboard: createUpdatedDashboard(state.currentDashboard, {
              layout: {
                ...state.currentDashboard.layout,
                ...layoutUpdates,
              },
            }),
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

      // 新增 actions
      fetchDefaultDashboard: async () => {
        set({ defaultDashboardLoading: true });
        try {
          const dashboard = await dashboardService.getDefaultDashboard();
          set({ defaultDashboard: dashboard, defaultDashboardLoading: false });
        } catch (error) {
          set({ defaultDashboard: null, defaultDashboardLoading: false });
          throw error;
        }
      },

      setPageMode: (mode) => {
        set({ pageMode: mode });
      },

      // WebSocket 和实时刷新 actions
      setWsStatus: (status) => {
        set((state) => ({
          wsStatus: status,
          wsLastConnected: status === "connected" ? new Date() : state.wsLastConnected,
        }));
      },

      setIsRefreshing: (refreshing) => {
        set({ isRefreshing: refreshing });
      },

      updateWidgetData: (widgetId, data) => {
        set((state) => {
          const newCache = new Map(state.widgetDataCache);
          newCache.set(widgetId, {
            data,
            timestamp: Date.now(),
          });
          return { widgetDataCache: newCache };
        });
      },

      reset: () => {
        set(initialState);
      },
    }),
    {
      name: "dashboard-storage",
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
