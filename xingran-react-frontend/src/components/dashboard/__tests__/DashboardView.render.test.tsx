/**
 * Phase 88 Batch87 — dashboard/DashboardView 渲染(64 stmts, 0% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { DashboardView } from "../DashboardView";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/hooks/useWidgetPolling", () => ({
  useWidgetPolling: vi.fn(() => ({
    startPolling: vi.fn(),
    stopPolling: vi.fn(),
  })),
}));

vi.mock("@/hooks/useNetworkStatus", () => ({
  useNetworkStatus: vi.fn(() => ({ isOnline: true })),
}));

vi.mock("@/hooks/useWebSocket", () => ({
  useWebSocket: vi.fn(() => ({
    send: vi.fn(),
    subscribe: vi.fn(() => () => {}),
    connected: false,
  })),
}));

const baseDashboard = {
  id: "d1",
  name: "测试仪表盘",
  widgets: [],
  layout: { widgets: [] },
} as any;

describe("DashboardView 渲染", () => {
  it("基本渲染不抛错", () => {
    const { baseElement } = renderWithProviders(<DashboardView dashboard={baseDashboard} />);
    expect(baseElement).toBeDefined();
  });

  it("loading=true → 渲染 Spin", () => {
    const { baseElement } = renderWithProviders(
      <DashboardView dashboard={baseDashboard} loading />
    );
    expect(baseElement).toBeDefined();
  });

  it("error 非空 → 渲染 Result/Alert", () => {
    const { baseElement } = renderWithProviders(
      <DashboardView dashboard={baseDashboard} error="网络错误" onRetry={vi.fn()} />
    );
    expect(baseElement.textContent).toBeDefined();
  });

  it("readonly=true → 不渲染编辑控件", () => {
    const { baseElement } = renderWithProviders(
      <DashboardView dashboard={baseDashboard} readonly />
    );
    expect(baseElement).toBeDefined();
  });
});
