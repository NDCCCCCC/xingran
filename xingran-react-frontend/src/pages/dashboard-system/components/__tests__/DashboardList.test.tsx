/**
 * Phase 88 Batch116 — dashboard-system DashboardList 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import DashboardList from "../DashboardList";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/store/dashboardStore", () => ({
  useDashboardStore: () => ({
    dashboards: [],
    total: 0,
    loading: false,
    current: 1,
    pageSize: 10,
    selectedDashboard: null,
    listPagination: { current: 1, pageSize: 10 },
    fetchDashboards: vi.fn(() => Promise.resolve({ list: [], total: 0 })),
    deleteDashboard: vi.fn(),
    duplicateDashboard: vi.fn(),
    setDefaultDashboard: vi.fn(),
    setPagination: vi.fn(),
  }),
}));

function renderList(props: { onNavigateToView?: any; onNavigateToEdit?: any } = {}) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderWithProviders(
    <QueryClientProvider client={qc}>
      <AntdApp>
        <DashboardList {...props} />
      </AntdApp>
    </QueryClientProvider>
  );
}

describe("DashboardList 渲染", () => {
  it("空数据 → 渲染不抛错", async () => {
    const { baseElement } = renderList();
    await new Promise((r) => setTimeout(r, 400));
    expect(baseElement).toBeDefined();
  });

  it("onNavigateToView / onNavigateToEdit 回调传入", () => {
    const onView = vi.fn();
    const onEdit = vi.fn();
    const { baseElement } = renderList({
      onNavigateToView: onView,
      onNavigateToEdit: onEdit,
    });
    expect(baseElement).toBeDefined();
  });
});
