---
phase: quick
plan: 260527-gra
subsystem: ad-auth
tags: [ad, model, migration, bugfix]
dependency_graph:
  requires: []
  provides: [user-model-ad-fields]
  affects: [ad-login-dept-sync, ad-user-ou-sync]
tech_stack:
  added: []
  patterns: [gorm-column-tags, sql-migration]
key_files:
  created:
    - internal/core/db/migrations/139_add_ad_ou_dn_user_fields.sql
  modified:
    - internal/models/user.go
decisions:
  - "Kept AdUserDn separate from existing AdDn field -- both store user DN but are written at different points in the AD login flow"
metrics:
  duration: 4m
  completed: "2026-05-27"
---

# Quick Task 260527-gra: Add AD OU/User DN Fields to User Model Summary

Add AdOuDn, AdUserDn, and AdSyncedAt fields to User model to fix SQL "column does not exist" errors during AD user login department sync.

## Tasks Completed

| Task | Name | Status | Commit |
|------|------|--------|--------|
| 1 | Add AD fields to User model and create migration | Done | 4792f7c |
| 2 | Verify column name consistency across service code | Done (verification only) | -- |

## Changes Made

### Task 1: User Model + Migration

**internal/models/user.go** -- Added three fields after `AdDn` in the AD section:
- `AdOuDn *string` -- maps to `ad_ou_dn` column (TEXT), stores the user's AD OU distinguished name
- `AdUserDn *string` -- maps to `ad_user_dn` column (TEXT), stores the user's full AD DN (updated by OU service)
- `AdSyncedAt *time.Time` -- maps to `ad_synced_at` column (TIMESTAMPTZ), tracks last AD sync time

**internal/core/db/migrations/139_add_ad_ou_dn_user_fields.sql** -- Explicit migration with:
- Three ALTER TABLE ADD COLUMN IF NOT EXISTS statements
- Partial index on `ad_ou_dn` for AD sync query performance

### Task 2: Verification

Confirmed all column name references in service code match the model GORM tags:
- `user_ou_service.go:69-71` map keys `"ad_user_dn"`, `"ad_ou_dn"`, `"ad_synced_at"` -- match
- `user_ad_sync_service.go:115` `Update("ad_ou_dn", ...)` -- match
- `user_ad_sync_service.go:175` `Update("ad_synced_at", ...)` -- match
- `user_ad_sync_service.go:254` `Update("ad_ou_dn", ...)` -- match
- Test files in `user_ou_service_test.go` already reference these columns in SQLite DDL

## Deviations from Plan

### Pre-existing Issues (Out of Scope)

1. **Build error:** `internal/api/router.go:477` references undefined `SetupADGroupMappingRouter`. Pre-existing, unrelated to this task.
2. **Vet error:** `dept_sync_service_test.go:147` references undefined `ADConfig`. Pre-existing, unrelated to this task.

Neither issue was introduced by this plan's changes.

## Verification Results

- `go build ./...` -- fails due to pre-existing `SetupADGroupMappingRouter` error (unrelated)
- Model fields compile correctly with `go vet ./internal/models/...` (passes)
- All service code column references match model GORM tags (manual verification)
- Test files already expect these columns to exist

## Commits

- `4792f7c` feat(ad): add AdOuDn, AdUserDn, AdSyncedAt fields to User model

## Self-Check: PASSED

All files and commits verified present.
