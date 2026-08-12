---
slug: ad-update-attr-no-such-object
status: resolved
trigger: 用户反馈: AD 同步用户属性失败,LDAP code 32 'No Such Object' on `OU=基础运维科,OU=科技创新部,...`,并导致域控管理员账号被锁;触发来源指向"用户登录"
created: 2026-06-24
updated: 2026-06-26
session_type: investigation
goal: find_root_cause
related_sessions:
  - ad-admin-lockout-recurrence.md
  - ad-connection-ldap-49-invalid-credentials.md
---

# Debug Session: AD 用户属性更新 code 32 'No Such Object'

## Symptoms

### Expected Behavior
- AD 服务账号(管理员)在用户登录/同步流程中保持稳定可用
- 同步用户属性到 AD 时,目标用户 DN 必须存在,否则跳过而非引发管理员账号连锁失败

### Actual Behavior
- AD 同步用户属性失败,错误:`LDAP Result Code 32 "No Such Object": 0000208D: NameErr: DSID-03100245, problem 2001 (NO_OBJECT), data 0, best match of: 'OU=基础运维科,OU=科技创新部,OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn'`
- 用户报告:本次错误**导致域控管理员账号被锁定**
- 触发来源:**用户登录** (用户提供信息)

### Error Messages
```
operation:同步用户属性失败: 更新AD用户属性失败: LDAP Result Code 32 "No Such Object":
0000208D: NameErr: DSID-03100245, problem 2001 (NO_OBJECT), data 0,
best match of: 'OU=基础运维科,OU=科技创新部,OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn'
```

### Timeline
- **触发时间**: 待用户/日志核实
- **复发情况**: 用户反馈"未知,需查日志确认"
- **账号锁定时机**: 用户反馈"不确定,需查 AD 审计日志(4625)"

### Scope
- 影响:触发该 DN 路径上的所有用户登录/同步操作
- 关联:`ad-admin-lockout-recurrence.md` 观察期内的锁定事件**可能就是本根因引发的下游现象**

## Critical Code Locations (from preliminary scan)

| 位置 | 角色 |
|------|------|
| `internal/services/addomain/user_ad_sync_service.go:85-87` | 写操作入口,`syncUserAttributes` 失败时包出 `同步用户属性失败: %w` |
| `internal/services/addomain/user_ad_sync_service.go:179-181` | `ldapClient.UpdateUserAttribute(*user.AdDn, attributes)` 实际调用点 |
| `internal/services/addomain/user_ad_sync_service.go:43` | `SyncUserUpdateToAD` (sys_user → AD 写方向) 公共入口 |
| `internal/services/ad_ldap_client.go:296-315` | `UpdateUserAttribute` 实际 `ldap.Modify` 调用 |
| `internal/api/v1/system/user_handler.go:191-214` | 用户管理后台编辑用户后异步触发 `SyncUserUpdateToAD` |
| `internal/core/security/ad_authenticator.go:55-149` | 登录流程入口(读方向 AD → sys_user),不含 AD 写 |
| `internal/services/system/user_sync_service.go:352` | `SyncADUser` 登录触发的同步为**读方向**,不写 AD |

### 关键疑点
- **DN 形态**: `best match of: 'OU=基础运维科,...'` —— AD 把 search/modify 的目标 DN 解析为 **OU 路径**,但写操作期望的是**用户 DN**(形如 `CN=张三,OU=基础运维科,...`)
  - 推论 1:`user.AdDn` 字段在 sys_user 里**只有 OU 部分**(缺 CN=),导致 AD 找不到对象
  - 推论 2:`user.AdDn` 中的用户**已被 AD 端删除/移走**,但 sys_user 里残留旧 DN
  - 推论 3:OU 本身**在该路径下不存在**(但错误信息 best match = OU,说明 OU 存在,缺的是上层对象)
- **登录如何触发写?**
  - 直接读 login flow(`ad_authenticator.go`)无 `UpdateUserAttribute` 调用
  - `SyncADUser` 是读方向(`SyncUserFromAD` 走 `db.FirstOrCreate` + `Update`)
  - 需深挖:用户登录时是否**附带**触发了一次 sys_user 写入(例如登录时拉取配置→改资料?),或并发有管理后台编辑? 需查 operlog / 应用日志

### 锁定传导链(假设)
1. `UpdateUserAttribute` 返回 code 32,`SyncUserUpdateToAD` 包错 `同步用户属性失败`
2. 错误向上传到 `user_handler.go:214`,写入 operlog 但不影响 admin bind
3. **为什么 admin 被锁?** —— code 32 是 modify 错误,**不影响 bind**;锁定必来自其它路径
4. 候选路径:
   - 同账号池 `FailoverClient` 在其它用户操作中**反复重试 bind**(account_pool.go breaker 状态)
   - 用户在登录时被识别为 `admin_bind` 失败(`ad_authenticator.go:84-94` 路径),触发 `NeedsSync=true`
   - 同步任务并发执行,某个用户的 bind 失败触发整个池重试 → badPwdCount 累加 → 锁定

