/**
 * Dict Data Hook
 * 字典数据管理 Hook
 */

import { useState, useCallback } from 'react';
import { message } from 'antd';
import type { FormInstance } from 'antd/es/form';
import type { DictType, DictData } from '@/types';
import { post } from '@/lib/api';
import { handleApiError } from '@/utils/errorHandler';

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
  const [dictTypes, setDictTypes] = useState<DictType[]>([]);
  const [dictDataList, setDictDataList] = useState<DictData[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [selectedType, setSelectedType] = useState<string>('');
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
  const loadDictTypes = useCallback(async (params: Record<string, unknown> = {}) => {
    setLoading(true);
    try {
      const values = searchForm.getFieldsValue() as Record<string, unknown>;
      const result = await post('/system/dicts/types/list', {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
        ...(values as object),
      }) as { data: { list: DictType[]; total: number } };
      setDictTypes(result.data.list);
      setTotal(result.data.total);
    } catch (error) {
      console.error('加载字典类型失败:', error);
    } finally {
      setLoading(false);
    }
  }, [current, pageSize, searchForm]);

  // 加载字典数据列表
  const loadDictData = useCallback(async (params: Record<string, unknown> = {}) => {
    if (!selectedType) {
      message.warning('请先选择字典类型');
      return;
    }
    setLoading(true);
    try {
      const values = dataSearchForm.getFieldsValue() as Record<string, unknown>;
      const result = await post('/system/dicts/data/list', {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
        dictType: selectedType,
        ...(values as object),
      }) as { data: { list: DictData[]; total: number } };
      setDictDataList(result.data.list);
      setTotal(result.data.total);
    } catch (error) {
      console.error('加载字典数据失败:', error);
    } finally {
      setLoading(false);
    }
  }, [selectedType, current, pageSize, dataSearchForm]);

  // 加载字典类型统计数据
  const loadTypeStatistics = useCallback(async () => {
    try {
      const result = await post<{ data: { list: DictType[] } }>('/system/dicts/types/list', { current: 1, pageSize: 1000 });
      const allTypes = result.data.list || [];
      const activeCount = allTypes.filter((t) => t.status === 0).length;
      const inactiveCount = allTypes.filter((t) => t.status === 1).length;
      setTypeStatistics({
        total: allTypes.length,
        active: activeCount,
        inactive: inactiveCount,
      });
    } catch (error) {
      handleApiError(error, '加载统计数据', false);
    }
  }, []);

  // 加载字典数据统计数据
  const loadDataStatistics = useCallback(async () => {
    if (!selectedType) {
      setDataStatistics({ total: 0, active: 0, inactive: 0 });
      return;
    }
    try {
      const result = await post<{ data: { list: DictData[] } }>('/system/dicts/data/list', { current: 1, pageSize: 1000, dictType: selectedType });
      const allData = result.data.list || [];
      const activeCount = allData.filter((d) => d.status === 0).length;
      const inactiveCount = allData.filter((d) => d.status === 1).length;
      setDataStatistics({
        total: allData.length,
        active: activeCount,
        inactive: inactiveCount,
      });
    } catch (error) {
      handleApiError(error, '加载统计数据', false);
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
