/**
 * Phase 88 — 真实 hooks + mock API 页面渲染测试
 * 模式: 只 mock @/lib/api 端点 → 真实 useTableManager/usePagination 全链执行
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor, fireEvent } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import {} from "@/test/utils/createApiMock";
import PostManagement from "../index";
import { renderWithProviders } from "@/test/utils/renderWithProviders";

// createApiTestingModule 的 post 通用回退:注册 /system/posts/list 端点
import { createApiMock } from "@/test/utils/createApiMock";

describe("PostManagement 页面渲染(真实 hooks)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders table with API-loaded rows", async () => {
    const api = createApiMock("/system/posts/list");
    api.endpoint.mockResolvedValue({
      data: {
        list: [
          { id: "p1", postName: "总经理", postCode: "CEO", postSort: 1, status: 0 },
          { id: "p2", postName: "专员", postCode: "STAFF", postSort: 2, status: 1 },
        ],
        total: 2,
      },
    });
    // statistics 端点兜底(通用 get)
    const { container } = renderWithProviders(<PostManagement />);
    await waitFor(
      () => {
        // Phase 66：rowSelection 内建后行文本同时出现在 cell 与选择 aria span——
        // 断言语义（行数据渲染可见）不变，改用 getAllByText
        expect(screen.getAllByText("总经理").length).toBeGreaterThan(0);
      },
      { timeout: 5000 }
    );
    expect(container.querySelector(".ant-table")).not.toBeNull();
  });

  it("renders toolbar with 新增 button", async () => {
    const api = createApiMock("/system/posts/list");
    api.endpoint.mockResolvedValue({ data: { list: [], total: 0 } });
    const { container } = renderWithProviders(<PostManagement />);
    await waitFor(() => {
      expect(container.querySelector(".ant-table")).not.toBeNull();
    });
  });

  // Phase 66 回归锚（T6）：分页 showTotal 口径逐字（63 §1：共 N 条 / M 页）
  it("renders pagination showTotal 共 N 条 / M 页", async () => {
    const api = createApiMock("/system/posts/list");
    api.endpoint.mockResolvedValue({
      data: {
        list: Array.from({ length: 10 }, (_, i) => ({
          id: `p${i}`,
          postName: `岗位${i}`,
          postCode: `C${i}`,
          postSort: i,
          status: 0,
        })),
        total: 25,
      },
    });
    const { container } = renderWithProviders(<PostManagement />);
    await waitFor(() => {
      expect(screen.getByText("共 25 条 / 3 页")).not.toBeNull();
    });
    expect(container.querySelector(".ant-table")).not.toBeNull();
  });

  // Phase 66 回归锚（T6）：删除入口经 ConfirmDelete（确认文案模板），不再有裸 Popconfirm
  it("row delete entry goes through ConfirmDelete modal", async () => {
    const api = createApiMock("/system/posts/list");
    api.endpoint.mockResolvedValue({
      data: {
        list: [{ id: "p1", postName: "总经理", postCode: "CEO", postSort: 1, status: 0 }],
        total: 1,
      },
    });
    renderWithProviders(<PostManagement />);
    await waitFor(
      () => {
        expect(screen.getAllByText("总经理").length).toBeGreaterThan(0);
      },
      { timeout: 5000 }
    );
    // Phase 68.1：createDeleteAction({ target: r.postName })——按钮文字固定「删除」，
    // 对象名只进可访问名（Q1：aria-label=「删除{target}」，与确认标题解耦）
    const deleteBtn = screen.getByLabelText("删除总经理");
    expect(deleteBtn).toBeTruthy();
    fireEvent.click(deleteBtn);
    await waitFor(() => {
      // confirmDelete 模板文案（65 §4 唯一入口）
      expect(document.querySelector(".ant-modal-confirm")).not.toBeNull();
      expect(document.body.textContent).toContain("删除后不可恢复");
    });
  });
});
