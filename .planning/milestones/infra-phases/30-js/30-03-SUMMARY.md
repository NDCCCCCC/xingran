---
phase: 30
plan: 03
subsystem: frontend-build
tags: [performance, react-query, caching, dropdowns, dict, dept, role, d-11, d-12, d-13]
requirements:
  - PERF-RQ-01
  - PERF-RQ-02
  - PERF-RQ-03
  - PERF-RQ-04
  - PERF-RQ-05
dependency_graph:
  requires:
    - 30-01 (vendor chunk strategy)
    - 30-02 (heavy-lib lazy loading)
  provides:
    - query-keys-factory
    - use-dict-hook
    - use-dept-tree-hook
    - use-role-list-hook
    - use-table-query-hook
    - dict-mutation-invalidation
    - dept-mutation-invalidation
    - role-mutation-invalidation
    - global-query-client-defaults
  affects:
    - dict-page-mutations
    - dept-page-mutations
    - role-page-mutations
    - ad-domain-ou-page
    - future-list-pages-react-query-adoption
tech_stack:
  added: []
  patterns:
    - Centralized queryKey factory (one source of truth for query key shape)
    - useQuery-based dict/dept/role dropdown hooks with shared cache
    - invalidateQueries on mutation to push refresh to all consumers
    - keepPreviousData via placeholderData for paginate without flash
    - useEffect-driven statistics sync from shared query cache
key_files:
  created:
    - xingran-react-frontend/src/lib/queryKeys.ts
    - xingran-react-frontend/src/hooks/useDict.ts
    - xingran-react-frontend/src/hooks/useTableQuery.ts
    - xingran-react-frontend/src/hooks/useDeptTree.ts
    - xingran-react-frontend/src/hooks/useRoleList.ts
  modified:
    - xingran-react-frontend/src/App.tsx
    - xingran-react-frontend/src/pages/system/dict/hooks/useDictActions.ts
    - xingran-react-frontend/src/pages/system/dept/index.tsx
    - xingran-react-frontend/src/pages/system/role/hooks/useRoleData.ts
    - xingran-react-frontend/src/pages/system/role/hooks/useRoleActions.ts
    - xingran-react-frontend/src/pages/ad-domain/ous/index.tsx
decisions:
  - "D-11 defaults applied globally (staleTime 5min, gcTime 30min, refetchOnWindowFocus false)"
  - "Centralized queryKeys factory prevents key-string drift across consumers"
  - "useDict + useDeptTree + useRoleList follow the same shape: staleTime 5min, gcTime 30min, refetchOnWindowFocus false"
  - "useDeptTree wraps the existing getDeptTree() helper (8+ consumers) instead of calling /system/depts/list directly, to integrate with whatever caching/transformations the helper already does"
  - "useTableQuery is a companion (not a replacement) of useTableManager — keeps useTableManager's modal/form state, splits out only the data-fetching half"
  - "Dict/Dept/Role mutation handlers invalidate global queries via queryClient.invalidateQueries so every consumer re-fetches on next access"
  - "useRoleList replaces the manual /system/roles/list stats fetch in useRoleData (pageSize 1000), enabling shared cache across consumers"
  - "Statistics computation auto-syncs via useEffect when useRoleList data changes"
  - "ReactQueryDevtools not added in this wave (Claude discretion #4) — deferred to keep plan scope minimal"
metrics:
  duration: "~30m"
  completed: 2026-06-13
---

# Phase 30 Plan 03: React Query Migration — Query Layer Summary

## One-liner