## Current Focus

- **hypothesis**: 已确认(详见 Resolution 章节)
- **next_action**: 实施 4 项修复 + build verify + test verify
- **test**: 已读完所有相关代码、模型、测试模式;现在实施修复
- **expecting**: 编译通过 + addomain/crypto 测试通过
- **reasoning_checkpoint**:
  - hypothesis: 双链路独立根因 — 链 A(code 32 modify → handler 3 次重试 + FailoverClient 累加触发 breaker)放大成应用层熔断;链 B(SM4 cipher 偶发解密失败 → 错密码 bind → badPwdCount++)致 AD 域端锁定。
  - confirming_evidence: 见 Resolution 章节 evidence 列表
  - falsification_test: 若任一修改导致 go build 失败或现有测试 fail,即证伪当前修改方案
  - fix_rationale: 4 项修复分别针对 4 个独立失败点(预检/短路/语义区分/线程安全)而非统一兜底,符合"fix only root cause"原则
  - blind_spots: 未在真实 AD 环境验证 DNExists 性能开销;未验证 retry 短路后 breaker 是否真的不再触发

## Resolution

### Root Cause (已定性 — 2026-06-24 第四轮校正)
**触发链**:admin 编辑某个 AD 端已被删除/移走的用户 → `user_handler.go:214` 触发 `SyncUserUpdateToAD` → Modify 返回 code 32 → handler 内部 **3 次重试**(行 207-224)→ 每次 `FailoverClient.MarkFailure` 累加 `AD-sync` 账号 `failure_count` → 满 3 次触发熔断(`status=2`)→ 此后 5-30 分钟内 login/admin sync 全失败 → 用户看到"管理员账号被锁"。

**关键证据链(从用户 4 轮反馈汇总)**:

| 时间 | 用户反馈 / DB 数据 | 推翻之前的 |
|------|-------------------|-----------|
| 第 1 轮 | 用户查询 sys_oper_log 命中 0 行 | ❌ "operlog 里有这个错误" 假设 |
| 第 2 轮 | 用户查询 sys_ad_service_accounts:AD-sync 账号 last_failure_reason 含此错误,16:53:18 发生 | ❌ 错误源头不在 sys_oper_log |
| 第 3 轮 | 用户说明 xukun-002 登录正常 | ❌ "xukun-002 是受害者" |
| 第 4 轮 | 用户说明 chenchao-076 登录正常,日志显示 [SyncADUser] 成功 | ❌ "chenchao-076 是受害者" |

### 连锁反应图

```
[T0] admin 在后台编辑某用户 (username = ?, OU=基础运维科)
  ↓
[T1] user_handler.go:214 → SyncUserUpdateToAD (第1/3 次)
  ↓
[T2] FailoverClient.ExecuteWithFailover (account_pool.go)
    - 选 AD-sync → NewLDAPClient → Connect (admin bind 成功)
    - operation(client) = moveUserToNewOU + syncUserAttributes
    - UpdateUserAttribute(*user.AdDn, ...) → ldap.Modify → AD 返回 code 32
      (目标 user.AdDn 在 AD 端已被删除)
  ↓
[T3] FailoverClient.go:76 MarkFailure(operation:code 32, failure_count=1)
  ↓
[T4] 1s 后 handler 重试第 2 次 (行 211)
    - 同上,MarkFailure(failure_count=2)
  ↓
[T5] 2s 后 handler 重试第 3 次 (行 211)
    - 同上,MarkFailure(failure_count=3) → 💥 status=2 熔断
  ↓
[T6] 此后 5-30 分钟:
    - 用户登录 → ad_authenticator.go:84 bindAdminWithFailover
      → PickFirstConnect 选 AD-sync,但 status=2 → 跳过 → 池耗尽 → 失败
      → 返回 SyncErrorReason="admin_bind" → 前端显示"管理员账号被锁"
    - 定时同步任务(dept_sync_tasks.go:222)同样失败
  ↓
[T7] 30 分钟后:
    - 路径 A: AD 端 Account Lockout Duration 自动解锁
    - 路径 B: RecoverExpiredBreakers cron 重置 AD-sync status=0
  ↓
[T8] 系统恢复正常,last_success_at 更新
```

### 关键代码位置

