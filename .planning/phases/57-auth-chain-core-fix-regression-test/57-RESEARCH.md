# Phase 57: 认证链核心修复 + 回归测试 - Research

**Researched:** 2026-08-13
**Domain:** Go 后端认证中间件 (Gin) + 类型断言修复 + 集成测试
**Confidence:** HIGH

## Summary

本 phase 是对 v1.6「API 密钥管理系统」(Phase 16) 的回归修复。核心病灶是两个已查明的确定性缺陷：P0-2（`setUserContextForAPIKey` 把 `*models.APIKey` 指针断言为局部值类型 `apiKeyType`，恒 false，导致 API Key 认证后 gin context 从未被写入）与 P0-1（`MultiAuth` 及其下游四个中间件从未挂载任何生产路由 + `RequireAPIKeyResourcePermission` 内含 `c.Next()` 副作用推进的脆弱写法）。

本次研究通过直接审读 `internal/middleware/apikey.go`、`internal/services/system/apikey_service.go`、`internal/models/api_key.go`、`internal/services/usage_logger.go`、`internal/services/rate_limiter.go` 与 `pkg/response/response.go` 的源码，**逐行确认了 STATE.md 根因表的每一项**，并给出了可落地的 HOW：精确的签名变更（`interface{}` → `*models.APIKey`）、`RequireAPIKeyResourcePermission` 的最小正确重写、手写 fake 满足的接口形状、以及基于项目既有 `gin.SetMode(gin.TestMode)` + `httptest` 约定的三条路径集成测试骨架。所有库调用（gin、testify）均与项目内 18 个既有 `*_test.go` 的写法一致，无需引入新依赖。

**Primary recommendation:** 严格按 D-05 改签名 + 直接 import `internal/models`（循环依赖已通过 grep 双向验证排除）；按 D-03 用"注册期计算 scope + 直接返回 `RequireScope(requiredScope)`"的最小写法消除内联 `RequireScope()(c)` 反模式；集成测试用手写 fake（满足 `system.APIKeyService` 9 方法 + `services.UsageLogger` 1 方法）驱动真实 `gin.Engine` 跑 `MultiAuth`，并额外实例化真实 `NewUsageLogger(db)` / `NewRateLimiter()` 证明构造函数可装配——不引入生产死代码。

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| API Key 提取与格式校验 (`extractAPIKey`/`isValidKeyFormat`) | API / Backend (middleware) | — | 请求头解析属 HTTP 边界，须在 handler 前完成 |
| API Key 业务校验 (`ValidateAPIKey`：查库+过期+IP 白名单) | API / Backend (service) | Database | `system.APIKeyService` 拥有查库逻辑；middleware 仅消费 `*models.APIKey` 结果 |
| gin context 写入 (`setUserContextForAPIKey`) | API / Backend (middleware) | — | 认证态注入请求上下文，是 middleware 专属职责（P0-2 病灶所在） |
| 作用域/资源权限校验 (`RequireScope`/`RequireAPIKeyResourcePermission`) | API / Backend (middleware) | — | 每请求拦截，基于 context 中已写入的 `scopes` 做决策 |
| 使用日志记录 (`UsageLogger`) | API / Backend (service) | Database | 异步落库；middleware 触发但实现属 service 层 |
| 限流 (`RateLimitByScope`) | API / Backend (middleware) | — (in-memory `sync.Map`) | 基于内存滑动窗口，无外部存储依赖 |
| 路由挂载 (`MultiAuth` 接入生产路由) | API / Backend (router) | — | **Phase 57 不挂载**——挂载属 Phase 60 AUTH-03 决策点 |

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01 (QUAL-02 测试替身):** 集成测试用**手写 fake/stub** 实现 `system.APIKeyService` + `services.UsageLogger`（fake 返回构造好的 `*models.APIKey`），通过**真实 `gin.Engine` + `httptest`** 跑 `MultiAuth` 中间件，断言 gin context 键被写入。无 DB 依赖、无 mock 框架，符合 TESTING.md 既有 testify 风格（"无 gomock"）。覆盖 SC#3 三条路径：① 有效 key+正确 scope→通过；② 有效 key+缺 scope→403；③ 无效 key→401。
- **D-02 (就绪可挂载证据):** 集成测试文件**额外实例化真实的 `NewUsageLogger(db)` / `NewRateLimiter()`**（用测试 DB 或 sqlite），证明构造函数可调用且类型签名与 MultiAuth 装配兼容。测试即真实调用点，不引入生产死代码。实际生产装配推迟 Phase 60。**注意 fake UsageLogger ≠ NewUsageLogger**。
- **D-03 (AUTH-02 审查深度):** 4 个中间件做到**内部自洽、无静默缺陷**。修**类型签名 + 调用路径反模式**：`RequireAPIKeyResourcePermission` 内联 `RequireScope(requiredScope)(c)` 改为正确的链式组合/直接逻辑。`resource` 参数被忽略、`getScopeFromContext` 只取 `scopes[0]`——这两项**不在 Phase 57 修**，已升级归 **Phase 61** (AUTH-04 / QUAL-03)。
- **D-04 (上下文键与 username 语义):** **保留现有行为**。仅修类型断言。保留全部 7 个 context 键（`user_id`/`username`/`nickname`/`api_key_id`/`scopes`/`auth_type`/`inherit_perms`）。`username := ak.Name`（Name 是 key 名而非用户名）**语义原样保留**——零行为变更。username 语义修正属 Phase 61。
- **D-05 (AUTH-01 修复方向):** `setUserContextForAPIKey(c, apiKey interface{}, scopes)` → `(c, apiKey *models.APIKey, scopes)`，直接 import `internal/models`，移除局部 `apiKeyType` 与 `interface{}` workaround。**已验证 `internal/models` 不导入 `internal/middleware`，无循环依赖**。

### Claude's Discretion
- 测试文件命名与组织（建议 `internal/middleware/apikey_integration_test.go`，与既有 `apikey_test.go` 3 个纯函数测试并存）。
- fake 实现的具体字段值构造（只要覆盖三条路径断言）。
- 内联 `RequireScope()(c)` 重构的具体写法（只要消除"靠 c.Next() 副作用推进"的脆弱性）。

