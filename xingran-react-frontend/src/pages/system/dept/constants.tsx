/**
 * Department Constants
 * 部门页面常量定义
 */

import { Tag } from "antd";
import { NORMAL_STOP_OPTIONS } from "@/constants/status";

/** 状态选项（Phase 69 DICT-03: 共享常量别名引用，本地导出名不变） */
// 字面量拷贝（浅拷贝）：react-refresh 对 Identifier 初始化误判为组件导出；无 mutate/引用比较
export const STATUS_OPTIONS = [...NORMAL_STOP_OPTIONS];

/** 外部机构选项 */
export const EXTERNAL_ORG_OPTIONS = [
  { label: "否", value: 0 },
  { label: "是", value: 1 },
] as const;

/**
 * 机构层级值（契约 §2.1 `sys_dept.org_level`）
 * 0=非机构（默认，存量零回归）/ 1=分公司 / 2=中支 / 3=支公司 / 4=网点（预留）
 * 本地声明，避免与 `src/types/system.ts` 的跨成员写冲突。
 */
export type OrgLevelValue = 0 | 1 | 2 | 3 | 4;

/** 机构层级选项（固定枚举 → 业务禁搜） */
export const ORG_LEVEL_OPTIONS = [
  { label: "非机构", value: 0 },
  { label: "分公司", value: 1 },
  { label: "中支", value: 2 },
  { label: "支公司", value: 3 },
  { label: "网点", value: 4 },
] as const;

/** 机构层级标签配色：1=分公司 red / 2=中支 orange / 3=支公司 blue；0 及其它 default */
const ORG_LEVEL_TAG_META: Record<number, { color: string; label: string }> = {
  0: { color: "default", label: "非机构" },
  1: { color: "red", label: "分公司" },
  2: { color: "orange", label: "中支" },
  3: { color: "blue", label: "支公司" },
  4: { color: "default", label: "网点" },
};

/** 渲染状态标签 */
export function renderStatusTag(status: number) {
  return <Tag color={status === 0 ? "success" : "error"}>{status === 0 ? "正常" : "停用"}</Tag>;
}

/** 渲染外部机构标签 */
export function renderExternalOrgTag(isExternalOrg: number) {
  return (
    <Tag color={isExternalOrg === 1 ? "blue" : "default"}>{isExternalOrg === 1 ? "是" : "否"}</Tag>
  );
}

/** 渲染机构层级标签（未设置的存量数据按 0=非机构 兜底） */
export function renderOrgLevelTag(level?: number) {
  const meta = ORG_LEVEL_TAG_META[level ?? 0] ?? ORG_LEVEL_TAG_META[0];
  return <Tag color={meta.color}>{meta.label}</Tag>;
}
