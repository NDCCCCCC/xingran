/**
 * Log Data Hook
 * 日志数据管理 Hook
 */

import { useState, useCallback } from "react";
import { App } from "antd";
import type { OperLog, LoginLog, SearchFormState } from "../types";
import { post } from "@/lib/api";
import type { PageResponse } from "@/types";

export interface UseLogDataParams {
  activeTab: string;
  searchForm: SearchFormState;
  loginSearchForm: SearchFormState;
  current: number;
  pageSize: number;
}

/** 分页覆盖参数（可选，缺省用 Hook 入参） */
export interface LogFetchParams {
  current?: number;
  pageSize?: number;
}

export interface UseLogDataReturn {
  // 数据状态
  operLogs: OperLog[];
  loginLogs: LoginLog[];
  loading: boolean;
  total: number;
  /** FIX-04b: 列表加载失败的可重试错误 */
  error: unknown;

  // 数据操作方法
  setOperLogs: React.Dispatch<React.SetStateAction<OperLog[]>>;
  setLoginLogs: React.Dispatch<React.SetStateAction<LoginLog[]>>;
  setTotal: React.Dispatch<React.SetStateAction<number>>;
  fetchOperLogs: (params?: LogFetchParams) => Promise<void>;
  fetchLoginLogs: (params?: LogFetchParams) => Promise<void>;
}

export function useLogData(params: UseLogDataParams): UseLogDataReturn {
  const { searchForm, loginSearchForm, current, pageSize } = params;
  const { message: _message } = App.useApp();

  const [operLogs, setOperLogs] = useState<OperLog[]>([]);
  const [loginLogs, setLoginLogs] = useState<LoginLog[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  // FIX-04b: 列表加载失败的可重试错误
  const [error, setError] = useState<unknown>(null);

  // 获取操作日志
  const fetchOperLogs = useCallback(
    async (params: LogFetchParams = {}) => {
      setLoading(true);
      try {
        const requestParams: Record<string, unknown> = {
          ...searchForm,
          current: params.current || current,
          pageSize: params.pageSize || pageSize,
        };

        // 处理时间范围
        if (searchForm.timeRange && searchForm.timeRange.length === 2) {
          requestParams.startTime = searchForm.timeRange[0].toISOString();
          requestParams.endTime = searchForm.timeRange[1].toISOString();
        }

        const result = await post<PageResponse<OperLog>>("/monitor/oper-logs/list", requestParams);

        setOperLogs(result.data?.list || []);
        setTotal(result.data?.total || 0);
        setError(null);
      } catch (error) {
        setError(error);
      } finally {
        setLoading(false);
      }
    },
    [searchForm, current, pageSize]
  );

  // 获取登录日志
  const fetchLoginLogs = useCallback(
    async (params: LogFetchParams = {}) => {
      setLoading(true);
      try {
        const requestParams: Record<string, unknown> = {
          ...loginSearchForm,
          current: params.current || current,
          pageSize: params.pageSize || pageSize,
        };

        // 处理时间范围
        if (loginSearchForm.timeRange && loginSearchForm.timeRange.length === 2) {
          requestParams.startTime = loginSearchForm.timeRange[0].toISOString();
          requestParams.endTime = loginSearchForm.timeRange[1].toISOString();
        }

        const result = await post<PageResponse<LoginLog>>(
          "/monitor/login-logs/list",
          requestParams
        );

        setLoginLogs(result.data?.list || []);
        setTotal(result.data?.total || 0);
        setError(null);
      } catch (error) {
        setError(error);
      } finally {
        setLoading(false);
      }
    },
    [loginSearchForm, current, pageSize]
  );

  return {
    operLogs,
    loginLogs,
    loading,
    total,
    error,
    setOperLogs,
    setLoginLogs,
    setTotal,
    fetchOperLogs,
    fetchLoginLogs,
  };
}
