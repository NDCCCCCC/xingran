/**
 * Phase 88 Batch73 — CronSelector utils 单元测试
 */
import { describe, it, expect } from "vitest";
import { cronConfigToExpression, expressionToCronConfig, validateCronExpression } from "../utils";
import type { CronConfig } from "../constants";

const everyConfig: CronConfig = {
  second: { periodType: "every" },
  minute: { periodType: "every" },
  hour: { periodType: "every" },
  day: { periodType: "every" },
  month: { periodType: "every" },
  week: { periodType: "every" },
} as unknown as CronConfig;

const specificConfig: CronConfig = {
  second: { periodType: "specific", specific: [15] },
  minute: { periodType: "specific", specific: [0] },
  hour: { periodType: "specific", specific: [12] },
  day: { periodType: "every" },
  month: { periodType: "every" },
  week: { periodType: "every" },
} as unknown as CronConfig;

const rangeConfig: CronConfig = {
  second: { periodType: "every" },
  minute: { periodType: "range", rangeStart: 0, rangeEnd: 30 },
  hour: { periodType: "every" },
  day: { periodType: "every" },
  month: { periodType: "every" },
  week: { periodType: "every" },
} as unknown as CronConfig;

const cycleConfig: CronConfig = {
  second: { periodType: "cycle", cycleStart: 0, cycleInterval: 5 },
  minute: { periodType: "every" },
  hour: { periodType: "every" },
  day: { periodType: "every" },
  month: { periodType: "every" },
  week: { periodType: "every" },
} as unknown as CronConfig;

describe("cronConfigToExpression", () => {
  it("全 every → '* * * * * *'", () => {
    expect(cronConfigToExpression(everyConfig)).toBe("* * * * * *");
  });

  it("specific second=15 → '15 * * * * *'", () => {
    expect(cronConfigToExpression(specificConfig)).toBe("15 0 12 * * *");
  });

  it("range minute 0-30 → '* 0-30 * * * *'", () => {
    expect(cronConfigToExpression(rangeConfig)).toBe("* 0-30 * * * *");
  });

  it("cycle second 0/5 → '0/5 * * * * *'", () => {
    expect(cronConfigToExpression(cycleConfig)).toBe("0/5 * * * * *");
  });
});

describe("expressionToCronConfig", () => {
  it("'* * * * * *' → everyConfig", () => {
    const cfg = expressionToCronConfig("* * * * * *");
    expect(cfg.second.periodType).toBe("every");
    expect(cfg.minute.periodType).toBe("every");
  });

  it("'15 0 12 * * *' → specific hour=12", () => {
    const cfg = expressionToCronConfig("15 0 12 * * *");
    expect(cfg.second.specific).toContain(15);
    expect(cfg.hour.specific).toContain(12);
  });

  it("5 字段表达式自动补 0 秒", () => {
    const cfg = expressionToCronConfig("0 12 * * *");
    expect(cfg.second.periodType).toBe("specific");
    expect(cfg.second.specific).toContain(0);
  });
});

describe("validateCronExpression", () => {
  it("合法表达式返回 true", () => {
    expect(validateCronExpression("0 0 12 * * *")).toBe(true);
  });

  it("非法表达式返回 false", () => {
    expect(validateCronExpression("not a cron")).toBe(false);
  });
});
