# Phase 38: AD 账号池统一（移除遗留单管理员双轨）- Pattern Map

**Mapped:** 2026-06-23
**Files analyzed:** 15（待改造/新建文件）
**Analogs found:** 15 / 15（全部找到现有 analog，无 "No Analog Found"）
**Phase 性质:** 纯重构清理——无新增文件，全部为既有文件改造。Pattern Map 核心价值是提取「FailoverClient 闭包封装样板」「AccountPool 注入样板」「改造前后对照」「前端删除样板」「migration 幂等样板」。

---

## File Classification

按 **「连接获取层」(role+data flow)** 维度分类。改造模式分 3 类（与 D-01 三个 Wave 对齐）。

### Wave 1 — 连接层统一（`NewLDAPClient(config)` → `FailoverClient.ExecuteWithFailover`）

| 待改造文件 | Role | Data Flow | 改造点 | Closest Analog | Match Quality |
|------------|------|-----------|--------|----------------|---------------|
| `internal/services/addomain/sync.go` | service (同步任务层) | 批量 transform (一次连接多次 LDAP) | line 90, 93 | `ad_authenticator.go:256-278` (`bindAdminWithFailover`) + `failover_client.go:33-82` | exact (API 已就绪) |
| `internal/services/addomain/user.go` | service (管理操作层) | 单次 request-response (Enable/Disable/Move/Update ×4) | lines 124, 193, 214, 235 | `failover_client.go:33-82` (单次 operation) | exact |
| `internal/services/addomain/group.go` | service (管理操作层) | 单次 request-response (×3) | lines 125, 155, 187 | `failover_client.go:33-82` | exact |
| `internal/services/addomain/group_sync_service.go` | service (同步任务层) | 批量 transform | lines 55, 93 | `failover_client.go:33-82` | exact |
| `internal/services/addomain/group_management_service.go` | service (管理操作层) | 单次 request-response (×4) | lines 86, 154, 202, 260 | `failover_client.go:33-82` | exact |
| `internal/services/addomain/dept_sync_service.go` | service (同步任务层) | 批量 transform | line 46 | `failover_client.go:33-82` | exact |
| `internal/services/addomain/user_ad_sync_service.go` | service (同步任务层 + 测试钩子) | 批量 errgroup 并发 + 测试 stub | lines 59, 207, 476 | `failover_client.go:33-82` + `user_ad_sync_service.go:471-482` (现钩子样板) | exact（**含特殊约束**：保留 `updateUserAttributeFn` 测试钩子） |
| `internal/scheduler/dept_sync_tasks.go` | scheduler 任务函数 | 批量 transform (任务内多次 LDAP) | lines 88-89, 160-163 | `scheduler/ad_sync_tasks.go:105` (已用 `NewAccountPool`) + `failover_client.go:33-82` | exact |
| `internal/services/addomain/config.go` (TestConnection) | service (配置测试连接层) | 单次 request-response (只建连接不操作) | line 205, 208 | `ad_authenticator.go:270-275` (`PickFirstConnect`) | exact |

### Wave 2 — 删除单管理员使用代码 + 前端字段清理

| 待改造文件 | Role | Data Flow | 改造点 | Closest Analog | Match Quality |
|------------|------|-----------|--------|----------------|---------------|
| `internal/services/addomain/config.go` (Create/Update) | service (配置 CRUD) | CRUD | lines 85-86 (CreateRequest `binding:"required"`) + 108, 169 (`encryptPassword`) | `internal/services/addomain/account_pool.go:67` (`AccountPool.Create` 字段名 `password_ciphertext` 自动脱敏) + 现有 CreateRequest 字段删除模式 | exact |
| `internal/services/addomain/service.go` (聚合根兼容层) | service (aggregator) | 兼容 shim | lines 29-30, 47, 168-169, 188-189 (AdminUsername/AdminPassword 兼容字段) | `service.go:119-136` (`NewADDomainService` 构造注入模式) | exact |
| `internal/core/security/ad_authenticator.go` | 认证 (登录认证层，已就位) | request-response | lines 99 (`config.AdminPassword = a.decryptPassword(...)`) + 238-247 (`bindAdmin` 死代码删除) + 258-267 (Phase 36 双读 fallback 分支删除) | `ad_authenticator.go:269-278` (现 FailoverClient 主路径) | exact |
| `internal/api/v1/system/ad_domain_handler.go` | handler | request-response | CreateConfig 返回清空 admin 字段（如有） | `ad_domain_handler.go` 现有 CreateConfig handler | exact |
| `xingran-react-frontend/src/lib/adDomainApi.ts` | frontend API 类型 | type 删除 | lines 13-14 (`ADConfig.adminUsername/adminPassword`), 161-162 (`ADConfigCreateRequest`), 175-176 (`ADConfigUpdateRequest`) | 字段直接删除（无替代 analog，类型字段删除标准 TS 操作） | role-match |
| `xingran-react-frontend/src/pages/ad-domain/configs/index.tsx` | frontend 组件 | form 提交 | lines 92 (`handleEdit` setFieldsValue `adminUsername/adminPassword: undefined`), 108-113 (注释), 131 (updateData `adminPassword`) | `configs/index.tsx:54-56` (Phase 36 已用的 `AccountPoolTab` 收敛模式) | role-match |

### Wave 3 — model struct tag 清理 + migration_162 幂等校验

