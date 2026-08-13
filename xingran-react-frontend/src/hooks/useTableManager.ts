import { useState, useCallback, useRef, useEffect } from "react";
import { Form } from "antd";
import { useLocation } from "react-router-dom";
import type { FormInstance } from "antd/es/form";
import type {
  TablePaginationConfig,
  FilterValue,
  SorterResult,
} from "antd/es/table/interface";
import { useServerSort, resolveSorter } from "@/hooks/useServerSort";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import { TABLE_STATE_PREFIX, sanitizePathForKey } from "@/constants/storage";
import type { SortOrder } from "@/hooks/useServerSort";
import type { SorterMeta } from "@/utils/tableHelpers";

export interface TableManagerOptions<T = unknown> {
  pageSize?: number;
  onSuccess?: () => void;
  externalPagination?: ExternalPagination;
  /** 可排序列元数据；传入后启用服务端排序，field 需对应后端白名单 key */
  sorterMetas?: Array<SorterMeta<T> | undefined>;
  /** 默认排序（首屏即按此排序） */
  defaultSort?: { orderByColumn: string; isAsc?: boolean };
}

export interface ExternalPagination {
  current: number;
  pageSize: number;
  setCurrent: (page: number) => void;
  setPageSize: (size: number) => void;
  setTotal: (total: number) => void;
}

export interface TableManagerReturn<T> {
  // 状态
  loading: boolean;
  data: T[];
  total: number;
  current: number;
  pageSize: number;
  selectedRowKeys: React.Key[];
  // 表单
  searchForm: FormInstance<unknown>;
  editForm: FormInstance<unknown>;
  editModalVisible: boolean;
  editingItem: T | null;
  // Setters
  setData: (data: T[]) => void;
  setTotal: (total: number) => void;
  setLoading: (loading: boolean) => void;
  setCurrent: (page: number) => void;
  setPageSize: (size: number) => void;
  setSelectedRowKeys: (keys: React.Key[]) => void;
  setEditModalVisible: (visible: boolean) => void;
  setEditingItem: (item: T | null) => void;
  // 操作方法
  handleSearch: () => void;
  /** 合并额外筛选条件(如外部选择的 deptId)到当前 filters，回到第 1 页并加载 */
  applyFilters: (extra?: Record<string, unknown>) => void;
  handleReset: () => void;
  handleRefresh: () => void;
  handleAdd: () => void;
  handleEdit: (item: T) => void;
  handleModalClose: () => void;
  loadData: (params?: Record<string, unknown>) => Promise<void>;
  resetSelection: () => void;
  // 服务端排序（sorterMetas 启用时生效，否则保持 undefined 不影响行为）
  orderByColumn: string | undefined;
  isAsc: boolean | undefined;
  sortOrder: SortOrder;
  /** 列级 sortOrder：只对当前排序列返回方向，其余 undefined（修"高亮恒落第一列"） */
  getColumnSortOrder: (field: string) => SortOrder | undefined;
  /** 统一 Table.onChange：分页 + 排序一起处理并自动 load，可直接喂 <Table onChange={}> */
  handleTableChange: (
    pagination: TablePaginationConfig,
    filters: Record<string, FilterValue | null>,
    sorter: SorterResult<T> | SorterResult<T>[]
  ) => void;
  resetSort: () => void;
}

/** 过滤掉 undefined / null / "" 的空值，避免污染请求参数 */
function filterEmpty(values?: Record<string, unknown>): Record<string, unknown> {
  if (!values) return {};
  return Object.keys(values).reduce((acc, key) => {
    const value = values[key];
    if (value !== undefined && value !== null && value !== "") {
      acc[key] = value;
    }
    return acc;
  }, {} as Record<string, unknown>);
}

/**
 * 读取 sessionStorage 中持久化的 filters 作为 filtersRef 初始值（mount 恢复用）。
 * 仅在 useRef 初始化时执行一次；损坏 / 非对象数据返回 {}。
 */
function readInitialFilters(storageKey: string): Record<string, unknown> {
  if (typeof window === "undefined") return {};
  try {
    const raw = window.sessionStorage.getItem(storageKey);
    if (!raw) return {};
    const parsed = JSON.parse(raw);
    return parsed && typeof parsed === "object" && !Array.isArray(parsed)
      ? (parsed as Record<string, unknown>)
      : {};
  } catch {
    return {};
  }
}

/**
 * 统一表格管理 Hook
 * 封装表格页面常见的状态和操作：加载/分页/搜索/排序/表单/弹窗/选中行。
 *
 * 服务端排序：传入 sorterMetas 即启用。内部复用 useServerSort + resolveSorter，
 * 通过 filtersRef / 排序 ref 持久化请求参数，分页/排序变化自动带上筛选条件
 * （修复旧实现"翻页丢失搜索条件"的隐性 bug）。
 *
 * 向后兼容：不传 sorterMetas 时排序相关字段为 undefined，现有页面行为不变。
 */
