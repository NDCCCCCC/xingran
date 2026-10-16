/**
 * Phase 88 Batch349 — components/markdown/MarkdownEditor 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@uiw/react-md-editor/nohighlight", () => ({
  default: ({ value, height }: any) => (
    <div data-testid="md-editor" data-value={value} data-height={height}>
      {value}
    </div>
  ),
}));

import MarkdownEditor from "../MarkdownEditor";

describe("components/markdown/MarkdownEditor", () => {
  it("加载编辑器 → 渲染 mock", async () => {
    render(<MarkdownEditor value="# Hello" />);
    await waitFor(() => {
      expect(screen.getByTestId("md-editor")).toBeInTheDocument();
    });
  });

  it("value 透传", async () => {
    render(<MarkdownEditor value="content test" />);
    await waitFor(() => {
      expect(screen.getByTestId("md-editor")).toBeInTheDocument();
    });
    expect(screen.getByText("content test")).toBeInTheDocument();
  });

  it("height 透传", async () => {
    render(<MarkdownEditor value="x" height={400} />);
    await waitFor(() => {
      const el = screen.getByTestId("md-editor");
      expect(el.getAttribute("data-height")).toBe("400");
    });
  });

  it("自定义 preview 模式", async () => {
    render(<MarkdownEditor value="x" preview="edit" />);
    await waitFor(() => {
      expect(screen.getByTestId("md-editor")).toBeInTheDocument();
    });
  });
});
