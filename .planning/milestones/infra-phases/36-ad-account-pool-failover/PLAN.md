# PLAN.md — Phase 36: AD 域控服务账号池 + 自动故障切换

## 任务拆分（Wave-based）

### Wave 1 — 数据基础（Migration + Model + 单元测试）
1. **T1.1** Migration: 建 `sys_ad_service_accounts` 表 + 索引（**字段名 `password_ciphertext`** — Issue 16 修复）
2. **T1.2** Migration: 把 `sys_ad_config.admin_username/admin_password` 拷贝到新表（status=0，1 行）
3. **T1.3** GORM Model: `internal/models/ad_service_account.go`（`PasswordCiphertext` 字段）
4. **T1.4** 单元测试: Model CRUD + 状态机转换

### Wave 2 — 核心 Service（AccountPool + 缓存 + 热加载）
5. **T2.1** `AccountPool` interface + 实现（无 Redis 版）
6. **T2.2** 单元测试: PickAvailable 过滤逻辑（可用/停用/熔断到期/冷却中）
7. **T2.3** 单元测试: MarkFailure 状态流转（3 次失败 → 熔断）
8. **T2.4** 单元测试: ManualUnlock 清除熔断
9. **T2.5** Redis pub/sub 集成: StartHotReload + 兜底 ticker
10. **T2.6** 集成测试: pub/sub 消息触发缓存失效
11. **T2.7** 完整测试矩阵（Issue 15）：见下方"测试矩阵清单"

#### T2.7 测试矩阵清单（Issue 15）

| 编号 | 测试名 | 验证点 | 优先级 |
|------|--------|--------|--------|
| TC1 | TestPickAvailable_EmptyPool | 空池 → ErrAllAccountsUnavailable | P0 |
| TC2 | TestPickAvailable_AllCircuitBroken | 池里 3 个账号全熔断 → 错误 | P0 |
| TC3 | TestListAvailable_BreakerExpired_StillStatus2 | 熔断到期但未恢复 → 不在 available 中 | P0 |
| TC4 | TestMarkFailure_Boundary | 第 1/2/3/4 次失败行为正确 | P0 |
| TC5 | TestMarkFailure_Concurrent | 5 个 goroutine 并发调用，failure_count = 5（验证 Issue 2） | P0 |
| TC6 | TestManualUnlock_ReasonEmpty | reason 为空 → 拒绝 | P1 |
| TC7 | TestManualUnlock_ReasonTooShort | reason 9 字符 → 拒绝 | P1 |
| TC8 | TestManualUnlock_ReasonValid | reason 10 字符 → 接受 | P1 |
| TC9 | TestManualUnlock_OperatorEmpty | operator 为空 → 拒绝（Issue 13） | P1 |
| TC10 | TestRecoverExpiredBreakers_InvalidateCache | 恢复后 InvalidateCache 被调用（Issue 11） | P1 |
| TC11 | TestFailoverClient_MaxHopsDynamic | 池 2 个账号 → 最多 2 次尝试（Issue 3） | P1 |
| TC12 | TestFailoverClient_NoDuplicatePick | 不重复选同一账号（Issue 3） | P2 |
| TC13 | TestPickAvailable_RecentlyFailedNotExcluded | Issue 14：失败账号仍可选 | P0 |
| TC14 | TestMarkFailure_NoRecentlyFailedUntil | Issue 14：不再写 recently_failed_until | P0 |
| TC15 | TestPickAvailable_DisabledExcluded | 停用账号不在 available | P0 |
| TC16 | TestManualUnlock_ClearsCircuitBreaker | 解锁后 status=0, circuit_breaker_until=nil | P0 |
| TC17 | TestPickAvailable_RandomDistribution | 1000 次 PickAvailable 分布均匀 | P2 |
| TC18 | TestRecoverExpiredBreakers_PublishRedis | Issue 17：恢复后通过 pub/sub 广播 | P1 |
| TC19 | TestFailoverClient_MaxHopsConstant | Issue 18：使用 DefaultMaxHops 常量 | P2 |

