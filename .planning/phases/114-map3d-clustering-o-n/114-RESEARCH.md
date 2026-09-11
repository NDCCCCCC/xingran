# Phase 114: map3d-clustering（地图聚类 O(n²) 消除） - Research

**Researched:** 2026-09-12
**Domain:** 前端纯性能重构 — 百度地图（BMap/BMapGL）点位聚类算法单遍化 + 像素网格分桶 + 双实现合并共享 + React 渲染体收敛
**Confidence:** HIGH（全库内部重构，零新增依赖，关键技术事实均经代码库验证 + 百度官方文档佐证）

## Summary

本 phase 是 H-2（全库最重 JS 热点）的修复：`HubeiMap.tsx`（BMap 2D 版）与 `HubeiMapGL.tsx`（BMapGL WebGL 版）各持有一份近似复制的 O(n²) 聚类实现——外层 forEach 内嵌套 forEach，内层对每个候选楼宇执行 `new BMap(Point)` + `map.pointToOverlayPixel()`（地图 API 调用），n=1000 时最多 100 万对比较 × 约 200 万次地图 API 调用，每次 zoomend 触发整段重跑导致秒级主线程卡死。`index.tsx:18` `PAGE_SIZE = 1000` 全量拉取放大了影响。

修复路径已在 REQUIREMENTS.md 量化锁定：单遍 `Map<id,pixel>` 预计算（n 次 API 调用替代 n² 次）+ 40px 像素网格分桶（spatial hash，3×3 邻域查询，典型近 O(n)）+ 两份实现合并为共享纯函数。核心设计洞察：**把"像素投影"从聚类算法中剥离**——组件适配层单遍计算 `Map<id, {x,y}>` 后传入共享函数，聚类本体变成纯数学（无 BMap/BMapGL 依赖），既天然可单测，又同时服务两套地图 API。现有算法是"锚点贪心"（首个未处理楼宇为锚，吸收全部距锚 <40px 的未处理楼宇），分桶版只要桶大小 ≥ 阈值 + 3×3 邻域 + 按原始数组下标升序扫描，输出可与旧实现逐位一致（聚类分组、成员顺序、中心坐标全部相同），满足 D-04 聚类结果一致性口径。

附带的四项收尾均有明确落点：HubeiMap 渲染体 5 道全量 filter（664/667/672/676/705，hover 即重算）收敛为一个 useMemo（deps `[buildings]`）；zoomend/tiltend 监听补 effect cleanup（注意 handler 引用一致性——HubeiMapGL 当前用内联箭头函数，必须提升为具名函数才能 remove）；死组件 `BuildingMarkers.tsx` / `CityMarkers.tsx` 确认零消费方（唯一 `@uiw/react-baidu-map` import 方），删除后为 Phase 120 DEAD-02 依赖移除解锁。本 phase **零新增 npm 包**。

**Primary recommendation:** 新建 `building-spaces-3d/cluster.ts` 导出纯函数 `clusterBuildings(buildings, pixels, threshold, toCenterLngLat?)`（pixels 为调用方单遍预计算的 `ReadonlyMap<id, {x,y}>`），两组件 effect 内保留一次性 `pointToOverlayPixel` 预计算循环 + 一次性 `pixelToPoint` 中心回调；配新建参考实现对照（旧 O(n²) 算法副本内嵌测试文件）+ 种子随机夹具的逐位一致性单元测试。

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
（无用户锁定决策——本 phase 为基础设施/纯性能 phase，全部实现选择归 Claude 自由裁量）

### Claude's Discretion
All implementation choices are at Claude's discretion — pure infrastructure/performance phase. Use REQUIREMENTS.md MAP3D-01~06 条目、ROADMAP success criteria 与 D-05 复用范本（info-points:603 Map 索引风格）指导实现。

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope。
另：@uiw/react-baidu-map 依赖移除明确不在本 phase（Phase 120 DEAD-02，依赖本 phase MAP3D-06 先行）。
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| MAP3D-01 | 聚类消除 O(n²)——单遍预计算 `Map<id,pixel>`，内层只做像素距离比较，消除内层 `new BMapGL.Point()` + `pointToOverlayPixel` | 两份实现精确位置与差异已盘点（HubeiMap.tsx:267-314 / HubeiMapGL.tsx:282-323）；共享函数设计将像素投影剥离到调用方单遍预计算循环（每组件恰好 1 处 API 调用点），聚类本体纯数学 |
| MAP3D-02 | 40px 像素网格分桶（spatial hash）降为近 O(n)；n=1000 缩放切换无主线程长任务 | 桶 key 公式、3×3 邻域正确性证明（桶大小=阈值=40 时 |Δ|<40 ⇒ 桶索引差 ≤1）、扫描顺序保持策略已给出；`CLUSTER_PIXEL_THRESHOLD = 40` 已存在于 constants.ts:53 |
| MAP3D-03 | 两份复制聚类算法合并为共享工具函数（单一实现 + 单元测试） | 共享函数签名与落位建议（`building-spaces-3d/cluster.ts`）；测试策略（参考实现对照 + 种子随机 + 边界用例）已设计；测试目录约定 `__tests__/` 已确认 |
| MAP3D-04 | HubeiMap 渲染体 5 道全量 filter（664/667/672/676/705）收敛为 useMemo 一次计算 `{level1, level2, withCoords}`（deps `[buildings]`） | 5 处 filter 逐行确认（664 level1 / 667 level2 / 672+676+705 withCoords 三连）；`buildings` 引用稳定性已验证（index.tsx 仅 mount 时 setBuildings 一次，deps `[buildings]` 安全）；`filterBuildingsByZoom`（utils.ts:138-140）可改为消费 useMemo 的 level1 数组 |
| MAP3D-05 | map 级事件监听（zoomend/tiltend）补 effect cleanup（removeEventListener） | HubeiMap.tsx:223（具名 handleZoomEnd）+ HubeiMapGL.tsx:150-151（内联箭头 ×2，需提升具名）；`BMapEventEmitter.removeEventListener` 类型已声明（baidu-map.ts:60）；异步 init 竞态已有 `if (!mapRef.current) return` 后置守卫兜底（unmount 时 ref 被 React 置 null） |
| MAP3D-06 | 死组件 BuildingMarkers.tsx / CityMarkers.tsx 删除 | 全库 grep 确认零消费方；二者是 `@uiw/react-baidu-map` 仅有的两处 import（解锁 Phase 120）；`components/utils.ts` / `components/constants.ts` / `components/types.ts` 有存活消费方（BuildingModel3D/AddressInput）不得连带删除 |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

