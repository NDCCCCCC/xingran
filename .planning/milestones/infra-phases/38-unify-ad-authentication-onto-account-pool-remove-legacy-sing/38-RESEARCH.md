# Phase 38: AD 账号池统一（移除遗留单管理员双轨）- Research

**Researched:** 2026-06-23
**Domain:** AD/LDAP 连接获取层重构 + 数据迁移补全 + 前端字段清理
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** 分波次降风险迁移（非一次性全量）：
  - **Wave 1 — 连接层统一**：所有 `NewLDAPClient(config)` 调用点改走 `FailoverClient.ExecuteWithFailover(operation)`，原同步逻辑封装进 `operation func(client *LDAPClient) error` 闭包。此波保留单管理员字段作回退兜底，每波可独立 `go build ./...` + `go test` + 回滚。
  - **Wave 2 — 移除使用代码 + 前端**：删除单管理员密码的 Go 使用逻辑（9 处 `decryptPassword(config.AdminPassword)`）+ 前端 AD 配置页移除 admin 输入项。
  - **Wave 3 — 字段清理**：model struct 清理（移除使用/tag）+ migration_162 补迁校验。
- **D-02:** 保留空列仅删代码。`sys_ad_config.admin_username` / `admin_password` DB 列保留为空（不做 DROP COLUMN），仅删除 Go 代码中的使用逻辑。理由：PostgreSQL DROP COLUMN 破坏性 DDL（锁表 + 重写 + 不可逆），保留空列回滚仅需恢复代码。
- **D-03:** 启动空池校验 + 明确错误，**不静默 fallback**：
  - 应用启动时检查：启用的 AD config 对应账号池为空 → 记 WARN 日志（**不阻断启动**，避免阻塞新环境首次部署）。
  - 登录/同步运行时遇空池 → 返回**明确错误**"请先在 AD 配置页添加服务账号"，绝不静默失败。
  - migration_162 补迁逻辑：保证现有 `sys_ad_config` 单管理员账号已迁入账号池（幂等，已迁则跳过）。
  - **不保留单管理员作 fallback**（与 D-02 删代码一致）。
- **D-04:** 移除 admin 账号输入项。AD 配置表单（`pages/system/ad-domain/config`，实际路径 `pages/ad-domain/configs/index.tsx`）移除 `admin_username` / `admin_password` 输入框，账号管理统一收敛到 `AccountPoolTab.tsx`。`lib/adDomainApi.ts` 同步移除相关 API 字段。

### Claude's Discretion

- Wave 内部的具体波次划分顺序、`operation` 闭包封装风格、model struct tag 清理粒度——由 planner 根据代码依赖图决定。
- 测试策略：`TestAccountPoolPasswordRoundTrip` 已守护账号池解密契约；同步改 FailoverClient 后的回归测试设计由 planner + executor 决定。
- operlog 记录：`ad_account_handler` 现有 operlog 记录保持不变（账号池 CRUD 已覆盖）。
- 配置测试连接（`config.go:208` `NewLDAPClient(config)`）：改为用账号池首个可用账号测试，具体由 planner 决定。

### Deferred Ideas (OUT OF SCOPE)

- **DROP COLUMN 彻底清理**：D-02 决定本 phase 保留空列。后续版本可在新 phase 做物理 DROP COLUMN。
- **同步任务 failover 行为精细化**：批量同步（如 SyncManagersToAD 遍历多用户）是否需要区分"账号失败"vs"单用户操作失败"的上报粒度——属账号池功能增强，留给后续 phase。
- **账号池为空的引导式 UI**：D-03 返回明确错误文本引导配置；更友好的前端引导（如空池时配置页高亮账号池 tab）可作为 UX 增强单独评估。
</user_constraints>

## Summary

Phase 38 是 Phase 36 锁定的"双读兼容期 + 1 版本后清理"中的**清理步骤**：删除 `sys_ad_config.admin_username` / `admin_password` 的 Go 使用逻辑，将所有 AD 连接（同步 / 管理 / 登录认证 / 配置测试连接）统一收敛到 Phase 36 账号池 `FailoverClient`。背景动机是 `.planning/debug/adpool-password-not-decrypted.md` 揭示的**双轨不对称风险**：单管理员路径 caller 必须手动 `decryptPassword`，账号池路径内部已解密——两条路径共存期间任何 caller 遗漏解密都会触发 LDAP error 49 连锁熔断。

研究中**实际盘点比 CONTEXT 多**：生产代码 `NewLDAPClient(config)` 不带 account 的调用点是 **21 处**（非 CONTEXT 所说的 ~15 处），分布在 9 个文件；`decryptPassword(config.AdminPassword)` 是 **17 处**（非 9 处），分布在 8 个文件。CONTEXT 计数偏少但不影响波次划分——按文件分波次能完整覆盖。FailoverClient API 已确认：`ExecuteWithFailover(ctx, operation func(*LDAPClient) error) error` 与 `PickFirstConnect(ctx) (*LDAPClient, *models.ADServiceAccount, error)`，闭包内拿到已 Bind 成功的 `*LDAPClient`，自动调用 MarkSuccess/MarkFailure——这与现有同步任务"建一次连接循环用"的语义高度匹配，改造工作量集中在**注入 AccountPool 依赖 + 闭包封装现有逻辑**。

最大的**架构决策点**（已由 CONTEXT deferred，但研究给出本 phase 合理边界建议）：批量同步（`SyncManagersToAD` 信号量并发遍历多用户、`BatchMoveUsersToNewOU` 批次循环、`scheduler/dept_sync_tasks.go` 两次任务内多次 LDAP 调用）的 `operation` 闭包边界——**建议"整个任务一个 operation"**（一次 `ExecuteWithFailover` 复用同一连接做所有用户的 LDAP 操作），而非"每用户一个 operation"。理由：(1) 与现有"复用单连接"语义一致，避免每用户重新 failover；(2) FailoverClient 的 MarkFailure 上报粒度是"账号级"而非"用户操作级"——一次 operation 失败即换账号，语义清晰；(3) CONTEXT 已明确 deferred "账号失败 vs 用户操作失败"精细化，本 phase 不引入复杂度。**特例**：`SyncManagersToAD` 现有 `updateUserAttributeFn` 测试钩子需保留（7 个回归测试依赖它绕过真实 AD），改造时务必让钩子优先级高于 FailoverClient 调用。

