---
phase: 115-selector-completion-zustand-selector
plan: 01
subsystem: ui
tags: [zustand, react-19, selector, performance, frontend]

# Dependency graph
requires:
  - phase: 114-map-cluster-perf
    provides: 同模块 3D 页 selector 化先例与 HubeiMap/HubeiMapGL 文件热度（本 plan 未触及同文件，仅范式参照）
provides:
  - useTabs hook 内部 14 个逐字段 useTabsStore selector（返回对象形态不变，消费组件零改动）
  - useLayout hook 内部 10 个逐字段 useLayoutStore selector（2 effect + layoutConfig + 6 派生布尔原样）
  - 全库 selector 收尾统一改造范式确立（Phase 116/120 后续波次的 D-05 实操基线）
affects: [116-dashboard-selector, 118-data-menu, 120-dead-code]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "hook 内逐字段 selector 订阅：每字段一次 useXxxStore((s) => s.field)（useRouteTabs.ts:45-48 D-05 范本），action 与 state 字段写法相同"

key-files:
  created: []
  modified:
    - xingran-react-frontend/src/store/tabsStore.ts
    - xingran-react-frontend/src/store/layoutStore.ts

key-decisions:
  - "useTabs 的 history 字段按 OQ-2 定案保留在返回对象（0 消费方，但移除属超范围行为变更，D-04 最小 diff 优先）"
  - "hooks 数量 1→14 / 1→10 固定展开，React 19 rules-of-hooks 合法；不引入 useShallow（维持全库 0 使用）"

patterns-established:
  - "Pattern: store 文件内导出 hook 的 selector 化——只动订阅来源（解构 → 逐字段 const），返回对象/effect/派生值一字不改，消费组件零改动"

requirements-completed: [SELECTOR-01, SELECTOR-02]

# Metrics
duration: 7min
completed: 2026-09-14
---

# Phase 115 Plan 01: useTabs/useLayout hook selector 化 Summary

**useTabs（14 字段）与 useLayout（10 字段）hook 内部整店解构改为逐字段 Zustand selector 订阅，返回对象一字不改，TabBar/LayoutSwitcher 等常驻组件消费面零改动**

## Performance

- **Duration:** 7 min
- **Started:** 2026-09-14T02:22:51Z
- **Completed:** 2026-09-14T02:29:12Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- useTabs 内部 14 个逐字段 `useTabsStore((s) => s.xxx)`（tabs/activeTab/history + 11 action），`hasTabs: tabs.length > 0` 派生由局部 tabs 计算原样保留——SELECTOR-01
- useLayout 内部 10 个逐字段 `useLayoutStore((s) => s.xxx)`，effect 1（data-layout/data-density 同步，依赖 [currentLayout, density]）与 effect 2（settings-changed 监听+卸载清理，依赖 [syncFromSettings]）、`layoutConfig`、6 个派生布尔全部原样——SELECTOR-02
- 两个 store 文件内 `useXxxStore()` 无参调用 grep 归零（src 全量非测试文件亦零命中）；tabsStore/layoutStore 任意未被订阅字段的 set() 不再牵连消费组件重渲

## Task Commits

Each task was committed atomically:

1. **Task 1: useTabs hook 14 字段逐字段 selector 化（SELECTOR-01）** - `579da16` (feat)
2. **Task 2: useLayout hook 10 字段逐字段 selector 化 + 派生/effect 原样保留（SELECTOR-02）** - `34ce99c` (feat)

## Files Created/Modified
- `xingran-react-frontend/src/store/tabsStore.ts` - useTabs hook 订阅来源改造（310 行区域，返回对象一字不改）
- `xingran-react-frontend/src/store/layoutStore.ts` - useLayout hook 订阅来源改造（287 行区域，2 effect + 派生值原样）

## Decisions Made
- history 字段保留在 useTabs 返回对象（OQ-2 planner 定案：D-04 最小 diff 优先，非验收项）
- 逐字段 selector 严格照抄 useRouteTabs.ts:45-48 形态（含 `(s)` 参数与空格风格），action 引用不包 useMemo（zustand 5.0.15 action 创建后稳定）

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- 无。4 个受影响测试文件（49 tests）与 layoutStore.test.ts（7 tests，含 :104-149 派生值/settings-changed 断言）改造后首跑即全绿，未触发 RESEARCH Pitfall 3 冷缓存假失败

## Verification

- grep：`use(Tabs|Layout)Store\(\)` 在 `src/store/` 与全 src 非测试文件均零命中
- selector 计数：tabsStore.ts 14 处 / layoutStore.ts 10 处 `useXxxStore((s) =>`
- 测试：5 文件 56 tests 全绿（tabsStore / layoutStore / TabBar.render / useRouteTabs / useUtilityHooks）
- gate：`npm run lint` exit 0（0 errors，两改动文件单独 eslint 0 警告）、`npm run type-check` exit 0

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- SELECTOR-01/02 完成，selector 统一范式已在两个 store hook 落地，115-02（SELECTOR-03 路由层 + SELECTOR-04 3D 页）可直接复用同型改造
- 本 plan 零新增依赖、零行为变更，D-02 七 gate 相关项（lint/type-check）无回归

## Self-Check: PASSED

- 115-01-SUMMARY.md 存在 ✓
- Task 1 commit `579da16` 存在 ✓
- Task 2 commit `34ce99c` 存在 ✓
- 提交无意外文件删除 ✓

---
*Phase: 115-selector-completion-zustand-selector*
*Completed: 2026-09-14*
