/**
 * knowledgeApi 端点契约测试 (Phase 83-03)
 *
 * 锁定:文章 / 分类 / 标签 / 工单转文章 各端点 URL 与请求体形状。
 */
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockPost = vi.fn();
vi.mock("@/lib/api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
  get: vi.fn(),
}));
vi.mock("./api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
  get: vi.fn(),
}));

import {
  convertWorkOrderToArticle,
  createKnowledgeArticle,
  createKnowledgeCategory,
  createKnowledgeTag,
  deleteKnowledgeArticle,
  deleteKnowledgeCategory,
  deleteKnowledgeTag,
  getAllKnowledgeTags,
  getKnowledgeArticle,
  getKnowledgeArticleList,
  getKnowledgeArticleStatistics,
  getKnowledgeCategory,
  getKnowledgeCategoryList,
  likeKnowledgeArticle,
  searchKnowledgeArticles,
  updateKnowledgeArticle,
  updateKnowledgeCategory,
  updateKnowledgeTag,
} from "./knowledgeApi";

const OK = { code: 0 };

describe("knowledgeApi — 文章", () => {
  beforeEach(() => mockPost.mockReset());

  it("getKnowledgeArticleList POST /knowledge/articles/list", async () => {
    mockPost.mockResolvedValueOnce(OK);
    const params = { current: 1, pageSize: 10, keyword: "VPN" };
    await getKnowledgeArticleList(params);
    expect(mockPost).toHaveBeenCalledWith("/knowledge/articles/list", params);
  });

  it("getKnowledgeArticleStatistics POST /knowledge/articles/statistics", async () => {
    mockPost.mockResolvedValueOnce(OK);
    await getKnowledgeArticleStatistics();
    expect(mockPost).toHaveBeenCalledWith("/knowledge/articles/statistics", {});
  });

  it("getKnowledgeArticle / create / update / delete 按 ID 拼接", async () => {
    mockPost.mockResolvedValue(OK);
    await getKnowledgeArticle("a1");
    expect(mockPost).toHaveBeenNthCalledWith(1, "/knowledge/articles/a1", {});
    const create = { title: "新文章", content: "正文", categoryId: "c1" };
    await createKnowledgeArticle(create);
    expect(mockPost).toHaveBeenNthCalledWith(2, "/knowledge/articles", create);
    await updateKnowledgeArticle("a1", { title: "改标题" });
    expect(mockPost).toHaveBeenNthCalledWith(3, "/knowledge/articles/a1/update", {
      title: "改标题",
    });
    await deleteKnowledgeArticle("a1");
    expect(mockPost).toHaveBeenNthCalledWith(4, "/knowledge/articles/a1/delete");
  });

  it("searchKnowledgeArticles POST /knowledge/articles/search", async () => {
    mockPost.mockResolvedValueOnce(OK);
    const data = { keyword: "断网", current: 1, pageSize: 5 };
    await searchKnowledgeArticles(data);
    expect(mockPost).toHaveBeenCalledWith("/knowledge/articles/search", data);
  });

  it("likeKnowledgeArticle POST /knowledge/articles/:id/like", async () => {
    mockPost.mockResolvedValueOnce(OK);
    await likeKnowledgeArticle("a1");
    expect(mockPost).toHaveBeenCalledWith("/knowledge/articles/a1/like");
  });
});

describe("knowledgeApi — 分类", () => {
  beforeEach(() => mockPost.mockReset());

  it("list 默认空对象,get/create/update/delete 按 ID 拼接", async () => {
    mockPost.mockResolvedValue(OK);
    await getKnowledgeCategoryList();
    expect(mockPost).toHaveBeenNthCalledWith(1, "/knowledge/categories/list", {});
    await getKnowledgeCategoryList({ keyword: "网络" });
    expect(mockPost).toHaveBeenNthCalledWith(2, "/knowledge/categories/list", { keyword: "网络" });
    await getKnowledgeCategory("c1");
    expect(mockPost).toHaveBeenNthCalledWith(3, "/knowledge/categories/c1", {});
    const create = { categoryName: "网络", parentId: "" };
    await createKnowledgeCategory(create);
    expect(mockPost).toHaveBeenNthCalledWith(4, "/knowledge/categories", create);
    await updateKnowledgeCategory("c1", { categoryName: "网络设备" });
    expect(mockPost).toHaveBeenNthCalledWith(5, "/knowledge/categories/c1/update", {
      categoryName: "网络设备",
    });
    await deleteKnowledgeCategory("c1");
    expect(mockPost).toHaveBeenNthCalledWith(6, "/knowledge/categories/c1/delete");
  });
});

describe("knowledgeApi — 标签与工单转换", () => {
  beforeEach(() => mockPost.mockReset());

  it("getAllKnowledgeTags POST /knowledge/tags/all", async () => {
    mockPost.mockResolvedValueOnce(OK);
    await getAllKnowledgeTags();
    expect(mockPost).toHaveBeenCalledWith("/knowledge/tags/all", {});
  });

  it("createKnowledgeTag / updateKnowledgeTag / deleteKnowledgeTag", async () => {
    mockPost.mockResolvedValue(OK);
    await createKnowledgeTag({ tagName: "VPN" });
    expect(mockPost).toHaveBeenNthCalledWith(1, "/knowledge/tags", { tagName: "VPN" });
    await updateKnowledgeTag("t1", { tagName: "SD-WAN" });
    expect(mockPost).toHaveBeenNthCalledWith(2, "/knowledge/tags/t1/update", { tagName: "SD-WAN" });
    await deleteKnowledgeTag("t1");
    expect(mockPost).toHaveBeenNthCalledWith(3, "/knowledge/tags/t1/delete");
  });

  it("convertWorkOrderToArticle POST /knowledge/workorders/:id", async () => {
    mockPost.mockResolvedValueOnce(OK);
    const data = { title: "工单复盘", content: "处理过程", categoryId: "c1" };
    await convertWorkOrderToArticle("w1", data);
    expect(mockPost).toHaveBeenCalledWith("/knowledge/workorders/w1", data);
  });
});
