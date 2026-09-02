/**
 * Phase 88 Batch418 — pages/system/menu 渲染
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import MenuPage from "../index";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/menuApi", () => ({
  getMenuList: vi.fn(async () => []),
  getAllMenus: vi.fn(async () => []),
  createMenu: vi.fn(async () => ({})),
  updateMenu: vi.fn(async () => ({})),
  deleteMenu: vi.fn(async () => ({})),
}));

function Wrap({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter initialEntries={["/system/menu"]}>
      <QueryClientProvider client={qc}>
        <AntdApp>{children}</AntdApp>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("menu page render", () => {
  it("基础渲染不抛错", () => {
    expect(() => render(<MenuPage />, { wrapper: Wrap })).not.toThrow();
  });
});