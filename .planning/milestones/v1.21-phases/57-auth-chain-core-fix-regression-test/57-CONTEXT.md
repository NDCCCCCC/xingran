# Phase 57: 认证链核心修复 + 回归测试 - Context

**Gathered:** 2026-08-13
**Status:** Ready for planning

<domain>
## Phase Boundary

让 API Key 认证链**代码功能正确**：修复 `setUserContextForAPIKey` 类型断言恒 false（P0-2 / AUTH-01），消除 `MultiAuth` 下游死代码使 4 个中间件经审查自洽、具备被挂载的条件（P0-1 / AUTH-02），并用集成测试锁住"API Key 认证 → 上下文写入 → 作用域校验"完整链路防回归（QUAL-02）。

**本 phase 不在生产路由挂载 MultiAuth**——挂载启用是 Phase 60 AUTH-03 的 discuss 决策点。本 phase 只让代码"正确且可挂载"，并用测试证明链路工作。

**Requirements:** AUTH-01, AUTH-02, QUAL-02

</domain>

<decisions>
## Implementation Decisions

### 测试替身策略（QUAL-02）
- **D-01:** 集成测试用**手写 fake/stub** 实现 `system.APIKeyService` + `services.UsageLogger`（fake 返回构造好的 `*models.APIKey`），通过**真实 `gin.Engine` + `httptest`** 跑 `MultiAuth` 中间件，断言 gin context 键被写入。无 DB 依赖、无 mock 框架，符合 TESTING.md 既有 testify 风格（"无 gomock"）。覆盖 SC#3 三条路径：① 有效 key+正确 scope→通过；② 有效 key+缺 scope→403；③ 无效 key→401。

### 就绪可挂载证据（AUTH-02 / SC#2）
- **D-02:** SC#2 要求 `services.NewUsageLogger` / `services.NewRateLimiter` 构造函数有"真实实例化路径"。**注意 fake UsageLogger ≠ NewUsageLogger**。证据方式：集成测试文件**额外实例化真实的 `NewUsageLogger(db)` / `NewRateLimiter()`**（用测试 DB 或 sqlite），证明构造函数可调用且类型签名与 MultiAuth 装配兼容。测试即真实调用点，不引入生产死代码。实际生产装配推迟 Phase 60。

### AUTH-02 审查深度（中间件自洽）
- **D-03:** 4 个中间件（`MultiAuth`/`RequireScope`/`RequireAPIKeyResourcePermission`/`RateLimitByScope`）做到**内部自洽、无静默缺陷**：
  - 修**类型签名 + 调用路径反模式**：`RequireAPIKeyResourcePermission` 内联 `RequireScope(requiredScope)(c)` 改为正确的链式组合/直接逻辑（成功路径靠 `c.Next()` 副作用推进属脆弱写法）。
  - `resource` 参数被忽略、`getScopeFromContext` 只取 `scopes[0]`——这两项**不在 Phase 57 修**，已升级为 v1 需求 `AUTH-04` / `QUAL-03` 归 **Phase 61**（见下方重规划说明，2026-08-13 用户决策）。

### 上下文键与 username 语义（AUTH-01）
- **D-04:** **保留现有行为**。`setUserContextForAPIKey` 仅修类型断言（签名 `interface{}` → `*models.APIKey`，直接 import `internal/models`，循环依赖已排除）。保留全部 7 个 context 键（`user_id`/`username`/`nickname`/`api_key_id`/`scopes`/`auth_type`/`inherit_perms`）。`username := ak.Name`（line 159，Name 是 key 名而非用户名）**语义原样保留**——零行为变更，避免破坏下游读 `username` 的 handler。username 语义修正属 Phase 61 资源权限领域。

### AUTH-01 修复方向（locked，承自 REQUIREMENTS）
- **D-05:** `setUserContextForAPIKey(c, apiKey interface{}, scopes)` → 签名改为 `(c, apiKey *models.APIKey, scopes)`，直接 `import "github.com/xingran-next/xingran-go-backend/internal/models"`，移除局部 `apiKeyType` 与 `interface{}` workaround。**已验证 `internal/models` 不导入 `internal/middleware`，无循环依赖**——`interface{}` 系原作者误判。`ValidateAPIKey` 返回 `*models.APIKey`（apikey_service.go:129），调用方 apikey.go:58 传入指针，断言到局部值类型恒 false 是 P0-2 根因。

