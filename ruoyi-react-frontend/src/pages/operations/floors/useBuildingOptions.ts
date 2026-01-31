/**
 * 楼宇选项 Hook
 */

import { useState, useCallback } from 'react';
import { buildingApi } from '@/lib/opsApi';
import { handleApiError } from '@/utils/errorHandler';

export interface BuildingOption {
  id: string;
  code: string;
  name: string;
}

export function useBuildingOptions() {
  const [buildingOptions, setBuildingOptions] = useState<BuildingOption[]>([]);

  const loadBuildingOptions = useCallback(async () => {
    try {
      const result = await buildingApi.list({ current: 1, pageSize: 1000 });
      const buildings = result.data?.list || [];

      setBuildingOptions(
        buildings.map((b: any) => ({
          id: b.id,
          code: b.code,
          name: b.name,
        }))
      );
    } catch (error) {
      handleApiError(error, '加载楼宇选项', false);
    }
  }, []);

  return {
    buildingOptions,
    loadBuildingOptions,
  };
}
