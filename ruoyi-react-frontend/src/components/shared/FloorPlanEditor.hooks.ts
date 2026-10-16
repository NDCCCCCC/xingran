/**
 * FloorPlanEditor 拖拽逻辑 Hook
 */

import { useState, useCallback } from "react";
import type { WorkstationNode } from "./FloorPlanEditor.types";
import { GRID_SIZE } from "./FloorPlanEditor.constants";

interface UseWorkstationDragOptions {
  workstations: WorkstationNode[];
  onUpdatePosition: (
    items: { id: string; positionX: number; positionY: number; rotation?: number }[]
  ) => Promise<void>;
}

interface DragState {
  isDragging: boolean;
  workstationId: string | null;
  startX: number;
  startY: number;
  originalX: number;
  originalY: number;
}

interface UseWorkstationDragReturn {
  dragState: DragState;
  draggedNodePos: { x: number; y: number } | null;
  handleStartDrag: (
    workstationId: string,
    startX: number,
    startY: number,
    originalX: number,
    originalY: number
  ) => void;
  handleDragMove: (clientX: number, clientY: number, scale: number) => void;
  handleEndDrag: () => Promise<void>;
  clearDraggedPos: () => void;
}

export function useWorkstationDrag(options: UseWorkstationDragOptions): UseWorkstationDragReturn {
  const { workstations, onUpdatePosition } = options;

  const [dragState, setDragState] = useState<DragState>({
    isDragging: false,
    workstationId: null,
    startX: 0,
    startY: 0,
    originalX: 0,
    originalY: 0,
  });

  const [draggedNodePos, setDraggedNodePos] = useState<{ x: number; y: number } | null>(null);

  // 开始拖拽
  const handleStartDrag = useCallback(
    (
      workstationId: string,
      startX: number,
      startY: number,
      originalX: number,
      originalY: number
    ) => {
      setDragState({
        isDragging: true,
        workstationId,
        startX,
        startY,
        originalX,
        originalY,
      });
    },
    []
  );

  // 拖拽移动
  const handleDragMove = useCallback(
    (clientX: number, clientY: number, scale: number) => {
      if (!dragState.isDragging || !dragState.workstationId) {
        return;
      }

      const dx = (clientX - dragState.startX) / scale;
      const dy = (clientY - dragState.startY) / scale;

      setDraggedNodePos({
        x: dragState.originalX + dx,
        y: dragState.originalY + dy,
      });
    },
    [dragState]
  );

  // 结束拖拽
  const handleEndDrag = useCallback(async () => {
    if (!dragState.isDragging || !dragState.workstationId || !draggedNodePos) {
      // 重置状态
      setDragState({
        isDragging: false,
        workstationId: null,
        startX: 0,
        startY: 0,
        originalX: 0,
        originalY: 0,
      });
      return;
    }

    const workstation = workstations.find((w) => w.id === dragState.workstationId);
    if (workstation) {
      // 网格吸附 - 使用临时位置
      const snappedX = Math.round(draggedNodePos.x / GRID_SIZE) * GRID_SIZE;
      const snappedY = Math.round(draggedNodePos.y / GRID_SIZE) * GRID_SIZE;

      await onUpdatePosition([
        {
          id: workstation.id,
          positionX: snappedX,
          positionY: snappedY,
        },
      ]);
    }

    // 清除临时拖拽位置并重置状态
    setDraggedNodePos(null);
    setDragState({
      isDragging: false,
      workstationId: null,
      startX: 0,
      startY: 0,
      originalX: 0,
      originalY: 0,
    });
  }, [dragState, draggedNodePos, workstations, onUpdatePosition]);

  // 清除临时拖拽位置
  const clearDraggedPos = useCallback(() => {
    setDraggedNodePos(null);
  }, []);

  return {
    dragState,
    draggedNodePos,
    handleStartDrag,
    handleDragMove,
    handleEndDrag,
    clearDraggedPos,
  };
}
