/**
 * Phase 88 Batch66 — useArticleData hook 测试(245 行大 hook)
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App, ConfigProvider } from "antd";

vi.mock("@/lib/knowledgeApi", () => ({
  getKnowledgeArticleList: vi.fn().mockResolvedValue({ data: { list: [], total: 0 } }),
  getKnowledgeArticleStatistics: vi.fn().mockResolvedValue({ data: {} }),
  getKnowledgeCategoryList: vi.fn().mockResolvedValue({ data: [] }),
  getAllKnowledgeTags: vi.fn().mockResolvedValue({ data: [] }),
  post: vi.fn().mockResolvedValue({ data: {} }),
  put: vi.fn().mockResolvedValue({ data: {} }),
  del: vi.fn().mockResolvedValue({ data: {} }),
}));

import { useArticleData } from "../hooks/useArticleData";

beforeEach(() => {
  vi.clearAllMocks();
});

const wrap = ({ children }: { children: React.ReactNode }) => (
  <ConfigProvider>
    <App>{children}</App>
  </ConfigProvider>
);

const opts = () => ({ current: 1, pageSize: 10, searchForm: {} as any });

describe("useArticleData", () => {
  it("initial state + handlers", () => {
    const { result } = renderHook(() => useArticleData(opts()), { wrapper: wrap });
    expect(result.current.articles).toEqual([]);
    expect(result.current.categories).toEqual([]);
    expect(result.current.tags).toEqual([]);
    expect(result.current.loading).toBe(false);
    expect(result.current.total).toBe(0);
    expect(result.current.statistics.total).toBe(0);
    expect(typeof result.current.fetchList).toBe("function");
    expect(typeof result.current.fetchCategories).toBe("function");
    expect(typeof result.current.fetchTags).toBe("function");
  });

  it("fetchList / fetchCategories / fetchTags", async () => {
    const { result } = renderHook(() => useArticleData(opts()), { wrapper: wrap });
    await act(async () => {
      await result.current.fetchList();
      await result.current.fetchCategories();
      await result.current.fetchTags();
    });
  });

  it("setters 直写 state", () => {
    const { result } = renderHook(() => useArticleData(opts()), { wrapper: wrap });
    act(() => {
      result.current.setArticles([{ id: "a1" } as any]);
      result.current.setCategories([{ id: "c1", name: "cat1", children: [] } as any]);
      result.current.setTags([{ id: "t1", name: "tag1" } as any]);
    });
    expect(result.current.articles.length).toBe(1);
    expect(result.current.categories.length).toBe(1);
    expect(result.current.tags.length).toBe(1);
  });
});
