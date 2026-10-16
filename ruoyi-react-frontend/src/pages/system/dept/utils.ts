/**
 * Department Utils
 * 部门工具函数
 */

import type { Department } from "@/types";
import type { ParentOption } from "./types";

// 树节点类型（包含 children 的 Department）
type DepartmentTreeNode = Department & { children?: DepartmentTreeNode[] };

/** 扁平化树形数据 */
export function flattenTreeToList(tree: DepartmentTreeNode[]): Department[] {
  const result: Department[] = [];
  if (!tree || !Array.isArray(tree)) {
    return result;
  }
  const traverse = (nodes: DepartmentTreeNode[]) => {
    if (!nodes || !Array.isArray(nodes)) {
      return;
    }
    nodes.forEach((node) => {
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

/** 转换为 TreeSelect 所需的树形结构 */
export function transformToParentTreeOptions(data: DepartmentTreeNode[]): ParentOption[] {
  if (!data || !Array.isArray(data)) {
    return [];
  }

  return data.map((item) => ({
    title: item.deptName,
    value: item.id,
    key: item.id,
    children:
      item.children && item.children.length > 0
        ? transformToParentTreeOptions(item.children)
        : undefined,
  }));
}

/** 渲染树形表格数据 */
export function renderTreeData(
  data: DepartmentTreeNode[]
): (DepartmentTreeNode & { key: string })[] {
  if (!data || !Array.isArray(data)) {
    return [];
  }
  return data.map((item) => ({
    ...item,
    key: item.id,
    children: item.children && item.children.length > 0 ? renderTreeData(item.children) : undefined,
  })) as (DepartmentTreeNode & { key: string })[];
}
