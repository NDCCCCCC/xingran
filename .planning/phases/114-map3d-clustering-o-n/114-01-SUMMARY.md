---
phase: 114-map3d-clustering-o-n
plan: 01
subsystem: ui
tags: [react, typescript, vitest, spatial-hash, clustering, performance, baidu-map]

# Dependency graph
requires:
  - phase: none（Wave 0 基础 plan，无前置依赖）
    provides: 既有 utils.ts（pixelDistance/averagePixelPosition）、types.ts（BuildingItem）、constants.ts（CLUSTER_PIXEL_THRESHOLD=40）、Vitest 测试基建
provides:
  - clusterBuildings 纯函数（40px 网格分桶 + 锚点贪心，输出与旧 O(n²) 实现逐位一致）
  - ClusterGroup / ClusterCenterProjector 共享类型导出
  - 聚类一致性单元测试（参考实现对照 + 种子随机 + 边界 + 回退 + 性能冒烟，15 用例）
affects: [114-02（HubeiMap/HubeiMapGL 两组件改造，消费本 plan 的唯一聚类实现）]

# Tech tracking
tech-stack:
  added: [] # 零新增依赖（纯重构既有代码）
  patterns: [像素投影剥离（调用方单遍预计算 Map<id,pixel> + 纯函数聚类）, 40px 网格分桶 spatial hash（bucketSize=threshold ⇒ 3×3 邻域完备）, 参考实现对照测试（旧算法内嵌测试文件锁定逐位一致）, mulberry32 种子 PRNG 保证 CI 可复现]

key-files:
  created:
    - xingran-react-frontend/src/pages/operations/building-spaces-3d/cluster.ts
    - xingran-react-frontend/src/pages/operations/building-spaces-3d/__tests__/cluster.test.ts
  modified: []

key-decisions:
  - "独立 cluster.ts 落位（非并入 utils.ts）——约 130 行专用逻辑 + 类型，避免与 Phase 120 DEAD-01 的 utils.ts 删函数改动同文件翻动（RESEARCH Open Question 1 已裁决）"
  - "桶大小 = threshold 取等号（40px）——|Δx|<40 且 |Δy|<40 蕴含桶索引差 ≤1，3×3 邻域数学完备，与 MAP3D-02 锁定参数一致"
  - "桶值存原始数组下标 + 邻域候选升序扫描——复刻旧 O(n²) 实现簇内成员顺序，toEqual 全结构对照锁定（Pitfall 3 顺序漂移防线）"
  - "保留 pixelDistance 的 sqrt 比较表达式（禁平方优化）——浮点边界语义与旧实现逐位一致"
  - "中心回退链两级语义保留（centerPoint?.lng || anchor.longitude!）——字段级 falsy 兜底（如 lng=0）有专属测试锁定"
  - "零 SDK 依赖红线：cluster.ts 仅 import type BuildingItem + utils 两函数，grep baidu-map 零命中，Vitest 免 mock 直测"

patterns-established:
  - "Pattern 像素投影剥离：地图 API 调用集中在调用方单遍循环，共享函数只收 ReadonlyMap<string,{x,y}>（Plan 02 两组件照此消费）"
  - "Pattern 参考实现对照：旧算法忠实移植只活在测试文件里，种子随机 × toEqual 锁定行为契约"

requirements-completed: [MAP3D-02, MAP3D-03]

# Metrics
duration: 11min
completed: 2026-09-12
---

# Phase 114 Plan 01: 共享聚类纯函数 cluster.ts + 一致性测试 Summary

**40px 网格分桶 + 锚点贪心的纯函数 clusterBuildings（零 SDK 依赖）+ 内嵌旧 O(n²) 参考实现的 15 用例一致性测试，输出与旧实现逐位一致（TDD RED→GREEN）**

## Performance

- **Duration:** 11 min
- **Started:** 2026-09-11T17:47:54Z
- **Completed:** 2026-09-11T17:59:18Z
- **Tasks:** 2（TDD：Task 1 RED → Task 2 GREEN）
- **Files modified:** 2 created / 0 modified

## Accomplishments
- clusterBuildings 唯一共享聚类实现落地：单遍分桶 + 3×3 邻域 + 下标升序扫描，n=1000 时距离比较从 ~100 万对降为 O(n·k)，地图 API 调用不再进入内层循环（Plan 02 消费）
- 参考实现对照测试锁定逐位一致：3 组种子随机（500 点团簇+散点混合）× toEqual 全结构相等（簇分组 + 成员顺序 + 中心坐标）+ 回调路径 1 组
- 四要素全有测试覆盖：桶=阈值（1000 点同桶用例）、3×3 邻域（跨桶边界随机夹具）、下标升序（toEqual 抓顺序漂移）、sqrt 严格小于（距离恰 40 不聚 / 39 聚 / 斜向 39.60 vs 41.01 边界）
- 中心回退链两级语义锁定：缺省/undefined 整体回退锚点；lng=0 字段级 falsy 半回退（经度兜底、纬度保留）
- 性能冒烟：1000 点合成夹具单次聚类 < 500ms（实测毫秒级），防退化回 O(n²)

