/**
 * Phase 88 Batch371 — components/cad-editor/editorUtils 测试
 */
import { describe, it, expect } from "vitest";
import {
  EDITOR_CONSTANTS,
  snapToGrid,
  checkRectCollision,
  checkWorkstationCollision,
  checkWorkstationCollisionForDrag,
  pointToLineDistance,
  findNearbyWallNode,
  getElementType,
} from "../editorUtils";

describe("components/cad-editor/editorUtils", () => {
  it("EDITOR_CONSTANTS 默认值", () => {
    expect(EDITOR_CONSTANTS.DEFAULT_GRID_SIZE).toBe(20);
    expect(EDITOR_CONSTANTS.MIN_SCALE).toBe(0.1);
    expect(EDITOR_CONSTANTS.MAX_SCALE).toBe(5);
  });

  it("snapToGrid 整数倍吸附", () => {
    expect(snapToGrid(15, 20)).toBe(20);
    expect(snapToGrid(25, 20)).toBe(20);
    expect(snapToGrid(0, 20)).toBe(0);
  });

  it("snapToGrid 不同 gridSize", () => {
    expect(snapToGrid(33, 10)).toBe(30);
    expect(snapToGrid(33, 5)).toBe(35);
  });

  it("checkRectCollision 无碰撞 → false", () => {
    const r1 = { x: 0, y: 0, width: 10, height: 10 };
    const r2 = { x: 100, y: 100, width: 10, height: 10 };
    expect(checkRectCollision(r1, r2, 0)).toBe(false);
  });

  it("checkRectCollision 部分重叠 → true", () => {
    const r1 = { x: 0, y: 0, width: 10, height: 10 };
    const r2 = { x: 5, y: 5, width: 10, height: 10 };
    expect(checkRectCollision(r1, r2, 0)).toBe(true);
  });

  it("checkRectCollision 完全包含 → true", () => {
    const r1 = { x: 0, y: 0, width: 20, height: 20 };
    const r2 = { x: 5, y: 5, width: 5, height: 5 };
    expect(checkRectCollision(r1, r2, 0)).toBe(true);
  });

  it("checkRectCollision minSpacing 隔离 → false", () => {
    const r1 = { x: 0, y: 0, width: 10, height: 10 };
    const r2 = { x: 20, y: 20, width: 10, height: 10 };
    expect(checkRectCollision(r1, r2, 5)).toBe(false);
  });

  it("checkWorkstationCollision 列表内碰撞 → true", () => {
    const ws = { x: 0, y: 0, width: 10, height: 10 };
    const list = [{ id: "x", x: 5, y: 5, width: 10, height: 10 }];
    expect(checkWorkstationCollision(ws, list)).toBe(true);
  });

  it("checkWorkstationCollision 无碰撞 → false", () => {
    const ws = { x: 0, y: 0, width: 10, height: 10 };
    const list = [{ id: "x", x: 100, y: 100, width: 10, height: 10 }];
    expect(checkWorkstationCollision(ws, list)).toBe(false);
  });

  it("checkWorkstationCollision excludeId 跳过自己", () => {
    const ws = { x: 0, y: 0, width: 10, height: 10 };
    const list = [{ id: "self", x: 0, y: 0, width: 10, height: 10 }];
    expect(checkWorkstationCollision(ws, list, "self")).toBe(false);
  });

  it("checkWorkstationCollisionForDrag 不与自己碰撞", () => {
    const target = { id: "w1", x: 50, y: 50, width: 10, height: 10 };
    const original = { x: 50, y: 50, width: 10, height: 10 };
    const list = [target];
    expect(checkWorkstationCollisionForDrag(target, list, original)).toBe(false);
  });

  it("checkWorkstationCollisionForDrag 新位置碰撞 → true", () => {
    const target = { id: "w1", x: 5, y: 5, width: 10, height: 10 };
    const original = { x: 50, y: 50, width: 10, height: 10 };
    const list = [{ id: "w2", x: 5, y: 5, width: 10, height: 10 }];
    expect(checkWorkstationCollisionForDrag(target, list, original)).toBe(true);
  });

  it("pointToLineDistance 点到线段距离", () => {
    expect(pointToLineDistance({ x: 0, y: 5 }, { x: 0, y: 0 }, { x: 10, y: 0 })).toBe(5);
  });

  it("pointToLineDistance 点在线段上 → 0", () => {
    expect(pointToLineDistance({ x: 5, y: 0 }, { x: 0, y: 0 }, { x: 10, y: 0 })).toBe(0);
  });

  it("findNearbyWallNode 数组", () => {
    const walls: any[] = [
      {
        id: "w1",
        points: [
          { x: 10, y: 10 },
          { x: 100, y: 100 },
        ],
      },
    ];
    const result = findNearbyWallNode(walls, { x: 12, y: 12 }, 20);
    expect(result?.wallId).toBe("w1");
    expect(result?.pointIndex).toBe(0);
  });

  it("findNearbyWallNode 无临近 → null", () => {
    const walls: any[] = [
      {
        id: "w1",
        points: [
          { x: 10, y: 10 },
          { x: 100, y: 100 },
        ],
      },
    ];
    expect(findNearbyWallNode(walls, { x: 500, y: 500 }, 10)).toBeNull();
  });

  it("findNearbyWallNode 空数组 → null", () => {
    expect(findNearbyWallNode([], { x: 0, y: 0 }, 10)).toBeNull();
  });

  it("getElementType null → null", () => {
    expect(getElementType(null)).toBeNull();
  });

  it("getElementType 非对象 → null", () => {
    expect(getElementType("string")).toBeNull();
    expect(getElementType(42)).toBeNull();
  });
});
