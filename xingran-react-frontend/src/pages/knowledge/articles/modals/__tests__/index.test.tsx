/**
 * Phase 88 Batch15 — knowledge EditModal / PreviewModal
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { ConfigProvider } from "antd";
import { EditModal, PreviewModal } from "../index";

const cats = [
  { id: "c1", categoryName: "运维", parentId: null, sort: 0, status: 0 },
  { id: "c2", categoryName: "开发", parentId: null, sort: 1, status: 0 },
];
const tags = [
  { id: "t1", tagName: "Go", color: "blue" },
  { id: "t2", tagName: "Rust", color: "orange" },
];

function wrap(ui: React.ReactElement) {
  return render(<ConfigProvider>{ui}</ConfigProvider>, { legacyRoot: false });
}

describe("knowledge/articles/modals — EditModal", () => {
  it("renders title for new article", () => {
    wrap(
      <EditModal
        open
        editingRecord={null}
        categories={cats}
        flatCategories={cats}
        tags={tags}
        onOk={vi.fn()}
        onCancel={vi.fn()}
      />
    );
    expect(screen.getByText("新增文章")).toBeTruthy();
  });

  it("renders title for editing article", () => {
    wrap(
      <EditModal
        open
        editingRecord={{
          id: "a1",
          title: "测试",
          content: "x",
          categoryId: "c1",
          tagIds: ["t1"],
          status: 0,
          summary: "s",
          viewCount: 1,
          likeCount: 0,
        }}
        categories={cats}
        flatCategories={cats}
        tags={tags}
        onOk={vi.fn()}
        onCancel={vi.fn()}
      />
    );
    expect(screen.getByText("编辑文章")).toBeTruthy();
  });

  it("renders all form labels", () => {
    wrap(
      <EditModal
        open
        editingRecord={null}
        categories={cats}
        flatCategories={cats}
        tags={tags}
        onOk={vi.fn()}
        onCancel={vi.fn()}
      />
    );
    expect(screen.getByText("文章标题")).toBeTruthy();
    expect(screen.getByText("文章分类")).toBeTruthy();
    expect(screen.getByText("状态")).toBeTruthy();
    expect(screen.getByText("标签")).toBeTruthy();
    expect(screen.getByText("摘要")).toBeTruthy();
    expect(screen.getByText("文章内容")).toBeTruthy();
  });
});

describe("knowledge/articles/modals — PreviewModal", () => {
  it("renders nothing when previewRecord null", () => {
    const { baseElement } = wrap(<PreviewModal open previewRecord={null} onClose={vi.fn()} />);
    expect(baseElement.querySelector("h1")).toBeNull();
  });

  it("renders preview content with title/category/views", () => {
    const { baseElement } = wrap(
      <PreviewModal
        open
        previewRecord={{
          id: "a1",
          title: "预览标题",
          content: "正文内容",
          summary: "摘要内容",
          categoryId: "c1",
          category: cats[0],
          tagIds: [],
          status: 0,
          viewCount: 42,
          likeCount: 7,
        }}
        onClose={vi.fn()}
      />
    );
    const html = baseElement.innerHTML;
    expect(html).toContain("预览标题");
    expect(html).toContain("运维");
    expect(html).toContain("42");
    expect(html).toContain("7");
    expect(html).toContain("摘要内容");
    expect(html).toContain("正文内容");
  });

  it("does not render summary block when summary empty", () => {
    wrap(
      <PreviewModal
        open
        previewRecord={{
          id: "a1",
          title: "无摘要",
          content: "正文",
          summary: "",
          categoryId: "c1",
          category: cats[0],
          tagIds: [],
          status: 0,
          viewCount: 1,
          likeCount: 0,
        }}
        onClose={vi.fn()}
      />
    );
    expect(screen.queryByText(/^摘要：/)).toBeNull();
  });
});
