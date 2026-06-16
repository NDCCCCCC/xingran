/**
 * User 工具函数
 */

import type { ReactElement } from "react";
import { Select } from "antd";
import { GENDER_OPTIONS, STATUS_TAG_CONFIG } from "./constants";

const { Option } = Select;

/**
 * TreeSelect 数据节点接口
 */
interface TreeNode {
  title: string;
  value: string;
  key: string;
  children?: TreeNode[];
}

/**
 * 将部门数据转换为 TreeSelect 需要的 treeData 格式
 */
export function convertDeptTreeData(departments: { id: string; deptName: string; children?: unknown[] }[]): TreeNode[] {
  if (!departments || departments.length === 0) {
    return [];
  }

  const convertNodes = (nodes: { id: string; deptName: string; children?: unknown[] }[]): TreeNode[] => {
    return nodes.map(node => ({
      title: node.deptName,
      value: node.id,
      key: node.id,
      children: node.children && node.children.length > 0 ? convertNodes(node.children as { id: string; deptName: string; children?: unknown[] }[]) : undefined,
    }));
  };

  return convertNodes(departments);
}

/**
 * 渲染树形部门选项（用于普通 Select）
 */
export function renderDeptTreeOptions(departments: { id: string; deptName: string; children?: unknown[] }[]): ReactElement[] {
  if (!departments || departments.length === 0) {
    return [<Option key="" value="">暂无部门数据</Option>];
  }

  // 递归渲染树形选项
  const renderOptions = (nodes: { id: string; deptName: string; children?: unknown[] }[], level = 0): ReactElement[] => {
    const options: ReactElement[] = [];

    nodes.forEach(node => {
      options.push(
        <Option key={node.id} value={node.id}>
          {"　".repeat(level * 2)}{node.deptName}
        </Option>
      );

      // 递归渲染子部门
      if (node.children && node.children.length > 0) {
        options.push(...renderOptions(node.children as { id: string; deptName: string; children?: unknown[] }[], level + 1));
      }
    });

    return options;
  };

  return renderOptions(departments);
}

/**
 * 格式化性别显示
 */
export function formatGender(gender: number): string {
  const option = GENDER_OPTIONS.find(opt => opt.value === gender);
  return option?.label || "保密";
}

/**
 * 格式化状态显示
 */
export function formatStatus(status: number): { text: string; color: string } {
  return STATUS_TAG_CONFIG[status] || { text: "未知", color: "default" };
}
