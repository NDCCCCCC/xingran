# Phase 38: AD 账号池统一（移除遗留单管理员双轨）- Context

**Gathered:** 2026-06-23
**Status:** Ready for planning

<domain>
## Phase Boundary

删除 `sys_ad_config` 单管理员遗留路径（`admin_username` / `admin_password`，已标 @Deprecated），
将所有 AD 连接（同步 / 管理 / 登录认证）统一收敛到 Phase 36 账号池 `FailoverClient`。
这是 Phase 36 CONTEXT 锁定的"严格迁移期（双读兼容）" + "老字段保留标 @Deprecated，1 版本后清理"
中**"1 版本后清理"步骤的执行**。

**In scope:**
- ~15 处 `NewLDAPClient(config)`（不带 account）的同步/管理调用点改走 `FailoverClient.ExecuteWithFailover`
- 移除单管理员密码的 Go 使用逻辑（`decryptPassword(config.AdminPassword)` 9 处调用点）
- 前端 AD 配置页移除 admin 账号输入项，账号管理收敛到 `AccountPoolTab`
- migration_162 补迁逻辑：保证现有单管理员账号已入账号池
- 启动时空池校验 + 登录/同步空池明确错误引导

**Out of scope:**
- 账号池本身的功能增强（熔断策略、优先级字段等——Phase 36 已锁定）
- DROP COLUMN 破坏性 DDL（本 phase 保留空列，仅删代码）
- AD 域控 OU/组/用户同步的业务逻辑变更（仅改连接获取方式）

</domain>

<decisions>
## Implementation Decisions

### 执行策略
- **D-01:** 分波次降风险迁移（非一次性全量）：
  - **Wave 1 — 连接层统一**：所有 `NewLDAPClient(config)` 调用点改走 `FailoverClient.ExecuteWithFailover(operation)`，原同步逻辑封装进 `operation func(client *LDAPClient) error` 闭包。此波保留单管理员字段作回退兜底，每波可独立 `go build ./...` + `go test` + 回滚。
  - **Wave 2 — 移除使用代码 + 前端**：删除单管理员密码的 Go 使用逻辑（9 处 `decryptPassword(config.AdminPassword)`）+ 前端 AD 配置页移除 admin 输入项。
  - **Wave 3 — 字段清理**：model struct 清理（移除使用/tag）+ migration_162 补迁校验。

### 字段去留
- **D-02:** 保留空列仅删代码。`sys_ad_config.admin_username` / `admin_password` DB 列**保留为空**（不做 DROP COLUMN），仅删除 Go 代码中的使用逻辑。理由：PostgreSQL DROP COLUMN 是破坏性 DDL（锁表 + 重写 + 不可逆），保留空列回滚仅需恢复代码，符合 @Deprecated 渐进语义。

### 数据迁移校验（安全边界）
- **D-03:** 启动空池校验 + 明确错误，**不静默 fallback**：
  - 应用启动时检查：启用的 AD config 对应账号池为空 → 记 WARN 日志（**不阻断启动**，避免阻塞新环境首次部署）。
  - 登录/同步运行时遇空池 → 返回**明确错误**"请先在 AD 配置页添加服务账号"，引导用户配置，绝不静默失败。
  - migration_162 补迁逻辑：保证现有 `sys_ad_config` 单管理员账号已迁入账号池（幂等，已迁则跳过）。
  - **不保留单管理员作 fallback**（与 D-02 删代码一致，避免重新引入双轨复杂度）。

### 前端配置页
- **D-04:** 移除 admin 账号输入项。AD 配置表单（`pages/system/ad-domain/config`）移除 `admin_username` / `admin_password` 输入框，账号管理统一收敛到 `AccountPoolTab.tsx`。配合后端删代码，避免用户误填已废弃字段。`lib/api/system.ts` 同步移除相关 API 字段。

### Claude's Discretion
- Wave 内部的具体波次划分顺序、`operation` 闭包封装风格、model struct tag 清理粒度——由 planner 根据代码依赖图决定。
- 测试策略：`TestAccountPoolPasswordRoundTrip` 已守护账号池解密契约；同步改 FailoverClient 后的回归测试设计由 planner + executor 决定。
- operlog 记录：`ad_account_handler` 现有 operlog 记录保持不变（账号池 CRUD 已覆盖）。
- 配置测试连接（`config.go:208` `NewLDAPClient(config)`）：改为用账号池首个可用账号测试，具体由 planner 决定。

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 账号池设计来源（Phase 36）
- `.planning/phases/36-ad-account-pool-failover/CONTEXT.md` — 账号池原始设计，锁定"严格迁移期（双读兼容）" + "1 版本后清理"（本 phase 的依据）
- `.planning/phases/36-ad-account-pool-failover/CONTEXT.md` — AccountPool 接口、表结构、API 端点、OperLog 映射、manualUnlock 审计
- `.planning/phases/36-ad-account-pool-failover/PLAN.md` — 账号池实现细节（FailoverClient、熔断、pub/sub 热加载）
- `CLAUDE.md` §"AD Service Account Pool (Phase 36)" — 架构约定（AccountPool/FailoverClient/LDAPClient/API 设计要点）

