# Phase 114: map3d-clustering（地图聚类 O(n²) 消除） - Pattern Map

**Mapped:** 2026-09-12
**Files analyzed:** 6（2 新建 / 2 修改 / 2 删除）
**Analogs found:** 6 / 6

## File Classification

| 新建/修改文件 | Role | Data Flow | Closest Analog | Match Quality |
|--------------|------|-----------|----------------|---------------|
| `src/pages/operations/building-spaces-3d/cluster.ts` | utility | transform（纯数学：楼宇+像素 → 聚类组） | 同目录 `utils.ts`（纯函数风格）+ `HubeiMapGL.tsx:282-323`（行为契约来源） | exact |
| `src/pages/operations/building-spaces-3d/__tests__/cluster.test.ts` | test | transform 断言 | 同目录 `__tests__/utils.test.ts` | exact |
| `src/pages/operations/building-spaces-3d/components/HubeiMap.tsx` | component | event-driven + transform | 自身（待替换区段）+ `info-points/index.tsx:603-609`（D-05 useMemo 范本） | exact |
| `src/pages/operations/building-spaces-3d/components/HubeiMapGL.tsx` | component | event-driven + transform | 自身（待替换区段）+ `info-points/index.tsx:603-609` | exact |
| `src/pages/operations/building-spaces-3d/components/BuildingMarkers.tsx` | component（死代码） | request-response（@uiw 封装） | 无需类比 — 全库 grep 确认零消费方，直接删除 | n/a |
| `src/pages/operations/building-spaces-3d/components/CityMarkers.tsx` | component（死代码） | request-response（@uiw 封装） | 同上 | n/a |

---

## Pattern Assignments

### `cluster.ts`（新建 — utility, transform）

**Analog:** 同目录 `utils.ts`（文件组织 + 纯函数风格 + import type 纪律）

**Imports pattern**（`utils.ts:1-5`）——`verbatimModuleSyntax: true` 下 type-only import 必须写 `import type`：
```typescript
// 楼宇空间可视化辅助函数

import type { BuildingItem } from "./types";
import { WORKSTATION_STATUS_COLORS } from "./constants";
import { getBMapGL, type BMapGLNamespace, type BMapPoint } from "@/types/baidu-map";
```
注意：`cluster.ts` 是**纯数学**（RESEARCH 核心设计），只应 import `BuildingItem` 类型——**不得** import `@/types/baidu-map`（无 SDK 依赖才能 Vitest 直测）。

**纯函数导出风格**（`utils.ts:70-75`）——中文 JSDoc + `export function` + 结构化参数：
```typescript
/**
 * 计算两点之间的像素距离
 * @param point1 第一个点的像素坐标
 * @param point2 第二个点的像素坐标
 * @returns 像素距离
 */
export function pixelDistance(
  point1: { x: number; y: number },
  point2: { x: number; y: number }
): number {
  return Math.sqrt(Math.pow(point1.x - point2.x, 2) + Math.pow(point1.y - point2.y, 2));
}
```
**cluster.ts 直接复用这两个函数，不重写**：`pixelDistance`（utils.ts:70）+ `averagePixelPosition`（utils.ts:82-92，reduce 求均值返回 `{x,y}`）。cluster.ts 顶部 `import { pixelDistance, averagePixelPosition } from "./utils";`。

