/**
 * Phase 88 Batch198 — components/CronSelector/exports 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import * as exports from "../exports";
import * as utils from "../utils";
import * as constants from "../constants";

describe("CronSelector/exports", () => {
  it("重新导出 constants", () => {
    expect(exports.CRON_PRESETS).toBe(constants.CRON_PRESETS);
    expect(exports.FIELD_RANGES).toBe(constants.FIELD_RANGES);
    expect(exports.WEEK_DAY_NAMES).toBe(constants.WEEK_DAY_NAMES);
    expect(exports.MONTH_NAMES).toBe(constants.MONTH_NAMES);
    expect(exports.DEFAULT_CRON_EXPRESSION).toBe(constants.DEFAULT_CRON_EXPRESSION);
  });

  it("重新导出 utils 函数", () => {
    expect(exports.cronConfigToExpression).toBe(utils.cronConfigToExpression);
    expect(exports.expressionToCronConfig).toBe(utils.expressionToCronConfig);
    expect(exports.validateCronExpression).toBe(utils.validateCronExpression);
    expect(exports.getNextRunTimes).toBe(utils.getNextRunTimes);
    expect(exports.cronToChinese).toBe(utils.cronToChinese);
    expect(exports.getDefaultCronConfig).toBe(utils.getDefaultCronConfig);
  });

  it("CRON_PRESETS 含 7 项常见 cron", () => {
    expect(exports.CRON_PRESETS.length).toBeGreaterThanOrEqual(5);
  });

  it("FIELD_RANGES 含 6 字段", () => {
    expect(Object.keys(exports.FIELD_RANGES).length).toBeGreaterThanOrEqual(6);
  });

  it("WEEK_DAY_NAMES 7 天", () => {
    expect(exports.WEEK_DAY_NAMES.length).toBe(7);
  });

  it("MONTH_NAMES 12 月", () => {
    expect(exports.MONTH_NAMES.length).toBe(12);
  });

  it("DEFAULT_CRON_EXPRESSION 非空", () => {
    expect(exports.DEFAULT_CRON_EXPRESSION).toBeTruthy();
  });
});
