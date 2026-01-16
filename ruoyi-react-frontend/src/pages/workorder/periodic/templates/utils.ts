/**
 * Periodic Template Utilities
 * 周期性工单模板工具函数
 */

import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import type { WorkOrderCategory } from '@/lib/workorderApi';

dayjs.extend(utc);

/** 格式化时间 */
export function formatTime(date: string | Date, format: string = 'YYYY-MM-DD HH:mm'): string {
  if (!date) return '-';
  const dateStr = typeof date === 'string' ? date.replace('Z', '').replace('z', '') : date;
  return dayjs(dateStr).format(format);
}

/** 将工单分类数据转换为 TreeSelect 需要的树形结构 */
export interface CategoryTreeNode {
  title: string;
  value: string;
  key: string;
  children?: CategoryTreeNode[];
}

export function buildCategoryTree(categories: WorkOrderCategory[]): CategoryTreeNode[] {
  return categories.map((cat) => ({
    title: cat.categoryName,
    value: cat.id,
    key: cat.id,
    children: cat.children && cat.children.length > 0 ? buildCategoryTree(cat.children) : undefined,
  }));
}

/** 变量帮助说明组件内容 */
export const VARIABLE_HELP_CONTENT = {
  title: '工单标题支持以下变量：',
  variables: [
    { code: '{date}', description: '当前日期 (2025-01-04)' },
    { code: '{datetime}', description: '日期时间 (2025-01-04 15:30:45)' },
    { code: '{year}', description: '年份 (2025)' },
    { code: '{month}', description: '月份 (01)' },
    { code: '{day}', description: '日 (04)' },
    { code: '{weekday}', description: '星期 (Monday)' },
    { code: '{hour}', description: '小时 (15)' },
    { code: '{minute}', description: '分钟 (30)' },
  ],
};
