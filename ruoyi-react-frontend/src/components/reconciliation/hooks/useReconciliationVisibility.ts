/**
 * 跨模块权限门:D-A1-03 静默降级
 *
 * 后端在 WorkstationHandler.GetByID + ReconciliationHandler.GetByWorkstation 内
 * 已设置 reconciliationVisible flag(per cross-module-permission.md §2.3)。
 * 前端 hook 同步检查 perm — 当两者不一致时,前端以**后端 visible 字段为准**
 * (defense in depth,useWorkstationHealth 三段 enabled gate 体现)。
 *
 * 修复点(B4):不再读 authStore.perms(authStore 没有该字段),改读 menuStore.permissions
 *   - menuStore.permissions 在 login 时通过 getUserPermissions() API 加载并缓存
 *     (menuStore.ts:114-121),是当前唯一可用的前端 perm 来源
 *   - 替代旧 R3 的 useSyncExternalStore(authStore) 模式(已不可用)
 */
import { useMenuStore } from "@/store/menuStore";

const REQUIRED_PERM = "asset:reconciliation:list";

export function useReconciliationVisibility(): boolean {
  const permissions = useMenuStore((s) => s.permissions);
  if (!permissions || permissions.length === 0) {
    return false;
  }
  return permissions.includes(REQUIRED_PERM);
}
