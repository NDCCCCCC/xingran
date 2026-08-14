/**
 * Dict Actions Hook
 * 字典操作管理 Hook
 */

import { useState, useCallback, type Key } from "react";
import { App, type FormInstance } from "antd";
import { useQueryClient } from "@tanstack/react-query";
import type { DictType, DictData } from "@/types";
import { post } from "@/lib/api";
import { handleApiError, handleSuccess } from "@/utils/errorHandler";
import { queryKeys } from "@/lib/queryKeys";

export interface UseDictActionsParams {
  selectedType: string;
  loadDictTypes: () => Promise<void>;
  loadDictData: () => Promise<void>;
  loadTypeStatistics: () => Promise<void>;
  loadDataStatistics: () => Promise<void>;
}

export interface UseDictActionsReturn {
  // 编辑状态
  editingType: DictType | null;
  editingData: DictData | null;
  typeModalVisible: boolean;
  dataModalVisible: boolean;

  // 状态设置
  setEditingType: (type: DictType | null) => void;
  setEditingData: (data: DictData | null) => void;
  setTypeModalVisible: (visible: boolean) => void;
  setDataModalVisible: (visible: boolean) => void;

  // 操作方法
  handleCreateType: (typeForm: FormInstance) => Promise<void>;
  handleDeleteType: (id: string) => Promise<void>;
  handleBatchDeleteType: (
    selectedRowKeys: Key[],
    setSelectedRowKeys: (keys: Key[]) => void
  ) => Promise<void>;
  handleCreateData: (dataForm: FormInstance) => Promise<void>;
  handleDeleteData: (id: string) => Promise<void>;
  handleBatchDeleteData: (
    selectedRowKeys: Key[],
    setSelectedRowKeys: (keys: Key[]) => void
  ) => Promise<void>;
  handleRefreshCache: () => Promise<void>;
  openTypeModal: (record: DictType | undefined, typeForm: FormInstance) => void;
  openDataModal: (record: DictData | undefined, dataForm: FormInstance) => void;
}

