/**
 * Dict Constants
 * 字典管理页面常量定义
 */

import type { BadgeProps } from "antd";
import { Tag } from "antd";

/** 状态选项 */
export const STATUS_OPTIONS = [
  { label: "启用", value: 0 },
  { label: "禁用", value: 1 },
] as const;

/** 状态配置 */
export const STATUS_CONFIG: Record<number, { text: string; color: BadgeProps["color"] }> = {
  0: { text: "启用", color: "success" },
  1: { text: "禁用", color: "error" },
};

/** 默认表单值 */
export const DEFAULT_TYPE_FORM_VALUES = {
  status: 0,
};

export const DEFAULT_DATA_FORM_VALUES = {
  dictSort: 0,
  status: 0,
  isDefault: false,
};

/** 渲染状态标签 */
export function renderStatusTag(status: number) {
  const config = STATUS_CONFIG[status];
  return <Tag color={config?.color}>{config?.text}</Tag>;
}
