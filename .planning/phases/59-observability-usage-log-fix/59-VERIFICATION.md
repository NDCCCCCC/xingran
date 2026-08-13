---
phase: 59-observability-usage-log-fix
verified: 2026-08-13T10:15:00Z
status: passed
score: 8/8 must-haves verified
overrides_applied: 0
source: plans/59-01 + 59-02, re-run by verifier
started: 2026-08-13T10:00:00Z
updated: 2026-08-13T10:15:00Z
---

# Phase 59: 可观测性 / 使用日志修复 — Verification Report

**Phase Goal:** API Key 使用日志真实反映请求结果——记录时机移到请求处理完成之后,`StatusCode` / `Duration` / `Success` 取真实值,`successRate` 聚合可信,异步 goroutine 不被请求生命周期取消竞态污染。

**Verified:** 2026-08-13T10:15:00Z
**Status:** passed
**Re-verification:** No — initial verification (no prior `*-VERIFICATION.md` existed)

---

## Goal Achievement

### Observable Truths (ROADMAP SC#1-#5 + PLAN must_haves merged)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC#1: 2xx 请求后 `sys_api_key_usage_log` 行 `StatusCode∈2xx` / `Duration>0` / `Success=true` (DB 行实证) | VERIFIED | `TestMultiAuthUsageLogTiming` PASS (re-run 2026-08-13) — `apikey_integration_test.go:268-306` 含 `assert.Equal(t, 200, log.StatusCode)` + `assert.Greater(t, log.Duration, 0)` + `assert.True(t, log.Success)`; 真实 `services.NewUsageLogger(db)` + 真实 sqlite DB 行断言 (non-fake) |
| 2 | SC#2: 下游失败 (RequireScope→403) 后 `Success=false` / `StatusCode=真实失败码` | VERIFIED | `TestMultiAuthUsageLogFailure` PASS — `apikey_integration_test.go:310-346` 含 `assert.Equal(t, 403, log.StatusCode)` + `assert.False(t, log.Success)`; pitfall 3 规避: 用下游 RequireScope→403 而非 pre-auth 401 |
| 3 | SC#3: `GetUsageLogSummary.successRate` 在混合 success 行后 ∈ (0,100) 开区间,不再恒 ≈ 0% | VERIFIED | `TestGetUsageLogSummaryMixed` PASS — `apikey_service_test.go:1258-1323` 含 `assert.Greater(t, summary.SuccessRate, 0.0)` + `assert.Less(t, summary.SuccessRate, 100.0)` + `assert.InDelta(t, 50.0, summary.SuccessRate, 0.1)`; seed 2×Success=true (200/204) + 2×Success=false (403/429) |
| 4 | SC#4: 调用方 ctx 取消后日志仍完整写入 — detached context 防 P2-b | VERIFIED | `TestLogUsageCancelledCtxStillWrites_D02` PASS — `usage_logger_test.go:482-511` 预取消 ctx + `waitForUsageLog` 轮询 DB 行存在; 源码 `usage_logger.go:60` `context.WithTimeout(context.Background(), 10*time.Second)` + `usage_logger.go:79` `s.db.WithContext(detachedCtx).Create(...)` |
| 5 | SC#5: `go test ./internal/middleware/... ./internal/services/...` 全绿 | VERIFIED | SC#1/#2/#4 + D-03a anchors (`TestMultiAuthIntegration` 3 子测试 + `TestGetUsageLogSummary` 7 子测试 + `TestConstructorsCallable_D02`) 全部 PASS (re-run 2026-08-13) |
| 6 | D-01 落地: `Success` 派生为 `statusCode >= 200 && statusCode < 300` (2xx-only) | VERIFIED | `apikey.go:84` 字面命中 `Success: statusCode >= 200 && statusCode < 300`; `Success` 字段在 LogUsageRequest 构造中真实填充 |
| 7 | D-02a: middleware 同步调用 LogUsage,MultiAuth 内无 `go func()` 包裹 LogUsage | VERIFIED | `apikey.go:76` 同步调用 `usageLogger.LogUsage(c.Request.Context(), ...)`; `grep -c "go func()" internal/middleware/apikey.go` = 0; LogUsage 内部已 `go logUsageAsync()` (usage_logger.go:50),无双重 goroutine |
| 8 | D-04: 写入失败用 `applogger.Errorf("[USAGE_LOG] 写入失败 key=%s path=%s: %v", req.APIKeyID, req.Path, err)` 暴露 | VERIFIED | `usage_logger.go:8` import `applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"` + `usage_logger.go:83` 字面命中 `applogger.Errorf("[USAGE_LOG] 写入失败 key=%s path=%s: %v", req.APIKeyID, req.Path, err)`; 原 `_ = err` 静默吞错已替换 |

**Score:** 8/8 must-haves verified

### Required Artifacts (Level 1-3 — Exists / Substantive / Wired)