### Deferred Ideas (OUT OF SCOPE)
- **资源级细粒度权限矩阵**（`RequireAPIKeyResourcePermission` 的 resource→permission 映射 + InheritPerms 资源校验）→ Phase 61 / AUTH-04。
- **限流生产接入与调优**（`RateLimitByScope` 多 scope 选择逻辑、生产路由全量接入）→ Phase 61 / QUAL-03。
- **username 语义修正**（`username=ak.Name` → 取关联 User 真名）→ Phase 61。
- **密钥轮换/吊销、配额告警** → FUTURE-APIKEY-03/04 (v2)。
- **MultiAuth 生产路由挂载** → Phase 60 AUTH-03（含安全评估）。
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| AUTH-01 | `setUserContextForAPIKey` 正确把 `user_id`/`api_key_id`/`scopes`/`auth_type` 写入 gin context（修复 P0-2 类型断言恒 false） | §「AUTH-01 修复规格」逐行给出签名变更、要删除的局部类型、修后将真正执行的 7 个 `c.Set`；§「循环依赖验证」证明 import `internal/models` 安全 |
| AUTH-02 | `MultiAuth` 及下游四中间件无死代码缺陷，类型签名/参数传递/作用域匹配经审查正确；`NewUsageLogger`/`NewRateLimiter` 有真实实例化路径 | §「AUTH-02 中间件自洽」给出 `RequireAPIKeyResourcePermission` 反模式根因 + 两种正确重写；§「构造函数签名与 D-02 证据路径」给出可装配证据；§「P0-1 死代码现状」确认未挂载 |
| QUAL-02 | 为 `MultiAuth`/`setUserContextForAPIKey`/`RequireScope` 补集成测试，覆盖完整链路防 P0-2 回归 | §「QUAL-02 测试规格」给出 fake 接口形状、gin+httptest 装配骨架、三条路径的精确断言、context 键断言清单 |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

本项目根 `CLAUDE.md` 是权威（全局 CLAUDE.md 是记忆系统，与本 phase 无关）。与 Phase 57 直接相关的指令：

- **响应包装**: middleware 已用 `response.Error(c, ...)` / `response.Success`——本 phase 维持，测试断言状态码须对齐 `pkg/response` 的 `ErrUnauthorized(HTTPStatus=401)` / `ErrForbidden(HTTPStatus=403)`。
- **编译验证**: CLAUDE.md 强制"任何 Go 改动后跑 `go build ./...`"——本 phase SC#4 即此项。
- **Status 约定**: `0=enabled/normal`，`1=disabled/stopped`——API Key 用 `IsActive bool`（`gorm:"default:true"`），不涉及此 0/1 约定，无冲突。
- **中间件分层**: 实际中间件位于 `internal/middleware/`（非 `pkg/middleware/`），ARCHITECTURE.md 图有偏差，以实际为准（CONTEXT.md 已确认）。
- **临时文件**: 根目录 `temp_*.go` / `test_*.go` 会导致 `main redeclared`——本 phase 测试文件写在 `internal/middleware/` 包内，命名 `apikey_integration_test.go`，不触犯。
- **Scope containment**: 修 bug 时"只修报告的问题，不主动改无关文件"——与 D-03/D-04 的"零行为变更、两项延后 Phase 61"完全一致。

## Standard Stack

### Core (本 phase 不引入新依赖，全部为既有)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/gin-gonic/gin` | (既有, go.mod) | HTTP 框架，`gin.HandlerFunc` 中间件 + `gin.TestMode` + `gin.CreateTestContext` | 项目唯一 web 框架；18 个既有测试文件均用此 `[VERIFIED: codebase]` |
| `github.com/stretchr/testify` | v1.11.1 (TESTING.md) | `assert` / `require` 断言 | TESTING.md 明确约定；`apikey_test.go` / `apikey_service_test.go` 既有用法 `[VERIFIED: codebase]` |
| `github.com/xingran-next/xingran-go-backend/internal/models` | (本仓) | `APIKey` struct (待 import 进 middleware) | 数据模型叶子包，无循环依赖 `[VERIFIED: codebase grep]` |
| `net/http/httptest` (stdlib) | go1.24 | `NewRecorder` / `NewRequest` 驱动 gin engine | 项目 `auth_integration_test.go` 等既有约定 `[VERIFIED: codebase]` |
| `gorm.io/driver/sqlite` | (既有, go.mod) | D-02 证据路径为 `NewUsageLogger(db)` 提供测试 DB | `usage_logger_test.go` 既有 `setupUsageLoggerTestDB` 模式 `[VERIFIED: codebase]` |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/xingran-next/xingran-go-backend/pkg/response` | (本仓) | `ErrUnauthorized`/`ErrForbidden`/`Error` — 测试状态码对齐基准 | 断言 `w.Code` 时须知道 middleware 走的是哪个 `AppError.HTTPStatus` |

**Installation:**
```bash
# 无需安装——本 phase 不引入任何新外部依赖
go mod tidy  # 仅当 go.sum 需要同步时
```

**Version verification:** 本 phase 零新增依赖，所有库已在 `go.mod`。已确认 `go build ./...` 退出码 0（baseline green）。

## Package Legitimacy Audit

> 本 phase **不安装任何外部包**——纯 Go stdlib + 既有项目依赖（gin / testify / gorm / sqlite driver 均已在 go.mod）。故无新包需要审计。fake 实现、测试骨架、签名修复全部用既有依赖完成。

**Packages removed due to slopcheck [SLOP] verdict:** none (gate not triggered — no new packages)
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram (API Key 认证链数据流)

```
 HTTP 请求 (带 X-API-Key 头)
        │
        ▼
 ┌─────────────────────────────────────────────────────────┐
 │ MultiAuth(apiKeyService, usageLogger)                   │  ← P0-1: 当前未挂载任何路由
 │   ├─ extractAPIKey(c) → "" ? ──yes──► c.Next() (退回 JWT 路径) │
 │   │                                   no                 │
 │   ├─ isValidKeyFormat ? ──no──► 401 ErrUnauthorized     │
 │   │                            yes                       │
 │   ├─ apiKeyService.ValidateAPIKey(ctx, keyStr)           │
 │   │      └─ 返回 *models.APIKey                          │
 │   │      └─ err ? ──yes──► 401  ◄── SC#3 路径③ 测试点    │
 │   │      └─ ok                                           │
 │   ├─ IP 白名单校验 (isIPAllowed) ──no──► 403             │
 │   │                                            yes        │
 │   ├─ setUserContextForAPIKey(c, apiKey, scopes) ◄── P0-2 病灶│
 │   │      └─ 断言 *models.APIKey → apiKeyType (值类型)     │
 │   │      └─ 恒 false → 7 个 c.Set 从未执行 ◄── SC#1 测试点│
 │   │      [修复后] 直接用 *models.APIKey → 7 个 c.Set 生效 │
 │   ├─ go usageLogger.LogUsage(...)  (异步, Phase 59 修)   │
 │   └─ c.Next() ─────────────────────────┐                │
 └─────────────────────────────────────────│────────────────┘
                                           ▼
 ┌─────────────────────────────────────────────────────────┐
 │ RequireScope(requiredScope)   (可挂载于后续路由组)        │
 │   ├─ c.Get("scopes") 不存在 → 403  ◄── SC#3 路径② 测试点 │
 │   ├─ 遍历 scopes，admin 或匹配 → c.Next()                │
 │   └─ 不匹配 → 403 ErrForbidden                           │
 └─────────────────────────────────────────────────────────┘
                                           ▼
 ┌─────────────────────────────────────────────────────────┐
 │ RequireAPIKeyResourcePermission(resource, action)        │
 │   └─ 当前: RequireScope(getRequiredScope(action))(c)     │ ◄── P0-1 反模式
 │      [修复后] 注册期算 scope，直接 return RequireScope()  │
 │      (resource 参数仍忽略 → Phase 61 AUTH-04)            │
 └─────────────────────────────────────────────────────────┘
                                           ▼
 ┌─────────────────────────────────────────────────────────┐
 │ RateLimitByScope(*services.RateLimiter)                  │
 │   └─ auth_type != "api_key" 跳过; 否则 Check + 设响应头  │
 │      (getScopeFromContext 取 scopes[0] → Phase 61 QUAL-03)│
 └─────────────────────────────────────────────────────────┘
                                           ▼
                                      业务 Handler
 (读 c.Get("user_id") / c.Get("api_key_id") / ...)
