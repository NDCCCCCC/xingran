/**
 * Phase 88 Batch422 — pages/asset/reconciliation/exception-rules 渲染
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import ExceptionRulesPage from "../index";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/assetApi", () => ({
  exceptionRuleApi: {
    list: vi.fn(async () => ({ list: [], total: 0 })),
    create: vi.fn(async () => ({})),
    update: vi.fn(async () => ({})),
    delete: vi.fn(async () => ({})),
  },
}));

function Wrap({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter initialEntries={["/asset/reconciliation/exception-rules"]}>
      <QueryClientProvider client={qc}>
        <AntdApp>{children}</AntdApp>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("exception-rules page render", () => {
  it("基础渲染不抛错", () => {
    expect(() => render(<ExceptionRulesPage />, { wrapper: Wrap })).not.toThrow();
  });
});