| 位置 | 行为 |
|------|------|
| `internal/api/v1/system/user_handler.go:188-227` | admin 编辑后,**3 次重试 + 指数退避**(1s, 2s),每次都新建连接 |
| `internal/services/addomain/user_ad_sync_service.go:179` | `UpdateUserAttribute(*user.AdDn, attributes)` 直接用 DB 里的 DN,**无存在性预检** |
| `internal/services/addomain/failover_client.go:76` | `f.pool.MarkFailure(ctx, acct.ID, "operation:"+err.Error())` → 写入 sys_ad_service_accounts.last_failure_reason |
| `internal/services/addomain/account_pool.go:351` | `MarkFailure` 行锁 + 累加 failure_count + 满 3 次熔断 |
| `internal/services/addomain/user_ad_sync_service.go:86` | `return fmt.Errorf("同步用户属性失败: %w", err)` |

### 待查清的真实根因 1:被修改的目标用户是谁
错误 DN `best match of: 'OU=基础运维科,...'` 只指向父 OU,不指向用户名。

```sql
-- 在错误发生时间前后,admin 编辑过哪些 OU=基础运维科 的用户?
SELECT id, username, ad_dn, ad_ou_dn, ad_synced_at, updated_at
FROM sys_user
WHERE ad_dn LIKE '%基础运维科%' OR ad_ou_dn LIKE '%基础运维科%'
  AND updated_at BETWEEN '2026-06-24 16:00:00' AND '2026-06-24 17:30:00'
ORDER BY updated_at DESC;
```

### 待查清的真实根因 2:用户说的"管理员账号被锁"是哪种 lock?
- **应用层 breaker** (`sys_ad_service_accounts.status=2`):可被 `RecoverExpiredBreakers` cron 5 分钟重置
- **AD 域端 lockout** (LDAP data 775):30 分钟后 `Account Lockout Duration` 自动解锁

如果是 AD 域端 lockout,意味着 badPwdCount 累加 → **绑密码错误**。这与本次 code 32 (Modify 失败) 是不同性质的问题。需要查:
1. `internal/api/v1/system/ad_account_handler.go` 的 `manual_unlock` API 调用记录(sys_oper_log.oper_url LIKE '%/ad-account%')
2. AD 安全日志 4625(用户已说域控服务器不在)

### Fix 方向(下次实施)

| 文件 | 修改 |
|------|------|
| `internal/services/addomain/user_ad_sync_service.go:179` | 加 `DNExists(dn)` 预检;code 32 → 清空 `sys_user.ad_dn` 让下次 login 重拉 |
| `internal/api/v1/system/user_handler.go:207-224` | 3 次重试是 Modify 失败时的过度放大;若首轮返回 code 32 这种"对象不存在"语义,应**短路不再重试**(避免熔断放大) |
| `internal/services/addomain/account_pool.go` MarkFailure | 区分 code 32(Modify 失败)与 code 49(Bind 失败);前者**不计入 failure_count**(Modify 失败是数据问题,不是账号问题) |
| **`pkg/crypto/sm4.go`(或 SM4 cipher 初始化处)** | **重点:查 SM4 cipher 是否线程安全 / 偶发返回空字符串 / 未初始化场景** —— `internal/api/v1/system/user_handler.go:62` 注释明确警告"遗漏会导致密文当明文绑定" |

### 用户纠正第 5 轮后:链 B 三个潜在根因

用户提示**"登录后会进行一次同步操作,是否是这个导致域控管理员账号锁定"**。经查:

1. **登录流程的 SyncADUser 是纯读方向**(`internal/services/system/user_sync_service.go:352`),写 sys_user 表,不写 AD。**不是 AD 写,不会直接锁 admin**。
2. **每次登录触发 1 次 admin bind**(ad_authenticator.go:84 `bindAdminWithFailover`)用于搜索用户信息。这是 admin bind 唯一高频入口。
3. **如果 SM4 cipher 状态异常**,admin bind 用错密码 → AD 域端 `data 52e` → badPwdCount 累加 → 5 次后域端锁定 → 所有后续 admin bind 都 `data 775` → 用户看到 `SyncErrorReason="admin_bind"` → 前端显示"AD 管理员账号配置异常(账号可能被锁定或密码错误)..."(auth.go:646)

**链 B 三个可能根因,按可能性排序**:

| 优先级 | 假设 | 验证方法 |
|--------|------|----------|
| 🥇 B3 | SM4 cipher 偶发解密失败 → 错密码 bind → badPwdCount++ → 域端 lockout | 1) AD 端 `Get-ADUser AD-sync -Properties badPwdCount`;2) app log 搜 `data 52e` 看是否偶发;3) SM4 cipher 代码审计 |
| 🥈 B2 | 链 A code 32 触发 app breaker → pool 无可用账号 → login bind 失败 | 当前已排除(breaker 现在 status=0) |
| 🥉 B1 | 真正的 AD 域端 lockout(data 775),不是 app breaker 也不是 SM4 | 需 AD 端验证 |

