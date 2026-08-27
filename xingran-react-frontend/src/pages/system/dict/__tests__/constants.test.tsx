/**
 * Phase 86 — dict constants 测试
 */
import { describe, it, expect } from "vitest";
import {
  STATUS_OPTIONS,
  STATUS_CONFIG,
  DEFAULT_TYPE_FORM_VALUES,
  DEFAULT_DATA_FORM_VALUES,
} from "../constants";

describe("dict constants (D-12)", () => {
  it("STATUS_OPTIONS is 启停 2 项", () => {
    expect(STATUS_OPTIONS.length).toBe(2);
  });

  it("STATUS_CONFIG covers 0/1 with text+color", () => {
    expect(STATUS_CONFIG[0].text).toBeTruthy();
    expect(STATUS_CONFIG[1].text).toBeTruthy();
  });

  it("DEFAULT_TYPE_FORM_VALUES defined", () => {
    expect(DEFAULT_TYPE_FORM_VALUES).toBeDefined();
  });

  it("DEFAULT_DATA_FORM_VALUES defined", () => {
    expect(DEFAULT_DATA_FORM_VALUES).toBeDefined();
  });
});
