/**
 * CAD 几何计算工具函数
 */

// ==================== 常量定义 ====================

/** 默认容差值 */
export const DEFAULT_TOLERANCE = 5;

/** 默认吸附距离 */
export const DEFAULT_SNAP_DISTANCE = 10;

/** 度数转弧度 */
export const DEG_TO_RAD = Math.PI / 180;

/** 弧度转度数 */
export const RAD_TO_DEG = 180 / Math.PI;

// ==================== 基础类型 ====================

export interface Point {
  x: number;
  y: number;
}

export interface Size {
  width: number;
  height: number;
}

export interface Rect {
  x: number;
  y: number;
  width: number;
  height: number;
}

// ==================== 距离和角度计算 ====================

/**
 * 计算两点之间的距离
 */
export function distance(p1: Point, p2: Point): number {
  const dx = p2.x - p1.x;
  const dy = p2.y - p1.y;
  return Math.sqrt(dx * dx + dy * dy);
}

/**
 * 计算两点之间的角度（度数）
 */
export function angle(p1: Point, p2: Point): number {
  return Math.atan2(p2.y - p1.y, p2.x - p1.x) * RAD_TO_DEG;
}

/**
 * 旋转点
 */
export function rotatePoint(point: Point, center: Point, angleDeg: number): Point {
  const angleRad = angleDeg * DEG_TO_RAD;
  const cos = Math.cos(angleRad);
  const sin = Math.sin(angleRad);

  const dx = point.x - center.x;
  const dy = point.y - center.y;

  return {
    x: center.x + dx * cos - dy * sin,
    y: center.y + dx * sin + dy * cos,
  };
}

/**
 * 判断点是否在线段上
 */
export function isPointOnLine(
  point: Point,
  lineStart: Point,
  lineEnd: Point,
  tolerance = DEFAULT_TOLERANCE
): boolean {
  const d1 = distance(point, lineStart);
  const d2 = distance(point, lineEnd);
  const lineLength = distance(lineStart, lineEnd);

  return Math.abs(d1 + d2 - lineLength) <= tolerance;
}

/**
 * 计算折线总长度
 */
export function getPolylineLength(points: Point[]): number {
  let total = 0;
  for (let i = 1; i < points.length; i++) {
    total += distance(points[i - 1], points[i]);
  }
  return total;
}

/**
 * 将点吸附到网格
 */
export function snapToGrid(point: Point, gridSize: number): Point {
  return {
    x: Math.round(point.x / gridSize) * gridSize,
    y: Math.round(point.y / gridSize) * gridSize,
  };
}

/**
 * 将点吸附到最近的参考点
 */
export function snapToPoint(
  point: Point,
  targetPoints: Point[],
  snapDistance = DEFAULT_SNAP_DISTANCE
): Point | null {
  let closestPoint: Point | null = null;
  let minDistance = snapDistance;

  for (const target of targetPoints) {
    const d = distance(point, target);
    if (d < minDistance) {
      minDistance = d;
      closestPoint = target;
    }
  }

  return closestPoint;
}

// ==================== 矩形操作 ====================

/**
 * 判断点是否在矩形内
 */
export function isPointInRect(point: Point, rect: Rect): boolean {
  return (
    point.x >= rect.x &&
    point.x <= rect.x + rect.width &&
    point.y >= rect.y &&
    point.y <= rect.y + rect.height
  );
}

/**
 * 获取矩形中心点
 */
export function getRectCenter(rect: Rect): Point {
  return {
    x: rect.x + rect.width / 2,
    y: rect.y + rect.height / 2,
  };
}

/**
 * 判断两个矩形是否相交
 */
export function isRectIntersect(r1: Rect, r2: Rect): boolean {
  return !(
    r1.x + r1.width < r2.x ||
    r2.x + r2.width < r1.x ||
    r1.y + r1.height < r2.y ||
    r2.y + r2.height < r1.y
  );
}

// ==================== 线段操作 ====================

/**
 * 获取点到线段的最近点
 */
export function getClosestPointOnLine(point: Point, lineStart: Point, lineEnd: Point): Point {
  const dx = lineEnd.x - lineStart.x;
  const dy = lineEnd.y - lineStart.y;
  const lengthSq = dx * dx + dy * dy;

  if (lengthSq === 0) return lineStart;

  const t = Math.max(
    0,
    Math.min(1, ((point.x - lineStart.x) * dx + (point.y - lineStart.y) * dy) / lengthSq)
  );

  return {
    x: lineStart.x + t * dx,
    y: lineStart.y + t * dy,
  };
}

/**
 * 计算点到线段的垂直距离
 */
export function perpendicularDistance(point: Point, lineStart: Point, lineEnd: Point): number {
  const closest = getClosestPointOnLine(point, lineStart, lineEnd);
  return distance(point, closest);
}

// ==================== 角度操作 ====================

/**
 * 将角度标准化到 [0, 360) 范围
 */
export function normalizeAngle(angle: number): number {
  let normalized = angle % 360;
  if (normalized < 0) normalized += 360;
  return normalized;
}

/**
 * 计算两个角度之间的最小差值
 */
export function angleDifference(angle1: number, angle2: number): number {
  const diff = normalizeAngle(angle2) - normalizeAngle(angle1);
  if (diff > 180) return diff - 360;
  if (diff < -180) return diff + 360;
  return diff;
}

// ==================== 多边形操作 ====================

/**
 * 判断点是否在多边形内（射线法）
 */
export function isPointInPolygon(point: Point, polygon: Point[]): boolean {
  let inside = false;

  for (let i = 0, j = polygon.length - 1; i < polygon.length; j = i++) {
    const xi = polygon[i].x;
    const yi = polygon[i].y;
    const xj = polygon[j].x;
    const yj = polygon[j].y;

    const intersect =
      yi > point.y !== yj > point.y && point.x < ((xj - xi) * (point.y - yi)) / (yj - yi) + xi;

    if (intersect) inside = !inside;
  }

  return inside;
}

/**
 * 判断三个点是否共线
 */
export function arePointsCollinear(
  p1: Point,
  p2: Point,
  p3: Point,
  tolerance = DEFAULT_TOLERANCE
): boolean {
  const area = Math.abs((p2.x - p1.x) * (p3.y - p1.y) - (p3.x - p1.x) * (p2.y - p1.y));
  return area < tolerance;
}

// ==================== 点变换 ====================

/**
 * 缩放点
 */
export function scalePoint(point: Point, scale: number): Point {
  return {
    x: point.x * scale,
    y: point.y * scale,
  };
}

/**
 * 平移点
 */
export function translatePoint(point: Point, offset: Point): Point {
  return {
    x: point.x + offset.x,
    y: point.y + offset.y,
  };
}