- **GSD workflow:** 文件改动须走 GSD 命令入口（本 phase 已在 `/gsd:plan-phase 114` 流程内，合规）。
- **前端测试:** Vitest，运行命令 `npx vitest run`；纯性能重构以现有测试零回归为准（D-04 口径）。
- **复用范本 D-05:** `info-points/index.tsx:603-609` useMemo + Map 索引风格（已读取确认：`new Map` + forEach set + deps 数组）。
- **useEffect 纪律:** 依赖必须稳定（对象/数组需 useMemo），防止无限请求循环——本 phase 的 useMemo/useCallback 改造直接受此约束。
- **response_language:** Chinese。
- **作用域约束（Scope Constrainment）:** 修复仅限报告的问题——不要顺带"修复"坐标系统混用（见 Pitfall 1）、level 全为 2 的过滤怪癖（见 Pitfall 6）等既有行为。

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| 点位→像素投影（pointToOverlayPixel） | Browser/Client（地图 SDK 适配层，组件 effect 内） | — | 依赖活地图实例与 BMap/BMapGL 构造器，只能留在组件侧；单遍预计算后不再进入内层循环 |
| 聚类计算（贪心分组 + 中心均值） | Client 纯函数（`building-spaces-3d/cluster.ts` 共享 util） | — | 输入输出全是普通对象（{x,y}/{lng,lat}/BuildingItem），无 SDK/DOM 依赖 → 可 Vitest 直测，两地图版本唯一实现 |
| 聚类标记渲染（SVG 图标 + Marker） | Browser/Client（BMap 与 BMapGL 各自的 renderClusterMarker） | — | 两套 API 面不同（BMapGL 有 3D 控件/倾斜），保留各自渲染层，仅聚类数据源收敛 |
| 层级/坐标过滤记忆化 | Client（React useMemo，deps `[buildings]`） | — | React 渲染性能域；hover/缩放路径不再重复全量 filter |
| 地图事件生命周期（zoomend/tiltend） | Client（useEffect cleanup） | — | React hooks 卸载契约；监听器随组件卸载解除 |
| 僵尸依赖解锁（@uiw/react-baidu-map） | 构建/依赖（仅删消费方文件） | Phase 120 | 本 phase 只删 BuildingMarkers/CityMarkers；package.json 卸载归 Phase 120 DEAD-02 |

## Standard Stack

### Core（零新增——全部复用既有依赖）
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| React | ^19.2.8 | useMemo/useCallback/effect cleanup | 项目既有 [VERIFIED: package.json] |
| TypeScript | strict（tsconfig.app.json） | 共享函数类型（`ReadonlyMap`、结构化 {x,y}/{lng,lat}） | `verbatimModuleSyntax: true`——type-only import 必须写 `import type` [VERIFIED: tsconfig.app.json] |
| Vitest | ^4.0.18 | 共享函数单元测试 | 项目既有，`__tests__/*.test.ts` 约定 [VERIFIED: vitest.config.ts + 目录实查] |

### Supporting
无。spatial hash 手写约 40 行纯函数——不引库（supercluster/mapv 等第三方聚类库会引入行为变更与依赖，违背 D-04 零回归与纯重构口径；且需求 MAP3D-02 已锁定手写网格方案）。

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| 自写 40px 均匀网格 | supercluster 等聚类库 | 库自带聚合语义（zoom 感知/球面距离）会改变聚类结果，破坏一致性要求；均匀网格保留锚点贪心语义且可逐位对照 |
| 注入 `toPixel` 回调让共享函数自己调地图 | 传入预计算 `Map<id,pixel>` | Map 方案使共享函数完全纯化（测试无需 mock 地图），且强制调用方单遍化——回调方案容易被实现成内层再调 API 回到老路 |
| Math.sqrt 距离比较（保留） | dx²+dy² < threshold² | 平方比较省 sqrt 但浮点边界语义与旧实现有理论差异；保留旧表达式 `pixelDistance(a,b) < threshold` 实现逐位一致（比较次数已从 O(n²) 降为 O(n·k)，sqrt 开销可忽略） |

**Installation:** 无（本 phase 不安装任何包）。

**Version verification:** 无新包需要 registry 验证。既有依赖版本实查：Node v24.19.0、Vitest ^4.0.18、@uiw/react-baidu-map ^2.7.5（Phase 120 才卸载）[VERIFIED: 本机 package.json + node --version]。

## Package Legitimacy Audit

