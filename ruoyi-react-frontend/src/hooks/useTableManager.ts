import { useState, useCallback, useRef } from 'react';
import { Form } from 'antd';

export interface TableManagerOptions {
  pageSize?: number;
  onSuccess?: () => void;
  // 支持外部传入的分页状态（用于全局分页管理）
  externalPagination?: {
    current: number;
    pageSize: number;
    setCurrent: (page: number) => void;
    setPageSize: (size: number) => void;
    setTotal: (total: number) => void;
  };
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
  // 方法
  setData: (data: T[]) => void;
  setTotal: (total: number) => void;
  setLoading: (loading: boolean) => void;
  setCurrent: (page: number) => void;
  setPageSize: (size: number) => void;
  setSelectedRowKeys: (keys: React.Key[]) => void;
  setEditModalVisible: (visible: boolean) => void;
  setEditingItem: (item: T | null) => void;
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

  const current = externalPagination?.current ?? internalCurrent;
  const pageSize = externalPagination?.pageSize ?? internalPageSize;
  const setCurrent = externalPagination?.setCurrent ?? setInternalCurrent;
  const setPageSize = externalPagination?.setPageSize ?? setInternalPageSize;

  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);

  // 表单状态
  const [searchForm] = Form.useForm();
  const [editForm] = Form.useForm();
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editingItem, setEditingItem] = useState<T | null>(null);

  // 使用 ref 存储 loadFunction 和 externalPagination，避免依赖导致的循环
  const loadFunctionRef = useRef(loadFunction);
  loadFunctionRef.current = loadFunction;

  const externalPaginationRef = useRef(externalPagination);
  externalPaginationRef.current = externalPagination;

  const loadData = useCallback(async (params: any = {}) => {
    setLoading(true);
    try {
      const result = await loadFunctionRef.current({
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
        ...params,
      });
      setData(result.list || []);
      const newTotal = result.total || 0;
      setTotal(newTotal);
      // 同步 total 到外部分页状态
      externalPaginationRef.current?.setTotal?.(newTotal);
      // eslint-disable-next-line no-useless-catch
    } catch (error) {
      // 错误由调用方处理
      throw error;
    } finally {
      setLoading(false);
    }
  }, [current, pageSize]);

  // 搜索
  const handleSearch = useCallback(() => {
    searchForm.validateFields().then((values) => {
      setCurrent(1);
      loadData(values);
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchForm, loadData]);

  // 重置
  const handleReset = useCallback(() => {
    searchForm.resetFields();
    setCurrent(1);
    loadData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchForm, loadData]);

  // 刷新
  const handleRefresh = useCallback(() => {
    loadData();
    onSuccess?.();
  }, [loadData, onSuccess]);

  // 新增
  const handleAdd = useCallback(() => {
    setEditingItem(null);
    editForm.resetFields();
    setEditModalVisible(true);
  }, [editForm]);

  // 编辑
  const handleEdit = useCallback((item: T) => {
    setEditingItem(item);
    setEditModalVisible(true);
  }, []);

  // 关闭编辑弹窗
  const handleModalClose = useCallback(() => {
    setEditModalVisible(false);
    setEditingItem(null);
    editForm.resetFields();
  }, [editForm]);

  // 重置选择
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
