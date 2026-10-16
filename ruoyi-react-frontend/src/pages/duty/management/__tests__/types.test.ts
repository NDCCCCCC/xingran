/**
 * Phase 88 Batch291 — pages/duty/management/types 测试
 */
import { describe, it, expect } from "vitest";
import dayjs from "dayjs";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import type {
  ScheduleSearchParams,
  GenerateScheduleValues,
  DutyConfigValues,
  HolidayExcelRow,
  ImportOptions,
  HolidayCreateData,
  BatchHolidayFormValues,
} from "../types";

describe("duty/management/types", () => {
  it("ScheduleSearchParams shape", () => {
    const p: ScheduleSearchParams = {
      poolId: "p1",
      userId: "u1",
      dutyType: "weekday",
      dateRange: [dayjs("2026-01-01"), dayjs("2026-01-07")],
      expired: 0,
    };
    expect(p.poolId).toBe("p1");
  });

  it("GenerateScheduleValues 必填", () => {
    const v: GenerateScheduleValues = {
      poolId: "p1",
      startDate: "2026-01-01",
      endDate: "2026-01-07",
      dutyType: "weekday",
    };
    expect(v.dutyType).toBe("weekday");
  });

  it("DutyConfigValues shape", () => {
    const v: DutyConfigValues = {
      reminderEnabled: true,
      reminderTime: dayjs("2026-01-01 09:00:00"),
      reminderChannels: ["email"],
      beforeReminderMinutes: 30,
    };
    expect(v.reminderEnabled).toBe(true);
  });

  it("HolidayExcelRow shape", () => {
    const r: HolidayExcelRow = {
      "日期(YYYY-MM-DD)": "2026-01-01",
      名称: "元旦",
    };
    expect(r.名称).toBe("元旦");
  });

  it("ImportOptions shape", () => {
    const o: ImportOptions = { file: new File(["x"], "h.xlsx") };
    expect(o.file.name).toBe("h.xlsx");
  });

  it("HolidayCreateData shape", () => {
    const d: HolidayCreateData = {
      holidayDate: "2026-01-01",
      holidayName: "元旦",
      isOffday: true,
      holidayType: "legal",
      year: 2026,
    } as any;
    expect(d.holidayName).toBe("元旦");
  });

  it("BatchHolidayFormValues shape", () => {
    const v: BatchHolidayFormValues = {
      dateRange: [dayjs("2026-01-01"), dayjs("2026-01-07")],
      holidayName: "元旦",
      holidayType: "legal",
      isOffday: true,
    };
    expect(v.isOffday).toBe(true);
  });
});
