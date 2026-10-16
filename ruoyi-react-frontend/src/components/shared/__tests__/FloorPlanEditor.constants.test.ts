/**
 * Phase 88 Batch236 — components/shared/FloorPlanEditor.constants 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

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

describe("shared/FloorPlanEditor.constants", () => {
  it("GRID_SIZE = 20", () => {
    expect(GRID_SIZE).toBe(20);
  });

  it("DEFAULT_SCALE = 1", () => {
    expect(DEFAULT_SCALE).toBe(1);
  });

  it("MIN_SCALE = 0.25", () => {
    expect(MIN_SCALE).toBe(0.25);
  });

  it("MAX_SCALE = 3", () => {
    expect(MAX_SCALE).toBe(3);
  });

  it("TOOLBAR_HEIGHT = 48", () => {
    expect(TOOLBAR_HEIGHT).toBe(48);
  });

  it("DEFAULT_WORKSTATION_WIDTH = 80", () => {
    expect(DEFAULT_WORKSTATION_WIDTH).toBe(80);
  });

  it("DEFAULT_WORKSTATION_HEIGHT = 60", () => {
    expect(DEFAULT_WORKSTATION_HEIGHT).toBe(60);
  });

  it("ZOOM_LEVELS 7 级别", () => {
    expect(ZOOM_LEVELS.length).toBe(7);
    expect(ZOOM_LEVELS[0].value).toBe(0.25);
    expect(ZOOM_LEVELS[6].value).toBe(3);
  });
});
