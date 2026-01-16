/**
 * Template Data Hook
 * 模板数据管理 Hook
 */

import { useState, useCallback } from 'react';
import { message } from 'antd';
import type { ConfigTemplate, BaseResponse, PageResponse } from '@/types';
import { post } from '@/lib/api';
import type { TemplateStatistics } from '../types';

export interface UseTemplateDataReturn {
  templates: ConfigTemplate[];
  loading: boolean;
  total: number;
  current: number;
  pageSize: number;
  statistics: TemplateStatistics;

  setTotal: React.Dispatch<React.SetStateAction<number>>;
  setCurrent: React.Dispatch<React.SetStateAction<number>>;
  setPageSize: React.Dispatch<React.SetStateAction<number>>;
  setStatistics: React.Dispatch<React.SetStateAction<TemplateStatistics>>;

  loadTemplates: (params?: any, searchValues?: any) => Promise<void>;
  loadStatistics: () => Promise<void>;
  updateStatistics: (templateList: ConfigTemplate[]) => void;
  handleApiError: (error: unknown, defaultMessage: string) => void;
}

export function useTemplateData(searchForm: any): UseTemplateDataReturn {
  const [templates, setTemplates] = useState<ConfigTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [statistics, setStatistics] = useState<TemplateStatistics>({
    total: 0,
    system: 0,
    custom: 0,
    init: 0,
  });

  const handleApiError = useCallback((error: unknown, defaultMessage: string) => {
    if (error && typeof error === 'object' && 'message' in error) {
      message.error(error.message as string);
    } else {
      message.error(defaultMessage);
    }
  }, []);

  const loadTemplates = useCallback(
    async (params: any = {}, searchValues?: any) => {
      setLoading(true);
      try {
        const values = searchValues || searchForm.getFieldsValue();
        const result = await post<BaseResponse<PageResponse<ConfigTemplate>>>('/network/templates/list', {
          current: params.current || current,
          pageSize: params.pageSize || pageSize,
          ...values,
        });
        setTemplates(result.data?.list || []);
        setTotal(result.data?.total || 0);
        updateStatistics(result.data?.list || []);
      } catch (error) {
        console.error('加载模板列表失败:', error);
      } finally {
        setLoading(false);
      }
    },
    [current, pageSize, searchForm]
  );

  const loadStatistics = useCallback(async () => {
    try {
      const result = await post<BaseResponse<PageResponse<ConfigTemplate>>>('/network/templates/list', { current: 1, pageSize: 10000 });
      const allTemplates = result.data?.list || [];
      updateStatistics(allTemplates);
    } catch (error) {
      console.error('加载统计数据失败:', error);
    }
  }, []);

  const updateStatistics = useCallback((templateList: ConfigTemplate[]) => {
    const systemCount = templateList.filter((t) => t.isSystem).length;
    const customCount = templateList.filter((t) => !t.isSystem).length;
    const initCount = templateList.filter((t) => t.templateType === 'init').length;
    setStatistics({
      total: templateList.length,
      system: systemCount,
      custom: customCount,
      init: initCount,
    });
  }, []);

  return {
    templates,
    loading,
    total,
    current,
    pageSize,
    statistics,
    setTotal,
    setCurrent,
    setPageSize,
    setStatistics,
    loadTemplates,
    loadStatistics,
    updateStatistics,
    handleApiError,
  };
}
