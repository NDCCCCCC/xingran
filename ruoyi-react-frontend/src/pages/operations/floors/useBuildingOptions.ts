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
      console.log("[useBuildingOptions] 开始加载楼宇选项...");
      const result = await buildingApi.list({ current: 1, pageSize: 1000 });

      console.log("[useBuildingOptions] API响应:", {
        code: result.code,
        message: result.message,
        dataType: typeof result.data,
        hasData: !!result.data,
        listType: typeof result.data?.list,
        listLength: result.data?.list?.length,
        list: result.data?.list,
      });

      const buildings = result.data?.list || [];

      console.log("[useBuildingOptions] 解析后的楼层数据:", buildings);
      console.log("[useBuildingOptions] 设置楼宇选项:", buildings.map((b: Building) => ({
        id: String(b.id ?? ""),
        code: String(b.code ?? ""),
        name: String(b.name ?? ""),
      })));

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
