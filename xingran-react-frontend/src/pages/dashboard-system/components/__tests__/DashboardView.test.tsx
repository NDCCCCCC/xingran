/**
 * Phase 88 Batch417 — pages/dashboard-system/components/DashboardView 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import DashboardView from "../DashboardView";
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
      <AntdApp>{children}</AntdApp>
    </MemoryRouter>
  );
}

describe("DashboardView", () => {
  it("isHome=true 不抛错（无 dashboard → Spin）", () => {
    expect(() => render(<DashboardView dashboardId="d1" isHome />, { wrapper })).not.toThrow();
  });

  it("默认不抛错", () => {
    expect(() => render(<DashboardView dashboardId="d1" />, { wrapper })).not.toThrow();
  });
});