/**
 * useDict hook — fetches dictionary entries (sys_dict data) via React Query.
 *
 * Multiple components calling `useDict('sys_user_sex')` share one cache entry.
 * After dict mutations, all `['dict', ...]` queries are invalidated globally
 * via `queryClient.invalidateQueries({ queryKey: queryKeys.dict.all })`, so
 * every consumer re-fetches on next mount without manual refresh.
 *
 * Defaults follow D-11 (5min stale, 30min gc, no refetch-on-focus).
 */

import { useQuery, useQueryClient, type UseQueryResult } from "@tanstack/react-query";
import { post } from "@/lib/api";
import { queryKeys } from "@/lib/queryKeys";

export interface DictItem {
  id: string;
  dictLabel: string;
  dictValue: string;
  dictSort?: number;
  cssClass?: string;
  listClass?: string;
  isDefault?: boolean;
  status?: number;
  remark?: string;
  dictType?: string;
  createdAt?: string;
  updatedAt?: string;
}

interface DictListResponse {
  list?: DictItem[];
  total?: number;
}

/**
 * Fetch all dictionary entries for the given dictType.
 *
 * Requests `pageSize: 1000` because dictionary lookups are typically used to
 * populate dropdowns / filter options, and the dataset is small (hundreds
 * of rows at most). If a single dict grows beyond 1000 entries, this should
 * be revisited.
 */
export function useDict(dictType: string): UseQueryResult<DictItem[]> {
  return useQuery({
    queryKey: queryKeys.dict.list(dictType),
    queryFn: async () => {
      const result = await post<DictListResponse>("/system/dicts/data/list", {
        dictType,
        current: 1,
        pageSize: 1000,
      });
      return (result.data?.list ?? []) as DictItem[];
    },
    staleTime: 5 * 60 * 1000,
    gcTime: 30 * 60 * 1000,
    refetchOnWindowFocus: false,
  });
}

/**
 * Returns a function that invalidates every `['dict', ...]` query in the cache.
 *
 * Call this after a successful dict mutation (create/update/delete) so all
 * `useDict(...)` consumers re-fetch on next access.
 */
export function useInvalidateDict() {
  const qc = useQueryClient();
  return () => qc.invalidateQueries({ queryKey: queryKeys.dict.all });
}