export function useDictActions(params: UseDictActionsParams): UseDictActionsReturn {
  const { message } = App.useApp();
  const { selectedType, loadDictTypes, loadDictData, loadTypeStatistics, loadDataStatistics } =
    params;

  const [editingType, setEditingType] = useState<DictType | null>(null);
  const [editingData, setEditingData] = useState<DictData | null>(null);
  const [typeModalVisible, setTypeModalVisible] = useState(false);
  const [dataModalVisible, setDataModalVisible] = useState(false);

  // 全局 dict 缓存失效器：每次字典变更后让所有 useDict(...) 消费者重新拉取 (D-11)
  const qc = useQueryClient();
  const invalidateAllDicts = useCallback(() => {
    qc.invalidateQueries({ queryKey: queryKeys.dict.all });
  }, [qc]);

  // 创建/更新字典类型
  const handleCreateType = useCallback(
    async (typeForm: FormInstance) => {
      try {
        const values = await typeForm.validateFields();
        if (editingType) {
          await post(`/system/dicts/types/${editingType.id}/update`, values);
          handleSuccess("更新");
        } else {
          await post("/system/dicts/types", values);
          handleSuccess("创建");
        }
        setTypeModalVisible(false);
        typeForm.resetFields();
        setEditingType(null);
        loadDictTypes();
        loadTypeStatistics();
        invalidateAllDicts();
      } catch (error: unknown) {
        if (error && typeof error === "object" && "errorFields" in error) {
          return;
        }
        handleApiError(error, "操作");
      }
    },
    [editingType, loadDictTypes, loadTypeStatistics, invalidateAllDicts]
  );

  // 删除字典类型
  const handleDeleteType = useCallback(
    async (id: string) => {
      try {
        await post(`/system/dicts/types/${id}/delete`, {});
        handleSuccess("删除");
        loadDictTypes();
        loadTypeStatistics();
        invalidateAllDicts();
      } catch (error) {
        handleApiError(error, "删除");
      }
    },
    [loadDictTypes, loadTypeStatistics, invalidateAllDicts]
  );

  // 批量删除字典类型
  const handleBatchDeleteType = useCallback(
    async (selectedRowKeys: Key[], setSelectedRowKeys: (keys: Key[]) => void) => {
      if (selectedRowKeys.length === 0) {
        message.warning("请选择要删除的数据");
        return;
      }
      try {
        await post("/system/dicts/types/batch-delete", { ids: selectedRowKeys });
        handleSuccess("批量删除");
        setSelectedRowKeys([]);
        loadDictTypes();
        loadTypeStatistics();
        invalidateAllDicts();
      } catch (error) {
        handleApiError(error, "批量删除");
      }
    },
    [loadDictTypes, loadTypeStatistics, invalidateAllDicts]
  );

  // 创建/更新字典数据
  const handleCreateData = useCallback(
    async (dataForm: FormInstance) => {
      try {
        const values = await dataForm.validateFields();
        if (editingData) {
          await post(`/system/dicts/data/${editingData.id}/update`, values);
          handleSuccess("更新");
        } else {
          await post("/system/dicts/data", { ...values, dictType: selectedType });
          handleSuccess("创建");
        }
        setDataModalVisible(false);
        dataForm.resetFields();
        setEditingData(null);
        loadDictData();
        loadDataStatistics();
        invalidateAllDicts();
      } catch (error: unknown) {
        if (error && typeof error === "object" && "errorFields" in error) {
          return;
        }
        handleApiError(error, "操作");
      }
    },
    [editingData, selectedType, loadDictData, loadDataStatistics, invalidateAllDicts]
  );

  // 删除字典数据
  const handleDeleteData = useCallback(
    async (id: string) => {
      try {
        await post(`/system/dicts/data/${id}/delete`, {});
        handleSuccess("删除");
        loadDictData();
        loadDataStatistics();
        invalidateAllDicts();
      } catch (error) {
        handleApiError(error, "删除");
      }
    },
    [loadDictData, loadDataStatistics, invalidateAllDicts]
  );

  // 批量删除字典数据
  const handleBatchDeleteData = useCallback(
    async (selectedRowKeys: Key[], setSelectedRowKeys: (keys: Key[]) => void) => {
      if (selectedRowKeys.length === 0) {
        message.warning("请选择要删除的数据");
        return;
      }
      try {
        await post("/system/dicts/data/batch-delete", { ids: selectedRowKeys });
        handleSuccess("批量删除");
        setSelectedRowKeys([]);
        loadDictData();
        loadDataStatistics();
        invalidateAllDicts();
      } catch (error) {
        handleApiError(error, "批量删除");
      }
    },
    [loadDictData, loadDataStatistics, invalidateAllDicts]
  );

  // 刷新缓存
  const handleRefreshCache = useCallback(async () => {
    try {
      await post("/system/dicts/refresh-cache", {});
      message.success("刷新缓存成功");
      invalidateAllDicts();
    } catch {
      message.error("刷新缓存失败");
    }
  }, [invalidateAllDicts]);

  // 打开类型编辑模态框
  const openTypeModal = useCallback((record: DictType | undefined, typeForm: FormInstance) => {
    if (record) {
      setEditingType(record);
      typeForm.setFieldsValue(record);
    } else {
      setEditingType(null);
      typeForm.resetFields();
      typeForm.setFieldsValue({ status: 0 });
    }
    setTypeModalVisible(true);
  }, []);

  // 打开数据编辑模态框
  const openDataModal = useCallback(
    (record: DictData | undefined, dataForm: FormInstance) => {
      if (!selectedType) {
        message.warning("请先选择字典类型");
        return;
      }
      if (record) {
        setEditingData(record);
        dataForm.setFieldsValue(record);
      } else {
        setEditingData(null);
        dataForm.resetFields();
        dataForm.setFieldsValue({ dictSort: 0, status: 0, isDefault: false });
      }
      setDataModalVisible(true);
    },
    [selectedType]
  );

  return {
    editingType,
    editingData,
    typeModalVisible,
    dataModalVisible,
    setEditingType,
    setEditingData,
    setTypeModalVisible,
    setDataModalVisible,
    handleCreateType,
    handleDeleteType,
    handleBatchDeleteType,
    handleCreateData,
    handleDeleteData,
    handleBatchDeleteData,
    handleRefreshCache,
    openTypeModal,
    openDataModal,
  };
}
