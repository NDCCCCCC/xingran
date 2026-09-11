/**
 * Phase 114 (MAP3D-03) — building-spaces-3d/cluster 聚类一致性单元测试
 *
 * 核心守护：clusterBuildings（40px 网格分桶版）与旧 O(n²) 锚点贪心实现
 * （HubeiMapGL.tsx:282-323 的忠实移植）输出逐位一致——簇分组 + 簇内成员顺序 + 中心坐标
 * （D-04 聚类结果一致性口径，toEqual 全结构相等才能抓住顺序漂移）。
 * 参考实现 legacyCluster 只活在测试文件里；生产代码保持单一实现。
 * 随机夹具全部使用 mulberry32 种子 PRNG，CI 可复现（禁止 Math.random）。
 */
import { describe, it, expect } from "vitest";
import { clusterBuildings, type ClusterGroup } from "../cluster";
import { pixelDistance, averagePixelPosition } from "../utils";
import type { BuildingItem } from "../types";

// ============ 参考实现：旧 O(n²) 锚点贪心（逐行复刻 HubeiMapGL.tsx:282-323） ============
// 唯一差异：像素来源从 map.pointToOverlayPixel(new Point(lng, lat)) 换成 pixels.get(building.id)
// （组件预计算循环对同一楼宇用同一 API 同一参数，结果逐位相同）。
// 不做任何"改进"——保留严格小于、保留 sqrt 距离表达式、保留两级回退链。
function legacyCluster(
  buildings: BuildingItem[],
  pixels: ReadonlyMap<string, { x: number; y: number }>,
  threshold: number,
  toCenterLngLat?: (pixel: { x: number; y: number }) => { lng: number; lat: number } | undefined
): ClusterGroup[] {
  const clusterGroups: ClusterGroup[] = [];
  const processedBuildings = new Set<string>();

  buildings.forEach((building) => {
    if (processedBuildings.has(building.id)) return;

    const buildingPixel = pixels.get(building.id)!;

    const overlappedBuildings: BuildingItem[] = [building];
    const clusterPixels: Array<{ x: number; y: number }> = [buildingPixel];

    buildings.forEach((otherBuilding) => {
      if (otherBuilding.id === building.id || processedBuildings.has(otherBuilding.id)) {
        return;
      }

      const otherPixel = pixels.get(otherBuilding.id)!;

      const distance = pixelDistance(buildingPixel, otherPixel);

      if (distance < threshold) {
        overlappedBuildings.push(otherBuilding);
        clusterPixels.push(otherPixel);
        processedBuildings.add(otherBuilding.id);
      }
    });

    const avgPixel = averagePixelPosition(clusterPixels);
    const centerPoint = toCenterLngLat?.(avgPixel) || {
      lng: building.longitude!,
      lat: building.latitude!,
    };

    clusterGroups.push({
      buildings: overlappedBuildings,
      centerPixel: avgPixel,
      clusterLng: centerPoint.lng || building.longitude!,
      clusterLat: centerPoint.lat || building.latitude!,
    });
    processedBuildings.add(building.id);
  });

  return clusterGroups;
}

// ============ 种子随机夹具 ============

/**
 * mulberry32 —— 标准 32 位种子 PRNG（CI 可复现，禁止 Math.random）
 */
