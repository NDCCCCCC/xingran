/**
 * Device Discovery Data Hook
 * 设备发现数据管理 Hook
 */

import { useState, useCallback } from "react";
import { App } from "antd";
import type { DeviceDiscovery, Department, PageResponse } from "@/types";
import { post } from "@/lib/api";
import type { DiscoveryStatistics, ModalState } from "../types";

export interface UseDiscoveryDataParams {
  current: number;
  pageSize: number;
}

export interface UseDiscoveryDataReturn {
  discoveries: DeviceDiscovery[];
  discoveredDevices: Record<string, unknown>[];
  departments: Department[];
  loading: boolean;
  total: number;
  statistics: DiscoveryStatistics;
  modalState: ModalState;
  currentDiscovery: DeviceDiscovery | null;

  setDiscoveries: React.Dispatch<React.SetStateAction<DeviceDiscovery[]>>;
  setDiscoveredDevices: React.Dispatch<React.SetStateAction<Record<string, unknown>[]>>;
  setDepartments: React.Dispatch<React.SetStateAction<Department[]>>;
  setModalState: React.Dispatch<React.SetStateAction<ModalState>>;
  setCurrentDiscovery: React.Dispatch<React.SetStateAction<DeviceDiscovery | null>>;

  loadDiscoveries: (params?: Record<string, unknown>) => Promise<void>;
  loadStatistics: () => Promise<void>;
  loadDepartments: () => Promise<void>;
  loadDiscoveryResults: (id: string) => Promise<void>;
}

export function useDiscoveryData(params: UseDiscoveryDataParams): UseDiscoveryDataReturn {
  const { message } = App.useApp();
  const { current, pageSize } = params;

  const [discoveries, setDiscoveries] = useState<DeviceDiscovery[]>([]);
  const [discoveredDevices, setDiscoveredDevices] = useState<Record<string, unknown>[]>([]);
  const [departments, setDepartments] = useState<Department[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [statistics, setStatistics] = useState<DiscoveryStatistics>({
    total: 0,
    pending: 0,
    running: 0,
    completed: 0,
    failed: 0,
    totalDevices: 0,
  });
  const [modalState, setModalState] = useState<ModalState>({
    modalVisible: false,
    resultModalVisible: false,
  });
  const [currentDiscovery, setCurrentDiscovery] = useState<DeviceDiscovery | null>(null);

  const loadDiscoveries = useCallback(async (params: Record<string, unknown> = {}) => {
    setLoading(true);
    try {
      // 直接透传所有 params(含 orderByColumn/isAsc)给后端,后端白名单过滤非法字段
      const result = await post<PageResponse<DeviceDiscovery>>("/network/discoveries/list", {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
        ...params,
      });
      setDiscoveries(result.data?.list || []);
      setTotal(result.data?.total || 0);
    } catch (error) {
      console.error("加载发现任务失败:", error);
    } finally {
      setLoading(false);
    }
  }, [current, pageSize]);

  // 加载统计数据(专用端点 COUNT 聚合,不受分页影响)
  const loadStatistics = useCallback(async () => {
    try {
      const result = await post<DiscoveryStatistics>("/network/discoveries/statistics");
      setStatistics({
        total: result.data?.total ?? 0,
        pending: result.data?.pending ?? 0,
        running: result.data?.running ?? 0,
        completed: result.data?.completed ?? 0,
        failed: result.data?.failed ?? 0,
        totalDevices: result.data?.totalDevices ?? 0,
      });
    } catch (error) {
      console.error("加载统计数据失败:", error);
    }
  }, []);

  const loadDepartments = useCallback(async () => {
    try {
      const result = await post<PageResponse<Department>>("/system/departments/list", { current: 1, pageSize: 50 });
      setDepartments(result.data?.list || []);
    } catch (error) {
      console.error("加载部门列表失败:", error);
    }
  }, []);

  const loadDiscoveryResults = useCallback(async (id: string) => {
    try {
      const result = await post<{ devices: Record<string, unknown>[] }>(`/network/discoveries/${id}/results`, {});
      setDiscoveredDevices(result.data?.devices || []);
    } catch (_error) {
      message.error("获取结果失败");
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
  }, []);

  return {
    discoveries,
    discoveredDevices,
    departments,
    loading,
    total,
    statistics,
    modalState,
    currentDiscovery,
    setDiscoveries,
    setDiscoveredDevices,
    setDepartments,
    setModalState,
    setCurrentDiscovery,
    loadDiscoveries,
    loadStatistics,
    loadDepartments,
    loadDiscoveryResults,
  };
}
