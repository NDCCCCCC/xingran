/**
 * useExceptionList hook — 异常列表分页查询 (Phase 42 R1)
 *
 * 用 keepPreviousData 保持分页/筛选切换时滚动位置稳定,
 * 配合 useMemo + JSON.stringify 序列化 params 作为 queryKey,
 * 避免父组件传入新对象引用导致缓存击穿(CLAUDE.md useEffect Dependencies 强约束)。
 *
 * 典型用法(配合 useSearchParams 读 URL 初始 filter):
 *
 *   const [params] = useSearchParams();
 *   const listParams = useMemo<ExceptionListParams>(
 *     () => ({
 *       current: 1, pageSize: 20,
 *       conflictType: params.get("type") || undefined,
 *       severity: params.get("severity") || undefined,
 *       ...
 *     }),
 *     [params]
 *   );
 *   const { data, isLoading } = useExceptionList(listParams);
 */

import { useMemo } from "react";
import { useQuery, keepPreviousData, type UseQueryResult } from "@tanstack/react-query";
import { reconciliationApi } from "@/lib/assetApi";
import type { ExceptionListParams, ExceptionListItem, PageResult } from "@/lib/assetApi";
import { queryKeys } from "@/lib/queryKeys";

const STALE_TIME_MS = 30 * 1000;
const GC_TIME_MS = 5 * 60 * 1000;

export interface UseExceptionListReturn {
  data: PageResult<ExceptionListItem> | undefined;
  isLoading: boolean;
  isError: boolean;
  isFetching: boolean;
}

export function useExceptionList(params: ExceptionListParams): UseExceptionListReturn {
  // JSON.stringify 稳定参数对象引用,避免 useQuery 误判 key 变化(CLAUDE.md)
  // 保持稳定的 queryKey 同时也保证 placeholderData 命中同一份缓存
  const stableParams = useMemo(() => params, [JSON.stringify(params)]);

  const query: UseQueryResult<PageResult<ExceptionListItem>> = useQuery({
    queryKey: queryKeys.reconciliation.exceptionList(stableParams),
    queryFn: () => reconciliationApi.exceptionList(stableParams),
    staleTime: STALE_TIME_MS,
    gcTime: GC_TIME_MS,
    refetchOnWindowFocus: false,
    placeholderData: keepPreviousData,
  });

  return {
    data: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    isFetching: query.isFetching,
  };
}
