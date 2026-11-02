/**
 * Phase 88 Batch327 — pages/system/dict/constants 测试
 */
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import {
  STATUS_OPTIONS,
  STATUS_CONFIG,
  DEFAULT_TYPE_FORM_VALUES,
  DEFAULT_DATA_FORM_VALUES,
  renderStatusTag,
} from "../constants";

describe("pages/system/dict/constants", () => {
  it("STATUS_OPTIONS 至少 2 项", () => {
    expect(STATUS_OPTIONS.length).toBeGreaterThanOrEqual(2);
  });

  it("STATUS_CONFIG 含 0 + 1", () => {
    expect(STATUS_CONFIG[0]).toBeDefined();
    expect(STATUS_CONFIG[1]).toBeDefined();
  });

  it("DEFAULT_TYPE_FORM_VALUES", () => {
    expect(DEFAULT_TYPE_FORM_VALUES.status).toBe(0);
  });

  it("DEFAULT_DATA_FORM_VALUES", () => {
    expect(DEFAULT_DATA_FORM_VALUES.dictSort).toBe(0);
    expect(DEFAULT_DATA_FORM_VALUES.status).toBe(0);
    expect(DEFAULT_DATA_FORM_VALUES.isDefault).toBe(false);
  });

  it("renderStatusTag 0", () => {
    render(renderStatusTag(0));
    expect(screen.getByText(STATUS_CONFIG[0].text)).toBeInTheDocument();
  });

  it("renderStatusTag 1", () => {
    render(renderStatusTag(1));
    expect(screen.getByText(STATUS_CONFIG[1].text)).toBeInTheDocument();
  });
});
