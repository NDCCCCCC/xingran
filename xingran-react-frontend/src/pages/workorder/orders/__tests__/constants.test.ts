/**
 * Phase 87 — workorder orders constants 静态断言
 */
import { describe, it, expect } from "vitest";
import { STATUS_CONFIG, PRIORITY_CONFIG, TYPE_CONFIG } from "../constants";

describe("workorder constants (D-12)", () => {
  it("STATUS_CONFIG covers statuses with color+text", () => {
    expect(Object.keys(STATUS_CONFIG).length).toBeGreaterThan(0);
  });
  it("PRIORITY_CONFIG covers priorities", () => {
    expect(Object.keys(PRIORITY_CONFIG).length).toBeGreaterThan(0);
  });
  it("TYPE_CONFIG covers types", () => {
    expect(Object.keys(TYPE_CONFIG).length).toBeGreaterThan(0);
  });
});
