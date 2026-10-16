/**
 * Dict Constants
 * 字典管理页面常量定义
 */

import type { BadgeProps } from "antd";
import { Tag } from "antd";
import { ENABLE_DISABLE_OPTIONS, ENABLE_DISABLE_TAG_CONFIG } from "@/constants/status";

/** 状态选项（Phase 69 DICT-03: 共享常量别名引用，本地导出名不变） */
export const STATUS_OPTIONS = ENABLE_DISABLE_OPTIONS;

/** 状态配置 */
export const STATUS_CONFIG: Record<number, { text: string; color: BadgeProps["color"] }> =
  ENABLE_DISABLE_TAG_CONFIG;

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