| 待改造文件 | Role | Data Flow | 改造点 | Closest Analog | Match Quality |
|------------|------|-----------|--------|----------------|---------------|
| `internal/models/ad_domain.go` | model | struct tag | lines 29-32 (`AdminUsername`/`AdminPassword` `@Deprecated` 注释 + `not null` tag 放宽) | `internal/models/ad_service_account.go` (账号池 model，作为新字段迁移目标) | role-match |
| `internal/core/db/migrations/migration_164_phase38_verify_admin_migrated.go` (新建) | migration | 幂等补迁校验 | 校验现有 `sys_ad_config.admin_*` 已迁入账号池 | `migration_162_ad_service_accounts.go:55-68` (现有"先 count，>0 则 skip"幂等样板) | exact |
| `internal/core/db/database.go`（AutoMigrate 真实注册点 — 追加 Migrate164 调用） | migration | 注册 | `AutoMigrate()` 注册点（database.go 第 401 行 Migrate163ADAccountPoolMenu 之后） | `internal/core/db/migrations/migration_162_ad_service_accounts.go` (前序注册样板) + CLAUDE.md「xingran 项目 .sql 迁移文件不会被自动加载」memory（用 `migration_NNN_*.go` 显式注册） | exact |

### 新增辅助（可选，D-03 启动空池校验落点）

| 待改造文件 | Role | Data Flow | 改造点 | Closest Analog | Match Quality |
|------------|------|-----------|--------|----------------|---------------|
| `internal/core/core.go` 或 `cmd/main.go` | bootstrap | startup hook | `initAuthFactory()` 末尾或 `initializeCoreModule` 内 coreModule.Init() 之后 | `core.go:581-588` (现 `accountPool` 创建 + `StartHotReload` 模式) + `account_pool.go:223-250` (`CountByStatus` API) | exact |

### 顺手技术债清理（Open Question 3，planner 决定是否纳入）

| 待改造文件 | Role | Data Flow | 改造点 | Closest Analog | Match Quality |
|------------|------|-----------|--------|----------------|---------------|
| `internal/services/ad_ldap_client.go` | 孤立死代码 | N/A | 整个文件删除（全项目无生产 caller） | — | exact（验证后直接 `rm`） |

---

## Pattern Assignments

### SP-1: FailoverClient 闭包封装（单次操作型） — Wave 1 核心 analog

**Analog 来源:** `internal/core/security/ad_authenticator.go:256-278` + `internal/services/addomain/failover_client.go:33-82`

**适用文件:** `user.go` / `group.go` / `group_management_service.go` / `config.go:TestConnection` / `dept_sync_service.go`（单次操作分支）

**Imports pattern** (ad_authenticator.go:1-15):
```go
import (
    "context"
    "crypto/tls"
    "errors"
    "fmt"

    "github.com/go-ldap/ldap/v3"
    "github.com/xingran-next/xingran-go-backend/internal/models"
    applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
    "github.com/xingran-next/xingran-go-backend/internal/services/addomain"
    "github.com/xingran-next/xingran-go-backend/pkg/ldaputils"
    "gorm.io/gorm"
)
```

**改造前 (user.go:192-210, Enable 操作典型样例):**
```go
// 改造前（同步服务内的单次操作型）:
func (s *UserService) Enable(ctx context.Context, config *models.ADConfig, userDN string) error {
    config.AdminPassword = decryptPassword(config.AdminPassword)  // ← Wave 2 删除

    client := NewLDAPClient(config)                              // ← Wave 1 删除
    if err := client.Connect(); err != nil {                     // ← Wave 1 删除
        return err
    }
    defer client.Close()                                         // ← Wave 1 删除

    if err := client.EnableUser(userDN); err != nil {            // ← Wave 1 包入闭包
        return err
    }

    s.db.WithContext(ctx).Model(&models.ADUser{}).              // DB 操作不动
        Where("ad_config_id = ? AND user_dn = ?", config.ID, userDN).
        Update("is_enabled", true)
    return nil
}
```

**改造后（Wave 1）— 参考 ad_authenticator.go:269-278 的 PickFirstConnect 主路径，单次操作型改用 ExecuteWithFailover:**
```go
// analog: ad_authenticator.go:269-278
func (s *UserService) Enable(ctx context.Context, config *models.ADConfig, userDN string) error {
    fc := NewFailoverClient(s.pool, config)                      // ← struct 字段注入（见 SP-2）
    if err := fc.ExecuteWithFailover(ctx, func(client *LDAPClient) error {
        return client.EnableUser(userDN)                         // LDAP 操作入闭包
    }); err != nil {
        return err                                                // 含 ErrAllAccountsUnavailable
    }
    // DB 操作放闭包外（DB 不依赖 LDAP 连接）
    s.db.WithContext(ctx).Model(&models.ADUser{}).
        Where("ad_config_id = ? AND user_dn = ?", config.ID, userDN).
        Update("is_enabled", true)
    return nil
}
```