```

读者可从 `X-API-Key` 头一路追踪到 handler 读 context 键，定位 P0-2 断点（context 从未写入）与 SC#1/SC#3 三个测试切入点。

### Recommended Project Structure
```
internal/middleware/
├── apikey.go                      # P0-2 修复 + P0-1 反模式重写 (源码)
├── apikey_test.go                 # 既有 3 个纯函数测试 (不回归)
└── apikey_integration_test.go     # [新增] 集成测试 + D-02 证据 (QUAL-02)
```

### Pattern 1: AUTH-01 修复规格 (P0-2 / D-05)

**What:** 把 `setUserContextForAPIKey` 的 `apiKey interface{}` 参数改为 `*models.APIKey`，删除局部 `apiKeyType` 与类型断言分支。
**When to use:** AUTH-01 唯一修复点。

**当前病灶代码** (`internal/middleware/apikey.go:144-179`)：
```go
// Source: internal/middleware/apikey.go:144-179 (VERIFIED by direct read)
// setUserContextForAPIKey 设置API Key认证的用户上下文（私有函数）
// apiKey 参数的类型是 *models.APIKey，使用 interface{} 避免循环导入   ← 误判
func setUserContextForAPIKey(c *gin.Context, apiKey interface{}, scopes []string) {
	type apiKeyType struct {           // ← 局部 VALUE 类型
		ID           string
		Name         string
		UserID       *string
		InheritPerms bool
		User         *interface{}
	}
	if ak, ok := apiKey.(apiKeyType); ok {   // ← *models.APIKey 断言为 apiKeyType → 恒 false
		// 这 7 行 c.Set 永远不执行
		c.Set("user_id", userID); ...
	}
}
```

**根因链**：`apikey.go:40` `apiKey, err := apiKeyService.ValidateAPIKey(...)` 返回 `*models.APIKey`（`apikey_service.go:129,160` 确认）→ `apikey.go:58` 传入 `setUserContextForAPIKey(c, apiKey, ...)` 传的是**指针** → `apikey.go:157` 断言到**值类型** `apiKeyType` → Go 类型断言对 pointer≠value 恒返回 `ok=false`。

**修复后（推荐写法）**：
```go
import "github.com/xingran-next/xingran-go-backend/internal/models"  // 新增 import

// setUserContextForAPIKey 设置API Key认证的用户上下文（私有函数）
func setUserContextForAPIKey(c *gin.Context, apiKey *models.APIKey, scopes []string) {
	userID := ""
	if apiKey.UserID != nil {
		userID = *apiKey.UserID
	}
	c.Set("user_id", userID)
	c.Set("username", apiKey.Name)   // D-04: 语义保留 (Name=密钥名, 非 username)
	c.Set("nickname", "")
	c.Set("api_key_id", apiKey.ID)
	c.Set("scopes", scopes)
	c.Set("auth_type", "api_key")
	if apiKey.InheritPerms && apiKey.User != nil {
		c.Set("inherit_perms", true)
	}
}
```

**D-04 保留的 7 个 context 键**（顺序对应上面 7 个 `c.Set`）：
| # | 键 | 值来源 | 语义保留点 |
|---|----|--------|-----------|
| 1 | `user_id` | `*apiKey.UserID` (空则 `""`) | — |
| 2 | `username` | `apiKey.Name` | **D-04: Name 是密钥名而非登录名，原样保留** |
| 3 | `nickname` | `""` (API Key 无昵称) | — |
| 4 | `api_key_id` | `apiKey.ID` (来自 `BaseModel.ID`, UUID) | — |
| 5 | `scopes` | 入参 `scopes` (即 `apiKey.Scopes`) | — |
| 6 | `auth_type` | `"api_key"` (字面量) | — |
| 7 | `inherit_perms` | `true` (仅当 `InheritPerms && User != nil`) | 条件分支保留 |

SC#1 断言其中 4 个：`user_id` / `api_key_id` / `scopes` / `auth_type="api_key"` 非空。

### Pattern 2: AUTH-02 中间件自洽 (P0-1 反模式 / D-03)

**What:** `RequireAPIKeyResourcePermission` 内联 `RequireScope(requiredScope)(c)` 是脆弱写法——`RequireScope` 成功路径调 `c.Next()` 会推进 gin 全局 handler 链索引，而此调用是"直接函数调用"而非"gin 链挂载"，导致下游 handler 可能被重复执行或链索引错乱。
**When to use:** AUTH-02 唯一重构点。

**当前病灶** (`apikey.go:223-231`)：
```go
func RequireAPIKeyResourcePermission(resource string, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		requiredScope := getRequiredScope(action)
		RequireScope(requiredScope)(c)   // ← 反模式：内联调用，靠内部 c.Next() 副作用推进
	}
}
```

**推荐重写（最小写法，注册期委托）**：
```go
func RequireAPIKeyResourcePermission(resource string, action string) gin.HandlerFunc {
	requiredScope := getRequiredScope(action)   // 纯函数，注册期即可算
	return RequireScope(requiredScope)           // 直接返回中间件，由 gin 正常挂载
}
```
`getRequiredScope` 是纯函数（不依赖 `*gin.Context`，`apikey.go:235-249` 确认），故可在注册时计算。返回的 `RequireScope(requiredScope)` 作为真正的 handler 被 gin 挂载，其内部 `c.Next()` 处于正确链位置——反模式消除。

**备选重写（显式内联，为 Phase 61 资源逻辑预留）**：
```go
// 提取无副作用 helper（两个中间件共享，Phase 61 资源校验可复用）
func hasRequiredScope(c *gin.Context, requiredScope string) bool {
	scopes, exists := c.Get("scopes")
	if !exists { return false }
	userScopes, ok := scopes.([]string)
	if !ok { return false }
	for _, s := range userScopes {
		if s == "admin" || s == requiredScope { return true }
	}
	return false
}

