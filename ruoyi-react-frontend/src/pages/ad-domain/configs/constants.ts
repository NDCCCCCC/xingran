/**
 * 对账 Tab 常量（v1.24 P82 自 ReconciliationTab.tsx 拆出：
 * react-refresh 要求组件文件只导出组件，非组件导出同文件会破坏 Fast Refresh 边界）
 */
import type { DeptOUMismatchType } from "@/lib/adDomainApi";

/** mismatch_type → 中文标签 / antd 预设 Tag color（UI-SPEC Color 契约，禁 hex） */
export const MISMATCH_TYPE_META: Record<DeptOUMismatchType, { label: string; color: string }> = {
  user_ou_mismatch: { label: "用户 OU 与部门映射不一致", color: "orange" },
  dept_soft_deleted: { label: "映射指向已删除部门", color: "red" },
  mapping_disabled: { label: "映射已停用", color: "gold" },
  config_unrouted: { label: "配置无法归属", color: "blue" },
};
