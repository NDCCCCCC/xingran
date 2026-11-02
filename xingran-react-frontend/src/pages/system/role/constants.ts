/**
 * Role Constants
 * 角色管理常量定义
 */

import { NORMAL_STOP_OPTIONS } from "@/constants/status";

/** 数据范围选项 */
export const DATA_SCOPE_OPTIONS = [
  { label: "全部数据", value: 1 },
  { label: "自定义数据", value: 2 },
  { label: "本部门数据", value: 3 },
  { label: "本部门及子部门数据", value: 4 },
  { label: "仅本人数据", value: 5 },
] as const;

/** 状态选项（Phase 69 DICT-03: 共享常量别名引用，本地导出名不变） */
export const STATUS_OPTIONS = NORMAL_STOP_OPTIONS;

/** 默认表单值 */
export const DEFAULT_FORM_VALUES = {
  dataScope: 1,
  status: 0,
  roleSort: 0,
  menuCheckStrictly: true,
  deptCheckStrictly: true,
};