**ExecuteWithFailover 完整契约** (failover_client.go:33-82，**不要重新实现**):
```go
// Source: internal/services/addomain/failover_client.go:33-82 (实测签名)
func (f *FailoverClient) ExecuteWithFailover(
    ctx context.Context,
    operation func(client *LDAPClient) error,
) error {
    available, err := f.pool.ListAvailable(ctx, f.config.ID)
    if err != nil { return fmt.Errorf("查询账号池失败: %w", err) }
    if len(available) == 0 {
        return ErrAllAccountsUnavailable                          // ← D-03 运行时空池错误源
    }
    maxAttempts := len(available)
    if maxAttempts > DefaultMaxHops { maxAttempts = DefaultMaxHops }
    var lastErr error
    for i := 0; i < maxAttempts; i++ {
        acct := &available[i]
        client := NewLDAPClient(f.config, acct)
        if err := client.Connect(); err != nil {
            f.pool.MarkFailure(ctx, acct.ID, "dial:"+err.Error())
            lastErr = err; continue
        }
        err := operation(client)                                  // ← caller 闭包在此执行
        client.Close()                                            // ← 关键：闭包返回后立即 Close
        if err == nil {
            f.pool.MarkSuccess(ctx, acct.ID); return nil
        }
        f.pool.MarkFailure(ctx, acct.ID, "operation:"+err.Error())
        lastErr = err
    }
    return fmt.Errorf("账号池 %d 个账号均失败: %w", maxAttempts, lastErr)
}
```

**关键约束（反模式 Pitfall 3）:** `client` 的所有 LDAP 使用必须在闭包内；闭包返回后 `client.Close()` 已调用，闭包外访问 → panic `use of closed network connection`。DB 操作（如 `Update("is_enabled", true)`）可放闭包外。

---

### SP-2: AccountPool struct 字段注入 — Wave 1 改造前提

**Analog 来源:** `internal/core/core.go:566-591` (`initAuthFactory`) + `internal/api/v1/system/ad_domain_router.go:12, 17`

**适用文件:** 全部 Wave 1 同步服务（`SyncService` / `UserService` / `GroupService` / `GroupSyncService` / `GroupManagementService` / `DeptToADSyncService` / `UserADSyncService` / `ConfigService`）

**Analog 1 — core 注入到 AuthFactory (core.go:579-588):**
```go
// Phase 36: 注入 AD 账号池（多账号故障切换）
// 单账号被 AD 锁定（data 775）不再阻断用户登录
accountPool := addomain.NewAccountPool(c.GetDB(), nil) // 无 Redis pub/sub 跨进程广播（单机部署）
c.AuthFactory.SetAccountPool(accountPool)

// Phase 36: 启动 Redis pub/sub 跨进程缓存失效订阅
if err := accountPool.StartHotReload(context.Background()); err != nil {
    applogger.Warnf("启动 AD 账号池热加载失败（不影响主流程）: %v", err)
}
```

**Analog 2 — router 层独立实例化 (ad_domain_router.go:11-18):**
```go
func SetupADDomainRouter(r *gin.RouterGroup, core *core.Core) {
    service := addomainServices.NewADDomainService(core.GetDB(), core.SM4Cipher)
    handler := NewADDomainHandler(service, core)
    syncHandler := NewADUserSyncHandler(core)

    // Phase 36: AD 服务账号池（多账号故障切换）
    accountPool := addomainServices.NewAccountPool(core.GetDB(), nil)
    accountHandler := NewADAccountHandler(accountPool, core)
    ...
}
```

**Authenticator setter 注入 (ad_authenticator.go:47-51):**
```go
// SetAccountPool Phase 36: 注入账号池
// 注入后 Authenticate 将使用 FailoverClient 进行管理员绑定
func (a *ADAuthenticator) SetAccountPool(pool addomain.AccountPool) {
    a.accountPool = pool
}
```

**改造模式（选项 A，planner 选 A 与 analog 一致）:** SyncService struct 增加 `pool AccountPool` 字段，构造函数注入。

```go
// 改造前 (sync.go:18-33):
type SyncService struct {
    db         *gorm.DB
    syncGroup  singleflight.Group
}
func NewSyncService(db *gorm.DB) *SyncService {
    return &SyncService{db: db}
}

// 改造后（Wave 1）:
type SyncService struct {
    db         *gorm.DB
    pool       AccountPool            // ← 新增
    syncGroup  singleflight.Group
}
func NewSyncService(db *gorm.DB, pool AccountPool) *SyncService {
    return &SyncService{db: db, pool: pool}
}
```

**`NewADDomainService` 聚合根同步改签名 (service.go:119-136):**
```go
// analog: service.go:119
func NewADDomainService(db *gorm.DB, pool AccountPool, cipher ...PasswordCipher) *ADDomainService {
    if len(cipher) > 0 && cipher[0] != nil { SetADSM4Cipher(cipher[0]) }
    return &ADDomainService{
        Config:         NewConfigService(db, pool),              // ← 透传 pool
        Sync:           NewSyncService(db, pool),
        OU:             NewOUService(db),
        User:           NewUserService(db, pool),                // ← 透传 pool
        Group:          NewGroupService(db, pool),
        GroupSync:      NewGroupSyncService(db, pool),
        OUGroupMapping: NewOUGroupMappingService(db),
        GroupMgmt:      NewGroupManagementService(db, pool),
        Log:            NewLogService(db),
        Computer:       NewComputerService(db),
    }
}
```

**router 调用点同步改 (ad_domain_router.go:12):**
```go
// 改造前: service := addomainServices.NewADDomainService(core.GetDB(), core.SM4Cipher)
// 改造后: 复用 router 已有的 accountPool 实例（避免重复 New，Pitfall 4 缓存不共享）
accountPool := addomainServices.NewAccountPool(core.GetDB(), nil)
service := addomainServices.NewADDomainService(core.GetDB(), accountPool, core.SM4Cipher)
```