### Wave 3 — LDAP 客户端改造
11. **T3.1** `LDAPClient` 改为接收 `ServiceAccount` 入参（不再持有整个 config）
12. **T3.2** `Bind()` 失败时调用 `pool.MarkFailure(accountID, reason)`
13. **T3.3** `Bind()` 成功时调用 `pool.MarkSuccess(accountID)`
14. **T3.4** 新增 `FailoverClient`: 包装 LDAPClient 提供多账号尝试语义
15. **T3.5** 单元测试: FailoverClient（mock pool）

### Wave 4 — 业务层接入
16. **T4.1** `ad_authenticator.go` 第 110 行（admin bind）改用 `FailoverClient.ExecuteWithFailover`
17. **T4.2** `user_ad_sync_service.go` 第 53 行（连接 AD）改用 FailoverClient
18. **T4.3** `dept_sync_service.go` 第 51 行（连接 AD）改用 FailoverClient
19. **T4.4** `dept_sync_tasks.go` 第 165 行（cron）改用 FailoverClient
20. **T4.5** 集成测试: 模拟账号被锁 → 走下一个账号成功
21. **T4.6** 注册 cron `RecoverExpiredBreakers`（每 5 分钟调用一次）

### Wave 5 — API 层
21. **T5.1** Handler: `internal/api/v1/system/ad_account_handler.go`（8 个 POST 方法）
22. **T5.2** Router: 注册 8 个 POST 端点（路径 `/accounts/list` `/accounts/create` `/accounts/update` `/accounts/delete` `/accounts/enable` `/accounts/disable` `/accounts/unlock` `/accounts/stats`）
23. **T5.3** OperLog 集成（敏感操作记录）
24. **T5.4** Swagger 注释

**端点契约（全部 POST）**：

```typescript
// 1. 列表（带分页）
POST /system/ad-config/accounts/list
Request:  { page: 1, pageSize: 10, configId?: string, status?: number }
Response: { list: ServiceAccount[], total, current, pageSize }

// 2. 新增
POST /system/ad-config/accounts/create
Request:  { configId, username, password, remark? }
Response: { id }

// 3. 更新
POST /system/ad-config/accounts/update
Request:  { id, username?, password?, remark? }
Response: { ok: true }

// 4. 删除
POST /system/ad-config/accounts/delete
Request:  { id }
Response: { ok: true }

// 5. 启用
POST /system/ad-config/accounts/enable
Request:  { id }
Response: { ok: true }

// 6. 停用
POST /system/ad-config/accounts/disable
Request:  { id }
Response: { ok: true }

// 7. 解锁
POST /system/ad-config/accounts/unlock
Request:  { id }
Response: { ok: true }

// 8. 统计
POST /system/ad-config/accounts/stats
Request:  { configId? }
Response: {
  total: number,
  available: number,
  disabled: number,
  circuitBroken: number,
  currentAccount?: string
}
```

### Wave 6 — 前端
25. **T6.1** API client: `adAccountApi` (8 个方法，全部 POST)
26. **T6.2** AD 配置页加 Tabs: "服务账号池"
27. **T6.3** ProTable + 状态 Tag + 操作列（编辑/解锁/启用停用/删除）
28. **T6.4** 新增/编辑 Modal（**密码字段必须 SM4 加密后传输**——Issue 8）
    - 提交前调用 `sm4Encrypt(password, getSM4Key())` 加密
    - 编辑时如不修改密码，password 字段为空字符串，后端不动
    - 后端 `encrypted_password` 字段存 SM4 密文
29. **T6.5** 解锁/启用停用/删除的二次确认
30. **T6.6** 解锁 Modal：强制填写原因（≥10 字符校验），提交时附带 `reason` 字段（Issue 10）

