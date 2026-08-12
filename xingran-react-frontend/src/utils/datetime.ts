import dayjs from "dayjs";
import utc from "dayjs/plugin/utc";

dayjs.extend(utc);

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
