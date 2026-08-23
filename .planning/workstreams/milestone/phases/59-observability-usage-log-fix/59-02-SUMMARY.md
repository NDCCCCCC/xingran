---
phase: 59-observability-usage-log-fix
plan: 02
subsystem: api-key-auth-testing
tags: [regression-tests, OBSERV-01, OBSERV-02, OBSERV-03, db-row-evidence, require.Eventually]
dependency_graph:
  requires: [phase-59-plan-01]
  provides: [SC#1, SC#2, SC#3, SC#4 regression coverage]
  affects: [internal/middleware/apikey_integration_test.go, internal/services/usage_logger_test.go, internal/services/system/apikey_service_test.go]
tech-stack:
  added: []
  patterns: [real-sqlite-DB-row-evidence, require.Eventually-async-poll, append-only-test-extension]
key-files:
  created: []
  modified:
    - internal/middleware/apikey_integration_test.go
    - internal/services/usage_logger_test.go
    - internal/services/system/apikey_service_test.go
decisions:
  - "D-03: SC#1/#2/#4 测试用 sqlite 真实文件 DB（既有 setupUsageLoggerTestDB helper，per-test 独立文件 os.TempDir+UnixNano+pid + busy_timeout=5000 + 裸 CREATE TABLE），DB 行实证而非 fake（fake 不写库无法满足 SC#1/#4）"
  - "D-03a: Phase 57 既有 fake 测试（fakeUsageLogger/fakeAPIKeyService/setupUsageLoggerTestDB/hex64/TestMultiAuthIntegration/TestConstructorsCallable_D02 全部；既有 time.Sleep 子测试；既有 TestGetUsageLogSummary + 「成功率计算」子测试）原样保留，新测试与之并存不回归"
  - "waitForUsageLog helper 在 middleware 包与 services 包各定义一份同名同形副本（Go 测试包隔离，无法跨包导入）— 形态统一遵循 RESEARCH.md §异步写入可测试性机制"
  - "SC#2 必须用下游 RequireScope→403 失败路径（发生在 c.Next() 下游），不用 pre-auth 401（pitfall 3: pre-auth 在 c.Next() 前 abort，无日志行，测试会因找不到行误判失败）"
metrics:
  duration: "~5 minutes (3 commits)"
  completed_date: 2026-08-13
  tasks_completed: 3
  tasks_total: 3
  files_changed: 3
---

# Phase 59 Plan 02: 使用日志修复回归测试 Summary

**One-liner:** 为 Plan 01 修复后的源码契约提供 DB 行实证级别的回归测试覆盖 — 4 个新子测试 + 1 个 `waitForUsageLog` 异步等待 helper，覆盖 OBSERV-01/02/03 全部 success criteria。fake 测试（Phase 57）原样保留，正交并存。

## Tasks Executed

### Task 1: `internal/middleware/apikey_integration_test.go` — SC#1（2xx 时序/字段）+ SC#2（下游失败）真实 DB 集成测试

**Status:** Completed
**Commit:** `c35e675`

**Change:** Append-only addition of two integration tests + `waitForUsageLog` helper:

1. **`waitForUsageLog` helper** (mirror of RESEARCH.md §async-write testability):
   ```go
   func waitForUsageLog(t *testing.T, db *gorm.DB, apiKeyID string, want int64) {
       t.Helper()
       require.Eventually(t, func() bool {
           var count int64
           db.Model(&models.APIKeyUsageLog{}).Where("api_key_id = ?", apiKeyID).Count(&count)
           return count >= want
       }, 2*time.Second, 10*time.Millisecond,
           "usage log for key=%s not persisted within 2s", apiKeyID)
   }
   ```
   Replaces `time.Sleep` flaky anti-pattern with deterministic async-poll.

2. **`TestMultiAuthUsageLogTiming` (SC#1)**: 2xx request path → asserts `StatusCode=200` / `Duration>0` / `Success=true` via **real** `services.NewUsageLogger(db)` + real gin.Engine + real sqlite DB row assertion (not fake). Added 1ms sleep in handler to ensure `Duration>0` (millisecond granularity).

3. **`TestMultiAuthUsageLogFailure` (SC#2)**: downstream `RequireScope("write")` → 403 path → asserts `StatusCode=403` / `Success=false` (D-01). Critical: uses downstream 403 NOT pre-auth 401 (pitfall 3: pre-auth aborts before `c.Next()`, so MultiAuth never reaches the post-`c.Next()` capture point, and no row is created).

**D-03a preserved:** `fakeUsageLogger` / `fakeAPIKeyService` / `setupUsageLoggerTestDB` / `hex64` / `TestMultiAuthIntegration` / `TestConstructorsCallable_D02` all untouched.

**Pitfall avoidance:**
- No `db.AutoMigrate(&models.APIKeyUsageLog{})` — uses existing `setupUsageLoggerTestDB` bare `CREATE TABLE` (sidesteps `gen_random_uuid()` PG-only trap).
- No `file::memory:?cache=shared` — per-test file DB (avoids lock contention with fire-and-forget goroutine).

---

### Task 2: `internal/services/usage_logger_test.go` — SC#4（cancel-race）cancel-race 单元测试

**Status:** Completed
**Commit:** `7643578`

**Change:** Append-only addition of `waitForUsageLog` helper (same shape, separate copy due to Go test package isolation) + `TestLogUsageCancelledCtxStillWrites_D02`:

1. **`waitForUsageLog` helper** — identical to middleware package copy; Go test package isolation requires per-package definition.

2. **`TestLogUsageCancelledCtxStillWrites_D02` (SC#4)**: Pre-cancels the caller ctx (simulates P2-b scenario where request ends and `ctx.Canceled` propagates), calls `LogUsage`, then waits via `waitForUsageLog`. Without the D-02 fix, `logUsageAsync` would propagate the cancelled ctx into `db.WithContext(ctx).Create()`, fail, and silently swallow the error (`_ = err`) → row never appears → `require.Eventually` timeout-fails. With D-02 (detached `context.WithTimeout(context.Background(), 10s)`), the row appears within 2s.

**D-03a preserved:** `setupUsageLoggerTestDB` / `TestNewUsageLogger` / `TestLogUsage` (with all subtests including `time.Sleep` ones) / `TestAsyncLogging` / `TestLogUsageErrorHandling` / `TestLogUsagePerformance` all untouched.

**Pitfall avoidance:** Same as Task 1 — no AutoMigrate, no shared in-memory DB.

---

### Task 3: `internal/services/system/apikey_service_test.go` — SC#3（混合 success 行 → successRate ∈ (0,100)）聚合防回归测试

**Status:** Completed
**Commit:** `10e5bb1`

**Change:** Append-only addition of `TestGetUsageLogSummaryMixed`:

Seeds 2 `Success=true` rows (StatusCode 200, 204) + 2 `Success=false` rows (StatusCode 403, 429 — downstream, not pre-auth 401), calls `GetUsageLogSummary`, asserts:
- `SuccessRate > 0` (regression anchor: pre-fix Success was always false → rate was always ≈0%)
- `SuccessRate < 100` (not all-success)
- `InDelta(50.0, SuccessRate, 0.1)` (exact anchor: 2/4 = 50%)

Uses existing `setupTestDB` / `createTestUser` / `createTestAPIKey` / `cleanupTestData` helpers — no new infrastructure.

**D-03a preserved:** `TestGetUsageLogSummary` (line 1015) + `「成功率计算」` subtest (line 1096, 70% case) untouched — still passes as second-layer regression anchor.

**Pitfall avoidance:** Uses `setupTestDB` (which uses `file::memory:?cache=shared&_enable_boolean=true` for boolean serialization correctness). This is correct for the aggregation tests — no fire-and-forget goroutines in `GetUsageLogSummary` so no lock contention with shared in-memory DB.

---

## Verification Results

All 4 SC verification commands from VALIDATION.md exit 0:

```bash
$ go test ./internal/middleware/ -run "TestMultiAuthUsageLogTiming|TestMultiAuthUsageLogFailure" -v
=== RUN   TestMultiAuthUsageLogTiming
--- PASS: TestMultiAuthUsageLogTiming (0.04s)
=== RUN   TestMultiAuthUsageLogFailure
--- PASS: TestMultiAuthUsageLogFailure (0.02s)
PASS
ok      github.com/xingran-next/xingran-go-backend/internal/middleware   0.251s

$ go test ./internal/services/ -run TestLogUsageCancelledCtxStillWrites_D02 -v
=== RUN   TestLogUsageCancelledCtxStillWrites_D02
--- PASS: TestLogUsageCancelledCtxStillWrites_D02 (0.04s)
PASS
ok      github.com/xingran-next/xingran-go-backend/internal/services      1.578s

$ go test ./internal/services/system/ -run TestGetUsageLogSummaryMixed -v
=== RUN   TestGetUsageLogSummaryMixed
--- PASS: TestGetUsageLogSummaryMixed (0.00s)
PASS
ok      github.com/xingran-next/xingran-go-backend/internal/services/system      1.750s
```

All D-03a regression anchors still pass:

```bash
$ go test ./internal/middleware/ -run TestMultiAuthIntegration -v  # Phase 57 fake tests
--- PASS: TestMultiAuthIntegration (0.00s)
    --- PASS: 有效key+正确scope_通过并写入context (0.00s)
    --- PASS: 有效key+缺失scope_403 (0.00s)
    --- PASS: 无效key_401 (0.00s)

$ go test ./internal/services/system/ -run "TestGetUsageLogSummary$" -v  # existing 70% case
--- PASS: TestGetUsageLogSummary (0.04s)
    --- PASS: TestGetUsageLogSummary/成功率计算 (0.00s)
    --- PASS: TestGetUsageLogSummary/平均耗时 (0.00s)
    --- PASS: TestGetUsageLogSummary/按方法分组 (0.00s)
    --- PASS: TestGetUsageLogSummary/按路径分组 (0.00s)
    --- PASS: TestGetUsageLogSummary/错误统计 (0.00s)
    --- PASS: TestGetUsageLogSummary/统计数据正确性 (0.00s)
    --- PASS: TestGetUsageLogSummary/总请求数 (0.00s)
```

Pitfall grep checks (literal pattern search):

```bash
$ grep -n "AutoMigrate.*APIKeyUsageLog" internal/middleware/apikey_integration_test.go internal/services/usage_logger_test.go internal/services/system/apikey_service_test.go
# 0 hits (pitfall 1 avoided)

$ grep -n "file::memory:?cache=shared" internal/middleware/apikey_integration_test.go internal/services/usage_logger_test.go internal/services/system/apikey_service_test.go
# 1 hit in apikey_service_test.go (line 38, setupTestDB pre-existing — correct for aggregation tests, no async contention)
# 0 hits in the 2 test files modified for SC#1/SC#2/SC#4 (pitfall 2 avoided for new tests)
```

## Success Criteria Status

- [x] All 3 tasks executed; each test file committed atomically (pathspec `git commit <file-path>`)
- [x] Test functions exist with EXACT names: `TestMultiAuthUsageLogTiming`, `TestMultiAuthUsageLogFailure`, `TestLogUsageCancelledCtxStillWrites_D02`, `TestGetUsageLogSummaryMixed`
- [x] `waitForUsageLog` helper added in BOTH `apikey_integration_test.go` and `usage_logger_test.go` (Go test package isolation)
- [x] All 4 SC verification commands exit 0
- [x] D-03a regression checks (`TestMultiAuthIntegration` + `TestGetUsageLogSummary$`) still pass
- [x] Pitfall grep checks: 0 hits for `AutoMigrate.*APIKeyUsageLog` in new tests; 0 hits for `file::memory:?cache=shared` in the 2 files that use fire-and-forget goroutines
- [x] SUMMARY.md created at `.planning/phases/59-observability-usage-log-fix/59-02-SUMMARY.md` (this file, committed below)
- [x] No modifications to STATE.md / ROADMAP.md / unrelated test files (orchestrator owns tracking; pre-existing failures are out of scope)

## Key Findings

1. **Duration granularity gotcha**: `time.Since(start).Milliseconds()` rounds down to 0 for sub-millisecond requests. TestMultiAuthUsageLogTiming adds a 1ms sleep in the handler to ensure `Duration>0` reliably — without it, the test fails on fast systems. Documented inline.

2. **Pitfall 3 is structural, not semantic**: Pre-auth failures (401) abort **before** `c.Next()`, so MultiAuth returns without ever reaching the post-`c.Next()` capture block. SC#2 must use downstream failures (RequireScope→403, RateLimitByScope→429, handler→5xx) — these execute in `c.Next()` and let the capture point run.

3. **Test package isolation matters**: `waitForUsageLog` had to be duplicated across `middleware` and `services` packages even though the shape is identical. The Go test package model prevents sharing helpers across packages in the same module unless both import a third package — which would add complexity disproportionate to the helper's simplicity.

4. **setupTestDB vs setupUsageLoggerTestDB**: Two different helpers serve different needs. The former uses shared in-memory SQLite (safe for synchronous aggregation tests with no async contention), the latter uses per-test file DB (needed for fire-and-forget goroutine tests to avoid lock contention). Both are correct for their respective domains.

## Commits

| Commit | File | Change |
|--------|------|--------|
| `c35e675` | `internal/middleware/apikey_integration_test.go` | +97 lines: SC#1 + SC#2 + waitForUsageLog helper |
| `7643578` | `internal/services/usage_logger_test.go` | +51 lines: SC#4 + waitForUsageLog helper (separate copy) |
| `10e5bb1` | `internal/services/system/apikey_service_test.go` | +74 lines: SC#3 (TestGetUsageLogSummaryMixed) |