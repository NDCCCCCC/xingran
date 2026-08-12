---
phase: 48-device-component-serials-planned
plan: 01
subsystem: operations (ops_asset + sys_data_reconciliation)
tags: [schema-migration, gorm, default-filter, reconciliation, phase-48]
requires:
  - Phase 42 R1 reconciliation base (migration_168/169 + uniq_recon_asset_type_open) — shipped
  - migration_148 ops_asset base — shipped
provides:
  - ops_asset 4 new columns (parent_asset_id / source_device_id / component_type / component_slot)
  - sys_data_reconciliation.recon_category sibling column
  - partial unique index uniq_recon_asset_type_cat_open (asset_id, conflict_type, recon_category) WHERE open
  - dict asset_reconciliation_recon_category (component_serial / future_expansion)
  - asset_service.List()/Statistics() default exclude component_type IS NOT NULL
affects:
  - assetService.List/Statistics default behavior (component rows now hidden by default)
  - Excel import path GetByDeviceSN intentionally NOT affected (still resolves component SNs)
tech-stack:
  added: []
  patterns:
    - ALTER TABLE ADD COLUMN IF NOT EXISTS (PG) for non-AutoMigrate schema extension
    - DO $$ + pg_indexes count-then-create for idempotent partial unique index switch
    - count-then-insert dict seed (migration_169 pattern)
    - Service-layer hardcoded default filter (D-07)
key-files:
  created:
    - internal/core/db/migrations/migration_201_phase48_component_columns.go
    - internal/services/operations/asset_listfilter_test.go
  modified:
    - internal/models/asset.go
    - internal/models/reconciliation.go
    - internal/core/db/database.go
    - internal/services/operations/asset_service.go
    - internal/services/operations/asset_statistics_test.go
decisions:
  - Use ALTER ADD COLUMN IF NOT EXISTS for new PG columns instead of GORM AutoMigrate (avoid touching deleted_at and triggering unnecessary ALTER TYPE per xingran-gorm-sql-constraint-naming-conflict)
  - Drop uniq_recon_asset_type_open + recreate as uniq_recon_asset_type_cat_open with 3-column key so existing 124 D-anomaly rows (recon_category=NULL) remain valid under NULL semantics
  - Service-layer (not controller) hardcoded component_type IS NULL filter to cover all entry points including statistics
  - GetByDeviceSN intentionally NOT filtered so reconciliation/Excel lookup still resolves component SNs
  - Keep SQLite branch AutoMigrate-only (no partial unique index support)
metrics:
  duration: ~25min
  completed: 2026-07-04
  tasks: 2/2
  files-touched: 6
---

# Phase 48 Plan 01: Schema + Asset-List Filter Summary

Wave 1 of Phase 48 — DB schema for component serials + default filter; no runtime collector code yet.

## What Was Built

**Migration 201** (`internal/core/db/migrations/migration_201_phase48_component_columns.go`):
- PostgreSQL: `ALTER ops_asset ADD COLUMN IF NOT EXISTS` × 4 (`parent_asset_id`, `source_device_id`, `component_type`, `component_slot`)
- PostgreSQL: `ALTER sys_data_reconciliation ADD COLUMN IF NOT EXISTS recon_category VARCHAR(32)`
- 5 indexes: `idx_ops_asset_parent_asset_id`, `idx_ops_asset_source_device_id`, `idx_ops_asset_component_type`, `idx_ops_asset_component_slot`, `idx_recon_category`
- Index switch (D-06): `DROP INDEX IF EXISTS uniq_recon_asset_type_open` → `CREATE UNIQUE INDEX IF NOT EXISTS uniq_recon_asset_type_cat_open (asset_id, conflict_type, recon_category) WHERE resolved_at IS NULL AND deleted_at IS NULL` via `DO $$ ... pg_indexes ... $$` for idempotency
- Dict seed (count-then-insert): `dict_type = 'asset_reconciliation_recon_category'` + 2 `dict_data` rows (`component_serial` default, `future_expansion`)
- SQLite branch: AutoMigrate only (no partial unique index support)

**Models** (`internal/models/asset.go`, `internal/models/reconciliation.go`):
- `Asset`: 4 new `*string` fields with `gorm:"size;index;column"` tags + `json:"...omitempty"`
- `SysDataReconciliation`: new `ReconCategory *string` field with `gorm:"size:32;column:recon_category;index:idx_recon_category,priority:1"`

**Asset service default filter** (`internal/services/operations/asset_service.go`):
- `List()`: hardcoded `query = query.Where("component_type IS NULL")` immediately after `Table("ops_asset")`, before `applyFilters` (D-07)
- `Statistics()`: chained `.Where("component_type IS NULL")` after `Model(&models.Asset{})`
- `GetByDeviceSN()` intentionally NOT modified — reconciliation and Excel lookup still resolve component SNs

**Tests** (`internal/services/operations/asset_listfilter_test.go`):
- `TestAssetListExcludesComponents` — 2 main + 2 component rows → List returns 2
- `TestAssetStatisticsExcludesComponents` — 3 main + 3 component rows (one component `status=1`) → Statistics Total=3/Normal=2/Stopped=1/NBF=1
- `TestAssetListFilterDoesNotBreakExistingFilters` — `devicesn LIKE 'SWITCH'` coexists with default filter (only 1 main device matched, 2 component devices excluded)

