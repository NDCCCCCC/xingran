/**
 * Network Command Data Hook
 * 网络命令数据管理 Hook
 */

import { useState, useCallback } from 'react';
import type { ConfigExecution, NetworkDevice, BaseResponse, PageResponse } from '@/types';
import { post } from '@/lib/api';
import type { CommandStatistics } from '../types';

export interface UseCommandDataParams {
  current: number;
  pageSize: number;
}

export interface UseCommandDataReturn {
  executions: ConfigExecution[];
  execLoading: boolean;
  execTotal: number;
  statistics: CommandStatistics;

  setExecTotal: React.Dispatch<React.SetStateAction<number>>;
  setStatistics: React.Dispatch<React.SetStateAction<CommandStatistics>>;
  setExecutions: React.Dispatch<React.SetStateAction<ConfigExecution[]>>;

  loadExecutions: (params?: any) => Promise<void>;
  loadStatistics: () => Promise<void>;
  loadDevices: () => Promise<NetworkDevice[]>;
  loadExecutionDetails: (executionId: string) => Promise<{ execution: ConfigExecution; details: any[] }>;
}

export function useCommandData(
  setExecLoading: React.Dispatch<React.SetStateAction<boolean>>,
  { current, pageSize }: UseCommandDataParams
): UseCommandDataReturn {
  const [executions, setExecutions] = useState<ConfigExecution[]>([]);
  const [execLoading, setExecLoadingState] = useState(false);
  const [execTotal, setExecTotal] = useState(0);
  const [statistics, setStatistics] = useState<CommandStatistics>({
    total: 0,
    pending: 0,
    running: 0,
    success: 0,
    failed: 0,
  });

  const updateStatistics = useCallback((executionList: ConfigExecution[]) => {
    const pendingCount = executionList.filter((e) => e.status === 'pending').length;
    const runningCount = executionList.filter((e) => e.status === 'running').length;
    const successCount = executionList.filter((e) => e.status === 'success').length;
    const failedCount = executionList.filter((e) => e.status === 'failed').length;
    setStatistics({
      total: executionList.length,
      pending: pendingCount,
      running: runningCount,
      success: successCount,
      failed: failedCount,
    });
  }, []);

  const loadExecutions = useCallback(
    async (params: any = {}) => {
      setExecLoading(true);
      setExecLoadingState(true);
      try {
        const result = await post<BaseResponse<PageResponse<ConfigExecution>>>('/network/command/list', {
          current: params.current || current,
          pageSize: params.pageSize || pageSize,
        });
        setExecutions(result.data?.list || []);
        setExecTotal(result.data?.total || 0);
        // 更新统计数据
        updateStatistics(result.data?.list || []);
      } catch (error) {
        console.error('加载执行记录失败:', error);
      } finally {
        setExecLoading(false);
        setExecLoadingState(false);
      }
    },
    [current, pageSize, setExecLoading, updateStatistics]
  );

  const loadStatistics = useCallback(async () => {
    try {
      const result = await post<BaseResponse<PageResponse<ConfigExecution>>>('/network/command/list', { current: 1, pageSize: 10000 });
      const allExecutions = result.data?.list || [];
      updateStatistics(allExecutions);
    } catch (error) {
      console.error('加载统计数据失败:', error);
    }
  }, [updateStatistics]);

  const loadDevices = useCallback(async (): Promise<NetworkDevice[]> => {
    try {
      const result = await post<BaseResponse<PageResponse<NetworkDevice>>>('/network/devices/list', {
        current: 1,
        pageSize: 1000,
        status: 0,
      });
      return result.data?.list || [];
    } catch (error) {
      console.error('加载设备列表失败:', error);
      return [];
    }
  }, []);

  const loadExecutionDetails = useCallback(
    async (executionId: string): Promise<{ execution: ConfigExecution; details: any[] }> => {
      try {
        const result = await post<BaseResponse<ConfigExecution & { details?: any[] }>>(`/network/command/${executionId}`, {});
        return {
          execution: result.data || {} as ConfigExecution,
          details: result.data?.details || [],
        };
      } catch (error) {
        console.error('加载执行明细失败:', error);
        throw error;
      }
    },
    []
  );

  return {
    executions,
    execLoading,
    execTotal,
    statistics,
    setExecTotal,
    setStatistics,
    setExecutions,
    loadExecutions,
    loadStatistics,
    loadDevices,
    loadExecutionDetails,
  };
}
