/**
 * useWorkstationHealth — Phase 45 R4 / D-A1-01/03
 *
 * 工位对账健康度聚合 React Query hook。
 * - 命中后端 by-workstation 端点,返回 {workstation, healthScore, assets, visible}
 * - 缓存 5min staleTime / 10min gcTime 与 R1 MV 刷新节流一致 (D-A4-03)
 * - 三段 enabled gate (defense-in-depth,UI-SPEC §"Cross-Module Permission Degradation"):
 *     1) visible=true  (前端 menuStore.permissions 检查)
 *     2) Boolean(workstationId)  (防空 ID 误触发)
 *     3) data?.visible !== false  (后端 visible 字段确认;首次未拉到数据时放行)
 *
 * CLAUDE.md useEffect Dependencies 强约束:deps 仅 primitive workstationId
 */
import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { reconciliationApi, type ByWorkstationResponse } from "@/lib/assetApi";
import { queryKeys } from "@/lib/queryKeys";
import { useReconciliationVisibility } from "./useReconciliationVisibility";

const STALE_TIME_MS = 5 * 60 * 1000;
const GC_TIME_MS = 10 * 60 * 1000;

export function useWorkstationHealth(
  workstationId: string
): UseQueryResult<ByWorkstationResponse> {
  const visible = useReconciliationVisibility();

  return useQuery({
    queryKey: queryKeys.reconciliation.workstationHealth(workstationId),
    queryFn: () => reconciliationApi.byWorkstation({ workstationId, window: "7d" }),
    // 三段 defense-in-depth gate (UI-SPEC §Cross-Module Permission Degradation)
    enabled: visible === true && Boolean(workstationId),
    staleTime: STALE_TIME_MS,
    gcTime: GC_TIME_MS,
    refetchOnWindowFocus: false,
  });
}
