/**
 * Phase 88 Batch188 — pages/duty/schedules/utils 测试
 */
import { describe, it, expect, vi } from "vitest";
import dayjs from "dayjs";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/utils/datetime", () => ({
  formatDateTime: vi.fn(() => "2026-08-30 10:00:00"),
  formatDate: vi.fn(() => "2026-08-30"),
}));

import {
  getWeekdayText,
  getWeekdayShortText,
  getWeekRangeText,
  getWeekDays,
  formatScheduleOptionLabel,
  formatScheduleOptionContent,
} from "../utils";

describe("duty/schedules/utils", () => {
  it("getWeekdayText 2026-08-30 是周日", () => {
    const day = dayjs("2026-08-30");
    expect(getWeekdayText(day)).toBe("周日");
  });

  it("getWeekdayText 2026-08-31 是周一", () => {
    const day = dayjs("2026-08-31");
    expect(getWeekdayText(day)).toBe("周一");
  });

  it("getWeekdayShortText 周日 → 日", () => {
    const day = dayjs("2026-08-30");
    expect(getWeekdayShortText(day)).toBe("日");
  });

  it("getWeekRangeText", () => {
    const week = dayjs("2026-08-30");
    expect(getWeekRangeText(week)).toContain("2026年08月30日");
  });

  it("getWeekDays 返回 7 天", () => {
    const week = dayjs("2026-08-30");
    const days = getWeekDays(week);
    expect(days.length).toBe(7);
    expect(days[0].format("YYYY-MM-DD")).toBe("2026-08-30");
    expect(days[6].format("YYYY-MM-DD")).toBe("2026-09-05");
  });

  it("formatScheduleOptionLabel nickname", () => {
    expect(
      formatScheduleOptionLabel({
        scheduleDate: "2026-08-30",
        user: { nickname: "张三" },
      })
    ).toBe("2026-08-30 - 张三");
  });

  it("formatScheduleOptionLabel username fallback", () => {
    expect(
      formatScheduleOptionLabel({
        scheduleDate: "2026-08-30",
        user: { username: "zhangsan" },
      })
    ).toBe("2026-08-30 - zhangsan");
  });

  it("formatScheduleOptionLabel 无 user", () => {
    expect(formatScheduleOptionLabel({ scheduleDate: "2026-08-30" })).toBe("2026-08-30 - ");
  });

  it("formatScheduleOptionContent dutyType", () => {
    expect(
      formatScheduleOptionContent(
        {
          scheduleDate: "2026-08-30",
          dutyType: "primary",
          user: { nickname: "李四" },
        },
        (t) => `duty:${t}`
      )
    ).toBe("2026-08-30 (duty:primary) - 李四");
  });
});
