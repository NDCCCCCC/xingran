/**
 * Workstation Data Hook
 * 工位数据管理 Hook
 */

import { useCallback } from "react";
import type { WorkstationOps } from "@/types";
import { workstationApi, floorApi } from "@/lib/opsApi";
import { post } from "@/lib/api";
import { handleApiError } from "@/utils/errorHandler";
import type { WorkstationStatistics, FloorOption, UserOption, DeptTreeNode } from "../types";

// 后端返回的部门节点类型
interface DepartmentNode {
  id: string;
  deptName?: string;
  name?: string;
  isExternalOrg?: number;
  children?: DepartmentNode[];
}

export interface UseWorkstationDataReturn {
  loadStatistics: (orgId?: string) => Promise<void>;
  loadFloorOptions: (orgId?: string) => Promise<void>;
  loadDeptOptions: () => Promise<void>;
  loadUserOptions: (deptId?: string) => Promise<void>;
  loadFloorPlanWorkstations: (floorCode: string) => Promise<WorkstationOps[]>;
}

export function useWorkstationData(
  setStatistics: React.Dispatch<React.SetStateAction<WorkstationStatistics>>,
  setFloorOptions: React.Dispatch<React.SetStateAction<FloorOption[]>>,
  setDeptTreeData: React.Dispatch<React.SetStateAction<DeptTreeNode[]>>,
  setUserOptions: React.Dispatch<React.SetStateAction<UserOption[]>>
): UseWorkstationDataReturn {
  const loadStatistics = useCallback(async (orgId?: string) => {
    try {
      const baseParams = { current: 1, pageSize: 1, ...(orgId && { orgId }) };

      // 并发请求各状态的计数
      const [totalResult, availableResult, occupiedResult, maintainResult] = await Promise.all([
        workstationApi.list(baseParams),
        workstationApi.list({ ...baseParams, status: 0 }),
        workstationApi.list({ ...baseParams, status: 1 }),
        workstationApi.list({ ...baseParams, status: 2 }),
      ]);

      setStatistics({
        total: totalResult.data?.total || 0,
        available: availableResult.data?.total || 0,
        occupied: occupiedResult.data?.total || 0,
        maintain: maintainResult.data?.total || 0,
      });
    } catch (error) {
      handleApiError(error, "加载统计数据", false);
    }
  }, [setStatistics]);

  const loadFloorOptions = useCallback(async (orgId?: string) => {
    try {
      const params = { current: 1, pageSize: 1000, ...(orgId && { orgId }) };
      const result = await floorApi.list(params);
      const floors = result.data?.list || [];
      setFloorOptions(
        floors.map((f) => ({
          id: f.id,
          code: f.code || String(f.floorNo),
          name: `${f.buildingName || f.buildingCode} - ${f.name || f.floorNo}`,
        }))
      );
    } catch (error) {
      handleApiError(error, "加载楼层选项", false);
    }
  }, [setFloorOptions]);

  // 构建树形结构数据
  // 节点的 title 显示从第二级开始的完整路径（参考 DepartmentTreeSelect 中
  // convertDeptTreeData 的约定），例如 "集团总部/财务中心/财务部"。
  // 这样在编辑模态框的"所属部门"字段中回显时，能看到完整部门链，
  // 而非只显示最末级部门名。
  //
  // 重要:必须透传 isExternalOrg。如果遗漏,filterExternalOrgDepts 拿到的
  // isExternalOrg 是 undefined,会把整棵树过滤成空,导致模态框"所属机构"
  // 下拉无法回显(显示 UUID)。
  const buildTreeData = useCallback((depts: DepartmentNode[]): DeptTreeNode[] => {
    const build = (nodes: DepartmentNode[], ancestorNames: string[]): DeptTreeNode[] => {
      return nodes.map((node) => {
        const name = node.deptName || node.name || "";
        // 顶级部门直接显示名称；从第二级开始拼接完整路径
        const title = ancestorNames.length === 0
          ? name
          : [...ancestorNames, name].join(" / ");
        return {
          title,
          value: node.id,
          key: node.id,
          // 透传:filterExternalOrgDepts 依赖此字段判断节点是否保留
          isExternalOrg: node.isExternalOrg,
          children: node.children?.length ? build(node.children, [...ancestorNames, name]) : undefined,
        };
      });
    };
    return build(depts, []);
  }, []);

  const loadDeptOptions = useCallback(async () => {
    try {
      const result = await post("/system/departments/tree", {}) as { data: DepartmentNode[] };
      const deptList = Array.isArray(result.data) ? result.data : [];
      // 加载所有部门（包括内部部门和外部机构），不进行过滤
      // 工位的 orgId 可能指向任何部门；外部机构过滤在页面 useMemo 派生 orgTreeData。
      setDeptTreeData(buildTreeData(deptList));
    } catch (error) {
      handleApiError(error, "加载部门选项", false);
    }
  }, [setDeptTreeData, buildTreeData]);

  const loadUserOptions = useCallback(async (deptId?: string) => {
    try {
      // 使用 recursiveDeptId 而非 deptId,后端会扩展为该部门+所有子部门的用户。
      // 原 deptId 单值语义在工位模态框里不够用——用户通常需要看到整个部门树下的人。
      const params = {
        current: 1,
        pageSize: 1000,
        ...(deptId && { recursiveDeptId: deptId }),
      };
      const result = await post("/system/users/list", params) as { data?: { list: Record<string, unknown>[] } };
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
  }, [setUserOptions]);

  const loadFloorPlanWorkstations = useCallback(async (floorCode: string): Promise<WorkstationOps[]> => {
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
  }, []);

  return {
    loadStatistics,
    loadFloorOptions,
    loadDeptOptions,
    loadUserOptions,
    loadFloorPlanWorkstations,
  };
}