### Wave 7 — 验证 + 文档
30. **T7.1** `go build ./...` 通过
31. **T7.2** `go test ./...` 通过
32. **T7.3** `npm run type-check` 通过
33. **T7.4** 手动 UAT: 锁住账号 → 业务走下个账号
34. **T7.5** 文档更新: CLAUDE.md 加 AccountPool 说明

## 依赖关系

```
Wave 1 (DB)
   ↓
Wave 2 (Service) ←─── 依赖 Wave 1
   ↓
Wave 3 (LDAP)  ←─── 依赖 Wave 2
   ↓
Wave 4 (业务接入) ←─── 依赖 Wave 3
   ↓
Wave 5 (API)    ←─── 可与 Wave 4 并行（接口先 mock）
   ↓
Wave 6 (前端)   ←─── 依赖 Wave 5（接口契约稳定后）
   ↓
Wave 7 (验证)   ←─── 依赖所有
```

## 关键实现点

### 1. PickAvailable 算法（随机轮询 + 自动过滤，纯读无副作用）

```go
// PickAvailable 随机选一个可用账号
func (p *poolImpl) PickAvailable(ctx context.Context, configID string) (*ServiceAccount, error) {
    accounts, err := p.ListAvailable(ctx, configID)
    if err != nil { return nil, err }
    if len(accounts) == 0 {
        return nil, ErrAllAccountsUnavailable
    }
    return &accounts[rand.Intn(len(accounts))], nil
}

// ListAvailable **纯读**，返回过滤后的可用账号（不修改 DB）
//
// **Issue 14 修复**：移除 recently_failed_until 过滤（不再让冷却期账号饥饿）
// 该字段现在仅用于 UI 展示"上次失败 X 秒前"，不再影响可用性判定
func (p *poolImpl) ListAvailable(ctx context.Context, configID string) ([]ServiceAccount, error) {
    var all []ServiceAccount
    if err := p.db.WithContext(ctx).
        Where("config_id = ? AND deleted_at IS NULL", configID).
        Find(&all).Error; err != nil {
        return nil, err
    }

    now := time.Now()
    available := make([]ServiceAccount, 0, len(all))
    for _, a := range all {
        if a.Status == 1 { continue } // 停用
        if a.Status == 2 {
            // 熔断中：未到期则跳过；到期但**不修改**，由 cron 任务统一恢复
            if a.CircuitBreakerUntil == nil || now.Before(*a.CircuitBreakerUntil) {
                continue
            }
        }
        // **Issue 14**：不再过滤 recently_failed_until
        // 原因：冷却期内账号被排除 → 池子所有账号都在冷却 → 业务中断 → 饥饿
        // 改为：失败计数照常累加（3 次 → 熔断 30min），不再设冷却期
        available = append(available, a)
    }
    return available, nil
}

// RecoverExpiredBreakers **显式副作用**：恢复所有已到期的熔断账号
// 由 cron 任务每 5 分钟调用一次（在 Wave 2 注册）
//
// **Issue 11 修复**：写完后主动 InvalidateCache（本地）
// **Issue 17 修复**：写完后通过 Redis pub/sub 广播，跨进程失效其他实例的本地缓存
func (p *poolImpl) RecoverExpiredBreakers(ctx context.Context) (int, error) {
    now := time.Now()

    // 1. 查询受影响的 configID 列表（用于精准失效缓存）
    var configIDs []string
    if err := p.db.WithContext(ctx).
        Model(&ServiceAccount{}).
        Where("status = ? AND circuit_breaker_until <= ?", 2, now).
        Distinct("config_id").
        Pluck("config_id", &configIDs).Error; err != nil {
        return 0, err
    }

    // 2. 批量恢复
    result := p.db.Model(&ServiceAccount{}).
        Where("status = ? AND circuit_breaker_until <= ?", 2, now).
        Updates(map[string]interface{}{
            "status":                0,
            "circuit_breaker_until": nil,
            "failure_count":         0,
        })
    if result.Error != nil {
        return 0, result.Error
    }

    // 3. 本地缓存失效
    for _, cid := range configIDs {
        p.InvalidateCache(cid)
    }

    // 4. **Issue 17**：Redis pub/sub 广播，其他进程收到后立即失效本地缓存
    // 避免其他进程的旧缓存继续返回"已恢复"的熔断账号
    if len(configIDs) > 0 && p.redisPubSub != nil {
        msg := strings.Join(configIDs, ",")
        if err := p.redisPubSub.Publish(ctx, "ad:account_pool:breaker_recovered", msg); err != nil {
            applogger.Warnf("[ADPool] 广播 breaker 恢复事件失败（不影响主流程）: %v", err)
        }
    }

    return int(result.RowsAffected), nil
}
```

