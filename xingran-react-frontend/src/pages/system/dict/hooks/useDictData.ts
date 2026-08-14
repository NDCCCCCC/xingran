/**
 * Dict Data Hook
 * 字典数据管理 Hook
 */

import { useState, useCallback } from "react";
import { App } from "antd";
import type { FormInstance } from "antd/es/form";
import type { DictType, DictData } from "@/types";
import { post } from "@/lib/api";
import { handleApiError } from "@/utils/errorHandler";

export interface DictStatistics {
  total: number;
  active: number;
  inactive: number;
}

export interface UseDictDataReturn {
  // 数据状态
  dictTypes: DictType[];
  dictDataList: DictData[];
  loading: boolean;
  total: number;
  selectedType: string;
  typeStatistics: DictStatistics;
  dataStatistics: DictStatistics;

  // 数据操作方法
  setSelectedType: (type: string) => void;
  loadDictTypes: (params?: Record<string, unknown>) => Promise<void>;
  loadDictData: (params?: Record<string, unknown>) => Promise<void>;
  loadTypeStatistics: () => Promise<void>;
  loadDataStatistics: () => Promise<void>;
}

export function useDictData(searchForm: FormInstance<unknown>, dataSearchForm: FormInstance<unknown>, current: number, pageSize: number): UseDictDataReturn {
  const { message } = App.useApp();
  const [dictTypes, setDictTypes] = useState<DictType[]>([]);
  const [dictDataList, setDictDataList] = useState<DictData[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [selectedType, setSelectedType] = useState<string>("");
  const [typeStatistics, setTypeStatistics] = useState<DictStatistics>({
    total: 0,
    active: 0,
    inactive: 0,
  });
  const [dataStatistics, setDataStatistics] = useState<DictStatistics>({
    total: 0,
    active: 0,
    inactive: 0,
  });

  // 加载字典类型列表
  // params 支持透传排序等服务端参数（current/pageSize 已单独处理；其余 key 如 orderByColumn/isAsc 原样展开）
  const loadDictTypes = useCallback(async (params: Record<string, unknown> = {}) => {
    setLoading(true);
    try {
      const values = searchForm.getFieldsValue() as Record<string, unknown>;
      const { current: _c, pageSize: _p, ...restParams } = params;
      const result = await post("/system/dicts/types/list", {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
        ...(values as object),
        ...(restParams as object),
      }) as { data: { list: DictType[]; total: number } };
      setDictTypes(result.data.list);
      setTotal(result.data.total);
    } catch (error) {
      console.error("加载字典类型失败:", error);
    } finally {
      setLoading(false);
    }
  }, [current, pageSize, searchForm]);

  // 加载字典数据列表
  // params 支持透传排序等服务端参数（current/pageSize 已单独处理；其余 key 如 orderByColumn/isAsc 原样展开）
  const loadDictData = useCallback(async (params: Record<string, unknown> = {}) => {
    if (!selectedType) {
      message.warning("请先选择字典类型");
      return;
    }
    setLoading(true);
    try {
      const values = dataSearchForm.getFieldsValue() as Record<string, unknown>;
      const { current: _c, pageSize: _p, ...restParams } = params;
      const result = await post("/system/dicts/data/list", {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
        dictType: selectedType,
        ...(values as object),
        ...(restParams as object),
      }) as { data: { list: DictData[]; total: number } };
      setDictDataList(result.data.list);
      setTotal(result.data.total);
    } catch (error) {
      console.error("加载字典数据失败:", error);
    } finally {
      setLoading(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedType, current, pageSize, dataSearchForm]);

  // 加载字典类型统计数据(专用 COUNT 端点,不受 MaxPageSize=100 钳制)
  const loadTypeStatistics = useCallback(async () => {
    try {
      const result = await post<{ total: number; active: number; inactive: number }>("/system/dicts/types/statistics");
      setTypeStatistics(result.data ?? { total: 0, active: 0, inactive: 0 });
    } catch (error) {
      handleApiError(error, "加载统计数据", false);
    }
  }, []);

  // 加载字典数据统计数据(专用 COUNT 端点,按 selectedType 过滤,不受 MaxPageSize=100 钳制)
  const loadDataStatistics = useCallback(async () => {
    if (!selectedType) {
      setDataStatistics({ total: 0, active: 0, inactive: 0 });
      return;
    }
    try {
      const result = await post<{ total: number; active: number; inactive: number }>(
        "/system/dicts/data/statistics",
        { dictType: selectedType }
      );
      setDataStatistics(result.data ?? { total: 0, active: 0, inactive: 0 });
    } catch (error) {
      handleApiError(error, "加载统计数据", false);
    }
  }, [selectedType]);

  return {
    dictTypes,
    dictDataList,
    loading,
    total,
    selectedType,
    typeStatistics,
    dataStatistics,
    setSelectedType,
    loadDictTypes,
    loadDictData,
    loadTypeStatistics,
    loadDataStatistics,
  };
}