**本 phase 不安装任何外部包**（纯重构既有代码）——审计协议不适用，无 audit 表。
Phase 120 DEAD-02 将卸载 `cron-parser` / `@react-spring/three` / `maath` / `@uiw/react-baidu-map`，届时需走完整 legitimacy gate（卸载方向风险低，但命令仍需走 npm uninstall 验证）。

## Architecture Patterns

### System Architecture Diagram

```
buildingApi.list(PAGE_SIZE=1000)            [index.tsx:42，mount 一次]
        │  setBuildings（引用稳定）
        ▼
┌─────────────────────────────────────────────────────────────┐
│ HubeiMap.tsx（BMap）          HubeiMapGL.tsx（BMapGL）        │
│                                                              │
│  useMemo(buildings) ──► { level1, level2, withCoords }       │  ← MAP3D-04（HubeiMap 渲染体；GL 版同理可选）
│        │                                                     │
│  effect deps [mapLoaded, buildings, currentZoom]             │
│        │                                                     │
│        ▼                                                     │
│  层级过滤（zoom===10 ? all : level1）→ 有坐标过滤             │
│        │                                                     │
│        ▼                                                     │
│  ①单遍预计算: for b of withCoords:                           │  ← MAP3D-01（每组件唯一 API 调用点）
│     pixels.set(b.id, map.pointToOverlayPixel(new Point(...)))│
│        │  Map<id, {x,y}>                                     │
│        ▼                                                     │
│  ②共享纯函数 clusterBuildings(buildings, pixels, 40,         │  ← MAP3D-02 + MAP3D-03
│     (p)=>map.pixelToPoint(...))            ▲                 │
│        │                                    │                │
│        │              ┌─────────────────────┘                │
│        │              │ building-spaces-3d/cluster.ts        │
│        │              │  · 40px 网格分桶 Map<bucketKey,idx[]>│
│        │              │  · 按原序锚点贪心 + 3×3 邻域          │
│        │              │  · 中心像素均值 → 回调转经纬度         │
│        ▼              │  （单元测试直接喂合成 pixels）         │
│  ClusterGroup[] ──► renderClusterMarker（各自保留）           │
│                                                              │
│  init effect: addEventListener(zoomend/tiltend) ⇄ cleanup    │  ← MAP3D-05
└─────────────────────────────────────────────────────────────┘
        ▼
地图 DOM（主线程不再有秒级长任务）

BuildingMarkers.tsx / CityMarkers.tsx ──删除──► @uiw/react-baidu-map 零 import（Phase 120 解锁）  ← MAP3D-06
```

主用例追踪：数据进入（1000 楼宇）→ 缩放切换（zoomend → setCurrentZoom → effect 重跑）→ 单遍投影 → 纯函数聚类 → 渲染标记，全程 O(n) 级地图 API 调用 + O(n·k) 纯数学比较。

### Recommended Project Structure
```
xingran-react-frontend/src/pages/operations/building-spaces-3d/
├── cluster.ts                     # [新建] 共享聚类纯函数（MAP3D-02/03 唯一实现）
├── utils.ts                       # [不动] filterBuildingsByZoom/pixelDistance 等既有共享函数
│                                  #        （Phase 120 会删 calculateWorkstationStats，独立文件避免同文件翻动）
├── __tests__/
│   ├── cluster.test.ts            # [新建] 聚类一致性测试（参考实现对照 + 边界用例）
│   ├── utils.test.ts              # [既有 25 用例，保持绿]
│   └── index.render.test.tsx      # [既有 3 用例，mock 两个地图组件，保持绿]
├── components/
│   ├── HubeiMap.tsx               # [改] 预计算循环替换双循环 / useMemo / zoomend cleanup / 删内联聚类
│   ├── HubeiMapGL.tsx             # [改] 同上 + tiltend cleanup（监听器提升具名）
│   ├── BuildingMarkers.tsx        # [删除]
│   ├── CityMarkers.tsx            # [删除]
│   ├── utils.ts                   # [不动] 存活消费方 BuildingModel3D
│   ├── constants.ts               # [不动] 存活消费方 BuildingModel3D（SCENE_DIMENSIONS 等）
│   └── __tests__/utils.test.ts    # [不动] 测的是 components/utils.ts，不受死组件删除影响
└── types.ts                       # [不动] BuildingItem 契约
```

