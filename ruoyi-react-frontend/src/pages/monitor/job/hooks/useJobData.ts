/**
 * Job Data Hook
 * 定时任务数据管理 Hook
 */

import { useState, useCallback } from 'react';
import type { JobInfo, JobLog, SearchFormState } from '../types';
import { post } from '@/lib/api';
import type { BaseResponse, PageResponse } from '@/types';

export interface UseJobDataParams {
  searchForm: SearchFormState;
  current: number;
  pageSize: number;
}

export interface UseJobDataReturn {
  // 数据状态
  jobs: JobInfo[];
  jobLogs: JobLog[];
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
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);

  // 获取任务列表
  const fetchJobs = useCallback(async (overrideParams?: Partial<SearchFormState>) => {
    setLoading(true);
    try {
      // 使用 overrideParams（如果提供）或 searchForm
      const formData = overrideParams !== undefined ? { ...searchForm, ...overrideParams } : searchForm;

      // 清理 undefined 字段，避免发送到后端
      const cleanedParams = {
        ...(formData.jobName !== undefined && formData.jobName !== '' ? { jobName: formData.jobName } : {}),
        ...(formData.jobGroup !== undefined && formData.jobGroup !== '' ? { jobGroup: formData.jobGroup } : {}),
        ...(formData.status !== undefined ? { status: formData.status } : {}),
        current,
        pageSize
      };

      const result = await post<BaseResponse<PageResponse<JobInfo>>>('/monitor/jobs/list', cleanedParams);

      setJobs(result.data?.list || []);
      setTotal(result.data?.total || 0);
    } catch (error) {
      console.error('获取任务列表失败:', error);
    } finally {
      setLoading(false);
    }
  }, [searchForm, current, pageSize]);

  // 获取任务日志
  const fetchJobLogs = useCallback(async (jobName?: string, jobGroup?: string) => {
    try {
      const result = await post<BaseResponse<PageResponse<JobLog>>>('/monitor/jobs/logs/list', {
        jobName: jobName || '',
        jobGroup: jobGroup || '',
        current: 1,
        pageSize: 50
      });

      setJobLogs(result.data?.list || []);
    } catch (error) {
      console.error('获取任务日志失败:', error);
    }
  }, []);

  return {
    jobs,
    jobLogs,
    loading,
    total,
    setJobs,
    setJobLogs,
    setTotal,
    fetchJobs,
    fetchJobLogs,
  };
}