**Primary recommendation:** Wave 1 按"服务文件"分波（sync.go / user.go / user_ad_sync_service.go / group*.go / dept_sync_service.go / scheduler / config.go），每波文件数 ≤ 3，每波独立 `go build ./...` + `go test ./internal/services/addomain/`；Wave 2 删 decryptPassword + 前端清理；Wave 3 model struct tag + migration_162 幂等校验。

<phase_requirements>
## Phase Requirements

本 phase 无 `phase_req_ids`（CONTEXT 内化了所有需求）。以下是 CONTEXT 决策映射到的研究支撑：

| 决策 | 描述 | 研究支撑 |
|------|------|----------|
| D-01 Wave 1 | 21 处 `NewLDAPClient(config)` 改走 `ExecuteWithFailover` | 见 §"调用点全量盘点"——按文件分波；§"FailoverClient API 实测签名"；§"批量同步 operation 闭包边界建议" |
| D-01 Wave 2 | 17 处 `decryptPassword(config.AdminPassword)` 删除 + 前端字段清理 | 见 §"Wave 2 清理点清单"；§"前端盘点（已部分完成的清理）" |
| D-01 Wave 3 | model struct tag 清理 + migration_162 补迁 | 见 §"migration_162 现状与补迁设计"；§"model struct 清理粒度" |
| D-03 启动空池校验 | 不阻断启动的 WARN 日志 | 见 §"启动空池校验落点" |
| D-03 运行时空池错误 | 明确错误文本引导配置 | 见 §"运行时空池错误处理路径" |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| AD 连接获取（Bind） | API / Backend (`addomain.LDAPClient`) | — | LDAP 连接是后端职责，bind 凭证来自 DB |
| 账号池 failover 选择 | API / Backend (`addomain.FailoverClient`) | — | 多账号遍历+MarkSuccess/Failure 是后端账号池语义 |
| 同步任务调度触发 | Backend Scheduler (`internal/scheduler/`) | — | cron 任务在 backend 进程内执行 |
| 启动空池校验 | Backend Bootstrap (`cmd/main.go` / `core.Core.Init`) | — | 应用启动钩子是后端进程职责 |
| 配置页表单（admin 字段清理） | Frontend (`pages/ad-domain/configs/`) | — | UI 输入项归前端 |
| 数据迁移（migration_162 补迁） | Backend (`internal/core/db/migrations/`) | Database | GORM migration 在启动时自动执行 |
| AD 登录认证（用户 bind） | API / Backend (`security.ADAuthenticator`) | — | 用户凭证验证 + 账号池绑管理员搜索用户详情 |

**关键边界：** Phase 38 不涉及 Browser/Client 层（除前端字段清理），不涉及 CDN/Static 层，不涉及 Database 物理结构变更（D-02 保留空列）。

## Standard Stack

### Core（沿用现有，本 phase 不引入新包）

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `go-ldap/ldap/v3` | v3.4.12 | LDAP 协议客户端 | 项目既有依赖，`LDAPClient` 封装基于此 [VERIFIED: go.mod] |
| `gorm.io/gorm` | v1.30.5 | ORM + 事务 + 行锁 | AccountPool.MarkSuccess/MarkFailure 用 `clause.Locking{Strength: "UPDATE"}` [VERIFIED: go.mod] |
| `google/uuid` | v1.6.0 | UUID 主键 | 现有 ADServiceAccount.ID 默认 `gen_random_uuid()` [VERIFIED: go.mod] |

### Supporting（沿用现有）

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `golang.org/x/sync/singleflight` | v0.19.0 | 同步任务防并发 | SyncService.syncDataInternal 已用 [VERIFIED: sync.go:14] |
| `golang.org/x/sync/errgroup` | v0.19.0 | 批量同步信号量并发 | SyncManagersToAD 已用 `g.SetLimit(MaxConcurrentADSync)` [VERIFIED: user_ad_sync_service.go:486] |
| `tjfoc/gmsm` | v1.4.1 | SM4 密码加解密 | 账号池密码密文加解密已通过 `PasswordCipher` 接口注入 [VERIFIED: go.mod] |

**Installation:** 无新增依赖。本 phase 仅做代码重构与清理。

**Version verification:** 所有包均为项目既有依赖，已在 `go.mod` 锁定，无需新增安装。

## Package Legitimacy Audit

> 本 phase **不安装任何外部包**（纯重构 + 清理）。无新增 npm/pip/cargo 依赖，无需运行 Package Legitimacy Gate。
>
> *slopcheck 在本环境不可用，但因无新增包，此节 N/A。*

## Architecture Patterns

### System Architecture Diagram

改造后所有 AD 连接获取统一经 FailoverClient（Wave 1 目标状态）：

```
┌─────────────────────────────────────────────────────────────────────┐
│                         触发源（5 类）                                │
└─────────────────────────────────────────────────────────────────────┘
       │              │              │              │              │
   用户登录        cron 同步       手动同步      AD 管理 CRUD    配置测试连接
   (auth)         (scheduler)    (handler)     (handler)       (handler)
       │              │              │              │              │
       ▼              ▼              ▼              ▼              ▼
┌─────────────────────────────────────────────────────────────────────┐
│  ADAuthenticator    SyncService   SyncService   User/Group/    Config │
│  .Authenticate      .SyncData     .SyncData     Mgmt Svc       .Test  │
│                     (OU/Grp/User) (Managers)    (Enable/...)   Conn   │
└─────────────────────────────────────────────────────────────────────┘
       │              │              │              │              │
       └──────────────┴──────┬───────┴──────────────┴──────────────┘
                             ▼
          ┌──────────────────────────────────────────────┐
          │   FailoverClient.ExecuteWithFailover(         │  ← Wave 1 统一入口
          │     ctx, operation func(*LDAPClient) error)   │
          └──────────────────────────────────────────────┘
                             │
              ┌──────────────┴───────────────┐
              ▼                              ▼
   ┌─────────────────────┐       ┌─────────────────────┐
   │  AccountPool.        │       │  遍历 available[i]   │
   │  ListAvailable       │──────▶│  NewLDAPClient(cfg,  │
   │  (configID)          │       │    acct)             │
   └─────────────────────┘       │  → Connect (bind)    │
                                 │  → operation(client) │
                                 │  → MarkSuccess/      │
                                 │    MarkFailure       │
                                 └─────────────────────┘
                                          │
                            ┌─────────────┴─────────────┐
                            ▼                            ▼
                   operation 成功              operation 失败
                   → 返回 nil                  → 换下一个账号
                                                → 全失败返回 ErrAllAccountsUnavailable
```

