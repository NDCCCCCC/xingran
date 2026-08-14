import { Tag, Button, Space } from "antd";
import type { ColumnsType, ColumnType, ColumnGroupType } from "antd/es/table";
import { EditOutlined, DeleteOutlined } from "@ant-design/icons";
import dayjs from "dayjs";
import { formatDateTime } from "@/utils/datetime";

/**
 * Sorter 类型：标识字段的排序语义
 * - string:  按本地化字符串比较（支持中英文）
 * - number:  按数字比较
 * - date:    按 dayjs 解析后的时间戳比较（兼容字符串/Date/dayjs 对象）
 * - boolean: true 排在前
 * - custom:  使用传入的 compareFn
 */
export type SorterType = "string" | "number" | "date" | "boolean" | "custom";

/**
 * 排序字段元数据：用于服务端排序场景下描述"该列可按哪个字段名请求后端排序"。
 *
 * 与 createSorter 的区别:
 *   - createSorter(field, type) 返回闭包，antd 内部对当前页数据做客户端比较（换页失效）
 *   - createSorterMeta(field, type) 返回元数据，useServerSort hook 据此发请求给后端（跨页一致）
 *
 * 两者 field 命名约定一致（应与后端白名单 key 对应），可同时挂在一列上：
 *
 *   {
 *     title: "用户名",
 *     dataIndex: "username",
 *     sorter: createSorter<User>("username", "string"),     // 兜底：客户端也能排序
 *     sorterMeta: createSorterMeta<User>("username"),       // 主路径：触发服务端排序
 *   }
 *
 * 但通常只挂 sorterMeta 即可——若后端 ORDER BY 生效，前端客户端排序只是浪费。
 */
export interface SorterMeta<T = unknown> {
  field: string;
  type?: SorterType;
  // 泛型占位，仅用于在列定义处保持类型推导一致
  _recordType?: T;
}

export function createSorterMeta<T = unknown>(field: string, type?: SorterType): SorterMeta<T> {
  return { field, type };
}

/**
 * 通用 sorter 工厂
 *
 * 用法：
 *   sorter: createSorter("name", "string")
 *   sorter: createSorter("createdAt", "date")
 *   sorter: createSorter("status", "number")
 *   sorter: createSorter("isActive", "boolean")
 *   sorter: createSorter("priority", "custom", (a, b) => ...)
 *
 * 设计原则：
 * 1. 默认安全 — 字段缺失/为 null/undefined 时按 0/空串处理（不抛错）
 * 2. 多语言 — 字符串比较走 localeCompare(zh-Hans-CN)
 * 3. 时间 — 优先 dayjs(...).valueOf()，解析失败回退 NaN 排序末端
 */
export function createSorter<T = unknown>(
  field: string,
  type: SorterType = "string",
  compareFn?: (a: T, b: T) => number
) {
  return (a: T, b: T): number => {
    if (type === "custom") {
      if (!compareFn) return 0;
      return compareFn(a, b);
    }

    const av = (a as Record<string, unknown>)[field];
    const bv = (b as Record<string, unknown>)[field];

    // null/undefined 统一排到末端
    if (av == null && bv == null) return 0;
    if (av == null) return 1;
    if (bv == null) return -1;

    switch (type) {
      case "number": {
        const an = Number(av);
        const bn = Number(bv);
        if (Number.isNaN(an) && Number.isNaN(bn)) return 0;
        if (Number.isNaN(an)) return 1;
        if (Number.isNaN(bn)) return -1;
        return an - bn;
      }
      case "date": {
        const at = dayjs(av as dayjs.ConfigType);
        const bt = dayjs(bv as dayjs.ConfigType);
        if (!at.isValid() && !bt.isValid()) return 0;
        if (!at.isValid()) return 1;
        if (!bt.isValid()) return -1;
        return at.valueOf() - bt.valueOf();
      }
      case "boolean": {
        const ab = Boolean(av);
        const bb = Boolean(bv);
        if (ab === bb) return 0;
        return ab ? -1 : 1; // true 优先
      }
      case "string":
      default: {
        const as = String(av);
        const bs = String(bv);
        return as.localeCompare(bs, "zh-Hans-CN", { numeric: true, sensitivity: "base" });
      }
    }
  };
}

export interface ColumnConfig {
  // 展示态字段（与 antd Table 列对齐）
  title?: string;
  dataIndex?: string;
  key: string;
  width?: number;
  align?: "left" | "center" | "right";
  sorter?: ((a: any, b: any) => number) | boolean;
  // 受控排序方向（配合 useTableManager.getColumnSortOrder 实现"高亮落所点列"）。
  // 与 antd Table sorter.order 一致：ascend/descend/null（null=未排序）。
  // 内联 union 而非 import useServerSort.SortOrder，避免 utils↔hooks type 循环。
  sortOrder?: "ascend" | "descend" | null;

  // 配置态字段（ColumnConfigModal 使用,允许用户配置列的展示/隐藏/排序/分组）
  label?: string;
  visible?: boolean;
  order?: number;
  group?: string;
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
      <Tag color={status === 0 ? "success" : "error"}>{status === 0 ? "正常" : "停用"}</Tag>
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

export function createIndexColumn<T>(currentPage: number, pageSize: number): TableColumn<T> {
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
