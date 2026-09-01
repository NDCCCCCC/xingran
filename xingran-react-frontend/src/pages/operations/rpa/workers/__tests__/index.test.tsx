/**
 * Phase 88 Batch409 — pages/operations/rpa/workers 测试
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

vi.mock("@/lib/rpaApi", () => ({
  getRPAWorkerList: vi.fn(async () => ({ list: [], total: 0 })),
  getRPAWorkerStats: vi.fn(async () => ({})),
  restartRPAWorker: vi.fn(async () => ({})),
}));

function wrapper({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter initialEntries={["/operations/rpa/workers"]}>
      <QueryClientProvider client={qc}>
        <AntdApp>{children}</AntdApp>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("pages/operations/rpa/workers", () => {
  it("导出为函数组件", async () => {
    const mod = await import("../index");
    expect(typeof mod.default).toBe("function");
  });

  it("基础渲染不抛错", async () => {
    const { default: Comp } = await import("../index");
    expect(() => render(<Comp />, { wrapper })).not.toThrow();
  }, 15000);
});