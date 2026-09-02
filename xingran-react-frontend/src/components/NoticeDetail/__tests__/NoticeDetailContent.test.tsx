/**
 * Phase 88 Batch424 — components/NoticeDetail/NoticeDetailContent 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";
import type { Notice } from "@/types";
import NoticeDetailContent from "../NoticeDetailContent";

vi.mock("@/utils/datetime", () => ({
  formatDateTime: vi.fn(() => "2026-08-30 10:00"),
}));

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

const baseNotice: Notice = {
  id: "n1",
  noticeTitle: "测试通知标题",
  noticeContent: "测试内容",
  noticeType: "1",
  priority: 1,
  publishStatus: 1,
  isRead: false,
  isMarkdown: false,
  createdByName: "admin",
  createdAt: "2026-08-01",
  publishTime: "2026-08-15",
} as unknown as Notice;

describe("NoticeDetailContent", () => {
  it("基础渲染 + 标题", () => {
    const { baseElement } = render(<NoticeDetailContent notice={baseNotice} />, { wrapper });
    expect(baseElement.textContent).toContain("测试通知标题");
    expect(baseElement.textContent).toContain("测试内容");
  });

  it("showReadStatus=true → 显示阅读状态", () => {
    const { baseElement } = render(
      <NoticeDetailContent notice={baseNotice} showReadStatus />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("未读");
  });

  it("isRead=true → 显示已读", () => {
    const { baseElement } = render(
      <NoticeDetailContent
        notice={{ ...baseNotice, isRead: true } as unknown as Notice}
        showReadStatus
      />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("已读");
  });

  it("showPublishStatus=true → 显示发布状态 + 发布时间", () => {
    const { baseElement } = render(
      <NoticeDetailContent notice={baseNotice} showPublishStatus />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("发布时间");
  });
});