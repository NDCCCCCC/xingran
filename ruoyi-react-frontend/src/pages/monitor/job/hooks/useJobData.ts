/**
 * Job Data Hook
 * 定时任务数据管理 Hook
 */

import { useState, useCallback } from "react";
import type { JobInfo, JobLog, SearchFormState } from "../types";
import { post } from "@/lib/api";
import type { PageResponse } from "@/types";

export interface UseJobDataParams {
  searchForm: SearchFormState;
  current: number;
  pageSize: number;
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

  // 数据操作方法
  setJobs: React.Dispatch<React.SetStateAction<JobInfo[]>>;
  setJobLogs: React.Dispatch<React.SetStateAction<JobLog[]>>;
  setTotal: React.Dispatch<React.SetStateAction<number>>;
  fetchJobs: (overrideParams?: Partial<SearchFormState>) => Promise<void>;
  fetchJobLogs: (jobName?: string, jobGroup?: string) => Promise<void>;
}

export function useJobData(params: UseJobDataParams): UseJobDataReturn {
  const { searchForm, current, pageSize } = params;

  const [jobs, setJobs] = useState<JobInfo[]>([]);
  const [jobLogs, setJobLogs] = useState<JobLog[]>([]);
  const [jobLogStats, setJobLogStats] = useState<JobLogStats>({ total: 0, success: 0, fail: 0 });
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);

  // 获取任务列表
  const fetchJobs = useCallback(
    async (overrideParams?: Partial<SearchFormState>) => {
      setLoading(true);
      try {
        // 使用 overrideParams（如果提供）或 searchForm
        const formData =
          overrideParams !== undefined ? { ...searchForm, ...overrideParams } : searchForm;

        // 清理 undefined 字段，避免发送到后端
        const cleanedParams = {
          ...(formData.jobName !== undefined && formData.jobName !== ""
            ? { jobName: formData.jobName }
            : {}),
          ...(formData.jobGroup !== undefined && formData.jobGroup !== ""
            ? { jobGroup: formData.jobGroup }
            : {}),
          ...(formData.status !== undefined ? { status: formData.status } : {}),
          current,
          pageSize,
        };

        const result = await post<PageResponse<JobInfo>>("/monitor/jobs/list", cleanedParams);

        setJobs(result.data?.list || []);
        setTotal(result.data?.total || 0);
      } catch (error) {
        console.error("获取任务列表失败:", error);
      } finally {
        setLoading(false);
      }
    },
    [searchForm, current, pageSize]
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
      console.error("获取任务日志统计失败:", error);
    }
  }, []);

  // 获取任务日志
  const fetchJobLogs = useCallback(
    async (jobName?: string, jobGroup?: string) => {
      try {
        const result = await post<PageResponse<JobLog>>("/monitor/jobs/logs/list", {
          jobName: jobName || "",
          jobGroup: jobGroup || "",
          current: 1,
          pageSize: 50,
        });

        setJobLogs(result.data?.list || []);
        // 顺带刷新统计（开抽屉 / 点刷新都覆盖）
        fetchJobStats(jobName, jobGroup);
      } catch (error) {
        console.error("获取任务日志失败:", error);
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
    setJobs,
    setJobLogs,
    setTotal,
    fetchJobs,
    fetchJobLogs,
  };
}
