/**
 * Phase 88 Batch329 — pages/network/command/constants 测试
 */
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import {
  STATUS_OPTIONS,
  STATUS_CONFIG,
  SIMPLE_STATUS_CONFIG,
  renderExecutionStatusTag,
  renderSimpleStatusTag,
} from "../constants";

describe("pages/network/command/constants", () => {
  it("STATUS_OPTIONS 4 项", () => {
    expect(STATUS_OPTIONS.length).toBe(4);
    expect(STATUS_OPTIONS.map((o) => o.value)).toEqual(["pending", "running", "success", "failed"]);
  });

  it("STATUS_CONFIG 4 状态", () => {
    expect(Object.keys(STATUS_CONFIG).length).toBe(4);
  });

  it("SIMPLE_STATUS_CONFIG 4 状态", () => {
    expect(Object.keys(SIMPLE_STATUS_CONFIG).length).toBe(4);
  });

  it("STATUS_CONFIG success 绿色", () => {
    expect(STATUS_CONFIG.success.color).toBe("success");
    expect(STATUS_CONFIG.success.text).toBe("成功");
  });

  it("STATUS_CONFIG failed 红色", () => {
    expect(STATUS_CONFIG.failed.color).toBe("error");
    expect(STATUS_CONFIG.failed.text).toBe("失败");
  });

  it("renderExecutionStatusTag success", () => {
    render(renderExecutionStatusTag("success"));
    expect(screen.getByText("成功")).toBeInTheDocument();
  });

  it("renderExecutionStatusTag failed", () => {
    render(renderExecutionStatusTag("failed"));
    expect(screen.getByText("失败")).toBeInTheDocument();
  });

  it("renderExecutionStatusTag unknown → pending", () => {
    render(renderExecutionStatusTag("xyz"));
    expect(screen.getByText("待执行")).toBeInTheDocument();
  });

  it("renderSimpleStatusTag running", () => {
    render(renderSimpleStatusTag("running"));
    expect(screen.getByText("执行中")).toBeInTheDocument();
  });

  it("renderSimpleStatusTag unknown → pending", () => {
    render(renderSimpleStatusTag("unknown"));
    expect(screen.getByText("待执行")).toBeInTheDocument();
  });
});
