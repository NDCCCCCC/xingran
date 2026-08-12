// ========================================
// 辅助函数：菜单层级和关系判断
// ========================================

import type { Menu as MenuType } from "@/types";

interface MenuPathInfo {
  topLevel: string;      // 所属一级菜单的key
  level: number;          // 层级(1/2/3)
  secondLevel?: string;   // 所属二级菜单的key(仅三级菜单需要)
  fullPath: string;       // 完整路径
}

/**
 * 构建菜单路径映射表
 * 用于快速查询任意菜单的父级菜单路径
 */
export function buildMenuPathMap(
  menuList: MenuType[]
): Map<string, MenuPathInfo> {
  const pathMap = new Map<string, MenuPathInfo>();

  const traverse = (
    menus: MenuType[],
    parentPath: string = "",
    topLevelKey: string = "",
    secondLevelKey: string = "",
    isTopLevel: boolean = true
  ) => {
    for (const menu of menus) {
      if (menu.menuType === "F" || menu.visible !== 1) continue;

      // 构建当前菜单的完整路径
      let menuPath = menu.path || "";
      if (menuPath && !menuPath.startsWith("/") && parentPath) {
        if (!menuPath.startsWith(parentPath + "/")) {
          menuPath = `${parentPath}/${menuPath}`;
        }
      }

      // 生成菜单key(与convertToMenuItem逻辑一致，统一使用menu.id)
      const menuKey = menu.id;

      // 确定当前菜单的层级
      const level = isTopLevel ? 1 : (topLevelKey && !secondLevelKey ? 2 : 3);

      // 构建菜单信息对象
      const menuInfo = {
        topLevel: topLevelKey || menuKey,
        level,
        secondLevel: level === 3 ? secondLevelKey : undefined,
        fullPath: menuPath || menu.id
      };

      // 存储映射关系 - 支持多种 key 查找方式
      // 1. 存储完整路径（去掉前导斜杠）
      if (menuPath) {
        const normalizedPath = menuPath.startsWith("/") ? menuPath.slice(1) : menuPath;
        pathMap.set(normalizedPath, menuInfo);

        // 同时存储带前导斜杠的路径（用于匹配 Ant Design Menu 的 key）
        pathMap.set(menuPath, menuInfo);
      }

      // 2. 存储菜单 ID（用于通过 ID 查找）
      if (menu.id) {
        pathMap.set(menu.id, menuInfo);
      }

      // 递归处理子菜单
      if (menu.children && menu.children.length > 0) {
        const newSecondLevelKey = level === 2 ? menuKey : secondLevelKey;
        traverse(
          menu.children,
          menuPath,
          topLevelKey || menuKey,
          newSecondLevelKey,
          false
        );
      }
    }
  };

  traverse(menuList);
  return pathMap;
}

/**
 * 判断两个菜单key是否属于同一个一级菜单
 */
export function isSameTopLevelMenu(
  key1: string,
  key2: string,
  pathMap: Map<string, MenuPathInfo>
): boolean {
  const info1 = pathMap.get(key1);
  const info2 = pathMap.get(key2);

  if (!info1 || !info2) return false;
  return info1.topLevel === info2.topLevel;
}

/**
 * 获取菜单的层级
 */
export function getMenuLevel(key: string, pathMap: Map<string, MenuPathInfo>): number {
  const info = pathMap.get(key);
  return info?.level || 0;
}

/**
 * 判断是否为三级菜单
 */
export function isThirdLevelMenu(key: string, pathMap: Map<string, MenuPathInfo>): boolean {
  return getMenuLevel(key, pathMap) === 3;
}

/**
 * 判断是否为二级菜单
 */
export function isSecondLevelMenu(key: string, pathMap: Map<string, MenuPathInfo>): boolean {
  return getMenuLevel(key, pathMap) === 2;
}

/**
 * 判断是否为一级菜单
 */
export function isTopLevelMenu(key: string, pathMap: Map<string, MenuPathInfo>): boolean {
  return getMenuLevel(key, pathMap) === 1;
}