**Wave 1 改造路径说明：** 每个 caller 将现有"建一次连接循环用"或"建连接做一次操作"的逻辑，整体塞进 `operation func(client *LDAPClient) error` 闭包。FailoverClient 自动处理账号遍历 + Bind 成功/失败上报 + 连接 Close。

### Recommended Project Structure

无新增文件结构变更。本 phase 在现有目录内改造：

```
internal/services/addomain/
├── failover_client.go     # 已就绪，Wave 1 改造目标 API
├── account_pool.go        # 已就绪，ListAvailable/Create/MarkSuccess/MarkFailure
├── ldap_client.go         # 已就绪（tryBindAttempts 内部已 decryptPassword）
├── sync.go                # Wave 1 改造点（1 处 NewLDAPClient）
├── user.go                # Wave 1 改造点（4 处 NewLDAPClient）
├── user_ad_sync_service.go # Wave 1 改造点（3 处，含 SyncManagersToAD 测试钩子）
├── group_sync_service.go  # Wave 1 改造点（2 处）
├── group_management_service.go # Wave 1 改造点（4 处）
├── group.go               # Wave 1 改造点（3 处）
├── dept_sync_service.go   # Wave 1 改造点（1 处）
├── config.go              # Wave 1 改造点（1 处 TestConnection）+ Wave 2 删 encryptPassword
├── service.go             # Wave 3 改造点（移除 AdminUsername/AdminPassword 兼容字段）
└── utils.go               # decryptPassword/encryptPassword（Wave 2 后仅账号池路径用）

internal/scheduler/
├── dept_sync_tasks.go     # Wave 1 改造点（2 处 NewLDAPClient + DecryptPassword）
└── ad_sync_tasks.go       # 已用 NewAccountPool，无需改

internal/core/
├── core.go                # 启动空池校验落点（Wave 2/3）+ AccountPool 注入已就位
└── security/ad_authenticator.go # 已用 FailoverClient，Wave 2 删 decryptPassword 调用

internal/api/v1/system/
└── ad_domain_handler.go   # Wave 2 改造点（CreateConfig 返回清空 admin 字段）

internal/core/db/migrations/
└── migration_162_ad_service_accounts.go # Wave 3 校验幂等性

xingran-react-frontend/src/
├── lib/adDomainApi.ts     # Wave 2 改造点（ADConfig / Create / Update 接口移除 admin 字段）
└── pages/ad-domain/configs/index.tsx # Wave 2 改造点（表单提交时清空 admin 字段）
```

### Pattern 1: ExecuteWithFailover 闭包封装（单次操作型）

**What:** 一次 LDAP 操作（如 EnableUser、AddGroupMember、TestConnection）的改造模式。
**When to use:** `user.go` / `group.go` / `group_management_service.go` / `config.go:TestConnection` 中的单次操作。
**Example:**

```go
// Source: internal/services/addomain/failover_client.go:33-82 (实测签名)
// 改造前（user.go:Enable）:
func (s *UserService) Enable(ctx context.Context, config *models.ADConfig, userDN string) error {
    config.AdminPassword = decryptPassword(config.AdminPassword) // ← Wave 2 删除
    client := NewLDAPClient(config)
    if err := client.Connect(); err != nil { return err }
    defer client.Close()
    if err := client.EnableUser(userDN); err != nil { return err }
    // ... DB update
    return nil
}

// 改造后（Wave 1）:
func (s *UserService) Enable(ctx context.Context, pool AccountPool, config *models.ADConfig, userDN string) error {
    fc := NewFailoverClient(pool, config)
    if err := fc.ExecuteWithFailover(ctx, func(client *LDAPClient) error {
        return client.EnableUser(userDN)
    }); err != nil {
        return err
    }
    // ... DB update（不变）
    return nil
}
```

**关键点：**
1. **删除** `config.AdminPassword = decryptPassword(...)` —— FailoverClient 内部已通过 `tryBindAttempts` → `decryptPassword(c.account.PasswordCiphertext)` 自动解密 [VERIFIED: ldap_client.go:143]。
2. **删除** `NewLDAPClient(config)` + `Connect()` + `defer Close()` —— FailoverClient 内部循环已封装。
3. **方法签名变化：** 需要注入 `pool AccountPool` 参数。两个选项供 planner 决定：
   - **选项 A（推荐）：** 在 `UserService` / `GroupService` / `GroupManagementService` struct 增加 `pool AccountPool` 字段，构造函数注入。改动小，符合现有 Handler-Service 模式。
   - **选项 B：** 方法级参数注入。改动量大但更灵活。
   - 建议 planner 选 A，与 `core.Core.initAuthFactory` 已有的 `NewAccountPool` 注入模式一致 [VERIFIED: core.go:581]。

### Pattern 2: 批量同步 operation 闭包边界（复用单连接型）

**What:** 批量同步任务（一次连接做多次 LDAP 操作）的改造模式。
**When to use:** `sync.go:syncDataInternal` / `user_ad_sync_service.go:SyncManagersToAD` / `BatchMoveUsersToNewOU` / `scheduler/dept_sync_tasks.go`。
**Example（SyncManagersToAD，最复杂的批量场景）:**

