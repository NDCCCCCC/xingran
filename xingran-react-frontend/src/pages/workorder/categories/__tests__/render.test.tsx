/**
 * Phase 88 Batch421 — pages/workorder/categories 渲染
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import CategoriesPage from "../index";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/workorderApi", () => ({
  getWorkorderCategoryList: vi.fn(async () => ({ list: [], total: 0 })),
  createWorkorderCategory: vi.fn(async () => ({})),
  updateWorkorderCategory: vi.fn(async () => ({})),
  deleteWorkorderCategory: vi.fn(async () => ({})),
}));

function Wrap({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter initialEntries={["/workorder/categories"]}>
      <QueryClientProvider client={qc}>
        <AntdApp>{children}</AntdApp>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("workorder categories page render", () => {
  it("基础渲染不抛错", () => {
    expect(() => render(<CategoriesPage />, { wrapper: Wrap })).not.toThrow();
  });
});
