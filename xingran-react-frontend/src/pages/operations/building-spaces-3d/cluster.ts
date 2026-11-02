/**
 * 楼宇像素聚类（40px 网格分桶 + 锚点贪心）—— HubeiMap / HubeiMapGL 唯一共享实现
 * （Phase 114 MAP3D-02/03）。
 *
 * 纯数学函数：像素投影（map.pointToOverlayPixel）由调用方在 effect 内单遍预计算为
 * Map<id, pixel> 后传入，本模块零 BMap/BMapGL/SDK/DOM 依赖，可被 Vitest 直接测试
 * （__tests__/cluster.test.ts 内嵌旧 O(n²) 参考实现做逐位一致性对照）。
 */

import type { BuildingItem } from "./types";
import { pixelDistance, averagePixelPosition } from "./utils";

/** 聚类群组 */
export interface ClusterGroup {
  buildings: BuildingItem[];
  centerPixel: { x: number; y: number };
  clusterLng: number;
  clusterLat: number;
}

/** 聚类中心像素 → 地理坐标（组件侧适配 map.pixelToPoint）；返回 undefined 时整体回退锚点经纬度 */
export type ClusterCenterProjector = (pixel: {
  x: number;
  y: number;
}) => { lng: number; lat: number } | undefined;

/**
 * 楼宇像素聚类（单遍网格分桶版）——与旧 O(n²) 锚点贪心实现输出逐位一致
 * （簇分组 + 簇内成员顺序 + 中心坐标，由参考实现对照测试锁定）。
 *
 * 行为契约（缺一不可）：
 *  - 锚点贪心：按数组顺序取首个未处理楼宇为锚，吸收全部距锚 < threshold（严格小于）的未处理楼宇；
 *  - 簇内成员顺序 = 原数组顺序；簇的输出顺序 = 锚点出现顺序；
 *  - 距离比较保留 sqrt 表达式 pixelDistance(a, b) < threshold（禁平方优化——浮点边界语义
 *    须与旧实现逐位一致）；
 *  - 中心 = 成员像素算术平均（averagePixelPosition）→ toCenterLngLat 回调；回调缺失/返回
 *    undefined/返回值字段为 falsy（如 lng=0）时逐级回退锚点（cluster.buildings[0]）的
 *    longitude/latitude（对齐旧代码 `centerPoint.lng || building.longitude!` 两级回退链）。
 *
 * 坐标系统说明（故意保留，勿改）：调用方继续用 pointToOverlayPixel 求均值像素 +
 * pixelToPoint 逆变换转回经纬度——严格说应配对 overlayPixelToPoint，但该混用是既有行为
 * （RESEARCH Pitfall 1），"纠正"会改变 clusterLng/clusterLat → 聚类标记视觉位置漂移。
 *
 * 逐位一致三要素（分桶正确性前提）：
 *  1. 桶大小 = threshold（铁律 bucketSize ≥ threshold；取等号时 |Δx|<40 且 |Δy|<40
 *     蕴含桶索引差 ≤ 1，3×3 邻域完备）；
 *  2. 每个锚点收集 3×3 邻域候选；
 *  3. 桶值存原始数组下标、候选按下标升序扫描（复刻旧实现簇内成员顺序）。
 *
 * @param buildings 待聚类楼宇（原数组顺序即成员顺序基准）
 * @param pixels 楼宇 id → 像素坐标映射（调用方单遍预计算；前置条件：覆盖待聚类楼宇，
 *               缺失项跳过——防御路径，正常调用不应触达）
 * @param threshold 聚类像素阈值（如 CLUSTER_PIXEL_THRESHOLD = 40），同时用作分桶桶大小
 * @param toCenterLngLat 中心像素 → 经纬度投影回调（组件侧 map.pixelToPoint 适配）；可省略
 * @returns 聚类组数组（顺序 = 锚点出现顺序）
 */
export function clusterBuildings(
  buildings: BuildingItem[],
  pixels: ReadonlyMap<string, { x: number; y: number }>,
  threshold: number,
  toCenterLngLat?: ClusterCenterProjector
): ClusterGroup[] {
  // ① 单遍分桶：key = "floor(x/threshold),floor(y/threshold)"（Math.floor 对负坐标同样正确）；
  //    桶值 push 原始下标 i（天然按插入序 = 原数组序）
  const buckets = new Map<string, number[]>();
  for (let i = 0; i < buildings.length; i++) {
    const p = pixels.get(buildings[i].id);
    if (!p) continue; // 防御路径：pixels 未覆盖的楼宇跳过（见 JSDoc 前置条件）
    const key = Math.floor(p.x / threshold) + "," + Math.floor(p.y / threshold);
    const bucket = buckets.get(key);
    if (bucket) {
      bucket.push(i);
    } else {
      buckets.set(key, [i]);
    }
  }

  const clusterGroups: ClusterGroup[] = [];
  const processed = new Set<string>();

  // ② 主扫描：按原数组顺序取锚点，收集 3×3 邻域候选（下标升序）逐个贪心吸收
  for (let i = 0; i < buildings.length; i++) {
    const anchor = buildings[i];
    if (processed.has(anchor.id)) continue;

    const anchorPixel = pixels.get(anchor.id);
    if (!anchorPixel) continue; // 防御路径与分桶跳过逻辑保持一致

    const bx = Math.floor(anchorPixel.x / threshold);
    const by = Math.floor(anchorPixel.y / threshold);
    const neighbors: number[] = [];
    for (let dx = -1; dx <= 1; dx++) {
      for (let dy = -1; dy <= 1; dy++) {
        const bucket = buckets.get(bx + dx + "," + (by + dy));
        if (bucket) {
          for (const j of bucket) {
            neighbors.push(j);
          }
        }
      }
    }
    neighbors.sort((a, b) => a - b); // 升序——复刻旧实现簇内成员顺序的关键

    const overlappedBuildings: BuildingItem[] = [anchor];
    const clusterPixels: Array<{ x: number; y: number }> = [anchorPixel];

    for (const j of neighbors) {
      if (j === i) continue;
      const candidate = buildings[j];
      if (processed.has(candidate.id)) continue;
      if (pixelDistance(anchorPixel, pixels.get(candidate.id)!) < threshold) {
        overlappedBuildings.push(candidate);
        clusterPixels.push(pixels.get(candidate.id)!);
        processed.add(candidate.id);
      }
    }

    // ③ 中心与回退：均值像素 → 回调投影；字段级 falsy 兜底对齐旧代码两级回退链
    const avgPixel = averagePixelPosition(clusterPixels);
    const centerPoint = toCenterLngLat?.(avgPixel);
    clusterGroups.push({
      buildings: overlappedBuildings,
      centerPixel: avgPixel,
      clusterLng: centerPoint?.lng || anchor.longitude!,
      clusterLat: centerPoint?.lat || anchor.latitude!,
    });
    processed.add(anchor.id);
  }

  return clusterGroups;
}
