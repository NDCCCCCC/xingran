/**
 * NetworkDevice 数据管理 Hook
 */

import { useState, useCallback } from 'react';
import { post } from '@/lib/api';
import { handleApiError } from '@/utils/errorHandler';
import type { NetworkDevice, Department, AuthCredential, BaseResponse, PageResponse } from '@/types';

export interface DeviceStatistics {
  total: number;
  online: number;
  offline: number;
  unknown: number;
}

export interface UseDeviceDataOptions {
  onUpdateStatistics?: (stats: DeviceStatistics) => void;
}

export interface UseDeviceDataReturn {
  departments: Department[];
  credentials: AuthCredential[];
  statistics: DeviceStatistics;
  loadDepartments: () => Promise<void>;
  loadCredentials: () => Promise<void>;
  loadStatistics: () => Promise<void>;
  updateStatistics: (deviceList: NetworkDevice[]) => void;
}

export function useDeviceData(
  options: UseDeviceDataOptions = {}
): UseDeviceDataReturn {
  const { onUpdateStatistics } = options;

  const [departments, setDepartments] = useState<Department[]>([]);
  const [credentials, setCredentials] = useState<AuthCredential[]>([]);
  const [statistics, setStatistics] = useState<DeviceStatistics>({
    total: 0,
    online: 0,
    offline: 0,
    unknown: 0,
  });

  // 更新统计数据
  const updateStatistics = useCallback((deviceList: NetworkDevice[]) => {
    const onlineCount = deviceList.filter(d => d.status === 0).length;
    const offlineCount = deviceList.filter(d => d.status === 1).length;
    const unknownCount = deviceList.filter(d => d.status === 2).length;
    const stats = {
      total: deviceList.length,
      online: onlineCount,
      offline: offlineCount,
      unknown: unknownCount,
    };
    setStatistics(stats);
    onUpdateStatistics?.(stats);
  }, [onUpdateStatistics]);

  // 加载统计数据
  const loadStatistics = useCallback(async () => {
    try {
      const result = await post<PageResponse<NetworkDevice>>('/network/devices/list', { current: 1, pageSize: 10000 });
      updateStatistics(result.data?.list || []);
    } catch (error) {
      handleApiError(error, '加载统计数据', false);
    }
  }, [updateStatistics]);

  // 加载部门列表（树形结构）
  const loadDepartments = useCallback(async () => {
    try {
      const result = await post<Department[]>('/system/departments/tree', {});
      setDepartments(result.data || []);
    } catch (error) {
      handleApiError(error, '加载部门列表', false);
      setDepartments([]);
    }
  }, []);

  // 加载授权凭证列表
  const loadCredentials = useCallback(async () => {
    try {
      const result = await post<PageResponse<AuthCredential>>('/network/credentials/list', { current: 1, pageSize: 1000 });
      setCredentials(result.data?.list || []);
    } catch (error) {
      handleApiError(error, '加载授权凭证列表', false);
    }
  }, []);

  return {
    departments,
    credentials,
    statistics,
    loadDepartments,
    loadCredentials,
    loadStatistics,
    updateStatistics,
  };
}
