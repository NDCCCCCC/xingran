/**
 * Role Data Hook
 * 角色辅助数据管理 Hook（菜单树 / 部门树 / 统计 / 权限勾选）
 *
 * 角色列表的分页 / 搜索 / 排序由 useTableManager 统一管理（见 index.tsx），
 * 本 hook 只负责列表之外的辅助数据。
 */

import { useState, useCallback, useEffect } from "react";
import type { TreeDataNode } from "antd";
import type { Key } from "antd/es/table/interface";
import type { Role } from "@/types";
import { post } from "@/lib/api";
import { useRoleList } from "@/hooks/useRoleList";

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

export interface UseRoleDataReturn {
  // 数据状态
  menuTree: TreeDataNode[];
  deptTree: TreeDataNode[];
  statistics: RoleStatistics;

  // 选中的数据
  checkedMenuKeys: Key[];
  checkedDeptKeys: Key[];
  currentDataScope: number;

  // Setters
  setMenuTree: React.Dispatch<React.SetStateAction<TreeDataNode[]>>;
  setDeptTree: React.Dispatch<React.SetStateAction<TreeDataNode[]>>;
  setStatistics: React.Dispatch<React.SetStateAction<RoleStatistics>>;
  setCheckedMenuKeys: React.Dispatch<React.SetStateAction<Key[]>>;
  setCheckedDeptKeys: React.Dispatch<React.SetStateAction<Key[]>>;
  setCurrentDataScope: React.Dispatch<React.SetStateAction<number>>;

  // 数据加载方法
  loadStatistics: () => void;
  loadMenuTree: () => Promise<void>;
  loadDeptTree: () => Promise<void>;
}

export function useRoleData(): UseRoleDataReturn {
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

  // 加载统计数据 — 使用 useRoleList 全量缓存 (D-13 Step 2)
  // useRoleList 数据更新时 statistics 自动同步。
  const loadStatistics = useCallback(() => {
    const list = allRoles as unknown as Role[];
    setStatistics({
      total: list.length,
      active: list.filter((r: Role) => r.status === 0).length,
      inactive: list.filter((r: Role) => r.status === 1).length,
    });
  }, [allRoles]);

  // 当 useRoleList 全量数据变化时自动同步 statistics
  useEffect(() => {
    loadStatistics();
  }, [loadStatistics]);

  // 加载菜单树
  const loadMenuTree = useCallback(async () => {
    try {
      const result = await post<MenuTreeNode[]>("/system/menus/tree-select");
      const processTreeData = (nodes: MenuTreeNode[]): TreeDataNode[] => {
        return nodes.map((node) => ({
          key: node.key || node.id || "",
          title: node.title || node.menuName || "",
          value: node.value || node.id || "",
          children: node.children && node.children.length > 0 ? processTreeData(node.children) : undefined,
        }));
      };
      setMenuTree(processTreeData(result.data || []));
    } catch (error) {
      console.error("加载菜单树失败:", error);
    }
  }, []);

  // 加载部门树
  const loadDeptTree = useCallback(async () => {
    try {
      const result = await post<DeptTreeNode[]>("/system/departments/tree-select");
      const processTreeData = (nodes: DeptTreeNode[]): TreeDataNode[] => {
        return nodes.map((node) => ({
          key: node.key || node.id || "",
          title: node.title || node.deptName || "",
          value: node.value || node.id || "",
          children: node.children && node.children.length > 0 ? processTreeData(node.children) : undefined,
        }));
      };
      setDeptTree(processTreeData(result.data || []));
    } catch (error) {
      console.error("加载部门树失败:", error);
    }
  }, []);

  return {
    menuTree,
    deptTree,
    statistics,
    checkedMenuKeys,
    checkedDeptKeys,
    currentDataScope,
    setMenuTree,
    setDeptTree,
    setStatistics,
    setCheckedMenuKeys,
    setCheckedDeptKeys,
    setCurrentDataScope,
    loadStatistics,
    loadMenuTree,
    loadDeptTree,
  };
}