```go
// Source: internal/services/addomain/user_ad_sync_service.go:471-482 (现有逻辑)
// 改造前（测试钩子优先，否则建真实连接复用）:
if s.updateUserAttributeFn != nil {
    updateAttr = s.updateUserAttributeFn  // 测试钩子
} else {
    ldapClient := NewLDAPClient(&adConfig)
    if err := ldapClient.Connect(); err != nil { return ... }
    defer ldapClient.Close()
    updateAttr = ldapClient.UpdateUserAttribute
}
// 后续 g.Go 并发调用 updateAttr(userDN, attrs)

// 改造后（Wave 1，保留测试钩子，FailoverClient 包裹整个批量）:
if s.updateUserAttributeFn != nil {
    updateAttr = s.updateUserAttributeFn  // 测试钩子保持不变（7 个回归测试依赖）
} else {
    fc := NewFailoverClient(pool, &adConfig)
    // 整个批量同步一个 operation：连接复用，所有用户共用一次 failover 选定的账号
    err := fc.ExecuteWithFailover(ctx, func(client *LDAPClient) error {
        updateAttr = client.UpdateUserAttribute
        // 在闭包内执行整个 errgroup 批量（注意：operation 返回 nil 表示账号成功，
        // errgroup 内单用户失败已在 result.Failed 计数，不让 operation 整体失败）
        g, gctx := errgroup.WithContext(ctx)
        g.SetLimit(constants.MaxConcurrentADSync)
        // ... 遍历 resolved 调 updateAttr
        _ = g.Wait()
        return nil // 即使部分用户失败，账号本身是成功的（MarkSuccess）
    })
    if err != nil { return nil, err }
}
```

**关键决策：operation 闭包边界 = 整个批量任务**（非每用户）。理由：
1. 与现有"复用单连接"语义一致 [CITED: user_ad_sync_service.go:476 注释"复用单连接"]。
2. FailoverClient MarkFailure 上报粒度是账号级——operation 失败即换账号，语义清晰。
3. CONTEXT deferred 了"账号失败 vs 用户操作失败"精细化，本 phase 不引入复杂度。
4. **警告：** SyncManagersToAD 的 `updateUserAttributeFn` 测试钩子**必须保留**，否则 7 个 `TestSyncManagersToAD_*` 回归测试会全部失败（它们依赖钩子绕过真实 AD）[VERIFIED: manager_sync_test.go:130-135]。

### Pattern 3: 配置测试连接（PickFirstConnect）

**What:** `config.go:TestConnection` 改造模式——只验证"能连上 + bind 成功"，不做后续操作。
**When to use:** 仅 `config.go:208`。
**Example:**

```go
// Source: internal/services/addomain/failover_client.go:88-119 (PickFirstConnect 实测签名)
// 改造后（Wave 1）:
func (s *ConfigService) TestConnection(ctx context.Context, pool AccountPool, config *models.ADConfig) error {
    fc := NewFailoverClient(pool, config)
    client, _, err := fc.PickFirstConnect(ctx) // 只建连接 + bind，不做操作
    if err != nil {
        return fmt.Errorf("AD 连接测试失败（账号池无可用账号或全部 bind 失败）: %w", err)
    }
    defer client.Close()
    return nil
}
```

**参考样板：** `ad_authenticator.go:270` 已用 `NewFailoverClient + PickFirstConnect` [VERIFIED: ad_authenticator.go:270-275]，是登录认证改造的现成模板。

### Anti-Patterns to Avoid

- **反模式 1：保留 `config.AdminPassword = decryptPassword(...)` 作为 fallback。** CONTEXT D-03 明确"不保留单管理员作 fallback"。理由：双轨不对称是本次清理的根因（debug session adpool-password-not-decrypted），保留 fallback 会重新引入"caller 遗漏解密"风险。
- **反模式 2：每用户一个 operation（SyncManagersToAD 改造）。** 会导致每用户重新 failover，账号池负载激增，且 MarkFailure 上报粒度混乱（账号失败 vs 用户操作失败混淆）。用 Pattern 2 的"整个批量一个 operation"。
- **反模式 3：删除 `updateUserAttributeFn` 测试钩子。** 7 个回归测试依赖它。改造时让钩子优先级高于 FailoverClient 调用（钩子非 nil 时直接用钩子，不走 FailoverClient）。
- **反模式 4：在 `operation` 闭包外访问 `client`。** FailoverClient 在循环内 `client.Close()` [VERIFIED: failover_client.go:65]，闭包返回后 client 已关闭，闭包外访问会 panic。所有 client 使用必须在闭包内完成。
- **反模式 5：DROP COLUMN。** D-02 明确保留空列。物理 DROP COLUMN 是 deferred 项。

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| 多账号 failover + MarkSuccess/Failure 上报 | 手写账号遍历 + 行锁 + 熔断判断 | `FailoverClient.ExecuteWithFailover` | 已实现 maxHops 动态计算、顺序遍历、自动 MarkSuccess/Failure [VERIFIED: failover_client.go:33-82] |
| 账号池密码解密 | caller 手动 `decryptPassword(account.PasswordCiphertext)` | `tryBindAttempts` 内部已自动解密 | debug session adpool-password-not-decrypted 的根因就是 caller 遗漏解密，已改为内部解密 [VERIFIED: ldap_client.go:143] |
| 账号池可用账号过滤（status/熔断到期） | 手写 SQL `WHERE status = 0 OR (status = 2 AND ...)` | `AccountPool.ListAvailable(ctx, configID)` | 已实现纯读 + 30s TTL 缓存 + 跨进程 pub/sub 失效 [VERIFIED: account_pool.go:134-174] |
| 启动时空池检查 | 手写 DB 查询 + 日志 | 直接用 `AccountPool.CountByStatus` 或 `ListAvailable` | 已实现 O(状态数) 统计 [VERIFIED: account_pool.go:223-250] |
| 数据迁移幂等 | 手写 `SELECT count(*) ... INSERT` | 沿用 migration_162 现有幂等模式 | 已实现"先 count，>0 则 skip"幂等 [VERIFIED: migration_162_ad_service_accounts.go:57-68] |

**Key insight:** Phase 36 已把所有"难的部分"（多账号 failover、密码内部解密、账号池缓存、熔断恢复 cron）实现完毕。Phase 38 是**纯调用方收敛**——把 21 处直连 `NewLDAPClient(config)` 改为调 FailoverClient，删 17 处手动解密。不引入新的复杂度。

## Runtime State Inventory

