/**
 * User 数据管理 Hook
 */

import { useState, useCallback } from "react";
import { post } from "@/lib/api";
import { handleApiError } from "@/utils/errorHandler";

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

  // 加载统计数据: 调用专用统计端点(COUNT 聚合)。
  // 不再用列表长度充当总数——后端 user list 的 pageSize 上限为 100(MaxPageSize),
  // 用户数超过 100 后统计会错误地卡在 100。
  const loadStatistics = useCallback(async () => {
    try {
      const result = await post<UserStatistics>("/system/users/statistics");
      setStatistics({
        total: result.data?.total ?? 0,
        active: result.data?.active ?? 0,
        inactive: result.data?.inactive ?? 0,
      });
    } catch (error) {
      handleApiError(error, "加载统计数据", false);
    }
  }, []);

  // 加载角色列表
  const loadRoles = useCallback(async () => {
    try {
      const result = await post<Role[]>("/system/roles/all");
      setRoles(result.data || []);
    } catch (error) {
      handleApiError(error, "加载角色列表", false);
    }
  }, []);

  // 加载部门列表
  const loadDepartments = useCallback(async () => {
    try {
      const result = await post<Department[]>("/system/departments/tree");
      setDepartments(result.data || []);
    } catch (error) {
      handleApiError(error, "加载部门列表", false);
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
