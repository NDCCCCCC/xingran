/**
 * Phase 66 契约组件 — PageTable（列表页一站式骨架）
 *
 * 用法示例（契约见 66-UI-SPEC §1）：
 * ```tsx
 * <PageTable
 *   request={(params) => postApi.list(params)}
 *   columns={columns}
 *   rowKey="id"
 *   searchFields={[{ name: "postCode", label: "岗位编码", type: "input" }]}
 *   toolbar={<Button type="primary" onClick={openCreate}>新增</Button>}
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
import { Button, Table } from "antd";
import { ReloadOutlined } from "@ant-design/icons";
import type { TableProps } from "antd";
import type { ColumnsType, GetRowKey } from "antd/es/table/interface";
import type { Key } from "react";
import { useTableManager } from "@/hooks/useTableManager";
import type { SorterMeta } from "@/utils/tableHelpers";
import ActionButtons from "./ActionButtons";
import type { ActionButton } from "./ActionButtons";
import { BatchDeleteButton } from "./BatchDeleteButton";
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
  /** 工具栏左区（新增/导入等业务按钮）；右区内建刷新按钮（aria-label+title） */
  toolbar?: React.ReactNode;
  /** 行操作列：组件注入 record，返回 ActionButton[]（删除经 createDeleteAction → ConfirmDelete） */
  rowActions?: (record: T) => ActionButton[];
  /** 批量删除：传入即渲染 BatchDeleteButton（内部走 confirmDelete 模板，不自建确认）并内建 rowSelection */
  batchDelete?: { onDelete: (keys: Key[]) => Promise<void>; target: string };
  /** 服务端排序元数据（透传 useTableManager，64 FIX-03 口径；列 sorter:true 由页面 columns 自带） */
  sorterMetas?: Array<SorterMeta<T> | undefined>;
  /** 默认排序（首次加载生效） */
  defaultSort?: { orderByColumn: string; isAsc?: boolean };
  /** 额外固定筛选（如部门树联动）：引用变化 → applyFilters 回第 1 页重载 */
  extraFilters?: Record<string, unknown>;
  /** 空态引导动作（转发 EmptyStateWithAction；空态且无错误时替换 antd 默认空态） */
  emptyAction?: React.ReactNode;
  /** 透传 antd Table 其余 props（scroll/expandable/rowSelection 等），全透传不吞不改名 */
  [key: string]: unknown;
}

/**
 * PageTable — 内建三态（loading=Table 蒙层 / 空态=antd 默认或 EmptyStateWithAction /
 * 错误态=ErrorAlertWithRetry）与分页 showTotal 契约（63 §1 口径：
 * `共 {total} 条 / {pages} 页`；total=0 不渲染）。
 */
function PageTableImpl<T extends object>({
  request,
  columns,
  rowKey,
  pageSize,
  toolbar,
  rowActions,
  batchDelete,
  sorterMetas,
  defaultSort,
  extraFilters,
  emptyAction,
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

  const manager = useTableManager<T>(wrappedRequest, { pageSize, sorterMetas, defaultSort });
  const {
    loading,
    data,
    total,
    current,
    pageSize: effectivePageSize,
    selectedRowKeys,
    setSelectedRowKeys,
    loadData,
    applyFilters,
    resetSelection,
    handleRefresh,
  } = manager;

  // 分页 showTotal：63 §1 逐字口径；total=0 → 空串（不渲染文案）
  const showTotal = useMemo(
    () => (t: number) => (t === 0 ? "" : `共 ${t} 条 / ${Math.ceil(t / effectivePageSize)} 页`),
    [effectivePageSize]
  );

  // 首次加载 + extraFilters 联动：mount 时（带 extra）首载；之后仅当 extraFilters 引用变化
  // 才 applyFilters 回第 1 页重载（对齐 applyFilters 语义，避免每渲染都触发）
  const applyFiltersRef = useRef(applyFilters);
  applyFiltersRef.current = applyFilters;
  useEffect(() => {
    if (extraFilters !== undefined) {
      void applyFiltersRef.current(extraFilters);
    } else {
      void applyFiltersRef.current();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- 以 extraFilters 引用为唯一触发器
  }, [extraFilters]);

  // 行操作列：追加到 columns 末尾（经 ActionButtons 渲染，<3 平铺 / >=3 收下拉）
  const mergedColumns = useMemo<ColumnsType<T>>(() => {
    if (!rowActions) return columns;
    const actionColumn: ColumnsType<T>[number] = {
      title: "操作",
      key: "__pt_actions",
      width: 160,
      render: (_: unknown, record: T) => <ActionButtons actions={rowActions(record)} />,
    };
    return [...columns, actionColumn];
  }, [columns, rowActions]);

  // 批删：选中 >0 时渲染 BatchDeleteButton；确认成功 → resetSelection + 重新 loadData
  const handleBatchDeleteConfirm = useCallback(async () => {
    if (!batchDelete) return;
    await batchDelete.onDelete(selectedRowKeys);
    resetSelection();
    void loadData();
  }, [batchDelete, selectedRowKeys, resetSelection, loadData]);

  // rowSelection 内建接线（仅 batchDelete 场景）；用户显式传 rowSelection 时 rest 覆盖内建值
  const builtInRowSelection = useMemo(
    () =>
      batchDelete
        ? {
            selectedRowKeys,
            onChange: (keys: Key[]) => setSelectedRowKeys(keys),
          }
        : undefined,
    [batchDelete, selectedRowKeys, setSelectedRowKeys]
  );

  // 空态：无错误 + 无数据 + 非 loading 时用 emptyAction（若传）替换 antd 默认空态
  const showEmptyAction = emptyAction !== undefined && !loading && loadError === null && total === 0;

  return (
    <div>
      {(toolbar !== undefined || batchDelete !== undefined) && (
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16 }}>
          <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
            {toolbar}
            {batchDelete && (
              <BatchDeleteButton
                selectedCount={selectedRowKeys.length}
                onConfirm={handleBatchDeleteConfirm}
              />
            )}
          </div>
          <Button
            type="text"
            icon={<ReloadOutlined />}
            aria-label="刷新"
            title="刷新"
            onClick={handleRefresh}
          />
        </div>
      )}
      {loadError !== null && <ErrorAlertWithRetry error={loadError as Error} onRetry={() => void loadData()} />}
      <Table<T>
        columns={mergedColumns}
        dataSource={data}
        loading={loading}
        rowKey={rowKey}
        rowSelection={builtInRowSelection}
        locale={showEmptyAction ? { emptyText: emptyAction } : undefined}
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
