# Phase 116: Code Review Report

**Reviewed:** 2026-09-14
**Depth:** standard
**Files Reviewed:** 12
**Status:** issues_found

## Summary

Phase 116 implements three DASH decisions: (DASH-01/03) selector conversion of `useWidgetData` and 9 dashboard-system component files from whole-store destructuring to per-field selectors; (DASH-02) widget L1 cache moved from reactive Zustand state to module-level Map; (DASH-04) `DashboardGrid` removal of `useWindowSize` subscription and `gridProps` useMemo stabilization.

Implementation is largely correct. The module-level `widgetDataCache` Map refactor is clean, all selector conversions verified as zero bare `useDashboardStore()` calls remain in the 10 source files, and `gridProps` is properly memoized. However, two MEDIUM-severity issues were found: a negative `staleTime` arithmetic bug in both `useWidgetData` and `useBatchWidgetData` that would cause undefined behavior when `refreshInterval < 5`, and a potential cache inconsistency where dashboard metadata mutations (name/description/scope change) do not invalidate widget data cache entries.

---

## Critical Issues

None found.

---

## Warnings

### WR-01: Negative `staleTime` when `refreshInterval < 5`

**File:** `xingran-react-frontend/src/hooks/useWidgetData.ts:95`
**Issue:** `staleTime: (refreshInterval - 5) * 1000` produces a negative value when `refreshInterval` is less than 5 seconds. React Query's `staleTime` option expects a non-negative number in milliseconds; a negative value causes undefined caching behavior (likely treated as 0 or throws). This is reachable if a widget is configured with a fast refresh interval.

The same pattern exists in `useBatchWidgetData` at line 156: `staleTime: minInterval - 5000` (negative when `minInterval < 5000`).

**Failure scenario:** Widget with `refreshInterval: 3` configured → `staleTime = -2000` ms → React Query may treat staleTime as 0 or exhibit unpredictable caching.

**Fix:**
```typescript
// useWidgetData.ts:94-95 — use max() to floor at 0
staleTime: Math.max(0, (refreshInterval - 5) * 1000),

// useBatchWidgetData.ts:155-156
staleTime: Math.max(0, minInterval - 5000),
```

**Verdict:** MEDIUM — functional bug, data staleness guarantees break when refreshInterval < 5.

---

### WR-02: Dashboard metadata mutations do not invalidate widget data cache

**File:** `xingran-react-frontend/src/store/dashboardStore.ts` (actions: `updateDashboard`, `deleteDashboard`, `duplicateDashboard`)
**Issue:** When a dashboard's metadata is mutated (name, description, scope, refreshInterval changed via `updateDashboard`), the action does not call `clearWidgetCache`. Since widget data cache entries are keyed by `widget.id` and stored in the module-level `widgetDataCache` Map, stale widget data could be returned after a dashboard configuration change.

The DASH-02 intent ("单个 widget 数据更新只重渲该 widget") focused on data-layer cache consistency (widget data refreshes), but the mutation-layer cache invalidation is incomplete. Only widget add/remove/reorder (which trigger `addWidget`/`removeWidget`/`updateWidgetLayouts`) preserve cache semantics because they affect which widgets exist. But changing `refreshInterval` on the dashboard would alter how often `useWidgetData` polls without clearing the old cached entries.

**Note:** This is partially mitigated because `getCachedWidgetData` has a 5-minute TTL (line 415-419), so stale data will naturally expire. However, the explicit `refreshInterval` change without cache invalidation could cause a temporarily incorrect poll rate perception by React Query.

**Failure scenario:** User changes dashboard refresh interval from 300s to 60s → cached widget data from the old 300s cycle (still within 5-min TTL) is returned instead of fresh data → widget shows stale data for up to 5 minutes.

**Fix:**
```typescript
// In updateDashboard action (line 202-213), add:
clearWidgetCache(); // invalidate all widget cache on dashboard config change

// Or scoped invalidation if widget IDs can be determined:
if (updatedWidgets) {
  updatedWidgets.forEach(id => clearWidgetCache(id));
}
```

**Verdict:** MEDIUM — cache inconsistency post-metadata mutation; bounded by 5-min TTL but not explicitly handled.

---

## Info

### IN-01: `useWidgetData.ts` `queryFn` stale closure on `widget` object

**File:** `xingran-react-frontend/src/hooks/useWidgetData.ts:79-85`
**Issue:** `queryFn` captures `widget` by reference, not `widget.id/dataSource/enabled` primitives. The comment at line 81-82 acknowledges `widget.dataSource` changes are "rare" and the query key includes `widget.id` for deduplication. However, if a widget's `dataSource` changes without the `widget.id` changing, the queryFn body reads the new `widget.dataSource` while the queryKey uses the stale closure's `widget.dataSource` — causing the data fetcher to use an outdated dataSource while the key suggests new data.

**Likely impact:** Low — per CLAUDE.md conventions, widget `dataSource` changes are intentional reconfigurations, not frequent runtime mutations. The queryKey deduplication would still fire on `widget.id` changes.

**Fix (if needed):** Add `widget.dataSource` to queryKey explicitly, or ensure widget object reference stability.

---

### IN-02: `gridProps` useMemo includes `children` as dependency

**File:** `xingran-react-frontend/src/components/dashboard/layout/DashboardGrid.tsx:220`
**Issue:** `children` is included in `gridProps` useMemo dependencies. If the parent passes a new `children` array/object reference on every render, `gridProps` will be reconstructed every render, defeating the useMemo optimization. However, in the observed usage in `DashboardView.tsx` and `DashboardEdit.tsx`, children are stable `widgets.map(...)` results, so this is not currently causing issues.