**database.go** registration: `Migrate201Phase48ComponentColumns` called after `Migrate200FixSuggestionConfigSeeds`, log-on-error pattern (no panic/fatal).

## Commits

- `b8fd2f45` — feat(48-01): add migration_201 + Asset/SysDataReconciliation column extensions (Task 1)
- `56c9b3d3` — feat(48-01): default-filter component_type IS NULL in asset List/Statistics (Task 2)

## Verification

- `go build ./...` — clean (exit 0)
- `go vet ./internal/core/db/migrations/... ./internal/models/... ./internal/services/operations/...` — clean
- 4 Asset-related tests PASS (3 new + 1 existing `TestAssetService_Statistics`)

## TDD Gate Compliance

Both tasks marked `tdd="true"` in plan. Flow observed:

- Task 1 (no behavior test in scope per VALIDATION map — DB introspection is UAT-only): build/vet gate only; `test(...)` commit not applicable, implementation commit `feat(...)` covers it.
- Task 2: RED-first confirmed (3 tests failed with expected messages before implementation), then GREEN (3 tests pass after implementation). Single combined `feat(...)` commit because test + implementation land together (no separate `test(...)` RED commit — RED phase was verified via `go test` output before writing the implementation). No refactor step needed.

Per execute-plan.md TDD gate rules: RED phase verified for Task 2 ✓; GREEN commit present for both tasks ✓.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated `asset_statistics_test.go` fixture schema**
- **Found during:** Task 2 GREEN run
- **Issue:** Existing `TestAssetService_Statistics` test's sqlite CREATE TABLE didn't include `component_type` column; after Task 2 added `WHERE component_type IS NULL` to `Statistics()`, the test failed with `no such column: component_type`.
- **Fix:** Added `component_type TEXT` to the test fixture schema (1-line addition). This is the test's local fixture schema, not production code.
- **Files modified:** `internal/services/operations/asset_statistics_test.go`
- **Commit:** `56c9b3d3`

### Out-of-Scope Discoveries

**Pre-existing test failures** — verified via `git stash` baseline run on commit `b8fd2f45` (Task 2 base). NOT regressions caused by Phase 48-01. Logged to `.planning/phases/48-device-component-serials-planned/deferred-items.md`:

- `TestValidator_ValidateFloor/存在的楼层`
- `TestValidator_ValidateWall/存在的墙体`
- `TestValidator_ValidateDoor/存在的门`
- `TestReferenceResolver_ResolveSingle`

Root cause category: sqlite in-memory fixture missing columns referenced by production `WHERE deleted_at IS NULL` clauses. Out of scope.

## Threat Mitigation (per plan's `<threat_model>`)

| Threat | Mitigation (per plan) | Implemented |
|--------|-----------------------|-------------|
| T-48-01 Tampering (DROP/CREATE partial unique) | IF EXISTS / IF NOT EXISTS + DO $$ block idempotent | ✓ migration_201 line ~88-105 |
| T-48-02 DoS (dict seed duplicate) | count-then-insert (0 rows on second run) | ✓ seedReconCategoryDict |
| T-48-03 Info Disclosure (filter hides legit asset) | accept per plan; main devices always NULL | ✓ D-07 locked |
| T-48-SC Tampering (no new packages) | N/A — no installs | ✓ |

## Known Stubs

None — this plan adds only DB columns and a hardcoded WHERE filter; no UI render paths or data sources introduced.

## Threat Flags

None — no new endpoints, auth paths, or schema changes at trust boundaries beyond what the plan's threat_model enumerated.

## Manual / UAT Coverage (deferred)

Per VALIDATION.md, the following cannot be automated and require user site visit:

| Item | Verification |
|------|--------------|
| Migration 201 applies cleanly on PostgreSQL 18 | Start backend, observe "Migration 201:" log + `\d ops_asset` / `\d sys_data_reconciliation` introspection |
| Existing 124 D-anomaly rows still satisfy new partial unique | `SELECT COUNT(*) FROM sys_data_reconciliation WHERE resolved_at IS NULL AND conflict_type='D'` unchanged before/after migration |
| Component-anomaly UI filter works against new dict | Frontend deferred to Wave 3 (48-03) |

## Self-Check: PASSED

Files verified:
- FOUND: internal/core/db/migrations/migration_201_phase48_component_columns.go
- FOUND: internal/services/operations/asset_listfilter_test.go
- FOUND: internal/models/asset.go (4 new cols present)
- FOUND: internal/models/reconciliation.go (ReconCategory present)
- FOUND: internal/core/db/database.go (Migrate201 registration present)
- FOUND: internal/services/operations/asset_service.go (List + Statistics WHERE component_type IS NULL present)
- FOUND: .planning/phases/48-device-component-serials-planned/deferred-items.md

Commits verified:
- FOUND: b8fd2f45 (Task 1)
- FOUND: 56c9b3d3 (Task 2)
