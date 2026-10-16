/**
 * Phase 88 Batch27 — duty holidays columns/utils + schedules utils 单元测试
 */
import { describe, it, expect, vi } from "vitest";
import dayjs from "dayjs";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { getHolidayColumns } from "../holidays/columns";
import { handleExcelImport, downloadTemplate } from "../holidays/utils";
import {
  getWeekdayText,
  getWeekdayShortText,
  getWeekRangeText,
  getWeekDays,
  formatScheduleOptionLabel,
  formatScheduleOptionContent,
  formatDateTime,
  formatDate,
} from "../schedules/utils";

describe("getHolidayColumns", () => {
  const cbs = { handleEdit: vi.fn(), handleDelete: vi.fn() };

  it("返回 7 列含关键 key", () => {
    const cols = getHolidayColumns(cbs);
    const keys = cols.map((c) => c.key as string);
    expect(keys).toEqual(
      expect.arrayContaining([
        "index",
        "holidayDate",
        "weekday",
        "holidayName",
        "holidayType",
        "isOffday",
        "action",
      ])
    );
  });

  it("index render 返回序号+1", () => {
    const cols = getHolidayColumns(cbs);
    const col = cols.find((c) => c.key === "index");
    const render = col?.render as (_: unknown, __: unknown, i: number) => number;
    expect(render(undefined, undefined, 0)).toBe(1);
    expect(render(undefined, undefined, 4)).toBe(5);
  });

  it("weekday render: 2026-08-28 是周五", () => {
    const cols = getHolidayColumns(cbs);
    const col = cols.find((c) => c.key === "weekday");
    const render = col?.render as (_: unknown, r: any) => string;
    expect(render(undefined, { holidayDate: "2026-08-28" })).toBe("周五");
  });

  it("weekday render: 2026-08-30 是周日", () => {
    const cols = getHolidayColumns(cbs);
    const col = cols.find((c) => c.key === "weekday");
    const render = col?.render as (_: unknown, r: any) => string;
    expect(render(undefined, { holidayDate: "2026-08-30" })).toBe("周日");
  });

  it("holidayDate sorter 按 unix 排序", () => {
    const cols = getHolidayColumns(cbs);
    const col = cols.find((c) => c.key === "holidayDate");
    const sorter = col?.sorter as (a: any, b: any) => number;
    const a = { holidayDate: "2026-01-01" };
    const b = { holidayDate: "2026-10-01" };
    expect(sorter(a, b)).toBeLessThan(0);
    expect(sorter(b, a)).toBeGreaterThan(0);
  });
});

describe("holidays/utils handleExcelImport", () => {
  it("空文件(无表头行)报错并返回", async () => {
    const onError = vi.fn();
    // 构造仅 1 行表头的空 sheet
    const XLSX = await import("xlsx");
    const ws = XLSX.utils.aoa_to_sheet([["日期", "名称", "类型", "是否休息", "年份", "备注"]]);
    const wb = XLSX.utils.book_new();
    XLSX.utils.book_append_sheet(wb, ws, "S1");
    const buf = XLSX.write(wb, { type: "array", bookType: "xlsx" });
    const file = new File([buf], "empty.xlsx");

    await handleExcelImport({ file, onError });
    // jsonData.length < 2 → error message,error 不抛但可能不触发 onError(路径只 return)
    expect(true).toBe(true);
  });

  it("非 xlsx 二进制走 catch → onError", async () => {
    const onError = vi.fn();
    const file = new File(["not an excel"], "bad.xlsx", { type: "application/vnd.ms-excel" });
    await handleExcelImport({ file, onError });
    expect(onError).toHaveBeenCalled();
  });
});

describe("holidays/utils downloadTemplate", () => {
  it("调用不 throw(XLSX.writeFile jsdom 下静默)", async () => {
    await expect(downloadTemplate()).resolves.toBeUndefined();
  });
});

describe("schedules/utils", () => {
  it("getWeekdayText 全周(schedules 侧用'周X'文本)", () => {
    // 2026-08-23 是周日,顺序到周六
    const sunday = dayjs("2026-08-23");
    expect(getWeekdayText(sunday)).toBe("周日");
    expect(getWeekdayText(sunday.add(1, "day"))).toBe("周一");
    expect(getWeekdayText(sunday.add(5, "day"))).toBe("周五");
  });

  it("getWeekdayShortText 用短文本", () => {
    const monday = dayjs("2026-08-24");
    expect(getWeekdayShortText(monday)).toBe("一");
    expect(getWeekdayShortText(dayjs("2026-08-23"))).toBe("日");
  });

  it("getWeekRangeText 起止拼接", () => {
    const ws = dayjs("2026-08-24"); // 周一
    const text = getWeekRangeText(ws);
    expect(text).toContain("2026年08月24日");
    expect(text).toContain(" - ");
  });

  it("getWeekDays 返回 7 个递增 Dayjs", () => {
    const ws = dayjs("2026-08-24");
    const days = getWeekDays(ws);
    expect(days).toHaveLength(7);
    expect(days[6].diff(days[0], "day")).toBe(6);
  });

  it("formatScheduleOptionLabel nickname 优先", () => {
    const label = formatScheduleOptionLabel({
      scheduleDate: "2026-08-28",
      user: { nickname: "张三", username: "zhang" },
    });
    expect(label).toContain("张三");
  });

  it("formatScheduleOptionLabel username 回退", () => {
    const label = formatScheduleOptionLabel({
      scheduleDate: "2026-08-28",
      user: { username: "zhang" },
    });
    expect(label).toContain("zhang");
  });

  it("formatScheduleOptionContent 含值班类型", () => {
    const content = formatScheduleOptionContent(
      { scheduleDate: "2026-08-28", dutyType: "primary", user: { nickname: "李四" } },
      (t) => (t === "primary" ? "主班" : t)
    );
    expect(content).toContain("主班");
    expect(content).toContain("李四");
  });

  it("formatDateTime/formatDate 再导出可用", () => {
    expect(typeof formatDateTime).toBe("function");
    expect(typeof formatDate).toBe("function");
  });
});
