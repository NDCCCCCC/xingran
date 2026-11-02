/**
 * Phase 88 Batch15 — knowledge/view 详情页
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { ConfigProvider } from "antd";
import { MemoryRouter } from "react-router-dom";

vi.mock("@/hooks/usePersistedState", () => ({
  usePersistedStateController: <T,>(opts: { defaultValue: T }) => {
    const ref = { current: opts.defaultValue };
    const set = vi.fn((v: T) => {
      ref.current = v;
    });
    return [ref.current, set] as const;
  },
}));

vi.mock("@/lib/knowledgeApi", async () => {
  return {
    searchKnowledgeArticles: vi.fn(async () => ({
      data: {
        list: [
          {
            id: "a1",
            title: "国密算法详解",
            content: "SM2/SM3/SM4",
            summary: "国密简介",
            categoryId: "c1",
            status: 1,
            viewCount: 100,
            likeCount: 5,
            createdAt: "2025-01-01",
          },
        ],
      },
    })),
    getKnowledgeArticle: vi.fn(async () => ({ data: null })),
    likeKnowledgeArticle: vi.fn(async () => ({ data: { message: "ok" } })),
    getKnowledgeCategoryList: vi.fn(async () => ({
      data: [
        { id: "c1", categoryName: "安全", parentId: null, sort: 0, status: 1 },
        { id: "c2", categoryName: "运维", parentId: null, sort: 1, status: 1 },
      ],
    })),
    getAllKnowledgeTags: vi.fn(async () => ({
      data: [
        { id: "t1", tagName: "Go" },
        { id: "t2", tagName: "Rust" },
      ],
    })),
  };
});

import KnowledgeViewPage from "../index";

function wrap() {
  return render(
    <ConfigProvider>
      <MemoryRouter initialEntries={["/knowledge"]}>
        <KnowledgeViewPage />
      </MemoryRouter>
    </ConfigProvider>
  );
}

describe("knowledge/view — index", () => {
  it("renders with article list", async () => {
    const { findByText } = wrap();
    expect(await findByText("国密算法详解")).not.toBeNull();
  });

  it("renders categories and tags in DOM", async () => {
    const { baseElement, findByText } = wrap();
    await findByText("国密算法详解");
    expect(baseElement.innerHTML).toContain("安全");
    expect(baseElement.innerHTML).toContain("运维");
    expect(baseElement.innerHTML).toContain("Rust");
  });

  it("renders search input or page chrome", async () => {
    const { baseElement, findByText } = wrap();
    await findByText("国密算法详解");
    // 至少有 Spin/Card 渲染(等待异步完成)
    expect(
      baseElement.querySelector(".ant-spin") ||
        baseElement.querySelector(".ant-card") ||
        baseElement.querySelector(".ant-empty")
    ).toBeTruthy();
  });
});
