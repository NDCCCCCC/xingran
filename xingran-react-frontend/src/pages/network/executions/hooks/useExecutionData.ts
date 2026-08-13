/**
 * Config Execution Data Hook
 * 配置执行数据管理 Hook
 */

import { useState, useCallback } from "react";
import { App } from "antd";
import type { ConfigExecution, NetworkDevice, ConfigTemplate, PageResponse } from "@/types";
import { post } from "@/lib/api";
import type { ExecutionStatistics, ExecutionDataState } from "../types";

export interface UseExecutionDataParams {
  current: number;
  pageSize: number;
  execCurrent: number;
}

export interface UseExecutionDataReturn {
  // 数据状态
  dataState: ExecutionDataState;
  loading: boolean;
  execLoading: boolean;
  total: number;
  execTotal: number;
  statistics: ExecutionStatistics;

  // 设置方法
  setDataState: React.Dispatch<React.SetStateAction<ExecutionDataState>>;
  setStatistics: React.Dispatch<React.SetStateAction<ExecutionStatistics>>;
  setTotal: React.Dispatch<React.SetStateAction<number>>;
  setExecTotal: React.Dispatch<React.SetStateAction<number>>;
  setLoading: React.Dispatch<React.SetStateAction<boolean>>;
  setExecLoading: React.Dispatch<React.SetStateAction<boolean>>;

  // 数据加载方法
  loadDevices: () => Promise<void>;
  loadTemplates: () => Promise<void>;
  loadExecutions: (params?: Record<string, unknown>) => Promise<void>;
  loadExecutionDetails: (executionId: string) => Promise<void>;
  loadStatistics: () => Promise<void>;
}

export function useExecutionData(params: UseExecutionDataParams): UseExecutionDataReturn {
  const { message } = App.useApp();
  const { current, pageSize, execCurrent } = params;

  const [dataState, setDataState] = useState<ExecutionDataState>({
    devices: [],
    templates: [],
    executions: [],
    executionDetails: [],
    currentExecution: null,
    selectedTemplate: null,
  });

  const [loading, setLoading] = useState(false);
  const [execLoading, setExecLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [execTotal, setExecTotal] = useState(0);

  const [statistics, setStatistics] = useState<ExecutionStatistics>({
    total: 0,
    pending: 0,
    running: 0,
    success: 0,
    failed: 0,
  });

  // 加载设备列表
  const loadDevices = useCallback(async () => {
    try {
      const result = await post<PageResponse<NetworkDevice>>("/network/devices/list", {
        current: 1,
        pageSize: 50,
        status: 0, // 只显示在线设备
      });
      setDataState(prev => ({ ...prev, devices: result.data?.list || [] }));
    } catch (error) {
      console.error("加载设备列表失败:", error);
    }
  }, []);

  // 加载模板列表
  const loadTemplates = useCallback(async () => {
    try {
      const result = await post<PageResponse<ConfigTemplate>>("/network/templates/list", {
        current: 1,
        pageSize: 50,
        templateType: "config",
      });
      setDataState(prev => ({ ...prev, templates: result.data?.list || [] }));
    } catch (error) {
      console.error("加载模板列表失败:", error);
    }
  }, []);

  // 加载执行记录列表
  const loadExecutions = useCallback(async (params: Record<string, unknown> = {}) => {
    setExecLoading(true);
    try {
      // 确保参数有有效值，避免 undefined 导致后端验证失败
      const requestCurrent = params.current ?? execCurrent ?? 1;
      const requestPageSize = params.pageSize ?? pageSize ?? 10;

      const result = await post<PageResponse<ConfigExecution>>("/network/executions/list", {
        ...params,
        current: requestCurrent,
        pageSize: requestPageSize,
      });
      setDataState(prev => ({ ...prev, executions: result.data?.list || [] }));
      setExecTotal(result.data?.total || 0);
    } catch (error) {
      console.error("加载执行记录失败:", error);
    } finally {
      setExecLoading(false);
    }
  }, [execCurrent, pageSize]);

  // 加载统计数据(专用端点 COUNT 聚合,不受分页影响)
  const loadStatistics = useCallback(async () => {
    try {
      const result = await post<ExecutionStatistics>("/network/executions/statistics");
      setStatistics({
        total: result.data?.total ?? 0,
        pending: result.data?.pending ?? 0,
        running: result.data?.running ?? 0,
        success: result.data?.success ?? 0,
        failed: result.data?.failed ?? 0,
      });
    } catch (error) {
      console.error("加载统计数据失败:", error);
    }
  }, []);

  // 加载执行明细
  const loadExecutionDetails = useCallback(async (executionId: string) => {
    try {
      const result = await post<ConfigExecution>(`/network/executions/${executionId}`, {});
      setDataState(prev => ({
        ...prev,
        currentExecution: result.data || null,
        executionDetails: result.data?.details || [],
      }));
    } catch (error) {
      message.error("加载执行明细失败");
    }
  }, []);

  return {
    dataState,
    loading,
    execLoading,
    total,
    execTotal,
    statistics,
    setDataState,
    setStatistics,
    setTotal,
    setExecTotal,
    setLoading,
    setExecLoading,
    loadDevices,
    loadTemplates,
    loadExecutions,
    loadExecutionDetails,
    loadStatistics,
  };
}
