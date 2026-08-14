/**
 * Workstation Data Hook
 * 工位数据管理 Hook
 *
 * Phase 37 批 2 迁移说明:
 * - 删除本地全路径转换函数与 loadDeptOptions 内的自 fetch post 调用。
 * - 部门数据通过 canonical hook `useDeptTree()` 获取(共享 `['dept','tree']` React Query 缓存)。
 * - 双向语义(D-LOCKED, 高风险)严格保留:
 *   - `deptTreeData` = 全路径(`toFullPathTree(rawDept)`)
 *   - 页面 `orgTreeData` = `trimTitleToLastSegment(filterExternalOrgDepts(deptTreeData))` 反向裁剪为短名
 * - `toFullPathTree` 透传 `isExternalOrg`(否则 `filterExternalOrgDepts` 会把整棵树过滤为空)。
 *   `SimpleDept` 类型签名不含 `isExternalOrg`,但运行时数据含此字段(后端 `sys_dept.is_external_org`),
 *   因此定义本地 `RawDept` 类型窄化(参考 DeptTree/index.tsx 的做法)。
 */

import { useCallback, useEffect } from "react";
import type { WorkstationOps } from "@/types";
import { workstationApi, floorApi } from "@/lib/opsApi";
import { post } from "@/lib/api";
import { handleApiError } from "@/utils/errorHandler";
import { useDeptTree } from "@/hooks/useDeptTree";
import type { SimpleDept } from "@/lib/dutyApi";
import { toFullPathTree } from "@/utils/deptUtils";
import type { WorkstationStatistics, FloorOption, UserOption, DeptTreeNode } from "../types";

// 本地类型:SimpleDept 在类型层面不含 isExternalOrg,但运行时部门树节点实际带此字段
// (sys_dept.is_external_org,见 @/types/system.ts 的 Department 超集)。
// toFullPathTree 依赖该字段透传给 filterExternalOrgDepts,否则 orgTreeData 会被过滤为空。
interface RawDept extends SimpleDept {
  isExternalOrg?: number;
  children?: RawDept[];
}

export interface UseWorkstationDataReturn {
  loadStatistics: (orgId?: string) => Promise<void>;
  loadFloorOptions: (orgId?: string, keyword?: string) => Promise<void>;
  loadDeptOptions: () => void;
  loadUserOptions: (deptId?: string) => Promise<void>;
  loadFloorPlanWorkstations: (floorCode: string) => Promise<WorkstationOps[]>;
  // 注入式兜底(2026-06-30):编辑回填时若 userId 不在 pageSize:50 列表,
  // 调此方法基于 record.userName 注入一条临时 Option,避免 Select 显示 raw UUID。
  ensureUser: (user: { id: string; username?: string; nickname?: string }) => void;
}