### Pattern 1: 像素投影剥离（适配层 + 纯函数）
**What:** 地图 API 调用集中在调用方单遍循环，共享函数只接收 `ReadonlyMap<string, {x,y}>`。
**When to use:** 任何"算法依赖外部系统坐标变换"的性能重构——把变换批量化、算法纯化。
**Example:**
```typescript
// building-spaces-3d/cluster.ts（新建，类型定义参考 utils.ts 风格）
import type { BuildingItem } from "./types";

export interface ClusterGroup {
  buildings: BuildingItem[];
  centerPixel: { x: number; y: number };
  clusterLng: number;
  clusterLat: number;
}

/** 聚类中心像素 → 地理坐标（组件侧适配 map.pixelToPoint）；返回 undefined 时回退锚点经纬度 */
export type ClusterCenterProjector = (
  pixel: { x: number; y: number }
) => { lng: number; lat: number } | undefined;

/**
 * 楼宇像素聚类（单遍网格分桶版）——HubeiMap / HubeiMapGL 唯一实现。
 * 行为契约（与旧 O(n²) 实现逐位一致）：
 *  - 锚点贪心：按数组顺序取首个未处理楼宇为锚，吸收全部距锚 < threshold 的未处理楼宇；
 *  - 簇内成员顺序 = 原数组顺序；簇的输出顺序 = 锚点出现顺序；
 *  - 距离比较保留 sqrt 表达式（< threshold，严格小于）；
 *  - 中心 = 成员像素算术平均 → toCenterLngLat；回调缺失/返回 undefined/返回值 lng 为 falsy
 *    时回退锚点（cluster.buildings[0]）的 longitude/latitude（对齐旧代码 `centerPoint.lng || ...`）。
 */
export function clusterBuildings(
  buildings: BuildingItem[],
  pixels: ReadonlyMap<string, { x: number; y: number }>,
  threshold: number,
  toCenterLngLat?: ClusterCenterProjector
): ClusterGroup[] {
  // ① 单遍分桶：bucketSize = threshold ⇒ 阈值内点对必落在 3×3 邻域
  //    key = `${Math.floor(x / threshold)},${Math.floor(y / threshold)}`（floor 对负像素坐标同样正确）
  //    桶值存原始数组下标（number[]，天然按插入序 = 原数组序）
  // ② 按原序扫描锚点（processed Set 跳过），收集 3×3 邻域下标 → 升序排序 → 逐个比较：
  //    pixelDistance(anchorPixel, candPixel) < threshold → 吸收 + processed.add
  // ③ 中心均值 + toCenterLngLat 回调（或锚点回退）
  // （完整实现约 40-50 行，由 planner 拆任务）
}
```

```typescript
// HubeiMapGL.tsx 渲染 effect 内（替换 282-323 的双循环；HubeiMap 同型，BMap namespace）
const pixels = new Map<string, { x: number; y: number }>();
for (const b of buildingsWithCoords) {
  pixels.set(b.id, map.pointToOverlayPixel(new BMapGL.Point(b.longitude!, b.latitude!)));
}
const clusterGroups = clusterBuildings(
  buildingsWithCoords,
  pixels,
  CLUSTER_PIXEL_THRESHOLD,
  (p) => map.pixelToPoint?.(new BMapGL.Pixel(p.x, p.y))  // 保留旧代码的 ?. 与回退链
);
```

### Pattern 2: 事件监听 effect cleanup（异步 init 场景）
**What:** map 实例在 async initMap 中创建，handler 必须提升到 effect 作用域，cleanup 按实例判空移除。
**When to use:** 本 phase 两组件的 zoomend/tiltend。
**Example:**
```typescript
// HubeiMapGL.tsx init effect（替换 150-151 的内联箭头）
useEffect(() => {
  if (!mapRef.current || !BAIDU_MAP_AK) { /* ...既有守卫... */ return; }
  let map: BMapMapGL | null = null;
  const handleZoomEnd = () => { if (map) setCurrentZoom(map.getZoom()); };
  const handleTiltEnd = () => { if (map) setCurrentTilt(map.getTilt()); };

  const initMap = async () => {
    // ...既有加载流程（await loadBaiduMapGLScript 后有 if (!mapRef.current) return 守卫——
    //    unmount 时 React 已将 ref 置 null，天然防"卸载后才建图挂监听"竞态，保留）
    map = new BMapGL.Map(/* ... */);
    map.addEventListener("zoomend", handleZoomEnd);
    map.addEventListener("tiltend", handleTiltEnd);
    // ...
  };
  initMap();

  return () => {
    if (map) {
      map.removeEventListener("zoomend", handleZoomEnd);
      map.removeEventListener("tiltend", handleTiltEnd);
    }
  };
}, []);
```
HubeiMap 同型但只有 zoomend（2D 无 tilt），既有具名 `handleZoomEnd`（219-222）提升到 effect 作用域即可。

### Pattern 3: 渲染体 filter 收敛 useMemo（D-05 范本风格）
**What:** 5 道全量 filter 合并为一个 useMemo 返回 `{ level1, level2, withCoords }`。
**Example:**
```typescript
// HubeiMap.tsx（对应 REQUIREMENTS MAP3D-04 锁定形态）
const { level1, level2, withCoords } = useMemo(
  () => ({
    level1: buildings.filter((b) => b.level === 1),
    level2: buildings.filter((b) => b.level === 2),
    withCoords: buildings.filter((b) => b.longitude && b.latitude),
  }),
  [buildings]
);
// 统计面板（664/667/672/676/705）改读 .length；676 的重复计数与 705 的 === 0 判断直接复用
// 聚类 effect 的层级过滤可顺带复用：currentZoom === 10 ? buildings : level1（避免缩放时再 filter 一遍）
```
D-05 范本 `info-points/index.tsx:603-609` 即此风格（Map 构建 + deps 数组）。

