import { describe, expect, it } from "vitest";
import {
  angle,
  angleDifference,
  arePointsCollinear,
  DEFAULT_SNAP_DISTANCE,
  DEFAULT_TOLERANCE,
  distance,
  getClosestPointOnLine,
  getPolylineLength,
  getRectCenter,
  isPointInPolygon,
  isPointInRect,
  isPointOnLine,
  isRectIntersect,
  normalizeAngle,
  perpendicularDistance,
  rotatePoint,
  scalePoint,
  snapToGrid,
  snapToPoint,
  translatePoint,
  type Point,
  type Rect,
} from "./geometry";

describe("geometry 基础常量", () => {
  it("默认容差与吸附距离", () => {
    expect(DEFAULT_TOLERANCE).toBe(5);
    expect(DEFAULT_SNAP_DISTANCE).toBe(10);
  });
});

describe("距离与角度", () => {
  it("distance 计算欧氏距离", () => {
    expect(distance({ x: 0, y: 0 }, { x: 3, y: 4 })).toBe(5);
    expect(distance({ x: 1, y: 1 }, { x: 1, y: 1 })).toBe(0);
  });

  it("angle 计算方位角（度）", () => {
    expect(angle({ x: 0, y: 0 }, { x: 1, y: 0 })).toBe(0);
    expect(angle({ x: 0, y: 0 }, { x: 0, y: 1 })).toBe(90);
    expect(angle({ x: 0, y: 0 }, { x: -1, y: 0 })).toBe(180);
  });

  it("rotatePoint 绕中心旋转（90 度直角边互换）", () => {
    const rotated = rotatePoint({ x: 2, y: 0 }, { x: 0, y: 0 }, 90);
    expect(rotated.x).toBeCloseTo(0, 6);
    expect(rotated.y).toBeCloseTo(2, 6);
    // 旋转不改变到中心的距离
    expect(distance(rotated, { x: 0, y: 0 })).toBeCloseTo(2, 6);
  });
});

describe("线段判定与长度", () => {
  it("isPointOnLine 命中与容差", () => {
    const a = { x: 0, y: 0 };
    const b = { x: 10, y: 0 };
    expect(isPointOnLine({ x: 5, y: 0 }, a, b)).toBe(true);
    expect(isPointOnLine({ x: 5, y: 2 }, a, b)).toBe(true); // 偏差 0.77 ≤ 默认容差 5
    expect(isPointOnLine({ x: 5, y: 12 }, a, b)).toBe(false); // 偏差 16 > 5
    expect(isPointOnLine({ x: 5, y: 6 }, a, b)).toBe(false); // 偏差 5.62 > 5
    expect(isPointOnLine({ x: 5, y: 6 }, a, b, 10)).toBe(true); // 放宽容差后命中
  });

  it("getPolylineLength 累加各段", () => {
    const points: Point[] = [
      { x: 0, y: 0 },
      { x: 3, y: 4 },
      { x: 3, y: 10 },
    ];
    expect(getPolylineLength(points)).toBe(11); // 5 + 6
    expect(getPolylineLength([{ x: 1, y: 1 }])).toBe(0);
  });

  it("getClosestPointOnLine 垂足计算与退化线段", () => {
    expect(getClosestPointOnLine({ x: 5, y: 3 }, { x: 0, y: 0 }, { x: 10, y: 0 })).toEqual({
      x: 5,
      y: 0,
    });
    // 投影超出线段两端时夹紧到端点
    expect(getClosestPointOnLine({ x: -3, y: 0 }, { x: 0, y: 0 }, { x: 10, y: 0 })).toEqual({
      x: 0,
      y: 0,
    });
    // 零长线段返回起点
    expect(getClosestPointOnLine({ x: 5, y: 5 }, { x: 1, y: 1 }, { x: 1, y: 1 })).toEqual({
      x: 1,
      y: 1,
    });
  });

  it("perpendicularDistance 点到线段距离", () => {
    expect(perpendicularDistance({ x: 5, y: 3 }, { x: 0, y: 0 }, { x: 10, y: 0 })).toBe(3);
  });

  it("arePointsCollinear 共线判定（面积 < 容差）", () => {
    expect(arePointsCollinear({ x: 0, y: 0 }, { x: 1, y: 1 }, { x: 2, y: 2 })).toBe(true);
    // 面积 = |1*10 - 0*1| = 10 > 默认容差 5 → 不共线
    expect(arePointsCollinear({ x: 0, y: 0 }, { x: 1, y: 1 }, { x: 0, y: 10 })).toBe(false);
    // 放宽容差后判共线
    expect(arePointsCollinear({ x: 0, y: 0 }, { x: 1, y: 1 }, { x: 0, y: 10 }, 20)).toBe(true);
  });
});

