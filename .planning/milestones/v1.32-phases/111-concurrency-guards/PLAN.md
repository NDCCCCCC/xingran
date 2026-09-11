---
phase: "111"
plan: "01"
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/services/oper_log_service.go
autonomous: true
requirements_addressed: [GOR-01]
must_haves:
  truths:
    - "OperLog async goroutines recover from panics and log SECURITY-level errors instead of crashing the process"
  artifacts:
    - path: "internal/services/oper_log_service.go:67"
      provides: "RecordAsync goroutine with defer recover"
    - path: "internal/services/oper_log_service.go:140"
      provides: "RecordFromGinContext async goroutine with defer recover"
  key_links:
    - from: "internal/services/oper_log_service.go:67,140"
      to: "applogger"
      via: "defer recover + SECURITY warn"
      pattern: "defer func.*recover"
---

<objective>
Fix GOR-01: Add `defer recover()` + SECURITY-level error logging to the two bare goroutines in `oper_log_service.go` at lines 67 and 140. Both async DB-write goroutines currently have no panic protection -- a DB write panic (e.g. nil pointer in GORM) would propagate and crash the process. This turns GUARD-07 GREEN.
</objective>

<context>
@internal/services/oper_log_service.go:42-73 (RecordAsync)
@internal/services/oper_log_service.go:139-146 (RecordFromGinContext)
@internal/api/v1/system/user_handler.go:200-236 (AD sync goroutine -- authoritative defer recover pattern)
@internal/services/oper_log_service_test.go (GUARD-07 test)

Reference pattern from user_handler.go (authoritative, Phase 111 GOR-03 reference):
```go
go func() {
    defer func() {
        h.adSyncInFlight.Delete(id) // 释放去重标记（无论成功/失败/panic）
        if r := recover(); r != nil {
            applogger.Errorf("[AD-SYNC] SyncUserUpdateToAD panic 已恢复 [ID=%s]: panic=%v", id, r)
        }
    }()
    ctx := context.Background()
    // ... work ...
}()
```

Current RED state (oper_log_service.go):
```go
// Line 67-72: NO recover
go func() {
    if err := db.Create(operLog).Error; err != nil {
        _ = err  // silent swallow
    }
}()

// Line 140-145: NO recover
go func() {
    if err := s.RecordOperLog(context.Background(), db, operLog); err != nil {
        _ = err  // silent swallow
    }
}()
```
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add defer recover to RecordAsync goroutine (line 67)</name>
  <files>internal/services/oper_log_service.go</files>
  <action>
In `internal/services/oper_log_service.go`, modify the `RecordAsync` goroutine (line 67-72) to wrap with defer recover.

Replace:
```go
go func() {
    if err := db.Create(operLog).Error; err != nil {
        // 静默处理日志记录失败
        _ = err
    }
}()
```

With:
```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            applogger.Errorf("[OPER-LOG] RecordAsync panic 已恢复: panic=%v", r)
        }
    }()
    if err := db.Create(operLog).Error; err != nil {
        applogger.Errorf("[OPER-LOG] 异步记录操作日志失败: %v", err)
    }
}()
```

Key changes:
- Wrap entire goroutine body with `defer func() { if r := recover(); r != nil { ... } }()`
- On panic: log at Error level with "已恢复" (recovered) marker
- On regular error (non-panic DB error): log at Error level (upgrade from silent swallow)
- Log prefix `[OPER-LOG]` for log filtering consistency
</action>
  <verify>
go build ./internal/services/...
go test ./internal/services/... -run TestOperLog_Async_DoesNotPanicOnDBError -v
</verify>
  <done>RecordAsync goroutine at line 67 has defer recover; GUARD-07 source-pattern check passes</done>
</task>

<task type="auto">
  <name>Task 2: Add defer recover to RecordFromGinContext async goroutine (line 140)</name>
  <files>internal/services/oper_log_service.go</files>
  <action>
In `internal/services/oper_log_service.go`, modify the `RecordFromGinContext` async goroutine (line 140-145) to wrap with defer recover.

