/**
 * NetworkDevice 工具函数
 */

import dayjs from 'dayjs';
import type { Department } from '@/types';
import type { SelectOption } from './constants';
import type { ReactElement } from 'react';
import { Select } from 'antd';

const { Option } = Select;

/**
 * 格式化日期时间
 */
export function formatDateTime(dateTime: string | undefined): string {
  if (!dateTime) return '-';
  return dayjs(dateTime).format('YYYY-MM-DD HH:mm:ss');
}

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