describe("吸附", () => {
  it("snapToGrid 吸附到最近网格点", () => {
    expect(snapToGrid({ x: 23, y: 47 }, 10)).toEqual({ x: 20, y: 50 });
    expect(snapToGrid({ x: 5, y: 5 }, 10)).toEqual({ x: 10, y: 10 });
  });

  it("snapToPoint 吸附到范围内最近点，超界返回 null", () => {
    const targets: Point[] = [
      { x: 0, y: 0 },
      { x: 8, y: 0 },
    ];
    expect(snapToPoint({ x: 7, y: 1 }, targets)).toEqual({ x: 8, y: 0 });
    expect(snapToPoint({ x: 30, y: 30 }, targets)).toBeNull();
  });
});

describe("矩形操作", () => {
  const rect: Rect = { x: 0, y: 0, width: 10, height: 5 };

  it("isPointInRect 含边界", () => {
    expect(isPointInRect({ x: 5, y: 2 }, rect)).toBe(true);
    expect(isPointInRect({ x: 0, y: 0 }, rect)).toBe(true);
    expect(isPointInRect({ x: 10, y: 5 }, rect)).toBe(true); // 边界含
    expect(isPointInRect({ x: 10.1, y: 2 }, rect)).toBe(false);
  });

  it("getRectCenter 计算中心", () => {
    expect(getRectCenter(rect)).toEqual({ x: 5, y: 2.5 });
  });

  it("isRectIntersect 相交 / 相离 / 接触", () => {
    expect(isRectIntersect(rect, { x: 5, y: 0, width: 10, height: 5 })).toBe(true);
    expect(isRectIntersect(rect, { x: 20, y: 20, width: 5, height: 5 })).toBe(false);
    expect(isRectIntersect(rect, { x: 10, y: 0, width: 5, height: 5 })).toBe(true); // 边接触算相交
  });
});

describe("角度标准化", () => {
  it("normalizeAngle 归一到 [0, 360)", () => {
    expect(normalizeAngle(370)).toBe(10);
    expect(normalizeAngle(-350)).toBe(10);
    expect(normalizeAngle(720)).toBe(0);
  });

  it("angleDifference 取最小差值（含跨 0 回绕）", () => {
    expect(angleDifference(10, 30)).toBe(20);
    expect(angleDifference(350, 10)).toBe(20); // 跨 0
    expect(angleDifference(10, 350)).toBe(-20);
    expect(angleDifference(0, 180)).toBe(180);
  });
});

describe("多边形", () => {
  const square: Point[] = [
    { x: 0, y: 0 },
    { x: 10, y: 0 },
    { x: 10, y: 10 },
    { x: 0, y: 10 },
  ];

  it("isPointInPolygon 射线法（内/外/顶点）", () => {
    expect(isPointInPolygon({ x: 5, y: 5 }, square)).toBe(true);
    expect(isPointInPolygon({ x: 15, y: 5 }, square)).toBe(false);
    // 顶点 (0,0) 按该射线法实现计为内部（右向射线穿过右边界奇数次）
    expect(isPointInPolygon({ x: 0, y: 0 }, square)).toBe(true);
  });

  it("凹多边形", () => {
    const lShape: Point[] = [
      { x: 0, y: 0 },
      { x: 10, y: 0 },
      { x: 10, y: 5 },
      { x: 5, y: 5 },
      { x: 5, y: 10 },
      { x: 0, y: 10 },
    ];
    expect(isPointInPolygon({ x: 7, y: 2 }, lShape)).toBe(true);
    expect(isPointInPolygon({ x: 7, y: 8 }, lShape)).toBe(false); // 凹口外
  });
});

describe("点变换", () => {
  it("scalePoint / translatePoint", () => {
    expect(scalePoint({ x: 2, y: -3 }, 2)).toEqual({ x: 4, y: -6 });
    expect(translatePoint({ x: 1, y: 1 }, { x: 10, y: 20 })).toEqual({ x: 11, y: 21 });
  });
});
