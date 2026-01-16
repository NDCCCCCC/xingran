/**
 * WorkOrder 工具函数
 */

import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import 'dayjs/locale/zh-cn';
import type { WorkOrderCategory } from '@/lib/workorderApi';

// 配置 dayjs - 使用 UTC 插件来正确处理后端返回的 UTC 时间
dayjs.extend(utc);
dayjs.locale('zh-cn');

/**
 * 格式化时间
 * 注意：后端返回的时间虽然带Z后缀，但实际存储的是本地时间（Asia/Shanghai）
 * 需要去掉Z后缀，让dayjs把时间当作本地时间处理
 */
export function formatTime(date: string | Date, formatStr: string = 'YYYY-MM-DD HH:mm:ss'): string {
  const dateStr = typeof date === 'string' ? date.replace('Z', '').replace('z', '') : date;
  return dayjs(dateStr).format(formatStr);
}

/**
 * 将工单分类数据转换为 TreeSelect 需要的树形结构
 */
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