Replace:
```go
go func() {
    if err := s.RecordOperLog(context.Background(), db, operLog); err != nil {
        // 静默处理日志记录失败
        _ = err
    }
}()
```

With:
```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            applogger.Errorf("[OPER-LOG] RecordFromGinContext panic 已恢复: panic=%v", r)
        }
    }()
    if err := s.RecordOperLog(context.Background(), db, operLog); err != nil {
        applogger.Errorf("[OPER-LOG] 异步记录操作日志失败: %v", err)
    }
}()
```

Key changes:
- Mirror the same defer/recover pattern as Task 1 for consistency
- Log prefix `[OPER-LOG]` matches the RecordAsync pattern
- Non-panic errors are logged (not silently swallowed), matching the GOR-01 requirement
</action>
  <verify>
go test ./internal/services/... -run TestOperLog_Async_DoesNotPanicOnDBError -v
</verify>
  <done>RecordFromGinContext goroutine at line 140 has defer recover; GUARD-07 fully GREEN</done>
</task>

</tasks>

<success_criteria>
- GUARD-07 (`TestOperLog_Async_DoesNotPanicOnDBError`) passes: `strings.Contains(srcStr, "defer func() { if r := recover()")` is true
- Both goroutines log at Error level instead of silent swallow
- No regressions in `go test ./internal/services/...`
</success_criteria>

<output>
Commit: feat(111-01): add defer recover to oper_log async goroutines (GOR-01)
Files: internal/services/oper_log_service.go
Summary: .planning/phases/111-concurrency-guards/111-01-SUMMARY.md
---

---
phase: "111"
plan: "02"
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/api/v1/system/ad_dept_sync_handler.go
autonomous: true
requirements_addressed: [GOR-02]
must_haves:
  truths:
    - "AD dept sync goroutine uses detached context (context.Background + timeout) not c.Request.Context()"
    - "AD dept sync goroutine recovers from panics with Error-level log"
  artifacts:
    - path: "internal/api/v1/system/ad_dept_sync_handler.go:99"
      provides: "SyncDeptStructureToAD goroutine with detached ctx + defer recover"
  key_links:
    - from: "internal/api/v1/system/ad_dept_sync_handler.go:99"
      to: "context.Background"
      via: "context.WithTimeout(context.Background(), ADSyncTimeout)"
      pattern: "context.WithTimeout"
---

<objective>
Fix GOR-02: Replace `c.Request.Context()` (which cancels when HTTP response completes) with `context.WithTimeout(context.Background(), ADSyncTimeout)` in `ad_dept_sync_handler.go:99`. Also add `defer recover()` to prevent panic propagation. This ensures the async sync task runs to completion regardless of HTTP response timing.
</objective>

<context>
@internal/api/v1/system/ad_dept_sync_handler.go:85-113
@internal/api/v1/system/user_handler.go:200-236 (AD sync goroutine -- authoritative pattern)
@pkg/constants/timeouts.go (ADSyncTimeout constant)

Current RED state (ad_dept_sync_handler.go:99-104):
```go
go func() {
    _, err := h.syncService.SyncDeptStructureToAD(c.Request.Context(), req.ADConfigID)
    //                                              ^^^^^^^^^^^^^^^^^^^^^^
    // PROBLEM: c.Request.Context() cancels when HTTP response is sent.
    // If the sync is still running, it will be cancelled mid-flight.
    if err != nil {
        applogger.Errorf("手动触发部门同步失败: %v", err)
    }
}()
```

ADSyncTimeout is defined in `pkg/constants/timeouts.go` as `30 * time.Minute`.

Reference pattern (user_handler.go:200-236):
```go
go func() {
    defer func() {
        h.adSyncInFlight.Delete(id)
        if r := recover(); r != nil {
            applogger.Errorf("[AD-SYNC] SyncUserUpdateToAD panic 已恢复 [ID=%s]: panic=%v", id, r)
        }
    }()
    ctx := context.Background()
    // ... retry loop ...
}()
```
</context>

<tasks>