func RequireAPIKeyResourcePermission(resource string, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		requiredScope := getRequiredScope(action)
		if _, exists := c.Get("scopes"); !exists {
			response.Error(c, response.ErrForbidden, "缺少权限作用域"); c.Abort(); return
		}
		if !hasRequiredScope(c, requiredScope) {
			response.Error(c, response.ErrForbidden, "权限不足"); c.Abort(); return
		}
		c.Next()
	}
}
```
CONTEXT.md 把"具体写法"留给 Claude 裁量——两种都满足"消除 c.Next() 副作用推进"。推荐用最小写法（diff 最小、provably correct）；若想为 Phase 61 留 hooks 则用显式内联。

**D-03 明确不在 Phase 57 修的两项**（须在 plan 里显式标注"延后"，避免 scope creep）：
1. `resource` 参数被忽略（line 223 入参 `resource string` 从未使用）→ Phase 61 AUTH-04。
2. `getScopeFromContext` 只取 `scopes[0]`（`apikey.go:310`）→ Phase 61 QUAL-03。

### Anti-Patterns to Avoid
- **跨包类型断言代替直接 import**: `interface{}` + 局部 mirror struct 是 P0-2 根因。Go 的类型断言对"同名不同包/值与指针"恒 false。直接 import 真实类型，让编译器帮你保证类型一致。
- **内联调用中间件 `mw()(c)`**: gin 中间件必须由引擎挂载（`r.Use`），其 `c.Next()` 才处于正确链位置。直接函数调用会让内部 `c.Next()` 错误推进全局链索引。
- **改 P0-2 时顺手"修正" username 语义**: D-04 锁定零行为变更。`username=ak.Name` 是既有契约，下游 handler 可能依赖，改它会破坏 SC#3 之外的未测路径。

## 循环依赖验证 (D-05 核心前提)

D-05 声称"`internal/models` 不导入 `internal/middleware`，无循环依赖"。本研究双向验证：

| 方向 | 验证方法 | 结果 |
|------|---------|------|
| middleware → models (拟新增) | `apikey.go` 加 `import ".../internal/models"` | 待加 |
| models → middleware (反向，须为空) | `Grep "internal/middleware" in internal/models/` | **No matches found** `[VERIFIED: codebase grep]` |
| models 包性质 | `internal/models/api_key.go` 仅 import `time`；`base.go` 仅 import `time`/`uuid`/`gorm` | models 是叶子数据包，无业务/HTTP 依赖 `[VERIFIED: codebase read]` |

**结论**: `internal/middleware` 已 import `internal/services` 与 `internal/services/system`（`apikey.go:9-10`），这两个包又 import `internal/models`（`apikey_service.go:10`）。故 `middleware → models` 的传递依赖**早已存在**，直接 import 不引入任何新环。原作者的"避免循环导入"注释系误判 `[VERIFIED]`。

## P0-1 死代码现状 (AUTH-02 背景)

| 中间件 | 挂载状态 | 证据 |
|--------|---------|------|
| `MultiAuth` | **未挂载** | `Grep router.go for "MultiAuth"` → No matches `[VERIFIED]` |
| `RequireScope` | **未挂载** | `Grep router.go` → No matches `[VERIFIED]` |
| `RequireAPIKeyResourcePermission` | **未挂载** | `Grep router.go` → No matches `[VERIFIED]` |
| `RateLimitByScope` | **未挂载** | `Grep router.go` → No matches `[VERIFIED]` |

`router.go:238-247` 显示 `/apikeys` 路由组用的是 `middleware.RequirePermissions`（JWT 路径的 RBAC），与 API Key 四中间件无关。**Phase 57 不挂载**（挂载属 Phase 60 AUTH-03 决策）；Phase 57 只让代码"正确且可挂载"，并用测试证明链路工作。

## 构造函数签名与 D-02 证据路径

SC#2 要求 `services.NewUsageLogger` / `services.NewRateLimiter` 有"真实实例化路径"。当前全仓 grep 结果：两者**仅在各自 `*_test.go` 内被实例化**，生产代码（router/main）零调用——这正是 P0-1 死代码的一部分。D-02 的解法：让 Phase 57 新增的集成测试文件**成为新的真实调用点**。

| 构造函数 | 签名 | 返回类型 | 装配目标 | D-02 证据写法 |
|---------|------|---------|---------|--------------|
| `services.NewUsageLogger` | `func (db *gorm.DB) UsageLogger` | `UsageLogger` (interface) | `MultiAuth(apiKeyService, usageLogger)` 第 2 参 | `db := setupUsageLoggerTestDB(t)`; `logger := services.NewUsageLogger(db)`; `MultiAuth(fakeSvc, logger)` |
| `services.NewRateLimiter` | `func () *RateLimiter` | `*RateLimiter` (具体类型, **无参**) | `RateLimitByScope(rateLimiter)` 第 1 参 | `rl := services.NewRateLimiter()`; `RateLimitByScope(rl)` |

**重要区分**：仓内存在**两个同名不同包**的 `NewRateLimiter`：
- `internal/services/rate_limiter.go:45` → `func NewRateLimiter() *RateLimiter`（无参，scope-based，**本 phase 目标**）
- `internal/services/operations/rate_limiter.go:28` → `func NewRateLimiter(maxTokens int, refillInterval time.Duration) *RateLimiter`（令牌桶，Baidu API 限流，**无关**）

`apikey.go:254` 的 `RateLimitByScope(rateLimiter *services.RateLimiter)` 取的是前者。证据实例化用 `services.NewRateLimiter()`（无参）`[VERIFIED: codebase read]`。

**setupUsageLoggerTestDB 既有模式** (`internal/services/usage_logger_test.go:30-57`)：用 `gorm.io/driver/sqlite` + `os.TempDir` 唯一文件名 + `busy_timeout=5000` + 手动 `CREATE TABLE sys_api_key_usage_logs`。D-02 证据可在 middleware 测试包内复制此 helper（或简化：`NewUsageLogger` 只需一个不 nil 的 `*gorm.DB` 即可证明签名兼容，LogUsage 是 fire-and-forget goroutine，测试不依赖其落库）。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 测试驱动中间件 | 手写 HTTP server | `net/http/httptest.NewRecorder` + `gin.New()` | 项目 18 个既有测试的约定；`httptest` 免真实端口 |
| fake `APIKeyService` | 全实现 9 方法逻辑 | 仅 `ValidateAPIKey` 给真值，其余 8 方法 `panic("not used")` | 集成测试只走 MultiAuth 路径，只调 `ValidateAPIKey`；其余方法实现是浪费且掩盖未预期调用 |
| 测试 DB (D-02 证据) | 起真实 PostgreSQL | `sqlite.Open(...)` (既有依赖) | `usage_logger_test.go` 既有模式；D-02 只需证明构造函数可调用 |

**Key insight:** 本 phase 的 fake 是"行为 stub"不是"完整 mock"。`ValidateAPIKey` 三条路径（返回有效 key / 返回 error / 触发 403）用同一 fake 的不同预置返回值切换即可。

## QUAL-02 测试规格

### fake 接口形状 (D-01)

**fake `system.APIKeyService`**——须满足接口全 9 方法（`apikey_service.go:29-39`），但只有 `ValidateAPIKey` 需要真逻辑：
```go
// Source: internal/services/system/apikey_service.go:29-39 (interface shape VERIFIED)
type fakeAPIKeyService struct {
	validKey   *models.APIKey   // 预置: 有效 key 时返回
	validateErr error           // 预置: 无效 key 时返回 (触发 401 路径)
}
func (f *fakeAPIKeyService) ValidateAPIKey(ctx context.Context, keyStr string) (*models.APIKey, error) {
	if f.validateErr != nil { return nil, f.validateErr }
	return f.validKey, nil
}
// 其余 8 方法: CreateAPIKey/ListAPIKeys/GetAPIKey/UpdateAPIKey/DeleteAPIKey/
// ToggleAPIKeyStatus/ListUsageLogs/GetUsageLogSummary → panic("not used in MultiAuth path")
```

**fake `services.UsageLogger`**——只需 1 方法（`usage_logger.go:12-17`）：
```go
// Source: internal/services/usage_logger.go:12-17 (interface shape VERIFIED)
type fakeUsageLogger struct{ logged bool }
func (f *fakeUsageLogger) LogUsage(ctx context.Context, req *services.LogUsageRequest) error {
	f.logged = true; return nil   // 记录被调用即可，不验证落库
}
```

### gin + httptest 装配骨架 (项目既有约定)

参考 `internal/api/v1/auth_integration_test.go:20-46` 的 `gin.SetMode(gin.TestMode)` + `httptest` 模式：
```go
// Source: internal/api/v1/auth_integration_test.go (pattern VERIFIED), adapted for middleware
func TestMultiAuthIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("有效key+正确scope_通过并写入context", func(t *testing.T) {
		// --- arrange ---
		uid := "11111111-2222-3333-4444-555555555555"
		fakeSvc := &fakeAPIKeyService{
			validKey: &models.APIKey{
				BaseModel: models.BaseModel{ID: "ak-test-id"},
				Name:      "test-key", UserID: &uid,
				Scopes: []string{"read"}, IsActive: true,
			},
		}
		fakeLogger := &fakeUsageLogger{}
		router := gin.New()
		router.Use(MultiAuth(fakeSvc, fakeLogger))           // 真实 MultiAuth
		router.GET("/ping", RequireScope("read"), func(c *gin.Context) {
			// SC#1: 断言 4 个 context 键被写入且非空
			assert.NotEmpty(t, c.GetString("user_id"))
			assert.Equal(t, "ak-test-id", c.GetString("api_key_id"))
			assert.Equal(t, []string{"read"}, c.MustGet("scopes"))
			assert.Equal(t, "api_key", c.GetString("auth_type"))
			c.JSON(200, gin.H{"ok": true})
		})

		// --- act ---
		req := httptest.NewRequest("GET", "/ping", nil)
		req.Header.Set("X-API-Key", "rec_"+hex64())          // 合法格式 68 字符
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// --- assert ---
		assert.Equal(t, 200, w.Code)
		assert.True(t, fakeLogger.logged)                     // 使用日志被触发
	})

	t.Run("有效key+缺失scope_403", func(t *testing.T) {
		fakeSvc := &fakeAPIKeyService{validKey: &models.APIKey{
			BaseModel: models.BaseModel{ID: "ak-2"},
			Scopes:    []string{"read"},   // 只有 read
		}}
		router := gin.New()
		router.Use(MultiAuth(fakeSvc, &fakeUsageLogger{}))
		router.GET("/write", RequireScope("write"), func(c *gin.Context) { c.JSON(200, nil) })
		// RequireScope("write") 因 scopes=["read"] 不含 write/admin → 403

		req := httptest.NewRequest("GET", "/write", nil)
		req.Header.Set("X-API-Key", "rec_"+hex64())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, 403, w.Code)                          // ErrForbidden.HTTPStatus
	})

	t.Run("无效key_401", func(t *testing.T) {
		fakeSvc := &fakeAPIKeyService{
			validateErr: errors.New("密钥不存在或已禁用"),
		}
		router := gin.New()
		router.Use(MultiAuth(fakeSvc, &fakeUsageLogger{}))
		router.GET("/any", func(c *gin.Context) { c.JSON(200, nil) })

		req := httptest.NewRequest("GET", "/any", nil)
		req.Header.Set("X-API-Key", "rec_"+hex64())
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, 401, w.Code)                          // ErrUnauthorized.HTTPStatus
	})
}

