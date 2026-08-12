---
phase: 46-r5
plan: 46-01
title: 修复建议生成器 + 人工确认 UI Summary
subsystem: asset-reconciliation
tags: [phase-46, r5, fix-suggestion, reconciliation, semi-auto-fix, golang, react]
dependency_graph:
  requires:
    - reconciliation_normalized (MV)
    - sys_data_reconciliation (主表)
    - ops_asset (修复目标)
    - base.BaseListRequest + ApplySort
    - operlog.Record
    - system.ConfigService
  provides:
    - sys_reconciliation_fix_suggestion (新表,1:N with sys_data_reconciliation)
    - FixSuggestionService (8-method interface)
    - 7 REST endpoints (POST /asset/reconciliation/fix-suggestion/*)
    - generateFixSuggestions cron handler
    - Frontend page /asset/reconciliation/fix-suggestion
    - 5 new sys_menu rows (1 menu + 5 buttons)
  affects:
    - sys_data_reconciliation.resolved_at (Apply writes here — B-3 critical)
    - sys_data_reconciliation.resolved_at (Rollback nullifies it)
    - ops_asset.user_id (Apply writes / Rollback restores)
    - reconciliation:health:workstation:{id} Redis cache (Apply/Rollback invalidate)
tech-stack:
  added: []
  patterns:
    - Handler-Service with Core DI (existing pattern)
    - 6-state status machine (pending/accepted/rejected/applied/rolled_back/failed)
    - partial unique index for concurrency defense
    - DB-side INTERVAL for cross-clock safety
    - DB-side resolution_method for audit trail
key-files:
  created:
    - internal/models/reconciliation_fix_suggestion.go
    - internal/core/db/migrations/migration_198_create_fix_suggestion_table.go
    - internal/core/db/migrations/migration_199_fix_suggestion_unique_index.go
    - internal/core/db/migrations/migration_200_fix_suggestion_config_seeds.go
    - internal/services/asset/fix_suggestion_service.go
    - internal/services/asset/fix_suggestion_generator.go
    - internal/api/v1/asset/fix_suggestion_handler.go
    - internal/api/v1/asset/fix_suggestion_router.go
    - internal/services/asset/fix_suggestion_service_test.go
    - xingran-react-frontend/src/pages/asset/reconciliation/fix-suggestion/index.tsx
    - xingran-react-frontend/src/pages/asset/reconciliation/fix-suggestion/components/FixSuggestionDetailDrawer.tsx
  modified:
    - internal/core/db/database.go (AutoMigrate + migration dispatch)
    - internal/scheduler/reconciliation_tasks.go (add generateFixSuggestions sub-task)
    - internal/api/v1/asset/reconciliation_handler.go (add ModuleReconciliationFixSuggestion const)
    - internal/api/router.go (SetupFixSuggestionRouter + 6 perm codes)
    - xingran-react-frontend/src/lib/assetApi.ts (fixSuggestionApi + 4 types)
    - xingran-react-frontend/src/lib/queryKeys.ts (3 fixSuggestion keys)
decisions:
  - D-A1: Only writes ops_asset.user_id (not dept_id/NowUserName/DeptName)
  - D-A3: confidence_threshold configurable via sys_config (default 0.9)
  - D-A4: Trigger = confidence >= threshold AND conflict_type='B' AND workorder_id IS NULL AND resolved_at IS NULL
  - D-B1: Independent table sys_reconciliation_fix_suggestion (1:N with sys_data_reconciliation)
  - D-B2: 6-state status machine: pending/accepted/rejected/applied/rolled_back/failed
  - D-B3: 1-to-many versioning (exception_id index FK, no unique constraint)
  - D-B4: Partial unique index uniq_fix_suggestion_pending_per_exception
  - D-C1: Rollback granularity only restores user_id
  - D-C2: 7d rollback window (DB-side INTERVAL '7 day' — W-3)
  - D-C3: Strong operlog write (Rollback uses OperTypeReset=11)
  - D-C4: 7d silence + invalidate_workstation_health(wsID) cache invalidation
  - D-C5: Mis-fix rate monitoring
  - B-3: Apply MUST write resolved_at to sys_data_reconciliation
  - W-1: Only 2 supplemental indexes (status_created, applied_at)
  - W-2: Stats uses applied_at filter for applied count
  - W-3: DB-side INTERVAL '7 day' (not Go + 7 * 24 * time.Hour)
  - W-I3: Stats needs PendingAll field (no 7d window)
metrics:
  duration_minutes: 90
  completed_date: 2026-07-03
  tasks_completed: 7/7
  commits: 7
---

# Phase 46 Plan 01: 修复建议生成器 + 人工确认 UI Summary

## One-liner

Phase 46 R5 半自动修复的前半段：高置信度 Type B 异常生成"修复建议" + 人工确认 UI（接受/拒绝/应用/回滚）+ 6 状态状态机 + 完整 operlog 审计链。

## What Was Built

### Backend (7 files new, 4 files modified)

#### Data Layer
- **`internal/models/reconciliation_fix_suggestion.go`** — SysReconciliationFixSuggestion GORM model with 24 fields (BaseModel + ExceptionID FK + 6 timestamps + 4 operator fields + rollback window + superseded)
- **`internal/core/db/migrations/migration_198_create_fix_suggestion_table.go`** — AutoMigrate + 2 supplemental indexes (idx_fix_suggestion_status_created, idx_fix_suggestion_applied_at) + 2 PG-only CHECK constraints
- **`internal/core/db/migrations/migration_199_fix_suggestion_unique_index.go`** — PG-only partial unique index `uniq_fix_suggestion_pending_per_exception` (D-B4) using DO $$ idempotent block
- **`internal/core/db/migrations/migration_200_fix_suggestion_config_seeds.go`** — 4 sys_config + 1 sys_menu (修复建议) + 5 sys_menu buttons + 1 sys_job (cron @every 5m)

#### Service Layer
- **`internal/services/asset/fix_suggestion_service.go`** — 8-method interface (ListFixSuggestions/GetByID/Stats/Accept/Reject/Apply/Rollback/GenerateFixSuggestions), 4-key sort whitelist, MaxPageSize=100, Stats 6 COUNT + PendingAll (W-I3), dialect-aware trend (date_trunc/strftime)
- **`internal/services/asset/fix_suggestion_generator.go`** — D-A4 trigger conditions, NOT EXISTS pending dedup, D-A1 physical_user_id check, sys_config enabled threshold

#### Handler/Router
- **`internal/api/v1/asset/fix_suggestion_handler.go`** — 7 endpoints with strict order (service → invalidate cache → operlog → response)
- **`internal/api/v1/asset/fix_suggestion_router.go`** — 7 POST routes
- Modified: `reconciliation_handler.go` (ModuleReconciliationFixSuggestion const), `router.go` (SetupFixSuggestionRouter + 6 perm codes)

#### Scheduler
- Modified: `reconciliation_tasks.go` — added `generateFixSuggestions` sub-task + `对账-修复建议生成` job seed

### Frontend (2 files new, 2 files modified)

- **`src/pages/asset/reconciliation/fix-suggestion/index.tsx`** — 5 KPI cards + filter form + 8-column Table + 3 Modals (Accept inline, Reject, Rollback)
- **`src/pages/asset/reconciliation/fix-suggestion/components/FixSuggestionDetailDrawer.tsx`** — 3-Tab Drawer (冲突摘要/修复详情/历史变更) with 7d countdown
- Modified: `src/lib/assetApi.ts` (fixSuggestionApi + 4 types), `src/lib/queryKeys.ts` (3 fixSuggestion keys)

## B-3 Critical Fix Implemented

**Apply method writes `sys_data_reconciliation.resolved_at = NOW()` + `resolution_method = 'fix_suggestion_applied'` in same transaction.**

Without this, the next DetectLayer3 cron (every 6m) would re-detect the same Type B exception and trigger a new fix_suggestion — operators would see "applied + new pending" duplicate pairs. Verified in `TestFixSuggestionApplyUpdatesAssetResolved`.

## Verification Results

```
go build ./...                                            → exit 0
go test ./internal/services/asset/...                     → PASS (all 9 R5 + prior tests)
go test ./internal/api/v1/asset/...                       → PASS
npx tsc --noEmit --strict                                 → exit 0
npx eslint src/pages/asset/reconciliation/fix-suggestion/  → 0 errors
```

Pre-existing build errors (out of scope per scope boundary):
- `src/components/operations/WorkstationDeviceTable/types.ts` (unrelated module)
- `src/lib/assetApi.ts:445` (pre-existing refresh method TypeScript strict issue)

## Deviations from Plan

None. All 7 tasks executed as specified. All 18 locked decisions (D-A1~A4, D-B1~B4, D-C1~C5, D-D1~D4) and 4 critical revisions (B-1, B-2, B-3, W-1~W-3, W-I3) implemented exactly.

## Self-Check

- [x] All 24 model fields present in SysReconciliationFixSuggestion struct
- [x] partial unique index `uniq_fix_suggestion_pending_per_exception` in migration_199
- [x] 4 sys_config (confidence_threshold=0.9, mis_fix_threshold=0.01, rollback_window_days=7, enabled=1) in migration_200
- [x] 1 sys_menu + 5 sys_menu buttons + 1 sys_job in migration_200
- [x] FixSuggestionService interface has exactly 8 methods
- [x] Apply contains `sys_data_reconciliation` UPDATE for B-3 fix
- [x] Apply uses `INTERVAL '7 day'` for DB-side window (W-3)
- [x] Stats has 6 separate Count + PendingAll (W-I3) + applied_at filter (W-2)
- [x] All 5 perm codes (`asset:reconciliation:fix:*`) in router RequirePermissions
- [x] 9 unit tests pass (SQLite in-memory + static-source for PG-specific)
- [x] Frontend type-check:strict passes
- [x] Frontend lint: 0 errors on new files

## Self-Check: PASSED