<task type="auto">
  <name>Task 1: Fix SyncDeptStructureToAD goroutine ctx + defer recover</name>
  <files>internal/api/v1/system/ad_dept_sync_handler.go</files>
  <action>
In `internal/api/v1/system/ad_dept_sync_handler.go`, modify the `TriggerDeptSync` handler goroutine (line 99-104).

The current code uses `c.Request.Context()` which cancels when the HTTP response is sent. Replace with `context.WithTimeout(context.Background(), ADSyncTimeout)` and add defer recover.

Replace:
```go
go func() {
    _, err := h.syncService.SyncDeptStructureToAD(c.Request.Context(), req.ADConfigID)
    if err != nil {
        applogger.Errorf("手动触发部门同步失败: %v", err)
    }
}()
```

With:
```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            applogger.Errorf("[AD-DEPT-SYNC] TriggerDeptSync panic 已恢复: panic=%v", r)
        }
    }()
    ctx, cancel := context.WithTimeout(context.Background(), constants.ADSyncTimeout)
    defer cancel()
    _, err := h.syncService.SyncDeptStructureToAD(ctx, req.ADConfigID)
    if err != nil {
        applogger.Errorf("[AD-DEPT-SYNC] 手动触发部门同步失败: %v", err)
    }
}()
```

Key changes:
- `context.WithTimeout(context.Background(), constants.ADSyncTimeout)` replaces `c.Request.Context()`
- `defer cancel()` ensures timeout context is cleaned up
- `defer recover()` catches panics and logs at Error level with "已恢复" marker
- Log prefix `[AD-DEPT-SYNC]` for log filtering
- Timeout constant from `pkg/constants/timeouts.go` (30 minutes)
</action>
  <verify>
go build ./internal/api/v1/system/...
go test ./internal/api/v1/system/... -run "AD" -v
</verify>
  <done>SyncDeptStructureToAD goroutine uses detached context + timeout; no c.Request.Context() leak; defer recover present</done>
</task>

</tasks>

<success_criteria>
- Goroutine uses `context.Background()` not `c.Request.Context()`
- `context.WithTimeout` applied with `constants.ADSyncTimeout`
- `defer recover()` present and logs at Error level
- No regressions in build or existing tests
</success_criteria>

<output>
Commit: feat(111-02): add detached ctx + defer recover to AD dept sync goroutine (GOR-02)
Files: internal/api/v1/system/ad_dept_sync_handler.go
Summary: .planning/phases/111-concurrency-guards/111-02-SUMMARY.md
---

---
phase: "111"
plan: "03"
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/agent/server/connection_manager.go
autonomous: true
requirements_addressed: [GOR-03]
must_haves:
  truths:
    - "Agent connection manager goroutines recover from panics instead of crashing the process"
  artifacts:
    - path: "internal/agent/server/connection_manager.go:186"
      provides: "handleReconnect goroutine with defer recover"
    - path: "internal/agent/server/connection_manager.go:196"
      provides: "cleanupConnection goroutine with defer recover"
  key_links:
    - from: "internal/agent/server/connection_manager.go:186,196"
      to: "applogger"
      via: "defer recover"
      pattern: "defer func.*recover"
---

<objective>
Fix GOR-03: Add `defer recover()` to the two bare reconnect goroutines in `connection_manager.go` at lines 186 and 196. Both `handleReconnect` and `cleanupConnection` launch `go func() { ... cm.Reconnect(...) }()` without panic protection. If Reconnect panics, it would crash the process.
</objective>

<context>
@internal/agent/server/connection_manager.go:175-210
@internal/api/v1/system/user_handler.go:200-236 (defer recover pattern)

Current RED state (connection_manager.go:186-193 and 196-203):
```go
// Line 186-193
go func() {
    if err := cm.Reconnect(context.Background()); err != nil {
        WithFields(logrus.Fields{
            "error":           err.Error(),
            "reconnect_count": cm.reconnectCount,
        }).Warn("Reconnect failed")
    }
}()

// Line 196-203
go func() {
    if err := cm.Reconnect(context.Background()); err != nil {
        WithFields(logrus.Fields{
            "error":           err.Error(),
            "reconnect_count": cm.reconnectCount,
        }).Warn("Reconnect failed")
    }
}()
```