**反模式 Pitfall 4（缓存不共享）:** 每处都 `NewAccountPool(db, nil)` 会创建独立 30s 内存缓存，MarkFailure 后其他实例仍用旧快照 → 账号熔断后仍被选中。**必须**通过 struct 字段注入同一实例。core 已在 `initAuthFactory` 创建（core.go:581），router 应复用 core 的实例（推荐 core 暴露 `GetAccountPool()` getter）或在 router setup 处创建一次后传给 `NewADDomainService`。

---

### SP-3: 批量同步 operation 闭包边界（复用单连接型） — Wave 1 复杂场景

**Analog 来源:** `internal/services/addomain/user_ad_sync_service.go:471-482`（现"建一次连接复用"模式）

**适用文件:** `sync.go:syncDataInternal` / `user_ad_sync_service.go:SyncManagersToAD` / `user_ad_sync_service.go:SyncUserUpdateToAD` (line 59) / `user_ad_sync_service.go:BatchMoveUsersToNewOU` (line 207) / `scheduler/dept_sync_tasks.go:executeDeptToADSyncTask` / `executeDeptMemberToADGroupSyncTask`

**改造前 — SyncManagersToAD 测试钩子优先样例 (user_ad_sync_service.go:471-482):**
```go
// 6. 构造 updateAttr：测试钩子优先，否则建真实 LDAP 连接（复用单连接）
var updateAttr func(string, map[string]string) error
if s.updateUserAttributeFn != nil {
    updateAttr = s.updateUserAttributeFn                                // 测试钩子
} else {
    ldapClient := NewLDAPClient(&adConfig)                              // ← Wave 1 替换
    if err := ldapClient.Connect(); err != nil {
        return nil, fmt.Errorf("连接 AD 失败: %w", err)
    }
    defer ldapClient.Close()
    updateAttr = ldapClient.UpdateUserAttribute                         // 闭包外抽方法 → 危险（client 闭包内才有效）
}
// 后续 g.Go 并发调用 updateAttr(userDN, attrs)
```

**改造后（Wave 1，**operation 闭包边界 = 整个批量任务**，**测试钩子分支保留**）:**
```go
// analog: failover_client.go:33-82 + 现有 user_ad_sync_service.go:471-482
var updateAttr func(string, map[string]string) error
if s.updateUserAttributeFn != nil {
    updateAttr = s.updateUserAttributeFn                                // ⚠️ 测试钩子保持不变（7 个回归测试依赖）
} else {
    fc := NewFailoverClient(s.pool, &adConfig)
    err := fc.ExecuteWithFailover(ctx, func(client *LDAPClient) error {
        updateAttr = client.UpdateUserAttribute                         // 在闭包内赋值
        // 在闭包内启动整个 errgroup 批量（继承 ctx 超时取消）
        g, gctx := errgroup.WithContext(ctx)
        g.SetLimit(constants.MaxConcurrentADSync)
        // ... 遍历 resolved 调 updateAttr（沿用现有 user_ad_sync_service.go:484-517）
        _ = g.Wait()
        return nil  // 即使部分用户失败，账号本身成功（MarkSuccess）
    })
    if err != nil { return nil, err }
}
```

**关键决策（Pitfall 2 + RESEARCH Pattern 2）:**
1. `updateUserAttributeFn` 测试钩子分支**完全保留**，且优先级高于 FailoverClient（钩子非 nil 时绕过 FailoverClient），否则 7 个 `TestSyncManagersToAD_*` 回归测试全失败。
2. operation 闭包边界 = **整个批量任务**（非每用户）。理由：FailoverClient MarkFailure 粒度是账号级；CONTEXT deferred 了"账号失败 vs 用户操作失败"精细化。
3. 用 `errgroup.WithContext(ctx)` 继承调用方 context，避免闭包内 `g.Wait()` 长时间持连接超时（Pitfall 5）。
4. 闭包内 `updateAttr = client.UpdateUserAttribute` —— 注意这是**方法值**，绑定当前 client；闭包返回后 client 已 Close，闭包外再调 `updateAttr` 会 panic（Pitfall 3）。**所有 client 使用必须在闭包内完成**（即 `g.Wait()` 必须在闭包内）。

**改造前 — sync.go:syncDataInternal 一次连接多次 Search (sync.go:90-98):**
```go
config.AdminPassword = decryptPassword(config.AdminPassword)            // ← Wave 2 删除
// 1. LDAP 连接
client := NewLDAPClient(config)                                         // ← Wave 1 替换
if err := client.Connect(); err != nil {
    s.updateSyncLog(ctx, syncLog.ID, models.ADSyncStatusFailed, 0, 0, 0, 0, err.Error())
    return nil, fmt.Errorf("连接AD服务器失败: %w", err)
}
defer client.Close()
result := &SyncResult{}
// 2. 搜索和同步 OU（后续多次 client.SearchOUs / SearchGroups / SearchUsers / SearchComputers）
```

**改造后（sync.go，整个同步流程一个 operation）:**
```go
// analog: failover_client.go:33-82
fc := NewFailoverClient(s.pool, config)
err := fc.ExecuteWithFailover(ctx, func(client *LDAPClient) error {
    // 所有 client.Search* 调用在闭包内
    ous, err := client.SearchOUs(config.BaseDN)
    if err != nil { return err }
    // ...（sync.go:103-153 现有逻辑全部包入）
    return nil
})
if err != nil {
    s.updateSyncLog(ctx, syncLog.ID, models.ADSyncStatusFailed, 0, 0, 0, 0, err.Error())
    return nil, fmt.Errorf("连接AD服务器失败: %w", err)
}
// 注意：syncOUs/syncGroups/syncUsers 等 DB 写入可放闭包外（不依赖 LDAP 连接），
// 但 Search 结果（[]*ldap.Entry）必须在闭包内 collect 后传出
```