export function useWorkstationData(
  setStatistics: React.Dispatch<React.SetStateAction<WorkstationStatistics>>,
  setFloorOptions: React.Dispatch<React.SetStateAction<FloorOption[]>>,
  setDeptTreeData: React.Dispatch<React.SetStateAction<DeptTreeNode[]>>,
  setUserOptions: React.Dispatch<React.SetStateAction<UserOption[]>>
): UseWorkstationDataReturn {
  // canonical 部门树 hook:共享 ['dept','tree'] 缓存(5min stale, 30min gc, refetchOnWindowFocus:false)
  const { data: rawDept = [] } = useDeptTree();

  const loadStatistics = useCallback(
    async (orgId?: string) => {
      try {
        // 专用统计端点(1 次请求替代此前 4 次 list 拼 total),支持 orgId 部门筛选
        const params: Record<string, unknown> = { ...(orgId ? { orgId } : {}) };
        const result = await workstationApi.statistics(params);
        setStatistics({
          total: result.total || 0,
          available: result.available || 0,
          occupied: result.occupied || 0,
          maintain: result.maintain || 0,
        });
      } catch (error) {
        handleApiError(error, "加载统计数据", false);
      }
    },
    [setStatistics]
  );

  const loadFloorOptions = useCallback(
    async (orgId?: string, keyword = "") => {
      try {
        // 使用专用 searchOptions 端点(LIMIT 50),替代 pageSize:1000 + filterOption 反模式
        const opts = await floorApi.searchOptions({
          ...(orgId ? { orgId } : {}),
          ...(keyword ? { name: keyword } : {}),
        });
        // 注:searchOptions 只返回 id+name,丢失了 buildingName/buildingCode 等上下文。
        // 如需显示 "楼宇 - 楼层" 复合 label,需用 workstationApi.list({pageSize:50, orgId})
        // 或后端扩展 dropdown DTO。这里保持 label=楼层名,够 filter 面板辨识。
        setFloorOptions(
          opts.map((o) => ({
            id: o.value,
            code: o.value, // dropdown-options 不返回 floorNo,前端用 id 兜底
            name: o.label,
          }))
        );
      } catch (error) {
        handleApiError(error, "加载楼层选项", false);
      }
    },
    [setFloorOptions]
  );

  // 加载部门选项:
  // - 数据源由 useDeptTree 提供(本组件顶层调用)。
  // - 全路径版本(toFullPathTree, startFromLevel=1 默认):顶级节点 title 直接显示其名,
  //   深级 title 拼接为 "祖先 / ... / 自身"。页面 orgTreeData useMemo 会在"所属机构"
  //   下拉中经 trimTitleToLastSegment 反向裁剪为短名——双向语义,不可破坏。
  // - isExternalOrg 透传依赖:toFullPathTree 输出节点的 isExternalOrg 取自输入节点,
  //   filterExternalOrgDepts 仅保留 isExternalOrg===1 的节点及后代。rawDept 运行时
  //   含此字段(见 RawDept 注释),所以 as unknown 窄化是安全的。
  //
  // Phase 37 code review WR-1 修复:
  // loadDeptOptions 保留为 init effect 占位(index.tsx Promise.all 调用它),但改为稳定引用
  // (依赖 [])。deptTreeData 实际由下方 useEffect 基于 rawDept 派生。这样 index.tsx 的 init
  // effect 不会因 rawDept 变化(loadDeptOptions 引用变化)而重复触发 statistics/floors/users
  // 请求——原实现依赖 [rawDept] 导致页面挂载时多一轮网络请求(37-03 引入的回归)。
  const loadDeptOptions = useCallback(() => {
    // no-op: deptTreeData 由下方 useEffect 监听 rawDept 自动派生
  }, []);

  // rawDept 到达/变化时派生 deptTreeData(全路径版本,透传 isExternalOrg)
  useEffect(() => {
    const fullPath = toFullPathTree(rawDept as unknown as RawDept[]) as unknown as DeptTreeNode[];
    setDeptTreeData(fullPath);
  }, [rawDept, setDeptTreeData]);

  const loadUserOptions = useCallback(
    async (deptId?: string) => {
      try {
        // 使用 recursiveDeptId 而非 deptId,后端会扩展为该部门+所有子部门的用户。
        // 原 deptId 单值语义在工位模态框里不够用——用户通常需要看到整个部门树下的人。
        // Phase 46: pageSize 1000 → 50,虽然 system MaxPageSize=100 仍允许 100,
        // 但前端 Select 仅渲染 ≤50 行,vDOM 节点数可控。
        const params = {
          current: 1,
          pageSize: 50,
          ...(deptId && { recursiveDeptId: deptId }),
        };
        const result = (await post("/system/users/list", params)) as {
          data?: { list: Record<string, unknown>[] };
        };
        const users = result.data?.list || [];
        setUserOptions(
          users.map((u: Record<string, unknown>) => ({
            id: String(u.id),
            username: String(u.username),
            nickname: u.nickname ? String(u.nickname) : undefined,
          }))
        );
      } catch (error) {
        handleApiError(error, "加载用户选项", false);
      }
    },
    [setUserOptions]
  );

  const loadFloorPlanWorkstations = useCallback(
    async (floorCode: string): Promise<WorkstationOps[]> => {
      if (!floorCode) {
        return [];
      }
      try {
        const result = await workstationApi.list({ floorCode, current: 1, pageSize: 1000 });
        return result.data?.list || [];
      } catch (error) {
        handleApiError(error, "加载平面图数据", false);
        return [];
      }
    },
    []
  );

  // 注入式兜底(2026-06-30):同 info-points 模式,若用户 id 已在列表中则保持原引用。
  // setUserOptions 列入 deps 以满足 React Compiler 推断(setState 引用稳定,不影响实际行为)。
  const ensureUser = useCallback(
    (user: { id: string; username?: string; nickname?: string }) => {
      setUserOptions((prev) =>
        prev.find((u) => u.id === user.id)
          ? prev
          : [
              ...prev,
              {
                id: user.id,
                username: user.username || "未命名用户",
                nickname: user.nickname,
              },
            ]
      );
    },
    [setUserOptions]
  );

  return {
    loadStatistics,
    loadFloorOptions,
    loadDeptOptions,
    loadUserOptions,
    loadFloorPlanWorkstations,
    ensureUser,
  };
}