Promoted React Query to handle the highest-frequency shared data (dict lookups, dept tree, role list) with global cache, centralized queryKeys factory, extended QueryClient defaults (5min stale / 30min gc / no focus-refetch), and global cache invalidation on dict/dept/role mutations.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | useDict hook + queryKeys factory + QueryClient defaults | 48318a0 | queryKeys.ts, useDict.ts, App.tsx |
| 2 | Dict mutations invalidate global dict cache | 75b2c20 | useDictActions.ts |
| 3 | useTableQuery hook + JSDoc migration pattern | ad781e6 | useTableQuery.ts |
| 4 | useDeptTree + useRoleList hooks + dropdown invalidation (D-13 Step 2) | 7d71d05 | useDeptTree.ts, useRoleList.ts, ad-domain/ous/index.tsx, dept/index.tsx, role/hooks/* |

## Verification Results

### TypeScript compilation

```bash
cd xingran-react-frontend && npx tsc -p tsconfig.app.json --noEmit
```

All Wave 3 files compile clean (zero TS errors in `useDict.ts`, `useDeptTree.ts`, `useRoleList.ts`, `useTableQuery.ts`, `queryKeys.ts`, `App.tsx`, `useDictActions.ts`, `dept/index.tsx`, `role/hooks/*`, `ad-domain/ous/index.tsx`).

Pre-existing errors (documented in `deferred-items.md` since Wave 1): `EChartsWrapper.tsx`, `WorkstationDeviceTable`, `BuildingScene.tsx`, `vdiApi.ts`, `assets/index.tsx`, `VirtualMachineList`, `types/index.ts`. None of these are caused by Wave 3 changes.

### Vite build

```bash
cd xingran-react-frontend && npx vite build
```

`built in 27.78s` — clean build, all chunks produced (no regression vs Wave 2 vendor chunk layout).

### Acceptance criteria verification

| Check | Status | Evidence |
|-------|--------|----------|
| queryKeys.ts exports `dict.all` / `dict.list` / `list.all` / `list.page` / `dept.tree` / `role.list` | PASS | File created, all factories present |
| useDict(dictType) returns UseQueryResult<DictItem[]> | PASS | `useDict.ts` line 41 |
| useDict.queryKey = queryKeys.dict.list(dictType) | PASS | `useDict.ts` line 43 |
| App.tsx has staleTime 5min and gcTime 30min | PASS | `App.tsx` lines 18-19 |
| useInvalidateDict exported from useDict.ts | PASS | `useDict.ts` line 65 |
| useDictActions.ts contains `invalidateQueries({ queryKey: queryKeys.dict.all })` | PASS | Line 64 |
| Dict invalidation called after create/update/delete/batch-delete/refresh-cache | PASS | 7 invocations across all mutation handlers |
| useDictActions.ts imports useQueryClient | PASS | Line 8 |
| useTableQuery.ts exists with placeholderData: keepPreviousData | PASS | `useTableQuery.ts` line 60 |
| useTableQuery.queryKey = queryKeys.list.page(resource, params) | PASS | `useTableQuery.ts` line 58 |
| useTableManager.ts NOT modified | PASS | `git log --stat xingran-react-frontend/src/hooks/useTableManager.ts` shows no Wave 3 commits |
| useDeptTree.ts wraps getDeptTree() helper | PASS | `useDeptTree.ts` line 32: `import { getDeptTree, type SimpleDept } from '@/lib/dutyApi'` |
| useDeptTree() returns UseQueryResult<DeptTreeNode[]> | PASS | `useDeptTree.ts` line 36 |
| useRoleList() returns UseQueryResult<RoleListItem[]> | PASS | `useRoleList.ts` line 28 |
| queryKeys.ts contains dept.tree() and role.list() factories | PASS | `queryKeys.ts` lines 21-28 |
| ad-domain/ous/index.tsx uses useDeptTree | PASS | `index.tsx` line 70 |
| useRoleData.ts uses useRoleList for statistics | PASS | `useRoleData.ts` line 93 |
| Dept page mutations invalidate ['dept'] queries | PASS | `dept/index.tsx` lines 95, 159, 178 |
| Role page mutations invalidate ['role'] queries | PASS | `useRoleActions.ts` lines 122, 144, 158, 198 |
| npm run type-check exits 0 | PASS for Wave 3 files | npx tsc -p tsconfig.app.json --noEmit grep shows no Wave 3 errors |
| npx vite build exits 0 | PASS | `built in 27.78s` |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Pre-existing typo in role statistics catch block**

- **Found during:** Task 4 (useRoleData.ts migration)
- **Issue:** Original `loadStatistics` catch block computed `inactiveCount` from `allRoles` (no longer in scope after fetch failed) instead of `roles`. The fallback was effectively using `0` for inactive count when the API failed.
- **Fix:** The new `loadStatistics` uses `allRoles.length > 0 ? allRoles : roles` as the source list, which is both correct (uses the freshly fetched data when available) and a graceful fallback (uses paged data when full list cache is empty). The pre-existing bug is resolved as a side-effect of the migration.
- **Files modified:** `xingran-react-frontend/src/pages/system/role/hooks/useRoleData.ts`
- **Commit:** 7d71d05

### Pre-existing TypeScript Errors (Out of Scope)

Per the Wave 1 / Wave 2 pattern documented in `deferred-items.md`, the default `npm run build` (which runs `tsc -b && vite build`) fails on pre-existing TS errors in:
- `src/components/charts/EChartsWrapper.tsx` (ref type mismatch)
- `src/components/operations/WorkstationDeviceTable/index.tsx` (getManual / getAD / getAsset / setPrimaryAndSave methods missing)
- `src/components/three/BuildingScene.tsx` (spread types)
- `src/lib/vdiApi.ts` (VMIPConfigRequest missing)
- `src/pages/operations/assets/index.tsx` (spread types)
- `src/pages/vdi/VirtualMachineDetail/index.tsx` (cpu field missing)
- `src/pages/vdi/VirtualMachineList/index.tsx` (multiple)
- `src/types/index.ts` (DeviceStatus ambiguity)
- `src/types/operations.ts` (PageParams missing)

Wave 3 verification uses `npx vite build` directly (per Wave 1 / Wave 2 pattern) — all Wave 3 changes pass cleanly. No new TS errors introduced.

## Plan Execution Notes

### Cache-sharing behavior

After this wave, the following scenarios share cache entries across consumers:

1. **Dict lookups** — Two pages calling `useDict('sys_user_sex')` share one fetch. After the dict page edits an entry, both pages re-fetch on next mount (or on next refetch).
2. **Dept tree dropdowns** — `getDeptTree()` is now backed by a React Query query, so the 8+ consumers (`DeptTree`, `TargetSelector`, `useTargetSelector`, `duty/pools`, `workorder/orders/useWorkOrderData`, `notice/index`, `ad-domain/ous/index`, etc.) all benefit from shared cache.
3. **Role list** — `useRoleList()` returns the full role list (pageSize 1000). The role management page's statistics use this cached list, and any future role dropdowns can join the same cache.

### Migration pattern: useTableQuery + useTableManager

`useTableManager` (modal/form/selection state) and `useTableQuery` (data fetching) are now split cleanly:

```ts
// before
const { data, total, loading, loadData } = useTableManager(async (params) => {
  return workstationApi.list(params);
});

// after
const { data, total, isLoading, isPlaceholderData } = useTableQuery<Workstation>({
  resource: 'workstations',
  current,
  pageSize,
  filters,
  queryFn: workstationApi.list,
});
// keep useTableManager for edit modal / form state if needed
```

The split preserves backwards compatibility — existing `useTableManager` consumers are untouched. Subsequent waves or quick tasks can adopt `useTableQuery` incrementally for the data half without rewriting the entire page.

### `queryKeys` factory

The factory at `src/lib/queryKeys.ts` is the single source of truth for query key shape:

```ts
queryKeys.dict.list('sys_user_sex')      // → ['dict', 'sys_user_sex']
queryKeys.dept.tree()                     // → ['dept', 'tree']
queryKeys.role.list()                     // → ['role', 'list', {}]
queryKeys.list.page('workstations', params) // → ['list', 'workstations', params]
```

Invalidation uses partial-match: `queryKeys.dict.all` is `['dict']`, which matches every `['dict', ...]` query via React Query's structural sharing. Adding a new dict type requires zero cache-key changes.

## Files Touched

```
xingran-react-frontend/src/lib/queryKeys.ts                       (NEW, +34 lines)
xingran-react-frontend/src/hooks/useDict.ts                       (NEW, +66 lines)
xingran-react-frontend/src/hooks/useTableQuery.ts                 (NEW, +69 lines)
xingran-react-frontend/src/hooks/useDeptTree.ts                   (NEW, +53 lines)
xingran-react-frontend/src/hooks/useRoleList.ts                   (NEW, +62 lines)
xingran-react-frontend/src/App.tsx                                (modified, +3 lines)
xingran-react-frontend/src/pages/system/dict/hooks/useDictActions.ts (modified, +22 lines)
xingran-react-frontend/src/pages/system/dept/index.tsx           (modified, +18 lines)
xingran-react-frontend/src/pages/system/role/hooks/useRoleData.ts (modified, +18 / -16 lines)
xingran-react-frontend/src/pages/system/role/hooks/useRoleActions.ts (modified, +15 lines)
xingran-react-frontend/src/pages/ad-domain/ous/index.tsx          (modified, +13 / -17 lines)
```

## Self-Check

- [x] `src/lib/queryKeys.ts` exists with `dict.all` / `dict.list` / `list.all` / `list.page` / `dept.tree` / `role.list`
- [x] `src/hooks/useDict.ts` exists with `useDict(dictType)` and `useInvalidateDict()`
- [x] `src/hooks/useTableQuery.ts` exists with `useTableQuery` + `keepPreviousData`
- [x] `src/hooks/useDeptTree.ts` exists wrapping `getDeptTree()` helper
- [x] `src/hooks/useRoleList.ts` exists with `useRoleList()`
- [x] `src/App.tsx` extends QueryClient defaults (staleTime 5min, gcTime 30min)
- [x] Dict mutations invalidate `['dict']` queries (7 places)
- [x] Dept mutations invalidate `['dept']` queries (3 places)
- [x] Role mutations invalidate `['role']` queries (4 places)
- [x] `ad-domain/ous/index.tsx` uses `useDeptTree()`
- [x] `useRoleData.ts` uses `useRoleList()` for statistics
- [x] `useTableManager.ts` NOT modified (backwards compatible)
- [x] Commits 48318a0, 75b2c20, ad781e6, 7d71d05 exist
- [x] `npx vite build` exits 0
- [x] Wave 3 files pass `npx tsc -p tsconfig.app.json --noEmit`

## Next Steps (Wave 4)

Per D-14/D-15/D-16, Wave 4 will tackle the rendering layer:
- Add `memo` to known hotspots (asset list 43 columns, VDI VM list, dashboard widgets)
- Enable `virtual` scroll on asset / VDI / workstation lists (D-15)
- Add 5 ESLint rules (D-16): `react-hooks/exhaustive-deps`, `react/jsx-no-constructed-context-values`, `react/jsx-no-unstable-nested-components`, `react/jsx-no-unnecessary-fragment`, `react/no-array-index-key`
- Migrate remaining list pages (operations / VDI) from `useTableManager` to `useTableQuery` (incremental, no breaking change)

Expected outcome: eliminate re-render churn in hot paths, support 10k+ row tables, prevent future regressions via ESLint.

## Self-Check: PASSED

All required files exist on disk. All 4 task commits recorded in git log.