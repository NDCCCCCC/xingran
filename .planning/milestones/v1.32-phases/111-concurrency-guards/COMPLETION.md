# Phase 111 COMPLETION — P0 并发裸 goroutine 守护 + Captcha

**Phase:** 111 | **Status:** COMPLETED | **Date:** 2026-09-09
**Goal:** 修复 4 个裸 goroutine 无 recover 的 P0 并发问题 + Captcha Increment fail-closed，让 GUARD-07 + GUARD-08 转 GREEN

---

## Summary

4 plans (111-01..111-04) in 2 waves, all completed green. **2 GUARD tests updated/verified.**

| Wave | Plans | Commits | GOR/CAP | GUARDs |
|------|-------|---------|---------|--------|
| 1 | 111-01 | `517bfd1` | GOR-01 | **GUARD-07 ✅ RED→GREEN** |
| 1 | 111-02 | `486c297` | GOR-02 | — |
| 1 | 111-03 | `e4873458` | GOR-03 | — |
| 2 | 111-04 | `8fb8518` | GOR-04 + CAP-01 | **GUARD-08 ✅ extended + PASS** |

**Total: 4 atomic commits, 5 source files modified, 1 GUARD extended**

---

## Changes by File

### `internal/services/oper_log_service.go` (Plan 111-01, GOR-01)
**Both async goroutines now have panic recovery:**
```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            applogger.Errorf("[OPERLOG] async write panic recovered: %v", r)
        }
    }()
    if err := db.Create(operLog).Error; err != nil {
        applogger.Warnf("[OPERLOG] async write failed: %v", err)  // was: _ = err
    }
}()
```
- Two locations: L67 (RecordAsync) + L140 (RecordFromGinContext)
- Silent error swallow (`_ = err`) replaced with `applogger.Warnf`
- Closes GUARD-07 (TestOperLog_Async_DoesNotPanicOnDBError) — now GREEN

### `internal/api/v1/system/ad_dept_sync_handler.go` (Plan 111-02, GOR-02)
**Detached context + panic recovery:**
```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            applogger.Errorf("[AD-DEPT-SYNC] panic recovered: %v", r)
        }
    }()
    ctx, cancel := context.WithTimeout(context.Background(), constants.ADSyncTimeout)
    defer cancel()
    h.syncService.SyncDeptStructureToAD(ctx, ...)  // was: c.Request.Context() — cancels on HTTP return
}()
```
- **Critical fix**: `c.Request.Context()` (cancels on HTTP return) → `context.Background() + WithTimeout(ADSyncTimeout)`
- Used existing `constants.ADSyncTimeout` constant (no hardcoded duration)

### `internal/agent/server/connection_manager.go` (Plan 111-03, GOR-03)
**Two reconnect goroutines now have panic recovery:**
- L186 (`handleReconnect` async) + L196 (cleanup) wrapped in `defer recover()`
- Uses existing `applogger.WithFields` logrus pattern
- Matches `user_handler.go:200-236` authoritative pattern

### `internal/api/v1/auth.go` (Plan 111-04, GOR-04)
**Login log async goroutine now has panic recovery:**
- L598 async login log write wrapped in `defer recover()` + `[GOR-04]` log

### `internal/core/captcha.go` (Plan 111-04, CAP-01)
**Three Increment call sites now log SECURITY warn on failure:**
```go
// Before (silent failure):
_, _ = s.cache.Increment(ctx, attemptsKey)

// After (fail-closed maintained + SECURITY warn):
if _, err := s.cache.Increment(ctx, attemptsKey); err != nil {
    applogger.Warnf("[SECURITY] captcha increment failed (key=%s): %v — fail-closed: rejecting verification", attemptsKey, err)
}
```
- Locations: L379 (VerifyNormal wrong code) + L439 (VerifySlider position) + L445 (VerifySlider token)
- **Fail-closed behavior maintained** — even if Increment fails, verification still rejects
- SECURITY warn provides audit trail for brute-force protection failures

### `internal/core/captcha_test.go` (Plan 111-04, GUARD-08 extended)
**TODO-111 resolved** — test now asserts SECURITY warn was logged:
- Uses new `applogger.SetTestBuffer()` log capture helper
- 3 SECURITY warn assertions (one per call site)
- Test still PASSES (fail-closed already worked)

### `pkg/logger/logger.go` (helper for test)
- Added `SetTestBuffer()` test helper
- Restore function for safe test cleanup
- Uninitialized-state safety (no-op if logger not initialized)

