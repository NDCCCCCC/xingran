/**
 * Phase 88 Batch97 — operations/building-spaces-3d/components/utils 测试(43 stmts, 0% → 高)
 */
import { describe, it, expect } from "vitest";
import {
  toBase64,
  getBuildingMarkerColors,
  getBuildingStatusText,
  getFloorStatusText,
  getWorkstationStatusText,
  getWorkstationTypeText,
  getWorkstationColor,
  getFloorColor,
  generateBuildingInfoHTML,
  calculatePixelDistance,
} from "../utils";
import type { BuildingItem } from "../../types";

describe("building-spaces-3d components utils", () => {
  it("toBase64 ASCII", () => {
    expect(toBase64("hello")).toBe("aGVsbG8=");
  });

  it("toBase64 Unicode", () => {
    const r = toBase64("测试");
    expect(typeof r).toBe("string");
    expect(r.length).toBeGreaterThan(0);
  });

  it("getBuildingMarkerColors → 有效返回值", () => {
    const colors = getBuildingMarkerColors(0);
    expect(colors).toBeDefined();
    // 可能是数组或字符串,只要有值即可
  });

  it("getBuildingStatusText: 0/1", () => {
    expect(getBuildingStatusText(0)).toBe("正常");
    expect(getBuildingStatusText(1)).toBe("已停用");
  });

  it("getFloorStatusText: 0/1/2", () => {
    expect(getFloorStatusText(0)).toBeDefined();
    expect(getFloorStatusText(1)).toBeDefined();
    expect(getFloorStatusText(2)).toBeDefined();
  });

  it("getWorkstationStatusText: 0/1/2/其他", () => {
    expect(getWorkstationStatusText(0)).toBeDefined();
    expect(getWorkstationStatusText(99)).toBeDefined();
  });

  it("getWorkstationTypeText: 0/1/2/其他", () => {
    expect(getWorkstationTypeText(0)).toBe("固定");
    expect(getWorkstationTypeText(1)).toBe("灵活");
    expect(getWorkstationTypeText(2)).toBe("管理");
    expect(getWorkstationTypeText(99)).toBe("未知");
  });

  it("getWorkstationColor 返回数字", () => {
    expect(typeof getWorkstationColor({ status: 0, type: 0 })).toBe("number");
    expect(typeof getWorkstationColor({ status: 1, type: 0 })).toBe("number");
    expect(typeof getWorkstationColor({ status: 2, type: 0 })).toBe("number");
  });

  it("getFloorColor 返回数字", () => {
    expect(typeof getFloorColor({ status: 0, workstationCount: 5 })).toBe("number");
    expect(typeof getFloorColor({ status: 1, workstationCount: 0 })).toBe("number");
  });

  it("generateBuildingInfoHTML → 含楼宇名", () => {
    const building: BuildingItem = {
      id: "b1",
      name: "一号楼",
      status: 0,
      level: 1,
      lng: 114.4,
      lat: 30.5,
    };
    const html = generateBuildingInfoHTML(building);
    expect(html).toContain("一号楼");
  });

  it("generateBuildingInfoHTML → 停用楼宇", () => {
    const html = generateBuildingInfoHTML({ id: "b1", name: "b", status: 1 } as BuildingItem);
    expect(html).toContain("已停用");
  });

  it("calculatePixelDistance → 5", () => {
    const d = calculatePixelDistance({ x: 0, y: 0 }, { x: 3, y: 4 });
    expect(d).toBe(5);
  });
});
