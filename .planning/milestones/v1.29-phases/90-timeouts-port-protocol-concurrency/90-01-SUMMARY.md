# Phase 90 Plan 01: TIMEOUTS/PORT/PROTOCOL/CONCURRENCY Constants Centralization - Summary

## Plan Overview

**Plan:** 90-01
**Phase:** 90-TIMEOUTS/PORT/PROTOCOL/CONCURRENCY 常量集中化
**Status:** ✅ COMPLETED
**Completed:** 2026-09-04

## Objective

Create 4 leaf const packages (`timeouts.go` / `ports.go` / `protocol.go` / `concurrency.go`) with all 10 constants, plus 4 AST lock tests.

## Decisions Honored

- **D-01**: 4 leaf const files (timeouts.go + ports.go + protocol.go + concurrency.go)
- **D-02**: timeouts.go uses `time.Duration` strong typing
- **D-05**: No `Default` prefix (CommandConcurrency=10, SNMPPort=161)
- **D-09**: Zero business behavior change — all values match current state exactly

## Constants Created

### pkg/constants/timeouts.go (6 time.Duration constants)

| Constant | Value | Source |
|----------|-------|--------|
| `CommandExecTimeout` | 300 * time.Second | command_handler.go:61 + execution_handler.go:104 |
| `CommandReadTimeout` | 60 * time.Second | command_handler.go:98 (QuickCommand) |
| `LDAPConnTimeout` | 30 * time.Second | ad_ldap_client.go:74 |
| `ADSyncTimeout` | 30 * time.Minute | scheduler/ad_sync_tasks.go:42 (renamed from adSchedulerSyncTimeout) |
| `SchedulerShutdownTimeout` | 5 * time.Second | scheduler/cron.go:18 (renamed from defaultShutdownTimeout) |
| `ADSyncTaskTimeout` | 1 * time.Minute | scheduler/ad_sync_tasks.go:164 (extracted from inline) |

### pkg/constants/ports.go (1 int constant)

| Constant | Value | Source |
|----------|-------|--------|
| `SNMPPort` | 161 | discovery_handler.go:126 |

### pkg/constants/protocol.go (2 string constants)

| Constant | Value | Source |
|----------|-------|--------|
| `HTTPProto` | "http" | ws_notice_handler.go:47 |
| `HTTPSProto` | "https" | ws_notice_handler.go:47 |

### pkg/constants/concurrency.go (1 int constant)

| Constant | Value | Source |
|----------|-------|--------|
| `CommandConcurrency` | 10 | command_handler.go:58 + execution_handler.go:99 |

## AST Lock Tests

| Test File | Test Function | Constants Locked |
|-----------|--------------|------------------|
| `timeouts_test.go` | `TestTimeoutsConstantStability` + `TestTimeoutsConstantCount` | 6 |
| `ports_test.go` | `TestPortsConstantStability` + `TestPortsConstantCount` | 1 |
| `protocol_test.go` | `TestProtocolConstantStability` + `TestProtocolConstantCount` | 2 |
| `concurrency_test.go` | `TestConcurrencyConstantStability` + `TestConcurrencyConstantCount` | 1 |

## Verification

```
go build ./pkg/constants/...  ✅ 0 errors
go test ./pkg/constants/...   ✅ 0 failures (10 tests pass)
go build ./...                ✅ 0 errors
```

## Files Created

| File | Constants | Lines |
|------|-----------|-------|
| `pkg/constants/timeouts.go` | 6 | 30 |
| `pkg/constants/ports.go` | 1 | 9 |
| `pkg/constants/protocol.go` | 2 | 13 |
| `pkg/constants/concurrency.go` | 1 | 10 |
| `pkg/constants/timeouts_test.go` | AST lock | 150 |
| `pkg/constants/ports_test.go` | AST lock | 89 |
| `pkg/constants/protocol_test.go` | AST lock | 88 |
| `pkg/constants/concurrency_test.go` | AST lock | 85 |

## Key Implementation Details

- **time.Duration parsing**: The AST parser handles `*ast.BinaryExpr` (e.g., `300 * time.Second`) by extracting the integer multiplier and the selector expression (`time.Second` / `time.Minute`)
- **Go duration string format**: `time.Duration.String()` uses the largest evenly-dividing unit (e.g., 300s = "5m0s", 60s = "1m0s") — expected values in tests reflect this
- **Pure const blocks**: All 4 files contain only `const (...)` blocks, no functions, following Phase 89 pagination.go pattern
- **No `Default` prefix**: Following D-05 decision, constants are named without `Default` prefix

## Phase Requirements

- **TIMEOUTS-01**: timeouts.go 6 constants ✅
- **TIMEOUTS-02**: protocol.go 2 constants ✅
- **TIMEOUTS-08**: AST lock tests ✅

## Commits

- `feat(90-01): add timeout/port/protocol/concurrency constants and AST lock tests`
  - 8 files created (4 const files + 4 test files)
  - All 10 constants defined
  - All tests passing