---

## GUARD Tests Status

| GUARD | Test | Before Phase 111 | After Phase 111 |
|-------|------|-----------------|----------------|
| GUARD-07 | `TestOperLog_Async_DoesNotPanicOnDBError` | **FAIL** (no defer recover) | **PASS ✅** (recover + warn log) |
| GUARD-08 | `TestCaptcha_Increment_Failure_FailsClosed` | PASS (fail-closed only) | **PASS ✅ extended** (fail-closed + 3 SECURITY warns) |

```
$ go test ./internal/services/... -run "TestOperLog_Async_DoesNotPanicOnDBError" -v
--- PASS: TestOperLog_Async_DoesNotPanicOnDBError (0.00s)

$ go test ./internal/core/... -run "TestCaptcha_Increment_Failure_FailsClosed" -v
--- PASS: TestCaptcha_Increment_Failure_FailsClosed (0.06s)
```

---

## Seven Gate Verification

| Gate | Result |
|------|--------|
| `go build ./internal/services/... ./internal/api/v1/... ./internal/core/... ./internal/agent/...` | ✅ PASS (no errors) |
| GUARD-07 flipped GREEN | ✅ |
| GUARD-08 extended + PASS | ✅ |
| Existing test suite (Phase 111 scope) | ✅ No regressions |

---

## Audit Findings Closed

| Audit Finding | Resolution |
|---------------|------------|
| P0-C1: OperLogService async goroutine no recover + silent error | ✅ Closed (GOR-01 defer recover + warn log) |
| P0-C2: AD dept sync handler bare goroutine + ctx cancellation | ✅ Closed (GOR-02 detached ctx + recover) |
| P0-C3: connection_manager reconnect bare goroutines | ✅ Closed (GOR-03 recover on both) |
| P0-C4: captcha.go Increment silent failure | ✅ Closed (CAP-01 SECURITY warn + fail-closed) |

---

## Plan Implementation Notes

### Plan 111-01 (OperLog)
- Both L67 + L140 goroutines share identical pattern fix
- Replaced silent `_ = err` with `applogger.Warnf` (visible to ops)
- Test flipped RED → GREEN on first run

### Plan 111-02 (AD dept sync)
- Most critical fix of the phase: `c.Request.Context()` was causing **all AD dept syncs to fail**
- Uses existing `constants.ADSyncTimeout` (Phase 90 timeout consolidation)
- Pattern matches `user_handler.go:200-236` exactly

### Plan 111-03 (Connection manager)
- Agent-side code (separate from server)
- Two reconnect goroutines independently recovered
- Uses `WithFields` logrus pattern consistent with file

### Plan 111-04 (Auth + Captcha + GUARD-08 extension)
- 4 files modified (atomic commit `8fb8518`)
- Added `SetTestBuffer()` helper to `pkg/logger` for future test capture needs
- GUARD-08 extended from 1 assertion → 4 assertions (fail-closed + 3 SECURITY warns)

---

## Diagnostic Warnings (non-blocking)

| File | Warning | Impact |
|------|---------|--------|
| `redis.go` L387, 414, 436, 437, 443, 456, 475, 497 | `interface{} can be replaced by any` | Pre-existing style suggestions |
| `pkg/logger/logger.go` L241, 246, 251, 256, 261, 266 | `interface{} can be replaced by any` | New logger API uses interface{} for compatibility |
| `ad_authenticator_test.go` L159 | `unreachable code [default]` | Pre-existing (Phase 108) |
| `connection_manager.go` L152, 235, 239 | minor style | Pre-existing |

These are cosmetic; will be cleaned up in subsequent phases.

---

## Key Artifacts

- **PLAN.md**: `.planning/phases/111-concurrency-guards/PLAN.md`
- **Plan summaries**: `111-01-SUMMARY.md`, `111-02-SUMMARY.md`, `111-03-SUMMARY.md`
- **Source files modified**: 5 (`oper_log_service.go`, `ad_dept_sync_handler.go`, `connection_manager.go`, `auth.go`, `captcha.go`)
- **Test files modified**: 1 (`captcha_test.go` — GUARD-08 extension)
- **Helper added**: `pkg/logger/logger.go` (SetTestBuffer for test capture)

---

## Next Step

Execute **Phase 112** — P1 handler 收敛补丁（让 GUARD-04 + GUARD-05 + GUARD-06a/b/c 转 GREEN）

Command: `/gsd-plan-phase 112`