/**
 * Duty Schedules Utilities
 * 值班排班页面工具函数
 */

import type { Dayjs } from "dayjs";
import { WEEKDAY_TEXTS, WEEKDAY_SHORT_TEXTS } from "./constants";
// 导入时间格式化函数供本模块使用
import { formatDateTime, formatDate } from "@/utils/datetime";

// 重新导出时间格式化函数，供其他模块使用
export { formatDateTime, formatDate };

export function getWeekdayText(day: Dayjs): string {
  return WEEKDAY_TEXTS[day.day()];
}

export function getWeekdayShortText(day: Dayjs): string {
  return WEEKDAY_SHORT_TEXTS[day.day()];
}

export function getWeekRangeText(weekStart: Dayjs): string {
  const weekEnd = weekStart.endOf("week");
  return `${weekStart.format("YYYY年MM月DD日")} - ${weekEnd.format("YYYY年MM月DD日")}`;
}

export function getWeekDays(weekStart: Dayjs): Dayjs[] {
  const days: Dayjs[] = [];
  for (let i = 0; i < 7; i++) {
    days.push(weekStart.add(i, "day"));
  }
  return days;
}

export function formatScheduleOptionLabel(schedule: {
  scheduleDate: string;
  user?: { nickname?: string; username?: string };
}): string {
  return `${formatDate(schedule.scheduleDate)} - ${schedule.user?.nickname || schedule.user?.username || ""}`;
}

export function formatScheduleOptionContent(
  schedule: {
    scheduleDate: string;
    dutyType: string;
    user?: { nickname?: string; username?: string };
  },
  getDutyTypeText: (type: string) => string
): string {
  return `${formatDate(schedule.scheduleDate)} (${getDutyTypeText(schedule.dutyType)}) - ${schedule.user?.nickname || schedule.user?.username || ""}`;
}
