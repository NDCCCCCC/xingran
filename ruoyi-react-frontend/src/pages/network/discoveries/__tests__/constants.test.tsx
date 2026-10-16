/**
 * Phase 88 Batch330 — pages/network/discoveries/constants 测试
 */
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import {
  DISCOVERY_TYPE_OPTIONS,
  STATUS_OPTIONS,
  STATUS_CONFIG,
  renderStatusTag,
} from "../constants";

describe("pages/network/discoveries/constants", () => {
  it("DISCOVERY_TYPE_OPTIONS 2 项", () => {
    expect(DISCOVERY_TYPE_OPTIONS.length).toBe(2);
    expect(DISCOVERY_TYPE_OPTIONS[0].value).toBe("snmp");
  });

  it("STATUS_OPTIONS 4 项", () => {
    expect(STATUS_OPTIONS.length).toBe(4);
    expect(STATUS_OPTIONS[0].value).toBe("pending");
  });

  it("STATUS_CONFIG 4 状态", () => {
    expect(Object.keys(STATUS_CONFIG).length).toBe(4);
  });

  it("STATUS_CONFIG pending default", () => {
    expect(STATUS_CONFIG.pending.color).toBe("default");
  });

  it("STATUS_CONFIG completed success", () => {
    expect(STATUS_CONFIG.completed.color).toBe("success");
  });

  it("STATUS_CONFIG failed error", () => {
    expect(STATUS_CONFIG.failed.color).toBe("error");
  });

  it("renderStatusTag pending", () => {
    render(renderStatusTag("pending"));
    expect(screen.getByText("待执行")).toBeInTheDocument();
  });

  it("renderStatusTag running", () => {
    render(renderStatusTag("running"));
    expect(screen.getByText("扫描中")).toBeInTheDocument();
  });

  it("renderStatusTag completed", () => {
    render(renderStatusTag("completed"));
    expect(screen.getByText("已完成")).toBeInTheDocument();
  });

  it("renderStatusTag failed", () => {
    render(renderStatusTag("failed"));
    expect(screen.getByText("失败")).toBeInTheDocument();
  });

  it("renderStatusTag unknown → pending", () => {
    render(renderStatusTag("xyz" as any));
    expect(screen.getByText("待执行")).toBeInTheDocument();
  });
});
