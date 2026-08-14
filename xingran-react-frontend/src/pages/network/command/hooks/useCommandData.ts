/**
 * Network Command Data Hook
 * 网络命令数据管理 Hook
 */

import { useState, useCallback } from "react";
import type {
  ConfigExecution,
  ConfigExecutionDetail,
  NetworkDevice,
  BaseResponse,
  PageResponse,
} from "@/types";
import { post } from "@/lib/api";
import type { CommandStatistics } from "../types";

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

  loadExecutions: (params?: Record<string, unknown>) => Promise<void>;
  loadStatistics: () => Promise<void>;
  loadDevices: () => Promise<NetworkDevice[]>;
  loadExecutionDetails: (
    executionId: string
  ) => Promise<{ execution: ConfigExecution; details: ConfigExecutionDetail[] }>;
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

  const loadExecutions = useCallback(
    async (params: Record<string, unknown> = {}) => {
      setExecLoading(true);
      setExecLoadingState(true);
      try {
        const result = await post<PageResponse<ConfigExecution>>("/network/command/list", {
          ...params,
          current: params.current || current,
          pageSize: params.pageSize || pageSize,
        });
        setExecutions(result.data?.list || []);
        setExecTotal(result.data?.total || 0);
      } catch (error) {
        console.error("加载执行记录失败:", error);
      } finally {
        setExecLoading(false);
        setExecLoadingState(false);
      }
    },
    [current, pageSize, setExecLoading]
  );

  // 加载统计数据(专用端点 COUNT 聚合,不受分页影响)
  const loadStatistics = useCallback(async () => {
    try {
      const result = await post<CommandStatistics>("/network/command/statistics");
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

  const loadDevices = useCallback(async (): Promise<NetworkDevice[]> => {
    try {
      const result = await post<PageResponse<NetworkDevice>>("/network/devices/list", {
        current: 1,
        pageSize: 50,
        status: 0,
      });
      return result.data?.list || [];
    } catch (error) {
      console.error("加载设备列表失败:", error);
      return [];
    }
  }, []);

  const loadExecutionDetails = useCallback(
    async (
      executionId: string
    ): Promise<{ execution: ConfigExecution; details: ConfigExecutionDetail[] }> => {
      try {
        const result = await post<ConfigExecution>(`/network/command/${executionId}`, {});
        return {
          execution: result.data || ({} as ConfigExecution),
          details: result.data?.details || [],
        };
      } catch (error) {
        console.error("加载执行明细失败:", error);
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
