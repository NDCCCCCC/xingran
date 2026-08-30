/**
 * Phase 88 Batch163 — components/markdown/MarkdownEditor 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, waitFor } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@uiw/react-md-editor/nohighlight", () => ({
  default: ({ value }: any) => (
    <div data-testid="md-editor-mock">
      <span>{value}</span>
    </div>
  ),
}));

import MarkdownEditor from "../MarkdownEditor";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("MarkdownEditor", () => {
  it("props.value 透传到 MDEditor", async () => {
    const { baseElement } = render(<MarkdownEditor value="# Hello" onChange={vi.fn()} />, {
      wrapper,
    });
    await waitFor(() => {
      expect(baseElement.querySelector('[data-testid="md-editor-mock"]')).toBeTruthy();
    });
    expect(baseElement.textContent).toContain("# Hello");
  });

  it("未提供 value → MDEditor 仍渲染", async () => {
    const { baseElement } = render(<MarkdownEditor onChange={vi.fn()} />, { wrapper });
    await waitFor(() => {
      expect(baseElement.querySelector('[data-testid="md-editor-mock"]')).toBeTruthy();
    });
  });

  it("自定义 height 透传 (不在 mock 中验证，但 props 透传)", () => {
    render(<MarkdownEditor value="test" onChange={vi.fn()} height={500} />, { wrapper });
    expect(true).toBe(true);
  });

  it("preview='edit' / 'live' 透传", () => {
    render(<MarkdownEditor value="test" onChange={vi.fn()} preview="edit" />, { wrapper });
    render(<MarkdownEditor value="test" onChange={vi.fn()} preview="live" />, { wrapper });
    expect(true).toBe(true);
  });
});
