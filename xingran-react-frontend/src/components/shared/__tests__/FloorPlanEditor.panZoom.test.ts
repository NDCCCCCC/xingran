/**
 * Phase 88 Batch425 — FloorPlanEditor constants/panZoom 测试
 */
import { describe, it, expect } from "vitest";

import {
  GRID_SIZE,
  DEFAULT_SCALE,
  MIN_SCALE,
  MAX_SCALE,
  TOOLBAR_HEIGHT,
  DEFAULT_WORKSTATION_WIDTH,
  DEFAULT_WORKSTATION_HEIGHT,
  ZOOM_LEVELS,
} from "../FloorPlanEditor.constants";

describe("FloorPlanEditor.constants", () => {
  it("GRID_SIZE 数值", () => expect(typeof GRID_SIZE).toBe("number"));
  it("DEFAULT_SCALE 数值", () => expect(typeof DEFAULT_SCALE).toBe("number"));
  it("MIN_SCALE < MAX_SCALE", () => {
    expect(MIN_SCALE).toBeLessThan(MAX_SCALE);
  });
  it("TOOLBAR_HEIGHT 数值", () => expect(typeof TOOLBAR_HEIGHT).toBe("number"));
  it("DEFAULT_WORKSTATION_WIDTH 数值", () => expect(typeof DEFAULT_WORKSTATION_WIDTH).toBe("number"));
  it("DEFAULT_WORKSTATION_HEIGHT 数值", () => expect(typeof DEFAULT_WORKSTATION_HEIGHT).toBe("number"));
  it("ZOOM_LEVELS 数组", () => {
    expect(Array.isArray(ZOOM_LEVELS)).toBe(true);
    expect(ZOOM_LEVELS.length).toBeGreaterThan(0);
  });
});