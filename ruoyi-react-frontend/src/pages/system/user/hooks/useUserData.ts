/**
 * User 数据管理 Hook
 */

import { useState, useCallback } from 'react';
import { post } from '@/lib/api';
import { handleApiError } from '@/utils/errorHandler';
import type { BaseResponse, PageResponse } from '@/types';

export interface UserStatistics {
  total: number;
  active: number;
  inactive: number;
}

interface Role {
  id: string;
  roleName: string;
  roleKey: string;
  status: number;
}

interface Department {
  id: string;
  deptName: string;
  status: number;
}

interface User {
  id: string;
  status: number;
}

export interface UseUserDataReturn {
  statistics: UserStatistics;
  roles: Role[];
  departments: Department[];
  loadStatistics: () => Promise<void>;
  loadRoles: () => Promise<void>;
  loadDepartments: () => Promise<void>;
}

export function useUserData(): UseUserDataReturn {
  const [statistics, setStatistics] = useState<UserStatistics>({
    total: 0,
    active: 0,
    inactive: 0,
  });
  const [roles, setRoles] = useState<Role[]>([]);
  const [departments, setDepartments] = useState<Department[]>([]);

  // 加载统计数据
  const loadStatistics = useCallback(async () => {
    try {
      const result = await post<BaseResponse<PageResponse<User>>>('/system/users/list', { current: 1, pageSize: 1000 });
      const allUsers = result.data?.list || [];
      const activeCount = allUsers.filter((u) => u.status === 0).length;
      const inactiveCount = allUsers.filter((u) => u.status === 1).length;
      setStatistics({
        total: allUsers.length,
        active: activeCount,
        inactive: inactiveCount,
      });
    } catch (error) {
      handleApiError(error, '加载统计数据', false);
    }
  }, []);

  // 加载角色列表
  const loadRoles = useCallback(async () => {
    try {
      const result = await post<BaseResponse<Role[]>>('/system/roles/all');
      setRoles(result.data || []);
    } catch (error) {
      handleApiError(error, '加载角色列表', false);
    }
  }, []);

  // 加载部门列表
  const loadDepartments = useCallback(async () => {
    try {
      const result = await post<BaseResponse<Department[]>>('/system/departments/tree');
      setDepartments(result.data || []);
    } catch (error) {
      handleApiError(error, '加载部门列表', false);
      setDepartments([]);
    }
  }, []);

  return {
    statistics,
    roles,
    departments,
    loadStatistics,
    loadRoles,
    loadDepartments,
  };
}
