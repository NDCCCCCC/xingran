/**
 * Holiday Constants
 * 节假日页面常量定义
 */

import type { HolidayType } from "./types";
import { Tag } from "antd";

/** 节假日类型选项 */
export const HOLIDAY_TYPE_OPTIONS = [
  { label: "法定节假日", value: "legal" },
  { label: "调休工作日", value: "workday" },
  { label: "自定义", value: "custom" },
] as const;

/** 星期文本映射 */
export const WEEKDAY_TEXTS = ["日", "一", "二", "三", "四", "五", "六"] as const;

/** 获取节假日类型标签 */
export function renderHolidayTypeTag(type: HolidayType) {
  const config: Record<HolidayType, { text: string; color: string }> = {
    legal: { text: "法定节假日", color: "red" },
    workday: { text: "调休工作日", color: "orange" },
    custom: { text: "自定义", color: "blue" },
  };
  const cfg = config[type] || { text: type, color: "default" };
  return <Tag color={cfg.color}>{cfg.text}</Tag>;
}

/** 获取是否休息标签 */
export function renderIsOffdayTag(isOffday: boolean) {
  return <Tag color={isOffday ? "green" : "default"}>{isOffday ? "休息日" : "工作日"}</Tag>;
}