---

### SP-4: 配置测试连接（PickFirstConnect） — Wave 1 特例

**Analog 来源:** `internal/core/security/ad_authenticator.go:269-278` (现成模板)

**适用文件:** `internal/services/addomain/config.go:TestConnection` (line 200-217)

**改造前 (config.go:200-217):**
```go
func (s *ConfigService) TestConnection(ctx context.Context, config *models.ADConfig) error {
    applogger.Infof("[ADTest] 开始测试AD连接...")
    config.AdminPassword = decryptPassword(config.AdminPassword)         // ← Wave 2 删除
    client := NewLDAPClient(config)                                      // ← Wave 1 替换
    if err := client.Connect(); err != nil { return err }
    defer client.Close()
    return nil
}
```

**改造后（Wave 1）— 参考 ad_authenticator.go:269-278:**
```go
// analog: ad_authenticator.go:269-278
func (s *ConfigService) TestConnection(ctx context.Context, config *models.ADConfig) error {
    applogger.Infof("[ADTest] 开始测试AD连接: configID=%s", config.ID)
    fc := NewFailoverClient(s.pool, config)
    client, _, err := fc.PickFirstConnect(ctx)                          // 只建连接 + bind，不做操作
    if err != nil {
        return fmt.Errorf("AD 连接测试失败（账号池无可用账号或全部 bind 失败）: %w", err)
    }
    defer client.Close()
    applogger.Infof("[ADTest] LDAP连接测试成功")
    return nil
}
```

**PickFirstConnect 完整契约** (failover_client.go:88-119):
```go
func (f *FailoverClient) PickFirstConnect(ctx context.Context) (*LDAPClient, *models.ADServiceAccount, error) {
    available, err := f.pool.ListAvailable(ctx, f.config.ID)
    if err != nil { return nil, nil, fmt.Errorf("查询账号池失败: %w", err) }
    if len(available) == 0 { return nil, nil, ErrAllAccountsUnavailable }
    // ...顺序遍历，连接成功即 MarkSuccess + 返回（不调 operation）
}
```

**注意:** PickFirstConnect 返回的 `*LDAPClient` 由 caller 负责调 `defer client.Close()`（ad_authenticator.go:275 模式）。这是唯一允许"闭包外用 client"的场景（与 ExecuteWithFailover 不同）。

---

### SP-5: Wave 2 单管理员使用代码删除 + 错误引导文本

**Analog 来源:** `internal/services/addomain/account_pool.go:34-39` (哨兵错误) + `failover_client.go:41-43`

**适用文件:** 全部 Wave 2 改造点（17 处 `decryptPassword(config.AdminPassword)` 删除）

**改造前（统一模式，散落 8 个文件 17 处）:**
```go
config.AdminPassword = decryptPassword(config.AdminPassword)            // ← Wave 2 删除
// 或（scheduler 用大写包名前缀）:
adConfig.AdminPassword = addomain.DecryptPassword(adConfig.AdminPassword) // ← dept_sync_tasks.go:88, 160
```

**改造后（Wave 2，调用 FailoverClient 后捕获 ErrAllAccountsUnavailable 返回引导文本）:**
```go
// analog: failover_client.go:41-43 (ErrAllAccountsUnavailable 哨兵) + RESEARCH Code Examples
err := fc.ExecuteWithFailover(ctx, op)
if errors.Is(err, addomain.ErrAllAccountsUnavailable) {
    return fmt.Errorf("AD 账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号: %w", err)
}
```

**`AdminPassword` 字段在 `tryBindAttempts` 的死代码清理（Wave 2 后）:**
```go
// ldap_client.go:134-145 现状（c.account 非 nil 时已忽略 c.config.AdminPassword）
// Wave 2 后所有 caller 都走 FailoverClient → c.account 必非 nil
// 可简化为（planner 决定是否清理）:
func (c *LDAPClient) tryBindAttempts(conn *ldap.Conn, domain string) error {
    if c.account == nil {
        return fmt.Errorf("账号池模式下 account 不能为空")   // 防御性 panic 替代
    }
    password := decryptPassword(c.account.PasswordCiphertext)
    username := c.account.Username
    // ...（剩余 UPN/NetBIOS/Direct 尝试逻辑不变）
}
```

**`bindAdmin` @Deprecated 死代码清理 (ad_authenticator.go:238-247, Wave 2 可删):** Wave 2 删除 `bindAdminWithFailover` 的 fallback 分支（line 257-267）后，`bindAdmin` 无 caller，整段删除。

---

### SP-6: 前端 admin 字段删除 — Wave 2 前端

**Analog 来源:** `xingran-react-frontend/src/pages/ad-domain/configs/AccountPoolTab.tsx` (Phase 36 已收敛 UI) + `configs/index.tsx:108-114` (Phase 36 已部分清理注释)

**适用文件:** `adDomainApi.ts` (类型) + `configs/index.tsx` (表单提交)

**改造前 — adDomainApi.ts:13-14 (ADConfig 类型):**
```typescript
export interface ADConfig {
  // ...
  adminUsername: string;       // ← Wave 2 删除
  adminPassword?: string;      // ← Wave 2 删除
  // ...
}
```

