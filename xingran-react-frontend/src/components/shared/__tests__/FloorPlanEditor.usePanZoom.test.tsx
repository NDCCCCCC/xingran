/**
 * Phase 88 Batch427 — FloorPlanEditor.usePanZoom hook 测试
 */
import { describe, it, expect } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useRef } from "react";
import { usePanZoom } from "../FloorPlanEditor.panZoom";
import { DEFAULT_SCALE, MIN_SCALE, MAX_SCALE, GRID_SIZE } from "../FloorPlanEditor.constants";

describe("usePanZoom", () => {
  it("初始 viewState 使用 DEFAULT_SCALE", () => {
    const { result } = renderHook(() => {
      const ref = useRef<SVGSVGElement | null>(null);
      return usePanZoom({ containerRef: ref });
    });
    expect(result.current.viewState.scale).toBe(DEFAULT_SCALE);
    expect(result.current.viewState.offsetX).toBe(0);
    expect(result.current.viewState.offsetY).toBe(0);
    expect(result.current.viewState.isDragging).toBe(false);
  });

  it("zoomIn 增大 scale 但不超过 MAX_SCALE", () => {
    const { result } = renderHook(() => {
      const ref = useRef<SVGSVGElement | null>(null);
      return usePanZoom({ containerRef: ref });
    });
    act(() => result.current.zoomIn());
    expect(result.current.viewState.scale).toBeCloseTo(DEFAULT_SCALE + 0.1);
    // 继续 zoomIn 直到上限
    for (let i = 0; i < 50; i++) act(() => result.current.zoomIn());
    expect(result.current.viewState.scale).toBeLessThanOrEqual(MAX_SCALE);
  });

  it("zoomOut 减小 scale 但不低于 MIN_SCALE", () => {
    const { result } = renderHook(() => {
      const ref = useRef<SVGSVGElement | null>(null);
      return usePanZoom({ containerRef: ref });
    });
    act(() => result.current.zoomOut());
    expect(result.current.viewState.scale).toBeCloseTo(DEFAULT_SCALE - 0.1);
    for (let i = 0; i < 50; i++) act(() => result.current.zoomOut());
    expect(result.current.viewState.scale).toBeGreaterThanOrEqual(MIN_SCALE);
  });

  it("resetView 恢复默认", () => {
    const { result } = renderHook(() => {
      const ref = useRef<SVGSVGElement | null>(null);
      return usePanZoom({ containerRef: ref });
    });
    act(() => result.current.zoomIn());
    act(() => result.current.resetView());
    expect(result.current.viewState.scale).toBe(DEFAULT_SCALE);
    expect(result.current.viewState.offsetX).toBe(0);
    expect(result.current.viewState.offsetY).toBe(0);
    expect(result.current.viewState.isDragging).toBe(false);
  });

  it("handlePanStart 设置 isDragging 与 dragStart", () => {
    const { result } = renderHook(() => {
      const ref = useRef<SVGSVGElement | null>(null);
      return usePanZoom({ containerRef: ref });
    });
    act(() => result.current.handlePanStart(100, 200));
    expect(result.current.viewState.isDragging).toBe(true);
    expect(result.current.viewState.dragStartX).toBe(100);
    expect(result.current.viewState.dragStartY).toBe(200);
  });

  it("handlePanMove 更新 offsetX/offsetY", () => {
    const { result } = renderHook(() => {
      const ref = useRef<SVGSVGElement | null>(null);
      return usePanZoom({ containerRef: ref });
    });
    act(() => result.current.handlePanStart(100, 200));
    act(() => result.current.handlePanMove(150, 250));
    expect(result.current.viewState.offsetX).toBe(50);
    expect(result.current.viewState.offsetY).toBe(50);
  });

  it("handlePanEnd 关闭 isDragging", () => {
    const { result } = renderHook(() => {
      const ref = useRef<SVGSVGElement | null>(null);
      return usePanZoom({ containerRef: ref });
    });
    act(() => result.current.handlePanStart(10, 20));
    act(() => result.current.handlePanEnd());
    expect(result.current.viewState.isDragging).toBe(false);
  });

  it("screenToSvg 在 containerRef 为 null 时返回 {0,0}", () => {
    const { result } = renderHook(() => {
      const ref = useRef<SVGSVGElement | null>(null);
      return usePanZoom({ containerRef: ref });
    });
    const out = result.current.screenToSvg(100, 200);
    expect(out).toEqual({ x: 0, y: 0 });
  });

  it("fitToScreen 在 containerRef 为 null 时 no-op", () => {
    const { result } = renderHook(() => {
      const ref = useRef<SVGSVGElement | null>(null);
      return usePanZoom({ containerRef: ref });
    });
    const before = { ...result.current.viewState };
    act(() => result.current.fitToScreen());
    expect(result.current.viewState.scale).toBe(before.scale);
    expect(result.current.viewState.offsetX).toBe(before.offsetX);
  });

  it("handleWheel 不按 Ctrl 时按 GRID_SIZE 平移", () => {
    const { result } = renderHook(() => {
      const ref = useRef<SVGSVGElement | null>(null);
      return usePanZoom({ containerRef: ref });
    });
    const evt = {
      ctrlKey: false,
      metaKey: false,
      deltaX: 1,
      deltaY: 1,
      clientX: 0,
      clientY: 0,
      preventDefault: () => {},
    } as unknown as WheelEvent;
    act(() => result.current.handleWheel(evt));
    expect(result.current.viewState.offsetX).toBe(-GRID_SIZE);
    expect(result.current.viewState.offsetY).toBe(-GRID_SIZE);
  });

  it("handleWheel 不按 Ctrl 且 deltaX/Y=0 不移动", () => {
    const { result } = renderHook(() => {
      const ref = useRef<SVGSVGElement | null>(null);
      return usePanZoom({ containerRef: ref });
    });
    const evt = {
      ctrlKey: false,
      metaKey: false,
      deltaX: 0,
      deltaY: 0,
      clientX: 0,
      clientY: 0,
      preventDefault: () => {},
    } as unknown as WheelEvent;
    act(() => result.current.handleWheel(evt));
    expect(result.current.viewState.offsetX).toBe(0);
    expect(result.current.viewState.offsetY).toBe(0);
  });
});
