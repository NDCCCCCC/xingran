/**
 * Duty Schedules Constants
 * 值班排班页面常量定义
 */

import type { BadgeProps } from "antd";

/** 值班类型选项 */
export const DUTY_TYPE_OPTIONS = [
  { label: "工作日", value: "weekday" },
  { label: "周末", value: "weekend" },
  { label: "节假日", value: "holiday" },
] as const;

/** 值班类型 */
export type DutyType = "weekday" | "weekend" | "holiday";

/** 星期文本映射 */
export const WEEKDAY_TEXTS = ["周日", "周一", "周二", "周三", "周四", "周五", "周六"] as const;
export const WEEKDAY_SHORT_TEXTS = ["日", "一", "二", "三", "四", "五", "六"] as const;

/** 值班状态配置 */
export const DUTY_STATUS_CONFIG: Record<number, { text: string; color: string }> = {
  0: { text: "正常", color: "green" },
  1: { text: "已调换", color: "orange" },
  2: { text: "已取消", color: "red" },
};

/** 获取值班类型颜色 */
export function getDutyTypeColor(type: DutyType | string): BadgeProps["color"] {
  switch (type) {
    case "weekday":
      return "blue";
    case "weekend":
      return "orange";
    case "holiday":
      return "red";
    default:
      return "default";
  }
}

/** 获取值班类型显示文本 */
export function getDutyTypeText(type: DutyType | string): string {
  const option = DUTY_TYPE_OPTIONS.find((opt) => opt.value === type);
  return option?.label || type;
}