> 本 phase 是"删代码 + 字段清理"型重构，触发运行时状态盘点（D-02 保留空列意味着 DB schema 不变，但需确认无运行时系统依赖旧字段）。

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | `sys_ad_config.admin_username` / `admin_password` 列数据仍存在（D-02 保留）；`sys_ad_service_accounts` 表已有 migration_162 迁入的账号 | migration_162 补迁逻辑保证单管理员账号已入账号池（Wave 3 校验幂等性）；DB 列数据不动 |
| Live service config | AccountPool 在 4 处 `NewAccountPool(db, nil)` 实例化（core.go:581、ad_domain_router.go:17、ad_sync_tasks.go:105、account_pool_test.go:47）——均为进程内单例或任务内临时实例，无外部服务配置 | 无需迁移；Wave 1 改造的同步服务需通过 struct 字段或参数注入复用 core 的 AccountPool 实例 |
| OS-registered state | 无（无 cron job 名称、无 systemd unit、无 Task Scheduler 任务引用 admin 字段；scheduler 内部 cron 在进程内注册） | None — verified by grep `AdminUsername` / `AdminPassword` 无 OS 注册点 |
| Secrets/env vars | `AD_LEGACY_AES_KEY` 环境变量用于 legacy AES 解密回退链 [VERIFIED: utils.go:117]；SM4 key 通过 `core.SM4Cipher` 注入 | 无需改动；Wave 2 删除 `decryptPassword(config.AdminPassword)` 后，`decryptPassword` 函数本身保留（账号池路径 `tryBindAttempts` 仍用） |
| Build artifacts | `internal/services/ad_ldap_client.go` 是**遗留孤立文件**（定义了与 `addomain.LDAPClient` 冲突的 `services.LDAPClient` 类型），全项目无生产代码调用 `services.NewLDAPClient`（仅 docs/ 引用）[VERIFIED: grep `services.NewLDAPClient` 仅命中定义处] | 建议本 phase 顺手删除该孤立文件（清理技术债），但不强制——planner 可决定是否纳入 |

**The canonical question:** *After every file in the repo is updated, what runtime systems still have the old string cached, stored, or registered?*
**Answer:** 无。所有 AD 连接获取都在 Go 进程内完成（`addomain.LDAPClient` → go-ldap → TCP），无外部服务、无 OS 注册、无跨进程缓存（Redis pub/sub 仅用于账号池 breaker 恢复事件广播，不缓存 admin 字段）。唯一需要校验的是 **DB 数据完整性**（migration_162 已迁入），由 Wave 3 校验幂等性覆盖。

## Common Pitfalls

### Pitfall 1: CONTEXT 计数偏少（21 处非 15 处，17 处非 9 处）

**What goes wrong:** 按 CONTEXT 说的"~15 处"/"9 处"清点，会漏改 6+ 处调用点，导致残留单管理员路径。
**Why it happens:** CONTEXT 基于早期盘点，未覆盖 `user_ad_sync_service.go`（3 处）、`scheduler/dept_sync_tasks.go`（2 处 DecryptPassword）等。
**How to avoid:** 用本研究的精确清单（见 §"调用点全量盘点"）逐文件核对。验证命令：

```bash
# Wave 1 完成后，无 account 参数的 NewLDAPClient(config) 调用点应为 0
grep -rn "NewLDAPClient(config)\|NewLDAPClient(&config)\|NewLDAPClient(&adConfig)" internal/ --include="*.go" | grep -v "_test.go" | grep -v "failover_client.go"
```

**Warning signs:** grep 仍有命中且不在 `failover_client.go` → 有遗漏。

### Pitfall 2: 测试钩子 `updateUserAttributeFn` 被误删

**What goes wrong:** SyncManagersToAD 改 FailoverClient 时删除测试钩子，7 个 `TestSyncManagersToAD_*` 回归测试全部失败。
**Why it happens:** 改造者认为"钩子是临时调试代码"。
**How to avoid:** 明确钩子优先级：`s.updateUserAttributeFn != nil` 时走钩子，否则走 FailoverClient。在闭包外判断，钩子分支完全绕过 FailoverClient。参考 `user_ad_sync_service.go:472-482` 现有模式。
**Warning signs:** `go test ./internal/services/addomain/` 报 `TestSyncManagersToAD_*` 失败。

### Pitfall 3: 在 operation 闭包外访问 client

**What goes wrong:** 把 `client` 变量声明在闭包外，闭包内赋值，闭包外再用 → 闭包返回后 FailoverClient 已 `client.Close()` [VERIFIED: failover_client.go:65]，访问已关闭连接 panic 或 LDAP 操作失败。
**Why it happens:** 改造者想"复用连接做后续 DB 操作"。
**How to avoid:** 所有 `client` 使用必须在 operation 闭包内完成。DB 操作（如更新 `is_enabled` 字段）可以在闭包外，但 LDAP 操作必须在闭包内。
**Warning signs:** panic `use of closed network connection` 或 LDAP 操作间歇性失败。

### Pitfall 4: FailoverClient 实例未复用导致缓存失效

**What goes wrong:** 每次调用都 `NewAccountPool(db, nil)` 创建新池实例，每个实例有独立的 30s 内存缓存 [VERIFIED: account_pool.go:107-108]，缓存不共享 → MarkFailure 后其他实例仍用旧快照。
**Why it happens:** 改造者在每个服务方法内 `NewAccountPool`。
**How to avoid:** 通过 struct 字段注入**同一个** AccountPool 实例（core 已在 `initAuthFactory` 创建并注入 [VERIFIED: core.go:581]）。Handler-Service 模式下，router setup 时注入。
**Warning signs:** 账号熔断后仍被选中尝试。

### Pitfall 5: SyncManagersToAD operation 闭包内 errgroup 阻塞

**What goes wrong:** 在 operation 闭包内启动 errgroup 并发，但 errgroup 的 `g.Wait()` 阻塞导致 operation 长时间持有连接 → failover 超时。
**Why it happens:** errgroup 默认无超时。
**How to avoid:** 用 `errgroup.WithContext(ctx)` 继承调用方 context，确保 context 超时能取消整个批量。现有代码已用此模式 [VERIFIED: user_ad_sync_service.go:485]。
**Warning signs:** 同步任务超时 `context deadline exceeded`。

