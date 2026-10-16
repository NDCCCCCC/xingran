/**
 * Phase 88 Batch325 — pages/system/dept/constants 测试
 */
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import {
  STATUS_OPTIONS,
  EXTERNAL_ORG_OPTIONS,
  renderStatusTag,
  renderExternalOrgTag,
} from "../constants";

describe("pages/system/dept/constants", () => {
  it("STATUS_OPTIONS 是 NORMAL_STOP_OPTIONS 别名", () => {
    expect(STATUS_OPTIONS.length).toBeGreaterThanOrEqual(2);
  });

  it("EXTERNAL_ORG_OPTIONS 2 项", () => {
    expect(EXTERNAL_ORG_OPTIONS.length).toBe(2);
    expect(EXTERNAL_ORG_OPTIONS[0]).toEqual({ label: "否", value: 0 });
    expect(EXTERNAL_ORG_OPTIONS[1]).toEqual({ label: "是", value: 1 });
  });

  it("renderStatusTag 0 → 正常", () => {
    render(renderStatusTag(0));
    expect(screen.getByText("正常")).toBeInTheDocument();
  });

  it("renderStatusTag 1 → 停用", () => {
    render(renderStatusTag(1));
    expect(screen.getByText("停用")).toBeInTheDocument();
  });

  it("renderExternalOrgTag 1 → 是", () => {
    render(renderExternalOrgTag(1));
    expect(screen.getByText("是")).toBeInTheDocument();
  });

  it("renderExternalOrgTag 0 → 否", () => {
    render(renderExternalOrgTag(0));
    expect(screen.getByText("否")).toBeInTheDocument();
  });
});
