/**
 * Workstation Data Hook
 * 工位数据管理 Hook
 */

import { useCallback } from 'react';
import type { WorkstationOps } from '@/types';
import { workstationApi, floorApi } from '@/lib/opsApi';
import { post } from '@/lib/api';
import { handleApiError } from '@/utils/errorHandler';
import type { WorkstationStatistics, FloorOption, UserOption, DeptTreeNode } from '../types';

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
      handleApiError(error, '加载统计数据', false);
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
      handleApiError(error, '加载楼层选项', false);
    }
  }, [setFloorOptions]);

  // 递归筛选外部机构部门
  const filterExternalOrgDepts = useCallback((nodes: DepartmentNode[]): DepartmentNode[] => {
    return nodes.reduce<DepartmentNode[]>((acc, node) => {
      const filteredChildren = node.children?.length
        ? filterExternalOrgDepts(node.children)
        : [];

      if (node.isExternalOrg === 1 || filteredChildren.length > 0) {
        acc.push({
          ...node,
          children: filteredChildren.length > 0 ? filteredChildren : undefined,
        });
      }

      return acc;
    }, []);
  }, []);

  // 构建树形结构数据
  const buildTreeData = useCallback((depts: DepartmentNode[]): DeptTreeNode[] => {
    return depts.map((dept) => ({
      title: dept.deptName || dept.name || '',
      value: dept.id,
      key: dept.id,
      children: dept.children?.length ? buildTreeData(dept.children) : undefined,
    }));
  }, []);

  const loadDeptOptions = useCallback(async () => {
    try {
      const result = await post('/system/departments/tree', {}) as { data: DepartmentNode[] };
      const deptList = Array.isArray(result.data) ? result.data : [];
      // 加载所有部门（包括内部部门和外部机构），不进行过滤
      // 工位的 orgId 可能指向任何部门
      setDeptTreeData(buildTreeData(deptList));
    } catch (error) {
      handleApiError(error, '加载部门选项', false);
    }
  }, [setDeptTreeData, buildTreeData]);

  const loadUserOptions = useCallback(async (deptId?: string) => {
    try {
      const params = { current: 1, pageSize: 1000, ...(deptId && { deptId }) };
      const result = await post('/system/users/list', params) as { data?: { list: Record<string, unknown>[] } };
      const users = result.data?.list || [];
      setUserOptions(
        users.map((u: Record<string, unknown>) => ({
          id: String(u.id),
          name: String(u.username),
        }))
      );
    } catch (error) {
      handleApiError(error, '加载用户选项', false);
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
      handleApiError(error, '加载平面图数据', false);
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