Note: These goroutines already use `context.Background()` (detached context), satisfying the context requirement. The fix is purely adding `defer recover()`.
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add defer recover to both reconnect goroutines (lines 186 and 196)</name>
  <files>internal/agent/server/connection_manager.go</files>
  <action>
In `internal/agent/server/connection_manager.go`, modify the two reconnect goroutines at lines 186 and 196.

**First goroutine (line 186-193)**:

Replace:
```go
go func() {
    if err := cm.Reconnect(context.Background()); err != nil {
        WithFields(logrus.Fields{
            "error":           err.Error(),
            "reconnect_count": cm.reconnectCount,
        }).Warn("Reconnect failed")
    }
}()
```

With:
```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            WithFields(logrus.Fields{
                "panic": r,
            }).Error("handleReconnect goroutine panic 已恢复")
        }
    }()
    if err := cm.Reconnect(context.Background()); err != nil {
        WithFields(logrus.Fields{
            "error":           err.Error(),
            "reconnect_count": cm.reconnectCount,
        }).Warn("Reconnect failed")
    }
}()
```

**Second goroutine (line 196-203)**:

Replace:
```go
go func() {
    if err := cm.Reconnect(context.Background()); err != nil {
        WithFields(logrus.Fields{
            "error":           err.Error(),
            "reconnect_count": cm.reconnectCount,
        }).Warn("Reconnect failed")
    }
}()
```

With:
```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            WithFields(logrus.Fields{
                "panic": r,
            }).Error("cleanupConnection goroutine panic 已恢复")
        }
    }()
    if err := cm.Reconnect(context.Background()); err != nil {
        WithFields(logrus.Fields{
            "error":           err.Error(),
            "reconnect_count": cm.reconnectCount,
        }).Warn("Reconnect failed")
    }
}()
```

Key changes:
- Each goroutine wrapped with `defer func() { if r := recover(); r != nil { ... } }()`
- On panic: log at Error level with "goroutine panic 已恢复" + panic value
- Uses `WithFields` (logrus) for consistent structured logging in the agent package
- Keep existing Warn log for non-panic reconnect failures
</action>
  <verify>
go build ./internal/agent/...
go test ./internal/agent/... -v
</verify>
  <done>Both reconnect goroutines (lines 186 and 196) have defer recover; GOR-03 complete</done>
</task>

</tasks>

<success_criteria>
- Both goroutines at lines 186 and 196 have `defer func() { if r := recover()` pattern
- Panic recovery logs at Error level with "panic 已恢复" message
- No regressions in build or tests
</success_criteria>

<output>
Commit: feat(111-03): add defer recover to agent connection manager goroutines (GOR-03)
Files: internal/agent/server/connection_manager.go
Summary: .planning/phases/111-concurrency-guards/111-03-SUMMARY.md
---

---
phase: "111"
plan: "04"
type: execute
wave: 2
depends_on: [111-01]
files_modified:
  - internal/api/v1/auth.go
  - internal/core/captcha.go
  - internal/core/captcha_test.go
autonomous: true
requirements_addressed: [GOR-04, CAP-01]
must_haves:
  truths:
    - "Login log async goroutine recovers from panics instead of crashing the process"
    - "Captcha Increment failures log a SECURITY-level warn (fail-closed: verification still rejects)"
  artifacts:
    - path: "internal/api/v1/auth.go:598"
      provides: "Login log async goroutine with defer recover"
    - path: "internal/core/captcha.go:379,439,445"
      provides: "SECURITY warn on cache Increment failure"
    - path: "internal/core/captcha_test.go"
      provides: "GUARD-08 extended with SECURITY warn assertion (TODO-111)"
  key_links:
    - from: "internal/api/v1/auth.go:598"
      to: "applogger"
      via: "defer recover"
      pattern: "defer func.*recover"
    - from: "internal/core/captcha.go:379,439,445"
      to: "applogger"
      via: "SECURITY warn on Increment failure"
      pattern: "SECURITY.*Increment"
