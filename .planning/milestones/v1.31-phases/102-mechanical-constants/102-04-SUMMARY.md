# Phase 102 Plan 04: STATUS-01 — Summary

**Plan:** 102-04
**Phase:** 102-mechanical-constants
**Status:** COMPLETE
**Completed:** 2026-09-07

## Objective

STATUS-01: Replace all status numeric literals in backend scheduler/service code with models named constants, extend AST usage-point guard, and register WorkOrderStatus in value lock.

## What Was Done

### Task 1: internal/scheduler 6 files, 14 sites replaced

| File | Change |
|------|--------|
| `cron.go` | `Status: 0` → `int(JobLogStatusSuccess)`, `jobLog.Status = 1` → `int(JobLogStatusFailure)`, `Where/Update("status", 0/1)` → `JobStatusNormal/JobStatusPause`, raw SQL `ds.status = 0` → `?` placeholder with `DutyStatusNormal` |
| `vdi_sync_tasks.go` | `Where("status = ?", 0)` → `VDIServerStatusNormal`, `server.Status != 0` → `int(VDIServerStatusNormal)` |
| `workorder_tasks.go` | `Status: 0` → `JobStatusNormal` (2 sites), schedule `status = 0` → `DutyStatusNormal` |
| `reconciliation_tasks.go` | `MisfirePolicy: 1` → `MisfirePolicyImmediately`, `Status: 0` → `JobStatusNormal` |
| `mac_history_tasks.go` | `Status: 0` → `JobStatusNormal` |
| `mac_history_matview_tasks.go` | `Status: 0` → `JobStatusNormal` |

### Task 2: Services layer + InfoPointStatus raw SQL

| File | Change |
|------|--------|
| `services/scheduler/job_service.go` | `if status == 0` → `if status == int(JobStatusNormal)` |
| `services/workorder/base.go` | `status IN ? []int{0, 1}` → `[]int{int(WorkOrderStatusPending), int(WorkOrderStatusProcessing)}` (2 sites) |
| `scheduler/reconciliation_tasks.go` | Added `operationsmodels` import; PG and sqlite branches `ip.status = 0` raw SQL → `?` placeholder + `InfoPointStatusNormal` argument |

### Task 3: Test extension

- **WorkOrderStatus value lock**: `"WorkOrderStatus"` added to `watchedStatusPrefixes`; 5 values (Pending=0/Processing=1/Completed=2/Closed=3/Rejected=4) added to `expectedStatusValues`
- **TestNoStatusLiteralUsage**: New AST scan covering 7 patterns across full `internal/` backend (excluding migrations, `internal/models/`, `_test.go`)
- **Self-tests**: 8 memory-snippet cases verifying all 7 patterns + constant-reference negatives
- **Whitelist**: 11 entries with documented reasons (geocoding API, sqlite DDL, operations domain services, captcha difficulty levels)

## Commits

| Hash | Message |
|------|---------|
| `464bf99` | fix(102-04): scheduler status literals use models constants |
| `9e40c2b` | fix(102-04): services layer status literals + InfoPointStatus placeholder |
| `4b58cdf` | test(102-04): add TestNoStatusLiteralUsage AST guard + WorkOrderStatus value lock |

## Verification

```
go build ./...                                           ✓
go test ./internal/models/                                ✓ (TestStatusConstants* + TestNoStatusLiteralUsage)
go test ./internal/scheduler/                            ✓
go test ./internal/services/scheduler/                    ✓
go test ./internal/services/workorder/                    ✓
```

**Acceptance criteria:**
- grep status literals in 6 scheduler files: 0 occurrences ✓
- `models.JobStatusPause` appears exactly once in cron.go ✓
- `ds.status = ?` placeholder in cron.go:832 ✓
- `int(WorkOrderStatusPending)` appears twice in base.go ✓
- `operationsmodels.InfoPointStatusNormal` appears once in reconciliation_tasks.go ✓

## Deviations from Plan

None — plan executed as written. All 21 RESEARCH sites + 2 newly exposed sites replaced with named constants.

## Known Stubs

None.

## Threat Surface

No new threat surface introduced. All SQL replacements use placeholder parameterization (`?`), not string concatenation.