**行为契约来源 — 必须逐位复刻的旧算法**（`HubeiMapGL.tsx:282-323`，GL 版已消费 utils 共享函数，是两份副本中更"干净"的一份）：
```typescript
buildingsWithCoords.forEach((building) => {
  if (processedBuildings.has(building.id)) return;

  const buildingPoint = new BMapGL.Point(building.longitude!, building.latitude!);
  const buildingPixel = map.pointToOverlayPixel(buildingPoint);   // ← 剥离到调用方单遍预计算

  const overlappedBuildings: BuildingItem[] = [building];
  const pixels: Array<{ x: number; y: number }> = [buildingPixel];

  buildingsWithCoords.forEach((otherBuilding) => {
    if (otherBuilding.id === building.id || processedBuildings.has(otherBuilding.id)) {
      return;
    }
    const otherPixel = map.pointToOverlayPixel(otherPoint);       // ← O(n²) 灾根源，消除
    const distance = pixelDistance(buildingPixel, otherPixel);    // ← 保留 sqrt 表达式
    if (distance < CLUSTER_PIXEL_THRESHOLD) {                     // ← 严格小于
      overlappedBuildings.push(otherBuilding);
      pixels.push(otherPixel);
      processedBuildings.add(otherBuilding.id);
    }
  });

  const avgPixel = averagePixelPosition(pixels);
  const centerPoint = map.pixelToPoint?.(new BMapGL.Pixel(avgPixel.x, avgPixel.y)) || {
    lng: building.longitude!,
    lat: building.latitude!,
  };

  const cluster: ClusterGroup = {
    buildings: overlappedBuildings,
    centerPixel: avgPixel,
    clusterLng: centerPoint.lng || building.longitude!,   // ← falsy 兜底语义，原样保留（Pitfall 1）
    clusterLat: centerPoint.lat || building.latitude!,
  };
  clusterGroups.push(cluster);
  processedBuildings.add(building.id);
});
```
改造后形态：内层 `pointToOverlayPixel` 消失，改为从传入的 `ReadonlyMap<string, {x,y}>` 取像素；`pixelToPoint?.()` 投影改为可选回调参数 `toCenterLngLat?`。**回退链 `|| 锚点经纬度` 与 `centerPoint.lng ||` falsy 兜底原样保留**——坐标系统混用（pointToOverlayPixel 求均值 + pixelToPoint 逆变换）是既有行为，JSDoc 写明"故意保留，勿改"。

**ClusterGroup 类型**（`HubeiMap.tsx:23-28` 与 `HubeiMapGL.tsx:54-59` 现各持一份相同定义）——上移到 cluster.ts 导出，两组件删除本地副本：
```typescript
interface ClusterGroup {
  buildings: BuildingItem[];
  centerPixel: { x: number; y: number };
  clusterLng: number;
  clusterLat: number;
}
```

---

### `__tests__/cluster.test.ts`（新建 — test）

**Analog:** `__tests__/utils.test.ts`（同目录既有测试组织）

**文件头 + imports + fixture 风格**（`utils.test.ts:1-27`）——vitest 具名导入、中文 describe 嵌套、fixture 用 `as BuildingItem` 局部断言构造：
```typescript
/**
 * Phase 88 Batch95 — operations/building-spaces-3d/utils 单元测试
 */
import { describe, it, expect, vi } from "vitest";
import {
  toBase64,
  // ... 被测函数逐个具名导入
} from "../utils";
import type { BuildingItem } from "../types";

describe("building-spaces-3d utils", () => {
  describe("楼宇相关", () => {
    it("isBuildingStopped: status=1 → true", () => {
      const b = { status: 1 } as BuildingItem;
      expect(isBuildingStopped(b)).toBe(true);
    });
  });
});
```

**一致性测试的关键构件：**
1. **参考实现**（内嵌测试文件的旧 O(n²) 忠实移植）——逐行复刻上面 `HubeiMapGL.tsx:282-323` 的算法，把 `map.pointToOverlayPixel(point)` 换成 `pixels.get(point的lnglat标识)` 即可（测试里喂合成 Map，无需 mock 地图）。
2. **种子随机夹具**（`utils.test.ts:42-50` 已示范 `vi.useFakeTimers`；随机用 mulberry32 等种子 PRNG 保证 CI 可复现）。
3. **边界用例清单**（RESEARCH 已锁定）：距离恰为 40 → 不聚 / 39.99 → 聚；负像素坐标 floor 分桶；空数组 / 单点 / 1000 点同桶；toCenterLngLat 缺省 / undefined / lng=0 回退锚点。
4. **deep-equal 断言**：`expect(clusterBuildings(...)).toEqual(legacyCluster(...))`——注意必须全结构相等（簇分组 + 成员顺序 + 中心坐标），Pitfall 3 的顺序漂移只有 toEqual 能抓住。

