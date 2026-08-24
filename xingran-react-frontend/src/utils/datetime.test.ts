import { describe, expect, it } from "vitest";
import { formatDate, formatDateTime, formatTime } from "./datetime";

describe("datetime 格式化", () => {
  it("默认格式 YYYY-MM-DD HH:mm:ss（本地无时区漂移）", () => {
    expect(formatDateTime("2026-08-24T10:00:00")).toBe("2026-08-24 10:00:00");
  });

  it("自定义格式", () => {
    expect(formatDateTime("2026-08-24T10:00:00", "YYYY/MM/DD")).toBe("2026/08/24");
    expect(formatDateTime("2026-08-24T10:00:00", "HH 时 mm 分")).toBe("10 时 00 分");
  });

  it("Z 后缀被剥离：UTC 标记不引起时区换算（时区无关确定性）", () => {
    // 剥离 Z 后按本地挂钟时间解析，任何机器时区输出一致
    expect(formatDateTime("2026-08-24T10:00:00Z")).toBe("2026-08-24 10:00:00");
    expect(formatDateTime("2026-08-24T10:00:00z")).toBe("2026-08-24 10:00:00");
  });

  it("Date 对象输入", () => {
    expect(formatDateTime(new Date(2026, 7, 24, 10, 0, 0))).toBe("2026-08-24 10:00:00");
  });

  it("null / undefined / 空字符串返回 '-'", () => {
    expect(formatDateTime(null)).toBe("-");
    expect(formatDateTime(undefined)).toBe("-");
    expect(formatDateTime("")).toBe("-");
    expect(formatDate(null)).toBe("-");
    expect(formatTime(undefined)).toBe("-");
  });

  it("formatDate 只输出日期", () => {
    expect(formatDate("2026-08-24T10:00:00")).toBe("2026-08-24");
  });

  it("formatTime 只输出时间", () => {
    expect(formatTime("2026-08-24T10:00:00")).toBe("10:00:00");
  });

  it("不可解析字符串按 dayjs Invalid Date 语义输出（不抛错）", () => {
    // try/catch 只兜底异常路径；dayjs 对非法输入不 throw，format 输出 'Invalid Date'
    expect(formatDateTime("not-a-date")).toBe("Invalid Date");
  });
});
