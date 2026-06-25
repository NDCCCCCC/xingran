---
slug: ad-connection-ldap-49-invalid-credentials
status: resolved
deferred_to: v1.16-tech-debt
trigger: 连接AD失败: LDAP Result Code 49 "Invalid Credentials": 80090308: LdapErr: DSID-0C0904AE, comment: AcceptSecurityContext error, data 52e, v3839 (尝试: UPN, NetBIOS, 直连)
created: 2026-05-25T00:00:00+08:00
updated: 2026-06-25
session_type: bug
---

# Debug Session: AD Connection LDAP Result Code 49

## Symptoms

### Expected Behavior
AD 同步应该正常工作，系统能够使用配置的 AD 凭据成功绑定到域控制器并执行同步操作。

### Actual Behavior
**定时任务中的"部门到AD同步"任务执行失败**，错误信息：
```
执行任务失败: 连接AD失败: 绑定失败: LDAP Result Code 49 "Invalid Credentials":
80090308: LdapErr: DSID-0C0904AE, comment: AcceptSecurityContext error, data 52e, v3839
(尝试: UPN, NetBIOS, 直连)
```

**但手动AD数据同步正常工作**（从日志确认）：
```
[AD同步] 同步完成! 总耗时: 9.749338s | OU=3306 Group=66 User=8490 Computer=4296
```

### Error Messages
- **错误来源**: `useJobActions.ts:134` - 前端调用任务执行API
- **LDAP Result Code 49**: Invalid Credentials (无效凭据)
- **data 52e**: 具体表示"用户名或密码错误"
- **影响范围**: 仅定时任务中的"部门到AD同步"功能

### Timeline
- **2026-05-25 15:07**: 手动AD数据同步执行成功
- **2026-05-25**: "部门到AD同步"定时任务失败
- **配置情况**: 用户确认系统中只有1个AD配置

### Scope
- **影响范围**: 定时任务 `dept_to_ad_sync` - 部门结构同步到AD OU
- **功能状态**: 手动AD数据同步正常，定时任务失败
- **关键线索**: 两者使用不同的配置获取逻辑

## Current Focus

- **hypothesis**: 定时任务 `dept_to_ad_sync` 使用了错误的AD配置（可能是旧配置或测试配置）
- **next_action**: 验证系统中是否有多个AD配置，检查定时任务的配置选择逻辑
- **test**: 检查数据库中所有AD配置的状态和凭据有效性
- **expecting**: 找到定时任务使用的配置与手动同步使用的配置不同

## Evidence

- timestamp: 2026-05-25T15:07:00+08:00
  source: user_logs
  finding: |
    手动AD数据同步成功执行：
    ```
    [AD同步] 开始同步配置: AD域控主机
    [用户同步] 完成: 创建 0 个, 更新 8490 个, 耗时 2.17s
    [AD同步] 同步完成! 总耗时: 9.749338s | OU=3306 Group=66 User=8490 Computer=4296
    ```
    说明AD连接和凭据本身是正确的。

- timestamp: 2026-05-25T15:10:00+08:00
  source: code_analysis
  finding: |
    定时任务 `executeDeptToADSyncTask` 的配置获取逻辑：
    ```go
    // internal/scheduler/ad_sync_tasks.go:69-74
    var config models.ADConfig
    err := globalADSyncScheduler.db.Where("sync_enabled = ? AND status = ?", 
        true, models.ADConfigStatusEnabled).First(&config).Error
    ```
    
    **问题**: 这里使用 `First()` 方法，如果存在多个启用的AD配置，会随机选择一个！

- timestamp: 2026-05-25T15:10:00+08:00
  source: code_comparison
  finding: |
    **手动AD数据同步**使用指定配置ID：
    ```go
    // internal/api/v1/system/ad_domain_handler.go:263
    result, err := h.service.SyncADData(ctx, id, req.SyncType)
    ```
    
    **定时任务**自动选择配置：
    ```go
    // internal/scheduler/ad_sync_tasks.go:69-74
    var config models.ADConfig
    err := globalADSyncScheduler.db.Where("sync_enabled = ? AND status = ?", 
        true, models.ADConfigStatusEnabled).First(&config).Error
    ```

