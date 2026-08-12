---
status: warnings
phase: 45-r4
verified: 2026-06-29T00:00:00Z
verifier: gsd-code-reviewer
files_reviewed: 30
critical: 2
warning: 4
info: 3
total: 9
build: PASS
tests: PASS
---

# Code Review: Phase 45 R4

## Summary

Phase 45 R4 implements the read-side integration of the asset reconciliation engine into the workstation and asset UIs, adds the `/asset/reconciliation/by-workstation` aggregate endpoint, wires cache invalidation into the existing R2 scheduler and R2 resolve handler, and adds cross-module permission degradation (silent hide). The implementation faithfully follows the CONTEXT/PLAN documents, builds and tests pass per the verification report, and the operlog/cache-invalidation race ordering (invalidate → operlog → response) is honored in `ResolveException`.

That said, several real defects surfaced during adversarial review. The most significant is a frontend regression: the assets list page passes a hard-coded `conflictType={null}` to every `HealthBadge`, which means the badge will always render as "healthy" (green) and the drawer can never receive a real conflict type. The page also never calls `useWorkstationHealth` so the lift-up cache is empty — every row's drawer open will trigger a brand-new `/by-workstation` fetch. The reconciliation service's `WorkstationIDForException` (and the inline duplicate inside `ResolveException`) use `Row().Scan()` and check for `gorm.ErrRecordNotFound` — but `Row().Scan` returns `sql.ErrNoRows` for the empty case, so the error code-mapping is dead. There is also a minor consistency bug in `computeByWorkstation` where the score formula path skips but later sets `Score=100` unconditionally.

## Critical Findings

### CR-01: Assets-list HealthBadge always renders healthy green; drawer never opens with a real conflict type

