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

export interface UseLogDataReturn {
  // 数据状态
  operLogs: OperLog[];
  loginLogs: LoginLog[];
  loading: boolean;
  total: number;

  // 数据操作方法
  setOperLogs: React.Dispatch<React.SetStateAction<OperLog[]>>;
  setLoginLogs: React.Dispatch<React.SetStateAction<LoginLog[]>>;
  setTotal: React.Dispatch<React.SetStateAction<number>>;
  fetchOperLogs: (params?: any) => Promise<void>;
  fetchLoginLogs: (params?: any) => Promise<void>;
}

export function useLogData(params: UseLogDataParams): UseLogDataReturn {
  const { searchForm, loginSearchForm, current, pageSize } = params;
  const { message } = App.useApp();

  const [operLogs, setOperLogs] = useState<OperLog[]>([]);
  const [loginLogs, setLoginLogs] = useState<LoginLog[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);

  // 获取操作日志
  const fetchOperLogs = useCallback(
    async (params: any = {}) => {
      setLoading(true);
      try {
        const requestParams: any = {
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
      } catch (error) {
        console.error("获取操作日志失败:", error);
        message.error("网络错误，请稍后重试");
      } finally {
        setLoading(false);
      }
    },
    [searchForm, current, pageSize, message]
  );

  // 获取登录日志
  const fetchLoginLogs = useCallback(
    async (params: any = {}) => {
      setLoading(true);
      try {
        const requestParams: any = {
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
      } catch (error) {
        console.error("获取登录日志失败:", error);
        message.error("网络错误，请稍后重试");
      } finally {
        setLoading(false);
      }
    },
    [loginSearchForm, current, pageSize, message]
  );

  return {
    operLogs,
    loginLogs,
    loading,
    total,
    setOperLogs,
    setLoginLogs,
    setTotal,
    fetchOperLogs,
    fetchLoginLogs,
  };
}
