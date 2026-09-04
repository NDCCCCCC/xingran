# Phase 90 Plan 03: LDAP/WS/Scheduler Constants Migration - Summary

## Plan Overview

**Plan:** 90-03
**Phase:** 90-TIMEOUTS/PORT/PROTOCOL/CONCURRENCY 常量集中化
**Status:** COMPLETED
**Completed:** 2026-09-04

## Objective

Migrate LDAP client, WebSocket notice handler, and scheduler cron constants to reference the centralized constants package. Replace 1 LDAP timeout + 1 WS protocol pair + 3 scheduler timeout constants (including 2 renames).

## Decisions Honored

- **D-02**: timeouts.go uses `time.Duration` strong typing -- `LDAPConnTimeout` is `time.Second*30` → direct use with `SetTimeout()`
- **D-03**: Bare const concatenation for WS protocol -- `constants.HTTPProto+"://"+host`
- **D-07**: Scheduler extension -- `adSchedulerSyncTimeout` → `pkgconstants.ADSyncTimeout`; `defaultShutdownTimeout` → `constants.SchedulerShutdownTimeout`; inline `1*time.Minute` → `pkgconstants.ADSyncTaskTimeout`
- **D-08**: Renames -- `adSchedulerSyncTimeout` → `ADSyncTimeout`, `defaultShutdownTimeout` → `SchedulerShutdownTimeout`
- **D-09**: Zero business behavior change -- all values match current state exactly

## Changes Made

### internal/services/ad_ldap_client.go

| Line | Before | After |
|------|--------|-------|
| 74 | `c.conn.SetTimeout(time.Second * 30)` | `c.conn.SetTimeout(constants.LDAPConnTimeout)` |
| imports | `time` package | removed (unused after migration) |
| imports | -- | added `pkg/constants` |

### internal/api/v1/ws_notice_handler.go

| Line | Before | After |
|------|--------|-------|
| 48 | `"http://"+host` and `"https://"+host` | `constants.HTTPProto+"://"+host` and `constants.HTTPSProto+"://"+host` |
| imports | -- | added `pkg/constants` |

### internal/scheduler/ad_sync_tasks.go

| Line | Before | After |
|------|--------|-------|
| 44-45 | `const adSchedulerSyncTimeout = 30 * time.Minute` | removed (now imported from `pkg/constants`) |
| 164 | `context.WithTimeout(ctx, 1*time.Minute)` | `context.WithTimeout(ctx, pkgconstants.ADSyncTaskTimeout)` |
| 235 | `adSchedulerSyncTimeout` | `pkgconstants.ADSyncTimeout` |
| 350 | `adSchedulerSyncTimeout` | `pkgconstants.ADSyncTimeout` |
| imports | -- | added `pkgconstants "github.com/xingran-next/xingran-go-backend/pkg/constants"` (aliased to avoid conflict with `internal/constants`) |

**Note**: `ad_sync_tasks.go` imports `internal/constants` for `MaxConcurrentADSync`, so `pkg/constants` is imported as alias `pkgconstants` to avoid redeclaration.

### internal/scheduler/cron.go

| Line | Before | After |
|------|--------|-------|
| 17-18 | `const defaultShutdownTimeout = 5 * time.Second` | removed (now imported from `pkg/constants`) |
| 264 | comment mentions `defaultShutdownTimeout` | comment updated to `constants.SchedulerShutdownTimeout` |
| 266 | `time.After(defaultShutdownTimeout)` | `time.After(constants.SchedulerShutdownTimeout)` |
| imports | -- | added `pkg/constants` |

## Success Criteria

| # | Criterion | Status |
|---|-----------|--------|
| 1 | ad_ldap_client.go:74 uses constants.LDAPConnTimeout | PASS |
| 2 | ws_notice_handler.go:48 uses constants.HTTPProto and constants.HTTPSProto | PASS |
| 3 | ad_sync_tasks.go no longer declares adSchedulerSyncTimeout; uses pkgconstants.ADSyncTimeout | PASS |
| 4 | ad_sync_tasks.go:164 uses pkgconstants.ADSyncTaskTimeout | PASS |
| 5 | cron.go no longer declares defaultShutdownTimeout; uses constants.SchedulerShutdownTimeout | PASS |
| 6 | go build ./... 0 errors | PASS |
| 7 | go test ./... 0 failures | PASS |

## Phase Requirements

- **TIMEOUTS-05**: ad_ldap_client LDAPConnTimeout migration
- **TIMEOUTS-06**: ws_notice_handler WS protocol migration
- **TIMEOUTS-07**: scheduler/cron renames (ADSyncTimeout, SchedulerShutdownTimeout, ADSyncTaskTimeout)

## Commits

- `feat(90-03): migrate LDAP/WS/scheduler constants to pkg/constants`
  - internal/services/ad_ldap_client.go -- LDAPConnTimeout
  - internal/api/v1/ws_notice_handler.go -- HTTPProto/HTTPSProto
  - internal/scheduler/ad_sync_tasks.go -- ADSyncTimeout + ADSyncTaskTimeout
  - internal/scheduler/cron.go -- SchedulerShutdownTimeout

## Verification Results

```
go build ./...  ✅ 0 errors
go test ./...   ✅ 0 failures (all packages pass)
```
