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
import { post } from "@/lib/api";

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

  // 加载统计数据: 调用专用统计端点(COUNT 聚合),不再用 pageSize:1000 全量列表的
  // .length 充当总数——system 模块 MaxPageSize=100 会把请求钳到 100,角色 >100 时
  // 统计会错误卡在 100。增删改后由 useRoleActions 显式调用本方法刷新。
  const loadStatistics = useCallback(async () => {
    try {
      const result = await post<RoleStatistics>("/system/roles/statistics");
      setStatistics({
        total: result.data?.total ?? 0,
        active: result.data?.active ?? 0,
        inactive: result.data?.inactive ?? 0,
      });
    } catch (error) {
      console.error("加载角色统计失败:", error);
    }
  }, []);

  // 初次挂载时加载统计;后续由 useRoleActions 在增删改后调用 loadStatistics 刷新。
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
          children:
            node.children && node.children.length > 0 ? processTreeData(node.children) : undefined,
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
          children:
            node.children && node.children.length > 0 ? processTreeData(node.children) : undefined,
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
