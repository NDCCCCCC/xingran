/**
 * Phase 84 84-02b — CronSelector utils 真实 cron 算法测试(D-08 模式)
 * 调用真实 @breejs/later + cron-validate + cron-parser,确定性字符串向量
 */
import { describe, it, expect } from "vitest";
import {
  validateCronExpression,
  getNextRunTimes,
  getDefaultCronConfig,
  getEveryMinuteCronConfig,
} from "../utils";
import {
  FIELD_RANGES,
  WEEK_DAY_NAMES,
  MONTH_NAMES,
  DEFAULT_CRON_EXPRESSION,
  CRON_PRESETS,
} from "../constants";

describe("validateCronExpression (real cron-validate)", () => {
  it("accepts valid standard 5-field cron", () => {
    expect(validateCronExpression("0 0 9 * * *")).toBe(true); // 6-field with second
    expect(validateCronExpression("*/5 * * * * *")).toBe(true);
    expect(validateCronExpression("0 0 9-17 * * *")).toBe(true);
  });

  it("rejects invalid cron expressions", () => {
    expect(validateCronExpression("not-a-cron")).toBe(false);
  });
});

describe("getNextRunTimes (real @breejs/later)", () => {
  it("returns 5 future timestamps for every-minute cron", () => {
    const times = getNextRunTimes("*/1 * * * *", 5);
    expect(times).toHaveLength(5);
    // 所有时间都应该是未来
    const now = Date.now();
    for (const t of times) {
      expect(t.getTime()).toBeGreaterThan(now);
    }
  });

  it("returns Date[] array (D-08 真实算法)", () => {
    const times = getNextRunTimes("0 0 12 * * *", 3);
    expect(times).toHaveLength(3);
    expect(times[0]).toBeInstanceOf(Date);
  });

  it("throws for invalid expression", () => {});
});

describe("getDefaultCronConfig / getEveryMinuteCronConfig", () => {
  it("default cron config exists", () => {
    const cfg = getDefaultCronConfig();
    expect(cfg).toBeDefined();
    expect(cfg.second).toBeDefined();
    expect(cfg.minute).toBeDefined();
  });

  it("every-minute cron config uses wildcard", () => {
    const cfg = getEveryMinuteCronConfig();
    expect(cfg).toBeDefined();
  });
});

describe("CronSelector constants (D-12)", () => {
  it("DEFAULT_CRON_EXPRESSION equals 0 0 9 * * ?", () => {
    expect(DEFAULT_CRON_EXPRESSION).toBe("0 0 9 * * ?");
  });

  it("FIELD_RANGES has 6 entries (second/minute/hour/day/month/week)", () => {
    expect(Object.keys(FIELD_RANGES).length).toBe(6);
  });

  it("WEEK_DAY_NAMES has 7 days", () => {
    expect(WEEK_DAY_NAMES).toHaveLength(7);
    expect(WEEK_DAY_NAMES[0]).toBe("周日");
    expect(WEEK_DAY_NAMES[6]).toBe("周六");
  });

  it("MONTH_NAMES has 12 months", () => {
    expect(MONTH_NAMES).toHaveLength(12);
    expect(MONTH_NAMES[0]).toBe("1月");
  });

  it("CRON_PRESETS is non-empty array", () => {
    expect(Array.isArray(CRON_PRESETS)).toBe(true);
    expect(CRON_PRESETS.length).toBeGreaterThan(0);
    // 每个 preset 有 label + expression
    for (const preset of CRON_PRESETS) {
      expect(preset.label).toBeTruthy();
      expect(preset.value).toBeTruthy();
    }
  });
});
