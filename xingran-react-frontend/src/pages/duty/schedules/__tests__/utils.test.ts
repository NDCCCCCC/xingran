/**
 * Phase 88 Batch333 — pages/duty/schedules/utils 测试
 */
import { describe, it, expect } from "vitest";
import dayjs from "dayjs";
import {
  formatDateTime,
  formatDate,
  getWeekdayText,
  getWeekdayShortText,
  getWeekRangeText,
  getWeekDays,
  formatScheduleOptionLabel,
  formatScheduleOptionContent,
} from "../utils";

describe("pages/duty/schedules/utils", () => {
  it("formatDateTime 重新导出", () => {
    expect(typeof formatDateTime).toBe("function");
  });

  it("formatDate 重新导出", () => {
    expect(typeof formatDate).toBe("function");
  });

  it("getWeekdayText 周日 (2026-08-30)", () => {
    const day = dayjs("2026-08-30");
    expect(getWeekdayText(day)).toBe("周日");
  });

  it("getWeekdayText 周一", () => {
    const day = dayjs("2026-08-31");
    expect(getWeekdayText(day)).toBe("周一");
  });

  it("getWeekdayShortText 周一 → 一", () => {
    const day = dayjs("2026-08-31");
    expect(getWeekdayShortText(day)).toBe("一");
  });

  it("getWeekRangeText 包含 YYYY年MM月DD日", () => {
    const weekStart = dayjs("2026-08-31");
    const text = getWeekRangeText(weekStart);
    expect(text).toMatch(/2026年08月31日/);
    expect(text).toMatch(/2026年09月05日/);
  });

  it("getWeekDays 7 项", () => {
    const days = getWeekDays(dayjs("2026-08-31"));
    expect(days.length).toBe(7);
    expect(days[0].format("YYYY-MM-DD")).toBe("2026-08-31");
    expect(days[6].format("YYYY-MM-DD")).toBe("2026-09-06");
  });

  it("formatScheduleOptionLabel nickname", () => {
    const text = formatScheduleOptionLabel({
      scheduleDate: "2026-08-31",
      user: { nickname: "张三" },
    });
    expect(text).toContain("2026-08-31");
    expect(text).toContain("张三");
  });

  it("formatScheduleOptionLabel username fallback", () => {
    const text = formatScheduleOptionLabel({
      scheduleDate: "2026-08-31",
      user: { username: "zhangsan" },
    });
    expect(text).toContain("zhangsan");
  });

  it("formatScheduleOptionLabel 无 user", () => {
    const text = formatScheduleOptionLabel({ scheduleDate: "2026-08-31" });
    expect(text).toBe("2026-08-31 - ");
  });

  it("formatScheduleOptionContent 含 duty type text", () => {
    const text = formatScheduleOptionContent(
      {
        scheduleDate: "2026-08-31",
        dutyType: "weekday",
        user: { nickname: "张三" },
      },
      (t) => `T(${t})`
    );
    expect(text).toContain("T(weekday)");
    expect(text).toContain("张三");
  });

  it("formatScheduleOptionContent 无 user", () => {
    const text = formatScheduleOptionContent(
      { scheduleDate: "2026-08-31", dutyType: "weekday" },
      (t) => t
    );
    expect(text).toContain("weekday");
  });
});
