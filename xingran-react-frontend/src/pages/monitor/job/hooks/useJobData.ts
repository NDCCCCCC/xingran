/**
 * Job Data Hook
 * 定时任务数据管理 Hook
 */

import { useState, useCallback } from "react";
import type { JobInfo, JobLog, SearchFormState } from "../types";
import { post } from "@/lib/api";
import { handleApiError } from "@/utils/errorHandler";
import type { PageResponse } from "@/types";

export interface UseJobDataParams {
  searchForm: SearchFormState;
  current: number;
  pageSize: number;
  /** 服务端排序（字段名对齐后端 jobAllowedSortFields；undefined 表示无排序） */
  orderByColumn?: string;
  isAsc?: boolean;
}

// 任务日志统计。status: 0=成功 1=失败。
export interface JobLogStats {
  total: number;
  success: number;
  fail: number;
}

export interface UseJobDataReturn {
  // 数据状态
  jobs: JobInfo[];
  jobLogs: JobLog[];
  jobLogStats: JobLogStats;
  loading: boolean;
  total: number;
  /** FIX-04b: 任务列表加载失败的可重试错误 */
  error: unknown;

  // 数据操作方法
  setJobs: React.Dispatch<React.SetStateAction<JobInfo[]>>;
  setJobLogs: React.Dispatch<React.SetStateAction<JobLog[]>>;
  setTotal: React.Dispatch<React.SetStateAction<number>>;
  fetchJobs: (
    overrideParams?: Partial<SearchFormState>,
    sort?: { orderByColumn?: string; isAsc?: boolean }
  ) => Promise<void>;
  fetchJobLogs: (
    jobName?: string,
    jobGroup?: string,
    sort?: { orderByColumn?: string; isAsc?: boolean }
  ) => Promise<void>;
}

export function useJobData(params: UseJobDataParams): UseJobDataReturn {
  const { searchForm, current, pageSize, orderByColumn, isAsc } = params;

  const [jobs, setJobs] = useState<JobInfo[]>([]);
  const [jobLogs, setJobLogs] = useState<JobLog[]>([]);
  const [jobLogStats, setJobLogStats] = useState<JobLogStats>({ total: 0, success: 0, fail: 0 });
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  // FIX-04b: 列表加载失败的可重试错误
  const [error, setError] = useState<unknown>(null);

  // 获取任务列表
  const fetchJobs = useCallback(
    async (
      overrideParams?: Partial<SearchFormState>,
      sort?: { orderByColumn?: string; isAsc?: boolean }
    ) => {
      setLoading(true);
      try {
        // 使用 overrideParams（如果提供）或 searchForm
        const formData =
          overrideParams !== undefined ? { ...searchForm, ...overrideParams } : searchForm;

        // 排序参数：事件回调内传入的 sort 优先（避免闭包旧值），否则用受控 state
        const sortParams = sort
          ? sort.orderByColumn
            ? { orderByColumn: sort.orderByColumn, isAsc: sort.isAsc }
            : {}
          : orderByColumn
            ? { orderByColumn, isAsc }
            : {};

        // 清理 undefined 字段，避免发送到后端
        const cleanedParams = {
          ...(formData.jobName !== undefined && formData.jobName !== ""
            ? { jobName: formData.jobName }
            : {}),
          ...(formData.jobGroup !== undefined && formData.jobGroup !== ""
            ? { jobGroup: formData.jobGroup }
            : {}),
          ...(formData.status !== undefined ? { status: formData.status } : {}),
          ...sortParams,
          current,
          pageSize,
        };

        const result = await post<PageResponse<JobInfo>>("/monitor/jobs/list", cleanedParams);

        setJobs(result.data?.list || []);
        setTotal(result.data?.total || 0);
        setError(null);
      } catch (error) {
        setError(error);
      } finally {
        setLoading(false);
      }
    },
    [searchForm, current, pageSize, orderByColumn, isAsc]
  );

  // 获取任务日志统计（专用端点 COUNT 聚合，全局计数，不受 pageSize:50 截断）。
  // 旧实现用 jobLogs.length / .filter().length 算「总/成功/失败次数」,执行 >50 次时卡在 50。
  const fetchJobStats = useCallback(async (jobName?: string, jobGroup?: string) => {
    try {
      const result = await post<JobLogStats>("/monitor/jobs/logs/statistics", {
        jobName: jobName || "",
        jobGroup: jobGroup || "",
      });
      setJobLogStats({
        total: result.data?.total ?? 0,
        success: result.data?.success ?? 0,
        fail: result.data?.fail ?? 0,
      });
    } catch (error) {
      handleApiError(error, "获取任务日志统计", false);
    }
  }, []);

  // 获取任务日志
  const fetchJobLogs = useCallback(
    async (
      jobName?: string,
      jobGroup?: string,
      sort?: { orderByColumn?: string; isAsc?: boolean }
    ) => {
      try {
        const result = await post<PageResponse<JobLog>>("/monitor/jobs/logs/list", {
          jobName: jobName || "",
          jobGroup: jobGroup || "",
          // FIX-03b: jobLog 服务端排序（startTime 已由调用方映射为 createdAt）
          ...(sort && sort.orderByColumn
            ? { orderByColumn: sort.orderByColumn, isAsc: sort.isAsc }
            : {}),
          current: 1,
          pageSize: 50,
        });

        setJobLogs(result.data?.list || []);
        // 顺带刷新统计（开抽屉 / 点刷新都覆盖）
        fetchJobStats(jobName, jobGroup);
      } catch (error) {
        handleApiError(error, "获取任务日志", false);
      }
    },
    [fetchJobStats]
  );

  return {
    jobs,
    jobLogs,
    jobLogStats,
    loading,
    total,
    error,
    setJobs,
    setJobLogs,
    setTotal,
    fetchJobs,
    fetchJobLogs,
  };
}
