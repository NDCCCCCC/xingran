/**
 * 楼层平面图编辑器 Hook
 */

import { useState, useCallback, useRef } from "react";
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

/**
 * Phase 149 / SCOPE-03：保存侧「可见基线 ID 集」。
 *
 * loadFloorPlanData 拉取的是经后端按可见集裁剪后的整层几何（walls/doors/texts），
 * 这里把本次加载到的元素 ID 固化为基线。保存时以基线为准做增量 diff：
 * 只提交「可见元素的 create/update/delete」，绝不把「自己看不到的几何 ID」
 * 拼进任何请求集（后端 filterEntityVisibleIDs 是权威边界，前端是最小请求面）。
 */
interface VisibleBaseline {
  walls: Set<string>;
  doors: Set<string>;
  texts: Set<string>;
}

const emptyVisibleBaseline = (): VisibleBaseline => ({
  walls: new Set<string>(),
  doors: new Set<string>(),
  texts: new Set<string>(),
});

/** 从一组带 id 的元素派生可见基线 ID 集（跳过新建占位前缀由调用方保证）。 */
function idsOf<T extends { id: string }>(items: T[]): Set<string> {
  return new Set(items.map((item) => item.id));
}

/**
 * 计算保存增量 diff（Phase 149 定案：后端裁剪为权威 + 前端最小请求面）。
 *
 * - toCreate：新建占位元素（id 带 editor 前缀，isNewElement 判定）；
 * - toUpdate：已存在元素且 id ∈ 基线（本次加载可见）；
 * - toDelete：基线可见 ID − 当前编辑态 ID（天然再与基线求交，恒为可见子集）；
 * - ignored：非新建且不在基线的 ID（越界 / 后端未返回 / 模拟注入）一律忽略，
 *   不进入 create / update / delete 任何集合 —— 防非超管保存误删越界几何。
 */
function computeIncrementalDiff<T extends { id: string }>(
  items: T[],
  baseline: Set<string>,
  isNew: (id: string) => boolean
): { toCreate: T[]; toUpdate: T[]; toDelete: string[]; ignored: string[] } {
  const toCreate: T[] = [];
  const toUpdate: T[] = [];
  const ignored: string[] = [];
  const currentIds = new Set<string>();

  for (const item of items) {
    currentIds.add(item.id);
    if (isNew(item.id)) {
      toCreate.push(item);
    } else if (baseline.has(item.id)) {
      toUpdate.push(item);
    } else {
      ignored.push(item.id);
    }
  }

  const toDelete: string[] = [];
  for (const visibleId of baseline) {
    if (!currentIds.has(visibleId)) {
      toDelete.push(visibleId);
    }
  }
  return { toCreate, toUpdate, toDelete, ignored };
}

