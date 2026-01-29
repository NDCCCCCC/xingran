/**
 * NetworkDevice 工具函数
 */

import dayjs from 'dayjs';
import type { Department } from '@/types';
import type { SelectOption } from './constants';
import type { ReactElement } from 'react';
import { Select } from 'antd';
import type { DataNode } from 'antd/es/tree';

const { Option } = Select;

// 统一时间格式化函数从 @/utils/datetime 导入
export { formatDateTime } from '@/utils/datetime';

/**
 * 根据值获取选项标签
 */
export function getOptionLabel(options: SelectOption[], value: string | number): string | undefined {
  return options.find(o => o.value === value)?.label;
}

/**
 * 渲染树形部门选项
 */
export function renderDeptTreeOptions(
  departments: Department[]
): ReactElement[] {
  if (!departments || departments.length === 0) {
    return [<Option key="" value="">暂无部门数据</Option>];
  }

  const renderOptions = (nodes: Department[], level = 0): ReactElement[] => {
    const options: ReactElement[] = [];

    nodes.forEach(node => {
      options.push(
        <Option key={node.id} value={node.id}>
          {'　'.repeat(level * 2)}{node.deptName}
        </Option>
      );

      if (node.children && node.children.length > 0) {
        options.push(...renderOptions(node.children, level + 1));
      }
    });

    return options;
  };

  return renderOptions(departments);
}

/**
 * 将部门数据转换为 TreeSelect 需要的树形数据格式
 */
export function transformDepartmentsToTreeData(departments: Department[]): DataNode[] {
  if (!departments || departments.length === 0) {
    return [];
  }

  const transform = (nodes: Department[]): DataNode[] => {
    return nodes.map(node => ({
      title: node.deptName,
      value: node.id,
      key: node.id,
      children: node.children && node.children.length > 0 ? transform(node.children) : undefined,
    }));
  };

  return transform(departments);
}

/**
 * 获取第一级节点的 key（用于 TreeSelect 默认展开）
 */
export function getFirstLevelKeys(departments: Department[]): string[] {
  if (!departments || departments.length === 0) {
    return [];
  }
  return departments.map(dept => dept.id);
}

/**
 * 获取状态颜色
 */
export function getStatusColor(status: number): string {
  const colorMap: Record<number, string> = {
    0: 'success',
    1: 'error',
    2: 'default',
  };
  return colorMap[status] || 'default';
}
