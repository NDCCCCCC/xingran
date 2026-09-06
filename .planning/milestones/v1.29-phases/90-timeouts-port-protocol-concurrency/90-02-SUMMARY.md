# Phase 90 Plan 02: Network Handlers Constant Migration - Summary

## Plan Overview

**Plan:** 90-02
**Phase:** 90-TIMEOUTS/PORT/PROTOCOL/CONCURRENCY 常量集中化
**Status:** COMPLETED
**Completed:** 2026-09-04

## Objective

Migrate network handlers (command_handler, execution_handler, discovery_handler) to reference the new constants package constants. Replace 3 concurrency literals + 3 timeout literals + 1 SNMP port literal.

## Decisions Honored

- **D-02**: timeouts.go uses `time.Duration` strong typing -- call sites use `int(constants.Xxx.Seconds())` conversion
- **D-05**: No `Default` prefix -- use `constants.CommandConcurrency` and `constants.SNMPPort`
- **D-06**: Shared `CommandExecTimeout=300s` between command_handler.go and execution_handler.go
- **D-09**: Zero business behavior change -- all values match current state exactly

## Changes Made

### internal/api/v1/network/command_handler.go

| Line | Before | After |
|------|--------|-------|
| 58-59 | `req.Concurrency = 10` | `req.Concurrency = constants.CommandConcurrency` |
| 61-62 | `req.Timeout = 300` | `req.Timeout = int(constants.CommandExecTimeout.Seconds())` |
| 98-99 | `req.Timeout = 60` | `req.Timeout = int(constants.CommandReadTimeout.Seconds())` |

### internal/api/v1/network/execution_handler.go

| Line | Before | After |
|------|--------|-------|
| 99 | `req.Concurrency = 10` | `req.Concurrency = constants.CommandConcurrency` |
| 104 | `req.Timeout = 300` | `req.Timeout = int(constants.CommandExecTimeout.Seconds())` |

### internal/api/v1/network/discovery_handler.go

| Line | Before | After |
|------|--------|-------|
| 126 | `req.SNMPPort = 161` | `req.SNMPPort = constants.SNMPPort` |

## Verification

```
go build ./internal/api/v1/network/...  0 errors
go test ./internal/api/v1/network/...    0 failures
```

## Success Criteria

| # | Criterion | Status |
|---|-----------|--------|
| 1 | command_handler.go:58 uses constants.CommandConcurrency | PASS |
| 2 | command_handler.go:61 uses int(constants.CommandExecTimeout.Seconds()) | PASS |
| 3 | command_handler.go:98 uses int(constants.CommandReadTimeout.Seconds()) | PASS |
| 4 | execution_handler.go:99 uses constants.CommandConcurrency | PASS |
| 5 | execution_handler.go:104 uses int(constants.CommandExecTimeout.Seconds()) | PASS |
| 6 | discovery_handler.go:126 uses constants.SNMPPort | PASS |
| 7 | go build ./internal/api/v1/network/... 0 errors | PASS |
| 8 | go test ./internal/api/v1/network/... 0 failures | PASS |

## Phase Requirements

- **TIMEOUTS-03**: command_handler timeout+concurrency constants migration
- **TIMEOUTS-04**: execution_handler timeout+concurrency constants migration
- **TIMEOUTS-07**: discovery_handler SNMPPort constant migration

## Commit

- `112400a` feat(90-02): migrate network handlers to timeout/concurrency/port constants
