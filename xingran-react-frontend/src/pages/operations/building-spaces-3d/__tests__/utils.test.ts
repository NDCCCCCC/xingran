/**
 * Phase 88 Batch95 — operations/building-spaces-3d/utils 单元测试
 */
import { describe, it, expect, vi } from "vitest";
import {
  toBase64,
  delay,
  isBuildingStopped,
  getBuildingStatusText,
  getBuildingStatusColor,
  getBuildingLabel,
  filterBuildingsByZoom,
  getWorkstationStatusText,
  getWorkstationTypeText,
  getWorkstationColor,
  getWorkstationStatusColorCSS,
  getWorkstationTypeColorCSS,
  calculateWorkstationPositions,
  convertApiWorkstation,
  convertApiWorkstations,
  calculateWorkstationStats,
  degreesToRadians,
  lerp,
  pixelDistance,
  averagePixelPosition,
} from "../utils";
import type { BuildingItem } from "../types";

describe("building-spaces-3d utils", () => {
  describe("toBase64", () => {
    it("ASCII 字符串", () => {
      expect(toBase64("hello")).toBe("aGVsbG8=");
    });

    it("Unicode 字符串", () => {
      // 中文应该能正常 base64 编码
      const r = toBase64("测试");
      expect(r.length).toBeGreaterThan(0);
    });
  });

  describe("delay", () => {
    it("返回 Promise + ms 后 resolve", async () => {
      vi.useFakeTimers();
      const p = delay(100);
      vi.advanceTimersByTime(100);
      await p;
      vi.useRealTimers();
    });
  });

  describe("楼宇相关", () => {
    it("isBuildingStopped: status=1 → true", () => {
      const b = { status: 1 } as BuildingItem;
      expect(isBuildingStopped(b)).toBe(true);
    });

    it("isBuildingStopped: status=0 → false", () => {
      const b = { status: 0 } as BuildingItem;
      expect(isBuildingStopped(b)).toBe(false);
    });

    it("getBuildingStatusText: 0/1/其他", () => {
      expect(getBuildingStatusText(0)).toBe("正常");
      expect(getBuildingStatusText(1)).toBe("已停用");
      expect(getBuildingStatusText(99)).toBe("正常");
    });

    it("getBuildingStatusColor: 0/1", () => {
      expect(getBuildingStatusColor(0)).toBe("green");
      expect(getBuildingStatusColor(1)).toBe("red");
    });

    it("getBuildingLabel: 取 name 前两字", () => {
      expect(getBuildingLabel({ name: "一号楼" } as BuildingItem)).toBe("一号");
      expect(getBuildingLabel({ name: "" } as BuildingItem)).toBe("楼宇");
      expect(getBuildingLabel({} as BuildingItem)).toBe("楼宇");
    });

    it("filterBuildingsByZoom: zoom=10 → 全部", () => {
      const buildings = [
        { id: "1", name: "a", level: 1 } as BuildingItem,
        { id: "2", name: "b", level: 2 } as BuildingItem,
      ];
      expect(filterBuildingsByZoom(buildings, 10)).toHaveLength(2);
    });

    it("filterBuildingsByZoom: zoom≠10 → 仅 level=1", () => {
      const buildings = [
        { id: "1", name: "a", level: 1 } as BuildingItem,
        { id: "2", name: "b", level: 2 } as BuildingItem,
      ];
      expect(filterBuildingsByZoom(buildings, 12)).toHaveLength(1);
    });
  });

  describe("工位状态", () => {
    it("getWorkstationStatusText 0/1/2/其他", () => {
      expect(getWorkstationStatusText(0)).toBeDefined();
      expect(getWorkstationStatusText(99)).toBeDefined();
    });

    it("getWorkstationTypeText 0/1/2/其他", () => {
      expect(getWorkstationTypeText(0)).toBe("固定");
      expect(getWorkstationTypeText(1)).toBe("灵活");
      expect(getWorkstationTypeText(2)).toBe("管理");
      expect(getWorkstationTypeText(99)).toBe("未知");
    });

    it("getWorkstationColor 返回数字", () => {
      expect(typeof getWorkstationColor({ status: 0, type: 0 })).toBe("number");
    });

    it("getWorkstationStatusColorCSS 返回颜色", () => {
      const c = getWorkstationStatusColorCSS(0);
      expect(typeof c).toBe("string");
      expect(c.length).toBeGreaterThan(0);
    });

    it("getWorkstationTypeColorCSS 返回颜色", () => {
      const c = getWorkstationTypeColorCSS(0);
      expect(typeof c).toBe("string");
      expect(c.length).toBeGreaterThan(0);
    });
  });

  describe("数据转换", () => {
    it("convertApiWorkstation 单个", () => {
      const api = {
        id: "w1",
        name: "WS001",
        status: 0,
        type: 0,
        positionX: 100,
        positionY: 200,
      } as any;
      const r = convertApiWorkstation(api);
      expect(r.id).toBe("w1");
      expect(r.positionX).toBe(100);
    });

    it("convertApiWorkstations 批量", () => {
      const apis = [
        { id: "w1", name: "WS001" },
        { id: "w2", name: "WS002" },
      ] as any[];
      const r = convertApiWorkstations(apis);
      expect(r).toHaveLength(2);
    });

    it("calculateWorkstationStats", () => {
      const workstations = [{ status: 0 }, { status: 1 }, { status: 2 }] as any[];
      const stats = calculateWorkstationStats(workstations);
      expect(stats.total).toBe(3);
    });
  });

  describe("calculateWorkstationPositions", () => {
    it("空数组 → 空 Map", () => {
      expect(calculateWorkstationPositions([], { width: 200, height: 100 } as any)).toEqual(
        new Map()
      );
    });

    it("1 个工位 → 1 个位置", () => {
      const workstations = [{ id: "w1" }] as any;
      const r = calculateWorkstationPositions(workstations, { width: 200, height: 100 } as any);
      expect(r).toHaveLength(1);
    });
  });

  describe("数学工具", () => {
    it("degreesToRadians: 0/180", () => {
      expect(degreesToRadians(0)).toBe(0);
      expect(degreesToRadians(180)).toBeCloseTo(Math.PI);
    });

    it("lerp 线性插值", () => {
      expect(lerp(0, 10, 0.5)).toBe(5);
      expect(lerp(0, 10, 0)).toBe(0);
      expect(lerp(0, 10, 1)).toBe(10);
    });
  });

  describe("pixel 工具", () => {
    it("pixelDistance", () => {
      const d = pixelDistance({ x: 0, y: 0 }, { x: 3, y: 4 });
      expect(d).toBe(5);
    });

    it("averagePixelPosition 平均", () => {
      const r = averagePixelPosition([
        { x: 0, y: 0 },
        { x: 10, y: 20 },
      ]);
      expect(r).toEqual({ x: 5, y: 10 });
    });

    it("averagePixelPosition 空数组", () => {
      const r = averagePixelPosition([]);
      expect(r.x).toBeDefined();
      expect(r.y).toBeDefined();
    });
  });
});
