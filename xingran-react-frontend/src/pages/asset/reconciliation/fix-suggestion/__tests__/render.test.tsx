/**
 * Phase 88 Batch422 — pages/asset/reconciliation/fix-suggestion 渲染
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import FixSuggestionPage from "../index";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/assetApi", () => ({
  fixSuggestionApi: {
    list: vi.fn(async () => ({ list: [], total: 0 })),
    detail: vi.fn(async () => ({})),
    apply: vi.fn(async () => ({})),
    ignore: vi.fn(async () => ({})),
  },
}));

function Wrap({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter initialEntries={["/asset/reconciliation/fix-suggestion"]}>
      <QueryClientProvider client={qc}>
        <AntdApp>{children}</AntdApp>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("fix-suggestion page render", () => {
  it("基础渲染不抛错", () => {
    expect(() => render(<FixSuggestionPage />, { wrapper: Wrap })).not.toThrow();
  });
});