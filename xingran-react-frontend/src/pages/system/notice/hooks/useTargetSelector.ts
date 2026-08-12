import { useState, useCallback, useMemo } from "react";
import { post } from "@/lib/api";
import type { Key } from "react";
import { useDeptTree } from "@/hooks/useDeptTree";
import { toShortNameDataNode } from "@/utils/deptUtils";

export interface Target {
  id: string;
  key?: Key;
  title?: string;
  value?: string;
  deptName?: string;
  roleName?: string;
  roleKey?: string;
  username?: string;
  nickname?: string;
  children?: Target[];
}

interface UseTargetSelectorResult {
  deptTree: Target[];
  roles: Target[];
  users: Target[];
  loadingDepts: boolean;
  loadingRoles: boolean;
  loadingUsers: boolean;
  loadRoles: () => Promise<void>;
  loadUsers: () => Promise<void>;
}

/**
 * 目标选择器数据 Hook
 * 处理部门、角色、用户数据的加载
 *
 * 部门树:消费 canonical useDeptTree (Phase 37 收敛),不再本地 GET fetch。
 * 本地短名转换 (旧函数已删) 改调 deptUtils.toShortNameDataNode:
 *   - dept 子树消费方 (TargetSelector 的 <Tree treeData>) 只读 {title,key,children},
 *     不依赖透传的 deptName/status/id 等额外字段,行为等价。
 *   - roles/users 子树仍保留本地 fetch (post /system/roles/all + /system/users/list),
 *     Target 接口保留 roleName/username 等字段供这两个子树使用。
 */
export function useTargetSelector(): UseTargetSelectorResult {
  const [roles, setRoles] = useState<Target[]>([]);
  const [users, setUsers] = useState<Target[]>([]);
  const [loadingRoles, setLoadingRoles] = useState(false);
  const [loadingUsers, setLoadingUsers] = useState(false);

  // 部门树:消费 useDeptTree 共享缓存 (queryKey queryKeys.dept.tree())。
  // 默认 staleTime 5min / gcTime 30min / refetchOnWindowFocus:false。
  const { data: rawDept = [], isLoading: loadingDepts } = useDeptTree();

  // 派生 antd Tree 期望的 DataNode 形状 (短名 title + key + children)。
  // 旧实现透传 ...node 是冗余(TargetSelector 只读 title/key/children),此处等价。
  const deptTree = useMemo<Target[]>(
    () => toShortNameDataNode(rawDept) as Target[],
    [rawDept]
  );

  // 加载角色列表
  const loadRoles = useCallback(async () => {
    setLoadingRoles(true);
    try {
      const response = await post<Target[]>("/system/roles/all", {});
      setRoles(response.data || []);
    } catch (error) {
      console.error("加载角色列表失败:", error);
    } finally {
      setLoadingRoles(false);
    }
  }, []);

  // 加载用户列表
  const loadUsers = useCallback(async () => {
    setLoadingUsers(true);
    try {
      const response = await post<{ list: Target[] }>("/system/users/list", { current: 1, pageSize: 100 });
      setUsers(response.data?.list || []);
    } catch (error) {
      console.error("加载用户列表失败:", error);
    } finally {
      setLoadingUsers(false);
    }
  }, []);

  return {
    deptTree,
    roles,
    users,
    loadingDepts,
    loadingRoles,
    loadingUsers,
    loadRoles,
    loadUsers,
  };
}
