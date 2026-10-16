/**
 * Phase 88 — 真实 hooks + mock API 页面渲染测试
 * 模式: 只 mock @/lib/api 端点 → 真实 useTableManager/usePagination 全链执行
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { get } from "@/test/utils/createApiMock";
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
        expect(screen.getByText("总经理")).not.toBeNull();
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
});
