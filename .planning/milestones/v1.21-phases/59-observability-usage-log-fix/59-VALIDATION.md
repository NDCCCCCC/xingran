---
phase: 59
slug: observability-usage-log-fix
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-13
---

# Phase 59 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> 内容源自 `59-RESEARCH.md` §Validation Architecture（Nyquist Dimension 8 验证架构）。

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + testify v1.11.1（assert / require，含 `require.Eventually`） |
| **Config file** | none — 测试紧邻源码 `*_test.go`（`.planning/codebase/TESTING.md` 约定） |
| **Quick run command** | `go test ./internal/middleware/... ./internal/services/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~数秒（sqlite 文件 DB，无外部 PG/Redis 进程） |

**真实 DB（SC#1/#3/#4 要求 DB 行实证）：** sqlite 文件 DB —— 复用既有 `setupUsageLoggerTestDB`（per-test 独立文件 `os.TempDir()` + `UnixNano()` + `pid` 唯一名 + `busy_timeout=5000`，裸 `CREATE TABLE` DDL 绕过 `gen_random_uuid()` PG 专有陷阱）。**不**用 `t.TempDir()`（fire-and-forget goroutine 测试结束后仍写文件）。

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/middleware/... ./internal/services/...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd:verify-work`:** Full suite must be green；SC#1/#4 必须 **DB 行实证**（非代码推断）
- **Max feedback latency:** ~10 秒

---

## 异步写入确定性策略（Nyquist Dimension 8 关键）

- **机制:** `require.Eventually(predicate, 2*time.Second, 10*time.Millisecond)` 轮询 DB 行数。
- **为何确定性:** 条件满足立刻返回（不浪费时间），超时则**确定性失败**（非偶发 flaky）；与 fire-and-forget goroutine「最终一致」语义天然对齐。替代既有 `time.Sleep` flaky 反模式。
- **封装 helper:** `waitForUsageLog(t, db, apiKeyID, want)`（见 `59-RESEARCH.md` §异步写入可测试性机制）。
- **SC#4 专用:** 预取消 ctx 后轮询——修复前行永不出现（超时失败），修复后行出现（成功）= P2-b 防回归核心断言。

---

## Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|--------------|
| OBSERV-01 / SC#1 | 2xx 请求后日志行 `StatusCode∈2xx` / `Duration>0` / `Success=true` | 集成（真实 gin.Engine + 真实 `NewUsageLogger` + sqlite） | `go test ./internal/middleware/ -run TestMultiAuthUsageLogTiming -v` | ❌ Wave 0 新增 |
| OBSERV-01 / SC#2 | 下游失败（403/429/500，**非 pre-auth 401**）后 `Success=false` / `StatusCode=真实码` | 集成（handler 返回失败码） | `go test ./internal/middleware/ -run TestMultiAuthUsageLogFailure -v` | ❌ Wave 0 新增 |
| OBSERV-02 / SC#3 | 混合 success 行后 `successRate ∈ (0,100)`，不恒 ≈ 0% | 单元（seed DB 行 + 调 `GetUsageLogSummary`） | `go test ./internal/services/system/ -run TestGetUsageLogSummaryMixed -v` | ❌ Wave 0 新增 |
| OBSERV-03 / SC#4 | 调用方 ctx cancel 后日志仍完整写入（detached context） | 单元（`NewUsageLogger` + 预取消 ctx + `require.Eventually`） | `go test ./internal/services/ -run TestLogUsageCancelledCtx -v` | ❌ Wave 0 新增 |
| SC#5 | 全绿 + 防回归（P1-2 时序 + P2-b 取消） | 全 suite | `go test ./internal/middleware/... ./internal/services/...` | ✅ 既有 + 新增 |

> 任务 ID（`{N}-01-01` 形式）在 planner 创建 PLAN.md 后回填。上表已锁定每条 Requirement/SC 的验证机制、测试类型与命令，planner 不得偏离。

---

## Wave 0 Requirements

- [ ] `internal/middleware/apikey_integration_test.go` — 新增 `TestMultiAuthUsageLogTiming`（SC#1）+ `TestMultiAuthUsageLogFailure`（SC#2）子测试；复用既有 `setupUsageLoggerTestDB`，注入**真实** `NewUsageLogger(db)`（非 fake）
- [ ] `internal/services/usage_logger_test.go` — 新增 `TestLogUsageCancelledCtxStillWrites_D02`（SC#4）+ `waitForUsageLog` helper
- [ ] `internal/services/system/apikey_service_test.go` — 新增 `TestGetUsageLogSummaryMixed`（SC#3）混合 success 行聚合
- [ ] Phase 57 既有 fake 测试**原样保留**（D-03a，职责正交不回归）

*既有测试基建覆盖认证链；本 phase 新增真实 DB 时序/字段/取消鲁棒性测试，fake ↔ 真实 DB 职责正交并存。*

---

## Hermetic 性保证（sqlite 测试 DB）

- **Per-test 独立文件 DB:** `os.TempDir()` + `UnixNano()` + `pid` 唯一名 → 测试间零共享状态。
- **`busy_timeout=5000`:** 写锁排队而非立即报错，消除并发 goroutine 撞锁。
- **不用 `t.TempDir()`:** fire-and-forget goroutine 测试结束后仍可能写文件，自动 cleanup 删占用文件 mark fail。
- **裸 `CREATE TABLE` DDL:** 绕过 `gen_random_uuid()` PG 专有陷阱（`59-RESEARCH.md` §Pitfall 1）。
- **零外部进程:** 无需启动 PostgreSQL/Redis，CI 友好。

---

## Manual-Only Verifications

*None — 全部 phase 行为均有自动化验证（DB 行实证 + suite 全绿）。*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