### Pitfall 6: 启动空池校验阻断新环境部署

**What goes wrong:** 启动时发现空池 → 返回 error → `core.Init()` 失败 → 应用无法启动 → 新环境首次部署阻塞。
**Why it happens:** 误把"校验"当"硬约束"。
**How to avoid:** D-03 明确"记 WARN 日志，**不阻断启动**"。校验函数返回 nil，仅 `applogger.Warnf` 记录引导信息。
**Warning signs:** 新环境部署时应用启动失败。

## Code Examples

### ExecuteWithFailover 完整调用（来自 FailoverClient 源码）

```go
// Source: internal/services/addomain/failover_client.go:33-82 (实测)
func (f *FailoverClient) ExecuteWithFailover(
    ctx context.Context,
    operation func(client *LDAPClient) error,
) error {
    available, err := f.pool.ListAvailable(ctx, f.config.ID)
    if err != nil {
        return fmt.Errorf("查询账号池失败: %w", err)
    }
    if len(available) == 0 {
        return ErrAllAccountsUnavailable  // ← 运行时空池错误，D-03 要求返回明确文本
    }
    maxAttempts := len(available)
    if maxAttempts > DefaultMaxHops { maxAttempts = DefaultMaxHops }
    var lastErr error
    for i := 0; i < maxAttempts; i++ {
        acct := &available[i]
        client := NewLDAPClient(f.config, acct)
        if err := client.Connect(); err != nil {
            f.pool.MarkFailure(ctx, acct.ID, "dial:"+err.Error())
            lastErr = err
            continue
        }
        err := operation(client)  // ← caller 的闭包在此执行
        client.Close()
        if err == nil {
            f.pool.MarkSuccess(ctx, acct.ID)  // ← 成功上报
            return nil
        }
        f.pool.MarkFailure(ctx, acct.ID, "operation:"+err.Error())  // ← 失败上报
        lastErr = err
    }
    return fmt.Errorf("账号池 %d 个账号均失败: %w", maxAttempts, lastErr)
}
```

### 运行时空池错误文本（D-03 引导配置）

```go
// 改造各同步/管理 caller 时，捕获 ErrAllAccountsUnavailable 返回引导文本
if errors.Is(err, addomain.ErrAllAccountsUnavailable) {
    return fmt.Errorf("AD 账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号: %w", err)
}
```

### 启动空池校验（D-03 WARN 日志）

