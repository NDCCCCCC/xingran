/**
 * Phase 88 Batch419 — pages/system/config 渲染
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import ConfigPage from "../index";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/configApi", () => ({
  getConfigList: vi.fn(async () => ({ list: [], total: 0 })),
  getConfigByKey: vi.fn(async () => ({})),
  createConfig: vi.fn(async () => ({})),
  updateConfig: vi.fn(async () => ({})),
  deleteConfig: vi.fn(async () => ({})),
}));

function Wrap({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter initialEntries={["/system/config"]}>
      <QueryClientProvider client={qc}>
        <AntdApp>{children}</AntdApp>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("config page render", () => {
  it("基础渲染不抛错", () => {
    expect(() => render(<ConfigPage />, { wrapper: Wrap })).not.toThrow();
  });
});
