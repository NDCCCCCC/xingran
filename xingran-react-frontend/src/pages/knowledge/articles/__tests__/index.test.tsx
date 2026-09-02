/**
 * Phase 88 Batch417 — pages/knowledge/articles 测试
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

vi.mock("@/lib/knowledgeApi", () => ({
  getArticleList: vi.fn(async () => ({ list: [], total: 0 })),
  createArticle: vi.fn(async () => ({})),
  updateArticle: vi.fn(async () => ({})),
  deleteArticle: vi.fn(async () => ({})),
  publishArticle: vi.fn(async () => ({})),
  KnowledgeArticleStatus: {},
  KnowledgeArticleCategory: {},
}));

function Wrap({ children }: { children: ReactNode }): ReactElement {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <MemoryRouter initialEntries={["/knowledge/articles"]}>
      <QueryClientProvider client={qc}>
        <AntdApp>{children}</AntdApp>
      </QueryClientProvider>
    </MemoryRouter>
  );
}

describe("pages/knowledge/articles", () => {
  it("导出为函数组件", async () => {
    const mod = await import("../index");
    expect(typeof mod.default).toBe("function");
  });

  it("基础渲染不抛错", async () => {
    const { default: Comp } = await import("../index");
    expect(() => render(<Comp />, { wrapper: Wrap })).not.toThrow();
  }, 15000);
});
