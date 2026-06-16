/**
 * Role Constants
 * 角色管理常量定义
 */

/** 数据范围选项 */
export const DATA_SCOPE_OPTIONS = [
  { label: "全部数据", value: 1 },
  { label: "自定义数据", value: 2 },
  { label: "本部门数据", value: 3 },
  { label: "本部门及子部门数据", value: 4 },
  { label: "仅本人数据", value: 5 },
] as const;

/** 状态选项 */
export const STATUS_OPTIONS = [
  { label: "正常", value: 0 },
  { label: "停用", value: 1 },
] as const;

/** 默认表单值 */
export const DEFAULT_FORM_VALUES = {
  dataScope: 1,
  status: 0,
  roleSort: 0,
  menuCheckStrictly: true,
  deptCheckStrictly: true,
};
