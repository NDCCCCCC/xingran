/**
 * Phase 88 Batch384 — hooks/useWallDrawing 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook } from "@testing-library/react";
import type { Point } from "@/utils/cad/geometry";

vi.mock("@/utils/cad/geometry", () => ({
  snapToGrid: vi.fn((point: Point, _gridSize: number) => point),
  distance: vi.fn((_a: Point, _b: Point) => 10), // always > MIN_DRAW_DISTANCE
}));

import { useWallDrawing } from "../useWallDrawing";

describe("hooks/useWallDrawing", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("初始状态 isDrawing=false, drawPoints=[], previewPoint=null", () => {
    const { result } = renderHook(() => useWallDrawing());
    expect(result.current.isDrawing).toBe(false);
    expect(result.current.drawPoints).toEqual([]);
    expect(result.current.previewPoint).toBeNull();
  });

  it("返回所有方法", () => {
    const { result } = renderHook(() => useWallDrawing());
    expect(typeof result.current.startDrawing).toBe("function");
    expect(typeof result.current.addPoint).toBe("function");
    expect(typeof result.current.updatePreview).toBe("function");
    expect(typeof result.current.finishDrawing).toBe("function");
    expect(typeof result.current.cancelDrawing).toBe("function");
    expect(typeof result.current.undoPoint).toBe("function");
  });

  it("startDrawing 不抛错", () => {
    const { result } = renderHook(() => useWallDrawing());
    const pt: Point = { x: 100, y: 200 };
    expect(() => result.current.startDrawing(pt)).not.toThrow();
  });

  it("addPoint 未在绘制时不抛错", () => {
    const { result } = renderHook(() => useWallDrawing());
    const pt: Point = { x: 150, y: 250 };
    expect(() => result.current.addPoint(pt)).not.toThrow();
  });

  it("finishDrawing 未完成时不抛错", () => {
    const { result } = renderHook(() => useWallDrawing());
    expect(() => result.current.finishDrawing()).not.toThrow();
  });

  it("finishDrawing 完成后重置状态", () => {
    const { result } = renderHook(() => useWallDrawing());
    const pt: Point = { x: 100, y: 100 };
    result.current.startDrawing(pt);
    result.current.finishDrawing();
    expect(result.current.isDrawing).toBe(false);
    expect(result.current.drawPoints).toEqual([]);
  });

  it("cancelDrawing 重置所有状态", () => {
    const { result } = renderHook(() => useWallDrawing());
    const pt: Point = { x: 100, y: 100 };
    result.current.startDrawing(pt);
    result.current.cancelDrawing();
    expect(result.current.isDrawing).toBe(false);
    expect(result.current.drawPoints).toEqual([]);
    expect(result.current.previewPoint).toBeNull();
  });

  it("undoPoint 不抛错", () => {
    const { result } = renderHook(() => useWallDrawing());
    expect(() => result.current.undoPoint()).not.toThrow();
  });

  it("updatePreview 未在绘制时不生效", () => {
    const { result } = renderHook(() => useWallDrawing());
    expect(() => result.current.updatePreview({ x: 50, y: 50 })).not.toThrow();
  });

  it("onPointsChange 回调可传", () => {
    const onPointsChange = vi.fn();
    const { result } = renderHook(() => useWallDrawing({ onPointsChange }));
    const pt: Point = { x: 100, y: 100 };
    result.current.startDrawing(pt);
    expect(onPointsChange).toHaveBeenCalled();
  });

  it("onComplete 回调可传不抛错", () => {
    const onComplete = vi.fn();
    const { result } = renderHook(() => useWallDrawing({ onComplete }));
    expect(() => result.current.finishDrawing()).not.toThrow();
  });

  it("snapEnabled=false 不抛错", () => {
    const { result } = renderHook(() => useWallDrawing({ snapEnabled: false }));
    expect(() => result.current.startDrawing({ x: 10, y: 20 })).not.toThrow();
  });

  it("angleSnapEnabled=false 不抛错", () => {
    const { result } = renderHook(() => useWallDrawing({ angleSnapEnabled: false }));
    expect(() => result.current.startDrawing({ x: 10, y: 20 })).not.toThrow();
  });

  it("自定义 gridSize 不抛错", () => {
    const { result } = renderHook(() => useWallDrawing({ gridSize: 50 }));
    expect(() => result.current.startDrawing({ x: 25, y: 75 })).not.toThrow();
  });
});
