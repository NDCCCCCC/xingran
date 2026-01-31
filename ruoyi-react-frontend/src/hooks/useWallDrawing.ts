/**
 * 墙体绘制Hook
 */

import { useState, useCallback } from 'react';
import type { Point } from '@/utils/cad/geometry';
import { snapToGrid, distance } from '@/utils/cad/geometry';

export interface WallDrawingResult {
  points: Point[];
  type: 'straight' | 'polyline';
}

export interface UseWallDrawingOptions {
  gridSize?: number;
  snapEnabled?: boolean;
  angleSnapEnabled?: boolean;
  onPointsChange?: (points: Point[]) => void;
  onComplete?: (wall: WallDrawingResult) => void;
}

// 角度吸附增量（度）
const ANGLE_SNAP_INCREMENT = 15; // 15度 = 0度、15度、30度、45度、60度、75度、90度等

/**
 * 角度吸附：将角度吸附到最近的增量角度
 */
function snapAngle(angle: number, increment: number): number {
  const degrees = (angle * 180) / Math.PI;
  const snappedDegrees = Math.round(degrees / increment) * increment;
  return (snappedDegrees * Math.PI) / 180;
}

/**
 * 应用角度吸附到点
 */
function applyAngleSnap(point: Point, origin: Point): Point {
  const dx = point.x - origin.x;
  const dy = point.y - origin.y;
  const angle = Math.atan2(dy, dx);
  const dist = Math.sqrt(dx * dx + dy * dy);

  // 吸附角度
  const snappedAngle = snapAngle(angle, ANGLE_SNAP_INCREMENT);

  return {
    x: origin.x + Math.cos(snappedAngle) * dist,
    y: origin.y + Math.sin(snappedAngle) * dist,
  };
}

export function useWallDrawing(options: UseWallDrawingOptions = {}) {
  const {
    gridSize = 20,
    snapEnabled = true,
    angleSnapEnabled = true,
    onPointsChange,
    onComplete,
  } = options;

  const [isDrawing, setIsDrawing] = useState(false);
  const [drawPoints, setDrawPoints] = useState<Point[]>([]);
  const [previewPoint, setPreviewPoint] = useState<Point | null>(null);
  // 用于判断是否按住 Shift 键禁用角度吸附
  const [disableAngleSnap, setDisableAngleSnap] = useState(false);

  const snapPoint = useCallback((point: Point, origin?: Point): Point => {
    let result = snapEnabled ? snapToGrid(point, gridSize) : point;

    // 如果有原点且启用角度吸附且没有禁用
    if (origin && angleSnapEnabled && !disableAngleSnap) {
      result = applyAngleSnap(result, origin);
    }

    return result;
  }, [gridSize, snapEnabled, angleSnapEnabled, disableAngleSnap]);

  const startDrawing = useCallback((point: Point, existingPoints?: Point[]) => {
    const snappedPoint = snapPoint(point);

    setIsDrawing(true);

    // 如果提供了已有墙体点，使用它们作为起点
    if (existingPoints && existingPoints.length > 0) {
      setDrawPoints(existingPoints);
      setPreviewPoint(snappedPoint);
      onPointsChange?.(existingPoints);
    } else {
      setDrawPoints([snappedPoint]);
      setPreviewPoint(snappedPoint);
      onPointsChange?.([snappedPoint]);
    }
  }, [snapPoint, onPointsChange]);

  const addPoint = useCallback((point: Point) => {
    if (!isDrawing || drawPoints.length === 0) return;

    const lastPoint = drawPoints[drawPoints.length - 1];
    const snappedPoint = snapPoint(point, lastPoint);

    if (distance(lastPoint, snappedPoint) < 5) {
      return;
    }

    const newPoints = [...drawPoints, snappedPoint];
    setDrawPoints(newPoints);
    setPreviewPoint(snappedPoint);

    onPointsChange?.(newPoints);
  }, [isDrawing, drawPoints, snapPoint, distance, onPointsChange]);

  const updatePreview = useCallback((point: Point, shiftKey?: boolean) => {
    if (!isDrawing) return;

    // Shift 键禁用角度吸附
    setDisableAngleSnap(!!shiftKey);

    const lastPoint = drawPoints[drawPoints.length - 1];
    const snappedPoint = snapPoint(point, lastPoint);
    setPreviewPoint(snappedPoint);
  }, [isDrawing, drawPoints, snapPoint]);

  const finishDrawing = useCallback(() => {
    if (!isDrawing || drawPoints.length < 2) {
      setIsDrawing(false);
      setDrawPoints([]);
      setPreviewPoint(null);
      setDisableAngleSnap(false);
      onPointsChange?.([]);
      return;
    }

    const wall: WallDrawingResult = {
      points: drawPoints,
      type: drawPoints.length === 2 ? 'straight' : 'polyline',
    };

    onComplete?.(wall);

    setIsDrawing(false);
    setDrawPoints([]);
    setPreviewPoint(null);
    setDisableAngleSnap(false);
  }, [isDrawing, drawPoints, onComplete, onPointsChange]);

  const cancelDrawing = useCallback(() => {
    setIsDrawing(false);
    setDrawPoints([]);
    setPreviewPoint(null);
    setDisableAngleSnap(false);
    onPointsChange?.([]);
  }, [onPointsChange]);

  const undoPoint = useCallback(() => {
    if (drawPoints.length <= 1) {
      cancelDrawing();
      return;
    }

    const newPoints = drawPoints.slice(0, -1);
    setDrawPoints(newPoints);
    setPreviewPoint(newPoints[newPoints.length - 1]);
    onPointsChange?.(newPoints);
  }, [drawPoints, cancelDrawing, onPointsChange]);

  return {
    isDrawing,
    drawPoints,
    previewPoint,
    startDrawing,
    addPoint,
    updatePreview,
    finishDrawing,
    cancelDrawing,
    undoPoint,
  };
}
