/**
 * Phase 88 Batch421 — pages/workorder/statistics 渲染
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import StatisticsPage from "../index";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/workorderApi", () => ({
  getWorkorderStatistics: vi.fn(async () => ({})),
  getWorkorderOverview: vi.fn(async () => ({})),
  getWorkorderTrend: vi.fn(async () => []),
  getWorkorderDistribution: vi.fn(async () => []),
  WorkOrderPriority: {},
  WorkOrderStatus: {},
  WorkOrderType: {},
}));

function Wrap({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter initialEntries={["/workorder/statistics"]}>
      <QueryClientProvider client={qc}>
        <AntdApp>{children}</AntdApp>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("workorder statistics page render", () => {
  it("基础渲染不抛错", () => {
    expect(() => render(<StatisticsPage />, { wrapper: Wrap })).not.toThrow();
  });
});
