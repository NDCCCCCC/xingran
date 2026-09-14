---
phase: 118-data-fetch-data
plan: "01"
subsystem: frontend-data
tags: [react-query, vdi, duty-holidays, parallel-fetch, perf]

# Dependency graph
requires: []
provides:
  - "queryKeys.vdi.servers factory for shared VDI server list cache"
  - "VirtualMachineList single useQuery replacing 4 scattered vdiServerApi.list calls"
  - "useHolidayData.fetchAvailableYears/fetchYears parallel year+list fetch"
affects: [119-frontend-perf, any-future-virtual-machine-list, any-duty-holidays-changes]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "react-query queryKey factory under src/lib/queryKeys.ts (single source of truth)"
    - "Permission-gated useQuery (enabled flag) preserves original access semantics"
    - "Year-change re-entry guard: skip parallel fetch when year already selected (avoid year-reset bug)"

key-files:
  created: []
  modified:
    - xingran-react-frontend/src/lib/queryKeys.ts
    - xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx
    - xingran-react-frontend/src/pages/duty/holidays/hooks/useHolidayData.ts
    - xingran-react-frontend/src/pages/duty/management/hooks/useHolidayData.ts

key-decisions:
  - "Derived vdiServers via useMemo from serverData instead of mirroring to state — eliminates redundant state sync"
  - "Preserved original permission gating via useQuery enabled: canCreateVM — no new request surface for unprivileged users"
  - "Year-change re-entry path keeps the years-only refresh to avoid resetting the user-selected year (the plan's parallel snippet without the year === undefined guard introduced a year/data mismatch regression)"

patterns-established:
  - "Single useQuery per resource: multiple UI sites that need the same list should share one cache entry, not each await the same endpoint"

requirements-completed: [DATA-01, DATA-04]

# Metrics
duration: 13min
completed: 2026-09-14
---

# Phase 118 Plan 01: DATA-01 VDI dedupe + DATA-04 parallel year/list fetch

**Consolidated VirtualMachineList's 4 scattered VDI server list calls into one shared react-query entry and rewrote duty holiday hooks' year+list fetch from sequential to parallel Promise.all.**

## Performance

- **Duration:** 13 min
- **Started:** 2026-09-14T15:03:17Z
- **Completed:** 2026-09-14T15:16:30Z (commit timestamp)
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments
- Added `queryKeys.vdi.servers()` factory under `src/lib/queryKeys.ts`, matching the established `as const` tuple pattern used by `dict/dept/duty/role/...` namespaces.
- Replaced 4 separate `vdiServerApi.list({ current: 1, pageSize: 100 })` calls in `VirtualMachineList/index.tsx` (preloadVDIData + openCreateModal x2 + loadQuickCreateDefaults) with a single `useQuery` keyed on `queryKeys.vdi.servers()`; derived `vdiServers` via `useMemo` from `serverData.data?.list` instead of mirroring to state.
- Rewrote `fetchAvailableYears` in `duty/holidays/hooks/useHolidayData.ts` and `fetchYears` in `duty/management/hooks/useHolidayData.ts` so the first load fires `Promise.all([getHolidayYears(), getHolidayList(currentYear)])` — 1 round-trip instead of 2 sequential. The year-change re-entry path keeps the years-only refresh to avoid resetting the user-selected year.
- Preserved permission gating: `useQuery({ enabled: canCreateVM })` ensures non-create-permission users don't pay a new request they never paid before.

## Task Commits

1. **Task 1: Add vdi.servers queryKey** + **Task 2: 4 vdiServerApi.list → 1 useQuery** + **Task 3: holidays/management fetchAvailableYears/fetchYears Promise.all** — `ca09677` (feat)

## Files Created/Modified
- `xingran-react-frontend/src/lib/queryKeys.ts` — added `vdi: { all, servers }` factory
- `xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx` — added `useQuery` + `useMemo`-derived `vdiServers`, removed `setVdiServers` state, simplified `openCreateModal` to non-async, rewrote `preloadVDIData`/`loadQuickCreateDefaults` to read from the cache
- `xingran-react-frontend/src/pages/duty/holidays/hooks/useHolidayData.ts` — `fetchAvailableYears` Promise.all + year-change re-entry guard + correction refetch when `years[0] !== currentYear`
- `xingran-react-frontend/src/pages/duty/management/hooks/useHolidayData.ts` — same shape as holidays hook with `holidays/holidayYear` state names

