# Phase 116 Plan 01 Summary: useWidgetData Selector + Dashboard 9-File Per-Field Selectors

**Plan:** 116-01  
**Phase:** 116-dashboard-cascade-dashboard  
**Wave:** 1  
**Commit:** `fdef7d3`  
**Completed:** 2026-09-14

## Objective

DASH-01 + DASH-03: useWidgetData selector form (action reference stabilization) + dashboard module 9-file whole-store subscription convergence to per-field selectors.

Purpose: After getCachedWidgetData moved out of the queryFn closure to getState() internal call, queryFn deps reduced from 5 to 4; action references are now stable so widget memo recovers effectiveness. 9 whole-store subscriptions changed to per-field selectors so any dashboardStore field change no longer triggers re-renders in components not subscribed to that field.

## What Was Done

### Task 1: useWidgetData getCachedWidgetData moved out of closure (DASH-01)

**Files modified:** `xingran-react-frontend/src/hooks/useWidgetData.ts`

- Removed `_getCachedWidgetData` parameter from `fetchWidgetData` function signature
- `cacheWidgetData` changed from destructuring `useDashboardStore()` to selector `useDashboardStore((s) => s.cacheWidgetData)`
- `getCachedWidgetData` now called internally via `useDashboardStore.getState().getCachedWidgetData(widget.id)` — not a hook dependency
- `useCallback` deps reduced from 5 to 4: `[widget.id, widget.enabled, widget.dataSource, cacheWidgetData]`
- Same changes applied to `useBatchWidgetData` (selector form for `cacheWidgetData`)

**Assertions:**
- `grep -c "useDashboardStore\(\)" useWidgetData.ts` = 0
- `grep -c "useDashboardStore\.getState\(\)\.getCachedWidgetData" useWidgetData.ts` >= 1
- `grep -c "getCachedWidgetData.*useCallback" useWidgetData.ts` = 0

### Task 2: dashboard-system 9 files converted to per-field selectors (DASH-03)

**Files modified (10 files total):**

| File | Fields converted |
|------|-----------------|
| `index.tsx` | setPageMode (1) |
| `DashboardHome.tsx` | defaultDashboard, defaultDashboardLoading, fetchDefaultDashboard (3) |
| `DashboardList.tsx` | dashboards, listLoading, listPagination, fetchDashboards, createDashboard, deleteDashboard, duplicateDashboard, setDefaultDashboard (8) |
| `DashboardView.tsx` | currentDashboard, currentLoading, fetchDashboard, setViewMode, updateWidgetLayouts, clearCurrentDashboard (6) |
| `DashboardEdit.tsx` | currentDashboard, currentLoading, fetchDashboard, setViewMode, updateWidgetLayouts, addWidget, selectWidget, selectedWidgetId, clearCurrentDashboard (9) |
| `edit.tsx` | Same 9 fields as DashboardEdit.tsx |
| `view.tsx` | Same 6 fields as DashboardView.tsx |
| `WidgetEditor.tsx` | updateWidget, removeWidget (2) |
| `DashboardSettings.tsx` | currentDashboard, updateDashboard (2) |

**Pattern applied:** `const fieldName = useDashboardStore((s) => s.fieldName);` (per `useRouteTabs.ts:45-48`)

**Assertions:**
- `grep -rE "useDashboardStore\(\)"` across all 9 files = 0
- Per-file `grep -c "useDashboardStore\(\(s\) => s\."` counts match original destructuring field counts

## Verification

- **Lint:** `npm run lint --silent` — 0 errors (1366 warnings, same as baseline)
- **Type-check:** `npm run type-check` — exit 0
- **Grep assertions:** all pass

## Deviations from Plan

None — plan executed exactly as written.

## Auth Gates

None.

## Threat Flags

None.

## TDD Gate Compliance

Not applicable — this plan does not follow TDD methodology.