function mulberry32(seed: number): () => number {
  let a = seed >>> 0;
  return () => {
    a = (a + 0x6d2b79f5) >>> 0;
    let t = a;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

/** 生成 n 个楼宇（id = "b{i}"，经纬度各点互异且非零——供回退断言分辨） */
function mkBuildings(n: number, rng: () => number): BuildingItem[] {
  const buildings: BuildingItem[] = [];
  for (let i = 0; i < n; i++) {
    buildings.push({
      id: `b${i}`,
      name: `楼宇${i}`,
      code: `c${i}`,
      cityCode: "076",
      cityName: "武汉",
      address: `测试地址${i}`,
      longitude: 100 + i,
      latitude: 30 + i * 0.1,
      level: 2,
      status: Math.floor(rng() * 2),
    } as BuildingItem);
  }
  return buildings;
}

/**
 * 生成稀疏团簇 + 散点混合的像素 Map（制造跨桶边界与重叠混合）：
 * 5-8 个随机簇心，成员在簇心 ±35px 内扰动（< 阈值 40 → 团簇倾向聚在一起但必然跨越桶边界）；
 * 另混 30% 全域随机散点（含负坐标——覆盖 Math.floor 负数分桶路径）。
 */
function mkRandomPixels(
  buildings: BuildingItem[],
  rng: () => number
): Map<string, { x: number; y: number }> {
  const pixels = new Map<string, { x: number; y: number }>();
  const centerCount = 5 + Math.floor(rng() * 4); // 5-8 个簇心
  const centers: Array<{ x: number; y: number }> = [];
  for (let i = 0; i < centerCount; i++) {
    centers.push({ x: rng() * 2000, y: rng() * 2000 });
  }
  buildings.forEach((b, i) => {
    if (i % 10 < 3) {
      // 30% 全域随机散点
      pixels.set(b.id, { x: rng() * 4000 - 1000, y: rng() * 4000 - 1000 });
    } else {
      const center = centers[i % centers.length];
      pixels.set(b.id, {
        x: center.x + (rng() * 2 - 1) * 35,
        y: center.y + (rng() * 2 - 1) * 35,
      });
    }
  });
  return pixels;
}

// ============ 测试 ============

describe("building-spaces-3d cluster（Phase 114 聚类一致性守护）", () => {
  describe("与旧 O(n²) 实现逐位一致（种子随机对照）", () => {
    const seedGroups = [
      { buildingsSeed: 42, pixelsSeed: 7 },
      { buildingsSeed: 2024, pixelsSeed: 99 },
      { buildingsSeed: 314, pixelsSeed: 27 },
    ];

    seedGroups.forEach(({ buildingsSeed, pixelsSeed }) => {
      it(`种子 buildings=${buildingsSeed}/pixels=${pixelsSeed}（500 点团簇+散点）输出完全相等`, () => {
        const buildings = mkBuildings(500, mulberry32(buildingsSeed));
        const pixels = mkRandomPixels(buildings, mulberry32(pixelsSeed));
        expect(clusterBuildings(buildings, pixels, 40, undefined)).toEqual(
          legacyCluster(buildings, pixels, 40)
        );
      });
    });

    it("带 toCenterLngLat 回调路径同样逐位一致", () => {
      const buildings = mkBuildings(300, mulberry32(11));
      const pixels = mkRandomPixels(buildings, mulberry32(13));
      const projector = (p: { x: number; y: number }) => ({ lng: p.x / 100, lat: p.y / 100 });
      expect(clusterBuildings(buildings, pixels, 40, projector)).toEqual(
        legacyCluster(buildings, pixels, 40, projector)
      );
    });
  });

  describe("边界：严格小于阈值", () => {
    it("距离恰为 40 → 不聚（2 簇各 1 成员）", () => {
      const buildings = mkBuildings(2, mulberry32(1));
      const pixels = new Map([
        ["b0", { x: 0, y: 0 }],
        ["b1", { x: 40, y: 0 }],
      ]);
      const groups = clusterBuildings(buildings, pixels, 40, undefined);
      expect(groups).toHaveLength(2);
      expect(groups[0].buildings.map((b) => b.id)).toEqual(["b0"]);
      expect(groups[1].buildings.map((b) => b.id)).toEqual(["b1"]);
    });

    it("距离 39 → 聚（1 簇 2 成员）", () => {
      const buildings = mkBuildings(2, mulberry32(1));
      const pixels = new Map([
        ["b0", { x: 0, y: 0 }],
        ["b1", { x: 39, y: 0 }],
      ]);
      const groups = clusterBuildings(buildings, pixels, 40, undefined);
      expect(groups).toHaveLength(1);
      expect(groups[0].buildings.map((b) => b.id)).toEqual(["b0", "b1"]);
    });

    it("斜向边界：(28,28) 同簇（sqrt(1568)≈39.60）；(29,29) 异簇（sqrt(1682)≈41.01）", () => {
      const buildings = mkBuildings(2, mulberry32(1));
      const pNear = new Map([
        ["b0", { x: 0, y: 0 }],
        ["b1", { x: 28, y: 28 }],
      ]);
      expect(clusterBuildings(buildings, pNear, 40, undefined)).toHaveLength(1);
      const pFar = new Map([
        ["b0", { x: 0, y: 0 }],
        ["b1", { x: 29, y: 29 }],
      ]);
      expect(clusterBuildings(buildings, pFar, 40, undefined)).toHaveLength(2);
    });

    it("负像素坐标：(-45,-10) 与 (-25,-5)（相距约 20.6）聚为一簇，且整体与参考实现一致（Math.floor 负数分桶正确）", () => {
      const buildings = mkBuildings(2, mulberry32(1));
      const pixels = new Map([
        ["b0", { x: -45, y: -10 }],
        ["b1", { x: -25, y: -5 }],
      ]);
      const groups = clusterBuildings(buildings, pixels, 40, undefined);
      expect(groups).toHaveLength(1);
      expect(groups[0].buildings.map((b) => b.id)).toEqual(["b0", "b1"]);
      expect(groups).toEqual(legacyCluster(buildings, pixels, 40));
    });
  });

  describe("边界：空 / 单点 / 全部同桶", () => {
    it("空数组 → []", () => {
      expect(clusterBuildings([], new Map(), 40, undefined)).toEqual([]);
    });

    it("单点 → 1 簇 1 成员，centerPixel 即该点像素", () => {
      const buildings = mkBuildings(1, mulberry32(1));
      const pixels = new Map([["b0", { x: 123.5, y: -77.25 }]]);
      const groups = clusterBuildings(buildings, pixels, 40, undefined);
      expect(groups).toHaveLength(1);
      expect(groups[0].buildings).toHaveLength(1);
      expect(groups[0].centerPixel).toEqual({ x: 123.5, y: -77.25 });
    });

    it("1000 点全部落在同一 40px 桶（锚点居中）→ 1 簇 1000 成员，无栈溢出且与参考实现一致", () => {
      const rng = mulberry32(5);
      const buildings = mkBuildings(1000, mulberry32(6));
      const pixels = new Map<string, { x: number; y: number }>();
      // 锚点 b0 置于桶 (0,0) 中心 (20,20)：桶内任意点距锚 ≤ 19.5√2 ≈ 27.6 < 40 → 全部被吸收
      pixels.set("b0", { x: 20, y: 20 });
      for (let i = 1; i < buildings.length; i++) {
        pixels.set(`b${i}`, { x: Math.floor(rng() * 40), y: Math.floor(rng() * 40) });
      }
      const groups = clusterBuildings(buildings, pixels, 40, undefined);
      expect(groups).toHaveLength(1);
      expect(groups[0].buildings).toHaveLength(1000);
      expect(groups).toEqual(legacyCluster(buildings, pixels, 40));
    });
  });

  describe("中心回退链（对齐旧代码两级兜底语义）", () => {
    const mkPair = () => {
      const buildings = mkBuildings(2, mulberry32(1));
      const pixels = new Map([
        ["b0", { x: 0, y: 0 }],
        ["b1", { x: 10, y: 0 }],
      ]);
      return { buildings, pixels };
    };

    it("不传 toCenterLngLat → clusterLng/clusterLat 回退锚点经纬度", () => {
      const { buildings, pixels } = mkPair();
      const groups = clusterBuildings(buildings, pixels, 40, undefined);
      expect(groups).toHaveLength(1);
      expect(groups[0].clusterLng).toBe(buildings[0].longitude);
      expect(groups[0].clusterLat).toBe(buildings[0].latitude);
    });

    it("回调返回 undefined → 同样回退锚点经纬度", () => {
      const { buildings, pixels } = mkPair();
      const groups = clusterBuildings(buildings, pixels, 40, () => undefined);
      expect(groups).toHaveLength(1);
      expect(groups[0].clusterLng).toBe(buildings[0].longitude);
      expect(groups[0].clusterLat).toBe(buildings[0].latitude);
    });

    it("falsy 半回退：回调返回 { lng: 0, lat: 5 } → 经度回退锚点（0 为 falsy 触发兜底），纬度保留 5", () => {
      // 锁定旧代码 centerPoint.lng || / centerPoint.lat || 的字段级兜底语义
      const { buildings, pixels } = mkPair();
      const groups = clusterBuildings(buildings, pixels, 40, () => ({ lng: 0, lat: 5 }));
      expect(groups).toHaveLength(1);
      expect(groups[0].clusterLng).toBe(buildings[0].longitude);
      expect(groups[0].clusterLat).toBe(5);
    });
  });

  describe("性能冒烟（宽松预算防 flake）", () => {
    it("1000 点合成夹具单次聚类 < 500ms（防退化回 O(n²) 地图 API 调用）", () => {
      const buildings = mkBuildings(1000, mulberry32(9));
      const pixels = mkRandomPixels(buildings, mulberry32(10));
      const start = performance.now();
      clusterBuildings(buildings, pixels, 40, undefined);
      const elapsed = performance.now() - start;
      expect(elapsed).toBeLessThan(500);
    });
  });
});
