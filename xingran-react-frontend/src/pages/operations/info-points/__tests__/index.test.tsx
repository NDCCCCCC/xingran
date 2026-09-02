/**
 * Phase 88 Batch414 — pages/operations/info-points 测试
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

vi.mock("@/lib/opsApi", () => ({
  infoPointApi: {
    list: vi.fn(async () => ({ list: [], total: 0 })),
    create: vi.fn(async () => ({})),
    update: vi.fn(async () => ({})),
    delete: vi.fn(async () => ({})),
  },
}));

function Wrap({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter initialEntries={["/operations/info-points"]}>
      <QueryClientProvider client={qc}>
        <AntdApp>{children}</AntdApp>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("pages/operations/info-points", () => {
  it("导出为函数组件", async () => {
    const mod = await import("../index");
    expect(typeof mod.default).toBe("function");
  });

  it("基础渲染不抛错", async () => {
    const { default: Comp } = await import("../index");
    expect(() => render(<Comp />, { wrapper: Wrap })).not.toThrow();
  }, 15000);
});