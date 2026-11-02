/**
 * Phase 88 Batch413 — pages/network/templates/modals/VariablesModal 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { TemplateVariablesModal } from "../VariablesModal";
import type { ReactElement, ReactNode } from "react";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("TemplateVariablesModal", () => {
  it("空 variables 不抛错", () => {
    expect(() =>
      render(<TemplateVariablesModal open={true} variables={{}} onClose={vi.fn()} />, { wrapper })
    ).not.toThrow();
  });

  it("有 variables 不抛错", () => {
    expect(() =>
      render(
        <TemplateVariablesModal
          open={true}
          variables={{
            hostName: { default: "switch1", description: "主机名" },
            port: 22,
          }}
          onClose={vi.fn()}
        />,
        { wrapper }
      )
    ).not.toThrow();
  });

  // BUGFIX-02 regression: columns should render exactly 3 columns (key/description/defaultValue), not 6
  it("columns 渲染恰好 3 列而非 6 列", async () => {
    render(
      <TemplateVariablesModal
        open={true}
        variables={{
          hostName: { default: "switch1", description: "主机名" },
        }}
        onClose={vi.fn()}
      />,
      { wrapper }
    );
    // Count unique column headers: "变量名" should appear exactly once (not twice as in buggy 6-col state)
    await vi.waitFor(() => {
      expect(screen.queryAllByText("变量名")).toHaveLength(1);
      expect(screen.queryAllByText("描述")).toHaveLength(1);
      expect(screen.queryAllByText("默认值")).toHaveLength(1);
    });
  });
});