**改造前 — adDomainApi.ts:155-167 (ADConfigCreateRequest):**
```typescript
export interface ADConfigCreateRequest {
  configName: string;
  serverAddress: string;
  serverPort: number;
  domainName: string;
  baseDn: string;
  adminUsername: string;       // ← Wave 2 删除（后端 binding:"required" 配套删，见 SP-7）
  adminPassword: string;       // ← Wave 2 删除
  useSsl?: boolean;
  // ...
}
```

**改造前 — adDomainApi.ts:169-182 (ADConfigUpdateRequest):**
```typescript
export interface ADConfigUpdateRequest {
  // ...
  adminUsername: string;       // ← Wave 2 删除
  adminPassword?: string;      // ← Wave 2 删除
  // ...
}
```

**改造前 — configs/index.tsx:106-114 (handleEdit 已部分清理):**
```typescript
const handleEdit = (config: ADConfig) => {
    setEditingConfig(config);
    // Phase 36: 不再回填 adminUsername/adminPassword（单管理员字段已废弃）
    editForm.setFieldsValue({
      ...config,
      adminUsername: undefined,      // ← Wave 2 删除（字段不存在后 setFieldsValue 报错）
      adminPassword: undefined,      // ← Wave 2 删除
    });
    setModalVisible(true);
};
```

**改造前 — configs/index.tsx:124-145 (handleSubmit 仍透传 adminPassword):**
```typescript
const handleSubmit = async () => {
    const values = await editForm.validateFields();
    if (editingConfig) {
        const updateData = {
          ...values,
          adminPassword: values.adminPassword || undefined,  // ← Wave 2 删除
        };
        await updateADConfig(editingConfig.id, updateData);
        // ...
    } else {
        await createADConfig(values);   // ← values 不再含 adminUsername/adminPassword
        // ...
    }
};
```

**改造后（Wave 2，直接删除字段引用，无需替代 analog）:**
```typescript
// adDomainApi.ts: 删除 adminUsername/adminPassword 字段
// configs/index.tsx:106-114: handleEdit 简化为
const handleEdit = (config: ADConfig) => {
    setEditingConfig(config);
    editForm.setFieldsValue({ ...config });   // 不再 spread admin 字段（已从类型移除）
    setModalVisible(true);
};
// configs/index.tsx:124-145: handleSubmit 简化为
const updateData = { ...values };             // 不再透传 adminPassword
await updateADConfig(editingConfig.id, updateData);
```

**配套后端删除（SP-7）— config.go:85-86 (`binding:"required"`):**
```go
// 改造前 (config.go:79-92):
type CreateRequest struct {
    // ...
    AdminUsername string `json:"adminUsername" binding:"required,max=255"`   // ← Wave 2 删 required
    AdminPassword string `json:"adminPassword" binding:"required,max=500"`   // ← Wave 2 删 required
    // ...
}
// 改造后（Wave 2）: 整行删除（字段保留在 model 作 DB 列，仅 service 层 request 不再接收）
```

**关键风险（Open Question 2）:** 前端不传 adminUsername → 后端 `binding:"required"` 拒绝 → 400。**必须前后端配套删除**（同一 Wave 2 commit）。

---

### SP-7: migration_164 补迁幂等校验 — Wave 3

**Analog 来源:** `internal/core/db/migrations/migration_162_ad_service_accounts.go:55-68`（现"先 count，>0 则 skip"幂等样板）

**演进说明（B-02 修复）:** CONTEXT D-03 原述扩展 migration_162 校验分支或新增 migration_163；orchestrator 已核实 `migration_163_ad_account_pool_menu.go` 已被 Phase 36 账号池菜单权限占用，且 AutoMigrate 真实注册点在 `internal/core/db/database.go`（不存在 `migrate.go`）。下一个可用序号 = **164**，注册点 = `database.go` 第 401 行（`Migrate163ADAccountPoolMenu` 调用块 `}` 之后）。

**适用文件:** 新增 `migration_164_phase38_verify_admin_migrated.go`（独立 migration，幂等校验，不重写已部署 migration_162/163）

**Analog — 现有幂等样板 (migration_162:55-68):**
```go
// 2. 数据迁移：从 sys_ad_config 拷贝第一行 admin 账号到新表
// 幂等：先查现有账号数，为 0 时才迁移，避免重复插入
var existingCount int64
if err := db.Model(&struct{}{}).Table("sys_ad_service_accounts").
    Where("deleted_at IS NULL").
    Count(&existingCount).Error; err != nil {
    log.Printf("Migration 162: failed to count existing accounts: %v", err)
    return err
}
if existingCount > 0 {
    log.Printf("Migration 162: %d accounts already exist, skipping data migration", existingCount)
    return nil
}
```

**Analog — 迁移 INSERT 字段映射 (migration_162:97-116):**
```go
// 插入新表（status=0 表示可用）
insertSQL := `
INSERT INTO sys_ad_service_accounts (
    id, config_id, username, password_ciphertext,
    status, failure_count, remark,
    created_at, updated_at
) VALUES (
    gen_random_uuid(), ?, ?, ?,
    0, 0, '从 sys_ad_config 迁移（v1.16 兼容期）',
    NOW(), NOW()
)`
if err := db.Exec(insertSQL,
    configRow.ID,
    configRow.AdminUsername,
    configRow.AdminPassword,                // ← 已是 SM4 密文（sys_ad_config 列保留密文）
).Error; err != nil { /* ... */ }
```

