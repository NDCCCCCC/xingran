---
phase: 36
slug: ad-account-pool-failover
title: AD 域控服务账号池 + 自动故障切换
status: planning
created: 2026-06-22
milestone: v1.16
priority: P0
---

# Phase 36 — AD 域控服务账号池 + 自动故障切换

## 业务背景

**问题**：当前 AD 集成在 `sys_ad_config` 单行配置里硬编码一个服务账号（如 `ninedrunk`）。
该账号一旦被 AD 锁定（data 775），导致：

1. 所有 AD 同步失败（不影响业务，但数据漂移）
2. **所有用户登录失败**（HTTP 500）— `ad_authenticator.go` 走单管理员绑定

## 设计目标

| 目标 | 实现 |
|------|------|
| 多账号并发可用 | 1~N 个账号保存到独立表 |
| 自动故障切换 | 随机轮询可用账号，失败立即换下一个 |
| 熔断保护 | 连续 3 次失败 → 30 分钟自动熔断 |
| 手动解锁 | 管理员一键解除熔断 |
| 热加载 | Redis pub/sub + 5 分钟兜底刷新 |
| 不增加页面 | 账号池 UI 嵌入现有 AD 配置页 |
| 单权限级别 | 由 AD 侧委派权限，代码不区分 |
| 严格迁移期 | 老字段保留并标 @Deprecated，1 版本后清理 |

## 关键决策（已锁定）

- **表结构**：新建 `sys_ad_service_accounts` 表
- **密码加密**：复用现有 SM4 + 同密钥（不引入新密钥管理）
- **优先级策略**：随机轮询（不设 priority 字段）
- **熔断恢复**：30 分钟定时 + 管理员手动
- **熔断记录保留**：保留行，灰显
- **改造范围**：全栈（登录 + 同步）+ 前端嵌入
- **迁移策略**：严格迁移期（双读兼容）

## 核心数据模型

```sql
CREATE TABLE sys_ad_service_accounts (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id               UUID NOT NULL REFERENCES sys_ad_config(id),
    username                VARCHAR(255) NOT NULL,
    password_ciphertext     TEXT NOT NULL,           -- SM4 复用同密钥（**Issue 16 修复**：原名 encrypted_password 改 password_ciphertext，含 password 关键词，operlog 自动脱敏）
    status                  INT NOT NULL DEFAULT 0,  -- 0=可用 1=停用 2=熔断
    failure_count           INT NOT NULL DEFAULT 0,
    circuit_breaker_until   TIMESTAMP,               -- 熔断到期时间
    last_success_at         TIMESTAMP,
    last_failure_at         TIMESTAMP,
    last_failure_reason     TEXT,
    manual_unlock_reason    TEXT,                    -- 管理员手动解锁原因（Issue 10: 强制审计）
    manual_unlocked_by      VARCHAR(64),             -- 解锁操作者用户名
    manual_unlocked_at      TIMESTAMP,               -- 解锁时间
    remark                  VARCHAR(500),
    created_at, updated_at, deleted_at
);

CREATE INDEX idx_ad_acct_active ON sys_ad_service_accounts(config_id, status)
    WHERE deleted_at IS NULL;
```

### Issue 16 修复：字段命名避开 OPERLOG-03 约束

**问题**：原计划扩展 operlog 敏感关键词 `encrypted_password`，但 `OPERLOG-03` 锁定 "34 个敏感关键词列表不变"，扩展会触发 regression test 失败。

**修复**：将字段名从 `encrypted_password` 改为 `password_ciphertext`：
- ✅ 含 `password` 关键词，operlog 自动脱敏为 `******`
- ✅ 不需要修改 operlog 敏感关键词列表（OPERLOG-03 兼容）
- ✅ 不需要新增 operlog API

**影响范围**：
- Model struct: `PasswordCiphertext string`
- API 请求/响应: `passwordCiphertext` (前端)
- Service 入参: `PasswordCiphertext`
- Migration 字段名

**注意**：如果业务方已有 `encrypted_password` 的引用，需要同步更新所有调用点（首次建立，无历史负担）。

## 核心 Service 接口