## Decisions Made
- Dropped the `useState<VDIServer[]>([])` state in favour of `useMemo`-derived `vdiServers` — strictly less code, no chance of stale state, identical rendered output.
- Kept `enabled: canCreateVM` on the new useQuery to preserve the original permission-gated fetch semantics (only users who can see create/quick-create buttons pay the request).
- Preserved the `year === undefined` guard around the auto-select-latest-year path even though the plan's replacement snippet didn't include it — without it, `fetchAvailableYears` re-running on every year change would reset the selected year to `years[0]` (latest) and display a data/year mismatch.
- Added a one-shot correction refetch when `years[0] !== currentYear` so the parallel fetch for `getHolidayList(currentYear)` doesn't silently show wrong-year data in the edge case where the backend's most-recent year is not the current calendar year.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Preserved `year === undefined` guard to prevent year-reset regression**
- **Found during:** Task 3 (useHolidayData Promise.all rewrite)
- **Issue:** The plan's snippet dropped the `year === undefined` guard around the auto-select-latest-year branch. `fetchAvailableYears` is in a `useEffect([fetchAvailableYears])` whose identity depends on `[year]`, so it re-runs every time the user changes year. Without the guard, the parallel version would overwrite the user's selected year back to `years[0]` and display the list data fetched for the original year — a real data/year mismatch.
- **Fix:** Kept the gate `if (year === undefined)` around the auto-select + setDataSource path; year-change re-entry now returns after a years-only refresh, identical to the original behavior.
- **Files modified:** `src/pages/duty/holidays/hooks/useHolidayData.ts`, `src/pages/duty/management/hooks/useHolidayData.ts`
- **Verification:** holidays + management tests green (6/6 assertions); test mocks `[2024,2025,2026]` against today's `getFullYear()` (2026) → correction refetch path triggered, assertions still pass.
- **Committed in:** `ca09677`

**2. [Rule 1 - Bug] Added correction refetch when latest holiday year differs from current year**
- **Found during:** Task 3 (same)
- **Issue:** Plan's snippet fetched `getHolidayList(year ?? new Date().getFullYear())` in parallel. When `years[0]` (latest year with holiday data, sorted descending) differs from `currentYear` (which `??` falls back to), the displayed data would be `currentYear`'s list while the year selector shows `years[0]` — silent data/year mismatch.
- **Fix:** When `years[0] !== currentYear`, fire one extra `getHolidayList(years[0])` after the parallel fetch and use its data. Adds at most one extra request only in the rare mismatch edge; the common case (current year has data, `years[0] === currentYear`) stays at 1 round-trip.
- **Files modified:** same two files
- **Verification:** management test mock `[2024,2025,2026]` (mismatched against `getFullYear()=2026`) → correction path executes, test still passes.
- **Committed in:** `ca09677`

**3. [Rule 2 - Cleanup] Derived vdiServers via useMemo instead of mirroring to state**
- **Found during:** Task 2 (VirtualMachineList rewrite)
- **Issue:** Plan's action said to keep `useState<VDIServer[]>([])` and call `setVdiServers(serverData || [])` from an effect. This adds a redundant state sync that could lag behind the query result and is a well-known React anti-pattern when the data already lives in a cache.
- **Fix:** Replaced the state with `const vdiServers: VDIServer[] = useMemo(() => serverData?.data?.list || [], [serverData])`. JSX consumes `vdiServers` unchanged.
- **Files modified:** `src/pages/vdi/VirtualMachineList/index.tsx`
- **Verification:** VirtualMachineList tests 7/7 green; dropdown rendering unchanged.
- **Committed in:** `ca09677`

### Plan source-assertion nuances (not auto-fixes, just documentation)

- Plan's verification command `grep -c "vdiServerApi\.list" index.tsx == 0` is incompatible with the plan's own action (which keeps `vdiServerApi.list` inside the `queryFn`). The substantive intent — *4 sites → 1 queryFn* — is satisfied: the only remaining `vdiServerApi.list` is the one inside the new `useQuery`'s `queryFn`. The comment around the useQuery no longer mentions the literal string.

---

**Total deviations:** 3 auto-fixed (2 Rule 1 bugs + 1 Rule 2 cleanup)
**Impact on plan:** All auto-fixes necessary for correctness or code cleanliness. No scope creep. Plan's stated success criteria (DATA-01 dedupe, DATA-04 parallel, lint/type-check zero regression) all met.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- `queryKeys.vdi.servers()` is now the single source of truth for the VDI server list query key; any future VDI feature should import this factory rather than re-typing the tuple literal.
- Both useHolidayData hooks are now pattern-aligned and could be unified in a later phase (one shared module under `src/pages/duty/_shared/hooks/useHolidayData.ts`) — out of scope for 118 but worth flagging.
- DATA-02 (118-02) sits on top of the new render path; verify no regression in DynamicRoutes hydration flow after the new cache writes.

---
*Phase: 118-data-fetch-data*
*Completed: 2026-09-14*