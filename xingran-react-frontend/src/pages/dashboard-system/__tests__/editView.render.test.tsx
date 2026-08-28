/**
 * Phase 88 Batch30 — dashboard-system edit/view 页渲染测试(原 0%)
 *
 * edit.tsx 依赖 dashboardStore.fetchDashboard → dashboardService.get
 * view.tsx 同链。mock @/lib/api 后 service 走 mock 返回 dashboard fixture。
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderWithProviders } from "@/test/utils/renderWithProviders";
import DashboardEdit from "../edit";
import DashboardView from "../view";
import { useDashboardStore } from "@/store/dashboardStore";

const dashboardFixture = {
  id: "dash-1",
  name: "运维总览",
  description: "测试仪表盘",
  isDefault: false,
  layout: { widgets: [] },
  createdAt: "2026-08-01T00:00:00Z",
  updatedAt: "2026-08-01T00:00:00Z",
};

function seedStore() {
  useDashboardStore.setState({
    currentDashboard: dashboardFixture as any,
    currentLoading: false,
    dashboards: [dashboardFixture] as any,
    listLoading: false,
    listError: null,
    selectedWidgetId: null,
  } as any);
}

describe("dashboard-system edit 页", () => {
  it("有 dashboard 时渲染 LayoutToolbar + 空 placeholder", async () => {
    seedStore();
    const { findByText, container } = renderWithProviders(<DashboardEdit />, {
      route: "/dashboard-system/dash-1/edit",
    });
    // widgets 空 → placeholder 文案
    expect(await findByText(/添加Widget/)).toBeDefined();
    expect(container).not.toBeNull();
  }, 15000);

  it("无 dashboard 时渲染 '仪表盘不存在'", async () => {
    useDashboardStore.setState({
      currentDashboard: null,
      currentLoading: false,
    } as any);
    const { findByText } = renderWithProviders(<DashboardEdit />, {
      route: "/dashboard-system/missing/edit",
    });
    expect(await findByText("仪表盘不存在")).toBeDefined();
  }, 15000);
});

describe("dashboard-system view 页", () => {
  it("有 dashboard 空 widgets 渲染占位文案", async () => {
    seedStore();
    const { findByText } = renderWithProviders(<DashboardView />, {
      route: "/dashboard-system/dash-1/view",
    });
    expect(await findByText("此仪表盘暂无Widget")).toBeDefined();
  }, 15000);

  it("无 dashboard 渲染 '仪表盘不存在'", async () => {
    useDashboardStore.setState({
      currentDashboard: null,
      currentLoading: false,
    } as any);
    const { findByText } = renderWithProviders(<DashboardView />, {
      route: "/dashboard-system/missing/view",
    });
    expect(await findByText("仪表盘不存在")).toBeDefined();
  }, 15000);
});