---

<objective>
Fix GOR-04: Add `defer recover()` to the login log async goroutine in `auth.go:598`. Also fix CAP-01: Add SECURITY-level warn logging when `s.cache.Increment` fails at three locations in `captcha.go:379,439,445`. Finally extend GUARD-08 test to assert the SECURITY warn is emitted (TODO-111 from Phase 109).
</objective>

<context>
@internal/api/v1/auth.go:595-603
@internal/core/captcha.go:370-460 (Increment calls at 379, 439, 445)
@internal/core/captcha_test.go:85-166 (GUARD-08 test + TODO-111 marker)
@internal/api/v1/system/user_handler.go:200-236 (defer recover pattern)

**GOR-04 current state (auth.go:598-602)**:
```go
go func() {
    if err := core.DB.GetDB().Create(&loginLog).Error; err != nil {
        applogger.Errorf("记录登录日志失败 (user: %s, ip: %s, status: %d): %v", username, clientIP, status, err)
    }
}()
// NOTE: No defer recover. A DB panic would crash the process.
```

**CAP-01 current state (captcha.go)**:
Line 379: `_, _ = s.cache.Increment(ctx, attemptsKey)` -- silently ignores error
Line 439: `_, _ = s.cache.Increment(ctx, attemptsKey)` -- silently ignores error
Line 445: `_, _ = s.cache.Increment(ctx, attemptsKey)` -- silently ignores error

From captcha_test.go TODO-111:
```
// TODO-111: uncomment the above assertion and implement log capture once
// Phase 111 CAP-01 adds the SECURITY warn to captcha.go:379,439,445.
```
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add defer recover to login log async goroutine (auth.go:598)</name>
  <files>internal/api/v1/auth.go</files>
  <action>
In `internal/api/v1/auth.go`, modify the login log async goroutine (line 598-602) to add defer recover.

Replace:
```go
go func() {
    if err := core.DB.GetDB().Create(&loginLog).Error; err != nil {
        applogger.Errorf("记录登录日志失败 (user: %s, ip: %s, status: %d): %v", username, clientIP, status, err)
    }
}()
```

With:
```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            applogger.Errorf("[AUTH-LOGIN] 记录登录日志 panic 已恢复: panic=%v", r)
        }
    }()
    if err := core.DB.GetDB().Create(&loginLog).Error; err != nil {
        applogger.Errorf("记录登录日志失败 (user: %s, ip: %s, status: %d): %v", username, clientIP, status, err)
    }
}()
```

Key changes:
- `defer func() { if r := recover(); r != nil { ... } }()` wraps the entire goroutine body
- On panic: log at Error level with "panic 已恢复" marker and `[AUTH-LOGIN]` prefix
- Non-panic errors remain logged at Error level (existing behavior preserved)
</action>
  <verify>
go build ./internal/api/v1/...
go test ./internal/api/v1/... -run "Login" -v
</verify>
  <done>Login log goroutine has defer recover; GOR-04 complete</done>
</task>

<task type="auto">
  <name>Task 2: Add SECURITY warn logs on Increment failure (captcha.go:379,439,445)</name>
  <files>internal/core/captcha.go</files>
  <action>
In `internal/core/captcha.go`, add SECURITY-level warn logging when `s.cache.Increment` fails at three locations. The function is `VerifyNormal` (called by `VerifyCaptcha`).

**Location 1 — captcha.go:379 (wrong code in VerifyNormal)**:

Replace:
```go
if storedCode != input {
    // 增加失败次数
    _, _ = s.cache.Increment(ctx, attemptsKey)
    return fmt.Errorf("验证码错误")
}
```

With:
```go
if storedCode != input {
    // 增加失败次数
    if _, err := s.cache.Increment(ctx, attemptsKey); err != nil {
        applogger.Warnf("[SECURITY] Captcha increment failed (attemptsKey=%s): %v", attemptsKey, err)
    }
    return fmt.Errorf("验证码错误")
}
```

