/**
 * Menu Utilities
 * 菜单管理页面工具函数
 */

import type { Menu } from "@/types";
import { getIconComponent } from "@/utils/iconUtils";
import { getMenuIcon } from "./constants";

/** 扁平化树形数据 */
export function flattenTree(tree: Menu[]): Menu[] {
  const result: Menu[] = [];
  if (!tree || !Array.isArray(tree)) {
    return result;
  }
  const traverse = (nodes: Menu[]) => {
    if (!nodes || !Array.isArray(nodes)) {
      return;
    }
    nodes.forEach(node => {
      if (node) {
        result.push(node);
        if (node.children && node.children.length > 0) {
          traverse(node.children);
        }
      }
    });
  };
  traverse(tree);
  return result;
}

/** 构建父级选项 */
export interface ParentOption {
  title: string;
  value: string;
  key: string;
  children?: ParentOption[];
}

export function buildParentOptions(tree: Menu[]): ParentOption[] {
  const options: ParentOption[] = [{ title: "顶级菜单", value: "", key: "" }];
  if (!tree || !Array.isArray(tree)) {
    return options;
  }
  const traverse = (nodes: Menu[], parentTitle = "") => {
    if (!nodes || !Array.isArray(nodes)) {
      return;
    }
    nodes.forEach(node => {
      if (node) {
        const option: ParentOption = {
          title: `${parentTitle}${node.menuName}`,
          value: node.id,
          key: node.id,
        };
        options.push(option);
        if (node.children && node.children.length > 0) {
          traverse(node.children, `${parentTitle}${node.menuName} / `);
        }
      }
    });
  };
  traverse(tree);
  return options;
}

/** 渲染树形表格数据 */
export function renderTreeData(data: Menu[]): Menu[] {
  return data.map(item => ({
    ...item,
    key: item.id,
    children: item.children && item.children.length > 0 ? renderTreeData(item.children) : undefined,
  }));
}

/** 计算统计数据 */
export interface MenuStatistics {
  total: number;
  directories: number;
  menus: number;
  buttons: number;
}

export function calculateStatistics(flatMenus: Menu[]): MenuStatistics {
  return {
    total: flatMenus.length,
    directories: flatMenus.filter(m => m.menuType === "M").length,
    menus: flatMenus.filter(m => m.menuType === "C").length,
    buttons: flatMenus.filter(m => m.menuType === "F").length,
  };
}

/** 渲染菜单名称（带图标） */
export function renderMenuName(record: Menu): React.ReactNode {
  return (
    <>
      {record.icon && (
        <span style={{ fontSize: 16 }}>
          {getIconComponent(record.icon)}
        </span>
      )}
      {!record.icon && getMenuIcon(record.menuType)}
      <span style={{ marginLeft: 8 }}>{record.menuName}</span>
    </>
  );
}
