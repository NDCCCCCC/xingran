/**
 * 楼层统计数据 Hook
 */

import { useState, useCallback } from "react";
import { handleApiError } from "@/utils/errorHandler";
import { floorApi } from "@/lib/opsApi";

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
      const stats = await floorApi.statistics();
      setStatistics({ total: stats.total ?? 0, active: stats.active ?? 0, inactive: stats.inactive ?? 0 });
    } catch (error) {
      handleApiError(error, "加载统计数据", false);
    }
  }, []);

  return {
    statistics,
    loadStatistics,
  };
}
