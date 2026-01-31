/**
 * 楼层统计数据 Hook
 */

import { useState, useCallback } from 'react';
import { floorApi } from '@/lib/opsApi';
import { handleApiError } from '@/utils/errorHandler';

interface FloorStatistics {
  total: number;
  active: number;
  inactive: number;
}

export function useFloorStatistics() {
  const [statistics, setStatistics] = useState<FloorStatistics>({
    total: 0,
    active: 0,
    inactive: 0,
  });

  const loadStatistics = useCallback(async () => {
    try {
      const result = await floorApi.list({ current: 1, pageSize: 1000 });
      const allFloors = result.data?.list || [];

      setStatistics({
        total: allFloors.length,
        active: allFloors.filter((f) => f.status === 0).length,
        inactive: allFloors.filter((f) => f.status === 1).length,
      });
    } catch (error) {
      handleApiError(error, '加载统计数据', false);
    }
  }, []);

  return {
    statistics,
    loadStatistics,
  };
}