**MAP3D-05 可选 cleanup spy 测试的 mock 风格**（`components/__tests__/BaiduMapScript.test.ts:12-21`）——直接操纵 `window.BMap / window.BMapGL` 全局 + `vi.fn()`：
```typescript
beforeEach(() => {
  delete (window as any).BMapGL;
  delete (window as any).BMap;
});
it("loadBaiduMapGLScript: 已加载 BMapGL → 立即 resolve", async () => {
  (window as any).BMapGL = {};
  await expect(loadBaiduMapGLScript("test-ak")).resolves.toBeUndefined();
});
```
若做组件级 cleanup 测试（render HubeiMapGL → mock 脚本加载 → spy removeEventListener → 同引用断言），模块 mock 用 `index.render.test.tsx:9-31` 的 `vi.mock` 风格：
```typescript
vi.mock("../components/HubeiMap", () => ({
  default: () => <div data-testid="hubei-map" />,
}));
```
配合 `@/test/utils/renderWithProviders` 渲染。

---

### `HubeiMap.tsx`（修改 — component, event-driven + transform）

**Analog:** 自身待替换区段 + D-05 范本 `info-points/index.tsx:603-609`

**① 渲染体 5 道 filter 收敛 useMemo — D-05 范本**（`info-points/index.tsx:603-609`，D-05 锁定的既有正确模式：`new Map` + forEach set + deps 数组）：
```typescript
// 创建工位ID到名称的映射
const workstationMap = useMemo(() => {
  const map = new Map<string, string>();
  workstationOptions.forEach((ws) => {
    map.set(ws.id, ws.name);
  });
  return map;
}, [workstationOptions]);
```
套用到 HubeiMap（对应 MAP3D-04 锁定形态，deps 严格 `[buildings]`）：
```typescript
const { level1, level2, withCoords } = useMemo(
  () => ({
    level1: buildings.filter((b) => b.level === 1),
    level2: buildings.filter((b) => b.level === 2),
    withCoords: buildings.filter((b) => b.longitude && b.latitude),
  }),
  [buildings]
);
```
被替换的 5 道全量 filter 现位于：`HubeiMap.tsx:664`（level1 Badge）、`667`（level2 Badge）、`672` + `676`（withCoords 计数与 >0 判断）、`705`（withCoords === 0 提示）。统计面板改读 `.length`。

**② 聚类双循环替换**（`HubeiMap.tsx:262-314`）——BMap 版与 GL 版的差异仅在 namespace 与"未消费 utils 共享函数"（内联 Math.sqrt 284-286 / 内联 reduce 296-297 / 内联阈值 260）。改造要点：
- 删除内联 `const CLUSTER_PIXEL_THRESHOLD = 40;`（260 行）→ 改 import `../constants` 的 `CLUSTER_PIXEL_THRESHOLD`（constants.ts:53，GL 版已是此用法，见 HubeiMapGL.tsx:21）。
- 内层循环删除 → 单遍预计算 + `clusterBuildings(...)` 调用（形态照 RESEARCH Pattern 1 的 HubeiMapGL 示例，namespace 换 BMap）。
- 254-257 的内联三目 `currentZoom === 10 ? buildings : buildings.filter((b) => b.level === 1)` 语义 = `filterBuildingsByZoom`（utils.ts:138-140），收敛时改为消费 useMemo 的 `level1`（`currentZoom === 10 ? buildings : level1`），**过滤语义不得变**（Pitfall 6：`=== 10` 原样保留，禁改 `>= 10`）。

**③ zoomend 监听 cleanup**（`HubeiMap.tsx:219-223` 现状）——handler 已具名但定义在 async initMap 内部，cleanup 闭包拿不到引用，必须提升到 effect 作用域：
```typescript
// 现状（219-223，initMap 内部）：
const handleZoomEnd = () => {
  const zoom = map.getZoom();
  setCurrentZoom(zoom);
};
map.addEventListener("zoomend", handleZoomEnd);
```
改造后（Pattern：handler 提升 effect 作用域 + cleanup 判空移除，事件签名见 `baidu-map.ts:59-60` `addEventListener(event, handler): BMapEventListener` / `removeEventListener(event, handler): void`）：
```typescript
const handleZoomEnd = () => { if (map) setCurrentZoom(map.getZoom()); };
// initMap 内： map.addEventListener("zoomend", handleZoomEnd);
// effect return: () => { if (map) map.removeEventListener("zoomend", handleZoomEnd); }
```
**保留** `await loadBaiduMapScript(...)` 之后的 `if (!mapRef.current) return;`（175 行）——异步 init 竞态守卫（Pitfall 5）。

