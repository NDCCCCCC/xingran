/**
 * Phase 88 Batch417 — pages/dashboard-system/components/DashboardEdit 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import DashboardEdit from "../DashboardEdit";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/components/dashboard/layout", () => ({
  LayoutToolbar: () => <div data-testid="layout-toolbar" />,
  DashboardGrid: ({ children }: any) => <div data-testid="dashboard-grid">{children}</div>,
  DashboardGridPlaceholder: () => <div data-testid="grid-placeholder" />,
}));

vi.mock("@/components/dashboard/layout/GridItem", () => ({
  GridItem: ({ children }: any) => <div data-testid="grid-item">{children}</div>,
}));

vi.mock("@/store/dashboardStore", () => ({
  useDashboardStore: vi.fn(() => ({
    currentDashboard: null,
    currentLoading: false,
    fetchDashboard: vi.fn(),
    setViewMode: vi.fn(),
    updateWidgetLayouts: vi.fn(),
    addWidget: vi.fn(),
    removeWidget: vi.fn(),
    updateWidget: vi.fn(),
    saveDashboard: vi.fn(),
    clearCurrentDashboard: vi.fn(),
  })),
}));

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return (
    <MemoryRouter initialEntries={["/dashboard-system/edit/123"]}>
      <AntdApp>{children}</AntdApp>
    </MemoryRouter>
  );
}

describe("DashboardEdit", () => {
  it("基础渲染不抛错", () => {
    expect(() => render(<DashboardEdit dashboardId="d1" />, { wrapper })).not.toThrow();
  });
});
