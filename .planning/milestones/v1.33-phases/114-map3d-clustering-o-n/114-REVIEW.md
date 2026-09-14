---
phase: 114-map3d-clustering-o-n
reviewed: 2026-09-12T03:40:00Z
depth: standard
files_reviewed: 5
files_reviewed_list:
  - xingran-react-frontend/src/pages/operations/building-spaces-3d/__tests__/cluster.test.ts
  - xingran-react-frontend/src/pages/operations/building-spaces-3d/__tests__/hubei-map-cleanup.test.tsx
  - xingran-react-frontend/src/pages/operations/building-spaces-3d/cluster.ts
  - xingran-react-frontend/src/pages/operations/building-spaces-3d/components/HubeiMap.tsx
  - xingran-react-frontend/src/pages/operations/building-spaces-3d/components/HubeiMapGL.tsx
findings:
  critical: 1
  warning: 2
  info: 3
  total: 6
status: issues_found
---

# Phase 114: Code Review Report

**Reviewed:** 2026-09-12T03:40:00Z
**Depth:** standard
**Files Reviewed:** 5
**Status:** issues_found

## Summary

对照 `ee5559c^..HEAD` 的 diff 逐文件审查了本 phase 全部变更：新增共享纯函数 `cluster.ts`、两组件（HubeiMap / HubeiMapGL）聚类 O(n²) → 单遍预计算 + 网格分桶替换、事件监听 cleanup（MAP3D-05）、死组件删除（BuildingMarkers / CityMarkers，全仓 grep 无残留引用，删除干净）、以及两个新增测试文件。实测两个测试文件 18/18 通过（vitest 4.1.10）。

**核心契约（与旧 O(n²) 实现逐位一致）验证结论：成立。** 审查独立复核了四条正确性支柱：

1. **分桶完备性**：桶大小 = threshold（取等号）时，`|Δx| < 40` 严格蕴含桶索引差 ≤ 1（若差 ≥ 2 则 |Δx| ≥ 40，矛盾），故锚点 3×3 邻域候选集完备，不漏点；
2. **顺序复刻**：桶值存原始下标（插入序 = 原数组序）+ `neighbors.sort((a,b) => a-b)` 升序扫描，与旧实现内层 `forEach`（数组序）的吸收顺序逐位一致；
3. **距离语义**：严格小于 `<` + `pixelDistance` 的 sqrt/Math.pow 表达式原样保留（utils.ts:70-75 未改），无平方优化引入的浮点边界漂移；
4. **回退链等价性**：新 `centerPoint?.lng || anchor.longitude!`（cluster.ts:124-125）与旧 `pixelToPoint?.(...) || {lng,lat}` + 字段级 `||` 在全部输入域等价（含 `{lng:0,lat:5}` 半回退、投影返回 null/0 等 falsy 值——optional chaining 对非对象 falsy 返回 undefined，与旧 `||` 整体回退殊途同归），测试 273-280 行已锁定该语义。

坐标系统混用（pointToOverlayPixel + pixelToPoint）按上下文确认为故意保留的既有行为，未作为问题记录。一处良性差异：旧代码 `pointToOverlayPixel` 返回 null 时 `pixelDistance` 抛 TypeError 中断整个 effect（全部标记消失），新代码防御路径（cluster.ts:68,87）静默跳过该楼宇——属健壮性改进，非回归。

本 phase 自身变更未发现正确性缺陷。下述 Critical / Warning 均为受审文件内的**既有问题（pre-existing，非本 phase 引入）**，已逐条标注，供 orchestrator 排期修复；其中 CR-01 为存储型 XSS，建议优先处理。

## Critical Issues

### CR-01: InfoWindow HTML 注入（存储型 XSS）——building.name/address 未转义直接拼入 HTML【pre-existing】

**File:** `xingran-react-frontend/src/pages/operations/building-spaces-3d/components/HubeiMap.tsx:414-431`；同型 `components/HubeiMapGL.tsx:407-424`

