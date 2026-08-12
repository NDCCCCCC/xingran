# REVIEWS.md — Phase 36 外部 AI Review

**Review 时间**: 2026-06-22
**Reviewers**: claude CLI + opencode CLI (big-pickle)
**审查文件**: account_pool.go, failover_client.go, ad_account_handler.go, migration_162_ad_service_accounts.go, ad_authenticator.go, ldap_client.go

---

## 双 Reviewer 共识问题（严重）

### 🔴 C1（共识）: LDAPClient 仍用 config 字段，没用 account 字段
- **共识度**: claude + opencode 都提到
- **位置**: `ldap_client.go:132-133` (tryBindAttempts)
- **影响**: **Phase 36 核心价值失效** — FailoverClient 遍历多账号时，所有账号都用同一个密码绑定 → 账号池功能形同虚设
- **修复**:
```go
func (c *LDAPClient) tryBindAttempts(conn *ldap.Conn, domain string) error {
    password := c.config.AdminPassword
    username := c.config.AdminUsername
    if c.account != nil {
        password = c.account.PasswordCiphertext
        username = c.account.Username
    }
    // ...
}
```

### 🔴 C2（共识）: MarkSuccess 从未被调用
- **共识度**: claude + opencode 都提到
- **位置**: FailoverClient (成功路径)
- **影响**: failure_count 单向增长永不归零，3 次非连续失败即误触熔断
- **修复**: 在 FailoverClient.Connect() 成功 + operation 成功时都调 MarkSuccess

### 🔴 H5/claude 评论: failover_client.go 注释失实
- "失败已在 client.Bind 内通过 pool 上报（同步）" — LDAPClient 根本没有 pool 引用
- **修复**: 修正注释 + 在 FailoverClient 层显式上报

---

## claude CLI 独有发现

### 🔴 问题 3: RecoverExpiredBreakers cron 未注册
- `account_pool.go:353` 定义了方法但无人调用
- T4.6 待办未实现
- **影响**: 熔断账号 30 分钟不会自动恢复

### 🔴 问题 4: StartHotReload 未被调用
- Redis pub/sub 订阅实现但从未启动
- 多实例部署时失效不跨进程

### 🔴 问题 5: ad_authenticator.bindAdmin 未接入
- `accountPool` 已注入但从未使用
- 单账号架构仍然是 Phase 36 之前的形态

### 🟡 问题 7: SetEnabled 不清除熔断状态
- 停用→启用后，熔断到期时间仍在
- 账号可能处于 status=0 但 circuit_breaker_until 未到期

### 🟡 问题 8: Stats 用 pageSize=9999 拿全集
- 大池子性能差
- 应单独写 Count 查询

---

## opencode CLI 独有发现

### 🟡 H3: Create/Update/Delete/Enable/Disable 缺 RecordWithBody
- 只有 Unlock 用 RecordWithBody
- 其他写操作只调 Record() — operlog 不记录请求 body
- **PLAN DoD 要求写操作都用 RecordWithBody**
- **修复**: 全部改为 RecordWithBody

### 🟡 H4: 权限粒度过粗
- 8 个端点共享 `["ops:ad:config:list", "ops:ad:config:edit"]`
- Create/Delete/Unlock 没有独立权限
- **修复**: 按端点分开中间件

### 🟡 M1: PickAvailable 用 time.Now().UnixNano() 伪随机
- Windows 15ms 精度，高并发同 idx
- **修复**: 用 math/rand + sync.Mutex

### 🟡 M2: strings 占位符
- `_ = strings.TrimSpace` 是占位 import
- **修复**: 移除 import

### 🟡 M3: Update 无法清空 Remark
- `if req.Remark != ""` 无法区分"未传"和"传空"
- **修复**: 用指针或约定 null

### 🟡 M4: InvalidateCache 在事务内部
- 语义上应在事务提交后执行
- 影响小但应修复

---

## 双 Reviewer 评分汇总

| 维度 | claude | opencode | 平均 |
|------|--------|----------|------|
| 架构 | - | 7/10 | 7/10 |
| 安全 | - | 6/10 | 6/10 |
| 可靠性 | - | 4/10 | 4/10 |
| 可维护性 | - | 8/10 | 8/10 |

**opencode 总结论**: "Phase 36 核心价值失效 — LDAPClient 仍绑 config 管理员账号"

---

## 必修优先级（按严重度排序）

| 优先级 | 问题 | 影响 |
|--------|------|------|
| **P0** | C1 LDAPClient 没用 account | 核心功能失效 |
| **P0** | C2 MarkSuccess 未调用 | 失败计数不归零 |
| **P0** | H5 FailoverClient 注释失实 | 误导性代码 |
| **P1** | H3 写操作缺 RecordWithBody | DoD 不达标 |
| **P1** | H4 权限粒度过粗 | 安全风险 |
| **P1** | 问题 3 cron 未注册 | 30min 恢复失效 |
| **P1** | 问题 4 StartHotReload 未调用 | 跨进程缓存不一致 |
| **P2** | 问题 7 SetEnabled 熔断状态 | 边缘场景 bug |
| **P2** | M1/M2/M3/M4 优化项 | 代码质量 |

---

## 下一步建议

1. **立即修复 P0（3 项）** — 这是 Phase 36 核心价值
2. **Wave 4 真正接入** — ad_authenticator.go 切换到 FailoverClient
3. **注册 cron** + **调用 StartHotReload**
4. **P1/P2 在 Wave 7 一并修复**