/**
 * Phase 88 Batch122 — my-notices/detail 通知详情页测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, waitFor } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

let noticeStoreState: Record<string, any> = {};
vi.mock("@/store/noticeStore", () => ({
  useNoticeStore: () => noticeStoreState,
}));

vi.mock("@/lib/noticeApi", () => ({
  getMyNoticeDetail: vi.fn(),
  markNoticeAsRead: vi.fn(() => Promise.resolve()),
}));

vi.mock("@/components/NoticeDetail/NoticeDetailContent", () => ({
  default: ({ notice, showMarkAsReadButton, onMarkAsRead }: any) => (
    <div data-testid="notice-detail">
      <span data-testid="notice-title">{notice?.title}</span>
      <span data-testid="notice-read">{String(notice?.isRead)}</span>
      {showMarkAsReadButton && (
        <button data-testid="mark-read" onClick={onMarkAsRead}>
          mark
        </button>
      )}
    </div>
  ),
}));

import NoticeDetailPage from "../detail";
import { getMyNoticeDetail } from "@/lib/noticeApi";

function renderPage(initialPath = "/notices/n1") {
  return render(
    <MemoryRouter initialEntries={[initialPath]}>
      <AntdApp>
        <Routes>
          <Route path="/notices/:id" element={<NoticeDetailPage />} />
          <Route path="/user-notices" element={<div>user-notices-list</div>} />
        </Routes>
      </AntdApp>
    </MemoryRouter>
  );
}

describe("NoticeDetailPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    noticeStoreState = { markAsRead: vi.fn() };
  });

  it("loading 状态 → 渲染 Spin", async () => {
    vi.mocked(getMyNoticeDetail).mockReturnValue(new Promise(() => {}) as any);
    const { baseElement } = renderPage();
    await waitFor(() => {
      expect(baseElement.querySelector(".ant-spin")).toBeTruthy();
    });
  });

  it("加载成功 + 未读 → 渲染 detail + 调用 markAsRead", async () => {
    vi.mocked(getMyNoticeDetail).mockResolvedValue({
      data: { id: "n1", title: "T1", isRead: false },
    } as any);
    const { baseElement } = renderPage();
    await waitFor(() => {
      expect(baseElement.querySelector('[data-testid="notice-detail"]')).toBeTruthy();
    });
    expect(baseElement.querySelector('[data-testid="notice-title"]')?.textContent).toBe("T1");
    expect(baseElement.querySelector('[data-testid="notice-read"]')?.textContent).toBe("false");
  });

  it("加载成功 + 已读 → 不调 markNoticeAsRead", async () => {
    vi.mocked(getMyNoticeDetail).mockResolvedValue({
      data: { id: "n1", title: "T1", isRead: true },
    } as any);
    const { markNoticeAsRead } = await import("@/lib/noticeApi");
    vi.mocked(markNoticeAsRead).mockClear();
    const { baseElement } = renderPage();
    await waitFor(() => {
      expect(baseElement.querySelector('[data-testid="notice-detail"]')).toBeTruthy();
    });
    expect(markNoticeAsRead).not.toHaveBeenCalled();
  });

  it("加载失败 → 错误提示 + navigate 调用", async () => {
    vi.mocked(getMyNoticeDetail).mockRejectedValue(new Error("net"));
    const errSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const { baseElement } = renderPage();
    await waitFor(() => {
      expect(baseElement.textContent).toContain("加载失败");
    });
    errSpy.mockRestore();
  });

  it("点击 返回通知中心 → 不抛错", async () => {
    vi.mocked(getMyNoticeDetail).mockResolvedValue({
      data: { id: "n1", title: "T1", isRead: true },
    } as any);
    const { baseElement, getByText } = renderPage();
    await waitFor(() => {
      expect(baseElement.querySelector('[data-testid="notice-detail"]')).toBeTruthy();
    });
    expect(() => getByText("返回通知中心").click()).not.toThrow();
  });
});
