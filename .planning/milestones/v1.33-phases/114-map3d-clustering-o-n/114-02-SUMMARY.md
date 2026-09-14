---
phase: 114-map3d-clustering-o-n
plan: 02
subsystem: ui
tags: [react, typescript, vitest, spatial-hash, clustering, performance, baidu-map, useMemo, event-cleanup]

# Dependency graph
requires:
  - phase: 114-01
    provides: clusterBuildings 纯函数（40px 网格分桶 + 锚点贪心）+ ClusterGroup/ClusterCenterProjector 共享类型 + 15 用例一致性测试
provides:
  - HubeiMapGL / HubeiMap 两组件聚类单遍化（每组件 pointToOverlayPixel 恰 1 处预计算调用点，n=1000 地图 API 调用从 ~200 万降到 n 次）
  - HubeiMap 渲染体 useMemo {level1, level2, withCoords}（MAP3D-04，hover/缩放不再重复全量 filter）
  - 两组件 zoomend/tiltend 监听器 effect cleanup（同引用 removeEventListener，MAP3D-05）
affects: [114-03（Phase 收尾验证与死组件删除）, Phase 120 DEAD-02（@uiw/react-baidu-map 卸载解锁）]

# Tech tracking
tech-stack:
  added: [] # 零新增依赖（纯重构既有代码）
  patterns: [像素投影剥离消费侧（effect 内单遍 pointToOverlayPixel 预计算 Map<id,pixel> + 共享纯函数聚类）, 渲染体 filter 记忆化（useMemo deps 严格 [buildings]）, 事件监听提升具名 handler + effect cleanup 同引用移除（异步 init 场景 let map 作用域提升）]

key-files:
  created: []
  modified:
    - xingran-react-frontend/src/pages/operations/building-spaces-3d/components/HubeiMapGL.tsx
    - xingran-react-frontend/src/pages/operations/building-spaces-3d/components/HubeiMap.tsx

key-decisions:
  - "两组件聚类双循环整段替换为『单遍 pointToOverlayPixel 预计算 Map<id,{x,y}> + clusterBuildings 纯函数调用』——保留 pixelToPoint?.() 回调与回退链（坐标系统既有混用原样，纯重构红线）"
  - "HubeiMap 聚类 effect 有坐标过滤保留在 effect 内、不复用 useMemo 的 withCoords——旧代码『先层级过滤再有坐标过滤』的输入集合语义不可变（withCoords 未做层级过滤）"
  - "层级过滤收敛为 currentZoom === 10 ? buildings : level1（消费 useMemo），zoom === 10 语义原样保留（禁改 >= 10，Pitfall 6）"
  - "HubeiMap 本地 toBase64 副本 / MAP_CONFIG / HUBEI_BOUNDARY / renderClusterMarker 一概不动（不在 MAP3D-01~05 范围，Scope Constrainment）"
  - "pixelDistance/averagePixelPosition 从 HubeiMapGL 的 utils 导入列表移除（双循环删除后组件内零调用点），距离/均值/阈值内联表达式在 HubeiMap 全部消除"

patterns-established:
  - "Pattern 单遍预计算消费侧：聚类 effect 内 for-of 一遍建 Map<id,pixel>（每组件唯一地图 API 调用点），聚类本体零 SDK 依赖"
  - "Pattern 异步 init 监听器生命周期：let map 提升 effect 作用域 + handler 具名 + cleanup if(map) 同引用 removeEventListener（内联箭头永远移除不掉的反模式终结）"

requirements-completed: [MAP3D-01, MAP3D-02, MAP3D-04, MAP3D-05]

# Metrics
duration: 12min
completed: 2026-09-12
---

# Phase 114 Plan 02: 两组件聚类单遍化 + 渲染体收敛 + 监听器 cleanup Summary

**HubeiMapGL 与 HubeiMap 聚类双循环替换为单遍预计算 + clusterBuildings 共享纯函数（n² 地图 API 调用归零），HubeiMap 渲染体 5 道 filter 收敛 useMemo，两组件事件监听补 effect cleanup——聚类视觉行为逐位不变**

