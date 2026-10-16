/**
 * Phase 88 Batch332 — pages/network/executions/constants 测试
 */
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import {
  STATUS_OPTIONS,
  STATUS_CONFIG,
  renderStatusTag,
  renderSimpleStatusTag,
} from "../constants";

describe("pages/network/executions/constants", () => {
  it("STATUS_OPTIONS 4 项", () => {
    expect(STATUS_OPTIONS.length).toBe(4);
  });

  it("STATUS_CONFIG 4 状态", () => {
    expect(Object.keys(STATUS_CONFIG).length).toBe(4);
  });

  it("STATUS_CONFIG success 绿", () => {
    expect(STATUS_CONFIG.success.color).toBe("success");
    expect(STATUS_CONFIG.success.text).toBe("成功");
  });

  it("renderStatusTag pending", () => {
    render(renderStatusTag("pending"));
    expect(screen.getByText("待执行")).toBeInTheDocument();
  });

  it("renderStatusTag running", () => {
    render(renderStatusTag("running"));
    expect(screen.getByText("执行中")).toBeInTheDocument();
  });

  it("renderStatusTag success", () => {
    render(renderStatusTag("success"));
    expect(screen.getByText("成功")).toBeInTheDocument();
  });

  it("renderStatusTag failed", () => {
    render(renderStatusTag("failed"));
    expect(screen.getByText("失败")).toBeInTheDocument();
  });

  it("renderStatusTag unknown → pending", () => {
    render(renderStatusTag("xyz" as any));
    expect(screen.getByText("待执行")).toBeInTheDocument();
  });

  it("renderSimpleStatusTag success", () => {
    render(renderSimpleStatusTag("success"));
    expect(screen.getByText("成功")).toBeInTheDocument();
  });

  it("renderSimpleStatusTag unknown → pending", () => {
    render(renderSimpleStatusTag("xyz" as any));
    expect(screen.getByText("待执行")).toBeInTheDocument();
  });
});
