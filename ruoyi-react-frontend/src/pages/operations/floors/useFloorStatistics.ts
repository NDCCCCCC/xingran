/**
 * 楼层统计数据 Hook
 */

import { useState, useCallback } from "react";
import { handleApiError } from "@/utils/errorHandler";
import { fetchFloorStatistics } from "@/utils/statisticsHelper";

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
      const stats = await fetchFloorStatistics();
      setStatistics(stats);
    } catch (error) {
      handleApiError(error, "加载统计数据", false);
    }
  }, []);

  return {
    statistics,
    loadStatistics,
  };
}
