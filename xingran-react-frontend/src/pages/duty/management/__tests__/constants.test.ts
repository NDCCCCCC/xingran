/**
 * Phase 87 — duty management constants 静态断言
 */
import { describe, it, expect } from "vitest";
import {
  SWAP_REASON_OPTIONS,
  MANUAL_REASON_OPTIONS,
  HOLIDAY_NAME_OPTIONS,
  HOLIDAY_TYPE_OPTIONS,
  MAX_BATCH_DAYS,
  DUTY_TYPE_OPTIONS,
} from "../constants";

describe("duty management constants (D-12)", () => {
  it("SWAP/MANUAL reason options non-empty", () => {
    expect(SWAP_REASON_OPTIONS.length).toBeGreaterThan(0);
    expect(MANUAL_REASON_OPTIONS.length).toBeGreaterThan(0);
  });

  it("HOLIDAY name/type options non-empty", () => {
    expect(HOLIDAY_NAME_OPTIONS.length).toBeGreaterThan(0);
    expect(HOLIDAY_TYPE_OPTIONS.length).toBeGreaterThan(0);
  });

  it("MAX_BATCH_DAYS is positive bound", () => {
    expect(MAX_BATCH_DAYS).toBeGreaterThan(0);
  });

  it("DUTY_TYPE_OPTIONS non-empty", () => {
    expect(DUTY_TYPE_OPTIONS.length).toBeGreaterThan(0);
  });
});
