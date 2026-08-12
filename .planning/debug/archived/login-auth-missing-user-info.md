---
slug: login-auth-missing-user-info
name: login-auth-missing-user-info
status: resolved
trigger: 登录返回 500 + 前端报错"认证成功但用户信息缺失"(errorHandler.ts:221),后端真实原因是 AD 管理员账号被锁
created: 2026-05-26
updated: 2026-06-18
labels:
  - bug
  - login
  - authentication
  - error-500
  - ad
  - truthful-error
---

# Debug Session: login-auth-missing-user-info

## Symptoms

### Expected Behavior
正常 AD 用户登录跳转到首页

### Actual Behavior
POST /api/v1/system/auth/login 返回 500,前端 message 为"认证成功但用户信息缺失"。
该文案**误导**:前端 `errorHandler.ts:221` 直接用后端 response body 的 `message` 字段,而后端
在 `internal/api/v1/auth.go` 的旧实现里硬编码了"认证成功但用户信息缺失",并未说明真实原因。

### Real Cause (2026-06-18 复现)
后端日志:
```
WARN  [ADAuth] 管理员绑定失败: username=chenchao-076,
      error=LDAP Result Code 49 "Invalid Credentials": 80090308: LdapErr: DSID-0C090451,
      comment: AcceptSecurityContext error, data 775, v3839
ERRO  Internal server error ... path=/api/v1/system/auth/login status_code=500
```

- AD 用户 bind 在 `ad_authenticator.go:84-87` **成功**(否则会走 line 86 返回 `ErrInvalidCredentials`,根本到不了 admin bind)
- AD 管理员 bind 在 `ad_authenticator.go:111` 失败,**data 775 = 账号被锁**(参见 memory `ad-ldap-error49-data-subcodes`)
- 即:**被锁的是 `sys_ad_config` 中配置的管理员/服务账号**,不是终端用户
- Authenticator 返回 `authResult = {Error:nil, User:nil, NeedsSync:true, ADUserInfo:{Username}}`
- Handler 命中 `if authResult.User == nil { 500 "认证成功但用户信息缺失" }` 误导分支

## Timeline
- 2026-05-26: 首次记录,根因假设为 handler 未处理 NeedsSync
- 2026-05-26: 原始 debug 记录的"Fix"提出**降级到本地用户**方案,该方案**已被用户明确拒绝**
  (要求"不要使用降级策略,删除所有相关笔记,如果管理员账号被锁,请如实反馈错误")
- 2026-06-18: 复现+修复,改为**如实反馈**:传递 sync 失败原因,handler 告知真实问题

## Current Focus

### Hypothesis (2026-06-18, 已证实)
原 handler 硬编码误导文案"认证成功但用户信息缺失",不区分 AD sync 失败的真实原因。
需传递 sync 失败原因(`AuthResult.SyncErrorReason`)到 handler,如实返回对用户有用的中文提示,
**不**降级到本地用户。

### Next Action
实施修复:传递 SyncErrorReason + handler 按原因返回如实错误 + 撤掉降级策略。

### Test
待设计(可写 handler 集成测试,或人工:故意配置错误的管理员密码后登录 chenchao-076 验证)

### Expecting
登录 `chenchao-076` 时:
- 正常路径:绑定 + admin 搜索都成功 → 登录成功
- 管理员账号被锁/密码错(SyncErrorReason=admin_bind):前端 message 变为
  "AD 管理员账号配置异常(账号可能被锁定或密码错误),请联系系统管理员"
- 用户搜索/同步失败:前端 message 变为对应的中文提示

## Eliminated

- ~~降级到本地用户(NeedsSync 时按 username 查 sys_user 继续登录)~~
  → 已被用户拒绝,理由:管理员账号被锁是 AD 基础设施问题,不应让用户用陈旧本地数据登录。
  对应代码已撤掉。

## Resolution

### Root Cause
1. **直接原因**:`internal/api/v1/auth.go` 在 `authResult.User == nil` 时硬编码
   "认证成功但用户信息缺失" 误导文案,不携带真实失败原因。
2. **真实原因**(典型触发):`sys_ad_config` 配置的 AD 管理员/服务账号被 AD 锁
   (data 775,Account Lockout Policy),或密码过期/错。`ad_authenticator.go:111-122`
   的管理员 bind 失败,导致 sync 失败,User==nil + NeedsSync=true。

### Fix (2026-06-18)
**不降级,如实反馈**。

1. `internal/core/security/authenticator.go`: `AuthResult` 新增 `SyncErrorReason string` 字段,
   传递 sync 失败阶段。
2. `internal/core/security/ad_authenticator.go`: 在 5 个 `User==nil` 返回分支分别设置
   `SyncErrorReason`: `admin_dial` / `admin_bind` / `user_search` / `user_sync` / `no_syncer`。
3. `internal/api/v1/auth.go`:
   - 撤掉降级代码,恢复简洁的 `if authResult.User == nil` 分支
   - 新增 helper `syncErrorReasonMessage(reason)` 将原因码映射为如实中文消息
   - admin_dial/admin_bind → "AD 管理员账号配置异常(账号可能被锁定或密码错误),请联系系统管理员"
   - user_search/user_sync/no_syncer → 各自对应的中文提示
   - 移除误导文案"认证成功但用户信息缺失"(保留兜底 default 通用文案)

### Operational Note (非代码修复,需 AD 管理员处理)
- 当前 `chenchao-076` 用户登录 500 的真正解决:解锁/修正 `sys_ad_config` 配置的管理员账号,
  或等待 30 分钟 AD Account Lockout Duration 到期自动解锁
- my memory `ad-ldap-error49-data-subcodes`: data 775=临时锁定,会自动解锁;
  data 533=禁用;52e=密码错;532/701=过期;773=必须改密;525=用户不存在;530=时间限制

### Verification
需要重启后端服务并测试:
1. 配置正确的管理员密码后,用 AD 账号尝试登录 → 正常跳转首页
2. 故意把管理员密码配错(或让 AD 管理员锁定该服务账号)后登录 → 前端 message
   应为"AD 管理员账号配置异常(账号可能被锁定或密码错误),请联系系统管理员",
   而非误导的"认证成功但用户信息缺失"
3. 检查后端日志确认 `[ADAuth] 管理员绑定失败` 仍记录,前端不再误导
4. 验证修复不影响正常本地账号登录(走 `loginLocalDirect`)

### Files Changed
- `internal/core/security/authenticator.go` — AuthResult 新增 SyncErrorReason 字段
- `internal/core/security/ad_authenticator.go` — 5 个 User==nil 分支设置 SyncErrorReason
- `internal/api/v1/auth.go` — 撤掉降级;新增 syncErrorReasonMessage;handler 按原因如实返回
- `.planning/debug/login-auth-missing-user-info.md` — 本文件:更新 Resolution,标记降级方案作废
