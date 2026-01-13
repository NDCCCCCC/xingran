import { useState, useCallback } from 'react';
import { get, post } from '@/lib/api';

interface Target {
  id: string;
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
  loadDeptTree: () => Promise<void>;
  loadRoles: () => Promise<void>;
  loadUsers: () => Promise<void>;
}

/**
 * 目标选择器数据 Hook
 * 处理部门、角色、用户数据的加载
 */
export function useTargetSelector(): UseTargetSelectorResult {
  const [deptTree, setDeptTree] = useState<Target[]>([]);
  const [roles, setRoles] = useState<Target[]>([]);
  const [users, setUsers] = useState<Target[]>([]);
  const [loadingDepts, setLoadingDepts] = useState(false);
  const [loadingRoles, setLoadingRoles] = useState(false);
  const [loadingUsers, setLoadingUsers] = useState(false);

  // 加载部门树
  const loadDeptTree = useCallback(async () => {
    setLoadingDepts(true);
    try {
      const response: { data: { list: any[] } } = await get('/system/departments/tree');
      const convertTree = (nodes: any[]): Target[] => {
        return nodes.map(node => ({
          ...node,
          title: node.deptName,
          key: node.id,
          value: node.id,
          children: node.children ? convertTree(node.children) : undefined,
        }));
      };
      setDeptTree(convertTree(response.data?.list || []));
    } catch (error) {
      console.error('加载部门树失败:', error);
    } finally {
      setLoadingDepts(false);
    }
  }, []);

  // 加载角色列表
  const loadRoles = useCallback(async () => {
    setLoadingRoles(true);
    try {
      const response: { data: any[] } = await post('/system/roles/all', {});
      setRoles(response.data || []);
    } catch (error) {
      console.error('加载角色列表失败:', error);
    } finally {
      setLoadingRoles(false);
    }
  }, []);

  // 加载用户列表
  const loadUsers = useCallback(async () => {
    setLoadingUsers(true);
    try {
      const response: { data: { list: any[] } } = await post('/system/users/list', { current: 1, pageSize: 100 });
      setUsers(response.data?.list || []);
    } catch (error) {
      console.error('加载用户列表失败:', error);
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
    loadDeptTree,
    loadRoles,
    loadUsers,
  };
}
