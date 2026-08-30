/**
 * Phase 88 Batch150 — components/NoticeDetail/NoticeDetailContent 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/utils/datetime", () => ({
  formatDateTime: vi.fn(() => "2026-08-30 10:00"),
}));

import NoticeDetailContent from "../NoticeDetailContent";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

const baseNotice = {
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
};

describe("NoticeDetailContent", () => {
  it("基础渲染 + 标题", () => {
    const { baseElement } = render(<NoticeDetailContent notice={baseNotice as any} />, { wrapper });
    expect(baseElement.textContent).toContain("测试通知标题");
    expect(baseElement.textContent).toContain("测试内容");
  });

  it("showReadStatus=true → 显示阅读状态", () => {
    const { baseElement } = render(
      <NoticeDetailContent notice={baseNotice as any} showReadStatus />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("未读");
  });

  it("isRead=true → 显示已读", () => {
    const { baseElement } = render(
      <NoticeDetailContent notice={{ ...baseNotice, isRead: true } as any} showReadStatus />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("已读");
  });

  it("showPublishStatus=true → 显示发布状态 + 发布时间", () => {
    const { baseElement } = render(
      <NoticeDetailContent notice={baseNotice as any} showPublishStatus />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("发布时间");
  });

  it("publishTime 未设置 → 显示'尚未发布'", () => {
    const { baseElement } = render(
      <NoticeDetailContent
        notice={{ ...baseNotice, publishTime: undefined } as any}
        showPublishStatus
      />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("尚未发布");
  });

  it("showCreator=true → 显示创建人/创建时间", () => {
    const { baseElement } = render(<NoticeDetailContent notice={baseNotice as any} showCreator />, {
      wrapper,
    });
    expect(baseElement.textContent).toContain("创建人");
    expect(baseElement.textContent).toContain("创建时间");
  });

  it("showMarkAsReadButton + !isRead + onMarkAsRead → 显示按钮", () => {
    const onMarkAsRead = vi.fn(() => Promise.resolve());
    const { baseElement } = render(
      <NoticeDetailContent
        notice={baseNotice as any}
        showMarkAsReadButton
        onMarkAsRead={onMarkAsRead}
      />,
      { wrapper }
    );
    const btn = baseElement.querySelector("button");
    fireEvent.click(btn!);
    expect(onMarkAsRead).toHaveBeenCalled();
  });

  it("showMarkAsReadButton + isRead → 不显示按钮", () => {
    const onMarkAsRead = vi.fn(() => Promise.resolve());
    const { baseElement } = render(
      <NoticeDetailContent
        notice={{ ...baseNotice, isRead: true } as any}
        showMarkAsReadButton
        onMarkAsRead={onMarkAsRead}
      />,
      { wrapper }
    );
    expect(baseElement.textContent).not.toContain("标记为已读");
  });

  it("isMarkdown=true → 显示 Markdown Tag", () => {
    const { baseElement } = render(
      <NoticeDetailContent notice={{ ...baseNotice, isMarkdown: true } as any} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("Markdown");
  });

  it("priority=0 → 不显示优先级 Tag", () => {
    const { baseElement } = render(
      <NoticeDetailContent notice={{ ...baseNotice, priority: 0 } as any} />,
      { wrapper }
    );
    expect(baseElement.textContent).not.toContain("高");
    expect(baseElement.textContent).not.toContain("紧急");
  });

  it("attachments → 显示附件列表", () => {
    const notice = {
      ...baseNotice,
      attachments: [{ id: "f1", fileName: "doc.pdf", fileSize: 1024 }],
    };
    const { baseElement } = render(<NoticeDetailContent notice={notice as any} />, { wrapper });
    expect(baseElement.textContent).toContain("doc.pdf");
    expect(baseElement.textContent).toContain("附件");
  });
});
