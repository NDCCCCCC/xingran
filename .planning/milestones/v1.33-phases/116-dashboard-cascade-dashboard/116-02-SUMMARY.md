# Phase 116 Plan 02 Summary: Widget L1 Cache Moved to Module-Level Map

**Plan:** 116-02  
**Phase:** 116-dashboard-cascade-dashboard  
**Wave:** 2  
**Commits:** `c8b2a82` (store refactor), `a208cd9` (test fixes)  
**Completed:** 2026-09-14

## Objective

DASH-02: widget data L1 cache moved from dashboardStore reactive state (widgetDataCache Map) to module-level Map, following the noticeStore P1-M4 pattern. `cacheWidgetData / getCachedWidgetData / clearWidgetCache / updateWidgetData` four actions now operate on the module-level Map without triggering any Zustand reactive subscriptions. useWidgetData already refactored in Plan 01 to use `getState().getCachedWidgetData`.

Purpose: After widget data is written to the module-level Map, polling refreshes no longer trigger any dashboardStore subscriptions (cacheWidgetData's set() call no longer triggers any component re-renders). Combined with Plan 01 selector conversion, this truly achieves "single widget data update only re-renders that widget."

## What Was Done

### Task 1: widget L1 cache moved out of reactive state (DASH-02)

**File modified:** `xingran-react-frontend/src/store/dashboardStore.ts`

**Changes:**
1. `widgetDataCache` removed from `DashboardState` interface
2. `widgetDataCache: new Map()` removed from `initialState`
3. Module-level Map declared at file scope (before store creation):

```typescript
// 模块级 widget L1 缓存（不进入响应式 state，参考 noticeStore P1-M4）
export const widgetDataCache = new Map<string, { data: unknown; timestamp: number }>();
export const getWidgetDataCache = () => widgetDataCache;
```

4. `cacheWidgetData`: direct `widgetDataCache.set()` — no `set()` call
5. `getCachedWidgetData`: direct `widgetDataCache.get()` — no `get()`穿透 state
6. `clearWidgetCache`: direct `widgetDataCache.delete()` / `widgetDataCache.clear()`
7. `updateWidgetData`: direct `widgetDataCache.set()`
8. `reset`: also calls `widgetDataCache.clear()` (module-level Map survives reset)
9. `persist partialize` unchanged (widgetDataCache was never persisted)

**Assertions:**
- `grep -c "^const widgetDataCache = new Map" dashboardStore.ts` = 1 (module-level)
- `grep -c "widgetDataCache: Map" dashboardStore.ts` = 0 (removed from state interface)
- `grep -c "set.*widgetDataCache" dashboardStore.ts` = 0 (no set() calls)
- `grep -c "new Map(state.widgetDataCache)" dashboardStore.ts` = 0

### Test Fixes (Rule 1 — auto-fixed)

**Files modified:** `dashboardStore.test.ts`, `DashboardView.test.tsx`, `DashboardEdit.test.tsx`, `DashboardList.test.tsx`, `DashboardSettings.test.tsx`

The module-level `widgetDataCache` is unreachable via `getState()`, so:
- `dashboardStore.test.ts`: replaced `useDashboardStore.getState().widgetDataCache` direct accesses with `getWidgetDataCache()` calls
- `beforeEach`: replaced `setState({ widgetDataCache: new Map() })` with `widgetDataCache.clear()`
- Component tests: updated mocks to handle selector form `useDashboardStore((s) => s.field)` in addition to destructuring form

**All 35 dashboard tests pass.**

## Verification

- **Lint:** `npm run lint --silent` — 0 errors (1367 warnings)
- **Type-check:** `npm run type-check` — exit 0
- **Tests:** 35/35 dashboard tests pass

## Deviations from Plan

1. **Added `getWidgetDataCache()` export**: Module-level Map needed a getter for test access. Not in original plan, required for test functionality.
2. **`reset()` now also clears module-level Map**: Not specified in plan, but necessary for test isolation (beforeEach cleanup). Semantic correctness preserved.

## Auth Gates

None.

## Threat Flags

None.

## TDD Gate Compliance

Not applicable — this plan does not follow TDD methodology.