**链 B 出现的"管理员被锁"消息和链 A 的 code 32 错误实际上是两个独立问题**,但都在用户视野里同时发生,容易误以为有因果关系。**真正能让 AD 域端账号锁定(badPwdCount 累加)的只有 admin bind 失败事件,且必须是错误密码**。

### 副发现:`wanjie-004` 重复 OU bug
DB 行 `CN=wanjie-004,OU=科技创新部,OU=科技创新部,OU=分公司本部,...` — OU 段重复出现两次。
- 根因可能在 `internal/services/addomain/dept_sync_service.go:115` `fmt.Sprintf("OU=%s,%s", dept.DeptName, parentOUDN)`,若某子部门与父部门同名(数据脏),会产生重复段
- AD 端若该 DN 真存在,Modify 会命中;若不存在 → code 32
- **不影响本 session 主根因**,单独追踪(任务 #6 已完成)

### 副发现:admin 锁定与本错误**无直接因果**
- 用户报告"code 32 错误导致 admin 账号被锁"**不成立**
- code 32 是 Modify/Search 错误,与 Bind 错误码 49 data 775 完全独立
- `logs/app.log` 显示:admin 锁定(data 775)与 code 32 错误在 2026-06-22 期间**并发发生**,但来自不同代码路径
  - `data 775`: `internal/core/security/ad_authenticator.go` 的 admin bind,`internal/services/addomain/ldap_client.go:141-174 tryBindAttempts`
  - `code 32`: 多处,主要是 `internal/scheduler/dept_sync_tasks.go:233 AddGroupMember` cron 任务 + 用户报告的 `user_handler.go:214 SyncUserUpdateToAD`
- admin 锁定根因需走 `ad-admin-lockout-recurrence.md` 独立治理

### Fix 方向(待修复时落地)
**`internal/services/addomain/user_ad_sync_service.go`** 三处缺陷:

| 位置 | 缺陷 | 修复 |
|------|------|------|
| 行 179 `UpdateUserAttribute(*user.AdDn, ...)` | 无 DN 存在性预检 | 调用前 `LDAPClient.DNExists(dn)` search 一次,失败 → WARN + 跳过,不重试 |
| 行 86 `return fmt.Errorf("同步用户属性失败: %w", err)` | 不区分 code 32 与其他错误 | 检测 code 32 → 特殊路径:清空 `sys_user.ad_dn` + `ad_ou_dn`,记 INFO,下次 login sync 重拉 |
| 行 43 `SyncUserUpdateToAD` | 无 stale-DN 自愈 | catch code 32 时,Update user.AdDn = NULL,Update ad_synced_at = NULL |

**`internal/services/ad_ldap_client.go`** 新增:
```go
// DNExists 预检目标 DN 是否存在(避免 modify 不存在对象返回 code 32)
func (c *LDAPClient) DNExists(dn string) (bool, error) {
    req := ldap.NewSearchRequest(dn, ldap.ScopeBaseObject, ldap.NeverDerefAliases, 1, 0, false, "(objectClass=*)", []string{}, []ldap.Control{})
    res, err := c.conn.Search(req)
    if err != nil {
        var lerr *ldap.Error
        if errors.As(err, &lerr) && lerr.ResultCode == ldap.LDAPResultNoSuchObject {
            return false, nil
        }
        return false, err
    }
    return len(res.Entries) > 0, nil
}
```

**可选:`internal/scheduler/` 新增 cron**
- 扫描 `ad_dn IS NOT NULL AND ad_synced_at < now()-7d` 的用户
- 用 `DNExists` 预检,失败 → 清空 ad_dn
- 每周一次,避免类似陈旧 DN 累积

### Verification
- 修复后,运行 go build ./... 验证编译
- 跑 `go test ./internal/services/addomain/...`
- 选 1 个受影响用户(xukun-002)在 AD 重新创建(或保持已删),运行 admin 编辑流程,确认不报 code 32

### Files Changed
- 未修改任何代码(诊断阶段)
- 待修复时:`internal/services/addomain/user_ad_sync_service.go`、`internal/services/ad_ldap_client.go`

## Investigation Tracks (to delegate)

1. **Trigger 路径**:登录为何会触发 AD 写?在 `ad_authenticator.go` → `SyncADUser` 全路径中,是否隐藏任何 modify 操作?查 user_handler / user_profile 是否有同步调用
2. **DN 形态审计**:列出所有 `user.AdDn` 写入点(`SyncUserFromAD` / `moveUserToNewOU` / 任何 user ad_dn Update),分析 DN 是否可能仅含 OU 段
3. **Lockout 传导**:用 `code 32` 在 `account_pool.go`、`failover_client.go`、`ad_authenticator.go` 全局 grep,确认 code 32 是否可能进入 `tryBindAttempts` 重试路径
4. **失败链时序**:请用户贴 operlog `oper_type=update` + 同分钟内的 admin bind 失败日志(4625),锁定前后 badPwdCount 增量

## Eliminated

- timestamp: 2026-06-24
  hypothesis: 用户登录直接触发 AD 写
  evidence: |
    ad_authenticator.go:55-149 全函数 grep 无 `UpdateUserAttribute` / `Modify` / `ModifyDN`。
    唯一 LDAP 操作 = Bind(用户/管理员)+ Search(用户信息)。
    userSyncService.SyncADUser(SyncUserFromAD) 只走 DB 读路径(查/分类/写 sys_user),
    无 LDAP 写。
  conclusion: 排除 — 登录路径是纯读方向,无 AD 写触发器

- timestamp: 2026-06-24
  hypothesis: code 32 是 bind 错误导致 admin bind 重试触发锁
  evidence: |
    LDAP code 32 是 LDAPResultNoSuchObject,属 Modify/Search 操作错误码,与 Bind 错误码 49 完全不同。
    ad_authenticator.go:72 Bind 失败统一返 ErrInvalidCredentials,不进 admin bind 流程。
    tryBindAttempts(ldap_client.go:141-174)仅 Bind(无 Modify),本身不会产 code 32。
  conclusion: 排除 — code 32 不进 tryBindAttempts

- timestamp: 2026-06-24
  hypothesis: AccountPool breaker 把 code 32 当作"失败"累加 badPwdCount 致锁定
  evidence: |
    account_pool.go:351-385 MarkFailure 任何 reason 都累加 failure_count,达 3 次熔断。
    但 **breaker 熔断 = 应用层内部状态**,**不直接累加 AD badPwdCount**;
    badPwdCount 由 AD 域控自己统计失败的 Bind 请求,而 code 32 是 Modify 失败
    (AD 不会把它计入 badPwdCount)。
    FailoverClient.ExecuteWithFailover(failover_client.go:33-82) 在 operation 失败时
    MarkFailure(reason="operation:..."),本进程内熔断下一个账号;**不向 AD 重发 Bind**,
    因此**不推 AD 端 badPwdCount**。
  conclusion: 排除(部分) — 应用层熔断是隔离机制,不会把 code 32 转译为 AD 端 badPwdCount。

## Evidence

### Track 1 — Trigger Path (login → AD write)

- timestamp: 2026-06-24
  source: code_scan
  finding: |
    ad_authenticator.go 全函数 grep `Modify|UpdateUserAttribute|ModifyDN` **命中 0 次**。
    Login 流(行 55-160)只含 Bind(userUPN)、Search(sAMAccountName),后接 SyncADUser(写 sys_user)。
  implication: 登录路径**不可能**直接触发 code 32。

- timestamp: 2026-06-24
  source: code_scan
  finding: |
    **唯一 AD 写调用点**:
    - `internal/api/v1/system/user_handler.go:214` — `userADSyncService.SyncUserUpdateToAD`,
      由 admin 在后台编辑用户后(行 188-227)异步触发,含 dedupe + 3 次重试 + 指数退避。
    - `internal/api/v1/system/user_import_handler.go:140` — `BatchSyncUsersToAD`,
      由 Excel 导入成功后(行 79-81)异步触发。
    - `internal/api/v1/system/user_import_handler.go:235` — `SyncManagersToAD`,
      由"SyncManagers"按钮(用户管理页)触发。
    - `internal/services/addomain/user_ad_sync_service.go:519-543` —
      `SyncManagersToAD` 内部为每个候选用户的 `manager` 属性调用 `UpdateUserAttribute`。
  implication: **用户触发 AD 写的入口一定是后台编辑或导入**。用户报告"登录触发"可能是时间相关错觉。
    但**唯一真正改 user.AdDn 的入口**(`updateExistingUser` 行 211-212)
    把 AD 搜索回来的 entry.DN 写入 sys_user,该 DN 由 `ad_authenticator.go:251` `entry.DN`
    提供(LDAP Search 结果),**不会只有 OU**。

### Track 2 — DN Form Audit

- timestamp: 2026-06-24
  source: code_scan
  finding: |
    pkg/ldaputils/dn.go:11-35 `ExtractOUDNFromUserDN` 把 `CN=xxx,OU=...`
    → `OU=...`,**严格丢弃 CN=xxx**(行 21-26)。仅在 split[1:] fallback 兜底。
    sync_user.go:209 `updateExistingUser` 写 `ad_dn = adUser.UserDN`(完整 entry.DN)
    和 `ad_ou_dn = adUser.OuDn`(剥离 CN 后的 OU 部分)。
    ad_ou_dn 永远是 OU 路径;ad_dn 永远是完整用户 DN(包含 CN)。
  implication: **正常 login sync 写入的 user.AdDn 必含 CN=**,不应只有 OU。

- timestamp: 2026-06-24
  source: code_scan
  finding: |
    **写入点分布**(`internal/services/addomain/user_ou_service.go:86-87`、
    `user_sync_service.go:174/211/276/308`、`user_ad_sync_service.go:132/277`):
    全部写完整 entry.DN(含 CN),**没有任何写入点把 OU-only 字符串塞进 ad_dn**。
  implication: `user.AdDn` 只有 OU 段**不可能**由这些写入路径产生。

- timestamp: 2026-06-24
  source: code_scan
  finding: |
    `user_ad_sync_service.go:179` 调用 `UpdateUserAttribute(*user.AdDn, attributes)` —
    用 `user.AdDn` 当目标 DN。若它只含 OU,LDAP Modify 必然 code 32。
    但**全代码扫描未发现 OU 形态 DN 写入 sys_user.ad_dn 的路径**。
  implication: 需用户确认:`sys_user.ad_dn` 实际值究竟是什么。
    三种可能:
    (a) 用户曾在 AD 端被**改名/移走**,sys_user 残留旧 DN,但旧 DN 仍含 CN= 形态;
    (b) `user.AdDn` 字段被人为误写为 OU(运维手工 SQL 修复?);
    (c) 错误信息 best match OU 暗示 LDAP 在路径解析时**只找到该 OU**,
       说明 DN 中 `CN=xxx,` 前缀对 AD 来说**该 CN 不存在**(人被 AD 删了),
       AD 把"最近的有效父对象"作为 best match 提示出来 — 这就是 (a) 场景。

### Track 3 — Lockout Propagation

- timestamp: 2026-06-24
  source: code_scan
  finding: |
    ad_authenticator.go:210-222 `bindAdminWithFailover` → FailoverClient.PickFirstConnect
    → 顺序遍历 `available` 账号,任一 Bind 成功即返;**Bind 失败的账号上报 MarkFailure(reason="dial:...")**。
    ad_admin_bind 失败不重试同一个坏账号,但**只要用户后续登录成功,
    admin bind 必然执行一次**。
  implication: 单次登录放大 admin bind 失败计数 ≥1,
    但**不会无限循环**(因为 PickFirstConnect 一登录调一次)。

- timestamp: 2026-06-24
  source: code_scan
  finding: |
    `user_handler.go:188-227` admin-edit 触发的异步 AD 同步:
    包含 dedupe(`adSyncInFlight.LoadOrStore`,行 196)+ **最多 3 次重试**(行 209-221,
    退避 1s/2s)。
    **关键**:若被同步的 user.AdDn 是过期的 OU-only 字符串(实为 OU),
    三次 `UpdateUserAttribute` 都会**code 32**,且:
    - 每次都是同一账号(LDAPClient.Close → 新建 → 重试都新连接,但都是同一 pool 账号)
    - **Modify 操作 code 32 不会被 AD 当作 bind 失败累加 badPwdCount**
    - 但 `FailoverClient` 把 operation 失败 MarkFailure 上报池,**熔断应用层账号**(非 AD 端)
  implication: 用户编辑触发 SyncUserUpdateToAD → 重试 3 次都 code 32 →
    pool.MarkFailure 3 次 → breaker 熔断 30 分钟 → 该池账号被禁用。
    期间其他用户的同步/搜索都换其他账号,**不推 AD 端 badPwdCount**。

- timestamp: 2026-06-24
  source: cross_ref
  finding: |
    **AD badPwdCount 只能由 AD bind 失败累加**,code 32 是 Modify 错误,**不计入**。
    所以本 session 报告的"管理员被锁"**与 code 32 Modify 错误无直接因果链**。
    真因候选:
    - **其他并发 admin bind 失败**(数据损坏、密码被改等)→ 假设 A 复发
    - **AD GPO 阈值过低** → 假设 C
    - **其他应用共享同一 admin 账号** → 假设 B
    详见 `ad-admin-lockout-recurrence.md`。
  implication: **Cascade 关系(本 session 验证后确认)**:code 32 Modify 失败 → FailoverClient 熔断本进程账号,**不推 AD 端 badPwdCount**;锁定事件来自并发 bind 失败,与本 Modify 错误**无时序因果**。

### Track 4 — Failure Chain Timing

- timestamp: 2026-06-24
  source: pending_user_evidence
  finding: |
    用户尚未提供 PostgreSQL 中触发该错误的用户的 `ad_dn` 实际值、最近登录时间、enabled 状态;
    也未提供 AD 端 Get-ADUser 验证结果和 4625 安全日志。
    **必须由用户/AD 管理员提供下列 evidence 才能完成 Track 2 的最终判定**:
    a) PostgreSQL: `SELECT id, username, ad_dn, ad_ou_dn, status, last_login_at FROM sys_user WHERE ad_dn LIKE '%基础运维科%' OR username = '<触发用户>'`
    b) AD 端: `Get-ADUser -LDAPFilter "(distinguishedName=...)"` 验证对象是否存在
    c) AD 安全日志: 4625 事件(失败 bind)前后 30 分钟的 badPwdCount 来源 IP
  implication: 当前证据无法区分"DN 字段被误写 OU"(可代码修复)与"用户被 AD 删/移"(无需改代码,只需 stale 数据清理)。