### Anti-Patterns to Avoid
- **内层循环调地图 API：** `pointToOverlayPixel` 每对比较一次是 O(n²) 灾难根源——新实现每楼宇恰好调用一次。
- **内联箭头函数挂监听后试图移除：** `removeEventListener` 需同一函数引用，内联箭头永远移除不掉（HubeiMapGL.tsx:150-151 现状）。
- **桶值存楼宇对象、跨桶乱序合并：** 会破坏簇内成员顺序 → tooltip/侧栏列表顺序漂移。桶值必须存原始下标并升序扫描。
- **改用平方距离比较"顺手优化"：** 数学等价但浮点边界语义与旧实现有差异，违背逐位一致性目标（比较已不是瓶颈，sqrt 无需省）。
- **顺手把 `pixelToPoint` 换成 `overlayPixelToPoint`：** 见 Pitfall 1——既有坐标系统混用是"正确的错误"，本 phase 保持原样。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 像素距离计算 | 新写距离函数 | 既有 `utils.ts:70 pixelDistance`（HubeiMapGL 已在用；HubeiMap 从内联表达式切换过来） | 已有共享实现 + 单测 |
| 像素均值 | 新写 reduce | 既有 `utils.ts:82 averagePixelPosition` | 同上 |
| 缩放层级过滤 | 内联三目 | 既有 `utils.ts:138 filterBuildingsByZoom`；MAP3D-04 后 HubeiMap 可直接消费 useMemo 的 level1 数组 | HubeiMap 当前内联复制了同一逻辑（254-257），收敛时消除这份复制 |
| 聚类一致性验证思路 | 人工肉眼比对地图 | 测试内嵌旧算法参考实现 + 种子随机对照 | 参考实现只活在测试里（生产单一实现），对照输出逐位相等 |
| 阈值常量 | 再内联一个 40 | 既有 `constants.ts:53 CLUSTER_PIXEL_THRESHOLD`（HubeiMap.tsx:260 有一份内联 40 副本，收敛时删除） | 常量三副本（constants.ts / components/constants.ts:116 / HubeiMap 内联）至少消掉组件内联那份 |

**Key insight:** 本 phase 的"手写"边界只有约 50 行纯函数（网格分桶贪心）——这是需求明确锁定的手写项；其余一切（距离/均值/过滤/常量/测试基建）全部复用既有资产。

## Common Pitfalls

### Pitfall 1: 坐标系统混用是既有行为，重构时不得"纠正"
**What goes wrong:** 旧代码用 `pointToOverlayPixel`（覆盖物像素系）求均值，却用 `pixelToPoint`（视口像素系的逆变换）转回经纬度——严格说应配对 `overlayPixelToPoint`。重构者"顺手修正"会改变 clusterLng/clusterLat → 聚类标记视觉位置漂移 → 违反 success criteria 2。
**Why it happens:** 百度 API 命名相似（pointToPixel / pointToOverlayPixel / pixelToPoint / overlayPixelToPoint 四件套），容易当成笔误。[CITED: lbsyun.baidu.com jspopular/guide/custom-markers + JSAPI 3.0 类参考——pointToOverlayPixel 的逆是 overlayPixelToPoint，pixelToPoint 的逆是 pointToPixel]
**How to avoid:** 共享函数原样保留 `pixelToPoint?.()` + `|| 锚点经纬度` 回退链（含 `centerPoint.lng ||` 的 falsy 兜底语义）；在共享函数 JSDoc 里写明"故意保留既有混用，勿改"。
**Warning signs:** code review 出现 overlayPixelToPoint 字样或回退链被简化。

### Pitfall 2: removeEventListener 函数引用不一致
**What goes wrong:** handler 定义在 async initMap 内部（HubeiMap 现状）或干脆内联（HubeiMapGL 现状），cleanup 闭包拿不到同一引用，移除静默无效。
**Why it happens:** addEventListener/removeEventListener 按（事件名 + 引用相等）匹配。
**How to avoid:** handler 提升到 effect 作用域（Pattern 2）；cleanup 用 `if (map)` 判空。可加自动化测试（mock 地图 + spy removeEventListener + 相同引用断言）。
**Warning signs:** 卸载后 DevTools Performance 面板缩放仍触发组件 setState。

### Pitfall 3: 分桶扫描顺序导致输出顺序漂移
**What goes wrong:** 桶是 Map 迭代序、邻域收集顺序不确定 → 簇内成员顺序与旧实现不同 → tooltip 前三名、侧栏列表顺序肉眼可见变化，"视觉结果一致"失败。
**Why it happens:** 忽略了贪心算法输出对遍历序的敏感性（分组结果不受影响——每候选独立判定——但成员数组的排列受影响）。
**How to avoid:** 桶值存原始下标；每个锚点的 3×3 邻域候选按下标升序排序后扫描。参考实现对照测试会立刻抓住这类漂移。
**Warning signs:** 一致性测试 deep-equal 失败但簇的分组（仅成员集合）相同。

### Pitfall 4: 桶大小 < 阈值
**What goes wrong:** 若 bucketSize < 40，距锚 39px 的点可能落在 2 个桶索引之外，3×3 邻域漏判 → 欠聚类。
**Why it happens:** 直觉上"桶小更精确"。
**How to avoid:** 铁律 bucketSize ≥ threshold；推荐直接 bucketSize = threshold（40px，与 MAP3D-02 锁定参数一致），此时 |Δx|<40 且 |Δy|<40 ⇒ 桶索引差 ≤1，3×3 完备（sqrt(dx²+dy²)<40 蕴含 |dx|<40）。
**Warning signs:** 随机对照测试在跨桶边界夹具上失败。

### Pitfall 5: 异步 init 竞态下的 cleanup 时序
**What goes wrong:** 组件在脚本加载完成前卸载 → cleanup 先跑（此时 map 还是 null）→ 之后 async 续体仍执行、建图挂监听、setState on unmounted。
**Why it happens:** initMap 是 async，cleanup 与之无同步点。
**How to avoid:** 既有代码 `await loadScript` 后的 `if (!mapRef.current) return` 已兜底（unmount 时 React 将 ref 置 null）——保留该守卫勿删。cleanup 里 `if (map)` 判空即可覆盖"已建图"路径。
**Warning signs:** StrictMode 双挂载或快速切换地图版本开关（index.tsx useWebGL Switch）时控制台警告。