- timestamp: 2026-05-25T15:10:00+08:00
  source: root_cause_analysis
  finding: |
    **根因**: 虽然用户说系统中只有1个AD配置，但实际上可能存在：
    1. 已禁用的旧配置（`status = 1`）
    2. 测试配置（凭据已过期）
    3. 隐藏的重复配置
    
    定时任务可能选中了这些无效配置，导致认证失败。

## Eliminated

- timestamp: 2026-05-25T15:00:00+08:00
  hypothesis: LDAP客户端绑定逻辑问题
  evidence: 手动AD数据同步使用相同LDAP客户端且成功
  conclusion: LDAP客户端本身没有问题

- timestamp: 2026-05-25T15:00:00+08:00
  hypothesis: 凭据格式问题
  evidence: 用户确认使用标准格式（纯用户名 + 标准域名）
  conclusion: 凭据格式正确

- timestamp: 2026-05-25T15:05:00+08:00
  hypothesis: 两个不同的LDAP客户端实现
  evidence: 所有AD功能都使用 `internal/services/addomain/ldap_client.go`
  conclusion: 只有一个LDAP客户端实现

## Resolution

### Root Cause

**`dept_sync_service.go` 中缺少密码解密步骤**

对比代码发现：
- ✅ **手动AD数据同步** (`sync.go:57`): `config.AdminPassword = decryptPassword(config.AdminPassword)`
- ❌ **部门到AD同步** (`dept_sync_service.go:44`): 直接使用加密的密码创建LDAP客户端

使用加密的密码尝试AD绑定，导致 `LDAP Result Code 49 "Invalid Credentials"` 错误。

### Fix Applied

**文件**: `internal/services/addomain/dept_sync_service.go`
**位置**: 第44行
**修改**: 在创建LDAP客户端前添加密码解密

```go
// 2. 连接LDAP
adConfig.AdminPassword = decryptPassword(adConfig.AdminPassword)  // ✅ 添加密码解密
ldapClient := NewLDAPClient(&adConfig)
```

### Verification

1. ✅ 代码修复已应用
2. ✅ 编译验证通过
3. ⏳ **待验证**: 重启后端服务
4. ⏳ **待验证**: 手动执行"部门到AD同步"定时任务
5. ⏳ **待验证**: 确认任务执行成功且无LDAP认证错误

### Files Changed

- `internal/services/addomain/dept_sync_service.go` (line 44-45) - 添加密码解密调用

### Status

**修复已应用** - 等待重启验证

## Phase 40 Closure (2026-06-25)

### Re-investigation against current codebase

Re-reading `internal/services/addomain/dept_sync_service.go` after Phase 38 Wave 1
(account pool refactor): the original Resolution (在 line 44 添加
`adConfig.AdminPassword = decryptPassword(adConfig.AdminPassword)` 再 `NewLDAPClient`)
**已不再适用** —— 当前代码已重构为走 `FailoverClient.ExecuteWithFailover`:

```go
fc := NewFailoverClient(s.pool, &adConfig)
if err := fc.ExecuteWithFailover(ctx, func(ldapClient *LDAPClient) error { ... }); ...
```

FailoverClient 内部用 `NewLDAPClient(f.config, acct)` 把池内 `*ADServiceAccount`
注入 `LDAPClient`；`ldap_client.go:144-150` 的 `tryBindAttempts` 检测到 `c.account != nil`
时会改走 `decryptPassword(c.account.PasswordCiphertext)` 路径，
密码密文从 `sys_ad_service_accounts.password_ciphertext` 读出并正确解密。

### Conclusion

LDAP 49 invalid credentials 路径在 `dept_sync_service.go` 中已被
Phase 36 (account pool) + Phase 38 Wave 1 (FailoverClient DI) 修复，
本 session 描述的根因不再可达。status 翻 `resolved`。

verification: 当前 dept_sync_service.go 不再直接 `NewLDAPClient(&adConfig)`，密码解密由 FailoverClient/账号池链路保证
files_changed: .planning/debug/ad-connection-ldap-49-invalid-credentials.md