**Wave 3 补迁校验模式（参考 analog 写新增 migration_164）:**
```go
// analog: migration_162:55-68 + CLAUDE.md memory「xingran 项目 .sql 迁移文件不会被自动加载」
// 用 migration_NNN_*.go 显式调用并加入 AutoMigrate()
// 演进说明：162 已生产部署，163 已被 Phase 36 菜单占用 → 用独立 164，不重写已部署 migration
func Migrate164Phase38VerifyAdminMigrated(db *gorm.DB) error {
    log.Println("Running migration 164: Phase 38 verify sys_ad_config admin migrated to pool")

    // 幂等校验：对每个启用的 AD config，确认账号池非空（若空且 admin_* 非空，补迁）
    type cfgRow struct {
        ID            string
        AdminUsername string
        AdminPassword string
    }
    var configs []cfgRow
    db.Raw(`SELECT id, admin_username, admin_password FROM sys_ad_config
            WHERE sync_enabled = true AND status = 0
              AND admin_username IS NOT NULL AND admin_password IS NOT NULL`).
       Scan(&configs)

    for _, cfg := range configs {
        var cnt int64
        db.Table("sys_ad_service_accounts").
            Where("config_id = ? AND deleted_at IS NULL", cfg.ID).Count(&cnt)
        if cnt > 0 {
            log.Printf("Migration 164: config %s already has %d accounts, skip", cfg.ID, cnt)
            continue
        }
        // 补迁（与 migration_162:97-116 同款 INSERT，但 config_id 用具体 cfg.ID）
        db.Exec(`INSERT INTO sys_ad_service_accounts
                 (id, config_id, username, password_ciphertext, status, failure_count, remark, created_at, updated_at)
                 VALUES (gen_random_uuid(), ?, ?, ?, 0, 0, 'Phase 38 补迁', NOW(), NOW())`,
            cfg.ID, cfg.AdminUsername, cfg.AdminPassword)
    }
    return nil
}
```

**注册点（B-01 修复）:** AutoMigrate 真实注册点在 `internal/core/db/database.go`（不存在 `internal/core/db/migrations/migrate.go`）。在 database.go 第 401 行（`Migrate163ADAccountPoolMenu(d.DB)` 调用块的闭合 `}` 之后）、第 403 行 `Migrate117CreateMacFilterRules` 之前，追加：
```go
// Phase 38: 验证 sys_ad_config 单管理员账号已迁入账号池（幂等补迁）
if err := migrations.Migrate164Phase38VerifyAdminMigrated(d.DB); err != nil {
    applogger.Errorf("Phase 38 账号池补迁校验失败: %v", err)
}
```
参考 migration_162 现有注册模式（database.go:395-397）。注意 CLAUDE.md memory「xingran-gorm-sql-constraint-naming-conflict」：手写 SQL 索引/约束命名避免与 GORM `uni_*_*` 冲突（本 migration 仅 INSERT 无 DDL，无冲突风险）。

---

### SP-8: 启动空池校验 WARN（D-03，不阻断启动）— Wave 2/3

**Analog 来源:** `internal/core/core.go:566-591` (`initAuthFactory` 启动钩子) + `account_pool.go:223-250` (`CountByStatus`)

**适用落点:** `core.go:initAuthFactory` 末尾 或 `cmd/main.go:initializeCoreModule`（coreModule.Init() 之后）

