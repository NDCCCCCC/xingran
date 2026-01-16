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

export interface UseWorkstationDataReturn {
  loadWorkstations: (params?: any) => Promise<void>;
  loadStatistics: () => Promise<void>;
  loadFloorOptions: () => Promise<void>;
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
  const loadStatistics = useCallback(async () => {
    try {
      const result = await workstationApi.list({ current: 1, pageSize: 1000 });
      const allWorkstations = result.data?.list || [];
      setStatistics({
        total: allWorkstations.length,
        available: allWorkstations.filter((w: WorkstationOps) => w.status === 0).length,
        occupied: allWorkstations.filter((w: WorkstationOps) => w.status === 1).length,
        maintain: allWorkstations.filter((w: WorkstationOps) => w.status === 2).length,
      });
    } catch (error) {
      handleApiError(error, '加载统计数据', false);
    }
  }, [setStatistics]);

  const loadFloorOptions = useCallback(async () => {
    try {
      const result = await floorApi.list({ current: 1, pageSize: 1000 });
      const floors = result.data?.list || [];
      setFloorOptions(
        floors.map((f: any) => ({
          code: f.id,
          name: `${f.buildingName || f.buildingCode} - ${f.name || f.floorNo}`,
        }))
      );
    } catch (error) {
      handleApiError(error, '加载楼层选项', false);
    }
  }, [setFloorOptions]);

  const loadDeptOptions = useCallback(async () => {
    try {
      const result = await post('/system/departments/list', {});
      // 构建树形结构数据
      const buildTreeData = (depts: any[]): DeptTreeNode[] => {
        return depts.map((dept) => ({
          title: dept.deptName || dept.name,
          value: dept.id,
          key: dept.id,
          children:
            dept.children && dept.children.length > 0 ? buildTreeData(dept.children) : undefined,
        }));
      };
      const deptList = Array.isArray(result.data?.list) ? result.data.list : [];
      setDeptTreeData(buildTreeData(deptList));
    } catch (error) {
      handleApiError(error, '加载部门选项', false);
    }
  }, [setDeptTreeData]);

  const loadUserOptions = useCallback(
    async (deptId?: string) => {
      try {
        const params: any = { current: 1, pageSize: 1000 };
        if (deptId) {
          params.deptId = deptId;
        }
        const result = await post('/system/users/list', params);
        const users = result.data?.list || [];
        setUserOptions(users.map((u: any) => ({ id: u.id, name: u.username })));
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
    async (params?: any) => {
      await workstationApi.list(params);
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