## Performance

- **Duration:** 12 min
- **Started:** 2026-09-11T18:05:38Z
- **Completed:** 2026-09-11T18:17:44Z
- **Tasks:** 2
- **Files modified:** 0 created / 2 modified

## Accomplishments

- **MAP3D-01 完成**：两组件 `pointToOverlayPixel` 各收敛为恰 1 处（预计算 for-of 循环内），内层循环零地图 API 调用——n=1000 时地图 API 调用从 ~200 万次降到 n 次
- **MAP3D-02 消费侧落地**：聚类本体走 Plan 01 的 `clusterBuildings`（40px 网格分桶 + 锚点贪心），40px 阈值 HubeiMap 内联副本删除、改消费 `../constants` 单点
- **MAP3D-03 完成态**：两组件本地 `ClusterGroup` interface 副本删除，均 `import { clusterBuildings, type ClusterGroup } from "../cluster"`
- **MAP3D-04 完成**：HubeiMap 渲染体 5 道全量 filter（664/667/672/676/705）收敛为单个 `useMemo` 返回 `{level1, level2, withCoords}`，deps 严格 `[buildings]`，统计面板 5 处改读 `.length`，渲染体不再出现任何 `buildings.filter(...)`（useMemo 内 3 处除外）
- **MAP3D-05 完成**：HubeiMapGL 的 2 条内联箭头监听（zoomend/tiltend）与 HubeiMap 的 initMap 内部 handleZoomEnd 全部提升为 effect 作用域具名函数，cleanup 以同一函数引用 `removeEventListener` 并 `if (map)` 判空；`if (!mapRef.current) return` 异步 init 竞态守卫两处保留
- **纯重构红线全数守住**：`overlayPixelToPoint` 零命中（坐标系混用未"纠正"）、`zoom === 10` 语义原样、index.tsx level 映射未动、distance 严格小于语义由共享函数继承、无新增 exhaustive-deps disable 注释（4 处全为既有）

## Task Commits

Each task was committed atomically:

1. **Task 1: HubeiMapGL——预计算替换双循环 + 监听器提升具名 + cleanup** - `f748fa7` (feat)
2. **Task 2: HubeiMap——useMemo 收敛 5 filter + 预计算替换双循环 + cleanup + 单点收敛** - `a6b3fdd` (feat)

## Files Created/Modified

- `xingran-react-frontend/src/pages/operations/building-spaces-3d/components/HubeiMapGL.tsx` - 聚类双循环（282-323 旧）替换为单遍预计算 + clusterBuildings；zoomend/tiltend 具名 handler + cleanup；本地 ClusterGroup 删除；utils 导入清理（pixelDistance/averagePixelPosition 移除）；+33/-60 行
- `xingran-react-frontend/src/pages/operations/building-spaces-3d/components/HubeiMap.tsx` - 同型聚类改造（BMap namespace）+ 渲染体 useMemo {level1, level2, withCoords} + 统计面板 5 处读 .length + 内联阈值 40/sqrt/reduce 消除 + zoomend handler 提升 + cleanup；本地 ClusterGroup 删除；+61/-97 行

## Decisions Made

- HubeiMap 聚类 effect 的有坐标过滤保留在 effect 内、不改用 useMemo 的 `withCoords`——useMemo 版未做层级过滤，换用会把 zoom≠10 时的聚类输入从"level1∩有坐标"扩大为"全量有坐标"，构成行为变更（Plan action 明确禁止）
- HubeiMap 聚类 effect 层级过滤改消费 `level1`（`currentZoom === 10 ? buildings : level1`）——顺带消掉 `filterBuildingsByZoom` 的内联复制（GL 版仍消费 utils 的 filterBuildingsByZoom，各自原状，不做跨组件统一）
- effect deps 保持 `[mapLoaded, buildings, currentZoom]` 原样（level1 引用随 buildings 变化，无 stale closure 风险；既有 eslint-disable 注释已覆盖该行）

