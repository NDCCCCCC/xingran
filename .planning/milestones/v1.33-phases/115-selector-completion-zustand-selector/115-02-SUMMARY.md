---
phase: 115-selector-completion-zustand-selector
plan: 02
subsystem: ui
tags: [zustand, react-19, selector, performance, router, 3d-visualization]

# Dependency graph
requires:
  - phase: 115-selector-completion-zustand-selector (plan 01)
    provides: useTabs/useLayout hook selector 化确立的统一改造范式（D-05 实操基线）
  - phase: 114-map-cluster-perf
    provides: HubeiMap/HubeiMapGL 同文件热度与聚类 useMemo/effect cleanup 成果（本 plan 零回退守护对象）
provides:
  - RouteGuard 仅订阅 s.permissions 的单字段 selector 订阅（41 行权限判断与 UX-only 注释逐字保留，V4 守护项）
  - DynamicRoutes 5 个独立 selector 订阅（menuStore 3 字段 + authStore 2 字段），菜单加载时序/getState/setState 静态方法一行不动（Phase 118 DATA-02 前向兼容）
  - 3D 页 5 处 visualizationStore 共 11 个字段级 selector（index/BuildingView3D/FloorView3D/HubeiMap/HubeiMapGL）
  - 菜单刷新过程（loading/lastFetchTime/error 高频写入）不再重渲整棵已认证路由树（本 phase 实质重渲收益主锚点）
affects: [116-dashboard-selector, 118-data-menu]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "guard/router 组件逐字段 selector：每字段一次 useXxxStore((s) => s.field)，订阅 s.allMenus 数组引用而非 .length 派生"
    - "3D 页 state+action 混合字段全部同型单值 selector，action 引用不包 useMemo/useCallback（zustand 5.0.15 action 身份稳定）"

key-files:
  created: []
  modified:
    - xingran-react-frontend/src/router/RouteGuard.tsx
    - xingran-react-frontend/src/router/DynamicRoutes.tsx
    - xingran-react-frontend/src/pages/operations/building-spaces-3d/index.tsx
    - xingran-react-frontend/src/pages/operations/building-spaces-3d/components/BuildingView3D.tsx
    - xingran-react-frontend/src/pages/operations/building-spaces-3d/components/FloorView3D.tsx
    - xingran-react-frontend/src/pages/operations/building-spaces-3d/components/HubeiMap.tsx
    - xingran-react-frontend/src/pages/operations/building-spaces-3d/components/HubeiMapGL.tsx

key-decisions:
  - "DynamicRoutes 严守只改订阅形态纪律：fetchAll effect（131-137）/ routeConfigManager.initialize（151）/ getState-setState 静态方法（117-125）一行不动，Phase 118 DATA-02 前向兼容"
  - "订阅 s.allMenus 数组引用而非 s.allMenus.length 派生（RESEARCH Anti-Pattern 明示），effect 依赖 [isAuthenticated, initialized, allMenus.length, fetchAll] 原样"
  - "FloorView3D 一次解构 4 字段拆 4 个独立 selector（禁对象 selector），selectedFloor 在 loadWorkstations useCallback 依赖中引用语义不变"

patterns-established:
  - "Pattern: 路由层/3D 页组件的整店解构 → 逐字段单值 selector 收敛（禁对象 selector / 禁 useShallow，维持全库 0 使用）"

requirements-completed: [SELECTOR-03, SELECTOR-04]

# Metrics
duration: 6min
completed: 2026-09-14
---

# Phase 115 Plan 02: 路由层与 3D 页 selector 化 Summary

**RouteGuard/DynamicRoutes 路由层 2 文件 + 3D 页 5 文件的整店订阅全部收敛为 17 个字段级 Zustand selector，菜单刷新高频写入不再重渲整棵已认证路由树，Phase 118 DATA-02 同文件前向兼容**

## Performance

- **Duration:** 6 min
- **Started:** 2026-09-14T02:33:46Z
- **Completed:** 2026-09-14T02:39:56Z
- **Tasks:** 2
- **Files modified:** 7