### Pitfall 6: `level` 怪癖与 `filterBuildingsByZoom` 语义
**What goes wrong:** index.tsx:55 把**所有**楼宇映射为 `level: 2`，而 zoom≠10 过滤 `level===1` → 非 zoom-10 时地图渲染零楼宇；GL 版 MAX_ZOOM=18，zoom 11-18 同样命中"非 10"分支。重构者可能想"修复"或改用 `zoom >= 10`。
**Why it happens:** 看起来像 bug，实为既有行为（且不在 33 项 findings 内）。
**How to avoid:** 纯重构口径——`filterBuildingsByZoom`（utils.ts:138-140）与 HubeiMap 内联三目的 `zoom === 10` 语义原样保留；O(n²) 痛点在 BMap 版 zoom=10（其 maxZoom=10，即常态）真实存在。
**Warning signs:** diff 里出现 `>= 10`、`level === 2`、index.tsx 的 level 映射改动。

### Pitfall 7: 死组件删除的连带边界
**What goes wrong:** 误删 `components/utils.ts` / `components/constants.ts` / `components/types.ts`（与死组件同名/相近的"伴生"文件）→ 打爆存活的 BuildingModel3D / AddressInput。
**Why it happens:** components/ 下存在**平行的第二套** constants.ts（嵌套 MARKER_COLORS）/ types.ts（CityGroup 另一定义）/ utils.ts（getBuildingMarkerColors），与父级同名文件极易混淆。
**How to avoid:** MAP3D-06 只删两个 .tsx。删除后 `components/utils.ts` 的 getBuildingMarkerColors / generateBuildingInfoHTML / generateClusterIconSVG / generateBuildingMarkerSVG / calculatePixelDistance沦为"仅测试引用"（knip 可见但 knip 非七 gate）——**不顺带清理**（属 Phase 120 死代码批次）；`components/__tests__/utils.test.ts` 不受影响照常跑。
**Warning signs:** `global.d.ts:57`（`import type {} from HubeiMapGL`）被误动，或 tsc/vite build 报 components/* 缺失。

### Pitfall 8: useMemo deps 写错引发无限循环
**What goes wrong:** useMemo/useEffect deps 里放内联对象/每次渲染新建的数组 → 每渲染重算，违背 MAP3D-04 目标甚至死循环（项目 CLAUDE.md 明令）。
**How to avoid:** 三个 useMemo/effect 的 deps 严格为 `[buildings]`（buildings 是 state 引用，index.tsx 仅 mount 时 setBuildings 一次，引用稳定 [VERIFIED: index.tsx:33-68]）；聚类 effect 保持既有 `[mapLoaded, buildings, currentZoom]`。
**Warning signs:** react-hooks/exhaustive-deps lint 告警被新 disable 注释压制。

## Code Examples

（核心三段已内嵌于 Architecture Patterns：共享函数契约 + 组件适配层 + cleanup 模式。以下补测试模式。）

### 聚类一致性单元测试（MAP3D-03 验收核心）
```typescript
// __tests__/cluster.test.ts（新建；约定：__tests__/ 目录 + *.test.ts，vitest globals: true）
import { describe, it, expect } from "vitest";
import { clusterBuildings } from "../cluster";
import type { BuildingItem } from "../types";

// 参考实现 = 旧 O(n²) 算法的忠实移植（只活在测试文件里，锁定行为契约）
function legacyCluster(buildings: BuildingItem[],
  pixels: ReadonlyMap<string, { x: number; y: number }>,
  threshold: number): ClusterGroup[] { /* 逐行复刻 HubeiMapGL.tsx:282-323，含 pixelDistance/均值/回退链 */ }

// 种子随机（mulberry32 等），CI 可复现
const mkBuildings = (n: number, rng: () => number): BuildingItem[] => /* ... */;

it("与旧 O(n²) 实现逐位一致（随机夹具 × 多组）", () => {
  const buildings = mkBuildings(500, mulberry32(42));
  const pixels = mkRandomPixels(buildings, mulberry32(7)); // 稀疏团簇 + 重叠混合
  expect(clusterBuildings(buildings, pixels, 40, undefined))
    .toEqual(legacyCluster(buildings, pixels, 40));
});

