/**
 * Menu Data Hook
 * 菜单数据管理 Hook
 */

import { useState, useCallback } from "react";
import type { Menu } from "@/types";
import { post } from "@/lib/api";
import {
  flattenTree,
  buildParentOptions,
  calculateStatistics,
  type MenuStatistics,
  type ParentOption,
} from "../utils";

export interface UseMenuDataReturn {
  // 数据状态
  menus: Menu[];
  parentOptions: ParentOption[];
  statistics: MenuStatistics;
  loading: boolean;

  // 数据加载方法
  loadMenus: (searchParams?: Record<string, unknown>) => Promise<void>;
}

export function useMenuData(): UseMenuDataReturn {
  const [menus, setMenus] = useState<Menu[]>([]);
  const [parentOptions, setParentOptions] = useState<ParentOption[]>([]);
  const [statistics, setStatistics] = useState<MenuStatistics>({
    total: 0,
    directories: 0,
    menus: 0,
    buttons: 0,
  });
  const [loading, setLoading] = useState(false);

  // 加载菜单列表
  const loadMenus = useCallback(async (searchParams?: Record<string, unknown>) => {
    setLoading(true);
    try {
      const result = (await post("/system/menus/tree", searchParams || {})) as { data: Menu[] };
      const menuData = result.data || [];
      // 处理树形数据
      const flatMenus = flattenTree(menuData);
      setMenus(menuData);
      setParentOptions(buildParentOptions(menuData));

      // 统计数据
      setStatistics(calculateStatistics(flatMenus));
    } catch (error) {
      console.error("加载菜单列表失败:", error);
      // 设置默认值
      setMenus([]);
      setParentOptions([]);
      setStatistics({ total: 0, directories: 0, menus: 0, buttons: 0 });
    } finally {
      setLoading(false);
    }
  }, []);

  return {
    menus,
    parentOptions,
    statistics,
    loading,
    loadMenus,
  };
}
