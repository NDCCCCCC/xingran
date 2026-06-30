/**
 * User 数据管理 Hook
 */

import { useState, useCallback } from "react";
import { post } from "@/lib/api";
import { handleApiError } from "@/utils/errorHandler";
import { useDeptTree } from "@/hooks/useDeptTree";
import type { SimpleDept } from "@/lib/dutyApi";

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

export interface UseUserDataReturn {
  statistics: UserStatistics;
  roles: Role[];
  departments: SimpleDept[];
  loadStatistics: () => Promise<void>;
  loadRoles: () => Promise<void>;
  // 注入式兜底(2026-06-30):编辑回填时若 record.roles 含 roleName 但
  // roles 列表未及时加载 → 多选 Select 显示 raw UUID。用 list 元素注入临时 Option。
  ensureRoles: (rolesToEnsure: Array<{ id: string; roleName?: string; roleKey?: string }>) => void;
}

export function useUserData(): UseUserDataReturn {
  const [statistics, setStatistics] = useState<UserStatistics>({
    total: 0,
    active: 0,
    inactive: 0,
  });
  const [roles, setRoles] = useState<Role[]>([]);

  // 部门列表:消费 canonical useDeptTree (Phase 37 收敛)。
  // 共享 React Query 缓存 (5min stale / 30min gc, refetchOnWindowFocus:false)。
  // 写操作后由调用方 useInvalidateDept() 主动失效。
  const { data: departments = [] } = useDeptTree();

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

  // 注入式兜底(2026-06-30):若角色 id 已在列表中则保持原引用,否则追加(去重)
  // setRoles 列入 deps 以满足 React Compiler 推断(setState 引用稳定,不影响实际行为)。
  const ensureRoles = useCallback((rolesToEnsure: Array<{ id: string; roleName?: string; roleKey?: string }>) => {
    setRoles(prev => {
      const known = new Set(prev.map(r => r.id));
      const toAdd: Role[] = [];
      rolesToEnsure.forEach(r => {
        if (!known.has(r.id)) {
          toAdd.push({
            id: r.id,
            roleName: r.roleName || r.roleKey || "未命名角色",
            roleKey: r.roleKey || "",
            status: 0,
          });
          known.add(r.id);
        }
      });
      return toAdd.length ? [...prev, ...toAdd] : prev;
    });
  }, [setRoles]);

  return {
    statistics,
    roles,
    departments,
    loadStatistics,
    loadRoles,
    ensureRoles,
  };
}
