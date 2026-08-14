/**
 * TableWidget - 表格 Widget
 *
 * 以表格形式展示数据列表
 */

import { useMemo } from "react";
import { Table } from "antd";
import type { ColumnsType } from "antd/es/table";
import type { TableDisplayConfig, WidgetConfig } from "@/types/dashboard";
import { BaseWidget } from "../base/BaseWidget";
import { useWidgetData } from "@/hooks/useWidgetData";

interface TableWidgetProps {
  widget: WidgetConfig;
  display: TableDisplayConfig;
  onEdit?: () => void;
  onDelete?: () => void;
}

// 提取数组数据
function extractArray(val: unknown): Record<string, unknown>[] {
  if (Array.isArray(val)) return val as Record<string, unknown>[];
  if (val !== null && val !== undefined) return [val as Record<string, unknown>];
  return [];
}

export const TableWidget: React.FC<TableWidgetProps> = ({ widget, display, onEdit, onDelete }) => {
  // 使用useWidgetData直接获取数据
  const { data, loading, error, refresh } = useWidgetData(widget);
  // 构建表格列配置
  // eslint-disable-next-line react-hooks/preserve-manual-memoization
  const columns = useMemo<ColumnsType<Record<string, unknown>>>(() => {
    // 如果配置了列，使用配置的列
    if (display.columns && display.columns.length > 0) {
      return display.columns.map((col) => ({
        title: col.title,
        dataIndex: col.dataIndex,
        width: col.width,
        align: col.align,
        render: col.render ? renderCustom(col.render) : undefined,
      }));
    }

    // 如果没有配置列，从数据中自动推断列
    // 确保data存在且是对象
    if (!data || typeof data !== "object") {
      return [];
    }

    const d = data as Record<string, unknown>;
    const items = extractArray(d.list ?? d.data ?? d.items ?? []);
    if (items.length > 0) {
      const firstItem = items[0] as Record<string, unknown>;
      return Object.keys(firstItem).map((key) => ({
        title: key,
        dataIndex: key,
        key,
        width: 150,
      }));
    }

    // 返回空数组
    return [];
  }, [display.columns, data]);

  // 提取表格数据
  const tableData = useMemo(() => {
    if (!data || typeof data !== "object") return [];
    const d = data as Record<string, unknown>;
    return extractArray(d.list ?? d.data ?? d.items ?? []);
  }, [data]);

  // 自定义渲染函数
  function renderCustom(renderer: string) {
    // 简单的自定义渲染实现
    return (value: unknown) => {
      if (renderer === "status") {
        return <span>{value as string}</span>;
      }
      if (renderer === "date") {
        return new Date(value as string).toLocaleString();
      }
      return value as string;
    };
  }

  return (
    <BaseWidget
      widget={widget}
      data={data}
      loading={loading}
      error={error}
      onEdit={onEdit}
      onDelete={onDelete}
      onRefresh={refresh}
    >
      <Table
        columns={columns}
        dataSource={tableData}
        pagination={
          display.pagination?.enabled
            ? {
                pageSize: display.pagination.pageSize,
                size: "small",
              }
            : false
        }
        size={display.size ?? "middle"}
        bordered={display.bordered ?? false}
        scroll={{ y: 300 }}
        rowKey={(record) =>
          String((record as Record<string, unknown>).id ?? JSON.stringify(record))
        }
      />
    </BaseWidget>
  );
};