**Location 2 — captcha.go:439 (slider xPos mismatch in VerifySliderCaptcha)**:

Replace:
```go
if abs(xPos-expectedX) > tolerance {
    _, _ = s.cache.Increment(ctx, attemptsKey)
    return fmt.Errorf("验证失败，位置不正确")
}
```

With:
```go
if abs(xPos-expectedX) > tolerance {
    if _, err := s.cache.Increment(ctx, attemptsKey); err != nil {
        applogger.Warnf("[SECURITY] Captcha increment failed (attemptsKey=%s): %v", attemptsKey, err)
    }
    return fmt.Errorf("验证失败，位置不正确")
}
```

**Location 3 — captcha.go:445 (slider token mismatch)**:

Replace:
```go
if token == "" || token != verifyData.Token {
    _, _ = s.cache.Increment(ctx, attemptsKey)
    return fmt.Errorf("验证失败，token无效")
}
```

With:
```go
if token == "" || token != verifyData.Token {
    if _, err := s.cache.Increment(ctx, attemptsKey); err != nil {
        applogger.Warnf("[SECURITY] Captcha increment failed (attemptsKey=%s): %v", attemptsKey, err)
    }
    return fmt.Errorf("验证失败，token无效")
}
```

Key changes:
- Replace `_, _ = s.cache.Increment(...)` with `if _, err := s.cache.Increment(...); err != nil { applogger.Warnf(...) }`
- Log level: `Warnf` with `[SECURITY]` prefix (distinct from Error, signals security-relevant event)
- Include `attemptsKey` in log for traceability
- Verification remains fail-closed (still returns error) — log is the only change
</action>
  <verify>
go build ./internal/core/...
go test ./internal/core/... -run TestCaptcha_Increment_Failure_FailsClosed -v
</verify>
  <done>All three Increment calls in captcha.go log SECURITY warn on failure; fail-closed behavior unchanged</done>
</task>

<task type="auto">
  <name>Task 3: Extend GUARD-08 test to assert SECURITY warn log (TODO-111)</name>
  <files>internal/core/captcha_test.go</files>
  <action>
In `internal/core/captcha_test.go`, extend `TestCaptcha_Increment_Failure_FailsClosed` to capture and assert that a SECURITY warn was logged when Increment fails.

The test currently uses a mockCache where `incrementFn` returns an error. After CAP-01 fix, the service calls `applogger.Warnf("[SECURITY] Captcha increment failed...")`. We need to capture this log output and assert it.

**Approach**: Use the standard library's `log/slog` or set a custom `applogger` (via the `applogger` package's test hook if available) to capture log output. Since `applogger` is a `logrus`-based logger, use `logrus` test hook pattern.

Check if `applogger` can be set as a custom `*logrus.Logger` in the test. The `applogger` package exposes `SetLogger` or uses a package-level `log *logrus.Logger`. If the package uses a fixed logger, we can use `logrus.NewGlobal` + `logrus.NewEntry` to capture output.

**Simpler approach** (test hook): Use `logrus.AddHook` to capture log entries during the test. Create a `testHook` that records all log entries with `[SECURITY]` prefix. After the test, assert the hook recorded a `Warn` level entry containing `[SECURITY]` and `Captcha increment failed`.

Add to the test file (before calling `VerifyCaptcha`):
```go
// Capture [SECURITY] warn logs
var securityWarnEmitted bool
var securityWarnMsg string
testHook := &logrusTestHook{entries: []struct {
    level  logrus.Level
    msg    string
}{}}

// Set up a buffer to capture log output
// Alternative: use applogger.SetOutput (if exposed) or logrus.SetOutput

// Simpler: use a custom hook
type captchaTestHook struct {
    mu    sync.Mutex
    warns []string
}

func (h *captchaTestHook) Levels() []logrus.Level {
    return []logrus.Level{logrus.WarnLevel}
}

func (h *captchaTestHook) Fire(e *logrus.Entry) error {
    h.mu.Lock()
    defer h.mu.Unlock()
    if strings.Contains(e.Message, "[SECURITY]") && strings.Contains(e.Message, "Captcha increment failed") {
        h.warns = append(h.warns, e.Message)
    }
    return nil
}
```

