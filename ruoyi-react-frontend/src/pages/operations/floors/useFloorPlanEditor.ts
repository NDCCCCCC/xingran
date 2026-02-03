/**
 * 楼层平面图编辑器 Hook
 */

import { useState, useCallback } from 'react';
import type { Floor } from '@/types';
import { wallApi, doorApi, workstationApi, floorPlanTextApi } from '@/lib/opsApi';
import type { FloorPlanData } from '@/components/cad-editor/types';
import { message } from 'antd';
import { processWorkstations, parseJsonField, stringifyJsonField, isNewElement } from './utils';
import { DEFAULT_FLOOR_PLAN_CONFIG } from './constants';

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

  /**
   * 加载平面图数据
   */
  const loadFloorPlanData = useCallback(async (floorId: string) => {
    setFloorPlanLoading(true);
    try {
      const [wallsResult, doorsResult, workstationsResult] = await Promise.all([
        wallApi.list({ floorId, current: 1, pageSize: 1000 }),
        doorApi.list({ floorId, current: 1, pageSize: 1000 }),
        workstationApi.list({ floorId, current: 1, pageSize: 1000 }),
      ]);

      const walls = wallsResult.data?.list || [];
      const doors = doorsResult.data?.list || [];
      const workstations = workstationsResult.data?.list || [];

      // 尝试加载文本元素
      let texts: any[] = [];
      try {
        const textsResult = await floorPlanTextApi.list({ floorId, current: 1, pageSize: 1000 });
        texts = textsResult.data?.list || [];
      } catch (textError) {
        console.warn('文本元素API不可用，可能需要重启后端服务:', textError);
      }

      const parsedWalls = walls.map((wall: any) => ({
        ...wall,
        points: parseJsonField(wall.points),
      }));

      const parsedDoors = doors.map((door: any) => ({
        ...door,
        position: parseJsonField(door.position),
      }));

      const parsedTexts = texts.map((text: any) => ({
        ...text,
        position: parseJsonField(text.position),
      }));

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
        gridSize: DEFAULT_FLOOR_PLAN_CONFIG.GRID_SIZE,
        showGrid: true,
        snapToGrid: true,
      });
    } catch (error) {
      console.error('加载平面图失败:', error);
      message.error('加载平面图失败');
      setFloorPlanData({
        floorId,
        floorName: currentFloor?.name || `${currentFloor?.floorNo}层`,
        width: DEFAULT_FLOOR_PLAN_CONFIG.CANVAS_WIDTH,
        height: DEFAULT_FLOOR_PLAN_CONFIG.CANVAS_HEIGHT,
        walls: [],
        doors: [],
        workstations: [],
        texts: [],
        gridSize: DEFAULT_FLOOR_PLAN_CONFIG.GRID_SIZE,
        showGrid: true,
        snapToGrid: true,
      });
    } finally {
      setFloorPlanLoading(false);
    }
  }, [currentFloor]);

  /**
   * 保存墙体
   */
  const saveWalls = useCallback(async (walls: any[], floorId: string) => {
    for (const wall of walls) {
      const isNew = isNewElement(wall.id, 'wall_');
      const { id: _wallId, ...wallWithoutId } = wall;
      const wallData = {
        ...(isNew ? wallWithoutId : wall),
        floorId,
        points: stringifyJsonField(wall.points),
        type: wall.type as 'straight' | 'curved' | 'l_shaped' | 'polyline',
      };

      if (isNew) {
        await wallApi.create(wallData);
      } else {
        await wallApi.update(wall.id, wallData);
      }
    }
  }, []);

  /**
   * 保存门
   */
  const saveDoors = useCallback(async (doors: any[], floorId: string) => {
    for (const door of doors) {
      const isNew = isNewElement(door.id, 'door_');
      const { id: _doorId, ...doorWithoutId } = door;
      const doorData = {
        ...(isNew ? doorWithoutId : door),
        floorId,
        position: stringifyJsonField(door.position),
      };

      if (isNew) {
        await doorApi.create(doorData);
      } else {
        await doorApi.update(door.id, doorData);
      }
    }
  }, []);

  /**
   * 保存文本元素
   */
  const saveTexts = useCallback(async (texts: any[], floorId: string) => {
    if (texts.length === 0) return;

    try {
      for (const text of texts) {
        const isNew = isNewElement(text.id, 'text_');
        const { id: _textId, ...textWithoutId } = text;
        const textData = {
          ...(isNew ? textWithoutId : text),
          floorId,
          position: stringifyJsonField(text.position),
        };

        if (isNew) {
          await floorPlanTextApi.create(textData);
        } else {
          await floorPlanTextApi.update(text.id, textData);
        }
      }
    } catch (textError) {
      console.warn('保存文本元素失败，可能需要重启后端服务:', textError);
    }
  }, []);

  /**
   * 保存工位数据
   */
  const saveWorkstations = useCallback(async (workstations: any[]) => {
    const updates = workstations.map((ws) => ({
      id: ws.id,
      positionX: Math.round(ws.x),
      positionY: Math.round(ws.y),
      rotation: ws.rotation || 0,
      width: Math.round((ws.width || 160) * 10),
      depth: Math.round((ws.height || 70) * 10),
      deskType: ws.type || 0,
    }));

    if (updates.length > 0) {
      await workstationApi.updatePositions(updates as any);
    }
  }, []);

  /**
   * 保存平面图
   */
  const saveFloorPlan = useCallback(async (data: FloorPlanData, floor: Floor) => {
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

      message.success('保存成功');
      await loadFloorPlanData(floorId);
    } catch (error) {
      console.error('保存失败:', error);
      message.error('保存失败');
    } finally {
      options?.onSaveEnd?.();
    }
  }, [loadFloorPlanData, saveWalls, saveDoors, saveTexts, saveWorkstations, options]);

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