**Issue:** `showBuildingInfo` 将 `building.name`、`building.address`（楼宇管理接口返回的用户可控字段）直接以模板字符串拼入 HTML，经 `infoWindowRef.current.setContent(...)` 交给百度地图 InfoWindow 渲染（百度 InfoWindow 以 innerHTML 注入，不消毒）。拥有楼宇创建/编辑权限的任意用户可提交形如 `<img src=x onerror=...>` 的名称，对查看地图的任何用户（含更高权限管理员）执行任意 JS——典型的垂直提权跳板。同一模板 431 行（GL 版 424 行）`onclick="window.viewBuildingDetails('${building.id}')"` 还将 id 拼入内联 JS 字符串。**该代码在本 phase diff 中未改动（pre-existing）**，但位于受审文件内，按安全漏洞定级 Critical。

**Fix:**

```tsx
// 两组件各加一个转义 helper（或提取到 ../utils 复用）
const escapeHtml = (s: string): string =>
  s.replace(/[&<>"']/g, (ch) =>
    ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[ch]!));

// content 模板内：
//   ${escapeHtml(building.name)} / ${escapeHtml(building.address || "暂无地址")}
// onclick 改为数据驱动，去掉内联 JS 拼接：
//   <button data-building-id="${encodeURIComponent(building.id)}" ...>
// 并在 InfoWindow 事件（或容器委托）中读取 data-building-id 调 window.viewBuildingDetails
```

## Warnings

### WR-01: HubeiMapGL 缩放到 11-18 级时二级楼宇反而消失（越放大信息越少）【pre-existing】

**File:** `xingran-react-frontend/src/pages/operations/building-spaces-3d/components/HubeiMapGL.tsx:282`；根因 `../utils.ts:139` + `../constants.ts:17`

**Issue:** `filterBuildingsByZoom` 用精确匹配 `zoom === 10 ? buildings : 只留 level===1`，而 GL 版 `MAP_CONFIG.MAX_ZOOM = 18`。GL 地图放大到 11-18 级时条件为 false，二级（具体楼宇）标记全部隐藏——用户放大想看细节，标记却消失，与 2D 版语义错位（2D 版 `maxZoom: 10`，zoom 10 即顶格，精确匹配无害）。**既有缺陷**：该调用行与 utils/constants 本 phase 均未改动，但聚类 effect 在本 phase 被重写且继续消费该语义。

**Fix:**

```ts
// utils.ts filterBuildingsByZoom —— 阈值语义改为"≥10 显示全部"：
export function filterBuildingsByZoom(buildings: BuildingItem[], zoom: number): BuildingItem[] {
  return zoom >= 10 ? buildings : buildings.filter((b) => b.level === 1);
}
// （对 2D 版行为不变：其 zoom 取值域为 {8,9,10}，>=10 与 ===10 等价）
// 或收敛 GL 版 constants.ts MAX_ZOOM 为 10，与 2D 版对齐——二选一，需产品确认
```

### WR-02: HubeiMap.tsx 与共享层大量逐字重复（toBase64 / HUBEI_BOUNDARY / 边界辅助函数），MAP_CONFIG 双源已分叉【pre-existing】

**File:** `xingran-react-frontend/src/pages/operations/building-spaces-3d/components/HubeiMap.tsx:28-37, 44-50, 81-104, 107-126`

**Issue:** 本地 `toBase64`（44-50）、`HUBEI_BOUNDARY`（81-104，19 个坐标点逐字重复）、`getOfficialHubeiBoundary` / `parseBoundaryPoints`（107-126）与 `../utils.ts:14-62`、`../constants.ts:6-50` 的共享实现完全重复——HubeiMapGL 走共享版，HubeiMap 走本地副本。同时本地 `MAP_CONFIG`（28-37）与 constants.ts `MAP_CONFIG`（9-24）同名双源且值已分叉（本地 maxZoom=10 vs 共享 MAX_ZOOM=18）。真实风险：降级路径（官方边界拉取失败）使用本地 HUBEI_BOUNDARY 副本，共享副本更新后两份边界数据漂移；本 phase 主目标之一即去重（cluster.ts 抽取），该文件内剩余重复未收敛。**既有问题**，建议后续小 phase 收敛。

