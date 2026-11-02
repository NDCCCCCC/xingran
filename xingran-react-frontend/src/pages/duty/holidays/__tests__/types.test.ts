/**
 * Phase 88 Batch276 — pages/duty/holidays/types 测试
 */
import { describe, it, expect } from "vitest";
import dayjs from "dayjs";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import type {
  HolidayType,
  BatchHolidayRow,
  ModalState,
  BatchState,
  ExcelImportOptions,
  ExcelHolidayRow,
} from "../types";

describe("duty/holidays/types", () => {
  it("HolidayType 3 类别", () => {
    const t: HolidayType[] = ["legal", "workday", "custom"];
    expect(t.length).toBe(3);
  });

  it("BatchHolidayRow shape", () => {
    const r: BatchHolidayRow = {
      holidayDate: dayjs("2026-01-01"),
      holidayName: "元旦",
      isOffday: true,
      holidayType: "legal",
      year: 2026,
    };
    expect(r.year).toBe(2026);
  });

  it("ModalState shape", () => {
    const s: ModalState = {
      modalVisible: true,
      batchModalVisible: false,
      editingRecord: null,
    };
    expect(s.modalVisible).toBe(true);
  });

  it("BatchState shape", () => {
    const s: BatchState = { batchHolidays: [] };
    expect(s.batchHolidays).toEqual([]);
  });

  it("ExcelImportOptions shape", () => {
    const file = new File(["x"], "h.xlsx");
    const o: ExcelImportOptions = { file };
    expect(o.file.name).toBe("h.xlsx");
  });

  it("ExcelHolidayRow shape", () => {
    const r: ExcelHolidayRow = {
      holidayDate: "2026-01-01",
      holidayName: "元旦",
      isOffday: true,
      holidayType: "legal",
      year: 2026,
    };
    expect(r.holidayName).toBe("元旦");
  });
});
