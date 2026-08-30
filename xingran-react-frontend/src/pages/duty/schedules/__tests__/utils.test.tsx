/**
 * Phase 88 Batch176 — pages/duty/schedules/utils 测试
 */
import { describe, it, expect } from "vitest";
import dayjs from "dayjs";
import {
  getWeekdayText,
  getWeekdayShortText,
  getWeekRangeText,
  getWeekDays,
  formatScheduleOptionLabel,
  formatDateTime,
  formatDate,
} from "../utils";

describe("duty/schedules/utils", () => {
  it("getWeekdayText → 返回 WEEKDAY_TEXTS[day.day()]", () => {
    const sunday = dayjs("2026-01-04"); // 周日
    expect(getWeekdayText(sunday)).toContain("日");
  });

  it("getWeekdayShortText → 返回 WEEKDAY_SHORT_TEXTS", () => {
    const sunday = dayjs("2026-01-04");
    expect(getWeekdayShortText(sunday)).toBeDefined();
  });

  it("getWeekRangeText → YYYY年MM月DD日 - YYYY年MM月DD日", () => {
    const weekStart = dayjs("2026-01-05");
    const result = getWeekRangeText(weekStart);
    expect(result).toContain("2026年01月05日");
    expect(result).toContain("-");
  });

  it("getWeekDays → 返回 7 个 dayjs", () => {
    const weekStart = dayjs("2026-01-05");
    const days = getWeekDays(weekStart);
    expect(days.length).toBe(7);
    expect(days[0].format("YYYY-MM-DD")).toBe("2026-01-05");
    expect(days[6].format("YYYY-MM-DD")).toBe("2026-01-11");
  });

  it("formatScheduleOptionLabel → date - nickname", () => {
    const schedule = {
      scheduleDate: "2026-01-05",
      user: { nickname: "Alice", username: "alice" },
    };
    const result = formatScheduleOptionLabel(schedule as any);
    expect(result).toContain("Alice");
    expect(result).toContain("-");
  });

  it("formatScheduleOptionLabel → date - username (无 nickname)", () => {
    const schedule = {
      scheduleDate: "2026-01-05",
      user: { username: "alice" },
    };
    const result = formatScheduleOptionLabel(schedule as any);
    expect(result).toContain("alice");
  });

  it("formatScheduleOptionLabel → date - (无 user)", () => {
    const schedule = { scheduleDate: "2026-01-05" };
    const result = formatScheduleOptionLabel(schedule as any);
    expect(result).toBeDefined();
    expect(result).toContain("2026-01-05");
  });

  it("formatDateTime re-export", () => {
    expect(typeof formatDateTime).toBe("function");
  });

  it("formatDate re-export", () => {
    expect(typeof formatDate).toBe("function");
  });
});