## Task Commits

Each task was committed atomically:

1. **Task 1: 创建聚类一致性测试（RED）** - `270142e` (test)
2. **Task 2: 实现 cluster.ts 共享聚类纯函数（GREEN）** - `ee5559c` (feat)

_Note: TDD 门控合规——test 提交在前（RED 非零退出验证）、feat 提交在后（GREEN 全绿），无 refactor 提交（实现一次成型无需清理）。_

## Files Created/Modified
- `xingran-react-frontend/src/pages/operations/building-spaces-3d/cluster.ts` - 共享聚类纯函数（导出 clusterBuildings/ClusterGroup/ClusterCenterProjector，约 131 行含 JSDoc；坐标系统混用故意保留声明 + 逐位一致三要素文档）
- `xingran-react-frontend/src/pages/operations/building-spaces-3d/__tests__/cluster.test.ts` - 15 用例一致性测试（legacyCluster 参考实现内嵌 + mulberry32 种子随机 + 边界/回退/性能冒烟，293 行）

## Decisions Made
- 独立 `cluster.ts` 落位而非并入 `utils.ts`（RESEARCH Open Question 1 已裁决，避免与 Phase 120 utils.ts 改动同文件翻动）
- 1000 点同桶用例的锚点置于桶中心 (20,20)——桶内任意点距锚 ≤ 19.5√2 ≈ 27.6 < 40，数学上保证"1 簇 1000 成员"断言成立（纯随机桶内点位不满足该断言，属计划行为条款的必要实现细节）
- 测试额外增加 1 组"带 toCenterLngLat 回调路径"的种子随机对照（计划未列但直接支撑 Plan 02 真实 projector 消费场景，零成本加强契约）

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- **commitlint body-max-line-length 拒绝首次 feat 提交**：提交信息正文行超 100 字符（Task 2 首次 commit 被 commit-msg hook 拒绝，lint-staged 已通过）。换行重写正文后提交成功。非代码问题，无代码改动。
- **既有测试 index.render.test.tsx 并行负载 flake（范围外，未修）**：目录级回归首跑时该文件 beforeAll 的 `await import("../index")` 在 10 文件并行下超时 10s（hookTimeout）报 FAIL；单跑通过（3 用例绿，import 仅 2.66s）、目录重跑全绿。属既有重模块导入在满载下的偶发超时，与本 plan 两个新建文件零关联（../index 不引用 cluster.ts/cluster.test.ts）。按 Scope Constrainment 原则不顺带修复，挂账后续 quick task。

## Verification Results

- `npx vitest run src/pages/operations/building-spaces-3d/__tests__/cluster.test.ts`：15/15 绿（exit 0）
- 目录级回归 `npx vitest run src/pages/operations/building-spaces-3d`：10 文件 79 用例全绿（既有 64 + 新增 15，D-04 零回归基线）
- `grep -c "baidu-map" cluster.ts` = 0（纯数学无 SDK 依赖）
- utils.ts / constants.ts / types.ts / 两组件零改动（git diff 确认）

## User Setup Required

None - no external service configuration required.

## Known Stubs

None - 无占位/未接线代码（纯函数 + 测试，全部实现完整）。

## Next Phase Readiness
- Plan 02（114-02）可直接消费：`import { clusterBuildings, type ClusterGroup, type ClusterCenterProjector } from "../cluster"`，签名与 114-01-PLAN.md context interfaces block 完全一致
- Plan 02 改造两组件时保留各自的预计算循环（pointToOverlayPixel 单遍）+ pixelToPoint 回调适配 ClusterCenterProjector
- 注意事项：既有 index.render.test.tsx 存在并行负载下的 import 超时 flake（见 Issues Encountered），Plan 02 跑回归若复现可单跑该文件确认

## Self-Check: PASSED

- cluster.ts / cluster.test.ts / 114-01-SUMMARY.md 三个文件全部存在
- 提交 270142e（test RED）与 ee5559c（feat GREEN）均在 git log 中确认
- 工件标记验证：cluster.ts 含 ReadonlyMap（1 处）、测试文件含 legacyCluster（6 处）

---
*Phase: 114-map3d-clustering-o-n*
*Completed: 2026-09-12*