**Fix (if needed):** Consider moving children rendering into a separate memoized child component to prevent unnecessary gridProps reconstruction.

---

### IN-03: `DashboardSettings` useEffect dependency array notes

**File:** `xingran-react-frontend/src/components/dashboard/settings/DashboardSettings.tsx:56`
**Issue:** `useEffect` at line 43-56 has `form` in its dependency array. `Form.useForm()` from Ant Design is stable across renders, but this is an implicit assumption rather than documented guarantee. The effect correctly guards against running when `visible` is false.

---

### IN-04: `DashboardView` duplicate CSS import

**File:** `xingran-react-frontend/src/components/dashboard/DashboardView.tsx:27-28`
**Issue:** `./DashboardView.css` is imported twice (lines 27 and 28). Second import is a no-op in JavaScript module semantics, but indicates a copy-paste error or IDE duplication artifact.

**Fix:** Remove the duplicate import at line 28.

---

## Structural Findings (fallow)

No structural findings provided for this phase.

---

## Regression Checklist

| Check | Result |
|-------|--------|
| `useDashboardStore()` bare calls in 10 source files | PASS (0 finds) |
| `useDashboardStore.getState().getCachedWidgetData` in useWidgetData.ts | PASS (found) |
| `useDashboardStore((s) => s.cacheWidgetData)` in useWidgetData.ts | PASS (found) |
| Module-level `widgetDataCache = new Map` exactly 1 occurrence | PASS |
| `widgetDataCache` NOT in DashboardState interface | PASS |
| No `set(...widgetDataCache...)` in store actions | PASS |
| `useWindowSize` removed from DashboardGrid.tsx | PASS |
| `useMemo<ExtendedResponsiveProps>` gridProps in DashboardGrid.tsx | PASS |
| `DashboardState` no longer has `widgetDataCache` field | PASS |
| persist partialize: viewMode, showGridLines, showWidgetBorders only | PASS |

---

## Test Coverage Assessment

**Test files reviewed:**
- `src/store/dashboardStore.test.ts` — comprehensive 355-line test covering all CRUD, widget operations, cache lifecycle (5-min TTL expiry with fake timers), WS state, reset, and persist. All assertions pass.
- `src/pages/dashboard-system/components/__tests__/DashboardView.test.tsx` — minimal smoke tests (render without throw). Mock supports both selector and destructuring forms.
- `src/pages/dashboard-system/components/__tests__/DashboardEdit.test.tsx` — not deeply reviewed but test suite was updated in fix commit `a208cd9` alongside store changes, suggesting tests were aligned.
- `src/pages/dashboard-system/components/__tests__/DashboardList.test.tsx` — same.

**Coverage sufficiency:** The store test's use of `getWidgetDataCache().get("w1")` directly validates module-level Map behavior, confirming DASH-02's `widgetDataCache` is correctly accessed. The 5-minute TTL expiry test (fake timers advancing 6 minutes) validates the expiration path. However, no test covers the `refreshInterval < 5` negative staleTime path (IN-01).

---

## Narratives by File

### `src/store/dashboardStore.ts`

**DASH-02 implementation:** Correct. Module-level `widgetDataCache` Map at line 15 with exported `getWidgetDataCache` accessor for testing. All four cache actions (`cacheWidgetData`, `getCachedWidgetData`, `clearWidgetCache`, `updateWidgetData`) directly manipulate the Map without triggering Zustand subscriptions. `reset()` at line 475-478 calls `widgetDataCache.clear()` correctly. `persist` partialize omits `widgetDataCache` intentionally.

**Bug:** `updateDashboard` (line 202-213) does not call `clearWidgetCache`, so widget data from prior fetch cycles remains cached across dashboard metadata changes.

**Action reference stability:** All actions are declared as arrow functions inside the Zustand store creator, which are stable references within the store lifetime. The selector form `useDashboardStore((s) => s.fetchDashboards)` returns stable function references.

### `src/hooks/useWidgetData.ts`

**DASH-01 implementation:** Correct. Line 70: `useDashboardStore((s) => s.cacheWidgetData)` is a per-field selector. `getCachedWidgetData` is called via `useDashboardStore.getState().getCachedWidgetData(widget.id)` inside `fetchWidgetData` (line 58) — not a hook dependency. `queryFn` deps correctly reduced to `[widget.id, widget.enabled, widget.dataSource, cacheWidgetData]`.

**Bug:** Line 95: `staleTime: (refreshInterval - 5) * 1000` can be negative. `useBatchWidgetData` line 156: `staleTime: minInterval - 5000` has the same pattern.

### `src/components/dashboard/layout/DashboardGrid.tsx`

**DASH-04 implementation:** Correct. `useWindowSize` import removed (grep confirmed 0 occurrences). `containerWidth` initialized with lazy `useState(() => typeof window !== 'undefined' ? window.innerWidth : 1200)` (line 50-52). ResizeObserver in `useEffect` (line 55-66) updates `containerWidth`. `gridProps` is fully useMemoized with correct deps (line 184-222). `handleLayoutChange` and drag/resize callbacks are stable via `useCallback` with stable deps.

**Note:** `children` in gridProps deps is correct per React behavior — if parent passes new children reference, grid should re-render.

### Dashboard-system Components (9 files)

All verified clean via grep — zero bare `useDashboardStore()` calls remain. Each file uses per-field selector pattern `useDashboardStore((s) => s.field)`. Effect dependency arrays are unchanged (field names unchanged, only source changed from destructured variable to selector expression).

---

_Reviewed: 2026-09-14_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
