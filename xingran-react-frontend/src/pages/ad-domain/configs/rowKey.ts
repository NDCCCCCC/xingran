/**
 * 对账 Tab rowKey 工具（v1.24 P82 自 ReconciliationTab.tsx 拆出：
 * react-refresh 要求组件文件只导出组件，非组件导出同文件会破坏 Fast Refresh 边界）
 */
import type { DeptOUMismatchRow } from "@/lib/adDomainApi";

/**
 * 组合 rowKey（UI-SPEC 锁定）：部门级与用户级行都唯一，
 * NULL 键统一落空串，避免 undefined 参与拼接。
 */
export function rowKeyOf(row: DeptOUMismatchRow) {
  return `${row.mismatchType}-${row.deptId ?? ""}-${row.userId ?? ""}-${row.adConfigId ?? ""}`;
}