// hex64 返回合法的 64 位十六进制字符串 (满足 isValidKeyFormat: rec_ + 64 hex = 68 字符)
func hex64() string { return strings.Repeat("0123456789abcdef", 4) }  // 64 chars
```

### 三条路径断言清单 (SC#3)

| 路径 | 预置 | 期望 `w.Code` | 对应 `response` 错误 | 关键断言 |
|------|------|--------------|---------------------|---------|
| ① 有效+正确 scope | fake 返回 `Scopes:["read"]`，路由挂 `RequireScope("read")` | **200** | — | 4 context 键非空 + `fakeLogger.logged==true` |
| ② 有效+缺失 scope | fake 返回 `Scopes:["read"]`，路由挂 `RequireScope("write")` | **403** | `ErrForbidden` (HTTPStatus=403) | body 含"权限不足" |
| ③ 无效 key | fake 返回 `validateErr` | **401** | `ErrUnauthorized` (HTTPStatus=401) | body 含"密钥验证失败" |

**状态码对齐依据** (`pkg/response/response.go:40-41`)：
- `ErrUnauthorized = &AppError{Code: 401, HTTPStatus: 401}`
- `ErrForbidden = &AppError{Code: 403, HTTPStatus: 403}`

middleware 调用 `response.Error(c, response.ErrUnauthorized, ...)` (`apikey.go:34,42`) / `response.Error(c, response.ErrForbidden, ...)` (`apikey.go:51,189,198,213`)。`response.Error` 用 `appErr.HTTPStatus` 作 HTTP 状态码 (`response.go:95`) `[VERIFIED]`。

### SC#1 context 键断言 (4 键)
在路径① 的 handler 内直接断言（此时 context 已被修复后的 `setUserContextForAPIKey` 写入）：
```go
assert.NotEmpty(t, c.GetString("user_id"))              // *UserID 解引用后非空
assert.Equal(t, "ak-test-id", c.GetString("api_key_id"))// BaseModel.ID
assert.Equal(t, []string{"read"}, c.MustGet("scopes"))  // []string
assert.Equal(t, "api_key", c.GetString("auth_type"))    // 字面量
```

### D-02 证据测试 (同文件内额外用例)
```go
func TestConstructorsCallable_D02(t *testing.T) {
	// 证明 NewUsageLogger(db) 可调用且返回值满足 MultiAuth 第 2 参类型
	db := setupUsageLoggerTestDB(t)              // sqlite, 既有 helper 模式
	logger := services.NewUsageLogger(db)         // 真实构造函数
	assert.NotNil(t, logger)
	// 装配证明: MultiAuth 接受 (system.APIKeyService, services.UsageLogger)
	_ = MultiAuth(&fakeAPIKeyService{}, logger)  // 编译通过即签名兼容

	// 证明 NewRateLimiter() 可调用且返回值满足 RateLimitByScope 第 1 参类型
	rl := services.NewRateLimiter()              // 真实构造函数 (无参)
	assert.NotNil(t, rl)
	_ = RateLimitByScope(rl)                      // 编译通过即签名兼容
}
```
**注意**: `_ = MultiAuth(...)` / `_ = RateLimitByScope(...)` 赋值给 `_` 是为了证明"返回的 HandlerFunc 类型存在且可编译"——这本身不引入生产死代码，只是测试内的类型兼容性证明。

## Common Pitfalls

### Pitfall 1: 修复时顺手改 username 语义
**What goes wrong:** 看到 `username := ak.Name` 觉得"Name 是密钥名不是用户名，是 bug"，顺手改成查关联 User。
**Why it happens:** 代码看起来明显"错"。
**How to avoid:** D-04 锁定零行为变更。username 语义修正是 Phase 61 领域（需加载 User）。Phase 57 只动类型断言。
**Warning signs:** diff 里出现 `apiKey.User` 的访问（除既有 `InheritPerms && User != nil` 分支外）。

### Pitfall 2: fake 未实现全部 9 方法导致编译失败
**What goes wrong:** fake 只写 `ValidateAPIKey`，`system.APIKeyService` 接口未满足，`MultiAuth(fakeSvc, ...)` 编译报错。
**Why it happens:** 接口有 9 方法（`apikey_service.go:29-39`）。
**How to avoid:** fake 必须实现全部 9 方法；其余 8 方法写 `panic("not used in MultiAuth path")` 即可（编译需满足，运行不触发）。
**Warning signs:** `cannot use fakeSvc (type *fakeAPIKeyService) as type system.APIKeyService`。

### Pitfall 3: 集成测试用错误 X-API-Key 格式
**What goes wrong:** fake 预置返回有效 key，但请求头的 key 字符串格式非法（长度≠68 或非 hex），被 `isValidKeyFormat` 在 `ValidateAPIKey` 之前拦截 → 走 401 而非预期路径。
**Why it happens:** `MultiAuth` 先校验格式（`apikey.go:33-37`）再调 service。
**How to avoid:** 测试请求头用 `"rec_" + 64位hex`（68 字符）。helper `hex64()` 返回 `strings.Repeat("0123456789abcdef", 4)`。
**Warning signs:** 路径①/② 实际返回 401 而非 200/403。

### Pitfall 4: D-02 证据误用 operations 包的 NewRateLimiter
**What goes wrong:** 写 `operations.NewRateLimiter(...)` 装配 `RateLimitByScope`，签名不匹配（两个不同 `*RateLimiter` 类型）。
**Why it happens:** 仓内两个同名构造函数。
**How to avoid:** 用 `services.NewRateLimiter()`（无参，`internal/services/rate_limiter.go:45`）。
**Warning signs:** `cannot use (type *operations.RateLimiter) as type *services.RateLimiter`。

### Pitfall 5: 改 RequireAPIKeyResourcePermission 时动了 resource 参数
**What goes wrong:** 顺手实现 resource→permission 映射。
**Why it happens:** 参数明摆着没用。
**How to avoid:** D-03 显式延后 resource 逻辑到 Phase 61 AUTH-04。Phase 57 只改调用路径反模式。
**Warning signs:** diff 里出现 resource 相关 map 或条件。

## Code Examples

### response.Error 的 HTTP 状态码映射 (测试断言基准)
```go
// Source: pkg/response/response.go:40-41,87-102 (VERIFIED by direct read)
// ErrUnauthorized → Code:401, HTTPStatus:401
// ErrForbidden    → Code:403, HTTPStatus:403
// response.Error(c, err) 用 appErr.HTTPStatus 作为 HTTP 状态码 (response.go:95)
// 故 middleware 的 response.Error(c, response.ErrUnauthorized, ...) → HTTP 401
```

### BaseModel.ID 是 UUID string
```go
// Source: internal/models/base.go:11-19 (VERIFIED)
type BaseModel struct {
	ID string `gorm:"type:uuid;primary_key" json:"id"`   // ← api_key_id 取此字段
	...
}
// BeforeCreate 钩子: ID 为空时 uuid.New().String() 自动生成
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `interface{}` + 局部 mirror struct "避免循环导入" | 直接 import 真实类型 | Phase 57 (本次) | 类型断言由编译器保证；消除 P0-2 恒 false |
| `RequireScope(requiredScope)(c)` 内联调用 | 注册期委托 `return RequireScope(scope)` | Phase 57 (本次) | c.Next() 处于正确链位置；消除重复执行风险 |
| API Key 明文存储 (`WHERE key = ?`) | (未变, Phase 60 SEC-01 决策) | — | 本 phase 不碰 |