### Claude's Discretion
- 测试文件命名与组织（建议 `internal/middleware/apikey_integration_test.go`，与既有 `apikey_test.go` 3 个纯函数测试并存）。
- fake 实现的具体字段值构造（只要覆盖三条路径断言）。
- 内联 `RequireScope()(c)` 重构的具体写法（只要消除"靠 c.Next() 副作用推进"的脆弱性）。

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 需求与规划
- `.planning/ROADMAP.md` §Phase 57 — Goal / Depends on / Requirements / Success Criteria (SC#1-4)
- `.planning/REQUIREMENTS.md` — AUTH-01 / AUTH-02 / QUAL-02 定义
- `.planning/STATE.md` §根因调查结论 — P0-1/P0-2 ground-truth 表（文件:行 + 根因）

### 待修复源码（核心）
- `internal/middleware/apikey.go` — P0-2（line 146-179 `setUserContextForAPIKey`）/ P0-1（4 中间件死代码）/ P2-a（line 274-275 限流头编码，属 Phase 60）/ P1-2+P2-b（line 60-75 使用日志，属 Phase 59）
- `internal/services/system/apikey_service.go:129` — `ValidateAPIKey` 返回 `*models.APIKey`（决定 AUTH-01 签名）
- `internal/models/api_key.go:8` — `APIKey` struct 定义（ID/Name/UserID/InheritPerms/User/Scopes/IPWhitelist）
- `internal/api/router.go` — MultiAuth 当前未挂载的上下文（挂载属 Phase 60 AUTH-03）

### 既有测试基建
- `internal/services/system/apikey_service_test.go:402` — `TestValidateAPIKey`（服务层既有测试，testify 风格参考）
- `internal/middleware/apikey_test.go` — 3 个纯函数测试（isValidKeyFormat / isIPAllowed / getRequiredScope），本 phase 新增集成测试与之并存，**不回归**
- `.planning/codebase/TESTING.md` — "无 gomock、需 DB 用真实连接"约定（D-01 依据）
- `.planning/codebase/ARCHITECTURE.md` — Handler-Service 分层 + Core DI

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `system.APIKeyService` 接口 + `services.UsageLogger` 接口：fake 实现只需满足 `ValidateAPIKey(ctx, keyStr) (*models.APIKey, error)` 与 `LogUsage(ctx, *LogUsageRequest)`。
- `pkg/response`（`response.ErrUnauthorized`/`ErrForbidden`/`Error`）：中间件已用，测试断言状态码对齐。
- testify `assert`（既有约定）：context 键断言 `assert.Equal(t, "api_key", authType)` 等。

### Established Patterns
- 中间件位于 `internal/middleware/`（非 `pkg/middleware/`，ARCHITECTURE.md 图有偏差，以实际为准）。
- 测试紧邻源码 `*_test.go`；无 mock 框架；DB 测试用真实连接（本 phase fake 规避 DB）。
- gin context 键约定：`user_id`/`username`/`api_key_id`/`scopes`/`auth_type`（JWT 与 API Key 路径共享，D-04 保留全部键避免破坏下游）。

### Integration Points
- `MultiAuth(apiKeyService, usageLogger)` 是挂载点签名——测试装配即证明签名可用（D-02）。
- `cmd/main.go` → `internal/api/router.go` 是 Phase 60 实际挂载位置（本 phase 不动）。

</code_context>

<specifics>
## Specific Ideas

- 用户在 Area 3 连续两次扩大选择（"修全部逻辑瑕疵" → "含完整 FUTURE 功能"），触发 milestone 重规划（见下方）。最终落定：Phase 57 只做中间件**自洽**（D-03），完整资源权限矩阵 + 限流调优独立为 Phase 61。这反映用户希望 API Key 系统"真正可用"的产品取向——downstream planner 应理解 Phase 57 是"让链路代码正确"，Phase 61 才是"让能力完整"。

</specifics>

<deferred>
## Deferred Ideas

- **资源级细粒度权限矩阵**（`RequireAPIKeyResourcePermission` 的 resource→permission 映射 + InheritPerms 资源校验）→ **Phase 61 / AUTH-04**（ex-FUTURE-APIKEY-01，conditional on Phase 60 AUTH-03=启用）。2026-08-13 重规划升级为 v1。
- **限流生产接入与调优**（`RateLimitByScope` 多 scope 选择逻辑、生产路由全量接入）→ **Phase 61 / QUAL-03**（ex-FUTURE-APIKEY-02，conditional on Phase 60 AUTH-03=启用）。
- **username 语义修正**（`username=ak.Name` → 取关联 User 真名）→ Phase 61 资源权限领域（需加载 User）。
- **密钥轮换/吊销、配额告警** → FUTURE-APIKEY-03/04（仍 v2 Future）。

### Milestone 重规划说明（2026-08-13，本 session 内完成）
用户决策把 FUTURE-APIKEY-01/02 拉入 v1.21 milestone。已落地：
- 新增 **Phase 61: 资源级权限矩阵 + 限流生产调优**（depends on Phase 60 AUTH-03=启用）。
- `FUTURE-APIKEY-01` → v1 `AUTH-04`；`FUTURE-APIKEY-02` → v1 `QUAL-03`（均归 Phase 61）。
- milestone 4→5 phases（57-61），需求 11→13。ROADMAP/REQUIREMENTS/STATE 已同步并 commit（`0d599e9`）。
- **Phase 57 边界不变**——重规划只影响 FUTURE 项落点，不影响 Phase 57 的 AUTH-01/02/QUAL-02。

### Reviewed Todos (not folded)
- `operlog-exclude-paths.md`（todo.match-phase 得分 0.2，关键词"phase"误匹配）——与 Phase 57 无真实关联，不 fold。

</deferred>

---

*Phase: 57-认证链核心修复 + 回归测试*
*Context gathered: 2026-08-13*
