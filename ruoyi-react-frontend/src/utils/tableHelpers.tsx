import { Tag, Button, Space } from "antd";
import type { ColumnsType, ColumnType, ColumnGroupType } from "antd/es/table";
import { EditOutlined, DeleteOutlined } from "@ant-design/icons";
import { formatDateTime } from "@/utils/datetime";

export interface ColumnConfig {
  title: string;
  dataIndex: string;
  key: string;
  width?: number;
  align?: "left" | "center" | "right";
}

// 使用更宽松的类型约束，同时保持类型安全
type TableColumn<T> = ColumnGroupType<T> | ColumnType<T>;

export function createStatusColumn<T>(
  field: string = "status",
  config: Partial<ColumnConfig> = {}
): TableColumn<T> {
  return {
    title: "状态",
    dataIndex: field,
    key: field,
    width: 80,
    render: (status: number) => (
      <Tag color={status === 0 ? "success" : "error"}>
        {status === 0 ? "正常" : "停用"}
      </Tag>
    ),
    ...config,
  };
}

export function createDateTimeColumn<T>(
  field: string = "createdAt",
  config: Partial<ColumnConfig> = {}
): TableColumn<T> {
  return {
    title: "创建时间",
    dataIndex: field,
    key: field,
    width: 180,
    render: (text: string) => formatDateTime(text),
    ...config,
  };
}

export function createActionColumn<T>(
  onEdit?: (record: T) => void,
  onDelete?: (record: T) => void,
  extraActions?: (record: T) => React.ReactNode
): TableColumn<T> {
  return {
    title: "操作",
    key: "action",
    width: 200,
    fixed: "right",
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

export function createTagColumn<T>(
  field: string,
  colorMap: Record<string, string> = {},
  labelMap: Record<string, string> = {}
): TableColumn<T> {
  return {
    title: field,
    dataIndex: field,
    key: field,
    render: (value: string) => {
      const color = colorMap[value] || "default";
      const label = labelMap[value] || value || "-";
      return <Tag color={color}>{label}</Tag>;
    },
  };
}

export function createIndexColumn<T>(
  currentPage: number,
  pageSize: number
): TableColumn<T> {
  return {
    title: "序号",
    key: "index",
    width: 60,
    align: "center",
    render: (_, __, index) => (currentPage - 1) * pageSize + index + 1,
  };
}

export const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
};