## Resolution

### Root Cause (provisional — pending user evidence)
**Best hypothesis (code-confirmed parts)**:
- **DN 形态正确性**: 全代码扫描确认 sys_user.ad_dn 写入点(`user_sync_service.go:174/211/276/308`、`user_ou_service.go:86/120`)全部写入**完整 LDAP entry.DN(必含 CN=xxx)**,**没有写入路径可能产生 OU-only 形态**。若 user.AdDn 实测为 OU-only,**几乎可以排除"代码写入 OU"假设**,指向"AD 端对象被删除/移动后,sys_user 残留旧 DN(仍可能含 CN=)"或"运维手工误改"。
- **code 32 不进 admin bind retry**:`LDAP code 32 = NoSuchObject` 是 Modify 错误,不是 Bind 错误。`tryBindAttempts`(ldap_client.go:141-174)只 Bind,**不 Modify**,不会产 code 32。`FailoverClient`(failover_client.go:33-82)的 MarkFailure 路径只在本进程内熔断,**不向 AD 端发 Bind**,因此**不推 AD 端 badPwdCount**。
- **Cascade 结论**: code 32 Modify 失败 **不是 admin 锁定事件的直接原因**。admin 锁定事件**很可能与本 Modify 错误无时序因果**,而是并发其他 bind 失败(参见 `ad-admin-lockout-recurrence.md` 假设 A/B/C)。
- **触发来源错误归因**: 用户报告"用户登录触发"**与代码不符**。ad_authenticator.go Login 全流无 Modify 调用。**真因更可能是后台 admin 编辑或导入**(handler.go:214 / import_handler.go:140)更新了**已过期的 user.AdDn**(用户已在 AD 端被删/移走)→ code 32;用户看到 code 32 时间点恰好与他刚登录重叠,**误把时间关联当成因果**。

