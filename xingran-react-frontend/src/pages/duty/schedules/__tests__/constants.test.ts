/**
 * Phase 87 — duty schedules constants 静态断言
 */
import { describe, it, expect } from "vitest";
import { DUTY_TYPE_OPTIONS, WEEKDAY_TEXTS } from "../constants";

describe("duty schedules constants (D-12)", () => {
  it("DUTY_TYPE_OPTIONS non-empty", () => {
    expect(DUTY_TYPE_OPTIONS.length).toBeGreaterThan(0);
  });

  it("WEEKDAY_TEXTS has 7 days starting 周日", () => {
    expect(WEEKDAY_TEXTS).toHaveLength(7);
    expect(WEEKDAY_TEXTS[0]).toBe("周日");
    expect(WEEKDAY_TEXTS[6]).toBe("周六");
  });
});
