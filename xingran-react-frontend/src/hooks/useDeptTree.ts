/**
 * useDeptTree hook — fetches the department tree via React Query.
 *
 * Backed by the existing `getDeptTree()` helper from `@/lib/dutyApi`, which
 * is the established codebase pattern (8+ consumers). The hook integrates
 * with whatever caching/transformations the helper already does, and shares
 * one cache entry across every consumer (D-13 Step 2).
 *
 * Defaults follow D-11 (5min stale, 30min gc, no refetch-on-focus).
 */

import { useQuery, useQueryClient, type UseQueryResult } from "@tanstack/react-query";
import { post } from "@/lib/api";
import type { SimpleDept } from "@/lib/dutyApi";
import { queryKeys } from "@/lib/queryKeys";
import { useAuthStore } from "@/store/authStore";

/**
 * Department tree node. We keep the helper's `SimpleDept` shape as the
 * canonical type to stay compatible with all existing consumers.
 */
export type DeptTreeNode = SimpleDept;

interface DeptTreeResponse {
  code: number;
  data?: DeptTreeNode[];
}

export function useDeptTree(options?: { scoped?: boolean }): UseQueryResult<DeptTreeNode[]> {
  const scoped = options?.scoped ?? false;
  // v1.54 修复：queryKey 必须带当前登录人 ID。
  // 部门树（尤其 scoped 收口形态）的内容依赖"当前登录者"的数据范围——同一浏览器
  // 里超管退出后换机构管理员登录，若 key 相同会命中 React Query 缓存里的超管全量树
  // （gcTime 30min），导致新账号看到全量部门（用户列表 412 已收口但树仍全量的真因）。
  // 带上 userID 后切账号自动隔离缓存条目，无需用户手动清缓存。
  const userID = useAuthStore((s) => s.user?.id) ?? "anonymous";
  return useQuery({
    // scoped 与全量树**分开缓存**：同一登录人两种结果并存（如超管的两个条目内容
    // 相同，但普通用户的 scoped 条目是收口后的子树）。共享一条缓存会让先加载的
    // 形态污染后加载的消费者。
    queryKey: [...(scoped ? [...queryKeys.dept.tree(), "scoped"] : queryKeys.dept.tree()), userID],
    queryFn: async () => {
      // scoped: true —— 后端按登录人数据范围/机构锚点收口部门树（v1.54）。
      // 语义见后端 internal/services/system/department_service.go filterDeptTreeByScope。
      const body = scoped ? { scoped: true } : {};
      const res = (await post("/system/departments/tree", body)) as DeptTreeResponse;
      return (res.data ?? []) as DeptTreeNode[];
    },
    staleTime: 5 * 60 * 1000,
    gcTime: 30 * 60 * 1000,
    refetchOnWindowFocus: false,
  });
}

/**
 * Returns a function that invalidates every `['dept', ...]` query in the cache.
 *
 * Call this after a successful dept mutation (create/update/delete) so all
 * `useDeptTree()` consumers re-fetch on next access.
 */
export function useInvalidateDept() {
  const qc = useQueryClient();
  return () => qc.invalidateQueries({ queryKey: queryKeys.dept.all });
}