Then after calling `svc.VerifyCaptcha(...)`:
```go
require.True(t, len(testHook.warns) > 0,
    "A [SECURITY] warn must be logged when Increment fails (CAP-01)")
assert.Contains(t, testHook.warns[0], "Captcha increment failed",
    "SECURITY warn must mention 'Captcha increment failed'")
```

**Important**: Remove the TODO-111 comment block after extending the test. The test should fully assert the CAP-01 requirement.
</action>
  <verify>
go test ./internal/core/... -run TestCaptcha_Increment_Failure_FailsClosed -v
</verify>
  <done>GUARD-08 extended with SECURITY warn assertion; TODO-111 marker removed; test PASS</done>
</task>

</tasks>

<success_criteria>
- GOR-04: `auth.go:598` goroutine has `defer func() { if r := recover()` pattern
- CAP-01: All three Increment calls log `[SECURITY]` Warn when they fail
- GUARD-08: Test passes and asserts SECURITY warn was emitted
- No regressions in `go test ./internal/core/...` or `go test ./internal/api/v1/...`
</success_criteria>

<output>
Commit: feat(111-04): add defer recover to login log async + captcha SECURITY warn (GOR-04, CAP-01)
Files: internal/api/v1/auth.go, internal/core/captcha.go, internal/core/captcha_test.go
Summary: .planning/phases/111-concurrency-guards/111-04-SUMMARY.md
</success_criteria>

<verification>
# Run all Phase 111 GUARD tests
go test ./internal/services/... -run TestOperLog_Async_DoesNotPanicOnDBError -v
go test ./internal/core/... -run TestCaptcha_Increment_Failure_FailsClosed -v

# All should pass
# Phase 109 GUARD-07 RED baseline confirmed (no defer recover in oper_log_service.go)
# After Phase 111: GUARD-07 GREEN, GUARD-08 GREEN (extended)

# Seven gate verification
go build ./...
go test ./...
</verification>

<success_criteria>
## GUARD Tests: Phase 111 Outcome

| GUARD | Test | Phase 109 State | Phase 111 After Fix |
|-------|------|-----------------|---------------------|
| GUARD-07 | `TestOperLog_Async_DoesNotPanicOnDBError` | **FAIL** (no defer recover) | **PASS** (GOR-01 applied) |
| GUARD-08 | `TestCaptcha_Increment_Failure_FailsClosed` | **PASS** (already fail-closed) | **PASS** (CAP-01 adds SECURITY warn + test extended) |

## All 4 GOR Requirements Addressed

| ID | File | Lines | Fix |
|----|------|-------|-----|
| GOR-01 | `internal/services/oper_log_service.go` | 67, 140 | `defer recover` + Error-level log |
| GOR-02 | `internal/api/v1/system/ad_dept_sync_handler.go` | 99 | `context.Background()` + `context.WithTimeout` + `defer recover` |
| GOR-03 | `internal/agent/server/connection_manager.go` | 186, 196 | `defer recover` on both reconnect goroutines |
| GOR-04 | `internal/api/v1/auth.go` | 598 | `defer recover` on login log async goroutine |

## CAP-01 Addressed

| ID | File | Lines | Fix |
|----|------|-------|-----|
| CAP-01 | `internal/core/captcha.go` | 379, 439, 445 | SECURITY warn log on `Increment` failure |

## Phase 111 Completeness

- 4 plans covering all 5 requirements (GOR-01..04 + CAP-01)
- GUARD-07 and GUARD-08 both turn GREEN
- 4 atomic commits (one per plan)
- All 7 gates verified
</success_criteria>

<output>
Create `.planning/phases/111-concurrency-guards/{padded_phase}-{NN}-SUMMARY.md` after each plan executes
</output>
