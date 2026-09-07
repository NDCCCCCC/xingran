---
plan: 57-01
phase: 57-auth-chain-core-fix-regression-test
status: complete
requirements: [AUTH-01, AUTH-02, QUAL-02]
completed: 2026-08-13
---

# Plan 57-01 Summary: API Key 认证链核心修复 + 回归测试

## Objective
修复 API Key 认证链两个 P0 确定性缺陷并用集成测试锁住完整链路防回归。Phase 57 是 v1.21 milestone 首个 phase，对 v1.6「API 密钥管理系统」(Phase 16) 的回归修复。

## Tasks Completed

| Task | Commit | Description |
|------|--------|-------------|
| 1 | `4c86752` | **AUTH-01**: `setUserContextForAPIKey` 签名 `interface{}`→`*models.APIKey`（消除恒 false 类型断言，7 个 c.Set 真实执行）；**AUTH-02**: `RequireAPIKeyResourcePermission` 改注册期委托 `return RequireScope(getRequiredScope(action))`（消除内联 `RequireScope(...)(c)` 反模式） |
| 2 | `cdc5144` | **QUAL-02**: 三路径集成测试 + D-02 构造函数证据 + 手写 fake |

## Key Files

**Modified:**
- `internal/middleware/apikey.go` — P0-2 类型断言修复 + P0-1 反模式重写

**Created:**
- `internal/middleware/apikey_integration_test.go` — `TestMultiAuthIntegration`(200/403/401) + `TestConstructorsCallable_D02`(D-02 证据) + fakeAPIKeyService(9 方法) + fakeUsageLogger(1 方法)

## Decisions Honored (D-01 ~ D-05)

- **D-05 (AUTH-01):** `setUserContextForAPIKey` 签名改为 `(c, apiKey *models.APIKey, scopes)`，直接 import `internal/models`；局部 `apiKeyType` + `interface{}` workaround + 「避免循环导入」误判注释全部移除；7 个 c.Set 提升到函数主体
- **D-04 (零行为变更):** 保留全部 7 个 context 键 + `username = apiKey.Name` 语义（密钥名而非登录名，语义修正延后 Phase 61）
- **D-03 (AUTH-02):** `RequireAPIKeyResourcePermission` 用最小委托写法（注册期算 scope + `return RequireScope(...)`），消除靠 `c.Next()` 副作用推进的内联反模式；`resource` 参数仍忽略（延后 Phase 61 AUTH-04）
- **D-02 (构造函数证据):** `TestConstructorsCallable_D02` 真实实例化 `services.NewUsageLogger(db)` + `services.NewRateLimiter()`（无参版，非 operations 令牌桶版）
- **D-01 (三路径测试):** 手写 fake（无 gomock）+ 真实 `gin.Engine` + `httptest` 驱动 `MultiAuth`，符合 TESTING.md 既有 testify 风格

## Scope Fence（全部遵守，未越界到 Phase 59/60/61）
- `resource` 参数仍忽略 → Phase 61 AUTH-04
- `getScopeFromContext` 仍取 `scopes[0]` → Phase 61 QUAL-03
- `string(rune(result.Limit))` P2-a 编码 bug 未改 → Phase 60 QUAL-01
- MultiAuth usage logger 异步 goroutine 未动 → Phase 59 (P1-2/P2-b)
- 不在生产路由挂载 MultiAuth → Phase 60 AUTH-03 决策点
- 零新增外部依赖，go.mod 未动

## Verification
- `go build ./...` exit 0
- `go vet ./internal/middleware/` exit 0
- **Task 1:** 12 条 acceptance criteria 全过（注：#9 实际 token 为 `userScopes[0]` 而非 plan 笔误的 `scopes[0]`，语义意图「getScopeFromContext 未动」已满足）
- **Task 2:** `TestMultiAuthIntegration` 三子测试 PASS（SC#1 四键断言 user_id/api_key_id/scopes/auth_type 非空 + SC#3 三路径 200/403/401）+ `TestConstructorsCallable_D02` PASS（SC#2）+ 既有 `TestIsValidKeyFormat`/`TestIsIPAllowed`/`TestGetRequiredScope` 不回归
- 测试在隔离 worktree 复现全绿（`ok internal/middleware 0.233s`）
- **全量回归:** `pkg/errors` + `tests/integration` 既有 pre-existing 失败（PROJECT.md 记录，先于 Phase 51-54），与本 plan 无关

## Deviations
- **Task 2 提交延迟（共享工作树冲突，已解决）：** Task 1 提交（`4c86752`）后、Task 2 提交前，一个并发工作流（`refactor/core-review-fixes` quick task）把共享工作树 HEAD 切走，executor 遵守 `<destructive_git_prohibition>` 停在 checkpoint 未冒险 checkout/stash。经用户决策，用独立 git worktree（项目外路径 `guoguo-p57wt`，避开 `.claude/worktrees` redirect 问题）完成 Task 2 提交 + SUMMARY。两个 task 均落在指派分支 `refactor/config-ctx-and-viper-cleanup`，内容完整，测试在隔离 worktree 复现全绿。
- **环境 note：** Claude Code Agent `isolation="worktree"` 因 `.claude/worktrees` 路径 git redirect（Windows 路径大小写不一致）创建失败，本 plan 改用 sequential 内联执行 + 手动 worktree 绕开冲突。

## Self-Check: PASSED
