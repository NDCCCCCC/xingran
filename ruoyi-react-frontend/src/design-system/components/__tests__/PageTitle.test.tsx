/**
 * Phase 88 Batch317 — design-system/components/PageTitle 测试
 */
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import PageTitle from "../PageTitle";

describe("design-system/components/PageTitle", () => {
  it("pre only → 只渲染 pre", () => {
    render(<PageTitle pre="系统" />);
    expect(screen.getByText("系统")).toBeInTheDocument();
    expect(document.querySelector(".page-title")).toBeTruthy();
    expect(document.querySelector(".dot")).toBeNull();
  });

  it("pre + post → 渲染 dot", () => {
    const { container } = render(<PageTitle pre="系统" post="用户" />);
    expect(container.textContent).toContain("系统");
    expect(container.textContent).toContain("用户");
    expect(container.querySelector(".dot")).toBeTruthy();
  });

  it("sub 渲染 page-sub", () => {
    render(<PageTitle pre="系统" sub="副标题文本" />);
    expect(screen.getByText("副标题文本")).toBeInTheDocument();
    expect(document.querySelector(".page-sub")).toBeTruthy();
  });

  it("无 sub → 不渲染 page-sub", () => {
    render(<PageTitle pre="系统" />);
    expect(document.querySelector(".page-sub")).toBeNull();
  });

  it("actions 渲染到 page-actions 容器", () => {
    render(<PageTitle pre="系统" actions={<button type="button">新增</button>} />);
    const actionsContainer = document.querySelector(".page-actions");
    expect(actionsContainer).toBeTruthy();
    expect(actionsContainer?.textContent).toContain("新增");
  });

  it("无 actions → 不渲染 page-actions", () => {
    render(<PageTitle pre="系统" />);
    expect(document.querySelector(".page-actions")).toBeNull();
  });

  it("post 空字符串 falsy → 不渲染 dot", () => {
    render(<PageTitle pre="系统" post="" />);
    expect(document.querySelector(".dot")).toBeNull();
  });
});
