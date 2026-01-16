/**
 * 备份数据管理 Hook
 */

import { useState, useCallback } from 'react';
import type { ConfigBackup, BaseResponse, PageResponse } from '@/types';
import type { DeviceBackupGroup, BackupStatistics } from '../types';
import { groupBackupsByDevice, calculateStatistics } from '../utils';
import { post } from '@/lib/api';

interface UseBackupDataOptions {
  current: number;
  pageSize: number;
  searchForm: any;
}

interface UseBackupDataReturn {
  devices: ConfigBackup[];
  deviceGroups: DeviceBackupGroup[];
  statistics: BackupStatistics;
  loading: boolean;
  total: number;
  loadDevices: () => Promise<void>;
  loadBackups: (params?: any) => Promise<void>;
  loadStatistics: () => Promise<void>;
}

export function useBackupData(
  options: UseBackupDataOptions
): UseBackupDataReturn {
  const { current, pageSize, searchForm } = options;

  const [devices, setDevices] = useState<ConfigBackup[]>([]);
  const [deviceGroups, setDeviceGroups] = useState<DeviceBackupGroup[]>([]);
  const [statistics, setStatistics] = useState<BackupStatistics>({
    total: 0,
    auto: 0,
    manual: 0,
    devices: 0,
  });
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);

  // 加载设备列表
  const loadDevices = useCallback(async () => {
    try {
      const result = await post<BaseResponse<PageResponse<ConfigBackup>>>('/network/devices/list', {
        current: 1,
        pageSize: 1000,
      });
      setDevices(result.data?.list || []);
    } catch (error) {
      console.error('加载设备列表失败:', error);
    }
  }, []);

  // 加载备份列表
  const loadBackups = useCallback(async (params: any = {}) => {
    setLoading(true);
    try {
      const values = searchForm.getFieldsValue();
      const result = await post<BaseResponse<PageResponse<any>>>('/network/backups/list', {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
        ...values,
      });
      const backups = result.data?.list || [];
      const groups = groupBackupsByDevice(backups);

      setDeviceGroups(groups);
      setTotal(result.data?.total || 0);

      // 更新统计数据
      const stats = calculateStatistics(backups);
      setStatistics(stats);
    } catch (error) {
      console.error('加载备份列表失败:', error);
    } finally {
      setLoading(false);
    }
  }, [current, pageSize, searchForm]);

  // 加载统计数据
  const loadStatistics = useCallback(async () => {
    try {
      const result = await post<BaseResponse<PageResponse<any>>>('/network/backups/list', { current: 1, pageSize: 10000 });
      const allBackups = result.data?.list || [];
      const stats = calculateStatistics(allBackups);
      setStatistics(stats);
    } catch (error) {
      console.error('加载统计数据失败:', error);
    }
  }, []);

  return {
    devices,
    deviceGroups,
    statistics,
    loading,
    total,
    loadDevices,
    loadBackups,
    loadStatistics,
  };
}
