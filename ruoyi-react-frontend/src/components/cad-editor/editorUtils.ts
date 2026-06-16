/**
 * CAD 编辑器辅助函数
 */

import type { Point, Wall } from "./types";

// ==================== 常量 ====================

export const EDITOR_CONSTANTS = {
  DEFAULT_GRID_SIZE: 20,
  DEFAULT_CANVAS_WIDTH: 2000,
  DEFAULT_CANVAS_HEIGHT: 2000,
  MIN_WORKSTATION_SPACING: 0,
  MIN_SCALE: 0.1,
  MAX_SCALE: 5,
  ZOOM_STEP: 0.1,
  DRAG_THRESHOLD: 5,
  NEARBY_NODE_THRESHOLD: 15,
} as const;

// ==================== 几何计算 ====================

/**
 * 将坐标吸附到网格
 */
export function snapToGrid(coordinate: number, gridSize: number): number {
  return Math.round(coordinate / gridSize) * gridSize;
}

/**
 * 检查两个矩形是否碰撞（考虑最小间距）
 */
export function checkRectCollision(
  rect1: { x: number; y: number; width: number; height: number },
  rect2: { x: number; y: number; width: number; height: number },
  minSpacing: number
): boolean {
  const rect1Left = rect1.x - rect1.width / 2 - minSpacing;
  const rect1Right = rect1.x + rect1.width / 2 + minSpacing;
  const rect1Top = rect1.y - rect1.height / 2 - minSpacing;
  const rect1Bottom = rect1.y + rect1.height / 2 + minSpacing;

  const rect2Left = rect2.x - rect2.width / 2;
  const rect2Right = rect2.x + rect2.width / 2;
  const rect2Top = rect2.y - rect2.height / 2;
  const rect2Bottom = rect2.y + rect2.height / 2;

  return !(
    rect1Right < rect2Left ||
    rect1Left > rect2Right ||
    rect1Bottom < rect2Top ||
    rect1Top > rect2Bottom
  );
}

/**
 * 检查工位是否与现有工位碰撞
 */
export function checkWorkstationCollision(
  workstation: { x: number; y: number; width: number; height: number },
  existingWorkstations: Array<{ id: string; x: number; y: number; width: number; height: number }>,
  excludeId?: string
): boolean {
  for (const ws of existingWorkstations) {
    if (excludeId && ws.id === excludeId) {
      continue;
    }
    if (checkRectCollision(workstation, ws, EDITOR_CONSTANTS.MIN_WORKSTATION_SPACING)) {
      return true;
    }
  }
  return false;
}

/**
 * 检查工位拖动时的碰撞（允许从碰撞状态移出）
 * 只检查目标位置是否会导致新的碰撞，不包括与当前已经在碰撞中的工位的碰撞
 */
export function checkWorkstationCollisionForDrag(
  targetWorkstation: { id: string; x: number; y: number; width: number; height: number },
  existingWorkstations: Array<{ id: string; x: number; y: number; width: number; height: number }>,
  originalWorkstation: { x: number; y: number; width: number; height: number }
): boolean {
  for (const ws of existingWorkstations) {
    if (ws.id === targetWorkstation.id) {
      continue;
    }

    // 检查与目标位置的碰撞
    const targetCollision = checkRectCollision(targetWorkstation, ws, EDITOR_CONSTANTS.MIN_WORKSTATION_SPACING);

    // 检查原始位置是否已经与该工位碰撞
    const originalCollision = checkRectCollision(originalWorkstation, ws, EDITOR_CONSTANTS.MIN_WORKSTATION_SPACING);

    // 如果目标位置碰撞，但原始位置没有碰撞，则阻止移动（新的碰撞）
    // 如果两者都碰撞，允许移动（从碰撞状态中移出）
    // 如果目标位置没有碰撞，允许移动
    if (targetCollision && !originalCollision) {
      return true;
    }
  }
  return false;
}

/**
 * 计算点到线段的距离
 */
export function pointToLineDistance(point: Point, lineStart: Point, lineEnd: Point): number {
  const A = point.x - lineStart.x;
  const B = point.y - lineStart.y;
  const C = lineEnd.x - lineStart.x;
  const D = lineEnd.y - lineStart.y;

  const dot = A * C + B * D;
  const lenSq = C * C + D * D;
  let param = -1;

  if (lenSq !== 0) {
    param = dot / lenSq;
  }

  let xx: number;
  let yy: number;

  if (param < 0) {
    xx = lineStart.x;
    yy = lineStart.y;
  } else if (param > 1) {
    xx = lineEnd.x;
    yy = lineEnd.y;
  } else {
    xx = lineStart.x + param * C;
    yy = lineStart.y + param * D;
  }

  const dx = point.x - xx;
  const dy = point.y - yy;
  return Math.sqrt(dx * dx + dy * dy);
}

/**
 * 查找附近的墙体节点
 */
export interface NearbyNode {
  point: Point;
  wallId: string;
  pointIndex: number;
}

export function findNearbyWallNode(
  walls: Wall[],
  point: Point,
  threshold = EDITOR_CONSTANTS.NEARBY_NODE_THRESHOLD
): NearbyNode | null {
  for (const wall of walls) {
    for (let i = 0; i < wall.points.length; i++) {
      const node = wall.points[i];
      const dist = Math.sqrt(Math.pow(point.x - node.x, 2) + Math.pow(point.y - node.y, 2));
      if (dist < threshold) {
        return { point: node, wallId: wall.id, pointIndex: i };
      }
    }
  }
  return null;
}

// ==================== 元素类型识别 ====================

export type ElementType = "wall" | "door" | "workstation" | "text";

export function getElementType(element: unknown): ElementType | null {
  if (!element || typeof element !== "object") return null;

  if ("points" in element) return "wall";
  if ("content" in element) return "text";
  if ("width" in element && "length" in element) return "door";
  if ("code" in element) return "workstation";

  return null;
}
