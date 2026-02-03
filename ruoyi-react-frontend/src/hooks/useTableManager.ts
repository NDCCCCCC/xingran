import { useState, useCallback, useRef, useMemo } from 'react';
import { Form } from 'antd';

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
  searchForm: any;
  editForm: any;
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
  loadData: (params?: any) => Promise<void>;
  resetSelection: () => void;
}

/**
 * 统一表格管理Hook
 * 封装表格页面常见的状态和操作
 */
export function useTableManager<T>(
  loadFunction: (params: any) => Promise<{ list: T[]; total: number }>,
  options: TableManagerOptions = {}
): TableManagerReturn<T> {
  const { pageSize: defaultPageSize = 10, onSuccess, externalPagination } = options;

  // 表格状态
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<T[]>([]);
  const [total, setTotal] = useState(0);

  // 使用外部分页状态或内部状态
  const [internalCurrent, setInternalCurrent] = useState(1);
  const [internalPageSize, setInternalPageSize] = useState(defaultPageSize);

  const current = useMemo(
    () => externalPagination?.current ?? internalCurrent,
    [externalPagination?.current, internalCurrent]
  );

  const pageSize = useMemo(
    () => externalPagination?.pageSize ?? internalPageSize,
    [externalPagination?.pageSize, internalPageSize]
  );

  const setCurrent = useMemo(
    () => externalPagination?.setCurrent ?? setInternalCurrent,
    [externalPagination?.setCurrent]
  );

  const setPageSize = useMemo(
    () => externalPagination?.setPageSize ?? setInternalPageSize,
    [externalPagination?.setPageSize]
  );

  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // 表单状态
  const [searchForm] = Form.useForm();
  const [editForm] = Form.useForm();
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editingItem, setEditingItem] = useState<T | null>(null);

  // 使用 ref 存储 loadFunction 和 externalPagination，避免依赖导致的循环
  const loadFunctionRef = useRef(loadFunction);
  const externalPaginationRef = useRef(externalPagination);

  // 使用 ref 存储最新的函数引用，避免依赖导致的循环
  loadFunctionRef.current = loadFunction;
  externalPaginationRef.current = externalPagination;

  const loadData = useCallback(async (params: any = {}) => {
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
    } catch (error) {
      throw error;
    } finally {
      setLoading(false);
    }
  }, [current, pageSize, setData, setTotal, setLoading]);

  const handleSearch = useCallback(() => {
    searchForm.validateFields().then((values) => {
      setCurrent(1);
      loadData(values);
    });
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
  }, [editForm, setEditingItem, setEditModalVisible]);

  const handleEdit = useCallback((item: T) => {
    setEditingItem(item);
    setEditModalVisible(true);
  }, [setEditingItem, setEditModalVisible]);

  const handleModalClose = useCallback(() => {
    setEditModalVisible(false);
    setEditingItem(null);
    editForm.resetFields();
  }, [setEditModalVisible, setEditingItem, editForm]);

  const resetSelection = useCallback(() => {
    setSelectedRowKeys([]);
  }, [setSelectedRowKeys]);

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
