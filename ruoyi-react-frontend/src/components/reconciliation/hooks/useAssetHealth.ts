/**
 * useAssetHealth — Phase 45 R4
 *
 * 资产对账健康度切片:从 useWorkstationHealth 缓存取单资产数据(避免 N+1,SC7)。
 *
 * 用法:ReconciliationDrawer Tab 1 用 useAssetHealth 拿 selectedAssetId 详情;
 *       父级 lift-up 已调过 useWorkstationHealth(workstationId),这里只走 cache select。
 */
import type { AssetHealthItem } from "@/lib/assetApi";
import { useWorkstationHealth } from "./useWorkstationHealth";

export function useAssetHealth(
  assetId: string | null,
  workstationId: string
): AssetHealthItem | undefined {
  const { data } = useWorkstationHealth(workstationId);
  if (!data?.assets || !assetId) return undefined;
  return data.assets.find((a) => a.assetId === assetId);
}