it("边界：距离恰为 40 → 不聚（严格小于）；39.99 → 聚", () => { /* ... */ });
it("边界：负像素坐标（Math.floor 分桶正确）", () => { /* ... */ });
it("边界：空数组 / 单点 / 全部同桶（1000 点一桶最坏情况不炸栈）", () => { /* ... */ });
it("中心回退：toCenterLngLat 缺省 / 返回 undefined / 返回 lng=0 → 锚点经纬度", () => { /* ... */ });
it("可选：性能冒烟——1000 点合成夹具 < 宽松预算（如 500ms），防退化回 O(n²)", () => { /* ... */ });
```

### 事件 cleanup 自动化测试（MAP3D-05，可选但推荐）
```typescript
// mock loadBaiduMapGLScript + getBMapGL 返回 fake map（addEventListener/removeEventListener 为 vi.fn()）
// render HubeiMapGL → 等 mapLoaded → unmount →
// expect(removeSpy).toHaveBeenCalledWith("zoomend", zoomHandlerRef)  // 同一引用
```
注：Baidu 脚本加载是组件外部依赖，测试只需 mock `./BaiduMapScript` 与 `@/types/baidu-map` 的 `getBMapGL`，jsdom 下可跑（参照既有 `components/__tests__/BaiduMapScript.test.ts` 的 mock 风格）。

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 覆盖物逐对 pointToOverlayPixel（O(n²) API 调用） | 单遍预计算 + 网格分桶纯数学 | 本 phase | n=1000 地图 API 调用从 ~200 万次降到 1000 次 |
| 双地图版本各自维护聚类副本 | 共享纯函数 + 参考实现对照测试 | 本 phase | 算法 bug 单点修复；行为被测试锁定 |
| BMap 内联表达式（距离/均值/过滤/阈值） | 统一消费 utils.ts 既有共享函数 | 本 phase | 消除 HubeiMap 内的第四处复制 |

**Deprecated/outdated:** 无废弃 API——BMap/BMapGL 的 pointToOverlayPixel/pixelToPoint/addEventListener 均为现行接口（项目类型声明 + 线上运行双重佐证）。

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | BMap/BMapGL 运行时均实现 `removeEventListener(event, handler)`（类型已声明 baidu-map.ts:60；百度官方事件 API 标准能力）[ASSUMED→in-phase 验证：MAP3D-05 测试即验证] | Pattern 2 | cleanup 静默无效 → 有自动化测试兜底，风险低 |
| A2 | `map.pixelToPoint` 两版运行时均存在（旧代码 `?.` 防御性调用暗示不确定；类型声明为必有）——保留 `?.` + 回退链则无论存在与否行为均不变 | Pitfall 1 | 无（防御链保留后不依赖此假设） |
| A3 | 分桶版与旧实现输出逐位一致（桶=阈值、3×3、升序下标、保留 sqrt 表达式四个条件下）[ASSUMED→in-phase 验证：一致性测试即验证] | Pattern 1 | 不一致会被参考实现对照测试当场抓住，不会带病合入 |
| A4 | 人工性能验证（n=1000 缩放切换）在开发机可用浏览器 + `VITE_BAIDU_MAP_AK` 完成（项目日常即依赖该 AK）[ASSUMED] | Validation Architecture | 退化为人工 review + 单测性能冒烟，验收弱化但不阻塞 |
| A5 | index.tsx 的 `buildings` state 引用仅在 mount 后变化一次（useCallback deps [] 且 effect 只跑一次）[VERIFIED: index.tsx:39-68 代码审查] | MAP3D-04 | useMemo deps `[buildings]` 失效 → 重算增多，无正确性风险 |

## Open Questions

1. **共享函数落位：独立 `cluster.ts` 还是并入 `utils.ts`？**
   - What we know: CONTEXT.md 建议"落位 building-spaces-3d/ 下 utils（与既有 utils.ts 同目录）"；utils.ts 在 Phase 120 还会被 DEAD-01 删 calculateWorkstationStats。
   - What's unclear: 无实质分歧——同目录已满足 CONTEXT 意图。
   - Recommendation: 独立 `cluster.ts`（约 50 行专用逻辑 + 类型，避免与 Phase 120 的 utils.ts 改动同文件翻动）；planner 可二选一，均合规。

2. **MAP3D-04 的 useMemo 是否顺带喂给聚类 effect（`currentZoom === 10 ? buildings : level1`）？**
   - What we know: success criteria 3 只要求渲染体 5 道 filter 收敛；聚类 effect 的过滤在 effect 内（deps 含 currentZoom）。
   - Recommendation: 顺带复用（零成本、消掉 HubeiMap.tsx:254-257 与 utils.ts:138-140 的重复），但不得改变过滤语义（Pitfall 6）。

3. **性能冒烟单测预算取值**
   - What we know: MAP3D-02 的正式验收是"人工性能验证 + 一致性单测"（REQUIREMENTS 回归纪律段）；自动性能断言非必需。
   - Recommendation: 若加，预算放宽到 500ms/1000 点量级防 CI flake；旧算法在同夹具下是秒级，断言仍有区分度。

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | 构建/测试 | ✓ | v24.19.0 | — |
| Vitest | MAP3D-03 单测 | ✓ | ^4.0.18（本目录 9 文件 64 用例实跑全绿，10.3s） | — |
| npm/既有依赖树 | 构建 | ✓ | — | — |
| 百度地图 AK（VITE_BAIDU_MAP_AK） | MAP3D-02 人工性能验证 | 环境 config 项（.env.development） | — | 缺失时人工验证降级为"单测性能冒烟 + code review"，不阻塞代码交付 |

**Missing dependencies with no fallback:** 无。
**Missing dependencies with fallback:** 浏览器人工性能验证依赖 AK 配置（上表 fallback）。

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Vitest ^4.0.18（jsdom, globals: true, testTimeout 15s, maxWorkers 4） |
| Config file | `xingran-react-frontend/vitest.config.ts`（coverage include `src/**/*.{ts,tsx}` 全量口径） |
| Quick run command | `npx vitest run src/pages/operations/building-spaces-3d`（实测 10.3s） |
| Full suite command | `npx vitest run`（前端）；七 gate 另含 lint / `npm run type-check` / build |

**基线状态（2026-09-12 实测）：** building-spaces-3d 目录 9 个测试文件 / 64 用例全绿。D-04 零回归即以此为基线。

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| MAP3D-01 | 内层循环无地图 API 调用（每组件 pointToOverlayPixel 恰 1 处=预计算循环） | 单元（一致性测试隐含验证算法正确）+ grep 断言 | `npx vitest run src/pages/operations/building-spaces-3d/__tests__/cluster.test.ts`；grep `pointToOverlayPixel` 在两组件各 ≤1 处（预计算循环内） | ❌ Wave 0 |
| MAP3D-02 | n=1000 无秒级长任务 | 人工验证（浏览器缩放/倾斜，需 AK）+ 可选性能冒烟单测 | 人工项——jsdom 无法测真实地图长任务（理由：BMapGL 为外部 WebGL SDK）；单测侧用合成 1000 点 + 宽松预算 | ❌ Wave 0（单测可选） |
| MAP3D-03 | 共享函数单一实现 + 一致性单元测试 | 单元（参考实现对照 + 种子随机 + 边界用例） | `npx vitest run src/pages/operations/building-spaces-3d/__tests__/cluster.test.ts` | ❌ Wave 0 |
| MAP3D-04 | 渲染体 5 道 filter → useMemo 一次 | 既有测试零回归 + type-check + code review（index.render.test.tsx mock 了地图组件，不执行 useMemo；组件级自动化需 mock 整条 Baidu 脚本链，性价比低） | `npx vitest run src/pages/operations/building-spaces-3d` + `npm run type-check` | ✅（回归口径） |
| MAP3D-05 | 卸载后监听不再触发 | 单元（mock 地图 spy removeEventListener + 引用一致断言）或 code review | `npx vitest run src/pages/operations/building-spaces-3d`（若加 cleanup 测试） | ❌ Wave 0（推荐） |
| MAP3D-06 | 死组件零引用 | grep 断言 + 全套件 + lint/type-check/build | `grep -r "BuildingMarkers\|CityMarkers" src/` 零命中；`grep -r "@uiw/react-baidu-map" src/` 零命中 | ✅（命令即验证） |

### Sampling Rate
- **Per task commit:** `npx vitest run src/pages/operations/building-spaces-3d`（<15s）
- **Per wave merge:** `npx vitest run` 全量 + `npm run lint` + `npm run type-check`
- **Phase gate:** 七 gate 全绿 + 人工性能验证（MAP3D-02）+ `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `src/pages/operations/building-spaces-3d/__tests__/cluster.test.ts` — 覆盖 MAP3D-01/02/03（含参考实现对照、边界、可选性能冒烟）
- [ ] （推荐）MAP3D-05 cleanup spy 测试（可并入上述文件或独立 `__tests__/hubei-map-cleanup.test.tsx`）
- [ ] 测试基建无需安装（Vitest/jsdom/mock 体系已就绪；`src/test/utils/renderWithProviders` 可复用）
- 覆盖率门禁：`.coverage-fe-floors` 无 building-spaces-3d 条目 [VERIFIED: 文件实查]——新增代码+测试不会触发 per-dir floor 门禁；`type-check`（tsconfig.app.json）排除 `*.test.ts(x)`，测试文件类型不被 tsc 检查（vitest esbuild 转换运行，宽松），共享函数本体（src 内非 test）受 strict 全检。