### Bug 背景（本次清理的直接动因）
- `.planning/debug/adpool-password-not-decrypted.md` — Phase 36 账号池 `tryBindAttempts` 密文未解密 bug 的根因与修复（双轨不对称风险的实证）

### 参考实现（已走账号池的路径，作为改造模板）
- `internal/services/addomain/failover_client.go` — `ExecuteWithFailover` / `PickFirstConnect`，Wave 1 改造的目标 API
- `internal/core/security/ad_authenticator.go:270` — 登录认证已用 `NewFailoverClient` + `PickFirstConnect`，是同步任务改造的参考样板
- `internal/services/addomain/ldap_client.go` `tryBindAttempts`（已修复，内部 `decryptPassword`）— 账号池密码解密的正确实现

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`FailoverClient.ExecuteWithFailover(ctx, operation func(*LDAPClient) error)`** (`failover_client.go`)：Wave 1 改造的核心 API。同步任务的现有逻辑（Search/Move/Update）直接放入 `operation` 闭包，FailoverClient 自动处理账号遍历 + 成功/失败上报。
- **`FailoverClient.PickFirstConnect(ctx)`**：只建连接不做后续操作的场景（如配置测试连接）。
- **`AccountPool` 接口** (`account_pool.go`)：`ListAvailable` / `Create` / `MarkSuccess` / `MarkFailure` 等，已就绪。
- **`decryptPassword` / `SetADSM4Cipher`** (`utils.go`)：账号池密码加解密，`tryBindAttempts` 已内部调用，同步任务改 FailoverClient 后无需再手动解密。

### Established Patterns
- **单管理员解密模式（改造对照，待移除）**：9 处 caller 在使用前 `config.AdminPassword = decryptPassword(config.AdminPassword)`——改走 FailoverClient 后这些手动解密全部删除。
- **`NewLDAPClient(config, account ...*ADServiceAccount)` variadic**：向后兼容设计，FailoverClient 内部已带 account 调用；同步任务改造后统一经 FailoverClient。
- **operlog `password_ciphertext` 自动脱敏**：字段名含 `password` 关键词，operlog 自动脱敏（OPERLOG-03 兼容），无需改敏感词列表。

### Integration Points（Wave 1 待迁移的 `NewLDAPClient(config)` 调用点）
- `internal/services/addomain/sync.go:93`
- `internal/services/addomain/user.go:126, 195, 216, 237`
- `internal/services/addomain/group_sync_service.go:55, 93`
- `internal/services/addomain/group_management_service.go:86, 154, 202, 260`
- `internal/services/addomain/group.go:125, 155, 187`
- `internal/services/addomain/dept_sync_service.go:46`
- `internal/scheduler/dept_sync_tasks.go:89, 163`
- `internal/services/addomain/user_ad_sync_service.go:62, 207, 476`（476 为测试钩子分支）
- `internal/services/addomain/config.go:208`（配置测试连接）

### 单管理员待清理（Wave 2）
- `internal/models/ad_domain.go:32` — `AdminPassword` / `AdminUsername` 字段（DB 列保留，清理 struct 使用）
- `internal/services/addomain/config.go` — Create/Update 里 `encryptPassword(req.AdminPassword)`（108/169 行）
- 前端：`pages/system/ad-domain/config`（admin 输入项）+ `lib/api/system.ts`（adminPassword 字段）

</code_context>

<specifics>
## Specific Ideas

无特殊定制——遵循 Phase 36 已锁定的账号池设计，本 phase 仅做收敛清理。回归守护复用 Phase 36 修复时新增的 `TestAccountPoolPasswordRoundTrip`（账号池加解密闭环契约）。

</specifics>

<deferred>
## Deferred Ideas

- **DROP COLUMN 彻底清理**：D-02 决定本 phase 保留空列。若后续版本确认无回滚需求，可在新 phase 做物理 DROP COLUMN（需评估 PG 表大小与锁表窗口）。
- **同步任务 failover 行为精细化**：当前 FailoverClient 失败即 MarkFailure（写库 + 可能熔断）。批量同步（如 SyncManagersToAD 遍历多用户）是否需要区分"账号失败"vs"单用户操作失败"的上报粒度——属账号池功能增强，留给后续 phase。
- **账号池为空的引导式 UI**：D-03 返回明确错误文本引导配置；更友好的前端引导（如空池时配置页高亮账号池 tab）可作为 UX 增强单独评估。

</deferred>

---

*Phase: 38-unify-ad-authentication-onto-account-pool-remove-legacy-sing*
*Context gathered: 2026-06-23*
