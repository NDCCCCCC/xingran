/**
 * User 常量定义
 */

export interface SelectOption {
  label: string;
  value: number;
}

// 性别选项
export const GENDER_OPTIONS: SelectOption[] = [
  { label: "男", value: 0 },
  { label: "女", value: 1 },
  { label: "保密", value: 2 },
];

// 状态选项
export const STATUS_OPTIONS: SelectOption[] = [
  { label: "启用", value: 0 },
  { label: "禁用", value: 1 },
];

// 状态标签配置
export const STATUS_TAG_CONFIG: Record<number, { text: string; color: string }> = {
  0: { text: "启用", color: "success" },
  1: { text: "禁用", color: "error" },
};