## Deviations from Plan

None - plan executed exactly as written.

（提交环节两次被 commit-msg hook 拒绝——subject 大写开头 [subject-case] / 正文行超 100 字符 [body-max-line-length]，均为提交信息格式问题，改写信息后原样提交成功，非代码偏差。）

## Issues Encountered

- commitlint `subject-case` 拒绝 Task 1 首次提交（subject 以大写 HubeiMapGL 开头）——中文开头重写后通过
- commitlint `body-max-line-length` 拒绝 Task 2 首次提交（正文行 >100 字符）——正文换行缩短后通过
- 既有 flake 未复现：目录级回归两次运行（Task 1 后 8.81s / Task 2 后 8.79s）`index.render.test.tsx` 均一次通过（Plan 01 期间记录的并行负载 import 超时未出现）

## Verification Results

- **grep 断言（MAP3D-01 验收口径）**：`pointToOverlayPixel` HubeiMapGL=1 / HubeiMap=1（各为预计算循环唯一调用点）；`processedBuildings` 两组件均 0（O(n²) 双循环已删）
- **grep 断言（MAP3D-04）**：HubeiMap `buildings.filter` 恰 3 处（全部在 useMemo 内）；`const CLUSTER_PIXEL_THRESHOLD = 40` 内联副本 0 命中、改 import `../constants`
- **grep 断言（MAP3D-05）**：`removeEventListener("tiltend", handleTiltEnd)` 与 `removeEventListener("zoomend", handleZoomEnd)` 均在，同引用具名 handler；`if (!mapRef.current) return` 守卫两处保留
- **红线断言**：`overlayPixelToPoint` 两组件 0 命中；exhaustive-deps disable 注释 2+2=4 处全为既有零新增
- **目录级回归**：`npx vitest run src/pages/operations/building-spaces-3d` 两次运行均 10 文件 79 用例全绿（含 Plan 01 一致性测试——聚类输出与旧 O(n²) 实现逐位一致，D-04 零回归）
- **type-check**：`npm run type-check` exit 0（提交 hook 内 lint-staged 再跑一遍亦通过）
- **lint**：`npm run lint` exit 0（0 errors；两组件单文件 eslint 0 errors、5 条 warning 全为既有 catch `_error`/`(window as any)`）

## User Setup Required

None - no external service configuration required.

## Known Stubs

None - 无占位/未接线代码（纯重构，全部消费路径已接线：预计算 Map → clusterBuildings → renderClusterMarker 既有渲染层）。

## Next Phase Readiness

- Plan 03（114-03，Phase 收尾）可执行：死组件 BuildingMarkers/CityMarkers 删除（MAP3D-06）+ 全套件验证；本 plan 未触碰 components/ 平行第二套 utils/constants/types（删除雷区完好）
- MAP3D-02 的人工性能验证（n=1000 缩放/倾斜切换无秒级卡顿，需浏览器 + VITE_BAIDU_MAP_AK）留给 Phase 收口步骤
- 两组件渲染层（renderClusterMarker）零改动——聚类标记视觉位置由 Plan 01 一致性测试 + 渲染层不动双重保障

## Self-Check: PASSED

- 两组件文件存在且已入库（git show f748fa7 / a6b3fdd 确认 1 file changed 各一次）
- 提交 f748fa7（Task 1）与 a6b3fdd（Task 2）均在 git log 中确认
- 工件标记验证：两组件含 `clusterBuildings`（各 1 处调用）与 `from "../cluster"`；HubeiMap 含 `CLUSTER_PIXEL_THRESHOLD.*from "../constants"`
- 提交误删检查：两次提交 `git diff --diff-filter=D` 均无文件删除；工作树无新增未跟踪文件

---
*Phase: 114-map3d-clustering-o-n*
*Completed: 2026-09-12*