**Deprecated/outdated:**
- `apikey.go:145` 注释 "使用 interface{} 避免循环导入" — 误判，本 phase 删除。
- `apikey.go:154` 局部 `apiKeyType` struct — 删除。

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | gin `c.Next()` 在直接函数调用（非链挂载）中会推进全局链索引导致重复执行 | Pattern 2 | LOW — 标准 gin 行为，多源佐证；即便影响有限，重写后 provably correct，无回归风险 |
| A2 | fake 其余 8 方法写 `panic` 不会影响测试（因为 MultiAuth 路径不调它们） | QUAL-02 | LOW — `apikey.go:22-77` MultiAuth 体内只调 `ValidateAPIKey`，已逐行确认 |

**说明:** 其余所有claim 均 `[VERIFIED: codebase]`（直接读源码）或 `[CITED]`（引用 CONTEXT.md 锁定决策）。本 phase 无需 `[ASSUMED]` 标记的库/API claim——所有库（gin/testify/gorm）均为项目既有依赖且用法已在 18 个测试文件中印证。

## Open Questions

1. **`RequireAPIKeyResourcePermission` 重写选哪种写法?**
   - What we know: 两种都满足 D-03（最小委托 vs 显式 helper 内联）。
   - What's unclear: Phase 61 是否需要 `hasRequiredScope` helper 复用。
   - Recommendation: 用最小委托写法（`return RequireScope(getRequiredScope(action))`）；Phase 61 若需 helper 再提取。CONTEXT.md 已把此裁量留给 Claude。

