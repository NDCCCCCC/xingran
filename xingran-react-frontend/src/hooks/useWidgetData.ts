/**
 * useWidgetData - Widget数据Hook
 *
 * 获取和管理Widget数据
 *
 * Sprint 1 Fix #5 (Vercel audit): migrated from manual setInterval + useState
 * to React Query useQuery with refetchInterval. Enables automatic request
 * deduplication across component instances — multiple widgets with the same
 * queryKey share one network request.
 *
 * The dashboardStore L1 cache is kept as a write-through so any code that
 * reads directly from the store keeps working.
 */

import { useCallback } from "react";
import { useQuery } from "@tanstack/react-query";
import { useDashboardStore } from "@/store/dashboardStore";
import { dataFetcher } from "@/components/dashboard/utils/dataFetcher";
import { queryKeys } from "@/lib/queryKeys";
import type { WidgetConfig } from "@/types/dashboard";

interface UseWidgetDataOptions {
  /** 是否禁用自动刷新 */
  disabled?: boolean;

  /** 刷新间隔（秒），默认使用Widget配置 */
  refreshInterval?: number;
}

interface UseWidgetDataResult<T = unknown> {
  /** 数据 */
  data: T | null;

  /** 加载状态 */
  loading: boolean;

  /** 错误信息 */
  error: string | null;

  /** 刷新数据 */
  refresh: () => void;

  /** 是否正在刷新 */
  isRefreshing: boolean;
}

/** Data fetcher wrapped so it matches the useQuery<T> contract. */
async function fetchWidgetData<T>(
  widget: WidgetConfig,
  _getCachedWidgetData: (id: string) => unknown | null,
  cacheWidgetData: (id: string, data: unknown) => void
): Promise<T | null> {
  if (!widget.enabled) return null;

  const result = await dataFetcher.fetch<T>(widget.dataSource);
  if (result.error) throw new Error(result.error);

  // Write-through L1 cache so store subscribers keep working
  cacheWidgetData(widget.id, result.data);
  return result.data;
}

/**
 * Widget数据Hook
 */
export function useWidgetData<T = unknown>(
  widget: WidgetConfig,
  options?: UseWidgetDataOptions
): UseWidgetDataResult<T> {
  const { getCachedWidgetData, cacheWidgetData } = useDashboardStore();

  const refreshInterval = options?.refreshInterval ?? widget.refreshInterval ?? 60;
  const disabled = options?.disabled ?? false;

  // Stable fetch function — reads current widget from the dataSource param so
  // callers can vary widget content without breaking queryKey identity. The
  // queryKey includes widget.id which is what React Query uses for deduplication.
  const queryFn = useCallback(
    () => fetchWidgetData<T>(widget, getCachedWidgetData, cacheWidgetData),
    // widget.id is embedded in the queryKey so this dependency is stable enough;
    // widget.dataSource changes only when the widget type changes (rare, intentional).
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [widget.id, widget.enabled, widget.dataSource, getCachedWidgetData, cacheWidgetData]
  );

  const queryKey = queryKeys.widget.data(widget.id, widget.dataSource);

  const query = useQuery({
    queryKey,
    queryFn,
    enabled: !disabled && widget.enabled,
    refetchInterval: disabled ? false : refreshInterval * 1000,
    // staleTime just under refetchInterval so background refresh feels instant
    staleTime: (refreshInterval - 5) * 1000,
    refetchOnWindowFocus: false,
  });

  // Map React Query shape to the legacy return shape
  return {
    data: (query.data as T | null) ?? null,
    loading: query.isLoading,
    error: query.error?.message ?? null,
    refresh: query.refetch,
    isRefreshing: query.isFetching && !query.isLoading,
  };
}

/**
 * 批量获取多个Widget数据的Hook
 */
export function useBatchWidgetData(
  widgets: WidgetConfig[],
  options?: UseWidgetDataOptions
): Record<string, unknown> {
  const { cacheWidgetData } = useDashboardStore();

  // Compute these outside useQuery so the hook is always called (hooks rules).
  // When disabled or empty, we pass enabled:false so the query never fires.
  const enabledWidgets = widgets.filter((w) => w.enabled);
  const isDisabled = options?.disabled ?? false;
  const effectiveWidgets = isDisabled ? [] : enabledWidgets;

  const minInterval =
    effectiveWidgets.reduce((min, w) => Math.min(min, w.refreshInterval ?? 60), 60) * 1000;

  const queryKey = queryKeys.widget.polling(
    effectiveWidgets.map((w) => w.id),
    minInterval
  );

  // Always call useQuery (hooks rules) — enabled:false when no widgets to fetch.
  const query = useQuery({
    queryKey,
    queryFn: async () => {
      // Empty batch guard — check again inside queryFn (parallel-safe).
      if (effectiveWidgets.length === 0) return {};
      const results: Record<string, unknown> = {};
      await Promise.all(
        effectiveWidgets.map(async (widget) => {
          try {
            const result = await dataFetcher.fetch(widget.dataSource);
            if (!result.error) {
              cacheWidgetData(widget.id, result.data);
              results[widget.id] = result.data;
            }
          } catch {
            // swallow — widget-level error doesn't break the batch
          }
        })
      );
      return results;
    },
    enabled: effectiveWidgets.length > 0,
    refetchInterval: minInterval > 0 ? minInterval : false,
    staleTime: minInterval - 5000,
    refetchOnWindowFocus: false,
  });

  return { dataMap: (query.data as Record<string, unknown>) ?? {}, loading: query.isLoading };
}
