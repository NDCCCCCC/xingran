/**
 * Phase 88 Batch334 — pages/monitor/job/utils 测试
 */
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import {
  formatLocalTime,
  formatDuration,
  renderJobStatusTag,
  renderConcurrentTag,
  renderCronExpression,
  renderExceptionInfo,
} from "../utils";

describe("pages/monitor/job/utils", () => {
  it("formatLocalTime", () => {
    expect(formatLocalTime("2026-08-31T10:00:00")).toMatch(/2026-08-31/);
  });

  it("formatLocalTime null → '-'", () => {
    expect(formatLocalTime(null)).toBe("-");
  });

  it("formatDuration 0 → '-'", () => {
    expect(formatDuration(0)).toBe("-");
  });

  it("formatDuration <1000 → ms", () => {
    expect(formatDuration(500)).toBe("500ms");
  });

  it("formatDuration >=1000 → s", () => {
    expect(formatDuration(1500)).toBe("1.50s");
  });

  it("formatDuration 10000 → 10.00s", () => {
    expect(formatDuration(10000)).toBe("10.00s");
  });

  it("renderJobStatusTag Normal → 正常", () => {
    render(renderJobStatusTag(0));
    expect(screen.getByText("正常")).toBeInTheDocument();
  });

  it("renderJobStatusTag Paused → 暂停", () => {
    render(renderJobStatusTag(1));
    expect(screen.getByText("暂停")).toBeInTheDocument();
  });

  it("renderConcurrentTag true → 允许", () => {
    render(renderConcurrentTag(true));
    expect(screen.getByText("允许")).toBeInTheDocument();
  });

  it("renderConcurrentTag false → 禁止", () => {
    render(renderConcurrentTag(false));
    expect(screen.getByText("禁止")).toBeInTheDocument();
  });

  it("renderCronExpression 渲染 code 元素", () => {
    const { container } = render(renderCronExpression("0 0 * * *"));
    expect(container.querySelector("code")).toBeTruthy();
    expect(container.textContent).toContain("0 0 * * *");
  });

  it("renderExceptionInfo 非空 → 渲染 Tooltip", () => {
    const { container } = render(renderExceptionInfo("Error msg"));
    expect(container.textContent).toContain("查看错误");
  });

  it("renderExceptionInfo 空 → '-'", () => {
    render(renderExceptionInfo(""));
    expect(screen.getByText("-")).toBeInTheDocument();
  });
});
