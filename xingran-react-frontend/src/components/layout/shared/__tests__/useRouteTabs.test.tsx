/**
 * Phase 88 Batch136 — components/layout/shared/useRouteTabs 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const tabsState: any = {
  tabs: [],
  activeTab: "",
  setActiveTab: vi.fn(),
  addTab: vi.fn((tab: any) => {
    tabsState.tabs.push(tab);
  }),
  updateTab: vi.fn((key: string, patch: any) => {
    const tab = tabsState.tabs.find((t: any) => t.key === key);
    if (tab) Object.assign(tab, patch);
  }),
  removeTab: vi.fn(),
};

vi.mock("@/store/tabsStore", () => ({
  useTabs: () => ({
    addTab: tabsState.addTab,
    updateTab: tabsState.updateTab,
  }),
  useTabsStore: {
    getState: () => tabsState,
  },
}));

vi.mock("@/store/dashboardStore", () => ({
  useDashboardStore: () => ({ currentDashboard: { name: "我的仪表盘" } }),
}));

vi.mock("@/router/routeConfigManager", () => ({
  routeConfigManager: {
    isInitialized: vi.fn(() => false),
    getRouteMeta: vi.fn(),
    getRouteTitle: vi.fn(() => "Page Title"),
  },
}));

import { useRouteTabs } from "../useRouteTabs";

function makeWrapper(initialPath: string) {
  return ({ children }: { children: ReactNode }): ReactElement => (
    <MemoryRouter initialEntries={[initialPath]}>{children}</MemoryRouter>
  );
}

describe("useRouteTabs", () => {
  beforeEach(() => {
    tabsState.tabs = [];
    tabsState.activeTab = "";
    vi.clearAllMocks();
  });

  it("/login 路径 → 不注册 dashboard 子页 tab (init effect 仍可能创建 dashboard tab)", () => {
    renderHook(() => useRouteTabs(), { wrapper: makeWrapper("/login") });
    // 主要 effect 检查 login 直接 return,但 init effect 仍可能执行
    // 验证核心点:location 监听 effect 未触发 addTab 路径分支
    const loginCalls = tabsState.addTab.mock.calls.filter((c: any[]) => c[0].key !== "/dashboard");
    expect(loginCalls.length).toBe(0);
  });

  it("/dashboard 路径 → 注册仪表盘 tab (pinned/closable=false)", () => {
    renderHook(() => useRouteTabs(), { wrapper: makeWrapper("/dashboard") });
    expect(tabsState.addTab).toHaveBeenCalledWith({
      key: "/dashboard",
      title: "仪表盘",
      path: "/dashboard",
      closable: false,
      pinned: true,
    });
  });

  it("/dashboard 子路径 → 使用 currentDashboard.name", () => {
    renderHook(() => useRouteTabs(), { wrapper: makeWrapper("/dashboard/abc") });
    const call = tabsState.addTab.mock.calls.find((c: any[]) => c[0].key === "/dashboard");
    expect(call?.[0].title).toBe("我的仪表盘");
  });

  it("已存在仪表盘 tab + title 变化 → updateTab", () => {
    tabsState.tabs = [{ key: "/dashboard", title: "旧标题", pinned: true, closable: false }];
    renderHook(() => useRouteTabs(), { wrapper: makeWrapper("/dashboard") });
    expect(tabsState.updateTab).toHaveBeenCalledWith("/dashboard", { title: "仪表盘" });
  });

  it("已存在但非 pinned → 修复 pinned/closable", () => {
    tabsState.tabs = [{ key: "/dashboard", title: "仪表盘", pinned: false, closable: true }];
    renderHook(() => useRouteTabs(), { wrapper: makeWrapper("/dashboard") });
    expect(tabsState.updateTab).toHaveBeenCalledWith("/dashboard", {
      pinned: true,
      closable: false,
    });
  });

  it("非仪表盘路径 → 注册 tab", () => {
    renderHook(() => useRouteTabs(), { wrapper: makeWrapper("/system/users") });
    expect(tabsState.addTab).toHaveBeenCalled();
    const call = tabsState.addTab.mock.calls[0][0];
    expect(call.path).toBe("/system/users");
  });

  it("初始化时不存在仪表盘 tab → 创建", () => {
    renderHook(() => useRouteTabs(), { wrapper: makeWrapper("/other") });
    // The init effect creates dashboard tab
    expect(tabsState.addTab).toHaveBeenCalledWith(
      expect.objectContaining({ key: "/dashboard", pinned: true, closable: false })
    );
  });
});
