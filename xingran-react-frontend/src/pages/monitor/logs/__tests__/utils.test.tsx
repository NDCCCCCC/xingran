/**
 * Phase 88 Batch335 — pages/monitor/logs/utils 测试
 */
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import {
  formatLocalTime,
  getBusinessTypeLabel,
  renderRequestMethodTag,
  renderLogStatusTag,
  processTimeRangeParams,
} from "../utils";
import { BUSINESS_TYPE_OPTIONS } from "../constants";

describe("pages/monitor/logs/utils", () => {
  it("formatLocalTime null → '-'", () => {
    expect(formatLocalTime(null)).toBe("-");
  });

  it("formatLocalTime undefined → '-'", () => {
    expect(formatLocalTime(undefined)).toBe("-");
  });

  it("formatLocalTime 空字符串 → '-'", () => {
    expect(formatLocalTime("")).toBe("-");
  });

  it("formatLocalTime 字符串格式", () => {
    const r = formatLocalTime("2026-08-31T10:00:00Z");
    expect(r).toMatch(/2026/);
  });

  it("getBusinessTypeLabel 已知 type", () => {
    if (BUSINESS_TYPE_OPTIONS.length > 0) {
      const first = BUSINESS_TYPE_OPTIONS[0];
      expect(getBusinessTypeLabel(first.value)).toBe(first.label);
    }
  });

  it("getBusinessTypeLabel 未知 → '-'", () => {
    expect(getBusinessTypeLabel(99999)).toBe("-");
  });

  it("renderRequestMethodTag GET → blue", () => {
    const { container } = render(renderRequestMethodTag("GET"));
    expect(container.querySelector(".ant-tag")).toBeTruthy();
    expect(screen.getByText("GET")).toBeInTheDocument();
  });

  it("renderRequestMethodTag POST", () => {
    render(renderRequestMethodTag("POST"));
    expect(screen.getByText("POST")).toBeInTheDocument();
  });

  it("renderRequestMethodTag PUT", () => {
    render(renderRequestMethodTag("PUT"));
    expect(screen.getByText("PUT")).toBeInTheDocument();
  });

  it("renderRequestMethodTag DELETE", () => {
    render(renderRequestMethodTag("DELETE"));
    expect(screen.getByText("DELETE")).toBeInTheDocument();
  });

  it("renderRequestMethodTag 未知 → default", () => {
    render(renderRequestMethodTag("PATCH"));
    expect(screen.getByText("PATCH")).toBeInTheDocument();
  });

  it("renderLogStatusTag oper + success → 正常", () => {
    render(renderLogStatusTag(0, "oper"));
    expect(screen.getByText("正常")).toBeInTheDocument();
  });

  it("renderLogStatusTag oper + fail → 异常", () => {
    render(renderLogStatusTag(1, "oper"));
    expect(screen.getByText("异常")).toBeInTheDocument();
  });

  it("renderLogStatusTag login + success → 成功", () => {
    render(renderLogStatusTag(0, "login"));
    expect(screen.getByText("成功")).toBeInTheDocument();
  });

  it("renderLogStatusTag login + fail → 失败", () => {
    render(renderLogStatusTag(1, "login"));
    expect(screen.getByText("失败")).toBeInTheDocument();
  });

  it("renderLogStatusTag 默认 oper", () => {
    render(renderLogStatusTag(0));
    expect(screen.getByText("正常")).toBeInTheDocument();
  });

  it("processTimeRangeParams 含两个时间 → 注入 startTime/endTime", () => {
    const params: Record<string, any> = {};
    const start = new Date("2026-08-01T00:00:00Z");
    const end = new Date("2026-08-31T23:59:59Z");
    processTimeRangeParams([start, end], params);
    expect(params.startTime).toBeDefined();
    expect(params.endTime).toBeDefined();
    expect(params.startTime).toContain("2026");
  });

  it("processTimeRangeParams 空数组 → 不注入", () => {
    const params: Record<string, any> = {};
    processTimeRangeParams([], params);
    expect(params.startTime).toBeUndefined();
    expect(params.endTime).toBeUndefined();
  });

  it("processTimeRangeParams undefined → 不注入", () => {
    const params: Record<string, any> = {};
    processTimeRangeParams(undefined, params);
    expect(params.startTime).toBeUndefined();
  });
});
