/**
 * NetworkDevice 数据管理 Hook
 */

import { useState, useCallback } from "react";
import { post } from "@/lib/api";
import { handleApiError } from "@/utils/errorHandler";
import type { Department, AuthCredential, BaseResponse, PageResponse } from "@/types";

export interface DeviceStatistics {
  total: number;
  online: number;
  offline: number;
  unknown: number;
}

export interface UseDeviceDataReturn {
  departments: Department[];
  credentials: AuthCredential[];
  statistics: DeviceStatistics;
  loadDepartments: () => Promise<void>;
  loadCredentials: () => Promise<void>;
  loadStatistics: () => Promise<void>;
}

export function useDeviceData(): UseDeviceDataReturn {
  const [departments, setDepartments] = useState<Department[]>([]);
  const [credentials, setCredentials] = useState<AuthCredential[]>([]);
  const [statistics, setStatistics] = useState<DeviceStatistics>({
    total: 0,
    online: 0,
    offline: 0,
    unknown: 0,
  });

  // 加载统计数据(专用端点 COUNT 聚合,不受分页/筛选影响)
  const loadStatistics = useCallback(async () => {
    try {
      const result = await post<{ totalDevices?: number; onlineDevices?: number; offlineDevices?: number; unknownDevices?: number }>("/network/devices/statistics");
      const data = result.data || {};
      setStatistics({
        total: data.totalDevices ?? 0,
        online: data.onlineDevices ?? 0,
        offline: data.offlineDevices ?? 0,
        unknown: data.unknownDevices ?? 0,
      });
    } catch (error) {
      handleApiError(error, "加载统计数据", false);
    }
  }, []);

  // 加载部门列表（树形结构）
  const loadDepartments = useCallback(async () => {
    try {
      const result = await post<Department[]>("/system/departments/tree", {});
      setDepartments(result.data || []);
    } catch (error) {
      handleApiError(error, "加载部门列表", false);
      setDepartments([]);
    }
  }, []);

  // 加载授权凭证列表
  const loadCredentials = useCallback(async () => {
    try {
      const result = await post<PageResponse<AuthCredential>>("/network/credentials/list", { current: 1, pageSize: 1000 });
      setCredentials(result.data?.list || []);
    } catch (error) {
      handleApiError(error, "加载授权凭证列表", false);
    }
  }, []);

  return {
    departments,
    credentials,
    statistics,
    loadDepartments,
    loadCredentials,
    loadStatistics,
  };
}
