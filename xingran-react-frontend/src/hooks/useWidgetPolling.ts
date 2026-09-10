/**
 * useWidgetPolling - Widget 数据轮询 Hook
 *
 * Sprint 1 Fix #5 (Vercel audit): migrated from manual setInterval + useState
 * to React Query useQuery with refetchInterval + background tab detection via
 * visibilitychange. Enables automatic request deduplication — multiple instances
 * with the same widgetIds+interval share one network request.
 *
 * Original design preserved:
 * - Tab visibility pauses / resumes polling
 * - Dashboard store write-through for store subscribers
 * - Manual refresh / pause / resume controls
 */

import { useState, useEffect, useCallback, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { useDashboardStore } from "@/store/dashboardStore";
import { dashboardService } from "@/services/dashboardService";
import { queryKeys } from "@/lib/queryKeys";

export interface UseWidgetPollingOptions {
  /** 需要轮询的 Widget ID 列表 */
  widgetIds: string[];
  /** 刷新间隔（秒） */
  interval: number;
  /** 是否启用轮询 */
  enabled?: boolean;
  /** 最小缓存时间（秒），默认 30 秒 */
  minCacheTime?: number;
}

export interface UseWidgetPollingReturn {
  /** 是否正在加载 */
  loading: boolean;
  /** 最后刷新时间 */
  lastRefreshTime: Date | null;
  /** 手动刷新 */
  refresh: () => void;
  /** 暂停轮询 */
  pause: () => void;
  /** 恢复轮询 */
  resume: () => void;
  /** 是否已暂停 */
  isPaused: boolean;
}

/**
 * Widget 数据轮询 Hook
 */
export function useWidgetPolling(options: UseWidgetPollingOptions): UseWidgetPollingReturn {
  const { widgetIds, interval, enabled = true, minCacheTime = 30 } = options;

  const { cacheWidgetData } = useDashboardStore();

  // Paused state — managed internally, not via React Query enabled flag (we want
  // to keep the query alive so cached data is still available when paused).
  const [isPaused, setIsPaused] = useState(false);
  const [lastRefreshTime, setLastRefreshTime] = useState<Date | null>(null);

  // Track tab visibility; polling pauses when tab is hidden.
  const [isTabVisible, setIsTabVisible] = useState(
    typeof document !== "undefined" ? !document.hidden : true
  );

  useEffect(() => {
    const handleVisibilityChange = () => setIsTabVisible(!document.hidden);
    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => document.removeEventListener("visibilitychange", handleVisibilityChange);
  }, []);

  // Batch fetch function — writes through to dashboardStore for L1 cache.
  const queryFn = useCallback(async () => {
    if (widgetIds.length === 0) return;

    const data = await dashboardService.getBatchWidgetData(widgetIds);
    for (const [id, widgetData] of data) {
      cacheWidgetData(id, widgetData);
    }
    setLastRefreshTime(new Date());
    return data;
  }, [widgetIds, cacheWidgetData]);

  // Memoize queryKey so it is stable across renders with same widgetIds content.
  const queryKey = useMemo(
    () => queryKeys.widget.polling(widgetIds, interval),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [JSON.stringify(widgetIds), interval]
  );

  // Only refetch when tab is visible AND polling is not paused.
  const effectiveEnabled = enabled && !isPaused && isTabVisible && widgetIds.length > 0;

  const query = useQuery({
    queryKey,
    queryFn,
    enabled: effectiveEnabled,
    refetchInterval: Math.max(interval, 30) * 1000,
    staleTime: Math.max(interval / 2, minCacheTime) * 1000,
    refetchOnWindowFocus: false,
  });

  const refresh = useCallback(() => {
    void query.refetch();
  }, [query]);

  const pause = useCallback(() => setIsPaused(true), []);
  const resume = useCallback(() => setIsPaused(false), []);

  return {
    loading: query.isLoading,
    lastRefreshTime,
    refresh,
    pause,
    resume,
    isPaused,
  };
}

export default useWidgetPolling;