```go
// Source: 参考 cmd/main.go:initializeCoreModule 模式
// 落点建议：core.Core.Init() 末尾或 cmd/main.go:initializeCoreModule 内 coreModule.Init() 之后
func checkEmptyAccountPoolOnStartup(db *gorm.DB) {
    var configs []models.ADConfig
    db.Where("status = ? AND sync_enabled = ?", models.ADConfigStatusEnabled, true).Find(&configs)
    pool := addomain.NewAccountPool(db, nil)
    for _, cfg := range configs {
        total, available, _, _ := 0, 0, 0, 0
        total, available, _, _, err := pool.CountByStatus(context.Background(), cfg.ID)
        if err != nil {
            applogger.Warnf("[启动校验] AD 配置 %s 账号池查询失败: %v", cfg.ConfigName, err)
            continue
        }
        if total == 0 || available == 0 {
            applogger.Warnf("[启动校验] AD 配置 %s (ID=%s) 账号池为空（total=%d, available=%d），"+
                "请在 AD 配置页详情 → 服务账号池 Tab 添加服务账号，否则登录/同步将失败",
                cfg.ConfigName, cfg.ID, total, available)
        }
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `NewLDAPClient(config)` 单管理员直连 | `FailoverClient.ExecuteWithFailover(ctx, operation)` 多账号故障切换 | Phase 36 (2026-06) | 单账号锁定（data 775）不再阻断所有用户登录/同步 |
| caller 手动 `decryptPassword(config.AdminPassword)` | `tryBindAttempts` 内部自动 `decryptPassword(c.account.PasswordCiphertext)` | debug session adpool-password-not-decrypted (2026-06-23) | 消除双轨不对称的"遗漏解密"风险 |
| 启动无空池校验 | D-03 启动 WARN + 运行时明确错误 | Phase 38（本 phase） | 新环境部署不阻断，运行时引导配置 |
| `bindAdmin`（ad_authenticator 旧方法，已 @Deprecated） | `bindAdminWithFailover`（FailoverClient.PickFirstConnect） | Phase 36 | Phase 38 Wave 2 可删除 `bindAdmin` 死代码 |

**Deprecated/outdated:**
- `internal/services/addomain/ldap_client.go:44` 注释"向后兼容：传 nil 时 fallback 到 config.AdminUsername/AdminPassword"——Wave 2 后此 fallback 分支变死代码（无 caller 传 nil），可清理 `tryBindAttempts` 的 `c.account != nil` 判断或保留为防御性代码（planner 决定）。
- `internal/core/security/ad_authenticator.go:238-247` `bindAdmin` 方法已 @Deprecated（被 `bindAdminWithFailover` 替代），Wave 2 可删除。
- `internal/services/ad_ldap_client.go` 整个文件是遗留孤立代码（无生产 caller），建议本 phase 顺手清理。

## Assumptions Log

> slopcheck 在本环境不可用（已尝试 `pip install slopcheck` 失败）。按 Package Legitimacy Protocol 的 graceful degradation，本应将所有包标 `[ASSUMED]`——但本 phase **不引入任何新包**（纯重构 + 清理），所有依赖均为项目既有 go.mod 锁定，故 Assumptions Log 仅记录架构决策假设。

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `updateUserAttributeFn` 测试钩子必须保留 | Pattern 2 / Pitfall 2 | 7 个回归测试失败；改造者误删需补回 |
| A2 | operation 闭包边界 = 整个批量任务（非每用户） | Pattern 2 | CONTEXT deferred 此决策，本研究建议但不锁定；planner 可重新评估 |
| A3 | `internal/services/ad_ldap_client.go` 是孤立死代码 | Runtime State Inventory | 若有未发现的 caller，删除会导致编译失败；建议删除前 `go build ./...` 验证 |
| A4 | struct 字段注入 AccountPool（选项 A）优于方法参数注入（选项 B） | Pattern 1 | planner 可选 B，改动量更大但更灵活 |
| A5 | 启动空池校验落点建议在 `core.Init()` 末尾或 `cmd/main.go` | Code Examples | 若 core.Init 无法访问完整 ADConfig 列表，需调整落点 |
| A6 | `syncADConfig` (ad_sync_tasks.go:251) 已用 `NewADDomainService` 不需改 | 调用点盘点 | SyncService 改造后，`NewADDomainService` 需注入 AccountPool 才能让 SyncService 用上 |

## Open Questions

1. **SyncService 如何注入 AccountPool？**
   - What we know: `NewSyncService(db)` 现签名只有 db；`NewADDomainService(db, cipher...)` 聚合了 SyncService 但未注入 pool。
   - What's unclear: 是否要改 `NewSyncService(db, pool)` 签名，还是 `SyncService` struct 加 `pool` 字段 + setter。
   - Recommendation: struct 加 `pool` 字段，`NewADDomainService` 构造时注入（与 `core.Core.initAuthFactory` 模式一致）。这影响所有 sub-service（UserService/GroupService/...），planner 需决定是统一注入还是按需注入。

2. **前端 `ADConfigCreateRequest.adminUsername/adminPassword` 是否能直接删？**
   - What we know: 后端 `CreateRequest.AdminUsername` 仍 `binding:"required"` [VERIFIED: config.go:85-86]；前端表单已不收集 [VERIFIED: configs/index.tsx:414-415 注释]。
   - What's unclear: 前端提交时 `adminUsername` 会是空字符串，后端 `required` 校验会 400。
   - Recommendation: Wave 2 同步删除后端 `binding:"required"`（改为 `omitempty` 或删除字段），前后端配套。这是 D-04 "lib/api/system.ts 同步移除相关 API 字段" 的具体落地。

3. **`internal/services/ad_ldap_client.go` 孤立文件是否纳入本 phase 清理？**
   - What we know: 全项目无生产 caller [VERIFIED: grep 仅命中定义处 + docs/]；定义了与 addomain 冲突的 `services.LDAPClient` 类型。
   - Recommendation: 纳入 Wave 3 顺手清理（删除整个文件 + `go build ./...` 验证）。但 planner 可决定 defer 到独立技术债 phase。

## Environment Availability

> 本 phase 是纯 Go 代码重构 + 前端字段清理，无外部工具依赖。所有验证用 Go 工具链 + 现有测试基础设施。

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | 编译验证 | ✓ | go1.24.5 [VERIFIED: CLAUDE.md] | — |
| SQLite (in-memory) | AccountPool 单元测试 | ✓ | gorm.io/driver/sqlite v1.5.4 [VERIFIED: go.mod] | — |
| PostgreSQL 18 | 集成测试（手动） | ✓（开发环境） | 18 [VERIFIED: CLAUDE.md] | 单元测试用 SQLite 覆盖，无需 PG |
| Node.js / npm | 前端 type-check + build | ✓ | 现有工具链 | — |

**Missing dependencies with no fallback:** 无。
**Missing dependencies with fallback:** 无。

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go testing + testify (v1.10.0) [VERIFIED: go.mod] |
| Config file | 无中央配置（Go 标准 `go test`） |
| Quick run command | `go test ./internal/services/addomain/ -count=1` |
| Full suite command | `go test ./... && go build ./...` |

### Phase Requirements → Test Map

本 phase 是重构清理，无新功能用户可观测行为。验证策略聚焦"证明所有 AD 连接已走账号池、无残留单管理员路径"。

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| D-01-W1 | 所有 `NewLDAPClient(config)` 无 account 调用点 = 0（除 failover_client.go 内部） | grep 断言 | `grep -rn "NewLDAPClient(config)\|NewLDAPClient(&config)\|NewLDAPClient(&adConfig)" internal/ --include="*.go" \| grep -v "_test.go" \| grep -v "failover_client.go" \| wc -l` 应为 0 | N/A（grep） |
| D-01-W2 | 所有 `decryptPassword(config.AdminPassword)` 调用点 = 0 | grep 断言 | `grep -rn "decryptPassword(config\.AdminPassword)\|decryptPassword(adConfig\.AdminPassword)" internal/ --include="*.go" \| grep -v "_test.go" \| wc -l` 应为 0 | N/A（grep） |
| D-01-W2 | 前端 `adminPassword` / `adminUsername` 在 adDomainApi.ts 中 = 0（或仅注释） | grep 断言 | `grep -n "adminPassword\|adminUsername" xingran-react-frontend/src/lib/adDomainApi.ts` 应仅命中注释或为空 | N/A（grep） |
| 回归 | 账号池密码加解密闭环契约 | unit | `go test ./internal/services/addomain/ -run TestAccountPoolPasswordRoundTrip -v` | ✅ |
| 回归 | FailoverClient maxHops 行为 | unit | `go test ./internal/services/addomain/ -run TestFailoverClient -v` | ✅ |
| 回归 | SyncManagersToAD 7 个场景 | unit | `go test ./internal/services/addomain/ -run TestSyncManagersToAD -v` | ✅ |
| 回归 | F-03 解密失败返回空（安全） | unit | `go test ./internal/services/addomain/ -run TestDecryptPassword_InvalidCiphertext -v` | ✅ |
| 编译 | 全项目编译通过 | build | `go build ./...` | N/A |
| 前端 | type-check 通过 | build | `cd xingran-react-frontend && npm run type-check` | N/A |
| 前端 | 生产构建通过 | build | `cd xingran-react-frontend && npm run build` | N/A |

### Sampling Rate

- **Per task commit:** `go build ./internal/services/addomain/... && go test ./internal/services/addomain/ -count=1`
- **Per wave merge:** `go build ./... && go test ./...`
- **Phase gate:** 全量 grep 断言（NewLDAPClient 无 account = 0 / decryptPassword(config.AdminPassword) = 0）+ `go test ./...` + 前端 `npm run type-check` + `npm run build`

### Wave 0 Gaps

- [ ] **回归测试增强（建议但非强制）：** 为 Wave 1 改造后的关键 caller 增加空池场景测试（如 `TestUserService_Enable_EmptyPool` 验证返回 ErrAllAccountsUnavailable 引导文本）。planner + executor 决定是否纳入。
- [ ] **grep 断言脚本（建议）：** 可选创建 `scripts/verify_no_legacy_admin_path.sh` 锁定 grep 断言，供后续 phase 回归。planner 决定。

*(现有测试基础设施已覆盖核心契约——TestAccountPoolPasswordRoundTrip + 7 个 SyncManagersToAD 场景 + FailoverClient 测试。Wave 0 主要靠 grep 断言验证"无残留路径"。)*

## Security Domain

> `security_enforcement` 在 config.json 中未显式设置（absent = enabled）。本 phase 涉及 AD 凭证处理，纳入 Security Domain 审查。

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | AD 用户登录认证（用户 bind）—— 本 phase 不改用户认证逻辑，仅改管理员账号获取方式 |
| V3 Session Management | no | 本 phase 不涉及 session |
| V4 Access Control | yes | AD 账号池权限粒度（list/edit/delete 三级）已由 Phase 36 锁定，本 phase 不改 [VERIFIED: ad_domain_router.go:50-75] |
| V5 Input Validation | yes | `CreateRequest` / `UpdateRequest` 移除 `adminPassword` 字段后，binding 校验需同步调整（`required` → 删除或 `omitempty`） |
| V6 Cryptography | yes | SM4 密码加解密——本 phase 删除手动 `decryptPassword(config.AdminPassword)` 后，解密逻辑收敛到 `tryBindAttempts` 内部（已由 `TestAccountPoolPasswordRoundTrip` 守护） |
| V7 Error Handling | yes | D-03 运行时空池错误返回明确文本（不泄露账号池内容），启动 WARN 日志不泄露敏感信息 |

### Known Threat Patterns for AD/LDAP Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| 凭证泄露（日志/错误消息） | Information Disclosure | F-03 已守护：`decryptPassword` 失败返回空而非明文 [VERIFIED: utils.go:88-99]；本 phase 错误文本不包含密码/账号详情 |
| 账号锁定 DoS（data 775） | Denial of Service | FailoverClient 多账号故障切换（Phase 36 已实现）——本 phase 收敛所有路径到 FailoverClient，消除单管理员单点 |
| LDAP injection | Tampering | 用户 DN 在 LDAP 操作前未转义——本 phase 不改 LDAP 操作逻辑，沿用现有 `ldap.EscapeFilter`（ad_authenticator.go:287 已用） |
| 双轨不对称导致遗漏解密 | Information Disclosure / DoS | 本 phase 核心动机：删除单管理员路径，消除"caller 遗漏 decryptPassword"风险（debug session adpool-password-not-decrypted 根因） |

**关键安全收益：** Phase 38 完成后，AD 凭证处理从"两条路径（单管理员 caller 解密 + 账号池内部解密）"收敛为"单一路径（账号池内部解密）"。任何密码处理 bug 只需修一处（`tryBindAttempts`），不再有"修了账号池路径忘了单管理员路径"或反之的风险。

## Sources

### Primary (HIGH confidence)

- `internal/services/addomain/failover_client.go` — ExecuteWithFailover / PickFirstConnect 实测签名（行 33-119）
- `internal/services/addomain/account_pool.go` — AccountPool 接口 + ListAvailable/MarkSuccess/MarkFailure 实现（行 49-101, 134-174, 303-371）
- `internal/services/addomain/ldap_client.go` — tryBindAttempts 内部解密契约（行 134-167），NewLDAPClient variadic 签名（行 52）
- `internal/core/security/ad_authenticator.go` — bindAdminWithFailover 参考样板（行 256-278），bindAdmin @Deprecated（行 238-247）
- `internal/core/db/migrations/migration_162_ad_service_accounts.go` — 幂等迁移模式（行 57-68）
- `internal/services/addomain/account_pool_test.go` / `ldap_client_test.go` / `manager_sync_test.go` — 现有测试覆盖范围
- `internal/api/v1/system/ad_domain_router.go` — AccountPool 注入点（行 17）
- `internal/core/core.go` — initAuthFactory AccountPool 创建（行 581）
- `internal/scheduler/ad_sync_tasks.go` — scheduler 已用 NewAccountPool（行 105）
- `.planning/debug/adpool-password-not-decrypted.md` — 双轨不对称根因实证

### Secondary (MEDIUM confidence)

- `internal/models/ad_domain.go` — AdminUsername/AdminPassword @Deprecated 注释（行 29-32）
- `internal/models/ad_service_account.go` — ADServiceAccount 表结构 + 状态机（行 19-79）
- `internal/services/addomain/utils.go` — decryptPassword/encryptPassword 实现与 legacy AES 回退（行 50-99）
- `internal/services/addomain/service.go` — ADDomainService 聚合根（行 100-136）
- `xingran-react-frontend/src/lib/adDomainApi.ts` — 前端 ADConfig/Create/Update 接口含 admin 字段（行 13-14, 161-162, 175-176）
- `xingran-react-frontend/src/pages/ad-domain/configs/index.tsx` — 前端表单已移除 admin 输入但提交逻辑仍含字段（行 92, 108-113, 131）

### Tertiary (LOW confidence)

- 无。所有发现均来自代码库直接读取或 debug session 记录。

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — 全部项目既有依赖，go.mod 锁定
- Architecture: HIGH — FailoverClient API 已实测，21 处调用点已全量盘点
- Pitfalls: HIGH — 6 个 pitfalls 中 5 个有 debug session 或现有测试守护佐证
- 调用点计数: HIGH — grep 实测，与 CONTEXT 计数偏差已明确标注（21 vs 15，17 vs 9）
- 前端盘点: HIGH — 实际路径 `pages/ad-domain/configs/`（非 CONTEXT 说的 `pages/system/ad-domain/config`），已确认 Phase 36 部分清理但 API 接口字段残留

**Research date:** 2026-06-23
**Valid until:** 2026-07-23（30 天，重构清理类稳定；若 Phase 36 账号池有大改动需重新评估）
