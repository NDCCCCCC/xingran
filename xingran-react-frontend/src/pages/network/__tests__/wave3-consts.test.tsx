/**
 * Phase 86 — executions + templates constants 测试
 */
import { describe, it, expect } from "vitest";
import {
  STATUS_OPTIONS as EXEC_STATUS,
  STATUS_CONFIG as EXEC_CONFIG,
} from "../executions/constants";
import { VENDOR_OPTIONS, DEVICE_TYPE_OPTIONS, TEMPLATE_TYPE_OPTIONS } from "../templates/constants";

describe("executions constants (D-12)", () => {
  it("STATUS_OPTIONS non-empty", () => {
    expect(EXEC_STATUS.length).toBeGreaterThan(0);
  });

  it("STATUS_CONFIG maps statuses to color+text", () => {
    expect(Object.keys(EXEC_CONFIG).length).toBeGreaterThan(0);
  });
});

describe("templates constants (D-12)", () => {
  it("VENDOR_OPTIONS non-empty", () => {
    expect(VENDOR_OPTIONS.length).toBeGreaterThan(0);
  });

  it("DEVICE_TYPE_OPTIONS non-empty", () => {
    expect(DEVICE_TYPE_OPTIONS.length).toBeGreaterThan(0);
  });

  it("TEMPLATE_TYPE_OPTIONS non-empty", () => {
    expect(TEMPLATE_TYPE_OPTIONS.length).toBeGreaterThan(0);
  });
});
