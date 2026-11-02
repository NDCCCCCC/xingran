/**
 * Phase 88 Batch403 — components/shared/BatchExportModal 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import BatchExportModal from "../BatchExportModal";
import type { ReactElement, ReactNode } from "react";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("components/shared/BatchExportModal", () => {
  it("导出为函数组件", async () => {
    const mod = await import("../BatchExportModal");
    expect(typeof mod.default).toBe("function");
  });

  it("visible=false 时不渲染内容", () => {
    const { container } = render(
      <BatchExportModal visible={false} onCancel={vi.fn()} onConfirm={vi.fn()} />,
      { wrapper }
    );
    expect(container.querySelector(".ant-modal")).toBeNull();
  });

  it("visible=true 不抛错", () => {
    expect(() =>
      render(<BatchExportModal visible={true} onCancel={vi.fn()} onConfirm={vi.fn()} />, {
        wrapper,
      })
    ).not.toThrow();
  });
});
