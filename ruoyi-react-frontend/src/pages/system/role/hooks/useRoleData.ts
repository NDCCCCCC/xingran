/**
 * Role Data Hook
 * 角色数据管理 Hook
 */

import { useState, useCallback, useEffect } from 'react';
import type { TreeDataNode } from 'antd';
import type { Key } from 'antd/es/table/interface';
import type { Role } from '@/types';
import { post } from '@/lib/api';
import { useRoleList } from '@/hooks/useRoleList';

// 菜单树节点类型（后端返回）
interface MenuTreeNode {
  id?: string;
  key?: string;
  menuName?: string;
  title?: string;
  value?: string;
  children?: MenuTreeNode[];
}

// 部门树节点类型（后端返回）
interface DeptTreeNode {
  id?: string;
  key?: string;
  deptName?: string;
  title?: string;
  value?: string;
  children?: DeptTreeNode[];
}

export interface RoleStatistics {
  total: number;
  active: number;
  inactive: number;
}

export interface UseRoleDataParams {
  current: number;
  pageSize: number;
}

export interface UseRoleDataReturn {
  // 数据状态
  roles: Role[];
  loading: boolean;
  total: number;
  menuTree: TreeDataNode[];
  deptTree: TreeDataNode[];
  statistics: RoleStatistics;

  // 选中的数据
  checkedMenuKeys: Key[];
  checkedDeptKeys: Key[];
  currentDataScope: number;

  // 数据操作方法
  setRoles: React.Dispatch<React.SetStateAction<Role[]>>;
  setTotal: React.Dispatch<React.SetStateAction<number>>;
  setMenuTree: React.Dispatch<React.SetStateAction<TreeDataNode[]>>;
  setDeptTree: React.Dispatch<React.SetStateAction<TreeDataNode[]>>;
  setStatistics: React.Dispatch<React.SetStateAction<RoleStatistics>>;
  setCheckedMenuKeys: React.Dispatch<React.SetStateAction<Key[]>>;
  setCheckedDeptKeys: React.Dispatch<React.SetStateAction<Key[]>>;
  setCurrentDataScope: React.Dispatch<React.SetStateAction<number>>;

  // 数据加载方法
  loadRoles: (params?: Record<string, unknown>) => Promise<void>;
  loadStatistics: () => void;
  loadMenuTree: () => Promise<void>;
  loadDeptTree: () => Promise<void>;
  loadRoleMenus: (roleId: string) => Promise<string[]>;
  loadRoleDepts: (roleId: string) => Promise<string[]>;
}

export function useRoleData(params: UseRoleDataParams): UseRoleDataReturn {
  const { current, pageSize } = params;

  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [menuTree, setMenuTree] = useState<TreeDataNode[]>([]);
  const [deptTree, setDeptTree] = useState<TreeDataNode[]>([]);
  const [statistics, setStatistics] = useState<RoleStatistics>({
    total: 0,
    active: 0,
    inactive: 0,
  });
  const [checkedMenuKeys, setCheckedMenuKeys] = useState<Key[]>([]);
  const [checkedDeptKeys, setCheckedDeptKeys] = useState<Key[]>([]);
  const [currentDataScope, setCurrentDataScope] = useState<number>(1);

  // 角色全量列表 — 通过 React Query 全局共享 (D-13 Step 2)
  // 用于统计计算；useRoleList 返回 pageSize=1000 的全量数据。
  const { data: allRoles = [] } = useRoleList();

  // 加载角色列表
  const loadRoles = useCallback(async (params: Record<string, unknown> = {}) => {
    setLoading(true);
    try {
      const result = await post('/system/roles/list', {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
        ...params,
      }) as { data: { list: Role[]; total: number } };
      setRoles(result.data.list);
      setTotal(result.data.total);
    } catch (error) {
      console.error('加载角色列表失败:', error);
    } finally {
      setLoading(false);
    }
  }, [current, pageSize]);

  // 加载统计数据 — 优先使用 useRoleList 全量缓存 (D-13 Step 2)
  // 当 useRoleList 数据更新时，statistics 自动同步；首次加载若全量缓存尚未就绪，
  // 回退到当前表格页的 roles 数据计算，避免统计延迟导致空白。
  const loadStatistics = useCallback(() => {
    const list = allRoles.length > 0 ? (allRoles as unknown as Role[]) : roles;
    const activeCount = list.filter((r: Role) => r.status === 0).length;
    const inactiveCount = list.filter((r: Role) => r.status === 1).length;
    setStatistics({
      total: list.length,
      active: activeCount,
      inactive: inactiveCount,
    });
  }, [allRoles, roles]);

  // 当 useRoleList 全量数据或当前页数据变化时自动同步 statistics
  useEffect(() => {
    loadStatistics();
  }, [loadStatistics]);

  // 加载菜单树
  const loadMenuTree = useCallback(async () => {
    try {
      const result = await post<MenuTreeNode[]>('/system/menus/tree-select');
      const processTreeData = (nodes: MenuTreeNode[]): TreeDataNode[] => {
        return nodes.map((node) => ({
          key: node.key || node.id || '',
          title: node.title || node.menuName || '',
          value: node.value || node.id || '',
          children: node.children && node.children.length > 0 ? processTreeData(node.children) : undefined,
        }));
      };
      setMenuTree(processTreeData(result.data || []));
    } catch (error) {
      console.error('加载菜单树失败:', error);
    }
  }, []);

  // 加载部门树
  const loadDeptTree = useCallback(async () => {
    try {
      const result = await post<DeptTreeNode[]>('/system/departments/tree-select');
      const processTreeData = (nodes: DeptTreeNode[]): TreeDataNode[] => {
        return nodes.map((node) => ({
          key: node.key || node.id || '',
          title: node.title || node.deptName || '',
          value: node.value || node.id || '',
          children: node.children && node.children.length > 0 ? processTreeData(node.children) : undefined,
        }));
      };
      setDeptTree(processTreeData(result.data || []));
    } catch (error) {
      console.error('加载部门树失败:', error);
    }
  }, []);

  // 加载角色的菜单权限
  const loadRoleMenus = useCallback(async (roleId: string): Promise<string[]> => {
    try {
      const result = await post(`/system/menus/role-menu-tree-select/${roleId}`) as { data: { checkedKeys?: string[] } };
      return result.data.checkedKeys || [];
    } catch (error) {
      console.error('加载角色菜单权限失败:', error);
      return [];
    }
  }, []);

  // 加载角色的部门权限
  const loadRoleDepts = useCallback(async (roleId: string): Promise<string[]> => {
    try {
      const result = await post(`/system/departments/role-dept-tree-select/${roleId}`) as { data: { checkedKeys?: string[] } };
      return result.data.checkedKeys || [];
    } catch (error) {
      console.error('加载角色部门权限失败:', error);
      return [];
    }
  }, []);

  return {
    roles,
    loading,
    total,
    menuTree,
    deptTree,
    statistics,
    checkedMenuKeys,
    checkedDeptKeys,
    currentDataScope,
    setRoles,
    setTotal,
    setMenuTree,
    setDeptTree,
    setStatistics,
    setCheckedMenuKeys,
    setCheckedDeptKeys,
    setCurrentDataScope,
    loadRoles,
    loadStatistics,
    loadMenuTree,
    loadDeptTree,
    loadRoleMenus,
    loadRoleDepts,
  };
}