**Fix:**

```tsx
// 删除本地 toBase64 / HUBEI_BOUNDARY / getOfficialHubeiBoundary / parseBoundaryPoints，
// 改从共享层导入（与 HubeiMapGL 一致）：
import { toBase64, getOfficialHubeiBoundary, parseBoundaryPoints } from "../utils";
import { HUBEI_BOUNDARY, CLUSTER_PIXEL_THRESHOLD } from "../constants";
// 本地 MAP_CONFIG 若 maxZoom=10 确属 2D 版故意差异：重命名为 MAP_CONFIG_2D 并注释
// "2D 版缩放上限 10，与 GL 版 MAX_ZOOM=18 不同属能力差异"，消除同名双源
```

## Info

### IN-01: init 竞态——unmount 早于异步 initMap 完成时，监听器不会被移除且续体仍创建地图/调用 setState【pre-existing】

**File:** `xingran-react-frontend/src/pages/operations/building-spaces-3d/components/HubeiMap.tsx:167-249`；同型 `components/HubeiMapGL.tsx:81-171`

**Issue:** cleanup 依赖 effect 作用域的 `map` 变量；若组件在 `loadBaiduMapScript` pending 期间卸载，cleanup 以 `map === null` 空跑，随后 initMap 续体仍在已脱管的容器上创建地图实例、挂 zoomend 监听（无人移除）、调用 setMapLoaded/setLoading。已确认项目未启用 StrictMode（main.tsx 无），实例随脱管 DOM 一起 GC，无用户可见缺陷——是 MAP3D-05 cleanup 改进的残留缺口，非缺陷级。

**Fix:** effect 作用域加取消标志：`let cancelled = false;`，cleanup 中 `cancelled = true;`，initMap 每个 `await` 之后检查 `if (cancelled) return;`，取消时不创建实例 / 不挂监听 / 不 setState。

### IN-02: polygonRef 死引用——声明并仅在降级路径赋值，从不读取，官方边界成功路径不赋值【pre-existing】

**File:** `xingran-react-frontend/src/pages/operations/building-spaces-3d/components/HubeiMap.tsx:58, 163`；同型 `components/HubeiMapGL.tsx:65, 226`

**Issue:** `polygonRef` 全文件无读取点；`addHubeiMask` 官方边界成功路径创建的 polygon 也未记录。推测原意是 unmount 时 `removeOverlay` 清理，从未落地。死代码。

**Fix:** 要么删除 polygonRef 声明与赋值；要么补全意图——成功/降级路径均记录，init cleanup 中 `polygonRef.current && map.removeOverlay(polygonRef.current)`。

### IN-03: 一致性对照测试存在共同模式盲区——averagePixelPosition 无绝对锚定

**File:** `xingran-react-frontend/src/pages/operations/building-spaces-3d/__tests__/cluster.test.ts:12, 52, 228`

**Issue:** 参考实现 legacyCluster 与生产 clusterBuildings 共享 `utils.ts` 的 `pixelDistance` / `averagePixelPosition`——这两个共享辅助若本身有错，对照测试两侧同错（common-mode failure）无法发现。`pixelDistance` 已被 39/40、(28,28)/(29,29) 边界用例绝对锚定；`averagePixelPosition` 仅经 `centerPixel` 出现在 toEqual 对照中，无手算绝对断言（影响面低：centerPixel 只用于 tooltip 定位）。

**Fix:** 单点用例（test 224-228 行）已隐含锚定单成员均值；建议再加一条两成员手算断言，如像素 (0,0) 与 (10,30) → centerPixel 期望 `{x: 5, y: 15}`。

---

_Reviewed: 2026-09-12T03:40:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
