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
import { getDeptTree, type SimpleDept } from "@/lib/dutyApi";
import { queryKeys } from "@/lib/queryKeys";

/**
 * Department tree node. We keep the helper's `SimpleDept` shape as the
 * canonical type to stay compatible with all existing consumers.
 */
export type DeptTreeNode = SimpleDept;

interface DeptTreeResponse {
  code: number;
  data?: DeptTreeNode[];
}

export function useDeptTree(): UseQueryResult<DeptTreeNode[]> {
  return useQuery({
    queryKey: queryKeys.dept.tree(),
    queryFn: async () => {
      // getDeptTree() returns the tree structure directly (established pattern)
      const res = (await getDeptTree()) as DeptTreeResponse;
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