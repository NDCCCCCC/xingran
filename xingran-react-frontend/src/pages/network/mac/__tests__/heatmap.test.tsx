/**
 * Phase 88 Batch413 — pages/network/mac/heatmap 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/api/macHeatmapApi", () => ({
  queryMACHeatmap: vi.fn(async () => ({ heatmap: [], topPorts: [] })),
}));

vi.mock("@/components/network/MACHeatmapChart", () => ({
  default: () => <div data-testid="mac-heatmap-chart" />,
}));

function wrapper({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter initialEntries={["/network/mac/heatmap"]}>
      <QueryClientProvider client={qc}>
        <AntdApp>{children}</AntdApp>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("pages/network/mac/heatmap", () => {
  it("导出为函数组件", async () => {
    const mod = await import("../heatmap");
    expect(typeof mod.default).toBe("function");
  });

  it("基础渲染不抛错", async () => {
    const { default: Comp } = await import("../heatmap");
    expect(() => render(<Comp />, { wrapper })).not.toThrow();
  }, 15000);
});