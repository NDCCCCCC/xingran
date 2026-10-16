/**
 * Phase 88 Batch99 — FloorPlanEditor.panZoom 测试(53 stmts, 0% → 高)
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { usePanZoom } from "../FloorPlanEditor.panZoom";

describe("usePanZoom", () => {
  const containerRef = { current: null as SVGSVGElement | null };

  it("初始化默认值", () => {
    const { result } = renderHook(() => usePanZoom({ containerRef }));
    expect(result.current.viewState.scale).toBe(1); // DEFAULT_SCALE
    expect(result.current.viewState.offsetX).toBe(0);
    expect(result.current.viewState.isDragging).toBe(false);
  });

  it("zoomIn → scale +0.1", () => {
    const { result } = renderHook(() => usePanZoom({ containerRef }));
    const before = result.current.viewState.scale;
    act(() => result.current.zoomIn());
    expect(result.current.viewState.scale).toBeCloseTo(before + 0.1);
  });

  it("zoomOut → scale -0.1", () => {
    const { result } = renderHook(() => usePanZoom({ containerRef }));
    const before = result.current.viewState.scale;
    act(() => result.current.zoomOut());
    expect(result.current.viewState.scale).toBeCloseTo(before - 0.1);
  });

  it("zoomIn 超过 MAX_SCALE 被 clamp", () => {
    const { result } = renderHook(() => usePanZoom({ containerRef }));
    for (let i = 0; i < 100; i++) act(() => result.current.zoomIn());
    expect(result.current.viewState.scale).toBeLessThanOrEqual(5);
  });

  it("zoomOut 低于 MIN_SCALE 被 clamp", () => {
    const { result } = renderHook(() => usePanZoom({ containerRef }));
    for (let i = 0; i < 100; i++) act(() => result.current.zoomOut());
    expect(result.current.viewState.scale).toBeGreaterThanOrEqual(0.1);
  });

  it("resetView → 还原默认", () => {
    const { result } = renderHook(() => usePanZoom({ containerRef }));
    act(() => result.current.zoomIn());
    act(() => result.current.resetView());
    expect(result.current.viewState.scale).toBe(1);
    expect(result.current.viewState.offsetX).toBe(0);
  });

  it("handlePanStart → isDragging=true + 记录起点", () => {
    const { result } = renderHook(() => usePanZoom({ containerRef }));
    act(() => result.current.handlePanStart(100, 50));
    expect(result.current.viewState.isDragging).toBe(true);
    expect(result.current.viewState.dragStartX).toBe(100);
    expect(result.current.viewState.dragStartY).toBe(50);
  });

  it("handlePanMove → 更新 offset", () => {
    const { result } = renderHook(() => usePanZoom({ containerRef }));
    act(() => result.current.handlePanStart(100, 50));
    act(() => result.current.handlePanMove(200, 100));
    expect(result.current.viewState.offsetX).toBe(100);
    expect(result.current.viewState.offsetY).toBe(50);
  });

  it("handlePanEnd → isDragging=false", () => {
    const { result } = renderHook(() => usePanZoom({ containerRef }));
    act(() => result.current.handlePanStart(100, 50));
    act(() => result.current.handlePanEnd());
    expect(result.current.viewState.isDragging).toBe(false);
  });

  it("screenToSvg: 无 scale 变化 → 返回 offset 调整", () => {
    const { result } = renderHook(() => usePanZoom({ containerRef }));
    const r = result.current.screenToSvg(100, 100);
    expect(r).toHaveProperty("x");
    expect(r).toHaveProperty("y");
  });

  it("fitToScreen 无容器 → 不操作", () => {
    const { result } = renderHook(() => usePanZoom({ containerRef }));
    expect(() => act(() => result.current.fitToScreen())).not.toThrow();
  });
});
