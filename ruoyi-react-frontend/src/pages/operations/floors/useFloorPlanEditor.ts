/**
 * 楼层平面图编辑器 Hook
 */

import { useState, useCallback } from "react";
import type { Floor } from "@/types";
import { wallApi, doorApi, workstationApi, floorPlanTextApi } from "@/lib/opsApi";
import type {
  FloorPlanData,
  Wall,
  Door,
  TextElement,
  WorkstationNode,
} from "@/components/cad-editor/types";
import { App } from "antd";
import { processWorkstations, parseJsonField, stringifyJsonField, isNewElement } from "./utils";
import { DEFAULT_FLOOR_PLAN_CONFIG } from "./constants";

interface UseFloorPlanEditorOptions {
  onSaveStart?: () => void;
  onSaveEnd?: () => void;
}

interface UseFloorPlanEditorReturn {
  floorPlanData: FloorPlanData | null;
  floorPlanLoading: boolean;
  isEditMode: boolean;
  setEditMode: (edit: boolean) => void;
  loadFloorPlanData: (floorId: string) => Promise<void>;
  saveFloorPlan: (data: FloorPlanData, floor: Floor) => Promise<void>;
  resetFloorPlan: () => void;
}

export function useFloorPlanEditor(
  currentFloor: Floor | null,
  options?: UseFloorPlanEditorOptions
): UseFloorPlanEditorReturn {
  const { message } = App.useApp();
  const [floorPlanData, setFloorPlanData] = useState<FloorPlanData | null>(null);
  const [floorPlanLoading, setFloorPlanLoading] = useState(false);
  const [isEditMode, setIsEditMode] = useState(false);

  const setEditMode = useCallback((edit: boolean) => {
    setIsEditMode(edit);
  }, []);

  const resetFloorPlan = useCallback(() => {
    setFloorPlanData(null);
    setIsEditMode(false);
  }, []);

  const loadFloorPlanData = useCallback(
    async (floorId: string) => {
      setFloorPlanLoading(true);
      try {
        const [wallsResult, doorsResult, workstationsAll] = await Promise.all([
          wallApi.list({ floorId, current: 1, pageSize: 1000 }),
          doorApi.list({ floorId, current: 1, pageSize: 1000 }),
          // V130R-09 D-03-6: 楼层全集走专用端点,不再用 pageSize:1000 List 反模式
          workstationApi.getFloorWorkstationsAll(floorId),
        ]);

        const walls = wallsResult.data?.list || [];
        const doors = doorsResult.data?.list || [];
        const workstations = workstationsAll || [];

        let texts: unknown[] = [];
        try {
          const textsResult = await floorPlanTextApi.list({ floorId, current: 1, pageSize: 1000 });
          texts = textsResult.data?.list || [];
        } catch (textError) {
          console.warn("文本元素API不可用，可能需要重启后端服务:", textError);
        }

        const parsedWalls = walls.map((wall) => {
          return {
            ...wall,
            points: parseJsonField(wall.points),
          } as unknown as Wall;
        });

        const parsedDoors = doors.map((door) => {
          return {
            ...door,
            position: parseJsonField(door.position),
          } as unknown as Door;
        });

        const parsedTexts = texts.map((text) => {
          const t = text as Record<string, unknown>;
          return {
            ...t,
            position: parseJsonField(t.position),
          } as TextElement;
        });

        const processedWorkstations = processWorkstations(workstations);

        setFloorPlanData({
          floorId,
          floorName: currentFloor?.name || `${currentFloor?.floorNo}层`,
          width: DEFAULT_FLOOR_PLAN_CONFIG.CANVAS_WIDTH,
          height: DEFAULT_FLOOR_PLAN_CONFIG.CANVAS_HEIGHT,
          walls: parsedWalls,
          doors: parsedDoors,
          workstations: processedWorkstations,
          texts: parsedTexts,
          planImageId: currentFloor?.planImageId,
          planImageUrl: currentFloor?.planImageUrl,
          gridSize: DEFAULT_FLOOR_PLAN_CONFIG.GRID_SIZE,
          showGrid: true,
          snapToGrid: true,
        });
      } catch (error) {
        console.error("加载平面图失败:", error);
        message.error("加载平面图失败");
        setFloorPlanData({
          floorId,
          floorName: currentFloor?.name || `${currentFloor?.floorNo}层`,
          width: DEFAULT_FLOOR_PLAN_CONFIG.CANVAS_WIDTH,
          height: DEFAULT_FLOOR_PLAN_CONFIG.CANVAS_HEIGHT,
          walls: [],
          doors: [],
          workstations: [],
          texts: [],
          planImageId: currentFloor?.planImageId,
          planImageUrl: currentFloor?.planImageUrl,
          gridSize: DEFAULT_FLOOR_PLAN_CONFIG.GRID_SIZE,
          showGrid: true,
          snapToGrid: true,
        });
      } finally {
        setFloorPlanLoading(false);
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    [currentFloor]
  );

  // 分批并行保存（每批 10 个元素），收集失败项统一提示
  const BATCH_SIZE = 10;

  const saveWalls = useCallback(
    async (walls: Wall[], floorId: string) => {
      if (walls.length === 0) return;
      const batchSave = async (item: Wall) => {
        const isNew = isNewElement(item.id, "wall_");
        const { id: _wallId, ...wallWithoutId } = item;
        const wallData = {
          ...(isNew ? wallWithoutId : item),
          floorId,
          points: stringifyJsonField(item.points),
          type: item.type as "straight" | "curved" | "l_shaped" | "polyline",
        };
        if (isNew) {
          await wallApi.create(wallData);
        } else {
          await wallApi.update(item.id, wallData);
        }
      };

      const results = await Promise.allSettled(walls.map((wall) => batchSave(wall)));
      const failed = results.filter((r) => r.status === "rejected");
      if (failed.length > 0) {
        message.error(`墙壁保存失败 ${failed.length} 项`);
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    []
  );

  const saveDoors = useCallback(
    async (doors: Door[], floorId: string) => {
      if (doors.length === 0) return;
      const batchSave = async (item: Door) => {
        const isNew = isNewElement(item.id, "door_");
        const { id: _doorId, ...doorWithoutId } = item;
        const doorData = {
          ...(isNew ? doorWithoutId : item),
          floorId,
          position: stringifyJsonField(item.position),
        };
        if (isNew) {
          await doorApi.create(doorData);
        } else {
          await doorApi.update(item.id, doorData);
        }
      };

      const results = await Promise.allSettled(doors.map((door) => batchSave(door)));
      const failed = results.filter((r) => r.status === "rejected");
      if (failed.length > 0) {
        message.error(`门保存失败 ${failed.length} 项`);
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    []
  );

  const saveTexts = useCallback(
    async (texts: TextElement[], floorId: string) => {
      if (texts.length === 0) return;
      const batchSave = async (item: TextElement) => {
        const isNew = isNewElement(item.id, "text_");
        const { id: _textId, ...textWithoutId } = item;
        const textData = {
          ...(isNew ? textWithoutId : item),
          floorId,
          position: stringifyJsonField(item.position),
        };
        if (isNew) {
          await floorPlanTextApi.create(textData);
        } else {
          await floorPlanTextApi.update(item.id, textData);
        }
      };

      const results = await Promise.allSettled(texts.map((text) => batchSave(text)));
      const failed = results.filter((r) => r.status === "rejected");
      if (failed.length > 0) {
        message.error(`文本保存失败 ${failed.length} 项`);
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    []
  );

  const saveWorkstations = useCallback(async (workstations: WorkstationNode[]) => {
    const updates = workstations.map((ws) => ({
      id: ws.id,
      positionX: Math.round(ws.x),
      positionY: Math.round(ws.y),
    }));

    if (updates.length > 0) {
      await workstationApi.updatePositions(updates);
    }
  }, []);

  const saveFloorPlan = useCallback(
    async (data: FloorPlanData, floor: Floor) => {
      if (!floor) return;

      options?.onSaveStart?.();

      try {
        const floorId = floor.id;

        await Promise.all([
          saveWalls(data.walls, floorId),
          saveDoors(data.doors, floorId),
          saveTexts(data.texts || [], floorId),
        ]);

        await saveWorkstations(data.workstations);

        message.success("保存成功");
        await loadFloorPlanData(floorId);
      } catch (error) {
        console.error("保存失败:", error);
        message.error("保存失败");
      } finally {
        options?.onSaveEnd?.();
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    [loadFloorPlanData, saveWalls, saveDoors, saveTexts, saveWorkstations, options]
  );

  return {
    floorPlanData,
    floorPlanLoading,
    isEditMode,
    setEditMode,
    loadFloorPlanData,
    saveFloorPlan,
    resetFloorPlan,
  };
}
