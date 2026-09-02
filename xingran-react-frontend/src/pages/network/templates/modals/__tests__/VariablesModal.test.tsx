/**
 * Phase 88 Batch413 — pages/network/templates/modals/VariablesModal 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
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
});
