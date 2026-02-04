import { Tag, Button, Space } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { formatDateTime } from '@/utils/datetime';

// 泛型类型定义，用于表格辅助函数
type TableRowData = Record<string, unknown>;

export interface ColumnConfig {
  title: string;
  dataIndex: string;
  key: string;
  width?: number;
  align?: 'left' | 'center' | 'right';
}

export function createStatusColumn<T extends TableRowData = TableRowData>(
  field: string = 'status',
  config: Partial<ColumnConfig> = {}
): ColumnsType<T>[0] {
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

export function createDateTimeColumn<T extends TableRowData = TableRowData>(
  field: string = 'createdAt',
  config: Partial<ColumnConfig> = {}
): ColumnsType<T>[0] {
  return {
    title: '创建时间',
    dataIndex: field,
    key: field,
    width: 180,
    render: (text: string) => formatDateTime(text),
    ...config,
  };
}

export function createActionColumn<T extends TableRowData = TableRowData>(
  onEdit?: (record: T) => void,
  onDelete?: (record: T) => void,
  extraActions?: (record: T) => React.ReactNode
): ColumnsType<T>[0] {
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

export function createTagColumn<T extends TableRowData = TableRowData>(
  field: string,
  colorMap: Record<string, string> = {},
  labelMap: Record<string, string> = {}
): ColumnsType<T>[0] {
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

export function createIndexColumn<T extends TableRowData = TableRowData>(
  currentPage: number,
  pageSize: number
): ColumnsType<T>[0] {
  return {
    title: '序号',
    key: 'index',
    width: 60,
    align: 'center',
    render: (_, __, index) => (currentPage - 1) * pageSize + index + 1,
  };
}

export const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};
