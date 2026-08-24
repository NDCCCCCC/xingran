/**
 * dashboardStore 仪表盘状态测试
 *
 * 覆盖:列表 CRUD(fetch/create/update/delete/duplicate/setDefault)、
 * 当前仪表盘(fetch/set/clear/save/reset)、widget 增删改布局
 * (addWidget/updateWidget/removeWidget/updateWidgetLayouts/updateLayout)、
 * 缓存(cacheWidgetData/getCachedWidgetData 过期/clearWidgetCache)、
 * UI 开关、fetchDefaultDashboard、setPageMode、WS 状态、reset、persist。
 */
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

const svc = vi.hoisted(() => ({
  getDashboards: vi.fn(),
  getDashboard: vi.fn(),
  createDashboard: vi.fn(),
  updateDashboard: vi.fn(),
  deleteDashboard: vi.fn(),
  duplicateDashboard: vi.fn(),
  setDefaultDashboard: vi.fn(),
  getDefaultDashboard: vi.fn(),
}));
vi.mock("@/services/dashboardService", () => ({
  dashboardService: svc,
}));

import { useDashboardStore } from "./dashboardStore";
import type { Dashboard, WidgetConfig } from "@/types/dashboard";

const widget = (id: string, overrides: Partial<WidgetConfig> = {}): WidgetConfig =>
  ({
    id,
    type: "stat-card",
    title: id,
    position: { x: 0, y: 0, w: 4, h: 3 },
    dataSource: { type: "static", data: null },
    display: { type: "stat-card" },
    ...overrides,
  }) as WidgetConfig;

const dashboard = (id: string, widgets: WidgetConfig[] = []): Dashboard =>
  ({
    id,
    name: `board-${id}`,
    description: "",
    layout: { widgets },
    refreshInterval: 60,
    isDefault: false,
  }) as Dashboard;