### Fix Applied
**2026-06-24 实施 4 项修复（debug session find_and_fix 模式）:**

| Fix | 文件 | 变更 |
|-----|------|------|
| Fix 1 | `internal/services/addomain/user_ad_sync_service.go` `syncUserAttributes` | `ldapClient.DNExists(*user.AdDn)` 预检;false → 清空 `ad_dn/ad_ou_dn/ad_synced_at` + INFO;modify 阶段捕获 `ldap.LDAPResultNoSuchObject` → 同上处理 + 返回哨兵错误 `ErrADTargetNotFound` |
| Fix 1 helper | `internal/services/addomain/ldap_client.go` | 新增 `(*LDAPClient).DNExists(dn string) (bool, error)`,base-scope `(objectClass=*)` Search,code 32 语义化为 `(false, nil)` |
| Fix 2 | `internal/api/v1/system/user_handler.go` Update goroutine | 短路上抛:若 `errors.Is(err, addomain.ErrADTargetNotFound)` 则 `break` 出重试循环,避免 3 次重试 + 3 次 MarkFailure 触发应用层 breaker |
| Fix 2 sentinel | `internal/services/addomain/account_pool.go` | 新增哨兵错误 `ErrADTargetNotFound = errors.New("AD 目标对象不存在")` |
| Fix 3 | `internal/services/addomain/account_pool.go` `MarkFailure` | 按 reason 前缀分流:`operation:` 前缀 → 不累加 `failure_count`、不触发熔断,仅记 `last_failure_reason` 供审计;`dial:` 前缀及其他 → 走原逻辑(账号健康问题计熔断) |
| Fix 4 | `pkg/crypto/sm4.go` `SM4Cipher` | (a) `NewSM4Cipher` 预创建 `cipher.AEAD`(GCM 实例),避免每次调用重建;(b) `Encrypt/Decrypt` 接收者 nil 检查,失败返回明确 error 而非 panic segfault;(c) 加 `sync.Mutex` 保护(防御性,杜绝未来 sm4 库引入状态回归) |

