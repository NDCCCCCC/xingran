/**
 * Phase 88 Batch200 — pages/operations/floors/constants 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import {
  WORKSTATION_LAYOUT,
  DEFAULT_FLOOR_PLAN_CONFIG,
  STATUS_OPTIONS,
  DEFAULT_FORM_VALUES,
} from "../constants";

describe("operations/floors/constants", () => {
  it("WORKSTATION_LAYOUT 字段", () => {
    expect(WORKSTATION_LAYOUT.DEFAULT_WIDTH).toBe(160);
    expect(WORKSTATION_LAYOUT.DEFAULT_DEPTH).toBe(70);
    expect(WORKSTATION_LAYOUT.GAP).toBe(120);
    expect(WORKSTATION_LAYOUT.ITEMS_PER_ROW).toBe(5);
    expect(WORKSTATION_LAYOUT.START_X).toBe(100);
    expect(WORKSTATION_LAYOUT.START_Y).toBe(100);
  });

  it("DEFAULT_FLOOR_PLAN_CONFIG 字段", () => {
    expect(DEFAULT_FLOOR_PLAN_CONFIG.CANVAS_WIDTH).toBe(2000);
    expect(DEFAULT_FLOOR_PLAN_CONFIG.CANVAS_HEIGHT).toBe(2000);
    expect(DEFAULT_FLOOR_PLAN_CONFIG.GRID_SIZE).toBe(20);
  });

  it("STATUS_OPTIONS 共享 NORMAL_STOP_OPTIONS 2 项", () => {
    expect(STATUS_OPTIONS.length).toBeGreaterThanOrEqual(2);
  });

  it("DEFAULT_FORM_VALUES.status = 0", () => {
    expect(DEFAULT_FORM_VALUES.status).toBe(0);
  });
});
