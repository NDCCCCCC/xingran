/**
 * 楼宇选项 Hook
 */

import { useState, useCallback } from "react";
import { buildingApi } from "@/lib/opsApi";
import { handleApiError } from "@/utils/errorHandler";
import type { Building } from "@/types/operations";

export interface BuildingOption {
  id: string;
  code: string;
  name: string;
}

export function useBuildingOptions() {
  const [buildingOptions, setBuildingOptions] = useState<BuildingOption[]>([]);

  const loadBuildingOptions = useCallback(async () => {
    try {
      const result = await buildingApi.list({ current: 1, pageSize: 50 });
      const buildings = result.data?.list || [];

      setBuildingOptions(
        buildings.map((b: Building) => ({
          id: String(b.id ?? ""),
          code: String(b.code ?? ""),
          name: String(b.name ?? ""),
        }))
      );
    } catch (error) {
      console.error("[useBuildingOptions] 加载楼宇选项失败:", error);
      handleApiError(error, "加载楼宇选项", false);
    }
  }, []);

  return {
    buildingOptions,
    loadBuildingOptions,
  };
}
