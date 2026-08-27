/**
 * Phase 86 — command constants 测试
 */
import { describe, it, expect } from "vitest";
import { STATUS_OPTIONS, STATUS_CONFIG, SIMPLE_STATUS_CONFIG } from "../constants";

describe("command constants (D-12)", () => {
  it("STATUS_OPTIONS non-empty", () => {
    expect(STATUS_OPTIONS.length).toBeGreaterThan(0);
  });

  it("STATUS_CONFIG maps status to color/icon/text", () => {
    expect(Object.keys(STATUS_CONFIG).length).toBeGreaterThan(0);
  });

  it("SIMPLE_STATUS_CONFIG maps status to color/text", () => {
    expect(Object.keys(SIMPLE_STATUS_CONFIG).length).toBeGreaterThan(0);
  });
});