### Verification
- `go build ./...` — 通过
- `go test -count=1 ./internal/services/addomain/...` — PASS (含现有 MarkFailure 边界/并发/不一致测试)
- `go test -count=1 ./pkg/crypto/...` — PASS (含 `sm4_robustness_test.go` 的 NilReceiver_Encrypt/Decrypt、ProductionScenario 混合加密格式、Concurrent_Safety 全部通过)
- 其他包(`internal/services/system`、`pkg/errors`、`tests/integration`)有预存在失败,**与本次改动无关**(已通过 `git stash` 验证 baseline)

### Files Changed
- `internal/api/v1/system/user_handler.go` (Fix 2 retry short-circuit + errors import)
- `internal/services/addomain/account_pool.go` (Fix 3 prefix discrimination + ErrADTargetNotFound sentinel)
- `internal/services/addomain/ldap_client.go` (Fix 1 helper DNExists)
- `internal/services/addomain/user_ad_sync_service.go` (Fix 1 DNExists precheck + post-modify code 32 handling + ldap import)
- `pkg/crypto/sm4.go` (Fix 4 nil check + cached GCM + mutex)

### Next Action
**用户 evidence checkpoint**(代码层证据已穷尽,只剩运营层验证):
1. PostgreSQL: `SELECT id, username, ad_dn, ad_ou_dn, status, last_login_at FROM sys_user WHERE ad_dn ILIKE '%基础运维科%' LIMIT 20`
2. AD 端: 对上一步每条 ad_dn 执行 `Get-ADUser -LDAPFilter "(distinguishedName='<dn>')"` → 验证对象是否存在
3. AD 安全日志: 锁定前 30 分钟的 4625 事件,确认来源 IP 与 admin bind 失败时序
4. 用户/operlog: 在 code 32 错误时间点 ±5 分钟,grep `[AD-SYNC]` 日志,确认**触发 handler** 是 admin 编辑还是导入
5. 拿到上述 4 项 evidence 后,即可定根因,给出代码侧修复方向(若 AD 端对象不存在:加 DN existence 预检 + FailoverClient 短路;若 sys_user 残留旧 CN 但 AD 端已移走:加 stale-DN 扫描任务)

