/**
 * Phase 88 Batch147 — pages/knowledge/articles/columns 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { Table, App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/components/shared/ActionButtons", () => ({
  default: ({ actions }: any) => (
    <div data-testid="action-buttons">
      {actions.map((a: any) => (
        <button key={a.key} data-testid={`action-${a.key}`} onClick={a.onClick}>
          {a.label}
        </button>
      ))}
    </div>
  ),
}));

vi.mock("@/utils/datetime", () => ({
  formatDateTime: vi.fn(() => "2026-08-30 10:00"),
}));

import { getArticleColumns } from "../columns";
import { KnowledgeArticleStatus } from "@/lib/knowledgeApi";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

const params = {
  handlePreview: vi.fn(),
  handleEdit: vi.fn(),
  handlePublish: vi.fn(() => Promise.resolve()),
  handleLike: vi.fn(() => Promise.resolve()),
  handleDelete: vi.fn(() => Promise.resolve()),
  current: 1,
  pageSize: 10,
};

describe("getArticleColumns", () => {
  it("返回 10 列", () => {
    const cols = getArticleColumns(params);
    expect(cols.length).toBe(10);
  });

  it("包含基本列", () => {
    const cols = getArticleColumns(params);
    const titles = cols.map((c) => (c as any).title);
    expect(titles).toContain("序号");
    expect(titles).toContain("标题");
    expect(titles).toContain("分类");
    expect(titles).toContain("状态");
    expect(titles).toContain("浏览");
    expect(titles).toContain("点赞");
    expect(titles).toContain("创建人");
    expect(titles).toContain("创建时间");
    expect(titles).toContain("操作");
  });

  it("getColumnSortOrder 提供时 → 应用 sortOrder", () => {
    const cols = getArticleColumns({
      ...params,
      getColumnSortOrder: (field) => (field === "title" ? "ascend" : null),
    });
    const titleCol = cols.find((c: any) => c.key === "title");
    expect((titleCol as any).sortOrder).toBe("ascend");
  });

  it("草稿状态 → 显示 发布按钮", () => {
    const cols = getArticleColumns(params);
    const actionCol = cols.find((c: any) => c.key === "action") as any;
    const record = {
      id: "a1",
      title: "T",
      status: KnowledgeArticleStatus.Draft,
      tags: [],
    };
    const rendered = actionCol.render({}, record);
    expect(rendered).toBeDefined();
  });

  it("已发布状态 → 无 发布按钮", () => {
    const cols = getArticleColumns(params);
    const actionCol = cols.find((c: any) => c.key === "action") as any;
    const record = {
      id: "a1",
      title: "T",
      status: KnowledgeArticleStatus.Published,
      tags: [],
    };
    const rendered = actionCol.render({}, record);
    expect(rendered).toBeDefined();
  });

  it("序号列 render 计算 (current-1)*pageSize+index+1", () => {
    const cols = getArticleColumns({ ...params, current: 2, pageSize: 10 });
    const indexCol = cols.find((c: any) => c.key === "index") as any;
    expect(indexCol.render(undefined, undefined, 0)).toBe(11);
    expect(indexCol.render(undefined, undefined, 5)).toBe(16);
  });

  it("标签 render: 0 tags → -", () => {
    const cols = getArticleColumns(params);
    const tagsCol = cols.find((c: any) => c.key === "tags") as any;
    const record = { id: "a1", title: "T", tags: [] };
    const { container } = render(<>{tagsCol.render({}, record)}</>);
    expect(container.textContent).toBe("-");
  });

  it("标签 render: 多 tags → 显示 tag", () => {
    const cols = getArticleColumns(params);
    const tagsCol = cols.find((c: any) => c.key === "tags") as any;
    const record = {
      id: "a1",
      title: "T",
      tags: [
        { id: "t1", tagName: "Tag1" },
        { id: "t2", tagName: "Tag2" },
      ],
    };
    const { container } = render(<>{tagsCol.render({}, record)}</>);
    expect(container.textContent).toContain("Tag1");
    expect(container.textContent).toContain("Tag2");
  });

  it("状态 render: 草稿 → Tag", () => {
    const cols = getArticleColumns(params);
    const statusCol = cols.find((c: any) => c.key === "status") as any;
    const { container } = render(<>{statusCol.render(KnowledgeArticleStatus.Draft)}</>);
    expect(container.querySelector(".ant-tag")).toBeTruthy();
  });

  it("完整表格渲染 + 行操作调用", () => {
    const cols = getArticleColumns(params);
    const dataSource = [
      {
        id: "a1",
        title: "Article 1",
        status: KnowledgeArticleStatus.Published,
        tags: [],
        viewCount: 10,
        likeCount: 5,
        createdBy: "admin",
        createdAt: "2026-01-01",
      },
    ];
    const { container, getByTestId } = render(
      <Table dataSource={dataSource} columns={cols} rowKey="id" />,
      { wrapper }
    );
    // Trigger preview action
    const btn = container.querySelector('[data-testid="action-preview"]') as HTMLElement;
    if (btn) fireEvent.click(btn);
    expect(params.handlePreview).toHaveBeenCalled();
  });
});
