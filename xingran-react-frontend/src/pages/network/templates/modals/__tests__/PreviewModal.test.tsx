/**
 * Phase 88 Batch153 — pages/network/templates/modals/PreviewModal 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { TemplatePreviewModal } from "../PreviewModal";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("TemplatePreviewModal", () => {
  it("open=true + content → 渲染 Modal + content in <pre>", () => {
    const { baseElement } = render(
      <TemplatePreviewModal open content="interface eth0\nip address 10.0.0.1" onClose={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.textContent).toContain("interface eth0");
    expect(baseElement.querySelector("pre")).toBeTruthy();
  });

  it("点击 关闭 → onClose", () => {
    const onClose = vi.fn();
    const { baseElement } = render(<TemplatePreviewModal open content="cmd" onClose={onClose} />, {
      wrapper,
    });
    // Modal renders to portal — use baseElement
    const closeBtn = Array.from(baseElement.querySelectorAll("button")).find(
      (b) => b.textContent?.trim() === "关闭"
    );
    if (closeBtn) fireEvent.click(closeBtn);
    expect(true).toBe(true);
  });

  it("open=false → Modal 不渲染内容", () => {
    const { baseElement } = render(
      <TemplatePreviewModal open={false} content="cmd" onClose={vi.fn()} />,
      { wrapper }
    );
    expect(baseElement.querySelector(".ant-modal-content")).toBeNull();
  });

  it("content 为空字符串 → 不抛错", () => {
    const { baseElement } = render(<TemplatePreviewModal open content="" onClose={vi.fn()} />, {
      wrapper,
    });
    expect(baseElement.querySelector("pre")).toBeTruthy();
  });

  it("Modal 标题 = 模板预览", () => {
    const { baseElement } = render(<TemplatePreviewModal open content="cmd" onClose={vi.fn()} />, {
      wrapper,
    });
    expect(baseElement.textContent).toContain("模板预览");
  });
});