| Artifact | Exists | Substantive | Wired | Status |
|----------|--------|-------------|-------|--------|
| `internal/middleware/apikey.go` (D-01/D-02a 落地) | YES (326 行) | YES (L63-85 完整 post-c.Next() capture 块 + 三字段填充) | YES (`apikey_integration_test.go:283` `router.Use(MultiAuth(fakeSvc, realLogger))` 真实调用) | VERIFIED |
| `internal/services/usage_logger.go` (D-02/D-04 落地) | YES (86 行) | YES (L60-61 detached ctx + L83 applogger.Errorf) | YES (`apikey_integration_test.go:272` `services.NewUsageLogger(db)` + `usage_logger_test.go:484` `NewUsageLogger(db)` 真实装配) | VERIFIED |
| `internal/middleware/apikey_integration_test.go` (SC#1/#2 测试) | YES (347 行) | YES (L268-306 SC#1 + L310-346 SC#2 + L255-263 waitForUsageLog helper) | YES (`go test` 直接运行,PASS) | VERIFIED |
| `internal/services/usage_logger_test.go` (SC#4 测试) | YES (583 行) | YES (L482-511 SC#4 + L467-475 waitForUsageLog helper) | YES (`go test` 直接运行,PASS) | VERIFIED |
| `internal/services/system/apikey_service_test.go` (SC#3 测试) | YES (1323 行) | YES (L1258-1323 SC#3 混合 seed + 三道断言) | YES (`go test` 直接运行,PASS) | VERIFIED |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `apikey.go:76` `usageLogger.LogUsage(c.Request.Context(), &services.LogUsageRequest{...})` | `usage_logger.go:48-52` `LogUsage` → `go s.logUsageAsync(ctx, req)` | 接口契约 `services.UsageLogger` 1 方法 (`LogUsage(ctx, *LogUsageRequest) error`) | WIRED | 编译通过 + 真实 sqlite DB 行实证 (SC#1/SC#2 测试均产生行) |
| `apikey.go:67` `statusCode := c.Writer.Status()` | `apikey.go:84` `Success: statusCode >= 200 && statusCode < 300` | 同步 capture (L67-L68) → 字段填充 (L82-L84) | WIRED | L67 在 `c.Next()` 之后,字面表达式 L84 派生自 L67 值;SC#1 测试断言 StatusCode=200 + Success=true;SC#2 测试断言 StatusCode=403 + Success=false |
| `usage_logger.go:60` `context.WithTimeout(context.Background(), 10*time.Second)` | `usage_logger.go:79` `s.db.WithContext(detachedCtx).Create(...)` | detached ctx 替代调用方 ctx | WIRED | `grep WithContext(detachedCtx)` 命中 L79;`grep WithContext(ctx)` 在 usage_logger.go 命中 0;SC#4 测试用预取消 ctx 仍能落库,PASS |
| `usage_logger.go:83` `applogger.Errorf("[USAGE_LOG] 写入失败 key=%s path=%s: %v", req.APIKeyID, req.Path, err)` | `pkg/logger/logger.go:206-208` `Errorf` 公开 API | applogger 别名 + 公共 Errorf | WIRED | import alias `applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"` (L8) + 调用形态与 `config_backup_service.go:247` 一致 |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| `apikey.go` MultiAuth post-c.Next() capture block | `statusCode`, `duration` | `c.Writer.Status()` (gin 内部维护的 int) + `time.Since(start).Milliseconds()` (单调时钟) | YES (SC#1 测试断言 `log.StatusCode == 200` + `log.Duration > 0` 均通过) | FLOWING |
| `usage_logger.go` logUsageAsync DB insert | `usageLog` struct | LogUsageRequest 全部字段 + CreatedAt=time.Now() | YES (SC#1/SC#2/SC#4 测试均成功 `db.Where("api_key_id = ?").First(&log)` 读取到行,字段值匹配预期) | FLOWING |
| `apikey_service.go` GetUsageLogSummary | `summary.SuccessRate` | DB COUNT WHERE success=true / COUNT(*) * 100 | YES (SC#3 测试 seed 2×true + 2×false,断言 `InDelta(50.0, SuccessRate, 0.1)` PASS) | FLOWING |

### Behavioral Spot-Checks (re-run by verifier 2026-08-13)

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| 全项目编译通过 | `go build ./internal/middleware/... ./internal/services/...` | exit 0,无输出 | PASS |
| SC#1 DB 行实证 | `go test ./internal/middleware/ -run TestMultiAuthUsageLogTiming -v` | PASS (0.04s) | PASS |
| SC#2 下游失败 DB 行实证 | `go test ./internal/middleware/ -run TestMultiAuthUsageLogFailure -v` | PASS (0.02s) | PASS |
| SC#3 successRate 聚合防回归 | `go test ./internal/services/system/ -run TestGetUsageLogSummaryMixed -v` | PASS (0.03s) | PASS |
| SC#4 cancel-race 防回归 | `go test ./internal/services/ -run TestLogUsageCancelledCtxStillWrites_D02 -v` | PASS (0.04s) | PASS |
| D-03a anchor: Phase 57 fake 测试 | `go test ./internal/middleware/ -run TestMultiAuthIntegration -v` | PASS — 3 子测试全绿 (有效key+正确scope→200 / 有效key+缺失scope→403 / 无效key→401) | PASS |
| D-03a anchor: 既有成功率计算 70% 用例 | `go test ./internal/services/system/ -run "^TestGetUsageLogSummary$" -v` | PASS — 7 子测试全绿 (含「成功率计算」line 1096) | PASS |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| (none) | — | — | — | 无 TBD/FIXME/XXX/HACK/PLACEHOLDER/TODO 债务标记;无 placeholder 字面量;无空 handler;无 console.log-only 实现 |

**Pitfall 字面检查:**
- `grep "AutoMigrate.*APIKeyUsageLog" internal/middleware/apikey_integration_test.go internal/services/usage_logger_test.go internal/services/system/apikey_service_test.go` → 0 hits (pitfall 1 规避)
- `grep -c "go func()" internal/middleware/apikey.go` → 0 (D-02a 去除冗余外层 goroutine 确认)
- `grep -c "_ = err" internal/services/usage_logger.go` → 0 (D-04 不静默吞错确认,注:仅有 `_ = ctx` 显式标注,语义不同)

### Requirement Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| OBSERV-01 | 59-01-PLAN.md + 59-02-PLAN.md | API Key 使用日志在请求处理完成后记录,`StatusCode`/`Duration`/`Success` 取真实值 (修复 P1-2) | SATISFIED | `apikey.go:63-85` post-c.Next() capture + 三字段填充;SC#1 (`TestMultiAuthUsageLogTiming`) + SC#2 (`TestMultiAuthUsageLogFailure`) DB 行实证 PASS |
| OBSERV-02 | 59-02-PLAN.md | `GetUsageLogSummary.successRate` 基于真实 `Success` 字段聚合,不再恒 ≈ 0% | SATISFIED | `apikey_service.go:519` 聚合逻辑未改 (连锁自动可信);SC#3 (`TestGetUsageLogSummaryMixed`) seed 2×true + 2×false → 50% PASS |
| OBSERV-03 | 59-01-PLAN.md + 59-02-PLAN.md | 异步 goroutine 使用独立、不被请求 ctx 取消的 context (消除 P2-b) | SATISFIED | `usage_logger.go:60-62` `context.WithTimeout(context.Background(), 10*time.Second)` + `_ = ctx` 标注;SC#4 (`TestLogUsageCancelledCtxStillWrites_D02`) 预取消 ctx 仍落库 PASS |

**Orphaned Requirements Check:** REQUIREMENTS.md grep 显示 OBSERV-01/02/03 均映射到 Phase 59,且两份 PLAN 的 `requirements:` 字段均声明 `[OBSERV-01, OBSERV-02, OBSERV-03]`。Phase 59 无未声明的孤儿需求。`requirements:` 字段完整覆盖 ROADMAP 分配。

### Human Verification Required

无。所有 5 项 ROADMAP Success Criteria 全部由 DB 行实证级别测试覆盖 (SC#1/#2/#4 真实 NewUsageLogger + sqlite 行断言;SC#3 直接 seed + 聚合断言;SC#5 全 suite PASS)。无视觉/实时/外部服务依赖型校验项;无 Nyquist 维度需要人工目视确认。

### Gaps Summary

无 gap。Phase 59 目标 100% 达成:

1. **OBSERV-01 (P1-2 记录时机过早)**: MultiAuth 记录点后移到 `c.Next()` 之后,`statusCode`/`duration`/`Success` 三字段真实填充。`apikey.go:67-68` capture + `apikey.go:82-84` 填充。
2. **OBSERV-02 (successRate ≈ 0% 永久失真)**: `Success` 字段链路打通后,`apikey_service.go:519` 聚合逻辑零修改即自动可信 (连锁点)。SC#3 50% 精确锚测试锁住防回归。
3. **OBSERV-03 (P2-b 请求 ctx 取消竞态)**: `logUsageAsync` 改用 `context.WithTimeout(context.Background(), 10s)` detached ctx 写 DB,调用方 cancel 不再传播。SC#4 预取消 ctx 测试锁住防回归。
4. **附加改进 (D-04)**: 写入失败用 `applogger.Errorf` 暴露,替代原 `_ = err` 静默吞错,提升观测系统自身可观测性。
5. **附加改进 (D-02a)**: middleware 去除冗余外层 `go func(){}` 包装 (LogUsage 内部已 `go logUsageAsync()`),消除双重 goroutine 复杂度。

四项 plan must_haves (D-01/D-02/D-02a/D-04) 字面落地,接口契约零破坏 (UsageLogger interface / LogUsage signature / LogUsageRequest struct / NewUsageLogger constructor / LogUsage 内部 `go logUsageAsync` 调用形态全部不变);3 requirement IDs 全部 SATISFIED;0 anti-pattern;0 human verification items。

**Status rationale:** 8/8 truths VERIFIED + 所有 artifacts Level 1-4 全通过 + 所有 key links WIRED + 3 requirements SATISFIED + 0 anti-patterns + 行为 spot-checks 全 PASS + 0 human verification items → status = **passed**。

---

_Verified: 2026-08-13T10:15:00Z_
_Verifier: Claude (gsd-verifier)_