2. **D-02 的 `NewUsageLogger(db)` 是否需完整 setupUsageLoggerTestDB?**
   - What we know: `NewUsageLogger` 只把 db 存进 struct，不立即用。
   - What's unclear: 测试是否要建表。
   - Recommendation: 复制既有 helper 建表最安全（与 `usage_logger_test.go` 一致）；但若只证明"构造函数可调用 + 类型兼容"，一个 `sqlite.Open` 出来的不 nil `*gorm.DB` 即足够（LogUsage 是 fire-and-forget goroutine，测试结束前不一定执行）。

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | `go build` / `go test` | ✓ | go1.24 (CLAUDE.md) | — |
| `gorm.io/driver/sqlite` | D-02 证据 `NewUsageLogger(db)` | ✓ | 既有 go.mod | — |
| `net/http/httptest` (stdlib) | 集成测试 | ✓ | stdlib | — |
| PostgreSQL | **不需要** | — | — | fake 规避 DB；D-02 用 sqlite |
| Redis | **不需要** | — | — | 本 phase 不涉及缓存 |

**Missing dependencies with no fallback:** 无。
**Missing dependencies with fallback:** 无。

Step 2.6 (Environment Audit): 全部依赖可用，无阻塞项。PostgreSQL/Redis 本 phase 不需要（D-01 fake 规避 DB，D-02 用既有 sqlite）。

## Validation Architecture

> `workflow.nyquist_validation: true`（config.json 第 19 行）——本节为必需，由下游生成 VALIDATION.md 消费。

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` + testify v1.11.1 (`assert`/`require`) |
| Config file | 无独立配置文件；`go test` 直接驱动，`gin.SetMode(gin.TestMode)` 在测试内设置 |
| Quick run command | `go test ./internal/middleware/ -run "TestMultiAuth|TestIsValidKeyFormat|TestIsIPAllowed|TestGetRequiredScope" -v` |
| Full suite command | `go test ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| AUTH-01 | `setUserContextForAPIKey` 写入 4 个 context 键 (P0-2 修复后) | integration | `go test ./internal/middleware/ -run TestMultiAuthIntegration/有效key -v` | ❌ Wave 0 新建 |
| AUTH-02 | 4 中间件类型签名自洽 + 构造函数可装配 (D-02) | integration | `go test ./internal/middleware/ -run TestConstructorsCallable_D02 -v` | ❌ Wave 0 新建 |
| AUTH-02 | `RequireAPIKeyResourcePermission` 重写后不靠 c.Next() 副作用 | unit (编译即证明 + 集成路径覆盖) | `go vet ./internal/middleware/` + `go build ./...` | ❌ Wave 0 (编译 gate) |
| QUAL-02 | 有效 key+正确 scope → 200 | integration | `go test ./internal/middleware/ -run TestMultiAuthIntegration/有效key -v` | ❌ Wave 0 新建 |
| QUAL-02 | 有效 key+缺失 scope → 403 | integration | `go test ./internal/middleware/ -run TestMultiAuthIntegration/有效key_缺失scope -v` | ❌ Wave 0 新建 |
| QUAL-02 | 无效 key → 401 | integration | `go test ./internal/middleware/ -run TestMultiAuthIntegration/无效key -v` | ❌ Wave 0 新建 |
| QUAL-02 | 既有 3 纯函数测试不回归 | unit (既有) | `go test ./internal/middleware/ -run "TestIsValidKeyFormat|TestIsIPAllowed|TestGetRequiredScope" -v` | ✅ 既有 `apikey_test.go` |

### 回归网 (Regression Net) — SC#1~4 验证契约
| SC# | 验证手段 | 命令 / 断言 | 通过标准 |
|-----|---------|-----------|---------|
| SC#1 | 集成测试路径① | handler 内断言 4 context 键非空 | `user_id`/`api_key_id`/`scopes`/`auth_type="api_key"` 全部断言通过 |
| SC#2 | grep 证据 + 编译 + D-02 测试 | `grep -rn "NewUsageLogger\|NewRateLimiter()" internal/` + `TestConstructorsCallable_D02` | 构造函数有真实调用点（测试文件内），类型兼容编译通过 |
| SC#3 | 三路径集成测试 | 见上表 QUAL-02 三行 | 200 / 403 / 401 全部命中 |
| SC#4 | 全量 build + grep | `go build ./...` + `grep -rn "避免循环导入\|interface{}" internal/middleware/apikey.go` | build 退出 0；无 workaround 注释残留 |

