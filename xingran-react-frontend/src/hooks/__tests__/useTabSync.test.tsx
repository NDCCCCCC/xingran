/**
 * Phase 88 Batch382 — hooks/useTabSync 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

let mockTabs: any[] = [];
let mockActiveTab = "";
const mockAddTab = vi.fn();
const mockUpdateTab = vi.fn();
const mockSetActiveTab = vi.fn();

vi.mock("@/store/tabsStore", () => ({
  useTabs: vi.fn(() => ({
    addTab: mockAddTab,
    updateTab: mockUpdateTab,
  })),
  useTabsStore: {
    getState: vi.fn(() => ({
      tabs: mockTabs,
      activeTab: mockActiveTab,
      addTab: mockAddTab,
      updateTab: mockUpdateTab,
      setActiveTab: mockSetActiveTab,
    })),
  },
}));

vi.mock("@/store/dashboardStore", () => ({
  useDashboardStore: vi.fn(() => ({
    currentDashboard: { id: "d1", name: "My Dashboard" },
  })),
}));

vi.mock("@/router/routeConfigManager", () => ({
  routeConfigManager: {
    isInitialized: vi.fn(() => false),
    getRouteMeta: vi.fn(() => undefined),
    getRouteTitle: vi.fn(() => ""),
  },
}));

import { useTabSync } from "../useTabSync";

describe("hooks/useTabSync", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockTabs = [];
    mockActiveTab = "";
  });

  it("挂载后调 addTab", () => {
    renderHook(() => useTabSync("/system/user"));
    expect(mockAddTab).toHaveBeenCalled();
  });

  it("/login → 不添加 tab", () => {
    renderHook(() => useTabSync("/login"));
    const loginAdds = mockAddTab.mock.calls.filter((call: any[]) => call[0]?.key === "/login");
    expect(loginAdds.length).toBe(0);
  });

  it("/dashboard 路径 → 添加仪表盘 tab", () => {
    renderHook(() => useTabSync("/dashboard"));
    expect(mockAddTab).toHaveBeenCalledWith(
      expect.objectContaining({
        key: "/dashboard",
        path: "/dashboard",
        closable: false,
        pinned: true,
      })
    );
  });

  it("/dashboard 已存在 → 不重复添加 dashboard tab", () => {
    mockTabs = [
      { key: "/dashboard", title: "仪表盘", path: "/dashboard", closable: false, pinned: true },
    ];
    renderHook(() => useTabSync("/dashboard"));
    const dashboardAdds = mockAddTab.mock.calls.filter(
      (call: any[]) => call[0]?.key === "/dashboard"
    );
    expect(dashboardAdds.length).toBe(0);
  });

  it("普通路径 → 添加带 key/pathname 的 tab", () => {
    renderHook(() => useTabSync("/system/role"));
    expect(mockAddTab).toHaveBeenCalledWith(
      expect.objectContaining({
        key: "/system/role",
        path: "/system/role",
      })
    );
  });
});