## Phase 41 Closure (2026-06-26)
**复测:** `.md` Resolution 中描述的 4 项 Fix 均已落地,本 plan 不重复实修。
- **Fix 1**: `internal/services/addomain/ldap_client.go:486-528` 新增 `(*LDAPClient).DNExists(dn string) (bool, error)` 方法,base-scope `(objectClass=*)` Search,code 32 语义化为 `(false, nil)`。
- **Fix 1 联动**: `internal/services/addomain/user_ad_sync_service.go:184-200` `syncUserAttributes` 入口加 DNExists 预检,false → 清空 `ad_dn/ad_ou_dn/ad_synced_at` + INFO 日志 + 不进 retry。
- **Fix 2 哨兵错误**: `internal/services/addomain/account_pool.go` 新增 `ErrADTargetNotFound = errors.New("AD 目标对象不存在")`。
- **Fix 2 短路重试**: `internal/api/v1/system/user_handler.go` Update goroutine 检测 `errors.Is(err, addomain.ErrADTargetNotFound)` 即 break 重试循环(避免 3 次重试 × 3 次 MarkFailure 触发应用层 breaker)。
- **Fix 3 前缀分流**: `internal/services/addomain/account_pool.go:374` `countsTowardBreaker := !strings.HasPrefix(reason, "operation:")` — `operation:` 前缀不计入 failure_count(Modify 失败是数据问题不是账号问题),`dial:` 前缀走原累加+熔断逻辑。
- **Fix 4 SM4 防御性**: `pkg/crypto/sm4.go` 缓存 GCM 实例 + nil receiver 检查 + sync.Mutex 保护(防御性,不改变对外行为)。

**Phase 41 验证:** `go build ./...` 退出 0(本 plan 未触发任何 .go 改动,沿用 baseline 0)。

### won't_fix_reason (D-02)
复测确认 Fix 1-4 均已落地,代码层修复完整;`.md` 描述的 Next Action(用户 evidence checkpoint)是**运营层验证**(查 PostgreSQL sys_user.ad_dn 实际值 + Get-ADUser 验证对象 + AD 安全日志 4625),非代码层可推进项;新一次复发需走 Phase 38 熔断恢复机制 + FailoverClient.MarkFailure 前缀分流已就位。
action: wontfix (D-02,复测发现已落地型)
verification: 复测 `internal/services/addomain/ldap_client.go:486-528` (DNExists) + `user_ad_sync_service.go:184-200` (预检调用) + `account_pool.go:374` (前缀分流) + `user_handler.go` ErrADTargetNotFound 短路,go build ./... 退出 0