> **设计改进**：原 `ListAvailable` 中隐藏的 DB 写操作已拆出。读路径无副作用，便于单元测试。

### 2. MarkFailure / MarkSuccess（**并发安全 + 行锁**）

```go
// MarkFailure 失败上报，使用悲观锁防止并发丢失更新
//
// **Issue 14 修复**：不再设置 recently_failed_until（冷却期），避免池子饥饿
// 失败计数仍生效：3 次累加 → 30 分钟熔断
func (p *poolImpl) MarkFailure(ctx context.Context, accountID, reason string) error {
    return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        var a ServiceAccount
        // SELECT ... FOR UPDATE 锁定行，防止并发写
        if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
            First(&a, "id = ?", accountID).Error; err != nil {
            return err
        }

        a.FailureCount++
        a.LastFailureAt = ptr(time.Now())
        a.LastFailureReason = reason
        // **Issue 14**：不再设置 RecentlyFailedUntil

        if a.FailureCount >= FailureThreshold {
            a.Status = 2
            a.CircuitBreakerUntil = ptr(time.Now().Add(CircuitBreakerDuration))
            applogger.Warnf("[ADPool] 账号 %s 进入熔断状态 [ID=%s], reason=%s",
                a.Username, a.ID, reason)
        }

        return tx.Save(&a).Error
    })
}

// MarkSuccess 成功上报，同样使用行锁
func (p *poolImpl) MarkSuccess(ctx context.Context, accountID string) error {
    return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        return tx.Clauses(clause.Locking{Strength: "UPDATE"}).
            Model(&ServiceAccount{}).
            Where("id = ?", accountID).
            Updates(map[string]interface{}{
                "failure_count":   0,
                "last_success_at": time.Now(),
            }).Error
    })
}
```

> **设计改进**：用 `SELECT ... FOR UPDATE` + Transaction 包装，确保并发安全。

### 3. FailoverClient（**动态 maxHops**）

