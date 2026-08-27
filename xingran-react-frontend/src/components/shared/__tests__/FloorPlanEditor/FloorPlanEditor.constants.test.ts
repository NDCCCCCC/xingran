/**
 * Phase 84 84-01a Task 2 — FloorPlanEditor 静态常量与类型断言
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
} from "../../FloorPlanEditor.constants";

describe("FloorPlanEditor.constants", () => {
  it("exports numeric constants with valid ranges", () => {
    expect(GRID_SIZE).toBe(20);
    expect(DEFAULT_SCALE).toBe(1);
    expect(MIN_SCALE).toBe(0.25);
    expect(MAX_SCALE).toBe(3);
    expect(TOOLBAR_HEIGHT).toBe(48);
    expect(DEFAULT_WORKSTATION_WIDTH).toBe(80);
    expect(DEFAULT_WORKSTATION_HEIGHT).toBe(60);
  });

  it("ZOOM_LEVELS contains expected entries", () => {
    expect(ZOOM_LEVELS).toHaveLength(7);
    expect(ZOOM_LEVELS[0]).toEqual({ label: "25%", value: 0.25 });
    expect(ZOOM_LEVELS[3]).toEqual({ label: "100%", value: 1 });
    expect(ZOOM_LEVELS[6]).toEqual({ label: "300%", value: 3 });
  });

  it("zoom levels are monotonically increasing", () => {
    for (let i = 1; i < ZOOM_LEVELS.length; i++) {
      expect(ZOOM_LEVELS[i].value).toBeGreaterThan(ZOOM_LEVELS[i - 1].value);
    }
  });

  it("MIN_SCALE < MAX_SCALE (valid zoom range)", () => {
    expect(MIN_SCALE).toBeLessThan(MAX_SCALE);
  });

  it("GRID_SIZE divides evenly into DEFAULT_WORKSTATION dimensions", () => {
    expect(DEFAULT_WORKSTATION_WIDTH % GRID_SIZE).toBe(0);
    expect(DEFAULT_WORKSTATION_HEIGHT % GRID_SIZE).toBe(0);
  });
});
