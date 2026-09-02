/**
 * Phase 88 Batch417 — pages/dashboard-system/index 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import DashboardPage from "../index";
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
    setPageMode: vi.fn(),
    currentDashboard: null,
    currentLoading: false,
    fetchDashboard: vi.fn(),
    fetchDefaultDashboard: vi.fn(),
    setViewMode: vi.fn(),
    updateWidgetLayouts: vi.fn(),
    clearCurrentDashboard: vi.fn(),
    addWidget: vi.fn(),
    removeWidget: vi.fn(),
    updateWidget: vi.fn(),
    saveDashboard: vi.fn(),
  })),
}));

function wrapper({ children, initialEntries }: { children: ReactNode; initialEntries: string[] }): ReactElement {
  return (
    <MemoryRouter initialEntries={initialEntries}>
      <AntdApp>{children}</AntdApp>
    </MemoryRouter>
  );
}

describe("DashboardPage", () => {
  it("无 id 无 mode → DashboardHome", () => {
    expect(() =>
      render(<DashboardPage />, { wrapper: (props) => wrapper({ ...props, initialEntries: ["/dashboard"] }) })
    ).not.toThrow();
  });
});