/**
 * Phase 88 Batch205 — components/cad-editor/theme 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import {
  WORKSTATION_STATUS,
  DEFAULT_SNAP_DISTANCE,
  DEFAULT_GRID_SIZE,
  CAD_COLOR_THEME,
  getWallColor,
  getDoorColor,
  getWorkstationColor,
  DEFAULT_WALL_COLOR,
  DEFAULT_DOOR_COLOR,
  DEFAULT_WORKSTATION_COLOR,
} from "../theme";

describe("cad-editor/theme", () => {
  it("WORKSTATION_STATUS 3 态", () => {
    expect(WORKSTATION_STATUS.AVAILABLE).toBe(0);
    expect(WORKSTATION_STATUS.OCCUPIED).toBe(1);
    expect(WORKSTATION_STATUS.MAINTAIN).toBe(2);
  });

  it("DEFAULT_SNAP_DISTANCE = 10", () => {
    expect(DEFAULT_SNAP_DISTANCE).toBe(10);
  });

  it("DEFAULT_GRID_SIZE = 20", () => {
    expect(DEFAULT_GRID_SIZE).toBe(20);
  });

  it("CAD_COLOR_THEME 含 background/grid/wall/door/workstation", () => {
    expect(CAD_COLOR_THEME.background).toBeDefined();
    expect(CAD_COLOR_THEME.grid).toBeDefined();
    expect(CAD_COLOR_THEME.wall.default).toBeDefined();
    expect(CAD_COLOR_THEME.door.default).toBeDefined();
    expect(CAD_COLOR_THEME.workstation.available).toBeDefined();
  });

  it("getWallColor selected 优先", () => {
    expect(getWallColor({}, true)).toBe(CAD_COLOR_THEME.wall.selected);
  });

  it("getWallColor hovered 优先", () => {
    expect(getWallColor({}, false, true)).toBe(CAD_COLOR_THEME.wall.hover);
  });

  it("getWallColor wall.color 优先", () => {
    expect(getWallColor({ color: "#abc" })).toBe("#abc");
  });

  it("getWallColor exterior 类型", () => {
    expect(getWallColor({ type: "exterior" })).toBe(CAD_COLOR_THEME.wall.exterior);
  });

  it("getWallColor 默认", () => {
    expect(getWallColor({})).toBe(CAD_COLOR_THEME.wall.default);
  });

  it("getDoorColor selected/hovered/color/emergency/default", () => {
    expect(getDoorColor({}, true)).toBe(CAD_COLOR_THEME.door.selected);
    expect(getDoorColor({}, false, true)).toBe(CAD_COLOR_THEME.door.hover);
    expect(getDoorColor({ color: "#abc" })).toBe("#abc");
    expect(getDoorColor({ type: "emergency" })).toBe(CAD_COLOR_THEME.door.emergency);
    expect(getDoorColor({})).toBe(CAD_COLOR_THEME.door.default);
  });

  it("getWorkstationColor selected/hovered/color", () => {
    expect(getWorkstationColor({}, true)).toBe(CAD_COLOR_THEME.workstation.selected);
    expect(getWorkstationColor({}, false, true)).toBe(CAD_COLOR_THEME.workstation.hover);
    expect(getWorkstationColor({ color: "#abc" })).toBe("#abc");
  });

  it("getWorkstationColor status AVAILABLE", () => {
    expect(getWorkstationColor({ status: WORKSTATION_STATUS.AVAILABLE })).toBe(
      CAD_COLOR_THEME.workstation.available
    );
  });

  it("getWorkstationColor status OCCUPIED", () => {
    expect(getWorkstationColor({ status: WORKSTATION_STATUS.OCCUPIED })).toBe(
      CAD_COLOR_THEME.workstation.occupied
    );
  });

  it("getWorkstationColor status MAINTAIN", () => {
    expect(getWorkstationColor({ status: WORKSTATION_STATUS.MAINTAIN })).toBe(
      CAD_COLOR_THEME.workstation.maintain
    );
  });

  it("getWorkstationColor status 未知 → available", () => {
    expect(getWorkstationColor({ status: 99 })).toBe(CAD_COLOR_THEME.workstation.available);
  });

  it("DEFAULT_WALL_COLOR 等于 wall.default", () => {
    expect(DEFAULT_WALL_COLOR).toBe(CAD_COLOR_THEME.wall.default);
  });

  it("DEFAULT_DOOR_COLOR 等于 door.default", () => {
    expect(DEFAULT_DOOR_COLOR).toBe(CAD_COLOR_THEME.door.default);
  });

  it("DEFAULT_WORKSTATION_COLOR 等于 available", () => {
    expect(DEFAULT_WORKSTATION_COLOR).toBe(CAD_COLOR_THEME.workstation.available);
  });
});
