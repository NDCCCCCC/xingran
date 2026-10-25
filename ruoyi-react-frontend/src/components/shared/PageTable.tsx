/**
 * Phase 66 契约组件 — PageTable（列表页一站式骨架）
 *
 * 用法示例（文件头 10 行内，契约见 66-UI-SPEC §1）：
 * ```tsx
 * <PageTable
 *   request={(params) => postApi.list(params)}
 *   columns={columns}
 *   rowKey="id"
 *   searchFields={[{ name: "postCode", label: "岗位编码", type: "input" }]}
 *   rowActions={(r) => [editAction(r), createDeleteAction(r.id, handleDelete, { label: r.postName })]}
 *   batchDelete={{ onDelete: postApi.batchDelete, target: "岗位" }}
 * />
 * ```
 *
 * 注意：组件内部实例化 useTableManager（依赖 useLocation），因此只能在
 * Router 上下文中使用（路由页天然满足；单测请包 MemoryRouter）。
 * 新列表页唯一合法列表数据层入口（Phase 66 lint 增量门强制）。
 */
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Table } from "antd";
import type { TableProps } from "antd";
import type { ColumnsType, GetRowKey } from "antd/es/table/interface";
import { useTableManager } from "@/hooks/useTableManager";
import ErrorAlertWithRetry from "./ErrorAlertWithRetry";

/** 列表请求：入参含 current/pageSize/filters/排序；返回 { list, total } —— 即 useTableManager 的 loadFunction 签名 */
export type PageTableRequest<T> = (
  params: Record<string, unknown>
) => Promise<{ list: T[]; total: number }>;

export interface PageTableProps<T> {
  /** 列表请求函数（唯一数据入口）；失败由组件内部呈现错误态并支持重试 */
  request: PageTableRequest<T>;
  /** 表格列定义 */
  columns: ColumnsType<T>;
  /** 行 key（antd GetRowKey 或字段名） */
  rowKey: GetRowKey<T>;
  /** 初始 pageSize（默认 10） */
  pageSize?: number;
  /** 透传 antd Table 其余 props（scroll/expandable/rowSelection 等），全透传不吞不改名 */
  [key: string]: unknown;
}

/**
 * PageTable — 内建三态（loading=Table 蒙层 / 空态=antd 默认 / 错误态=ErrorAlertWithRetry）
 * 与分页 showTotal 契约（63 §1 口径：`共 {total} 条 / {pages} 页`；total=0 不渲染）。
 */
function PageTableImpl<T extends object>({
  request,
  columns,
  rowKey,
  pageSize,
  ...rest
}: PageTableProps<T>) {
  // 错误态升格：包装 request，失败记 error state（不改 useTableManager 内部兜底行为）
  const [loadError, setLoadError] = useState<unknown>(null);
  const wrappedRequest = useCallback(
    async (params: Record<string, unknown>) => {
      try {
        const result = await request(params);
        setLoadError(null);
        return result;
      } catch (error) {
        setLoadError(error);
        throw error;
      }
    },
    [request]
  );

  const manager = useTableManager<T>(wrappedRequest, { pageSize });
  const { loading, data, total, current, pageSize: effectivePageSize, loadData } = manager;

  // 分页 showTotal：63 §1 逐字口径；total=0 → 空串（不渲染文案）
  const showTotal = useMemo(
    () => (t: number) => (t === 0 ? "" : `共 ${t} 条 / ${Math.ceil(t / effectivePageSize)} 页`),
    [effectivePageSize]
  );

  const handleRetry = useCallback(() => {
    void loadData();
  }, [loadData]);

  // 首次加载：useTableManager 不在 mount 自动 loadData（存量页自行首调），由包装层统一发起
  const loadDataRef = useRef(loadData);
  loadDataRef.current = loadData;
  useEffect(() => {
    void loadDataRef.current();
  }, []);

  return (
    <div>
      {loadError !== null && <ErrorAlertWithRetry error={loadError as Error} onRetry={handleRetry} />}
      <Table<T>
        columns={columns}
        dataSource={data}
        loading={loading}
        rowKey={rowKey}
        pagination={{
          current,
          pageSize: effectivePageSize,
          total,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal,
        }}
        onChange={manager.handleTableChange}
        {...rest}
      />
    </div>
  );
}

export const PageTable = PageTableImpl;
export default PageTable;

// rest 透传类型辅助（antd TableProps 其余 props 由调用方以 JSX 形式传入）
export type PageTableTableProps<T> = TableProps<T>;