---

### `HubeiMapGL.tsx`（修改 — component, event-driven + transform）

**Analog:** 自身待替换区段（与 HubeiMap 同型改造，以下仅列差异点）

**① 内联箭头监听 ×2**（`HubeiMapGL.tsx:150-151`）——比 HubeiMap 更差：内联箭头永远无法被 removeEventListener 移除（引用不等），必须提升为具名函数：
```typescript
// 现状：
map.addEventListener("zoomend", () => setCurrentZoom(map.getZoom()));
map.addEventListener("tiltend", () => setCurrentTilt(map.getTilt()));
```
改造：`handleZoomEnd` / `handleTiltEnd` 提升到 effect 作用域（闭包内经局部 `map` 变量访问实例），cleanup 同时移除两个事件。BMapGL 的 MAX_ZOOM=18（constants.ts:17），tiltend 是 GL 独有事件（2D 无 tilt）。

**② 聚类循环**（`HubeiMapGL.tsx:275-323`）——改造形态与 HubeiMap 完全同型（见上方 Pattern 1 示例），差异仅 namespace（`BMapGL.Point` / `BMapGL.Pixel`）。此版本已消费 `pixelDistance`（299）/ `averagePixelPosition`（308）/ `filterBuildingsByZoom`（275）/ `CLUSTER_PIXEL_THRESHOLD` 常量（21 行 import），cluster.ts 消费同一批共享函数即天然对齐。

**③ 渲染层保留**（`HubeiMapGL.tsx:332-415` `renderClusterMarker`）——两套 API 面不同（GL 有 3D 控件/倾斜/MARKER_COLORS 常量化），各自保留，仅聚类数据源收敛为共享函数输出。GL 版的常量/工具消费风格（`HubeiMapGL.tsx:17-41` 从 `../constants` 与 `../utils` 集中具名导入）就是 HubeiMap 收敛后的目标风格——HubeiMap 改造时把 `toBase64` 等内联副本切换为 utils 导入（可选收敛项，注意 toBase64 在 HubeiMap.tsx:50-56 有一份本地复制）。

**effect deps 保持**（`HubeiMapGL.tsx:330`）：`[mapLoaded, buildings, currentZoom]` 原样——聚类 effect 依赖缩放级别，不得并入 useMemo 收敛。

---

### `BuildingMarkers.tsx` / `CityMarkers.tsx`（删除 — 死组件）

**确认依据（实查）：**
- 两文件 `:6` 均为 `import { CustomOverlay, InfoWindow } from "@uiw/react-baidu-map";`——全库仅有的两处该包 import，删除后解锁 Phase 120 DEAD-02。
- 全库 grep `BuildingMarkers|CityMarkers` 仅命中两文件自身定义，零外部消费方。
- `BuildingMarkers.tsx:10` import `./utils` 的 `getBuildingMarkerColors`、`CityMarkers.tsx:10` import `./constants` 的 `MARKER_COLORS`——删除组件后这些 components/ 下导出沦为"仅测试引用"，**不顺带清理**（属 Phase 120 死代码批次）。

**删除雷区（最高优先级警告）：** `components/` 下存在**平行的第二套**同名基础设施文件，与父级极易混淆：

| 文件 | 命运 | 存活消费方 |
|------|------|-----------|
| `components/utils.ts` | **保留** | BuildingModel3D / AddressInput |
| `components/constants.ts` | **保留** | BuildingModel3D（嵌套 MARKER_COLORS 等场景常量） |
| `components/types.ts` | **保留** | CityGroup 另一定义等 |
| `components/__tests__/utils.test.ts` | **保留** | 测的是 components/utils.ts，不受影响 |
| `components/BuildingMarkers.tsx` | **删除** | — |
| `components/CityMarkers.tsx` | **删除** | — |
| 父级 `types.ts` / `utils.ts` / `constants.ts` | **保留**（本 phase 不动） | cluster.ts / 两组件 |

另：`global.d.ts:57`（`import type {} from HubeiMapGL`）不得误动。

---

## Shared Patterns

