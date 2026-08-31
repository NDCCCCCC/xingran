/**
 * Phase 88 Batch263 — pages/duty/schedules/constants 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import {
  DUTY_TYPE_OPTIONS,
  WEEKDAY_TEXTS,
  WEEKDAY_SHORT_TEXTS,
  DUTY_STATUS_CONFIG,
  getDutyTypeColor,
  getDutyTypeText,
} from "../constants";

describe("duty/schedules/constants", () => {
  it("DUTY_TYPE_OPTIONS 3 项", () => {
    expect(DUTY_TYPE_OPTIONS.length).toBe(3);
    expect(DUTY_TYPE_OPTIONS[0].value).toBe("weekday");
  });

  it("WEEKDAY_TEXTS 7 天", () => {
    expect(WEEKDAY_TEXTS.length).toBe(7);
    expect(WEEKDAY_TEXTS[0]).toBe("周日");
  });

  it("WEEKDAY_SHORT_TEXTS 7 天", () => {
    expect(WEEKDAY_SHORT_TEXTS.length).toBe(7);
    expect(WEEKDAY_SHORT_TEXTS[0]).toBe("日");
  });

  it("DUTY_STATUS_CONFIG 3 状态", () => {
    expect(Object.keys(DUTY_STATUS_CONFIG).length).toBe(3);
    expect(DUTY_STATUS_CONFIG[0].text).toBe("正常");
    expect(DUTY_STATUS_CONFIG[1].text).toBe("已调换");
    expect(DUTY_STATUS_CONFIG[2].text).toBe("已取消");
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

  it("getDutyTypeColor 未知 → default", () => {
    expect(getDutyTypeColor("unknown")).toBe("default");
  });

  it("getDutyTypeText 已知", () => {
    expect(getDutyTypeText("weekday")).toBe("工作日");
  });

  it("getDutyTypeText 未知 → 返回原", () => {
    expect(getDutyTypeText("unknown")).toBe("unknown");
  });
});