```go
type AccountPool interface {
    PickAvailable(ctx, configID) (*ServiceAccount, error)
    ListAvailable(ctx, configID) ([]ServiceAccount, error)        // 纯读，返回可用账号
    ListAll(ctx, configID) ([]ServiceAccount, error)              // 含停用/熔断，供 UI 展示
    Create(ctx, account) error
    Update(ctx, account) error
    Delete(ctx, accountID) error
    MarkSuccess(ctx, accountID) error                              // 行锁 + 事务
    MarkFailure(ctx, accountID, reason) error                     // 行锁 + 事务 + 自动熔断
    ManualUnlock(ctx, accountID, operator, reason string) error   // Issue 10: 强制传原因+操作者
    SetEnabled(ctx, accountID, enabled) error
    RecoverExpiredBreakers(ctx) (int, error)                       // 显式副作用：cron 5min 调用
    StartHotReload(ctx) error
    InvalidateCache(configID string)
}

const (
    FailureThreshold         = 3
    CircuitBreakerDuration   = 30 * time.Minute
    // **Issue 14 修复**：移除 CooldownAfterFailure 常量
    // 不再让失败账号进入"冷却不可用"状态，避免池子全冷导致饥饿
    // 失败计数器仍生效：3 次失败 → 30 分钟熔断
    CacheRefreshInterval     = 5 * time.Minute
    BreakerRecoveryCronSpec  = "*/5 * * * *"      // 每 5 分钟
)
```

## 改造点

| 层 | 文件 |
|----|------|
| Migration | `internal/core/db/migrations/migration_160_create_ad_service_accounts.go` |
| Model | `internal/models/ad_service_account.go` |
| Service | `internal/services/addomain/account_pool.go` |
| LDAP Client | `internal/services/addomain/ldap_client.go` (接收 account 入参 + 失败上报) |
| Authenticator | `internal/core/security/ad_authenticator.go` (用 pool.PickAvailable) |
| Sync Service | `internal/services/addomain/user_ad_sync_service.go` + `dept_sync_service.go` |
| API Handler | `internal/api/v1/system/ad_account_handler.go` |
| Router | `internal/api/router.go` |
| Frontend Tab | `pages/system/ad-domain/config/index.tsx` |
| Frontend API | `lib/api/system.ts` |

## API 端点（全部 POST，与项目约定一致）

```
POST /system/ad-config/accounts/list      列表（分页，body 带 page/pageSize/status）
POST /system/ad-config/accounts/create    新增（body: configId, username, password, remark?）
POST /system/ad-config/accounts/update    更新（body: id, username?, password?, remark?）
POST /system/ad-config/accounts/delete    删除（body: id）
POST /system/ad-config/accounts/enable    启用（body: id）
POST /system/ad-config/accounts/disable   停用（body: id）
POST /system/ad-config/accounts/unlock    立即解锁（body: id）
POST /system/ad-config/accounts/stats     池状态摘要（body: configId?）
```

所有 ID 通过 body 传递，避免 URL path 中特殊字符问题。

## 风险点

| 风险 | 缓解 |
|------|------|
| 熔断后被人工 DB 清掉 | 状态机注释 + 应用层封装 |
| pub/sub 消息丢失 | 5 分钟兜底刷新 |
| LDAP error 49 子码识别错误 | 复用 `extractDataSubcode` |
| 池里只有一个账号且被锁 | 熔断后允许 manual_unlock；UI 醒目提示 |
| SM4 同密钥 | 风险可控（系统已有此密钥） |

## API 权限映射（Issue 7）

| 端点 | 所需权限 | OperType | 模块名 |
|------|---------|----------|--------|
| POST `/accounts/list` | `system:ad-config:query` | — | AD域控配置 |
| POST `/accounts/stats` | `system:ad-config:query` | — | AD域控配置 |
| POST `/accounts/create` | `system:ad-config:add` | `OperTypeCreate` | AD域控配置 |
| POST `/accounts/update` | `system:ad-config:edit` | `OperTypeUpdate` | AD域控配置 |
| POST `/accounts/delete` | `system:ad-config:remove` | `OperTypeDelete` | AD域控配置 |
| POST `/accounts/enable` | `system:ad-config:edit` | `OperTypeEnable` | AD域控配置 |
| POST `/accounts/disable` | `system:ad-config:edit` | `OperTypeDisable` | AD域控配置 |
| POST `/accounts/unlock` | `system:ad-config:edit` | `OperTypeUnlock` | AD域控配置 |

