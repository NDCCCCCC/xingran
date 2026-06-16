import { useState, useCallback, useRef, useMemo } from "react";
import { Form } from "antd";
import type { FormInstance } from "antd/es/form";

export interface TableManagerOptions {
  pageSize?: number;
  onSuccess?: () => void;
  externalPagination?: ExternalPagination;
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
  handleReset: () => void;
  handleRefresh: () => void;
  handleAdd: () => void;
  handleEdit: (item: T) => void;
  handleModalClose: () => void;
  loadData: (params?: Record<string, unknown>) => Promise<void>;
  resetSelection: () => void;
}

/**
 * 统一表格管理Hook
 * 封装表格页面常见的状态和操作
 */
export function useTableManager<T>(
  loadFunction: (params: Record<string, unknown>) => Promise<{ list: T[]; total: number }>,
  options: TableManagerOptions = {}
): TableManagerReturn<T> {
  const { pageSize: defaultPageSize = 10, onSuccess, externalPagination } = options;

  // 表格状态
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<T[]>([]);
  const [total, setTotal] = useState(0);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // 内部分页状态
  const [internalCurrent, setInternalCurrent] = useState(1);
  const [internalPageSize, setInternalPageSize] = useState(defaultPageSize);

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

  // 使用 ref 存储可变引用，避免依赖循环
  const loadFunctionRef = useRef(loadFunction);
  const externalPaginationRef = useRef(externalPagination);
  loadFunctionRef.current = loadFunction;
  externalPaginationRef.current = externalPagination;

  const loadData = useCallback(async (params: Record<string, unknown> = {}) => {
    setLoading(true);
    try {
      const requestParams = {
        current: params.current ?? current,
        pageSize: params.pageSize ?? pageSize,
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
  }, [current, pageSize]);

  const handleSearch = useCallback(() => {
    // 使用 getFieldsValue 获取所有字段值（包括没有验证规则的字段）
    const values = searchForm.getFieldsValue();

    // 过滤掉空值、undefined、null
    const filteredValues = Object.keys(values).reduce((acc, key) => {
      const value = values[key];
      if (value !== undefined && value !== null && value !== "") {
        acc[key] = value;
      }
      return acc;
    }, {} as Record<string, unknown>);

    setCurrent(1);
    loadData(filteredValues);
  }, [searchForm, setCurrent, loadData]);

  const handleReset = useCallback(() => {
    searchForm.resetFields();
    setCurrent(1);
    loadData();
  }, [searchForm, setCurrent, loadData]);

  const handleRefresh = useCallback(() => {
    loadData();
    onSuccess?.();
  }, [loadData, onSuccess]);

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
    handleReset,
    handleRefresh,
    handleAdd,
    handleEdit,
    handleModalClose,
    loadData,
    resetSelection,
  };
}
