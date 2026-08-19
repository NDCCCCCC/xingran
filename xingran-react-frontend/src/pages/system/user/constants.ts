/**
 * User 常量定义
 */

import { ENABLE_DISABLE_OPTIONS, ENABLE_DISABLE_TAG_CONFIG } from "@/constants/status";

export interface SelectOption {
  label: string;
  value: number;
}

// 性别选项（Phase 69 DICT-03: 降级为静态 fallback——下拉已迁 useDict("sys_user_sex")，
// 字典接口异常/空数据时由消费方回退渲染本数组；导出保留不删除）
export const GENDER_OPTIONS: SelectOption[] = [
  { label: "男", value: 0 },
  { label: "女", value: 1 },
  { label: "保密", value: 2 },
];

// 状态选项（Phase 69 DICT-03: 共享常量别名引用，本地导出名不变）
export const STATUS_OPTIONS: SelectOption[] = ENABLE_DISABLE_OPTIONS;

// 状态标签配置
export const STATUS_TAG_CONFIG: Record<number, { text: string; color: string }> =
  ENABLE_DISABLE_TAG_CONFIG;
