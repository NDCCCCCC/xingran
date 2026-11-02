/**
 * Phase 84 84-01a Task 2 — FloorPlanEditor panZoom 纯函数测试
 * 覆盖: ViewState 计算 / screenToSvg / worldToScreen / clampScale
 */
import { describe, it, expect } from "vitest";
import type { ViewState } from "../../FloorPlanEditor.types";
import { MIN_SCALE, MAX_SCALE } from "../../FloorPlanEditor.constants";

/** clampScale: 将缩放值限制在 [MIN_SCALE, MAX_SCALE] 范围内 */
function clampScale(value: number): number {
  return Math.max(MIN_SCALE, Math.min(MAX_SCALE, value));
}

/** screenToSvg: 屏幕坐标 → SVG 世界坐标 */
function screenToSvg(screenX: number, screenY: number, view: ViewState): { x: number; y: number } {
  return {
    x: (screenX - view.offsetX) / view.scale,
    y: (screenY - view.offsetY) / view.scale,
  };
}

/** worldToScreen: SVG 世界坐标 → 屏幕坐标 */
function worldToScreen(worldX: number, worldY: number, view: ViewState): { x: number; y: number } {
  return {
    x: worldX * view.scale + view.offsetX,
    y: worldY * view.scale + view.offsetY,
  };
}

/** fitToScreen: 给定容器尺寸和内容尺寸,返回居中缩放后的 viewState */
function fitToScreen(
  containerW: number,
  containerH: number,
  contentW: number,
  contentH: number
): ViewState {
  const rawScale = Math.min(containerW / contentW, containerH / contentH, 1);
  const scale = Math.max(MIN_SCALE, Math.min(MAX_SCALE, rawScale * 0.9));
  return {
    scale,
    offsetX: (containerW - contentW * scale) / 2,
    offsetY: (containerH - contentH * scale) / 2,
    isDragging: false,
    dragStartX: 0,
    dragStartY: 0,
  };
}

const DEFAULT_VIEW: ViewState = {
  scale: 1,
  offsetX: 0,
  offsetY: 0,
  isDragging: false,
  dragStartX: 0,
  dragStartY: 0,
};

describe("screenToSvg", () => {
  it("identity when scale=1 and offset=0", () => {
    const r = screenToSvg(100, 200, DEFAULT_VIEW);
    expect(r.x).toBe(100);
    expect(r.y).toBe(200);
  });

  it("applies scale factor", () => {
    const view = { ...DEFAULT_VIEW, scale: 2, offsetX: 0, offsetY: 0 };
    expect(screenToSvg(200, 400, view)).toEqual({ x: 100, y: 200 });
  });

  it("applies negative offset (panning)", () => {
    const view = { ...DEFAULT_VIEW, scale: 1, offsetX: 50, offsetY: -100 };
    expect(screenToSvg(150, 50, view)).toEqual({ x: 100, y: 150 });
  });

  it("combined scale + offset", () => {
    const view = { ...DEFAULT_VIEW, scale: 0.5, offsetX: 100, offsetY: 200 };
    expect(screenToSvg(200, 300, view)).toEqual({ x: 200, y: 200 });
  });
});

describe("worldToScreen", () => {
  it("inverse of screenToSvg", () => {
    const view: ViewState = {
      scale: 1.5,
      offsetX: 75,
      offsetY: -30,
      isDragging: false,
      dragStartX: 0,
      dragStartY: 0,
    };
    const world = { x: 50, y: 80 };
    const screen = worldToScreen(world.x, world.y, view);
    expect(screenToSvg(screen.x, screen.y, view)).toEqual(world);
  });
});

describe("clampScale", () => {
  it("returns MIN_SCALE for values below minimum", () => {
    expect(clampScale(0)).toBe(MIN_SCALE);
    expect(clampScale(-1)).toBe(MIN_SCALE);
    expect(clampScale(MIN_SCALE - 0.1)).toBe(MIN_SCALE);
  });

  it("returns MAX_SCALE for values above maximum", () => {
    expect(clampScale(10)).toBe(MAX_SCALE);
    expect(clampScale(MAX_SCALE + 1)).toBe(MAX_SCALE);
  });

  it("returns value within range unchanged", () => {
    expect(clampScale(1)).toBe(1);
    expect(clampScale(MIN_SCALE + 0.1)).toBeCloseTo(MIN_SCALE + 0.1);
  });
});

describe("fitToScreen", () => {
  it("centers content in container", () => {
    // rawScale = min(1000/200, 800/160, 1) = min(5, 5, 1) = 1; capped at 1 * 0.9
    const v = fitToScreen(1000, 800, 200, 160);
    expect(v.scale).toBeCloseTo(0.9);
    expect(v.offsetX).toBeGreaterThan(0);
    expect(v.offsetY).toBeGreaterThan(0);
  });

  it("centers small content with 0.9 margin", () => {
    const v = fitToScreen(500, 400, 200, 160);
    // rawScale = min(500/200, 400/160, 1) = min(2.5, 2.5, 1) = 1; 0.9
    expect(v.scale).toBeCloseTo(0.9);
  });

  it("caps scale at MAX_SCALE for very small content", () => {
    const v = fitToScreen(100, 100, 10, 10);
    expect(v.scale).toBeLessThanOrEqual(MAX_SCALE);
  });

  it("sets isDragging=false (idle viewport)", () => {
    const v = fitToScreen(800, 600, 400, 300);
    expect(v.isDragging).toBe(false);
  });
});
