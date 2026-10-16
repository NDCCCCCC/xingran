/**
 * Phase 88 Batch30 — NotificationBell 交互深测(自定义 svg 铃铛,非 antd icon)
 */
import { describe, it, expect, vi } from "vitest";
import { fireEvent } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/noticeApi", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/noticeApi")>();
  return {
    ...actual,
    getNotificationList: vi.fn().mockResolvedValue({
      data: {
        list: [
          {
            id: "n1",
            noticeTitle: "告警通知",
            content: "CPU 过高",
            priority: 2,
            noticeType: "alert",
            isRead: false,
            createdAt: "2026-08-28T10:00:00Z",
          },
        ],
        total: 1,
      },
    }),
    markNoticeAsRead: vi.fn().mockResolvedValue({}),
    markAllNoticesAsRead: vi.fn().mockResolvedValue({}),
    ignoreNotice: vi.fn().mockResolvedValue({}),
  };
});

import { renderWithProviders } from "@/test/utils/renderWithProviders";
import NotificationBell from "../NotificationBell";
import { useNoticeStore } from "@/store/noticeStore";
import { getNotificationList } from "@/lib/noticeApi";

function resetNoticeStore() {
  useNoticeStore.setState({
    notifications: [],
    unreadCount: 1,
    loading: false,
  } as any);
}

describe("NotificationBell 交互", () => {
  it("渲染自定义铃铛按钮", () => {
    resetNoticeStore();
    const { container } = renderWithProviders(<NotificationBell />);
    // 自定义 svg trigger(非 .anticon-bell)
    expect(container.querySelector("button.notif-btn")).not.toBeNull();
    expect(container.querySelector("button.notif-btn svg")).not.toBeNull();
  });

  it("点击铃铛打开下拉并拉取列表", async () => {
    resetNoticeStore();
    const { container, findByText } = renderWithProviders(<NotificationBell />);

    const trigger = container.querySelector("button.notif-btn");
    expect(trigger).not.toBeNull();
    fireEvent.click(trigger!);

    // 下拉内容渲染
    expect(await findByText("通知中心")).toBeDefined();
    expect(await findByText("告警通知")).toBeDefined();
    expect(getNotificationList).toHaveBeenCalledWith({ current: 1, pageSize: 10 });
  }, 15000);

  it("unreadCount>0 时 aria-label 带未读数", () => {
    resetNoticeStore();
    const { container } = renderWithProviders(<NotificationBell />);
    const btn = container.querySelector("button.notif-btn");
    expect(btn?.getAttribute("aria-label")).toContain("1 条未读");
  });
});