export function useFloorPlanEditor(
  currentFloor: Floor | null,
  options?: UseFloorPlanEditorOptions
): UseFloorPlanEditorReturn {
  const { message } = App.useApp();
  const [floorPlanData, setFloorPlanData] = useState<FloorPlanData | null>(null);
  const [floorPlanLoading, setFloorPlanLoading] = useState(false);
  const [isEditMode, setIsEditMode] = useState(false);
  // Phase 149：可见基线用 ref 承载（不进 render 依赖，遵循 useEffect/useCallback
  // 依赖稳定约定）；每次 loadFloorPlanData 成功后刷新。
  const visibleBaselineRef = useRef<VisibleBaseline>(emptyVisibleBaseline());

  const setEditMode = useCallback((edit: boolean) => {
    setIsEditMode(edit);
  }, []);

  const resetFloorPlan = useCallback(() => {
    setFloorPlanData(null);
    setIsEditMode(false);
    // Phase 149：切换 / 重置楼层时清空可见基线，防止跨楼层误用。
    visibleBaselineRef.current = emptyVisibleBaseline();
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

        // Phase 149 / SCOPE-03：固化「本次加载可见集」为保存增量 diff 基线。
        // 后端 List 已按可见集裁剪，故这里拿到的 walls/doors/texts 天然是可见子集。
        visibleBaselineRef.current = {
          walls: idsOf(parsedWalls),
          doors: idsOf(parsedDoors),
          texts: idsOf(parsedTexts),
        };

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
        // 加载失败 → 清空可见基线，避免用上一次楼层的基线误删本次几何。
        visibleBaselineRef.current = emptyVisibleBaseline();
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
  const _BATCH_SIZE = 10;

  const saveWalls = useCallback(
    async (walls: Wall[], floorId: string, baseline: Set<string>) => {
      // Phase 149 / SCOPE-03：只提交可见基线内元素的增量 diff，越界 ID 一律忽略。
      const { toCreate, toUpdate, toDelete } = computeIncrementalDiff(walls, baseline, (id) =>
        isNewElement(id, "wall_")
      );
      if (toCreate.length === 0 && toUpdate.length === 0 && toDelete.length === 0) return;
      const batchSave = async (item: Wall, isNew: boolean) => {
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

      const results = await Promise.allSettled([
        ...toCreate.map((wall) => batchSave(wall, true)),
        ...toUpdate.map((wall) => batchSave(wall, false)),
        // 删除集 = 基线可见 ID − 当前编辑态 ID（再与基线求交，恒为可见子集），
        // 越界几何 ID 不在基线 → 绝不进入删除集。
        ...(toDelete.length > 0 ? [wallApi.batch("delete", toDelete)] : []),
      ]);
      const failed = results.filter((r) => r.status === "rejected");
      if (failed.length > 0) {
        message.error(`墙壁保存失败 ${failed.length} 项`);
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    []
  );

  const saveDoors = useCallback(
    async (doors: Door[], floorId: string, baseline: Set<string>) => {
      // Phase 149 / SCOPE-03：可见基线增量 diff，越界 ID 不进 create/update/delete。
      const { toCreate, toUpdate, toDelete } = computeIncrementalDiff(doors, baseline, (id) =>
        isNewElement(id, "door_")
      );
      if (toCreate.length === 0 && toUpdate.length === 0 && toDelete.length === 0) return;
      const batchSave = async (item: Door, isNew: boolean) => {
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

      const results = await Promise.allSettled([
        ...toCreate.map((door) => batchSave(door, true)),
        ...toUpdate.map((door) => batchSave(door, false)),
        ...(toDelete.length > 0 ? [doorApi.batch("delete", toDelete)] : []),
      ]);
      const failed = results.filter((r) => r.status === "rejected");
      if (failed.length > 0) {
        message.error(`门保存失败 ${failed.length} 项`);
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    []
  );

  const saveTexts = useCallback(
    async (texts: TextElement[], floorId: string, baseline: Set<string>) => {
      // Phase 149 / SCOPE-03：可见基线增量 diff，越界 ID 不进 create/update/delete。
      const { toCreate, toUpdate, toDelete } = computeIncrementalDiff(texts, baseline, (id) =>
        isNewElement(id, "text_")
      );
      if (toCreate.length === 0 && toUpdate.length === 0 && toDelete.length === 0) return;
      const batchSave = async (item: TextElement, isNew: boolean) => {
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

      const results = await Promise.allSettled([
        ...toCreate.map((text) => batchSave(text, true)),
        ...toUpdate.map((text) => batchSave(text, false)),
        ...(toDelete.length > 0 ? [floorPlanTextApi.batch("delete", toDelete)] : []),
      ]);
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
        // Phase 149 / SCOPE-03：以「本次加载可见集」为基线做增量 diff，
        // 只提交可见元素的 create/update/delete；越界几何 ID 不进任何请求集。
        const baseline = visibleBaselineRef.current;

        await Promise.all([
          saveWalls(data.walls, floorId, baseline.walls),
          saveDoors(data.doors, floorId, baseline.doors),
          saveTexts(data.texts || [], floorId, baseline.texts),
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