export function useTableManager<T>(
  loadFunction: (params: Record<string, unknown>) => Promise<{ list: T[]; total: number }>,
  options: TableManagerOptions<T> = {}
): TableManagerReturn<T> {
  const {
    pageSize: defaultPageSize = 10,
    onSuccess,
    externalPagination,
    sorterMetas,
    defaultSort,
  } = options;
  const location = useLocation();

  // 表格状态（运行时，不持久化）
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<T[]>([]);
  const [total, setTotal] = useState(0);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // 内部分页状态（持久化；slot 与 usePagination 一致，同页互洽）
  const [internalCurrent, setInternalCurrent] = usePersistedStateController<number>({
    keyPrefix: location.pathname,
    keySuffix: "current",
    defaultValue: 1,
  });
  const [internalPageSize, setInternalPageSize] = usePersistedStateController<number>({
    keyPrefix: location.pathname,
    keySuffix: "pageSize",
    defaultValue: defaultPageSize,
  });

  // 分页状态（优先使用外部）
  const current = externalPagination?.current ?? internalCurrent;
  const pageSize = externalPagination?.pageSize ?? internalPageSize;
  const setCurrent = externalPagination?.setCurrent ?? setInternalCurrent;
  const setPageSize = externalPagination?.setPageSize ?? setInternalPageSize;

  // 表单状态
  const [searchForm] = Form.useForm();
  const [editForm] = Form.useForm();
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editingItem, setEditingItem] = useState<T | null>(null);

  // 服务端排序
  const sort = useServerSort<T>({ sorterMetas, defaultSort });

  // filters 持久化：ref + sessionStorage mirror。
  // 保持 filtersRef 的同步可读性（loadData 空依赖读 ref），mount 时从 sessionStorage 恢复，
  // 写入时同步镜像；切 tab / 刷新可恢复筛选条件，关闭 tab / 登出由外部统一清理。
  const filtersStorageKey = `${TABLE_STATE_PREFIX}${sanitizePathForKey(location.pathname)}_filters`;
  const filtersRef = useRef<Record<string, unknown>>(readInitialFilters(filtersStorageKey));

  const persistFilters = useCallback((next: Record<string, unknown>) => {
    filtersRef.current = next;
    try {
      if (typeof window !== "undefined") {
        window.sessionStorage.setItem(filtersStorageKey, JSON.stringify(next));
      }
    } catch {
      // 隐私模式 / 配额溢出 — 静默
    }
  }, [filtersStorageKey]);

  const clearPersistedFilters = useCallback(() => {
    filtersRef.current = {};
    try {
      if (typeof window !== "undefined") {
        window.sessionStorage.removeItem(filtersStorageKey);
      }
    } catch {
      // ignore
    }
  }, [filtersStorageKey]);

  // ref 镜像层：loadData 等用空依赖 useCallback 读 ref，规避 stale 闭包（沿用既定模式）
  const loadFunctionRef = useRef(loadFunction);
  const externalPaginationRef = useRef(externalPagination);
  const onSuccessRef = useRef(onSuccess);
  const currentRef = useRef(current);
  const pageSizeRef = useRef(pageSize);
  const orderByColumnRef = useRef(sort.orderByColumn);
  const isAscRef = useRef(sort.isAsc);
  loadFunctionRef.current = loadFunction;
  externalPaginationRef.current = externalPagination;
  onSuccessRef.current = onSuccess;
  currentRef.current = current;
  pageSizeRef.current = pageSize;
  orderByColumnRef.current = sort.orderByColumn;
  isAscRef.current = sort.isAsc;

  // mount 时若存在恢复的 filters，回填到搜索表单，保持 UI 与查询参数一致
  useEffect(() => {
    const restored = filtersRef.current;
    if (restored && Object.keys(restored).length > 0) {
      searchForm.setFieldsValue(restored);
    }
    // 仅 mount 一次：searchForm 引用稳定
     
  }, [searchForm]);

  // loadData 读 ref 组装参数：current/pageSize + filters(持久化) + 排序。
  // params 作为 override 最后展开，可覆盖任意字段（分页/排序联动显式传新值）。
  const loadData = useCallback(async (params: Record<string, unknown> = {}) => {
    setLoading(true);
    try {
      const requestParams = {
        current: currentRef.current,
        pageSize: pageSizeRef.current,
        ...filtersRef.current,
        ...(orderByColumnRef.current
          ? { orderByColumn: orderByColumnRef.current, isAsc: isAscRef.current }
          : {}),
        ...params,
      };

      const result = await loadFunctionRef.current(requestParams);
      setData(result.list ?? []);
      const newTotal = result.total ?? 0;
      setTotal(newTotal);
      externalPaginationRef.current?.setTotal?.(newTotal);
    } finally {
      setLoading(false);
    }
  }, []);

  // applyFilters：读 searchForm + 合并 extra → filtersRef（持久化），回第 1 页并加载。
  // 供外部筛选联动（如部门树点击）显式调用；handleSearch 是它的无参便捷形式。
  const applyFilters = useCallback(
    (extra?: Record<string, unknown>) => {
      persistFilters({
        ...filterEmpty(searchForm.getFieldsValue() as Record<string, unknown>),
        ...filterEmpty(extra),
      });
      currentRef.current = 1;
      setCurrent(1);
      loadData({ current: 1 });
    },
    [searchForm, setCurrent, loadData, persistFilters]
  );

  const handleSearch = useCallback(() => {
    applyFilters();
  }, [applyFilters]);

  const handleReset = useCallback(() => {
    searchForm.resetFields();
    clearPersistedFilters();
    sort.resetSort();
    orderByColumnRef.current = undefined;
    isAscRef.current = undefined;
    currentRef.current = 1;
    setCurrent(1);
    loadData({ current: 1 });
  }, [searchForm, setCurrent, loadData, sort, clearPersistedFilters]);

  const handleRefresh = useCallback(() => {
    loadData();
    onSuccessRef.current?.();
  }, [loadData]);

  const handleAdd = useCallback(() => {
    setEditingItem(null);
    editForm.resetFields();
    setEditModalVisible(true);
  }, [editForm]);

  const handleEdit = useCallback((item: T) => {
    setEditingItem(item);
    editForm.setFieldsValue(item);
    setEditModalVisible(true);
  }, [editForm]);

  const handleModalClose = useCallback(() => {
    setEditModalVisible(false);
    setEditingItem(null);
    editForm.resetFields();
  }, [editForm]);

  const resetSelection = useCallback(() => {
    setSelectedRowKeys([]);
  }, []);

  // 统一 Table.onChange：分页 + 排序一起处理并自动 load。
  // resolveSorter 同步取排序新值（规避 setState 时序），写 ref 供 loadData 立即使用。
  const handleTableChange = useCallback(
    (
      pagination: TablePaginationConfig,
      _filters: Record<string, FilterValue | null>,
      sorter: SorterResult<T> | SorterResult<T>[]
    ) => {
      const newPage = pagination.current ?? 1;
      const newSize = pagination.pageSize ?? pageSizeRef.current;
      // 分页 UI（兼容 externalPagination）
      currentRef.current = newPage;
      pageSizeRef.current = newSize;
      setCurrent(newPage);
      setPageSize(newSize);
      // 排序受控 UI（更新 sortOrder/orderByColumn/isAsc state，驱动列高亮）
      sort.handleTableChange(pagination, _filters, sorter);
      // 同步取排序值写 ref（loadData 不依赖尚未提交的 setState）
      const { orderByColumn: newOrderBy, isAsc: newIsAsc } = resolveSorter(
        sorter,
        sorterMetas ?? []
      );
      orderByColumnRef.current = newOrderBy;
      isAscRef.current = newIsAsc;
      // 立即加载：带新分页 + 新排序 + 旧 filters（filtersRef 持久化）
      loadData({ current: newPage, pageSize: newSize });
    },
    [setCurrent, setPageSize, loadData, sorterMetas, sort]
  );

  const getColumnSortOrder = useCallback(
    (field: string): SortOrder | undefined => {
      if (orderByColumnRef.current !== String(field)) return undefined;
      return sort.sortOrder;
    },
    [sort.sortOrder]
  );

  return {
    // 状态
    loading,
    data,
    total,
    current,
    pageSize,
    selectedRowKeys,
    searchForm,
    editForm,
    editModalVisible,
    editingItem,
    // Setters
    setData,
    setTotal,
    setLoading,
    setCurrent,
    setPageSize,
    setSelectedRowKeys,
    setEditModalVisible,
    setEditingItem,
    // 方法
    handleSearch,
    applyFilters,
    handleReset,
    handleRefresh,
    handleAdd,
    handleEdit,
    handleModalClose,
    loadData,
    resetSelection,
    // 服务端排序
    orderByColumn: sort.orderByColumn,
    isAsc: sort.isAsc,
    sortOrder: sort.sortOrder,
    getColumnSortOrder,
    handleTableChange,
    resetSort: sort.resetSort,
  };
}
