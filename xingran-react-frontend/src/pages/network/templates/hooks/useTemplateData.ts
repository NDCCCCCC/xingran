/**
 * Template Data Hook
 * 模板数据管理 Hook
 */

import { useState, useCallback } from "react";
import { App } from "antd";
import type { ConfigTemplate, BaseResponse, PageResponse } from "@/types";
import type { FormInstance } from "antd/es/form";
import { post } from "@/lib/api";
import type { TemplateStatistics } from "../types";

export interface UseTemplateDataReturn {
  templates: ConfigTemplate[];
  loading: boolean;
  total: number;
  statistics: TemplateStatistics;

  loadTemplates: (params?: Record<string, unknown>, searchValues?: Record<string, unknown>) => Promise<void>;
  loadStatistics: () => Promise<void>;
  handleApiError: (error: unknown, defaultMessage: string) => void;
}

export function useTemplateData(searchForm: FormInstance<unknown>, setTotal: (total: number) => void): UseTemplateDataReturn {
  const { message } = App.useApp();
  const [templates, setTemplates] = useState<ConfigTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotalLocal] = useState(0);
  const [statistics, setStatistics] = useState<TemplateStatistics>({
    total: 0,
    system: 0,
    custom: 0,
    init: 0,
  });

  const handleApiError = useCallback((error: unknown, defaultMessage: string) => {
    if (error && typeof error === "object" && "message" in error) {
      message.error(error.message as string);
    } else {
      message.error(defaultMessage);
    }
  }, []);

  const loadTemplates = useCallback(
    async (params: Record<string, unknown> = {}, searchValues?: Record<string, unknown>) => {
      setLoading(true);
      try {
        const formValues = searchValues ?? (searchForm.getFieldsValue() as Record<string, unknown>);
        const result = await post<PageResponse<ConfigTemplate>>("/network/templates/list", {
          current: params.current || 1,
          pageSize: params.pageSize || 10,
          ...formValues,
        });
        setTemplates(result.data?.list || []);
        setTotalLocal(result.data?.total || 0);
        setTotal(result.data?.total || 0);
      } catch (error) {
        console.error("加载模板列表失败:", error);
      } finally {
        setLoading(false);
      }
    },
    [searchForm, setTotal]
  );

  // 加载统计数据(专用端点 COUNT 聚合,不受分页/筛选影响)。
  const loadStatistics = useCallback(async () => {
    try {
      const result = await post<TemplateStatistics>("/network/templates/statistics");
      setStatistics({
        total: result.data?.total ?? 0,
        system: result.data?.system ?? 0,
        custom: result.data?.custom ?? 0,
        init: result.data?.init ?? 0,
      });
    } catch (error) {
      console.error("加载统计数据失败:", error);
    }
  }, []);

  return {
    templates,
    loading,
    total: total,
    statistics,
    loadTemplates,
    loadStatistics,
    handleApiError,
  };
}
