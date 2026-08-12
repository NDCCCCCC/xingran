/**
 * useRoleList hook — fetches the role list via React Query.
 *
 * Shared cache entry across every consumer (D-13 Step 2).
 * Requests `pageSize: 1000` because role lists are typically used to populate
 * dropdowns / filter options, and the dataset is small (hundreds of rows).
 *
 * Defaults follow D-11 (5min stale, 30min gc, no refetch-on-focus).
 */

import { useQuery, useQueryClient, type UseQueryResult } from "@tanstack/react-query";
import { post } from "@/lib/api";
import { queryKeys } from "@/lib/queryKeys";

export interface RoleListItem {
  roleId: string;
  roleName: string;
  roleKey: string;
  status: number;
  remark?: string;
  createdAt?: string;
  updatedAt?: string;
}

interface RoleListResponse {
  list?: RoleListItem[];
  total?: number;
}

export function useRoleList(): UseQueryResult<RoleListItem[]> {
  return useQuery({
    queryKey: queryKeys.role.list(),
    queryFn: async () => {
      const result = await post<RoleListResponse>("/system/roles/list", {
        current: 1,
        pageSize: 1000,
      });
      return (result.data?.list ?? []) as RoleListItem[];
    },
    staleTime: 5 * 60 * 1000,
    gcTime: 30 * 60 * 1000,
    refetchOnWindowFocus: false,
  });
}

/**
 * Returns a function that invalidates every `['role', ...]` query in the cache.
 *
 * Call this after a successful role mutation (create/update/delete) so all
 * `useRoleList()` consumers re-fetch on next access.
 */
export function useInvalidateRole() {
  const qc = useQueryClient();
  return () => qc.invalidateQueries({ queryKey: queryKeys.role.all });
}