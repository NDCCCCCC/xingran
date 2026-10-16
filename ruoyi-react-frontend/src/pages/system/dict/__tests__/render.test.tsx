/**
 * Phase 88 Batch419 — pages/system/dict 渲染
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import DictPage from "../index";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/dictApi", () => ({
  getDictTypeList: vi.fn(async () => ({ list: [], total: 0 })),
  getDictDataList: vi.fn(async () => []),
  createDictType: vi.fn(async () => ({})),
  updateDictType: vi.fn(async () => ({})),
  deleteDictType: vi.fn(async () => ({})),
  createDictData: vi.fn(async () => ({})),
  updateDictData: vi.fn(async () => ({})),
  deleteDictData: vi.fn(async () => ({})),
}));

function Wrap({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter initialEntries={["/system/dict"]}>
      <QueryClientProvider client={qc}>
        <AntdApp>{children}</AntdApp>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("dict page render", () => {
  it("基础渲染不抛错", () => {
    expect(() => render(<DictPage />, { wrapper: Wrap })).not.toThrow();
  });
});
