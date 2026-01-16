import { Tag, Button, Space } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { EditOutlined, DeleteOutlined } from '@ant-design/icons';

/**
 * 表格配置工具
 * 提供常用的表格列配置生成器
 */

export interface ColumnConfig {
  title: string;
  dataIndex: string;
  key: string;
  width?: number;
  align?: 'left' | 'center' | 'right';
}

/**
 * 创建状态列配置
 * @param field 字段名
 * @param config 列配置
 */
export function createStatusColumn(
  field: string = 'status',
  config: Partial<ColumnConfig> = {}
): ColumnsType<any>[0] {
  return {
    title: '状态',
    dataIndex: field,
    key: field,
    width: 80,
    render: (status: number) => (
      <Tag color={status === 0 ? 'success' : 'error'}>
        {status === 0 ? '正常' : '停用'}
      </Tag>
    ),
    ...config,
  };
}

/**
 * 创建时间列配置
 * @param field 字段名
 * @param config 列配置
 */
export function createDateTimeColumn(
  field: string = 'createdAt',
  config: Partial<ColumnConfig> = {}
): ColumnsType<any>[0] {
  return {
    title: '创建时间',
    dataIndex: field,
    key: field,
    width: 180,
    render: (text: string) => (text ? new Date(text).toLocaleString() : '-'),
    ...config,
  };
}

/**
 * 创建操作列配置
 * @param onEdit 编辑回调
 * @param onDelete 删除回调
 * @param extraActions 额外操作按钮
 */
export function createActionColumn(
  onEdit?: (record: any) => void,
  onDelete?: (record: any) => void,
  extraActions?: (record: any) => React.ReactNode
): ColumnsType<any>[0] {
  return {
    title: '操作',
    key: 'action',
    width: 200,
    fixed: 'right',
    render: (_, record) => (
      <Space size="middle">
        {onEdit && (
          <Button type="link" icon={<EditOutlined />} onClick={() => onEdit(record)}>
            编辑
          </Button>
        )}
        {onDelete && (
          <Button type="link" danger icon={<DeleteOutlined />} onClick={() => onDelete(record)}>
            删除
          </Button>
        )}
        {extraActions?.(record)}
      </Space>
    ),
  };
}

/**
 * 创建标签列配置
 * @param field 字段名
 * @param colorMap 标签颜色映射
 * @param labelMap 标签文本映射
 */
export function createTagColumn(
  field: string,
  colorMap: Record<string, string> = {},
  labelMap: Record<string, string> = {}
): ColumnsType<any>[0] {
  return {
    title: field,
    dataIndex: field,
    key: field,
    render: (value: string) => {
      const color = colorMap[value] || 'default';
      const label = labelMap[value] || value || '-';
      return <Tag color={color}>{label}</Tag>;
    },
  };
}

/**
 * 创建索引列配置
 * @param currentPage 当前页码
 * @param pageSize 每页条数
 */
export function createIndexColumn(
  currentPage: number,
  pageSize: number
): ColumnsType<any>[0] {
  return {
    title: '序号',
    key: 'index',
    width: 60,
    align: 'center',
    render: (_, __, index) => (currentPage - 1) * pageSize + index + 1,
  };
}

/**
 * 时间格式化工具函数
 */
export const formatDateTime = (date: string | Date): string => {
  if (!date) return '-';
  const d = typeof date === 'string' ? new Date(date) : date;
  return d.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
};

/**
 * 时间格式化（仅日期）
 */
export const formatDate = (date: string | Date): string => {
  if (!date) return '-';
  const d = typeof date === 'string' ? new Date(date) : date;
  return d.toLocaleDateString('zh-CN');
};

/**
 * 文件大小格式化
 */
export const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};
