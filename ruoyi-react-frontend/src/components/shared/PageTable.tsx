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
 *   rowActions={(r) => [editAction(r), createDeleteAction(r.id, handleDelete, { target: r.postName })]}
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
import type { FormInstance } from "antd";
import { ReloadOutlined } from "@ant-design/icons";
import type { TableProps } from "antd";
import type { ColumnsType, GetRowKey } from "antd/es/table/interface";
import type { Key } from "react";
import { useTableManager } from "@/hooks/useTableManager";
import type { SorterMeta } from "@/utils/tableHelpers";
import ActionButtons from "./ActionButtons";
import type { ActionButton } from "./ActionButtons";
import { BatchDeleteButton } from "./BatchDeleteButton";
import { SearchForm } from "./SearchForm";
import type { SearchField } from "./SearchForm";
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
  /** 搜索区 schema（§2）；与 searchChildren 二选一，同时传时 schema 优先并 console.warn 一次 */
  searchFields?: SearchField[];
  /** schema 覆盖不了的复杂搜索区逃生口：ReactNode 或 (searchForm: FormInstance) => ReactNode */
  searchChildren?: React.ReactNode | ((searchForm: FormInstance) => React.ReactNode);
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
  /**
   * 渲染上下文暴露（UI-SPEC §1 决策 2「searchForm 等经 render-props 暴露」）：
   * 函数形态 (ctx) => ReactNode，渲染在组件最上方（PageTable 之前的业务区，
   * 如统计卡片 / 已选择 Alert）；ctx 提供 selectedRowKeys / clearSelection /
   * total / loadData / searchForm。
   */
  children?: (ctx: PageTableRenderContext<T>) => React.ReactNode;
  /** 透传 antd Table 其余 props（scroll/expandable/rowSelection 等），全透传不吞不改名 */
  [key: string]: unknown;
}

/** children render-prop 收到的上下文（业务区接线用：统计卡片/已选择 Alert/提交后刷新） */
export interface PageTableRenderContext<T> {
  /** 当前选中行 key（batchDelete 内建 rowSelection 同源） */
  selectedRowKeys: Key[];
  /** 清空选中（「取消选择」入口） */
  clearSelection: () => void;
  /** 当前列表总数 */
  total: number;
  /** 手动刷新列表（提交/删除成功后回调） */
  loadData: () => void;
  /** 内部 useTableManager 的 searchForm（searchChildren 逃生口同源） */
  searchForm: FormInstance;
  /** 类型占位（TS 泛型推断辅助） */
  __type?: T;
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
  searchFields,
  searchChildren,
  toolbar,
  rowActions,
  batchDelete,
  sorterMetas,
  defaultSort,
  extraFilters,
  emptyAction,
  children,
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
    searchForm,
    loadData,
    applyFilters,
    handleSearch,
    handleReset,
    resetSelection,
    handleRefresh,
  } = manager;

  // 分页 showTotal：63 §1 逐字口径；total=0 → 空串（不渲染文案）
  const showTotal = useMemo(
    () => (t: number) => (t === 0 ? "" : `共 ${t} 条 / ${Math.ceil(t / effectivePageSize)} 页`),
    [effectivePageSize]
  );

  // 首次加载 + extraFilters 联动：mount 时（带 extra）首载；之后仅当 extraFilters 引用变化
  // 才 applyFilters 回第 1 页重载（对齐 applyFilters 语义，避免每渲染都触发）。
  // FAIL-1 修复：mount 首载走 keepPage——保留 sessionStorage 恢复的页码，
  // 不得硬编码回第 1 页（tab 切换 remount 后恢复语义优先）；extraFilters 引用变化才回第 1 页。
  const applyFiltersRef = useRef(applyFilters);
  // latest-ref 同步收敛进 effect（react-hooks/refs：禁止渲染期写 ref；
  // 声明在下方的 extraFilters effect 之前，保证其执行时已拿到最新 applyFilters）
  useEffect(() => {
    applyFiltersRef.current = applyFilters;
  }, [applyFilters]);
  const isFirstMountRef = useRef(true);
  useEffect(() => {
    if (isFirstMountRef.current) {
      isFirstMountRef.current = false;
      if (extraFilters !== undefined) {
        void applyFiltersRef.current(extraFilters, { keepPage: true });
      } else {
        void applyFiltersRef.current(undefined, { keepPage: true });
      }
      return;
    }
    if (extraFilters !== undefined) {
      void applyFiltersRef.current(extraFilters);
    } else {
      void applyFiltersRef.current();
    }
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

  // searchFields 与 searchChildren 二选一：同时传时 schema 优先并警告一次
  // （ref 读写收敛进 effect，react-hooks/refs：禁止渲染期访问 ref）
  const warnedRef = useRef(false);
  useEffect(() => {
    if (searchFields !== undefined && searchChildren !== undefined && !warnedRef.current) {
      warnedRef.current = true;
      console.warn(
        "[PageTable] searchFields 与 searchChildren 同时传入，schema 优先，searchChildren 将被忽略。"
      );
    }
  }, [searchFields, searchChildren]);
  const showSearchCard = searchFields !== undefined || searchChildren !== undefined;
  const effectiveChildren =
    searchFields !== undefined
      ? undefined
      : typeof searchChildren === "function"
        ? searchChildren(searchForm)
        : searchChildren;

  return (
    <div>
      {typeof children === "function" &&
        children({
          selectedRowKeys,
          clearSelection: resetSelection,
          total,
          loadData: () => void loadData(),
          searchForm,
        })}
      {showSearchCard && (
        <SearchForm
          form={searchForm}
          fields={searchFields}
          children={effectiveChildren}
          onSearch={handleSearch}
          onReset={handleReset}
          onRefresh={handleRefresh}
        />
      )}
      {(toolbar !== undefined || batchDelete !== undefined) && (
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16 }}>
          <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
            {toolbar}
            {batchDelete && (
              <BatchDeleteButton
                selectedCount={selectedRowKeys.length}
                onConfirm={handleBatchDeleteConfirm}
                target={batchDelete.target}
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