### 1. import type 纪律（verbatimModuleSyntax）
**Source:** `building-spaces-3d/utils.ts:3` / `tsconfig.app.json`
**Apply to:** `cluster.ts`、`cluster.test.ts`、两组件改动区
```typescript
import type { BuildingItem } from "./types";
```
type-only import 必须写 `import type`，否则 `type-check` 失败。

### 2. useMemo deps 稳定性（CLAUDE.md 明令）
**Source:** `info-points/index.tsx:603-609`（D-05 范本）+ 项目 CLAUDE.md useEffect 规则
**Apply to:** MAP3D-04 useMemo 与所有新 hook
- 渲染体 useMemo deps 严格 `[buildings]`（index.tsx 仅 mount 时 setBuildings 一次，引用稳定）。
- 聚类 effect 保持 `[mapLoaded, buildings, currentZoom]`。
- 禁止在 deps 放内联对象/数组；禁止新增 exhaustive-deps disable 注释压制告警。

### 3. 常量/工具单点消费（消除第四处复制）
**Source:** `constants.ts:53`（CLUSTER_PIXEL_THRESHOLD）+ `utils.ts:70/82/138`（pixelDistance / averagePixelPosition / filterBuildingsByZoom）
**Apply to:** 两组件 + cluster.ts
```typescript
import { CLUSTER_PIXEL_THRESHOLD } from "../constants";
import { pixelDistance, averagePixelPosition } from "../utils";
```
HubeiMap.tsx:260 的内联 `40`、284-286 的内联 sqrt、296-297 的内联 reduce、254-257 的内联过滤，全部收敛到上述单点。**禁再内联任何阈值/距离/均值表达式。**

### 4. 事件监听配对（引用相等才能移除）
**Source:** `baidu-map.ts:59-60` 类型声明 + 两组件现状反例
**Apply to:** 两组件 zoomend/tiltend
```typescript
addEventListener(event: string, handler: (e: BMapEvent) => void): BMapEventListener;
removeEventListener(event: string, handler: (e: BMapEvent) => void): void;
```
handler 提升至 effect 作用域（具名函数），cleanup `if (map)` 判空后用**同一引用**移除；`await loadScript` 后的 `if (!mapRef.current) return` 守卫保留。

### 5. 既有行为保持清单（纯重构红线）
**Source:** RESEARCH.md Pitfall 1/6 + 两组件实读
- `pointToOverlayPixel`（求均值）+ `pixelToPoint?.()`（逆变换）的坐标系混用**原样保留**，禁改 overlayPixelToPoint。
- 回退链 `|| { lng, lat }` 与 `centerPoint.lng ||` falsy 兜底语义保留。
- `zoom === 10` 三目语义保留（禁 `>= 10`）；index.tsx 的 level 全 2 映射不动。
- 距离比较保留 `pixelDistance(...) < threshold` 的 sqrt 表达式（禁平方优化）。
- 分桶铁律：bucketSize = threshold = 40；桶值存原始数组下标；3×3 邻域候选按**下标升序**扫描（输出逐位一致的三要素）。

## No Analog Found

无——全部 6 个文件均有直接类比或自身即契约来源。

| 文件 | Role | Data Flow | Reason |
|------|------|-----------|--------|
| （无） | | | 死组件删除无需类比；纯函数/测试/组件改造均有同目录 exact 类比 |

## Metadata

**Analog search scope:**
- `xingran-react-frontend/src/pages/operations/building-spaces-3d/`（全目录 36 文件含 __tests__ 与 components/）
- `xingran-react-frontend/src/pages/operations/info-points/index.tsx`（D-05 范本）
- `xingran-react-frontend/src/types/baidu-map.ts`（事件/坐标 API 类型声明）
- 全库 grep（死组件消费方验证）

**Files scanned:** 12（实读 9：HubeiMap.tsx 全文 / HubeiMapGL.tsx 1-420 / utils.ts 全文 / types.ts / constants.ts / utils.test.ts 1-120 / BuildingMarkers.tsx 1-60 / CityMarkers.tsx 1-50 / BaiduMapScript.test.ts / index.render.test.tsx 1-90 / info-points grep 区段 / baidu-map.ts grep 区段）

**Pattern extraction date:** 2026-09-12
