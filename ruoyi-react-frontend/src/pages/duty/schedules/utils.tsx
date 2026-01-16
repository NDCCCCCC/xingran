/**
 * Duty Schedules Utilities
 * 值班排班页面工具函数
 */

import dayjs, { type Dayjs } from 'dayjs';
import { WEEKDAY_TEXTS } from './constants';

/** 格式化日期时间为 YYYY-MM-DD HH:mm:ss */
export function formatDateTime(dateStr: string): string {
  if (!dateStr) return '-';
  return dayjs(dateStr).format('YYYY-MM-DD HH:mm:ss');
}

/** 格式化日期为 YYYY-MM-DD */
export function formatDate(dateStr: string): string {
  if (!dateStr) return '-';
  return dayjs(dateStr).format('YYYY-MM-DD');
}

/** 获取星期几文本 */
export function getWeekdayText(day: Dayjs): string {
  return WEEKDAY_TEXTS[day.day()];
}

/** 获取星期几短文本 */
export function getWeekdayShortText(day: Dayjs): string {
  return WEEKDAY_SHORT_TEXTS[day.day()];
}

/** 获取周范围显示文本 */
export function getWeekRangeText(weekStart: Dayjs): string {
  const weekEnd = weekStart.endOf('week');
  return `${weekStart.format('YYYY年MM月DD日')} - ${weekEnd.format('YYYY年MM月DD日')}`;
}

/** 获取一周的日期数组 */
export function getWeekDays(weekStart: Dayjs): Dayjs[] {
  const days: Dayjs[] = [];
  for (let i = 0; i < 7; i++) {
    days.push(weekStart.add(i, 'day'));
  }
  return days;
}

/** 格式化排班选项标签 */
export function formatScheduleOptionLabel(schedule: { scheduleDate: string; dutyType: string; user?: { nickName?: string; username?: string } }): string {
  return `${formatDate(schedule.scheduleDate)} - ${schedule.user?.nickName || schedule.user?.username || ''}`;
}

/** 格式化排班选项内容 */
export function formatScheduleOptionContent(schedule: { scheduleDate: string; dutyType: string; user?: { nickName?: string; username?: string } }, getDutyTypeText: (type: string) => string): string {
  return `${formatDate(schedule.scheduleDate)} (${getDutyTypeText(schedule.dutyType)}) - ${schedule.user?.nickName || schedule.user?.username || ''}`;
}
