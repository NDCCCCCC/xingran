/**
 * Phase 88 Batch299 — components/shared/ModernTag 测试
 */
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

import ModernTag, { renderStatusTag } from "../ModernTag";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("shared/ModernTag", () => {
  it("默认 status='default' 显示 默认", () => {
    render(<ModernTag />, { wrapper });
    expect(screen.getByText("默认")).toBeInTheDocument();
  });

  it("status=success 显示 正常", () => {
    render(<ModernTag status="success" />, { wrapper });
    expect(screen.getByText("正常")).toBeInTheDocument();
  });

  it("status=error 显示 停用", () => {
    render(<ModernTag status="error" />, { wrapper });
    expect(screen.getByText("停用")).toBeInTheDocument();
  });

  it("status=warning 显示 警告", () => {
    render(<ModernTag status="warning" />, { wrapper });
    expect(screen.getByText("警告")).toBeInTheDocument();
  });

  it("status=processing 显示 进行中", () => {
    render(<ModernTag status="processing" />, { wrapper });
    expect(screen.getByText("进行中")).toBeInTheDocument();
  });

  it("children 覆盖默认 label", () => {
    render(<ModernTag status="success">自定义</ModernTag>, { wrapper });
    expect(screen.getByText("自定义")).toBeInTheDocument();
  });

  it("showIcon=true 渲染图标 span", () => {
    const { container } = render(<ModernTag status="success" showIcon />, { wrapper });
    const iconSpan = container.querySelector("span.anticon");
    expect(iconSpan).toBeTruthy();
  });

  it("modern=false 走原生 Tag 路径", () => {
    const { container } = render(<ModernTag status="success" modern={false} />, { wrapper });
    // native path uses Tag color attribute
    expect(container.querySelector(".ant-tag")).toBeTruthy();
  });

  it("modern=false 时 status=default 仍渲染", () => {
    render(<ModernTag status="default" modern={false} />, { wrapper });
    expect(screen.getByText("默认")).toBeInTheDocument();
  });

  it("renderStatusTag status=0 → 正常", () => {
    render(renderStatusTag(0), { wrapper });
    expect(screen.getByText("正常")).toBeInTheDocument();
  });

  it("renderStatusTag status=1 → 停用", () => {
    render(renderStatusTag(1), { wrapper });
    expect(screen.getByText("停用")).toBeInTheDocument();
  });

  it("renderStatusTag 自定义文本", () => {
    render(renderStatusTag(0, { normalText: "已启用" }), { wrapper });
    expect(screen.getByText("已启用")).toBeInTheDocument();
  });

  it("renderStatusTag 自定义停用文本", () => {
    render(renderStatusTag(1, { stopText: "已停用" }), { wrapper });
    expect(screen.getByText("已停用")).toBeInTheDocument();
  });

  it("renderStatusTag status=其他值 → 停用", () => {
    render(renderStatusTag(99), { wrapper });
    expect(screen.getByText("停用")).toBeInTheDocument();
  });
});
