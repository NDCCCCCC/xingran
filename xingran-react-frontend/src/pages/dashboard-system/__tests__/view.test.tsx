/**
 * Phase 88 Batch410 — pages/dashboard-system/view 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/components/dashboard/layout/LayoutToolbar", () => ({
  LayoutToolbar: () => <div data-testid="layout-toolbar" />,
}));

vi.mock("@/components/dashboard/layout/DashboardGrid", () => ({
  DashboardGrid: ({ children }: any) => <div data-testid="dashboard-grid">{children}</div>,
  DashboardGridPlaceholder: () => <div data-testid="grid-placeholder" />,
}));

vi.mock("@/components/dashboard/layout/GridItem", () => ({
  GridItem: ({ children }: any) => <div data-testid="grid-item">{children}</div>,
}));

vi.mock("@/components/dashboard/Widget", () => ({
  Widget: () => <div data-testid="widget" />,
}));

vi.mock("@/store/dashboardStore", () => ({
  useDashboardStore: vi.fn(() => ({
    currentDashboard: null,
    currentLoading: false,
    fetchDashboard: vi.fn(),
    setViewMode: vi.fn(),
    updateWidgetLayouts: vi.fn(),
    clearCurrentDashboard: vi.fn(),
  })),
}));

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return (
    <MemoryRouter initialEntries={["/dashboard-system/view/123"]}>
      <Routes>
        <Route path="/dashboard-system/view/:id" element={<>{children}</>} />
      </Routes>
      <AntdApp>{children}</AntdApp>
    </MemoryRouter>
  );
}

describe("pages/dashboard-system/view", () => {
  it("导出为函数组件", async () => {
    const mod = await import("../view");
    expect(typeof mod.default).toBe("function");
  });

  it("基础渲染不抛错 (无 dashboard → 空状态)", async () => {
    const { default: Comp } = await import("../view");
    expect(() => render(<Comp />, { wrapper })).not.toThrow();
  }, 15000);
});
