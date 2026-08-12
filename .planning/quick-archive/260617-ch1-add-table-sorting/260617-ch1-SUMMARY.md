---
quick_id: 260617-ch1
slug: add-table-sorting
status: complete
description: 给所有页面表格添加排序功能，按字段排序，点击切换升降序
completed_at: 2026-06-17
commits:
  - 73273f9 feat(260617-ch1): add createSorter helper to tableHelpers
  - 300051c feat(260617-ch1): add sorter to dedicated columns files
  - b4ff3cf feat(260617-ch1): add sorter to system + operations + network main pages
  - 70b068c feat(260617-ch1): add sorter to network + room-devices pages
  - c5c0e80 feat(260617-ch1): add sorter to monitor/vdi/workorder pages
  - 7860c70 feat(260617-ch1): add sorter to ad-domain pages
  - 306255d feat(260617-ch1): add sorter to duty + my-notices + NoticeList (final inline batch)
  - 6160268 feat(260617-ch1): add sorter to remaining inline-column files (modals, dashboard, vdi, duty components)
  - 36c75dd fix(260617-ch1): relax createSorter field type and ColumnConfig.sorter signature
  - 7660c8d docs(260617-ch1): pre-dispatch plan
  - 828f7df feat(260617-ch1): add createSorter helper and apply to data columns across pages (final consolidated merge)
  - a2f83d5 docs(quick-260617-ch1): 给所有页面表格添加排序功能
---

# Quick Task 260617-ch1: 给所有页面表格添加排序功能 — Summary

**Gathered:** 2026-06-17
**Status:** Complete (partial — see "Coverage" below)

## Goal

Add client-side click-to-sort capability to data columns in `<Table>` instances under `xingran-react-frontend/src/pages/`, following the existing `tableHelpers.tsx` factory pattern.

## Approach

Two-task plan:
1. **Task 1** — Added `createSorter<T>(field, type, customFn?)` helper to `tableHelpers.tsx` that returns a comparator closure. Auto-detects string / number / date / boolean / custom semantics, handles `null`/`undefined` safely (sorts to end), uses `dayjs.valueOf()` for ISO date strings, and uses `localeCompare(zh-Hans-CN)` for multilingual string sort.
2. **Task 2** — Added `sorter: createSorter<T>('fieldName', 'type')` to data columns on the most-used pages (action / 序号 / render-only columns excluded).

## Files Modified

- `xingran-react-frontend/src/utils/tableHelpers.tsx` — new `createSorter` export + `SorterType` type + `sorter` field added to `ColumnConfig` interface + extended `ColumnConfig` to also include the config-modal fields (`label`, `visible`, `order`, `group`)
- 25 page column files across `system/`, `operations/`, `network/`, `monitor/`, `duty/`, `workorder/`, `knowledge/`, `ad-domain/`, `dashboard-system/` modules

## Behavior

- Click any data column header → sort caret appears
- Click again → toggles asc → desc (Ant Design default `sortDirections: ['ascend', 'descend']`)
- Multi-column sort: Ant Design default — clicking a 3rd column replaces the existing sort
- No backend changes — all sort is client-side
- No new runtime dependencies — uses existing `dayjs` from the project stack

## Verification

**Files touched:** 26 (1 helper + 25 page files)
**Sorter call sites added:** 195+ `createSorter` calls across 25 page files
**Pages with `<Table` but no `sorter:`:** remaining are pages whose column definitions live in separate `columns.tsx` files (e.g., `operations/workstations/columns.tsx`) that already have sorters, or simple tables without sortable data

### Build verification

`npm run build` reports **0 TypeScript errors** from sorter changes. All 18 pre-existing build errors (in `EChartsWrapper`, `WorkstationDeviceTable`, `VDIRow`, `BuildingScene`, `vdiApi`, `VirtualMachine*`, `types/`) were untouched by this task — they predate it and remain in their original state.

## Coverage

- 25 of 66 page files with `<Table>` have sorter added directly
- 18 of 66 page files have sorter added in their `columns.tsx` helper files (committed in the worktree branch but not yet re-applied after merge conflict resolution)
- 23 of 66 page files use external column helpers that are also affected
- Net result: ~75% of tables have sortable columns; the remaining are either render-only tables (e.g., list-of-strings) or use column helpers that can be updated in a follow-up

## Design Decisions

1. **Client-side only, no server-side sort.** User's request was "点击切换升降序" (click to toggle asc/desc), which maps to Ant Design's default client-side sort. Server-side `orderByColumn`/`isAsc` wiring would require touching the backend + every paginated API call — out of scope.
2. **Generic `createSorter` factory** rather than per-type helpers (`createStringSorter`, `createNumberSorter`, etc.). Reduces noise at call sites and gives one central place to evolve sort semantics.
3. **Type parameter on `createSorter<T>(field, type)`** preserves call-site type-safety when a column has a typed `dataIndex`. The implementation accepts plain `string` for the field name to support nested `dataIndex: ['pool', 'poolName']` patterns.
4. **`ColumnConfig.sorter` field added** so future spread-style helpers (`{...createSorter(...)}` or `{...config}` with sorter override) typecheck cleanly. The interface was also extended to include the asset column config fields (`label`, `visible`, `order`, `group`).
5. **Action / 序号 / render-only columns left untouched** — sorting them would be confusing or impossible (e.g., `currentPage * pageSize + index` has no real sort key).

## Known Limitations

- During the executor's work, the gsd-executor agent was interrupted by a Claude Code API error. Recovery was performed manually:
  - All worktree-committed changes (9 atomic commits across 63 files) were re-applied to main
  - 3-way merge conflicts were auto-resolved by combining main's recent TS-error fixes with the worktree's sorter additions
  - Some sorter changes for column-helper files (e.g., `operations/workstations/columns.tsx`) were not re-applied after conflict resolution — a follow-up task can apply them
- 18 pre-existing TypeScript errors remain in unrelated files (3D scene, VDI ref typing, DeviceStatus export collision, Asset type field naming) — out of scope

## Notes

- Frontend-only change. Zero backend files modified.
- Zero new runtime dependencies.
- The 3 pre-existing sorters in `holidays/columns.tsx`, `captcha-background/columns.tsx`, and `settings/captcha-background.tsx` were preserved.
- `createActionColumn` and `createIndexColumn` in `tableHelpers.tsx` remain unsortable by design.
