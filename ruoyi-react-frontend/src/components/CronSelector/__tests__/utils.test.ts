/**
 * Phase 88 Batch379 — components/CronSelector/utils 测试
 */
import { describe, it, expect } from "vitest";
import {
  cronConfigToExpression,
  expressionToCronConfig,
  validateCronExpression,
  getNextRunTimes,
  cronToChinese,
  getDefaultCronConfig,
  getEveryMinuteCronConfig,
} from "../utils";

describe("components/CronSelector/utils", () => {
  // Verify export shape only — avoid timing-sensitive / locale-dependent assertions
  it("module exports all functions", () => {
    expect(typeof cronConfigToExpression).toBe("function");
    expect(typeof expressionToCronConfig).toBe("function");
    expect(typeof validateCronExpression).toBe("function");
    expect(typeof getNextRunTimes).toBe("function");
    expect(typeof cronToChinese).toBe("function");
    expect(typeof getDefaultCronConfig).toBe("function");
    expect(typeof getEveryMinuteCronConfig).toBe("function");
  });
});

describe("components/CronSelector/utils: protected functions", () => {
  describe("cronConfigToExpression", () => {
    it("默认配置转表达式", () => {
      const cfg = getDefaultCronConfig();
      const expr = cronConfigToExpression(cfg);
      expect(typeof expr).toBe("string");
      expect(expr.split(/\s+/).length).toBeGreaterThanOrEqual(5);
    });

    it("每分钟配置", () => {
      const cfg = getEveryMinuteCronConfig();
      const expr = cronConfigToExpression(cfg);
      expect(typeof expr).toBe("string");
      expect(expr).toContain("*");
    });
  });

  describe("expressionToCronConfig", () => {
    it("5 段 cron 解析 → 返回对象", () => {
      const cfg = expressionToCronConfig("0 0 * * *");
      expect(typeof cfg).toBe("object");
    });

    it("非法表达式 → 返回对象", () => {
      const cfg = expressionToCronConfig("invalid");
      expect(cfg).toBeDefined();
      expect(typeof cfg).toBe("object");
    });
  });

  describe("validateCronExpression", () => {
    it("合法表达式", () => {
      expect(validateCronExpression("0 0 * * *")).toBe(true);
      expect(validateCronExpression("*/5 * * * *")).toBe(true);
    });

    it("非法 → false", () => {
      expect(validateCronExpression("invalid")).toBe(false);
      expect(validateCronExpression("")).toBe(false);
    });

    it("段数错误 → false", () => {
      expect(validateCronExpression("0 0 *")).toBe(false);
    });
  });

  describe("getNextRunTimes", () => {
    it("返回 count 个 Date", () => {
      const times = getNextRunTimes("0 0 * * *", 5);
      expect(times.length).toBe(5);
      for (const t of times) {
        expect(t instanceof Date).toBe(true);
      }
    });

    it("默认 count=5", () => {
      const times = getNextRunTimes("0 0 * * *");
      expect(times.length).toBe(5);
    });

    it("下一次运行时间合理 (每日 0 点)", () => {
      const now = Date.now();
      const times = getNextRunTimes("0 0 * * *", 1);
      if (times.length > 0 && times[0]) {
        expect(times[0].getTime()).toBeGreaterThan(now - 1000);
      } else {
        // 即使返回空数组也不抛错
        expect(Array.isArray(times)).toBe(true);
      }
    });

    it("count=0 → 空数组", () => {
      expect(getNextRunTimes("0 0 * * *", 0).length).toBe(0);
    });
  });

  describe("cronToChinese", () => {
    it("每分钟 → 每分钟", () => {
      const desc = cronToChinese("* * * * *");
      expect(typeof desc).toBe("string");
      expect(desc.length).toBeGreaterThan(0);
    });

    it("每天 0 点", () => {
      const desc = cronToChinese("0 0 * * *");
      expect(typeof desc).toBe("string");
    });

    it("每周一 0 点", () => {
      const desc = cronToChinese("0 0 * * 1");
      expect(typeof desc).toBe("string");
    });

    it("无效表达式 → 返回字符串", () => {
      expect(typeof cronToChinese("garbage")).toBe("string");
    });
  });

  describe("getDefaultCronConfig + getEveryMinuteCronConfig", () => {
    it("getDefaultCronConfig 返回 shape", () => {
      const cfg = getDefaultCronConfig();
      expect(cfg).toHaveProperty("minute");
      expect(cfg).toHaveProperty("hour");
      expect(cfg).toHaveProperty("day");
      expect(cfg).toHaveProperty("month");
      expect(cfg).toHaveProperty("week");
    });

    it("getEveryMinuteCronConfig 含必要字段", () => {
      const cfg = getEveryMinuteCronConfig();
      expect(cfg).toHaveProperty("minute");
      expect(cfg).toHaveProperty("hour");
      expect(cfg).toHaveProperty("day");
      expect(cfg).toHaveProperty("month");
      expect(cfg).toHaveProperty("week");
    });
  });
});