### 非 CI 门禁项 (人工审查，SC#2 一部分)
| 审查项 | 方法 | 期望 |
|--------|------|------|
| `RequireAPIKeyResourcePermission` 反模式消除 | 读 `apikey.go` diff | 不再有 `RequireScope(...)(c)` 内联调用 |
| 4 中间件类型签名一致 | 读 `apikey.go` | `MultiAuth(system.APIKeyService, services.UsageLogger)` / `RequireScope(string)` / `RequireAPIKeyResourcePermission(string, string)` / `RateLimitByScope(*services.RateLimiter)` |
| `resource` 参数仍忽略 / `getScopeFromContext` 仍取 scopes[0] | 读 diff 确认**未动** | 这两项延后 Phase 61，本 phase 不碰 |

### Sampling Rate
- **Per task commit:** `go build ./... && go test ./internal/middleware/ -v`
- **Per wave merge:** `go test ./...`（全量，含既有 `apikey_test.go` 3 纯函数 + `apikey_service_test.go` + `usage_logger_test.go` + `rate_limiter_test.go` 不回归）
- **Phase gate:** `go build ./...` 退出 0 + `go test ./...` 全绿 + grep 证据（无 workaround 残留 + 构造函数有调用点）后才可 `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/middleware/apikey_integration_test.go` — 新建，覆盖 AUTH-01/AUTH-02/QUAL-02 集成测试 + D-02 证据用例
- [ ] fake 类型定义 (`fakeAPIKeyService` 9 方法 + `fakeUsageLogger` 1 方法) — 同文件内
- [ ] `setupUsageLoggerTestDB` helper — 从 `internal/services/usage_logger_test.go` 模式复制（或简化版，仅需不 nil `*gorm.DB`）
- [ ] 源码修复: `internal/middleware/apikey.go` — P0-2 签名 + import + P0-1 反模式重写

*(既有 `apikey_test.go` 3 纯函数测试已覆盖 `isValidKeyFormat`/`isIPAllowed`/`getRequiredScope`，不回归即可)*

## Security Domain

> `security_enforcement` 未在 config.json 显式设 false → 视为启用。本 phase 是认证链修复，与 ASVS V2/V5 直接相关。

### Applicable ASVS Categories
| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | API Key 是 JWT 之外的第二认证通道；`ValidateAPIKey` (service) + `MultiAuth` (middleware) 构成认证链。Phase 57 修复的是"认证成功后上下文丢失"（P0-2），属认证功能正确性 |
| V3 Session Management | no | API Key 无 session（无状态令牌）；gin context 是每请求态，非会话态 |
| V4 Access Control | partial | `RequireScope` 是作用域级访问控制；资源级 (`resource` 参数) 延后 Phase 61 AUTH-04。Phase 57 保证作用域校验链路可达 |
| V5 Input Validation | yes | `isValidKeyFormat` (68 字符 + hex 前缀校验) + IP 白名单 `isIPAllowed` (CIDR 解析) 是既有输入校验，本 phase 不改但测试覆盖其生效 |
| V6 Cryptography | no | 本 phase 不涉及加解密（SM2/SM4 在请求加密层，与 API Key 认证链无关） |

### Known Threat Patterns for 认证链
| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| 认证后上下文丢失 (P0-2) | Elevation of Privilege | 修复类型断言 → 上下文真实写入；集成测试锁住防回归 |
| 中间件死代码 (P0-1) | 信息泄露 / 认证绕过 (若错误挂载) | Phase 57 让代码自洽可挂载；Phase 60 AUTH-03 决策是否启用 + 安全评估 |
| IP 白名单绕过 | Spoofing | `isIPAllowed` 既有 CIDR 校验（本 phase 不改，测试覆盖） |

**本 phase 安全影响范围:** 修复认证链功能正确性，**不改变安全策略**（不启用挂载、不改存储、不改加密）。启用决策与威胁建模归 Phase 60 AUTH-03。

## Sources

### Primary (HIGH confidence)
- `internal/middleware/apikey.go` — 逐行读取，确认 P0-2 (line 146-179) + P0-1 反模式 (line 223-231) + 4 中间件签名
- `internal/services/system/apikey_service.go` — 确认 `APIKeyService` 接口 9 方法 (line 29-39) + `ValidateAPIKey` 返回 `*models.APIKey` (line 129,160)
- `internal/models/api_key.go` + `internal/models/base.go` — 确认 `APIKey` struct 形状 + `BaseModel.ID` (UUID string) + models 包仅 import time/uuid/gorm
- `internal/services/usage_logger.go` — 确认 `UsageLogger` 接口 (1 方法) + `NewUsageLogger(db) UsageLogger` 签名
- `internal/services/rate_limiter.go` — 确认 `NewRateLimiter() *RateLimiter` (无参) + `RateLimitByScope` 入参类型
- `pkg/response/response.go` — 确认 `ErrUnauthorized`(401) / `ErrForbidden`(403) 的 HTTPStatus
- `internal/api/router.go` — grep 确认 4 中间件零挂载 (P0-1 死代码)
- `internal/api/v1/auth_integration_test.go` — 确认项目 gin+httptest 既有约定
- `internal/services/usage_logger_test.go` — 确认 `setupUsageLoggerTestDB` sqlite 测试 DB 模式
- `internal/middleware/apikey_test.go` — 确认既有 3 纯函数测试 (不回归基准)
- `internal/services/system/apikey_service_test.go:402` — 确认 testify+real-DB 既有风格
- `go build ./...` 退出 0 — baseline green 确认

### Secondary (MEDIUM confidence)
- `.planning/codebase/TESTING.md` — "无 gomock、需 DB 用真实连接" 约定 (D-01 fake 策略依据)
- `.planning/STATE.md` §根因调查结论 — P0-1/P0-2 ground-truth 表 (本研究逐项复核一致)

### Tertiary (LOW confidence)
- 无。本 phase 所有 claim 均由 codebase 直接读取或 CONTEXT.md 锁定决策支撑。

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — 全部为既有依赖，零新增包，18 个测试文件印证用法
- Architecture (P0-2 fix): HIGH — 逐行读源码确认根因链 + 双向 grep 排除循环依赖
- Architecture (P0-1 refactor): HIGH — 标准 gin 中间件反模式，两种重写均 provably correct
- Pitfalls: HIGH — 均来自 codebase 实际形状（接口 9 方法、双 NewRateLimiter、格式校验前置）
- Test strategy: HIGH — 项目既有 gin+httptest 模式直接复用

**Research date:** 2026-08-13
**Valid until:** 2026-09-12 (stable — 本 phase 是内部 codebase 修复，不依赖外部库版本演进)
