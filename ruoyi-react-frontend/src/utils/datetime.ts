import dayjs from "dayjs";
import utc from "dayjs/plugin/utc";

dayjs.extend(utc);

/**
 * 附加格式常量（Phase 68 DEBT-02b：分档展示格式统一从本文件取，禁止散落字面量）
 */
export const FORMAT_DATETIME_MINUTE = "YYYY-MM-DD HH:mm";
export const FORMAT_DATE_SHORT = "MM-DD";
export const FORMAT_TIME_MINUTE = "HH:mm";

/**
 * 统一时间格式化函数
 * 默认格式: YYYY-MM-DD HH:mm:ss
 */
export function formatDateTime(
  date: string | Date | null | undefined,
  format: string = "YYYY-MM-DD HH:mm:ss"
): string {
  if (!date) return "-";
  const dateStr = typeof date === "string" ? date.replace(/[Zz]$/, "") : date;
  try {
    return dayjs(dateStr).format(format);
  } catch {
    return String(date);
  }
}

export function formatDate(date: string | Date | null | undefined): string {
  return formatDateTime(date, "YYYY-MM-DD");
}

export function formatTime(date: string | Date | null | undefined): string {
  return formatDateTime(date, "HH:mm:ss");
}
