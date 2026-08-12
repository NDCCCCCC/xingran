---
phase: 57
slug: auth-chain-core-fix-regression-test
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-13
---

# Phase 57 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.
> Derived from `57-RESEARCH.md` §Validation Architecture + §Security Domain.

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

> Task IDs (`{N}-{plan}-{task}`) 由 planner 在 PLAN.md 创建后回填。下表为需求→测试的初始映射，源自 `57-RESEARCH.md` §Phase Requirements → Test Map。

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| TBD | 01 | 1 | AUTH-01 | T-57-01 / 认证后上下文丢失 (EoP) | `setUserContextForAPIKey` 修复后写入 7 个 context 键（4 必断言键：`user_id`/`api_key_id`/`scopes`/`auth_type="api_key"`） | integration | `go test ./internal/middleware/ -run TestMultiAuthIntegration -v` | ❌ W0 新建 | ⬜ pending |
| TBD | 01 | 1 | AUTH-02 | T-57-02 / 中间件死代码 (认证绕过风险) | 4 中间件类型签名自洽 + `NewUsageLogger(db)`/`NewRateLimiter()` 构造函数真实实例化（D-02 证据） | integration | `go test ./internal/middleware/ -run TestConstructorsCallable_D02 -v` | ❌ W0 新建 | ⬜ pending |
| TBD | 01 | 1 | AUTH-02 | — | `RequireAPIKeyResourcePermission` 重写后不靠 `c.Next()` 副作用推进 | unit (编译 gate) | `go vet ./internal/middleware/ && go build ./...` | ❌ W0 (编译 gate) | ⬜ pending |
| TBD | 01 | 1 | QUAL-02 | — | 有效 key + 正确 scope → 200 通过 | integration | `go test ./internal/middleware/ -run "TestMultiAuthIntegration/有效key" -v` | ❌ W0 新建 | ⬜ pending |
| TBD | 01 | 1 | QUAL-02 | — | 有效 key + 缺失 scope → 403 | integration | `go test ./internal/middleware/ -run "TestMultiAuthIntegration/有效key_缺失scope" -v` | ❌ W0 新建 | ⬜ pending |
| TBD | 01 | 1 | QUAL-02 | — | 无效 key → 401 | integration | `go test ./internal/middleware/ -run "TestMultiAuthIntegration/无效key" -v` | ❌ W0 新建 | ⬜ pending |
| — (既有) | — | — | QUAL-02 | — | 既有 3 纯函数测试不回归 | unit (既有) | `go test ./internal/middleware/ -run "TestIsValidKeyFormat|TestIsIPAllowed|TestGetRequiredScope" -v` | ✅ 既有 `apikey_test.go` | ⬜ baseline |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `internal/middleware/apikey_integration_test.go` — 新建，覆盖 AUTH-01/AUTH-02/QUAL-02 集成测试 + D-02 证据用例
- [ ] fake 类型定义（`fakeAPIKeyService` 9 方法仅 `ValidateAPIKey` 给真值 + `fakeUsageLogger` 1 方法）— 同文件内
- [ ] `setupUsageLoggerTestDB` helper — 从 `internal/services/usage_logger_test.go` 模式复制（或简化版，仅需不 nil `*gorm.DB`）
- [ ] 源码修复 `internal/middleware/apikey.go` — P0-2 签名改 `*models.APIKey` + 直接 import `internal/models` + P0-1 反模式重写

*既有 `apikey_test.go` 3 纯函数测试已覆盖 `isValidKeyFormat`/`isIPAllowed`/`getRequiredScope`，不回归即可。*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `RequireAPIKeyResourcePermission` 反模式消除 | AUTH-02 | diff 审查（编译无法证明"不靠 c.Next() 副作用"的语义） | 读 `apikey.go` diff，确认不再有 `RequireScope(...)(c)` 内联调用 |
| 4 中间件类型签名一致 | AUTH-02 | 签名审查（grep 编译可证存在，但"自洽"需人读） | 确认 `MultiAuth(system.APIKeyService, services.UsageLogger)` / `RequireScope(string)` / `RequireAPIKeyResourcePermission(string, string)` / `RateLimitByScope(*services.RateLimiter)` |
| `resource` 参数仍忽略 / `getScopeFromContext` 仍取 `scopes[0]` 确认未动 | AUTH-02 (scope fence) | 防御性确认——这两项延后 Phase 61，本 phase 不得触碰 | 读 diff 确认**未改动**这两处 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 15s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