## Security Domain

（config.json 未显式关闭 security_enforcement → 按启用处理。本 phase 为纯前端性能重构，无新增攻击面，逐项如实标注。）

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | 无认证改动；页面仍走既有 JWT 中间链 |
| V3 Session Management | no | 不触及 |
| V4 Access Control | no | 不触及（数据仍来自既有已鉴权 buildingApi.list） |
| V5 Input Validation | no | 无新输入面；聚类输入为已鉴权 API 的既有字段 |
| V6 Cryptography | no | 不触及国密路径 |

### Known Threat Patterns for {React 19 + Baidu Maps SDK 前端}

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| SVG data-URI 模板内插楼宇名（HubeiMap.tsx:401 `buildingLabel` / GL 同型）——名称含 `<` 理论上可注入标记 SVG | Tampering（低危、既有、审计未列） | **不在本 phase 处理**（作用域约束：33 项 findings 之外的既有行为保持原样；如需处理归后续 quick task） |

## Sources

### Primary (HIGH confidence)
- 代码库实读（2026-09-12）：HubeiMap.tsx / HubeiMapGL.tsx / utils.ts（父+components 双份）/ constants.ts（双份）/ types.ts（双份）/ index.tsx / `@/types/baidu-map.ts` / vitest.config.ts / tsconfig.app.json / global.d.ts / 死组件 ×2 / 三份既有测试文件——全部行号引用出自此处 [VERIFIED: codebase]
- 基线实测：`npx vitest run src/pages/operations/building-spaces-3d` → 9 文件 64 用例全绿 [VERIFIED: 本机运行]
- 百度地图官方文档：[自定义标注·pointToOverlayPixel](https://lbsyun.baidu.com/index.php?title=jspopular/guide/custom-markers)、[JSAPI 3.0 类参考](https://lbsyun.baidu.com/cms/jsapi/reference/jsapi_reference_3_0.html)、[极速版覆盖物指南](https://lbsyun.baidu.com/index.php?title=jsextreme/guide/cover) [CITED]

### Secondary (MEDIUM confidence)
- WebSearch（z.ai web_search_prime）交叉确认 pointToOverlayPixel/pixelToPoint/overlayPixelToPoint/pointToPixel 四方法语义与配对关系，与官方文档一致 [VERIFIED: 官方文档交叉]

### Tertiary (LOW confidence)
- 无。

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — 零新增依赖，全部既有资产经实读/实跑确认
- Architecture: HIGH — 纯函数剥离设计基于两份实现的逐行差异分析；输出逐位一致性有数学论证 + in-phase 测试闭环
- Pitfalls: HIGH — 全部来自本仓库真实代码形态（双 constants/types 命名地雷、坐标系统混用、内联监听器、level 怪癖），非泛泛清单

**Research date:** 2026-09-12
**Valid until:** 2026-10-12（稳定域——代码库内部重构，无外部快速演进依赖；若 main 上这些文件被先行改动需复检行号）