→ **新增 menu + role_menu 权限种子**（migration N），对应 sys_menu 已有"AD域控配置"父菜单。

## OperLog 字段映射（Issue 9）

| 操作 | 记录字段（脱敏后） |
|------|------------------|
| 创建账号 | id / configId / **username** / **encrypted_password=******（强制脱敏）/ remark |
| 更新账号 | id / 变更字段（password 始终脱敏） |
| 删除账号 | id / username |
| 启用/停用 | id / username / newStatus |
| 立即解锁 | id / username / **manual_unlock_reason**（不脱敏，可追溯） / manual_unlocked_by |

→ 写操作必须用 `operlog.RecordWithBody`，**password 字段名命中敏感词自动 `******`**。

### Issue 12 修复：敏感关键词扩展

**问题**：字段名 `encrypted_password` 不命中现有敏感关键词（`password`/`secret`/`key` 等都不匹配下划线连接的 `encrypted_password`），operlog 脱敏失效。

**修复**：在 Wave 1 同一 migration 中扩展敏感关键词列表：

```go
// migration_162_extend_operlog_sensitive_keywords.go
// 在 OperLog 的 sensitiveKeywords 列表中加入 "encrypted_password"
sensitiveKeywords = append(sensitiveKeywords, "encrypted_password")
```

**验证**：在 `internal/utils/operlog/regression_test.go` 加 case 断言 `encrypted_password` 命中脱敏（确保 `OPERLOG-03` 兼容性不破坏）。

→ **风险**：regression_test 锁定了"敏感关键词列表"，新增属于扩展，符合 `OPERLOG-03` "任何新增 API 都是扩展（不替换）" 的约束。

## manualUnlock 强制审计（Issue 10）

**端点变更**：body 必须包含 `reason`（必填）和操作者（从 JWT 自动取 username）。

```typescript
POST /system/ad-config/accounts/unlock
Request: {
  id: string,
  reason: string,        // 必填，≥10 字符（前端校验，后端兜底）
}
Response: { ok: true }
```

后端校验：
```go
if strings.TrimSpace(req.Reason) == "" || len(req.Reason) < 10 {
    return errors.New("解锁原因必填且不少于10字符")
}
// 同时写入 manual_unlocked_by / manual_unlocked_at / manual_unlock_reason 字段
// 走 operlog.RecordWithBody 记录解锁事件
```

### Issue 13 修复：校验下沉到 Service 层

**问题**：原方案校验在 handler，未来若有批量解锁 API 会绕过。

**修复**：把 reason 校验放在 `AccountPool.ManualUnlock` interface 约束里：

```go
type AccountPool interface {
    // ManualUnlock 强制要求 reason ≥10 字符 + operator 非空（service 层 invariant）
    ManualUnlock(ctx context.Context, accountID, operator, reason string) error
}
```

**实现**：
```go
func (p *poolImpl) ManualUnlock(ctx context.Context, accountID, operator, reason string) error {
    // 校验放在 service 层（不依赖调用方）
    operator = strings.TrimSpace(operator)
    reason = strings.TrimSpace(reason)
    if operator == "" {
        return errors.New("operator 必填")
    }
    if reason == "" || len(reason) < 10 {
        return errors.New("解锁原因必填且不少于10字符")
    }
    // ... 原有解锁逻辑 + 写入 manual_unlocked_by / manual_unlocked_at / manual_unlock_reason
}
```

→ handler 只做参数绑定，不重复校验。

## manualUnlock 强制审计（Issue 10）

**端点变更**：body 必须包含 `reason`（必填）和操作者（从 JWT 自动取 username）。

```typescript
POST /system/ad-config/accounts/unlock
Request: {
  id: string,
  reason: string,        // 必填，≥10 字符（前端校验，后端兜底）
}
Response: { ok: true }
```

后端校验：
```go
if strings.TrimSpace(req.Reason) == "" || len(req.Reason) < 10 {
    return errors.New("解锁原因必填且不少于10字符")
}
// 同时写入 manual_unlocked_by / manual_unlocked_at / manual_unlock_reason 字段
// 走 operlog.RecordWithBody 记录解锁事件
```