- file: `xingran-react-frontend/src/pages/operations/assets/index.tsx:439-460`
- issue: The "对账健康" column in the assets list passes `conflictType={null}` as a literal. `HealthBadge` (`HealthBadge.tsx:46-50`) treats `null`/`""` as healthy (green) and never wires up `onClick`, so users cannot click through to the drawer for any asset. This silently breaks the entire R4 SC2/SC3/SC4 user journey on the asset side. The comment "Plan 02 接入工位关联后填充" confirms this is a known TODO, but it was not closed in 45-02.
- fix: Either (a) replace the literal with `null` only when no data is available and document the limitation, or (b) make the column fetch `useWorkstationHealth(workstationId ?? null)` at the page level (mirroring the workstation page's `assetConflictMap` lift-up pattern) and look up the per-asset conflict type from `data.assets`. Since Asset row lacks `workstationId` on the front-end model today, the second option requires a `workstationId` column on the asset list page or a new per-asset endpoint.

### CR-02: `WorkstationIDForException` and the inline duplicate inside `ResolveException` cannot detect "no row found" — cache invalidation silently no-ops

- file: `internal/services/asset/reconciliation_workorder.go:491-514` and `internal/api/v1/asset/reconciliation_handler.go:164-177`
- issue: Both call sites use `Row().Scan(&wsID)` after `Limit(1)`, and check `errors.Is(err, gorm.ErrRecordNotFound)`. `gorm.Row()` returns `*sql.Row` from database/sql, whose `.Scan()` returns `sql.ErrNoRows` when no rows match — never `gorm.ErrRecordNotFound`. The `ErrRecordNotFound` branch is therefore dead code, and the path instead falls through to the bottom error branch which bubbles up to `logrus.Warnf`/`applogger.Warnf` even when the result is the expected "no row" outcome. Worse, `ResolveException` swallows the err into `_ =`, so the warning is dropped — but `WorkstationIDForException` (called from the scheduler) returns the error. Combined with `Limit(1)` semantics this masks real DB failures behind "no workstation" misses.
- fix: Change the error check to `errors.Is(err, sql.ErrNoRows)` (and return `("", nil)`) and treat any other error as a real failure:

```go
err := ...Row().Scan(&wsID)
if err != nil {
    if errors.Is(err, sql.ErrNoRows) {
        return "", nil // expected: exception has no associated workstation
    }
    return "", err   // real DB failure, propagate
}
```

Apply the same fix in `ResolveException` (or factor out a shared helper in `reconciliation_workorder.go` and call it from both sites — the duplication itself is a smell).

## Warnings

### WR-01: `workstationHealthQuery` is mounted but never consumed on the workstations page; the lift-up optimization claim is half-true

- file: `xingran-react-frontend/src/pages/operations/workstations/index.tsx:128-139`
- issue: The page calls `useWorkstationHealth(expandedWorkstationId)` and computes `assetConflictMap`, then passes it down. That works for the expanded row. But `useReconciliationVisibility()` is also called unconditionally, so for non-permissioned users the query is enabled-but-invisible. That part is fine. The problem: when the user clicks a HealthBadge in the asset device sub-table, the drawer opens with `workstationId` set, but `useAssetHealth(selectedAssetId, workstationId)` in the drawer calls `useWorkstationHealth` again — at that point `expandedWorkstationId` may already have been reset if the user collapsed the row. The cache key is the same so it hits, but the mount is fragile: if a future refactor splits `ReconciliationDrawer` to a different provider tree it would N+1 again.
- fix: Move `workstationHealthQuery` (or a fetched `ByWorkstationResponse`) into a React context at the page level; both `WorkstationDeviceTable` and `ReconciliationDrawer` consume it. Alternatively, hoist into `queryKeys` and use `useQuery` with `initialData` from a parent suspense boundary.

### WR-02: `computeByWorkstation` sets `Score` to 100 only on the early-return branch but still calls `setExceptionHit` and `Trend` only on the no-asset branch

- file: `internal/services/asset/reconciliation_service.go:710-716`
- issue: When `len(assets)==0` the code does an early-return with `Score=100` and Trend. When assets exist, `Score` is computed correctly afterward. The intent is correct, but the early-return path is brittle — if the upstream "b) 关联资产 ID 列表" ever returns an error (e.g., transient PG outage on `ops_asset`), the caller already returns the error and never hits this branch, so it's fine in practice. The risk: future contributors may add code after this early-return expecting it to run in the empty-assets case (e.g., to populate Trend). Pull `computeHealthTrend(ctx, 7)` into a deferred call so both paths converge.
- fix: Refactor to a single linear flow that always computes Trend and conditionally sets `Score=100` when Total is 0.

### WR-03: `HealthCard.tsx:101` has `void trendOption` followed by reusing the same `trendOption` value inline in JSX — dead statement and confusing

- file: `xingran-react-frontend/src/components/reconciliation/HealthCard.tsx:101,123-129`
- issue: `const void trendOption` is a no-op statement that does nothing; the JSX still uses `trendOption` from the closure. Likely a leftover from an earlier draft. Pure noise that obscures intent.
- fix: Delete line 101.

### WR-04: `useEffect` dependency in `pages/operations/assets/index.tsx:283` includes functions from `useTableManager` that may not be stable

- file: `xingran-react-frontend/src/pages/operations/assets/index.tsx:278-283`
- issue: Pre-existing, not introduced by R4, but R4 added new R4 code (`useReconciliationVisibility`) in the same file. Not a regression — flagged because the file now has more consumers of the same `useEffect`. `useTableManager.loadData` etc. should be wrapped in `useCallback` by the hook itself; if not, every render re-fetches.
- fix: Audit `useTableManager` for stable callback identity; add a useCallback wrapper at the call site if it isn't stable.

## Info

### I-01: `reconciliation_handler.go:159-177` duplicates the `WorkstationIDForException` query instead of calling the service method

- file: `internal/api/v1/asset/reconciliation_handler.go:164-177`
- issue: The handler reaches into `reconciliation_normalized` and `sys_data_reconciliation` directly via `core.GetDB()`, replicating the query from `reconciliation_workorder.go:WorkstationIDForException`. The service method already exists; the duplication violates Handler-Service layering and creates two places to maintain.
- fix: Inject the workorder service into the handler (or expose `WorkstationIDForException` as a top-level helper in the asset package) and call it.

### I-02: `WorkstationIDForException` is wired into scheduler but not into the resolve handler — the resolve handler bypasses it

- file: `internal/api/v1/asset/reconciliation_handler.go:164-177`
- issue: The Plan-02 work adds `woSvc.InvalidateWorkstationHealth` in the scheduler path, but the resolve path uses the duplicated inline query plus `asset.InvalidateWorkstationHealth`. Functionally equivalent; just inconsistent.
- fix: Either consolidate on the service method, or accept the duplication and add a comment.

### I-03: `HealthCard.tsx:37` types trend points inline (`t: { date: string; openCount: number }`) instead of importing `TrendPoint` from `@/lib/assetApi`

- file: `xingran-react-frontend/src/components/reconciliation/HealthCard.tsx:37`
- issue: `TrendPoint` is already exported from `assetApi.ts:36-46`. Re-declaring the shape here is a maintenance risk if fields are added.
- fix: `import { ..., type TrendPoint } from "@/lib/assetApi"` and use `TrendPoint` in the map callback.

---

## Fix Log

Applied 2026-06-29 via `/gsd:code-review --fix` workflow. All Critical and Warning findings addressed; Info findings (I-01/I-02/I-03) skipped per fix_scope=critical_warning.

| ID  | Severity | Status | Commit      | Notes |
|-----|----------|--------|-------------|-------|
| CR-01 | Critical | Fixed | `266b2e55` | Assets page HealthBadge → literal '-' placeholder + comment (workstationId not in /ops/asset/list DTO, lift-up pattern not feasible without backend DTO extension) |
| CR-02 | Critical | Fixed | `b41e80c4` | Both WorkstationIDForException and ResolveException now check sql.ErrNoRows; real DB errors still propagate with Warnf |
| WR-01 | Warning  | Fixed | `84c65f9d` | Added comment block documenting cache-key contract for workstationHealthQuery lift-up |
| WR-02 | Warning  | Fixed | `523cfaec` | Trend computed once at top; Score=100 only on Total==0 branch; redundant Trend call removed |
| WR-03 | Warning  | Fixed | `6d94fec0` | Deleted dead `void trendOption` statement in HealthCard.tsx:101 |
| WR-04 | Warning  | Fixed | `ad71c988` | useEffect deps audited — all 4 deps already stable (useCallback); comment added |
| I-01  | Info     | Skipped | n/a | Out of fix scope |
| I-02  | Info     | Skipped | n/a | Out of fix scope |
| I-03  | Info     | Skipped | n/a | Out of fix scope |

**Build:** PASS (go build ./...)  
**Tests:** PASS (go test ./internal/services/asset/... ./internal/api/v1/asset/... ./internal/scheduler/...)  
**TypeScript:** PASS (tsc --noEmit)  
**Vitest:** PASS (src/components/reconciliation — 9 tests across 2 files)

**Final status:** All in-scope findings fixed. No skipped Critical/Warning findings. Build + tests green.

---

_Reviewed: 2026-06-29T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Fixer: Claude (gsd-code-fixer)_
_Depth: standard_
