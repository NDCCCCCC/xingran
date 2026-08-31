/**
 * Phase 88 Batch369 — components/layout/shared/useRouteTabs 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const mockAddTab = vi.fn();
const mockUpdateTab = vi.fn();
const mockSetActiveTab = vi.fn();
const mockUseTabsReturn: any = {
  tabs: [],
  activeTab: "",
  addTab: mockAddTab,
  updateTab: mockUpdateTab,
  setActiveTab: mockSetActiveTab,
};
vi.mock("@/store/tabsStore", () => ({
  useTabs: vi.fn(() => mockUseTabsReturn),
  useTabsStore: {
    getState: vi.fn(() => ({
      ...mockUseTabsReturn,
      tabs: mockUseTabsReturn.tabs,
    })),
  },
}));

vi.mock("@/store/dashboardStore", () => ({
  useDashboardStore: vi.fn(() => ({ currentDashboard: { name: "My Dashboard" } })),
}));

vi.mock("@/router/routeConfigManager", () => ({
  routeConfigManager: {
    isInitialized: vi.fn(() => false),
    getRouteMeta: vi.fn(() => undefined),
    getRouteTitle: vi.fn(() => ""),
  },
}));

import { useRouteTabs } from "../useRouteTabs";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <MemoryRouter initialEntries={["/system/user"]}>{children}</MemoryRouter>;
}

describe("components/layout/shared/useRouteTabs", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockUseTabsReturn.tabs = [];
    mockUseTabsReturn.activeTab = "";
  });

  it("自动添加 tab", () => {
    renderHook(() => useRouteTabs(), { wrapper });
    expect(mockAddTab).toHaveBeenCalled();
  });

  it("/login 路径 → 不添加 /login tab", () => {
    renderHook(() => useRouteTabs(), {
      wrapper: ({ children }: { children: ReactNode }) => (
        <MemoryRouter initialEntries={["/login"]}>{children}</MemoryRouter>
      ),
    });
    const loginCalls = mockAddTab.mock.calls.filter((call: any[]) => call[0]?.key === "/login");
    expect(loginCalls.length).toBe(0);
  });

  it("/dashboard 路径 → 添加仪表盘 tab", () => {
    renderHook(() => useRouteTabs(), {
      wrapper: ({ children }: { children: ReactNode }) => (
        <MemoryRouter initialEntries={["/dashboard"]}>{children}</MemoryRouter>
      ),
    });
    expect(mockAddTab).toHaveBeenCalledWith(
      expect.objectContaining({
        key: "/dashboard",
        path: "/dashboard",
        closable: false,
        pinned: true,
      })
    );
  });

  it("/dashboard 已存在 → 不重复 addTab", () => {
    mockUseTabsReturn.tabs = [
      { key: "/dashboard", title: "仪表盘", closable: false, pinned: true, path: "/dashboard" },
    ];
    renderHook(() => useRouteTabs(), {
      wrapper: ({ children }: { children: ReactNode }) => (
        <MemoryRouter initialEntries={["/dashboard"]}>{children}</MemoryRouter>
      ),
    });
    // 没有调用 addTab 添加 dashboard key
    const dashboardAdds = mockAddTab.mock.calls.filter(
      (call: any[]) => call[0]?.key === "/dashboard"
    );
    expect(dashboardAdds.length).toBe(0);
  });
});
