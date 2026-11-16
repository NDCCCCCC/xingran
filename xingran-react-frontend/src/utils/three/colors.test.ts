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

  // handoff R13：静态表未命中的新增类型按字典 listClass 配色
  const dictItems = [
    { id: "1", dictLabel: "固定", dictValue: "0", listClass: "primary" },
    { id: "2", dictLabel: "灵活", dictValue: "1", listClass: "success" },
    { id: "3", dictLabel: "管理", dictValue: "2", listClass: "processing" },
    { id: "4", dictLabel: "虚拟", dictValue: "3", listClass: "warning" },
    { id: "5", dictLabel: "无色类", dictValue: "4" },
  ] as const;

  it("新增类型(type=3)按字典 listClass=warning 取色，label 走字典", () => {
    const color = getWorkstationTypeColor(3, dictItems);
    expect(color.main).not.toBe(WORKSTATION_TYPE_COLORS[0].main); // 不再回落 0 号蓝
    expect(color.label).toBe("虚拟");
    expect(color.bg).toBeDefined();
    expect(color.border).toBeDefined();
  });

  it("新增类型无 listClass → 仍回落 0 号色（既有行为）", () => {
    const color = getWorkstationTypeColor(4, dictItems);
    expect(color.main).toBe(WORKSTATION_TYPE_COLORS[0].main);
    expect(color.label).toBe("无色类");
  });

  it("静态表命中(type=0/1/2)行为不变（引用相等语义保留）", () => {
    expect(getWorkstationTypeColor(0)).toBe(WORKSTATION_TYPE_COLORS[0]);
    expect(getWorkstationTypeColor(2)).toBe(WORKSTATION_TYPE_COLORS[2]);
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