**改造模式（基于 analog）:**
```go
// analog: core.go:581-588 (StartHotReload 启动模式) + account_pool.go:223-250 (CountByStatus API)
// 落点建议: core.go:initAuthFactory 末尾（accountPool 已在此创建）
func (c *Core) initAuthFactory() {
    // ...现有逻辑...
    accountPool := addomain.NewAccountPool(c.GetDB(), nil)
    c.AuthFactory.SetAccountPool(accountPool)
    if err := accountPool.StartHotReload(context.Background()); err != nil { /* ... */ }

    // Phase 38: 启动空池校验（D-03，WARN 不阻断）
    c.checkEmptyAccountPoolOnStartup(accountPool)

    applogger.Infof("认证策略工厂初始化完成（支持 local/ad/hybrid 模式 + AD 账号池）")
}

func (c *Core) checkEmptyAccountPoolOnStartup(pool addomain.AccountPool) {
    var configs []models.ADConfig
    c.GetDB().Where("status = ? AND sync_enabled = ?", models.ADConfigStatusEnabled, true).Find(&configs)
    ctx := context.Background()
    for _, cfg := range configs {
        total, available, _, _, err := pool.CountByStatus(ctx, cfg.ID)
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

**关键约束（Pitfall 6）:** 仅 `applogger.Warnf`，**不返回 error**，避免阻断新环境首次部署。

---

## Shared Patterns（跨文件复用）

### SHA-1: AccountPool struct 字段注入（Wave 1 前提，所有同步服务）
**Source:** SP-2（core.go:581 + ad_domain_router.go:17 + ad_authenticator.go:47-51）
**Apply to:** `SyncService` / `UserService` / `GroupService` / `GroupSyncService` / `GroupManagementService` / `DeptToADSyncService` / `UserADSyncService` / `ConfigService`
**关键:** 复用同一 `AccountPool` 实例（Pitfall 4），不每方法 `NewAccountPool`。

### SHA-2: FailoverClient operation 闭包封装（Wave 1 全部 21 处）
**Source:** SP-1（failover_client.go:33-82）
**Apply to:** 全部 Wave 1 改造文件（9 个文件 21 处调用点）
**关键:** 所有 `client.*` LDAP 调用必须在闭包内（Pitfall 3）。

### SHA-3: ErrAllAccountsUnavailable 引导文本（D-03 运行时空池错误）
**Source:** SP-5（account_pool.go:35 哨兵错误）
**Apply to:** 所有 Wave 1 改造后的 FailoverClient caller（`errors.Is(err, addomain.ErrAllAccountsUnavailable)` 分支）
**关键:** 错误文本引导用户到"AD 配置页详情 → 服务账号池 Tab"，不泄露账号池内部状态。

### SHA-4: 单管理员字段配套删除（Wave 2，前后端同步）
**Source:** SP-6 + SP-5
**Apply to:**
- Go: `config.go:85-86` (`binding:"required"`) + 108, 169 (`encryptPassword`) + `service.go:29-30, 47`（兼容 shim）+ `models/ad_domain.go:29-32`（struct tag 放宽）
- 前端: `adDomainApi.ts:13-14, 161-162, 175-176` + `configs/index.tsx:92, 108-113, 131`
**关键:** 前后端必须同一 commit（Open Question 2 警告：前端不传 + 后端 required → 400）。

### SHA-5: 测试钩子优先级（仅 SyncManagersToAD）
**Source:** SP-3（user_ad_sync_service.go:471-482）
**Apply to:** 仅 `user_ad_sync_service.go:SyncManagersToAD`
**关键:** `s.updateUserAttributeFn != nil` 时绕过 FailoverClient，保护 7 个 `TestSyncManagersToAD_*` 回归测试。

### SHA-6: migration 幂等（Wave 3）
**Source:** SP-7（migration_162:55-68 样板）
**Apply to:** 新增 migration_164_phase38_verify_admin_migrated（**不是** 163——已被 Phase 36 菜单占用）
**关键:** 先 count 再 insert；手写 SQL 索引/约束命名避免与 GORM `uni_*_*` 冲突（CLAUDE.md memory 警示）；注册点在 `database.go`（不是 `migrate.go`）。

---

## No Analog Found

无。所有 15 个待改造文件均找到现有 analog：
- 9 个 Wave 1 服务文件 → `failover_client.go` + `ad_authenticator.go:269-278`（API 已就绪）
- Wave 2 删除类 → 散落各处的现有 `decryptPassword` / `encryptPassword` / 字段定义本身（删除样板即"找到位置直接删"）
- Wave 3 migration → `migration_162_ad_service_accounts.go` 现有幂等样板
- 前端 → `AccountPoolTab.tsx`（Phase 36 收敛模式）

唯一需 planner 判断的：`internal/services/ad_ldap_client.go` 孤立文件是否纳入清理（Open Question 3，RESEARCH 已验证无生产 caller，建议纳入 Wave 3 但 planner 决定）。

---

## Metadata

**Analog search scope:**
- `internal/services/addomain/` (核心包：failover_client / account_pool / ldap_client / sync / user / group / group_management / group_sync / dept_sync / config / service / utils / user_ad_sync_service)
- `internal/core/security/ad_authenticator.go` (已就位的 FailoverClient 参考样板)
- `internal/core/core.go` (AccountPool 注入 + 启动钩子)
- `internal/api/v1/system/ad_domain_router.go` (router 注入样板)
- `internal/scheduler/dept_sync_tasks.go` + `ad_sync_tasks.go` (scheduler 模式)
- `internal/core/db/migrations/migration_162_ad_service_accounts.go` (幂等迁移样板)
- `internal/models/ad_domain.go` (待清理字段)
- `xingran-react-frontend/src/lib/adDomainApi.ts` + `pages/ad-domain/configs/index.tsx` (前端字段位置)
- `internal/services/ad_ldap_client.go` (孤立文件候选)

**Files scanned:** 18（含 2 个核心 API 文件 failover_client.go + account_pool.go，1 个登录认证 analog ad_authenticator.go，8 个 Wave 1 待改造服务文件，2 个 scheduler 文件，1 个 migration 样板，1 个 model，1 个聚合根 service.go，2 个前端文件，1 个孤立文件）

**Pattern extraction date:** 2026-06-23

**Notes for planner:**
1. Wave 1 前置任务：先做 SP-2（struct 字段注入），再做 SP-1/SP-3/SP-4（闭包封装）。否则改造中服务方法签名无法通过编译。
2. `NewADDomainService` 签名变更会影响 `ad_domain_router.go:12` 和所有 caller（如 `ad_sync_tasks.go` 用的 `NewADDomainService`），planner 需在 Wave 1 同步更新这些 caller。
3. RESEARCH 已实测：21 处 `NewLDAPClient(config)` 无 account 调用点（非 CONTEXT 说的 ~15 处）；17 处 `decryptPassword(config.AdminPassword)`（非 9 处）。Phase gate 用 grep 断言：`grep -rn "NewLDAPClient(config)\|NewLDAPClient(&config)\|NewLDAPClient(&adConfig)" internal/ --include="*.go" | grep -v "_test.go" | grep -v "failover_client.go"` 应为 0。
4. 回归守护（已有，无需新增）：`TestAccountPoolPasswordRoundTrip` / `TestFailoverClient` / `TestSyncManagersToAD_*` (7 个) / `TestDecryptPassword_InvalidCiphertext`。
5. `updateUserAttributeFn` 测试钩子**禁止删除**（7 个回归测试依赖，SHA-5）。
