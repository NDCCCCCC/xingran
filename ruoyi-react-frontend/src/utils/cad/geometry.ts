/**
 * CAD几何计算工具
 */

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

export function distance(p1: Point, p2: Point): number {
  return Math.sqrt(Math.pow(p2.x - p1.x, 2) + Math.pow(p2.y - p1.y, 2));
}

export function angle(p1: Point, p2: Point): number {
  return (Math.atan2(p2.y - p1.y, p2.x - p1.x) * 180) / Math.PI;
}

export function rotatePoint(point: Point, center: Point, angleDeg: number): Point {
  const angleRad = (angleDeg * Math.PI) / 180;
  const cos = Math.cos(angleRad);
  const sin = Math.sin(angleRad);

  const dx = point.x - center.x;
  const dy = point.y - center.y;

  return {
    x: center.x + dx * cos - dy * sin,
    y: center.y + dx * sin + dy * cos,
  };
}

export function isPointOnLine(
  point: Point,
  lineStart: Point,
  lineEnd: Point,
  tolerance = 5
): boolean {
  const d1 = distance(point, lineStart);
  const d2 = distance(point, lineEnd);
  const lineLength = distance(lineStart, lineEnd);

  return Math.abs(d1 + d2 - lineLength) <= tolerance;
}

export function getPolylineLength(points: Point[]): number {
  let total = 0;
  for (let i = 1; i < points.length; i++) {
    total += distance(points[i - 1], points[i]);
  }
  return total;
}

export function snapToGrid(point: Point, gridSize: number): Point {
  return {
    x: Math.round(point.x / gridSize) * gridSize,
    y: Math.round(point.y / gridSize) * gridSize,
  };
}

export function snapToPoint(
  point: Point,
  targetPoints: Point[],
  snapDistance = 10
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

export function isPointInRect(point: Point, rect: Rect): boolean {
  return (
    point.x >= rect.x &&
    point.x <= rect.x + rect.width &&
    point.y >= rect.y &&
    point.y <= rect.y + rect.height
  );
}

export function getRectCenter(rect: Rect): Point {
  return {
    x: rect.x + rect.width / 2,
    y: rect.y + rect.height / 2,
  };
}

export function isRectIntersect(r1: Rect, r2: Rect): boolean {
  return !(
    r1.x + r1.width < r2.x ||
    r2.x + r2.width < r1.x ||
    r1.y + r1.height < r2.y ||
    r2.y + r2.height < r1.y
  );
}

// ==================== 线段操作 ====================

export function getClosestPointOnLine(
  point: Point,
  lineStart: Point,
  lineEnd: Point
): Point {
  const dx = lineEnd.x - lineStart.x;
  const dy = lineEnd.y - lineStart.y;
  const lengthSq = dx * dx + dy * dy;

  if (lengthSq === 0) return lineStart;

  const t = Math.max(0, Math.min(1, ((point.x - lineStart.x) * dx + (point.y - lineStart.y) * dy) / lengthSq));

  return {
    x: lineStart.x + t * dx,
    y: lineStart.y + t * dy,
  };
}

export function perpendicularDistance(point: Point, lineStart: Point, lineEnd: Point): number {
  const closest = getClosestPointOnLine(point, lineStart, lineEnd);
  return distance(point, closest);
}

// ==================== 角度操作 ====================

export function normalizeAngle(angle: number): number {
  let normalized = angle % 360;
  if (normalized < 0) normalized += 360;
  return normalized;
}

export function angleDifference(angle1: number, angle2: number): number {
  const diff = normalizeAngle(angle2) - normalizeAngle(angle1);
  if (diff > 180) return diff - 360;
  if (diff < -180) return diff + 360;
  return diff;
}

// ==================== 多边形操作 ====================

export function isPointInPolygon(point: Point, polygon: Point[]): boolean {
  let inside = false;

  for (let i = 0, j = polygon.length - 1; i < polygon.length; j = i++) {
    const xi = polygon[i].x;
    const yi = polygon[i].y;
    const xj = polygon[j].x;
    const yj = polygon[j].y;

    const intersect =
      yi > point.y !== yj > point.y &&
      point.x < ((xj - xi) * (point.y - yi)) / (yj - yi) + xi;

    if (intersect) inside = !inside;
  }

  return inside;
}

export function arePointsCollinear(p1: Point, p2: Point, p3: Point, tolerance = 1): boolean {
  const area = Math.abs(
    (p2.x - p1.x) * (p3.y - p1.y) - (p3.x - p1.x) * (p2.y - p1.y)
  );
  return area < tolerance;
}

// ==================== 点变换 ====================

export function scalePoint(point: Point, scale: number): Point {
  return {
    x: point.x * scale,
    y: point.y * scale,
  };
}

export function translatePoint(point: Point, offset: Point): Point {
  return {
    x: point.x + offset.x,
    y: point.y + offset.y,
  };
}
