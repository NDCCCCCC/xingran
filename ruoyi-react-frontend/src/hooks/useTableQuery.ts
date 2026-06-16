/**
 * List page data fetching via React Query (companion to useTableManager).
 *
 * Why split?
 * - `useTableManager` owns modal/form/selection state and stays a plain
 *   React hook — no React Query coupling.
 * - `useTableQuery` owns the *data-fetching half* of a list page. It gives
 *   the page a cached, refetch-aware, deduplicated list query without
 *   forcing a full rewrite of `useTableManager`-based pages.
 *
 * Migration example (incremental, no breaking change):
 *
 *   // before
 *   const { data, total, loading, loadData } = useTableManager(
 *     (params) => workstationApi.list(params),
 *   );
 *
 *   // after
 *   const { data, total, isLoading, isPlaceholderData } = useTableQuery<Workstation>({
 *     resource: 'workstations',
 *     current,
 *     pageSize,
 *     filters,
 *     queryFn: workstationApi.list,
 *   });
 *   // keep useTableManager for edit modal / form state if needed
 *
 * Why `placeholderData: keepPreviousData`?
 * - During page transitions the previous page's data stays rendered, so
 *   the table doesn't flash to a blank/loading state (D-12).
 */

import {
  useQuery,
  keepPreviousData,
  type UseQueryResult,
} from "@tanstack/react-query";
import { queryKeys } from "@/lib/queryKeys";

export interface UseTableQueryParams {
  resource: string;
  current: number;
  pageSize: number;
  filters?: Record<string, unknown>;
}

export interface PageData<T> {
  list: T[];
  total: number;
  current?: number;
  pageSize?: number;
}

export function useTableQuery<T>({
  resource,
  current,
  pageSize,
  filters = {},
  queryFn,
}: UseTableQueryParams & {
  queryFn: (params: Record<string, unknown>) => Promise<PageData<T>>;
}): UseQueryResult<PageData<T>> {
  return useQuery({
    queryKey: queryKeys.list.page(resource, { current, pageSize, ...filters }),
    queryFn: () => queryFn({ current, pageSize, ...filters }),
    placeholderData: keepPreviousData, // D-12: avoid blank flash on paginate
    staleTime: 30 * 1000, // 30s list freshness
  });
}