import { describe, expect, it } from "vitest";
import {
  BUILDING_3D_COLORS,
  MAP_MARKER_COLORS,
  WORKSTATION_STATUS_COLORS,
  WORKSTATION_TYPE_COLORS,
  getMapMarkerColor,
  getWorkstationStatusColor,
  getWorkstationTypeColor,
} from "./colors";

describe("three/colors 颜色常量", () => {
  it("工位状态色覆盖 0/1/2 三态且带中文 label", () => {
    expect(Object.keys(WORKSTATION_STATUS_COLORS).sort()).toEqual(["0", "1", "2"]);
    expect(WORKSTATION_STATUS_COLORS[0].label).toBe("空闲");
    expect(WORKSTATION_STATUS_COLORS[1].label).toBe("占用");
    expect(WORKSTATION_STATUS_COLORS[2].label).toBe("维护");
  });

  it("工位类型色覆盖 0/1/2 三类", () => {
    expect(WORKSTATION_TYPE_COLORS[0].label).toBe("固定");
    expect(WORKSTATION_TYPE_COLORS[1].label).toBe("灵活");
    expect(WORKSTATION_TYPE_COLORS[2].label).toBe("管理");
  });

  it("3D 模型色为 number 字面量（three.js Color 兼容）", () => {
    expect(typeof BUILDING_3D_COLORS.normal).toBe("number");
    expect(BUILDING_3D_COLORS.hover).toBe(0x764ba2);
  });
});

describe("getWorkstationStatusColor", () => {
  it("已知状态返回对应色，未知状态回落 0（空闲）", () => {
    expect(getWorkstationStatusColor(2)).toBe(WORKSTATION_STATUS_COLORS[2]);
    expect(getWorkstationStatusColor(99)).toBe(WORKSTATION_STATUS_COLORS[0]);
  });
});

describe("getWorkstationTypeColor", () => {
  it("已知类型返回对应色，未知类型回落 0（固定）", () => {
    expect(getWorkstationTypeColor(1)).toBe(WORKSTATION_TYPE_COLORS[1]);
    expect(getWorkstationTypeColor(99)).toBe(WORKSTATION_TYPE_COLORS[0]);
  });
});

describe("getMapMarkerColor", () => {
  it("city 标记返回城市色", () => {
    expect(getMapMarkerColor("city")).toBe(MAP_MARKER_COLORS.city);
  });

  it("building 正常返回绿色 normal（无 main 字段，色值在 normal 上）", () => {
    const color = getMapMarkerColor("building", 0);
    expect(color.normal).toBe(MAP_MARKER_COLORS.building.normal);
    expect(color).toEqual(MAP_MARKER_COLORS.building);
  });

  it("building status=1（停用）主色替换为灰色", () => {
    const color = getMapMarkerColor("building", 1);
    expect(color.main).toBe(MAP_MARKER_COLORS.building.stopped);
    expect(color.normal).toBe(MAP_MARKER_COLORS.building.normal); // 其余字段保留
  });

  it("building 未传 status 视为正常", () => {
    expect(getMapMarkerColor("building")).toEqual(MAP_MARKER_COLORS.building);
  });
});