```go
type FailoverClient struct {
    pool   AccountPool
    config *models.ADConfig
}

// **Issue 18 修复**：maxHops 改为常量 + 注释说明选择理由
//
// 选择 10 的理由：
//   - 单账号最大连续失败次数 = FailureThreshold = 3
//   - 加 7 次缓冲 = 最多尝试 10 次仍失败即可放弃（避免极端场景下长时间循环）
//   - 实际执行时还会被池大小截断：min(maxHops, len(available))
const DefaultMaxHops = 10

func NewFailoverClient(pool AccountPool, config *models.ADConfig) *FailoverClient {
    return &FailoverClient{pool: pool, config: config}
}

// **Issue 19 修复**：简化实现，移除 triedIDs 防重逻辑
//
// 简化理由：
//   - ListAvailable 已过滤 status≠0 的账号（熔断/停用排除）
//   - MarkFailure 同步触发熔断，列表快照期内账号状态不变
//   - 顺序遍历比 random pick + 防重更简洁，性能更优（cache friendly）
//
// 一致性保证：
//   - ListAvailable 走事务 + SELECT FOR UPDATE（Issue 2 配套）
//   - 期间其他进程的 MarkFailure 会阻塞等待事务提交
//   - 拿到快照后顺序遍历，账号状态稳定
func (f *FailoverClient) ExecuteWithFailover(
    ctx context.Context,
    operation func(client *LDAPClient) error,
) error {
    available, err := f.pool.ListAvailable(ctx, f.config.ID)
    if err != nil {
        return fmt.Errorf("查询账号池失败: %w", err)
    }
    if len(available) == 0 {
        return ErrAllAccountsUnavailable
    }

    // 限制最大尝试次数
    maxAttempts := len(available)
    if maxAttempts > DefaultMaxHops {
        maxAttempts = DefaultMaxHops
    }

    var lastErr error
    for i := 0; i < maxAttempts; i++ {
        acct := available[i]

        client := NewLDAPClient(f.config, &acct, f.pool)
        if err := client.Connect(); err != nil {
            f.pool.MarkFailure(ctx, acct.ID, "dial:"+err.Error())
            lastErr = err
            continue
        }

        err = operation(client)
        client.Close()

        if err == nil {
            return nil
        }
        // 失败已在 client.Bind 内上报（同步）
        lastErr = err
    }
    return fmt.Errorf("账号池 %d 个账号均失败: %w", maxAttempts, lastErr)
}
```

> **设计改进**：动态调整 maxHops + 防重复选中，池子大小自适应。

## 验收标准（DoD）

- [ ] 至少 3 个账号保存可用
- [ ] 锁住账号 1 后，业务自动切到账号 2
- [ ] 锁住后 3 次失败 → 30 分钟熔断
- [ ] 熔断期间 UI 显示灰显但保留
- [ ] 30 分钟后熔断账号自动恢复（cron RecoverExpiredBreakers）
- [ ] **Issue 11**：cron 写完后主动 InvalidateCache（不等 pub/sub 回环）
- [ ] 管理员可一键手动解锁（强制填原因）
- [ ] 增删账号 → 1 分钟内生效（热加载）
- [ ] 单元测试覆盖率 > 80%（Service 层）
- [ ] `go build ./...` + `go test ./...` + `npm run type-check` 全通过
- [ ] 老字段保留并标 @Deprecated，下个 phase 清理
- [ ] **并发安全**：两个并发 MarkFailure 不丢失更新
- [ ] **动态 maxHops**：池子 2 个账号时只尝试 2 次
- [ ] **ListAvailable 纯读**：单元测试可重复断言，无副作用
- [ ] **API 权限**：8 个端点都有权限校验
- [ ] **密码加密**：前端传输前 SM4 加密，后端存密文
- [ ] **解锁审计**：manual_unlock_reason / manual_unlocked_by / manual_unlocked_at 三字段必填且记录到 operlog
- [ ] **敏感字段脱敏**：operlog 中 password 字段自动 `******`
- [ ] **Issue 12**：operlog 中 `encrypted_password` 字段命中脱敏关键词
- [ ] **Issue 13**：service 层 ManualUnlock 强制 reason ≥10 字符 + operator 非空
- [ ] **Issue 14**：失败账号不在冷却期被排除（避免饥饿）；失败计数仍累加触发熔断
- [ ] **Issue 15**：T2.7 测试矩阵 19 个 case 全部通过
- [ ] **Issue 16**：字段名 `password_ciphertext` 含 password 关键词，operlog 自动脱敏（不破坏 OPERLOG-03）
- [ ] **Issue 17**：cron 恢复熔断后通过 Redis pub/sub 跨进程广播，失效其他实例本地缓存
- [ ] **Issue 18**：maxHops 使用 `DefaultMaxHops` 常量（值=10，注释说明选择理由）
- [ ] **Issue 19**：FailoverClient 简化实现（顺序遍历替代 random pick + 防重）；ListAvailable 走事务保证快照一致