## Accomplishments
- RouteGuard.tsx:33 仅订阅 `s.permissions`（userPermissions 别名保留），41 行 `permissions.some((p) => userPermissions.includes(p))` 与文件头"这不是安全边界"UX-only 注释逐字不变——SELECTOR-03 + V4 守护项（grep -qF 锁定命中）
- DynamicRoutes.tsx:105-106 拆 5 个独立 selector（menuStore allMenus/fetchAll/permissions + authStore isAuthenticated/initialized），fetchAll effect / routeConfigManager.initialize / getState-setState 静态方法一行不动——SELECTOR-03（Phase 118 前向兼容）
- 3D 页 5 处 visualizationStore 整店订阅全部为字段级 selector 共 11 个：index:36 viewLevel（1）/ BuildingView3D:59（2）/ FloorView3D:58-61 拆 4 个独立调用 / HubeiMap:78-79（2）/ HubeiMapGL:77-78（2）——SELECTOR-04
- 路由层 + 3D 目录 `use(Menu|Auth|Visualization)Store()` 无参调用 grep 归零；menuStore loading/lastFetchTime/error 高频写入不再牵连 RouteGuard/DynamicRoutes 重渲（本 phase 实质重渲收益主锚点）

## Task Commits

Each task was committed atomically:

1. **Task 1: 路由层 selector 化——RouteGuard:33 + DynamicRoutes:105-106（SELECTOR-03，含 V4 守护项）** - `393f852` (feat)
2. **Task 2: 3D 页 5 处 visualizationStore 逐字段 selector 化（SELECTOR-04）** - `9e364b6` (feat)

## Files Created/Modified
- `xingran-react-frontend/src/router/RouteGuard.tsx` - permissions 单字段 selector 订阅（1 行，权限逻辑/注释逐字保留）
- `xingran-react-frontend/src/router/DynamicRoutes.tsx` - 2 行解构 → 5 个独立 selector（数据流零改动）
- `xingran-react-frontend/src/pages/operations/building-spaces-3d/index.tsx` - viewLevel 单字段 selector
- `xingran-react-frontend/src/pages/operations/building-spaces-3d/components/BuildingView3D.tsx` - selectedBuilding + navigateToMap 两个 selector
- `xingran-react-frontend/src/pages/operations/building-spaces-3d/components/FloorView3D.tsx` - 拆 4 个独立 selector（selectedFloor/selectedBuilding/navigateToBuilding/navigateToMap）
- `xingran-react-frontend/src/pages/operations/building-spaces-3d/components/HubeiMap.tsx` - clearSelection + navigateToBuilding 两个 selector
- `xingran-react-frontend/src/pages/operations/building-spaces-3d/components/HubeiMapGL.tsx` - 同 HubeiMap

## Decisions Made
- DynamicRoutes 严守"只改订阅形态"硬纪律（ROADMAP 跨 phase 冲突约束）：菜单加载时序、TTLMenuCache 语义、store 静态方法调用全部原样，Phase 118 (DATA-02) 前向兼容
- 订阅 `s.allMenus`（数组引用）而非 `.length` 派生值，effect 依赖数组 `[isAuthenticated, initialized, allMenus.length, fetchAll]` 不变
- 3D 页 action 引用直接 `s => s.action`，不包 useMemo/useCallback（zustand 5.0.15 action 创建后身份稳定）；不引入 useShallow（维持全库 0 使用）
- 7 个改动文件上的 9 条 eslint 警告（`_typeColor`/`_e`/`_error` 未用变量、2 处 no-explicit-any、3 处 react-refresh/only-export-components）经 diff 比对确认为 pre-existing（改动行仅订阅行，行号漂移由本次 +1~+2 行引入），按 scope boundary 规则不修

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- 首次 Task 1 提交被 commitlint `body-max-line-length` 拒绝（CJK 正文行超 100 字符），拆短正文行后重提成功（lint-staged 全链路 0 错误，未用 --no-verify）
- 未触发 RESEARCH Pitfall 3 冷缓存假失败：受影响测试（routeGuard 8 tests + 3D 簇 91 tests）合并首跑即全绿

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- SELECTOR-03/04 完成，本 phase 仅余 SELECTOR-05（115-03：action-only/user 单字段 5 文件 + NotificationBell 7 项 + 4 个 mock 文件 dual-form 改造）
- Phase 118 (DATA-02) 可安全触达 DynamicRoutes.tsx——本 plan 对该文件仅订阅形态变更，数据流/时序语义完全未动
- Phase 116 (DASH) 的 dashboard 11 处整店订阅不在本 plan 范围，selector 统一范式已三波次（store hook / 路由层 / 3D 页）落地可复用

## Threat Flags

无新增攻击面：本 plan 零新增依赖（T-115-SC 不适用确认）、零网络端点/auth 路径/schema 变更；RouteGuard 权限判断逐字保留（T-115-02-A mitigate 落实，grep -qF 验证通过）。

## Self-Check: PASSED

- 115-02-SUMMARY.md 存在 ✓
- Task 1 commit `393f852` 存在 ✓
- Task 2 commit `9e364b6` 存在 ✓
- 提交无意外文件删除 ✓

---
*Phase: 115-selector-completion-zustand-selector*
*Completed: 2026-09-14*
