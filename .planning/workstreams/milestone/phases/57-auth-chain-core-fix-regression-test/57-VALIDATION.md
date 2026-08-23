---
phase: 57
slug: auth-chain-core-fix-regression-test
status: ready
nyquist_compliant: true
wave_0_complete: false
created: 2026-08-13
last_updated: 2026-08-13
---

# Phase 57 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from `57-RESEARCH.md` §Validation Architecture + §Security Domain.
> Task IDs backfilled from `57-01-PLAN.md` (single-plan phase, 2 tasks, Wave 1).

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` + testify v1.11.1 (`assert`/`require`) |
| **Config file** | 无独立配置文件；`go test` 直接驱动，`gin.SetMode(gin.TestMode)` 在测试内设置 |
| **Quick run command** | `go test ./internal/middleware/ -run "TestMultiAuth|TestIsValidKeyFormat|TestIsIPAllowed|TestGetRequiredScope" -v` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go build ./... && go test ./internal/middleware/ -v`
- **After every plan wave:** Run `go test ./...`（全量，含既有 `apikey_test.go` 3 纯函数 + `apikey_service_test.go` + `usage_logger_test.go` + `rate_limiter_test.go` 不回归）
- **Before `/gsd:verify-work`:** 全量 `go test ./...` 全绿 + grep 证据（无 workaround 残留 + 构造函数有真实调用点）后才可进入验证
- **Max feedback latency:** ~15 seconds

---

## Per-Task Verification Map

> Task IDs `{N}-{plan}-{task}` backfilled from `57-01-PLAN.md` (2 tasks, single plan, single wave).

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 57-01-1 | 01 | 1 | AUTH-01 | T-57-01 / 认证后上下文丢失 (EoP) | `setUserContextForAPIKey` 签名 `interface{}`→`*models.APIKey` 后 7 个 context 键真实写入（4 必断言键：`user_id`/`api_key_id`/`scopes`/`auth_type="api_key"`）；`apiKeyType` 局部值类型与 `避免循环导入` 注释消除 | compile + grep gate | `go build ./... && go vet ./internal/middleware/` + grep assertions (acceptance_criteria #1-#10) | ✅ 既有 apikey.go (modify) | ⬜ pending |
| 57-01-1 | 01 | 1 | AUTH-02 | T-57-03 / RequireAPIKeyResourcePermission c.Next() 副作用 (Tampering) | `RequireAPIKeyResourcePermission` 重写为注册期委托 `return RequireScope(getRequiredScope(action))`，不再含 `RequireScope(requiredScope)(c)` 内联调用 | compile + grep gate | `go build ./...` + `grep -c "RequireScope(requiredScope)(c)" internal/middleware/apikey.go == 0` | ✅ 既有 apikey.go (modify) | ⬜ pending |
| 57-01-2 | 01 | 1 | AUTH-01 | T-57-01 | 修复后的链路真实把 4 个 context 键写入 gin context（由 handler 内 assert 断言，非手工打印） | integration | `go test ./internal/middleware/ -run "TestMultiAuthIntegration/有效key" -v` | ❌ W0 新建 (apikey_integration_test.go) | ⬜ pending |
| 57-01-2 | 01 | 1 | AUTH-02 | T-57-02 / 中间件死代码 (认证绕过风险，未挂载状态) | 4 中间件类型签名自洽 + `NewUsageLogger(db)`/`NewRateLimiter()` 构造函数真实实例化（D-02 证据） | integration | `go test ./internal/middleware/ -run TestConstructorsCallable_D02 -v` | ❌ W0 新建 (apikey_integration_test.go) | ⬜ pending |
| 57-01-2 | 01 | 1 | QUAL-02 | — | 有效 key + 正确 scope → 200 | integration | `go test ./internal/middleware/ -run "TestMultiAuthIntegration/有效key\+正确scope" -v` | ❌ W0 新建 | ⬜ pending |
| 57-01-2 | 01 | 1 | QUAL-02 | — | 有效 key + 缺失 scope → 403 | integration | `go test ./internal/middleware/ -run "TestMultiAuthIntegration/有效key\+缺失scope" -v` | ❌ W0 新建 | ⬜ pending |
| 57-01-2 | 01 | 1 | QUAL-02 | — | 无效 key → 401 | integration | `go test ./internal/middleware/ -run "TestMultiAuthIntegration/无效key" -v` | ❌ W0 新建 | ⬜ pending |
| — (既有) | — | — | QUAL-02 | — | 既有 3 纯函数测试（TestIsValidKeyFormat/TestIsIPAllowed/TestGetRequiredScope）不回归 | unit (既有) | `go test ./internal/middleware/ -run "TestIsValidKeyFormat\|TestIsIPAllowed\|TestGetRequiredScope" -v` | ✅ 既有 `apikey_test.go` | ⬜ baseline |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

> Wave 0 = 本 phase 的"新建/修复前置"。57-01-PLAN.md Task 1 + Task 2 共同覆盖以下全部前置：

- [x] 源码修复 `internal/middleware/apikey.go` — P0-2 签名改 `*models.APIKey` + 直接 import `internal/models` + P0-1 反模式重写 (Task 57-01-1)
- [x] `internal/middleware/apikey_integration_test.go` — 新建，覆盖 AUTH-01/AUTH-02/QUAL-02 集成测试 + D-02 证据用例 (Task 57-01-2)
- [x] fake 类型定义（`fakeAPIKeyService` 9 方法仅 `ValidateAPIKey` 给真值 + `fakeUsageLogger` 1 方法）— 同文件内 (Task 57-01-2)
- [x] `setupUsageLoggerTestDB` helper — 从 `internal/services/usage_logger_test.go` 模式复制 (Task 57-01-2)

*既有 `apikey_test.go` 3 纯函数测试已覆盖 `isValidKeyFormat`/`isIPAllowed`/`getRequiredScope`，不回归即可。*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `RequireAPIKeyResourcePermission` 反模式消除 | AUTH-02 | diff 审查（编译无法证明"不靠 c.Next() 副作用"的语义） | 读 `apikey.go` diff，确认不再有 `RequireScope(...)(c)` 内联调用；改为 `return RequireScope(getRequiredScope(action))` 注册期委托 |
| 4 中间件类型签名一致 | AUTH-02 | 签名审查（grep 编译可证存在，但"自洽"需人读） | 确认 `MultiAuth(system.APIKeyService, services.UsageLogger)` / `RequireScope(string)` / `RequireAPIKeyResourcePermission(string, string)` / `RateLimitByScope(*services.RateLimiter)` 四签名不变 |
| `resource` 参数仍忽略 / `getScopeFromContext` 仍取 `scopes[0]` / `string(rune(result.Limit))` P2-a 未改 / MultiAuth usage logger goroutine 未动 | AUTH-02 (scope fence) | 防御性确认——这四项延后 Phase 60/61，本 phase 不得触碰 | 读 diff 确认**未改动**这四处（见 57-01-PLAN.md Task 1 acceptance_criteria #9/#10 + scope fence 全集） |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify (每 task 均有 `go build` / `go test` 自动验证)
- [x] Wave 0 covers all MISSING references (Task 1 修源码 + Task 2 新建测试，互为前置)
- [x] No watch-mode flags
- [x] Feedback latency < 15s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** ready (planner signed off 2026-08-13)
