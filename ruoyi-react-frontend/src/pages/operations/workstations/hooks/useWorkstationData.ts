/**
 * Workstation Data Hook
 * 工位数据管理 Hook
 */

import { useCallback } from 'react';
import type { WorkstationOps, Floor } from '@/types';
import { workstationApi, floorApi } from '@/lib/opsApi';
import { post } from '@/lib/api';
import { handleApiError } from '@/utils/errorHandler';
import type { WorkstationStatistics, FloorOption, UserOption, DeptTreeNode } from '../types';

// 后端返回的部门节点类型
interface DepartmentNode {
  id: string;
  deptName?: string;
  name?: string;
  children?: DepartmentNode[];
}

export interface UseWorkstationDataReturn {
  loadWorkstations: (params?: Record<string, unknown>) => Promise<void>;
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
      const baseParams: { current: number; pageSize: number; orgId?: string } = { current: 1, pageSize: 1 };
      if (orgId) {
        baseParams.orgId = orgId;
      }

      // 并发请求：总数（不带 status 筛选）和各状态的计数
      const [totalResult, availableResult, occupiedResult, maintainResult] = await Promise.all([
        workstationApi.list(baseParams),                        // 获取总数
        workstationApi.list({ ...baseParams, status: 0 }),      // 空闲
        workstationApi.list({ ...baseParams, status: 1 }),      // 占用
        workstationApi.list({ ...baseParams, status: 2 }),      // 维护
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
      const params: { current: number; pageSize: number; orgId?: string } = { current: 1, pageSize: 1000 };
      if (orgId) {
        params.orgId = orgId;
      }
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

  const loadDeptOptions = useCallback(async () => {
    try {
      // 后端返回的是树形结构数组，不是分页格式
      const result = await post('/system/departments/tree', {}) as { data: DepartmentNode[] };
      // 构建树形结构数据，使用 fieldNames 指定字段映射
      const buildTreeData = (depts: DepartmentNode[]): DeptTreeNode[] => {
        return depts.map((dept) => ({
          title: dept.deptName || dept.name || '',
          value: dept.id,
          key: dept.id,
          children:
            dept.children && dept.children.length > 0 ? buildTreeData(dept.children) : undefined,
        }));
      };
      // 数据直接在 result.data 中，不是 result.data.list
      const deptList = Array.isArray(result.data) ? result.data : [];
      setDeptTreeData(buildTreeData(deptList));
    } catch (error) {
      handleApiError(error, '加载部门选项', false);
    }
  }, [setDeptTreeData]);

  const loadUserOptions = useCallback(
    async (deptId?: string) => {
      try {
        const params: Record<string, unknown> = { current: 1, pageSize: 1000 };
        if (deptId) {
          params.deptId = deptId;
        }
        const result = await post('/system/users/list', params) as { data?: { list: Record<string, unknown>[] } };
        const users = result.data?.list || [];
        setUserOptions(users.map((u: Record<string, unknown>) => ({ id: String(u.id), name: String(u.username) })));
      } catch (error) {
        handleApiError(error, '加载用户选项', false);
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
        handleApiError(error, '加载平面图数据', false);
        return [];
      }
    },
    []
  );

  const loadWorkstations = useCallback(
    async (params?: Record<string, unknown>) => {
      await workstationApi.list(params || {});
    },
    []
  );

  return {
    loadWorkstations,
    loadStatistics,
    loadFloorOptions,
    loadDeptOptions,
    loadUserOptions,
    loadFloorPlanWorkstations,
  };
}
