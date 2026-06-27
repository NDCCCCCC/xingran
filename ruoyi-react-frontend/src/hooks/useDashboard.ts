/**
 * useDashboard hook — 5 个并行 useQuery 拉 Dashboard 数据 (Phase 42 R1)
 *
 * 返回 5 个 useQuery 结果:
 *   - summary       5 KPI 卡片
 *   - byConflictType 饼图
 *   - bySeverity    柱状图
 *   - healthTrend   趋势图
 *   - topUnresolved Top N 长期未解决(可被 R4 复用)
 *
 * 设计要点:
 *   - staleTime 30s 适配 5min 物化视图刷新周期(D-01)
 *   - refetchOnWindowFocus: false 避免离开再回来重复拉
 *   - 5 个端点并行,各自独立 useQuery(互不阻塞,某端点 5xx 不影响其他)
 *
 * R4 工位详情页将新增 useWorkstationHealth(per CONTEXT.md deferred)。
 */

import {
  useQuery,
  type UseQueryResult,
} from "@tanstack/react-query";
import { reconciliationApi } from "@/lib/assetApi";
import type {
  SummaryResult,
  TrendPoint,
  ExceptionSummary,
  RuleStats,
} from "@/lib/assetApi";
import { queryKeys } from "@/lib/queryKeys";

/** 30s stale 适配 5min 物化视图刷新(D-01) */
const STALE_TIME_MS = 30 * 1000;
/** 5min gc */
const GC_TIME_MS = 5 * 60 * 1000;

export interface UseDashboardReturn {
  summary: UseQueryResult<SummaryResult>;
  byConflictType: UseQueryResult<Record<string, number>>;
  bySeverity: UseQueryResult<Record<string, number>>;
  healthTrend: UseQueryResult<TrendPoint[]>;
  topUnresolved: UseQueryResult<ExceptionSummary[]>;
  /** 任一查询 loading */
  isLoading: boolean;
  /** 任一查询 error */
  isError: boolean;
}

export function useDashboard(windowDays: number = 7): UseDashboardReturn {
  const summary = useQuery({
    queryKey: queryKeys.reconciliation.summary(windowDays),
    queryFn: () => reconciliationApi.summary(windowDays),
    staleTime: STALE_TIME_MS,
    gcTime: GC_TIME_MS,
    refetchOnWindowFocus: false,
  });

  const byConflictType = useQuery({
    queryKey: queryKeys.reconciliation.byConflictType(windowDays),
    queryFn: () => reconciliationApi.byConflictType(windowDays),
    staleTime: STALE_TIME_MS,
    gcTime: GC_TIME_MS,
    refetchOnWindowFocus: false,
  });

  const bySeverity = useQuery({
    queryKey: queryKeys.reconciliation.bySeverity(windowDays),
    queryFn: () => reconciliationApi.bySeverity(windowDays),
    staleTime: STALE_TIME_MS,
    gcTime: GC_TIME_MS,
    refetchOnWindowFocus: false,
  });

  const healthTrend = useQuery({
    queryKey: queryKeys.reconciliation.healthTrend(windowDays),
    queryFn: () => reconciliationApi.healthTrend(windowDays),
    staleTime: STALE_TIME_MS,
    gcTime: GC_TIME_MS,
    refetchOnWindowFocus: false,
  });

  const topUnresolved = useQuery({
    queryKey: queryKeys.reconciliation.topUnresolved(10),
    queryFn: () => reconciliationApi.topUnresolved(10),
    staleTime: STALE_TIME_MS,
    gcTime: GC_TIME_MS,
    refetchOnWindowFocus: false,
  });

  const isLoading =
    summary.isLoading ||
    byConflictType.isLoading ||
    bySeverity.isLoading ||
    healthTrend.isLoading ||
    topUnresolved.isLoading;

  const isError =
    summary.isError ||
    byConflictType.isError ||
    bySeverity.isError ||
    healthTrend.isError ||
    topUnresolved.isError;

  return {
    summary,
    byConflictType,
    bySeverity,
    healthTrend,
    topUnresolved,
    isLoading,
    isError,
  };
}

/**
 * 例外规则命中统计 hook(R3 启用后才有数据)。
 * R1 单独暴露,避免污染 useDashboard 接口。
 */
export function useExceptionRuleStats(): UseQueryResult<RuleStats[]> {
  return useQuery({
    queryKey: queryKeys.reconciliation.ruleStats(),
    queryFn: () => reconciliationApi.exceptionRuleStats(),
    staleTime: STALE_TIME_MS,
    gcTime: GC_TIME_MS,
    refetchOnWindowFocus: false,
  });
}
