/**
 * useAliasByLocation — 按 location(机构)Id 拉取该机构下的 alias 映射部门列表,
 * 用于工位编辑模态框"所属部门"下拉的 union 注入(D-06/D-07 决策)。
 *
 * 行为契约:
 * - locationId 为空时 `enabled: false`,不发起请求(避免无效网络请求)
 * - 调用 workstationApi.deptOptions(orgId), 返回 DeptOption[]
 * - 前端消费时按 isAlias=true 追加 [映射] 后缀(D-01 决策)
 * - 与 useDeptTree 不耦合 — alias 数据来自后端 union SQL,不需要客户端合并
 *
 * 失效策略:
 * - 写操作(Plan 39-07 Drawer CRUD)成功后,调用方应同时:
 *   1) invalidateQueries(queryKeys.dept.all) → 触发 useDeptTree 重新拉取
 *   2) invalidateQueries(queryKeys.locationAlias.byLocation(orgId)) → 触发本 hook 重新拉取
 * - 或简单地 invalidateQueries(queryKeys.locationAlias.all) 一把全清
 */

import { useQuery, type UseQueryResult } from "@tanstack/react-query";
import { workstationApi, type DeptOption } from "@/lib/opsApi";
import { queryKeys } from "@/lib/queryKeys";

export function useAliasByLocation(locationId: string | undefined | null): UseQueryResult<DeptOption[]> {
  return useQuery({
    queryKey: queryKeys.locationAlias.byLocation(locationId ?? ""),
    queryFn: async () => {
      if (!locationId) return [];
      return await workstationApi.deptOptions(locationId);
    },
    enabled: !!locationId,
    staleTime: 5 * 60 * 1000,   // 5 min,与 useDeptTree 对齐
    gcTime: 30 * 60 * 1000,
    refetchOnWindowFocus: false,
  });
}
