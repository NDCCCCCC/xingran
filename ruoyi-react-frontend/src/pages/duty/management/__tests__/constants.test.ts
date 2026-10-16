/**
 * Phase 88 Batch314 — pages/duty/management/constants 测试
 */
import { describe, it, expect } from "vitest";
import {
  SWAP_REASON_OPTIONS,
  MANUAL_REASON_OPTIONS,
  HOLIDAY_NAME_OPTIONS,
  HOLIDAY_TYPE_OPTIONS,
  HOLIDAY_REMARK_OPTIONS,
  BATCH_HOLIDAY_NAME_OPTIONS,
  MAX_BATCH_DAYS,
  DUTY_TYPE_OPTIONS,
} from "../constants";

describe("pages/duty/management/constants", () => {
  it("SWAP_REASON_OPTIONS 4 项", () => {
    expect(SWAP_REASON_OPTIONS.length).toBe(4);
    expect(SWAP_REASON_OPTIONS[0].value).toBe("临时有事");
    expect(SWAP_REASON_OPTIONS[3].value).toBe("其他");
  });

  it("MANUAL_REASON_OPTIONS 4 项", () => {
    expect(MANUAL_REASON_OPTIONS.length).toBe(4);
    expect(MANUAL_REASON_OPTIONS[1].value).toBe("替班");
  });

  it("HOLIDAY_NAME_OPTIONS 7 项", () => {
    expect(HOLIDAY_NAME_OPTIONS.length).toBe(7);
    expect(HOLIDAY_NAME_OPTIONS.map((o) => o.value)).toEqual([
      "元旦",
      "春节",
      "清明节",
      "劳动节",
      "端午节",
      "中秋节",
      "国庆节",
    ]);
  });

  it("HOLIDAY_TYPE_OPTIONS 3 项", () => {
    expect(HOLIDAY_TYPE_OPTIONS.length).toBe(3);
    expect(HOLIDAY_TYPE_OPTIONS.map((o) => o.value)).toEqual(["legal", "workday", "custom"]);
  });

  it("HOLIDAY_REMARK_OPTIONS 2 项", () => {
    expect(HOLIDAY_REMARK_OPTIONS.length).toBe(2);
    expect(HOLIDAY_REMARK_OPTIONS[0].value).toBe("放假");
    expect(HOLIDAY_REMARK_OPTIONS[1].value).toBe("调休");
  });

  it("BATCH_HOLIDAY_NAME_OPTIONS 2 项", () => {
    expect(BATCH_HOLIDAY_NAME_OPTIONS.length).toBe(2);
    expect(BATCH_HOLIDAY_NAME_OPTIONS[0].value).toBe("春节");
  });

  it("MAX_BATCH_DAYS = 90", () => {
    expect(MAX_BATCH_DAYS).toBe(90);
  });

  it("DUTY_TYPE_OPTIONS 3 项", () => {
    expect(DUTY_TYPE_OPTIONS.length).toBe(3);
    expect(DUTY_TYPE_OPTIONS.map((o) => o.value)).toEqual(["weekday", "weekend", "holiday"]);
  });
});
