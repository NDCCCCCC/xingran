/**
 * Sidebar 组件工具函数
 * Sidebar component utility functions
 */

import type { Menu as MenuType } from "@/types";

/**
 * 构建完整的菜单路径
 */
export function buildFullPath(menu: MenuType, parentPath: string = ""): string {
  const menuPath = menu.path || "";
  if (!menuPath) return parentPath;

  // 如果已经是绝对路径，直接返回
  if (menuPath.startsWith("/")) {
    return menuPath;
  }

  // 如果没有父路径，返回菜单路径
  if (!parentPath) {
    return `/${menuPath}`;
  }

  // 拼接父路径和菜单路径
  return `${parentPath}/${menuPath}`;
}

/**
 * 在菜单树中查找菜单
 */
export function findMenuById(menuList: MenuType[], id: string): MenuType | null {
  for (const menu of menuList) {
    if (menu.id === id) {
      return menu;
    }
    // 递归查找子菜单
    if (menu.children) {
      const found = findMenuById(menu.children, id);
      if (found) return found;
    }
  }
  return null;
}

/**
 * 通过完整路径查找菜单ID
 */
export function findMenuByFullPath(menuList: MenuType[], fullPath: string): string | null {
  const _normalizePath = (path: string): string => {
    return path.startsWith("/") ? path.slice(1) : path;
  };

  for (const menu of menuList) {
    if (menu.menuType === "F") continue;

    const menuPath = menu.path || "";
    let computedPath = menuPath;
    const buildFullPath = (m: MenuType, parent: string = ""): string => {
      const p = m.path || "";
      if (!p) return parent;
      if (p.startsWith("/")) return p.slice(1);
      if (!parent) return p;
      return p.startsWith(parent + "/") ? p : `${parent}/${p}`;
    };

    computedPath = buildFullPath(menu, "");

    // 检查一级菜单
    if (computedPath === fullPath || menuPath === fullPath) {
      return menu.id;
    }

    // 递归检查子菜单
    if (menu.children) {
      for (const child of menu.children) {
        const childFullPath = buildFullPath(child, computedPath);
        if (childFullPath === fullPath || child.path === fullPath) {
          return child.id;
        }

        // 检查三级菜单
        if (child.children) {
          for (const grandChild of child.children) {
            const grandChildFullPath = buildFullPath(grandChild, childFullPath);
            if (grandChildFullPath === fullPath || grandChild.path === fullPath) {
              return grandChild.id;
            }
          }
        }
      }
    }
  }

  return null;
}
