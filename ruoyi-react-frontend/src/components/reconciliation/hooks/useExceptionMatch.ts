/**
 * useExceptionMatch — Phase 44 R3 复用
 *
 * 例外规则命中测试 React Query hook。供 ReconciliationDrawer Tab 3 使用。
 * queryKey 入参对象 useMemo 稳定(CLAUDE.md useEffect 强约束)。
 */
import { useMemo } from "react";
import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { reconciliationApi } from "@/lib/assetApi";
import { queryKeys } from "@/lib/queryKeys";

const STALE_TIME_MS = 5 * 60 * 1000;
const GC_TIME_MS = 10 * 60 * 1000;

export interface ExceptionMatchResult {
  matchedRules: unknown[];
  mergedActions: string[];
  finalSeverity: string;
  isSilence: boolean;
  needsUserDept: boolean;
}

export function useExceptionMatch(
  params: { ip: string; conflictType?: string }
): UseQueryResult<ExceptionMatchResult> {
  // 入参对象 useMemo 稳定
  const stableKey = useMemo(
    () => ({ ip: params.ip, conflictType: params.conflictType ?? "" }),
    [params.ip, params.conflictType]
  );

  return useQuery({
    queryKey: queryKeys.reconciliation.matchTest({
      ip: stableKey.ip,
      userId: undefined,
      deptId: undefined,
    }),
    queryFn: () => reconciliationApi.exceptionRule.test({ ip: params.ip }),
    enabled: Boolean(params.ip),
    staleTime: STALE_TIME_MS,
    gcTime: GC_TIME_MS,
    refetchOnWindowFocus: false,
  });
}
