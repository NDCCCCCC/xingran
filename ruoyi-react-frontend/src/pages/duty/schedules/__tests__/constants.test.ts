/**
 * Phase 88 Batch311 — pages/duty/schedules/constants 测试
 */
import { describe, it, expect } from "vitest";
import {
  DUTY_TYPE_OPTIONS,
  WEEKDAY_TEXTS,
  WEEKDAY_SHORT_TEXTS,
  DUTY_STATUS_CONFIG,
  getDutyTypeColor,
  getDutyTypeText,
} from "../constants";

describe("pages/duty/schedules/constants", () => {
  it("DUTY_TYPE_OPTIONS 3 项", () => {
    expect(DUTY_TYPE_OPTIONS.length).toBe(3);
  });

  it("DUTY_TYPE_OPTIONS 正确", () => {
    expect(DUTY_TYPE_OPTIONS[0]).toEqual({ label: "工作日", value: "weekday" });
    expect(DUTY_TYPE_OPTIONS[1]).toEqual({ label: "周末", value: "weekend" });
    expect(DUTY_TYPE_OPTIONS[2]).toEqual({ label: "节假日", value: "holiday" });
  });

  it("WEEKDAY_TEXTS 7 项 周日-周六", () => {
    expect(WEEKDAY_TEXTS.length).toBe(7);
    expect(WEEKDAY_TEXTS[0]).toBe("周日");
    expect(WEEKDAY_TEXTS[6]).toBe("周六");
  });

  it("WEEKDAY_SHORT_TEXTS 7 项", () => {
    expect(WEEKDAY_SHORT_TEXTS.length).toBe(7);
    expect(WEEKDAY_SHORT_TEXTS[1]).toBe("一");
  });

  it("DUTY_STATUS_CONFIG 3 状态", () => {
    expect(DUTY_STATUS_CONFIG[0].text).toBe("正常");
    expect(DUTY_STATUS_CONFIG[1].text).toBe("已调换");
    expect(DUTY_STATUS_CONFIG[2].text).toBe("已取消");
  });

  it("DUTY_STATUS_CONFIG colors", () => {
    expect(DUTY_STATUS_CONFIG[0].color).toBe("green");
    expect(DUTY_STATUS_CONFIG[1].color).toBe("orange");
    expect(DUTY_STATUS_CONFIG[2].color).toBe("red");
  });

  it("getDutyTypeColor weekday → blue", () => {
    expect(getDutyTypeColor("weekday")).toBe("blue");
  });

  it("getDutyTypeColor weekend → orange", () => {
    expect(getDutyTypeColor("weekend")).toBe("orange");
  });

  it("getDutyTypeColor holiday → red", () => {
    expect(getDutyTypeColor("holiday")).toBe("red");
  });

  it("getDutyTypeColor unknown → default", () => {
    expect(getDutyTypeColor("unknown")).toBe("default");
  });

  it("getDutyTypeText weekday → 工作日", () => {
    expect(getDutyTypeText("weekday")).toBe("工作日");
  });

  it("getDutyTypeText weekend → 周末", () => {
    expect(getDutyTypeText("weekend")).toBe("周末");
  });

  it("getDutyTypeText holiday → 节假日", () => {
    expect(getDutyTypeText("holiday")).toBe("节假日");
  });

  it("getDutyTypeText 未知值 → 原值", () => {
    expect(getDutyTypeText("custom-type")).toBe("custom-type");
  });
});
