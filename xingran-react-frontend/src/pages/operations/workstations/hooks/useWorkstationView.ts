/**
 * Workstation View Hook
 * 工位视图管理 Hook
 */

import { useState, useCallback, useEffect } from "react";
import type { WorkstationNode } from "@/components/shared/FloorPlanEditor.types";
import { workstationApi } from "@/lib/opsApi";
import { handleApiError } from "@/utils/errorHandler";
import { toWorkstationNode } from "../constants";
import type { FloorOption, ViewMode } from "../types";

export interface UseWorkstationViewReturn {
  viewMode: ViewMode;
  setViewMode: React.Dispatch<React.SetStateAction<ViewMode>>;
  selectedFloorForPlan: string;
  setSelectedFloorForPlan: React.Dispatch<React.SetStateAction<string>>;
  floorPlanWorkstations: WorkstationNode[];

  handleFloorChangeForPlan: (floorId: string) => void;
  handlePositionUpdate: (
    items: { id: string; positionX: number; positionY: number; rotation?: number }[]
  ) => Promise<void>;
  handleFloorPlanEdit: (workstation: WorkstationNode, openModal: () => void) => void;
}

export function useWorkstationView(floorOptions: FloorOption[]): UseWorkstationViewReturn {
  const [viewMode, setViewMode] = useState<ViewMode>("table");
  const [selectedFloorForPlan, setSelectedFloorForPlan] = useState<string>("");
  const [floorPlanWorkstations, setFloorPlanWorkstations] = useState<WorkstationNode[]>([]);

  const loadFloorPlanWorkstations = useCallback(async (floorCode: string) => {
    if (!floorCode) {
      setFloorPlanWorkstations([]);
      return;
    }
    try {
      const result = await workstationApi.list({ floorCode, current: 1, pageSize: 1000 });
      const list = result.data?.list || [];
      // 转换为 WorkstationNode 格式
      setFloorPlanWorkstations(list.map(toWorkstationNode));
    } catch (error) {
      handleApiError(error, "加载平面图数据", false);
      setFloorPlanWorkstations([]);
    }
  }, []);

  const handlePositionUpdate = useCallback(
    async (items: { id: string; positionX: number; positionY: number; rotation?: number }[]) => {
      try {
        await workstationApi.updatePositions(items);

        // 更新本地平面图数据
        setFloorPlanWorkstations((prev) =>
          prev.map((ws) => {
            const updatedItem = items.find((item) => item.id === ws.id);
            if (updatedItem) {
              return {
                ...ws,
                x: updatedItem.positionX,
                y: updatedItem.positionY,
                ...(updatedItem.rotation !== undefined && { rotation: updatedItem.rotation }),
              };
            }
            return ws;
          })
        );
      } catch (error) {
        handleApiError(error, "更新位置");
        throw error;
      }
    },
    []
  );

  const handleFloorChangeForPlan = useCallback(
    (floorId: string) => {
      setSelectedFloorForPlan(floorId);
      loadFloorPlanWorkstations(floorId);
    },
    [loadFloorPlanWorkstations]
  );

  const handleFloorPlanEdit = useCallback(
    (workstation: WorkstationNode, openModal: (record?: WorkstationNode) => void) => {
      // 传入完整的工位数据，确保表单能正确填充
      openModal(workstation);
    },
    []
  );

  // 当切换到平面图视图时，默认加载第一个楼层的工位
  useEffect(() => {
    if (viewMode === "floorplan" && !selectedFloorForPlan && floorOptions.length > 0) {
      const firstFloor = floorOptions[0].code;
      // 使用 setTimeout 避免同步 setState
      setTimeout(() => {
        setSelectedFloorForPlan(firstFloor);
        loadFloorPlanWorkstations(firstFloor);
      }, 0);
    }
  }, [viewMode, selectedFloorForPlan, floorOptions, loadFloorPlanWorkstations]);

  return {
    viewMode,
    setViewMode,
    selectedFloorForPlan,
    setSelectedFloorForPlan,
    floorPlanWorkstations,
    handleFloorChangeForPlan,
    handlePositionUpdate,
    handleFloorPlanEdit,
  };
}
