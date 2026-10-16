/**
 * Phase 88 Batch413 — pages/network/templates/modals/PreviewModal 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { TemplatePreviewModal } from "../PreviewModal";
import type { ReactElement, ReactNode } from "react";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("TemplatePreviewModal", () => {
  it("open=true 不抛错", () => {
    expect(() =>
      render(<TemplatePreviewModal open={true} content="测试内容" onClose={vi.fn()} />, { wrapper })
    ).not.toThrow();
  });

  it("open=false 不抛错", () => {
    expect(() =>
      render(<TemplatePreviewModal open={false} content="" onClose={vi.fn()} />, { wrapper })
    ).not.toThrow();
  });
});
