/**
 * 部门数据管理 Hook (Phase 37 批 2 方案 B)
 *
 * 历史:
 * - 本文件原通过自 fetch 调用部门树端点并扁平化为 Map。
 * - Phase 37 批 2 的目标是收敛到 canonical hook `useDeptTree`, 但 `pages/operations/floors`
 *   仍依赖本文件(批 5 才迁移 floors), 不能直接删除。
 * - 采用**方案 B**:保留对外 API (`departments / loading / loadDepartments / getOrgName`),
 *   内部改为委托 `useDeptTree` 获取数据(共享 `['dept','tree']` 缓存)。
 *   `loadDepartments` 保留为 no-op(数据已由 React Query 自动获取),供现有调用点使用。
 *
 * 消费方迁移完毕后(floors 批 5 完成移除), 本文件可整体删除。
 */

import { useCallback, useMemo } from "react";
import { useDeptTree } from "@/hooks/useDeptTree";

interface DepartmentOption {
  id: string;
  deptName: string;
  children?: DepartmentOption[];
}

/**
 * 部门数据管理 Hook
 * 处理部门列表的加载和格式转换
 */
export function useDepartmentData() {
  // canonical hook:数据由 React Query 自动获取(5min stale, 共享缓存),无需手动触发
  const { data: rawDept = [], isLoading: loading } = useDeptTree();

  // SimpleDept 形状已与 DepartmentOption 兼容(id/deptName/children),
  // 仅做结构窄化。运行时数据不变。
  const departments = useMemo<DepartmentOption[]>(
    () => rawDept as unknown as DepartmentOption[],
    [rawDept]
  );

  // 兼容旧 API:数据由 useDeptTree 自动获取,这里返回 no-op。
  // 保留导出是为了避免破坏现有调用点(如 floors/index.tsx 的 init effect)。
  const loadDepartments = useCallback(() => {
    /* no-op: useDeptTree 自动获取数据 */
  }, []);

  // 扁平化部门数据用于查找
  const deptMap = useMemo(() => {
    const map = new Map<string, string>();
    const flattenDeptsToMap = (nodes: DepartmentOption[]) => {
      nodes.forEach(node => {
        map.set(node.id, node.deptName);
        if ((node.children?.length ?? 0) > 0) {
          flattenDeptsToMap(node.children!);
        }
      });
    };
    flattenDeptsToMap(departments);
    return map;
  }, [departments]);

  // 获取所属机构名称
  const getOrgName = useCallback((orgId?: string): string => {
    if (!orgId) return "-";
    return deptMap.get(orgId) || "-";
  }, [deptMap]);

  return {
    departments,
    loading,
    loadDepartments,
    getOrgName,
  };
}