describe("dashboardStore", () => {
  let consoleSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    localStorage.clear();
    vi.clearAllMocks();
    useDashboardStore.getState().reset();
    useDashboardStore.setState({ widgetDataCache: new Map() });
    consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    consoleSpy.mockRestore();
    vi.useRealTimers();
  });

  it("fetchDashboards 写入列表与分页;失败置 listError 并抛出", async () => {
    svc.getDashboards.mockResolvedValue({
      list: [dashboard("d1")],
      total: 1,
      current: 1,
      pageSize: 10,
    });
    await useDashboardStore.getState().fetchDashboards();

    const state = useDashboardStore.getState();
    expect(state.dashboards).toHaveLength(1);
    expect(state.listPagination).toEqual({ total: 1, current: 1, pageSize: 10 });
    expect(state.listLoading).toBe(false);

    svc.getDashboards.mockRejectedValue(new Error("list fail"));
    await expect(useDashboardStore.getState().fetchDashboards()).rejects.toThrow("list fail");
    expect(useDashboardStore.getState().listError).toBe("list fail");
    expect(useDashboardStore.getState().listLoading).toBe(false);
  });

  it("fetchDashboards 无参数时使用当前分页作为默认", async () => {
    useDashboardStore.getState().reset();
    svc.getDashboards.mockResolvedValue({
      list: [],
      total: 0,
      current: 2,
      pageSize: 20,
    });
    await useDashboardStore.getState().fetchDashboards();
    expect(svc.getDashboards).toHaveBeenCalledWith({ current: 1, pageSize: 10 });
  });

  it("createDashboard 前插列表并 total+1", async () => {
    useDashboardStore.setState({ dashboards: [dashboard("old")] });
    svc.createDashboard.mockResolvedValue(dashboard("new"));

    const created = await useDashboardStore.getState().createDashboard({
      name: "new",
    } as never);

    expect(created.id).toBe("new");
    const state = useDashboardStore.getState();
    expect(state.dashboards.map((d) => d.id)).toEqual(["new", "old"]);
    expect(state.listPagination.total).toBe(1);
  });

  it("updateDashboard 同步列表与当前仪表盘,清 hasUnsavedChanges", async () => {
    useDashboardStore.setState({
      dashboards: [dashboard("d1"), dashboard("d2")],
      currentDashboard: dashboard("d1"),
      hasUnsavedChanges: true,
    });
    svc.updateDashboard.mockResolvedValue(undefined);

    await useDashboardStore.getState().updateDashboard("d1", {
      name: "renamed",
    } as never);

    const state = useDashboardStore.getState();
    expect(state.dashboards[0].name).toBe("renamed");
    expect(state.dashboards[1].name).toBe("board-d2");
    expect(state.currentDashboard?.name).toBe("renamed");
    expect(state.hasUnsavedChanges).toBe(false);
  });

  it("deleteDashboard 移除列表/当前仪表盘,total-1", async () => {
    useDashboardStore.setState({
      dashboards: [dashboard("d1"), dashboard("d2")],
      currentDashboard: dashboard("d2"),
    });
    svc.deleteDashboard.mockResolvedValue(undefined);

    await useDashboardStore.getState().deleteDashboard("d2");

    const state = useDashboardStore.getState();
    expect(state.dashboards.map((d) => d.id)).toEqual(["d1"]);
    expect(state.currentDashboard).toBeNull();
    expect(state.listPagination.total).toBe(-1); // 0-1
  });

  it("duplicateDashboard 复制前插;setDefaultDashboard 更新默认标记", async () => {
    useDashboardStore.setState({
      dashboards: [dashboard("d1"), dashboard("d2")],
    });
    svc.duplicateDashboard.mockResolvedValue(dashboard("copy"));
    svc.setDefaultDashboard.mockResolvedValue(undefined);

    const dup = await useDashboardStore.getState().duplicateDashboard("d1");
    expect(dup.id).toBe("copy");
    expect(useDashboardStore.getState().dashboards[0].id).toBe("copy");

    await useDashboardStore.getState().setDefaultDashboard("d2");
    const state = useDashboardStore.getState();
    expect(state.dashboards.find((d) => d.id === "d2")!.isDefault).toBe(true);
    expect(state.dashboards.find((d) => d.id === "d1")!.isDefault).toBe(false);
  });

  it("fetchDashboard 写当前仪表盘;失败置 currentError 并抛出", async () => {
    svc.getDashboard.mockResolvedValue(dashboard("d1", [widget("w1")]));
    await useDashboardStore.getState().fetchDashboard("d1");

    const ok = useDashboardStore.getState();
    expect(ok.currentDashboard?.id).toBe("d1");
    expect(ok.currentLoading).toBe(false);
    expect(ok.hasUnsavedChanges).toBe(false);

    svc.getDashboard.mockRejectedValue(new Error("detail fail"));
    await expect(useDashboardStore.getState().fetchDashboard("d1")).rejects.toThrow("detail fail");
    expect(useDashboardStore.getState().currentError).toBe("detail fail");
  });

  it("setCurrentDashboard / clearCurrentDashboard", () => {
    const d = dashboard("d1");
    useDashboardStore.setState({ hasUnsavedChanges: true });

    useDashboardStore.getState().setCurrentDashboard(d);
    expect(useDashboardStore.getState().currentDashboard).toEqual(d);
    expect(useDashboardStore.getState().hasUnsavedChanges).toBe(false);

    useDashboardStore.getState().selectWidget("w1");
    useDashboardStore.getState().clearCurrentDashboard();
    const state = useDashboardStore.getState();
    expect(state.currentDashboard).toBeNull();
    expect(state.currentError).toBeNull();
    expect(state.selectedWidgetId).toBeNull();
  });

  it("widget CRUD:无当前仪表盘时 no-op", () => {
    useDashboardStore.getState().addWidget(widget("w1"));
    expect(useDashboardStore.getState().currentDashboard).toBeNull();

    useDashboardStore.getState().updateWidget("w1", { title: "x" });
    useDashboardStore.getState().removeWidget("w1");
    useDashboardStore.getState().updateWidgetLayouts([]);
    useDashboardStore.getState().updateLayout({});
    expect(useDashboardStore.getState().currentDashboard).toBeNull();
  });

  it("addWidget/updateWidget/removeWidget 修改当前仪表盘 widgets 并置脏", () => {
    useDashboardStore.setState({ currentDashboard: dashboard("d1", [widget("w1")]) });

    useDashboardStore.getState().addWidget(widget("w2"));
    let d = useDashboardStore.getState().currentDashboard!;
    expect(d.layout.widgets.map((w) => w.id)).toEqual(["w1", "w2"]);
    expect(useDashboardStore.getState().hasUnsavedChanges).toBe(true);

    useDashboardStore.getState().updateWidget("w2", { title: "改名" });
    d = useDashboardStore.getState().currentDashboard!;
    expect(d.layout.widgets.find((w) => w.id === "w2")!.title).toBe("改名");

    useDashboardStore.getState().selectWidget("w2");
    useDashboardStore.getState().removeWidget("w2");
    d = useDashboardStore.getState().currentDashboard!;
    expect(d.layout.widgets.map((w) => w.id)).toEqual(["w1"]);
    expect(useDashboardStore.getState().selectedWidgetId).toBeNull(); // 移除选中项时清空
  });

  it("removeWidget 非选中 widget 不清空 selectedWidgetId", () => {
    useDashboardStore.setState({ currentDashboard: dashboard("d1", [widget("w1"), widget("w2")]) });
    useDashboardStore.getState().selectWidget("w1");
    useDashboardStore.getState().removeWidget("w2");
    expect(useDashboardStore.getState().selectedWidgetId).toBe("w1");
  });

  it("updateWidgetLayouts 批量更新位置;updateLayout 合并布局字段", () => {
    useDashboardStore.setState({ currentDashboard: dashboard("d1", [widget("w1"), widget("w2")]) });

    useDashboardStore
      .getState()
      .updateWidgetLayouts([{ id: "w1", position: { x: 4, y: 0, w: 4, h: 3 } }]);
    let d = useDashboardStore.getState().currentDashboard!;
    expect(d.layout.widgets.find((w) => w.id === "w1")!.position).toEqual({
      x: 4,
      y: 0,
      w: 4,
      h: 3,
    });

    useDashboardStore.getState().updateLayout({ widgets: [widget("only")] } as never);
    d = useDashboardStore.getState().currentDashboard!;
    expect(d.layout.widgets).toHaveLength(1);
  });

  it("saveCurrentDashboard 调 updateDashboard;无当前仪表盘 no-op", async () => {
    svc.updateDashboard.mockResolvedValue(undefined);
    await useDashboardStore.getState().saveCurrentDashboard();
    expect(svc.updateDashboard).not.toHaveBeenCalled();

    useDashboardStore.setState({ currentDashboard: dashboard("d1", [widget("w1")]) });
    await useDashboardStore.getState().saveCurrentDashboard();
    expect(svc.updateDashboard).toHaveBeenCalledWith(
      "d1",
      expect.objectContaining({ name: "board-d1", layout: { widgets: [widget("w1")] } })
    );
  });

  it("resetCurrentDashboard 重新拉取当前仪表盘", async () => {
    const fresh = dashboard("d1", [widget("fresh")]);
    svc.getDashboard.mockResolvedValue(fresh);
    useDashboardStore.setState({ currentDashboard: dashboard("d1", [widget("stale")]) });

    await useDashboardStore.getState().resetCurrentDashboard();
    expect(useDashboardStore.getState().currentDashboard?.layout.widgets[0].id).toBe("fresh");
  });

  it("选择/拖拽状态与 UI 开关", () => {
    const s = useDashboardStore.getState();
    s.selectWidget("w9");
    expect(useDashboardStore.getState().selectedWidgetId).toBe("w9");

    s.startDragging("w9");
    expect(useDashboardStore.getState().draggingWidgetId).toBe("w9");
    useDashboardStore.getState().stopDragging();
    expect(useDashboardStore.getState().draggingWidgetId).toBeNull();

    useDashboardStore.getState().toggleGridLines();
    useDashboardStore.getState().toggleWidgetBorders();
    const state = useDashboardStore.getState();
    expect(state.showGridLines).toBe(true);
    expect(state.showWidgetBorders).toBe(true);
  });

  it("widget 数据缓存:写入/读取/过期清除(fake timers)/全清", () => {
    vi.useFakeTimers();
    const store = useDashboardStore.getState();
    store.cacheWidgetData("w1", { v: 1 });

    expect(useDashboardStore.getState().getCachedWidgetData("w1")).toEqual({ v: 1 });
    expect(useDashboardStore.getState().getCachedWidgetData("missing")).toBeNull();

    // 5 分钟后过期 → 返回 null 并清缓存
    vi.advanceTimersByTime(6 * 60 * 1000);
    expect(useDashboardStore.getState().getCachedWidgetData("w1")).toBeNull();
    expect(useDashboardStore.getState().widgetDataCache.has("w1")).toBe(false);

    useDashboardStore.getState().cacheWidgetData("a", 1);
    useDashboardStore.getState().cacheWidgetData("b", 2);
    useDashboardStore.getState().clearWidgetCache("a");
    expect(useDashboardStore.getState().widgetDataCache.has("a")).toBe(false);
    expect(useDashboardStore.getState().widgetDataCache.has("b")).toBe(true);
    useDashboardStore.getState().clearWidgetCache();
    expect(useDashboardStore.getState().widgetDataCache.size).toBe(0);
  });

  it("fetchDefaultDashboard 成功/失败;pageMode/ws 状态/updateWidgetData/reset", async () => {
    svc.getDefaultDashboard.mockResolvedValue(dashboard("default"));
    await useDashboardStore.getState().fetchDefaultDashboard();
    expect(useDashboardStore.getState().defaultDashboard?.id).toBe("default");
    expect(useDashboardStore.getState().defaultDashboardLoading).toBe(false);

    svc.getDefaultDashboard.mockRejectedValue(new Error("no default"));
    await expect(useDashboardStore.getState().fetchDefaultDashboard()).rejects.toThrow();
    expect(useDashboardStore.getState().defaultDashboard).toBeNull();

    useDashboardStore.getState().setPageMode("edit");
    expect(useDashboardStore.getState().pageMode).toBe("edit");

    useDashboardStore.getState().setWsStatus("connected");
    expect(useDashboardStore.getState().wsStatus).toBe("connected");
    expect(useDashboardStore.getState().wsLastConnected).not.toBeNull();
    useDashboardStore.getState().setWsStatus("disconnected");
    expect(useDashboardStore.getState().wsLastConnected).not.toBeNull(); // 保留最后连接时间

    useDashboardStore.getState().setIsRefreshing(true);
    expect(useDashboardStore.getState().isRefreshing).toBe(true);

    useDashboardStore.getState().updateWidgetData("w1", { updated: true });
    expect(useDashboardStore.getState().widgetDataCache.get("w1")!.data).toEqual({
      updated: true,
    });

    useDashboardStore.getState().reset();
    const state = useDashboardStore.getState();
    expect(state.pageMode).toBe("home");
    expect(state.viewMode).toBe("view");
    expect(state.widgetDataCache.size).toBe(0);
  });

  it("persist 只落盘 viewMode/开关(T-83-04-03)", () => {
    useDashboardStore.getState().setViewMode("edit");
    useDashboardStore.getState().toggleGridLines();

    const raw = localStorage.getItem("dashboard-storage");
    expect(raw).toBeTruthy();
    const persisted = JSON.parse(raw!).state;
    expect(persisted.viewMode).toBe("edit");
    expect(persisted.showGridLines).toBe(true);
    expect(persisted.currentDashboard).toBeUndefined(); // 不持久化
  });
});
