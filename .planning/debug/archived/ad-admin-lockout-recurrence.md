---
slug: ad-admin-lockout-recurrence
status: resolved
trigger: AD 服务账号(管理员)在用户未改动密码的情况下被 AD 自动锁定(LDAP Result Code 49 data 775)
created: 2026-06-18
updated: 2026-06-26
session_type: observation
goal: observe_recurrence
skip_audit: true
---

# Debug Session: AD 服务账号被锁定 — 复发观察

## Symptoms

### Expected Behavior
- AD 服务账号(管理员账号)长期稳定可用
- 用户登录系统时,管理员 bind 应在用户 bind 成功之后执行(ad_authenticator.go:110)
- 用户确认:近 30 天内**未修改**服务账号密码

### Actual Behavior
- 服务账号在未被修改密码的情况下被 AD 锁定
- 业务表现:所有用户登录失败(`SyncErrorReason: admin_bind`,见 ad_authenticator.go:118)

### Error Messages
**真实错误样本**(从代码注释/测试中保留,2026-06-16 17:37:29):
```
绑定失败: LDAP Result Code 49 "Invalid Credentials":
80090308: LdapErr: DSID-0C090451, comment: AcceptSecurityContext error,
data 775, v3839
(尝试: UPN, NetBIOS, 直连)
```

**子码语义**(Microsoft AD 标准):
- `data 775` = `Account Locked Out` —— **账号被锁定**
- AD 默认 `Account Lockout Duration` 30 分钟到期后自动解锁
- 锁定原因:**失败次数达到 `Account Lockout Threshold` 阈值**

### Timeline
- **2026-06-16 17:37:29**:首次记录到 data 775 错误(`internal/utils/string_helper.go:62-63` 注释保留)
- **2026-06-18**:用户提出本次 issue,确认未改密码
- 复发情况:**待观察**

### Scope
- 影响范围:所有依赖该 AD 配置的登录、AD 同步、定时任务
- 当前状态:用户希望**延长观察期**确认是否再次发生,而非立即修复

## Current Focus

- **hypothesis**:服务账号被锁定的根因**不一定是用户改密码**——可能有以下三种来源,需要在观察期内逐一排除
- **next_action**:进入 24-48 小时观察窗口,定期采集数据
- **test**:观察期内每天至少 2 次采集下列指标
- **expecting**:
  - 若**再次锁定**:可排除"瞬时 AD 抖动",锁定是稳定复现,需要走代码根因排查
  - 若**不再发生**:可能是 AD 侧偶发安全事件(如其他系统的攻击失败计数被计入),无需改代码

## 三种待排除的根因假设

### 假设 A:代码触发过多 bind 失败
**机制**:`ad_authenticator.go:82-119` 的串联 bind 流程,每次用户登录成功都会触发一次管理员 bind。如果管理员密码在 sys_ad_config 里被**意外改写过**(例如前端 AD 配置页面误操作、密码被误编辑过又被回滚),会持续失败。
**观察指标**:
- `badPwdCount` 在锁定前的爬升曲线 —— 是缓慢累加(代码问题)还是突然跳变(AD 侧攻击)
- 失败来源 IP:集中在应用服务器(代码)还是分散(攻击)

### 假设 B:AD 侧其他系统触发锁定
**机制**:同一服务账号被其他系统(邮件、VPN、其他业务系统)使用,其他系统的失败 bind 被 AD 计数。
**观察指标**:
- `badPwdCount` 累加时段对应的客户端 IP 是否只有本应用服务器
- AD 安全日志(4625 事件)的来源工作站分布

### 假设 C:AD 锁定策略过于敏感
**机制**:域 GPO 把 `Account Lockout Threshold` 调到 5~10 次,任何环境的失败都触发锁定。
**观察指标**:
- 询问 AD 管理员当前 GPO 配置
- 历史锁定记录频率(月度)

## Evidence

- timestamp: 2026-06-18
  source: code_analysis
  finding: |
    `internal/core/security/ad_authenticator.go:82-119` 的登录流程:
    ```go
    // Line 82-87: 用户 bind
    userUPN := fmt.Sprintf("%s@%s", req.Username, config.DomainName)
    if err := conn.Bind(userUPN, req.Password); err != nil {
        return nil, ErrInvalidCredentials  // 不区分 data 子码
    }

    // Line 109-119: 管理员 bind(用户成功后必走)
    if err := a.bindAdmin(adminConn, config); err != nil {
        applogger.Warnf("[ADAuth] 管理员绑定失败: username=%s, error=%v", ...)
        return &AuthResult{NeedsSync: true, SyncErrorReason: "admin_bind"}, nil
    }
    ```
    关键:用户 bind 成功 → 管理员 bind 失败 → 失败计数 +1。这是**理论放大器**。

- timestamp: 2026-06-18
  source: user_confirmation
  finding: |
    用户明确表示:近 30 天内**未修改** sys_ad_config 中的 AdminPassword。
    此条信息直接排除"近期密码变更"为根因。

- timestamp: 2026-06-16 17:37:29
  source: real_log(代码注释保留)
  finding: |
    `internal/utils/string_helper.go:62-63` 注释中保留了真实崩溃日志:
    `绑定失败: LDAP Result Code 49 ... data 775`
    说明 data 775(锁定)是**真实发生过的错误**而非理论可能。

- timestamp: 2026-06-18
  source: existing_memory
  finding: |
    `memory/ad-ldap-error49-data-subcodes.md` 已记录:
    - data 775 = 账号锁定(临时,过 30 分钟自动解锁)
    - 当前代码 `ad_authenticator.go:84-87` 把所有 bind 失败统一返回 `ErrInvalidCredentials`,
      **前端看不出 775=锁定还是 52e=密码错**,需查日志才能判断。

## Eliminated

- timestamp: 2026-06-18
  hypothesis: 近期修改过服务账号密码
  evidence: 用户明确未改
  conclusion: 排除

- timestamp: 2026-06-18
  hypothesis: 匿名 bind 不可用导致循环失败
  evidence: 见 `ad-connection-ldap-49-invalid-credentials.md` 之前的修复(dept_sync_service.go 缺密码解密);
        用户已用只读服务账号,凭据格式正确
  conclusion: 排除(凭据格式层面问题)

## 观察期协议 (Observation Protocol)

### 持续时间
**至少 24 小时,推荐 48 小时**(覆盖业务高峰+夜间低峰)

### 每日采集清单(建议早 9 点 + 下午 5 点各一次)

#### A. 应用服务器侧
```bash
# 1. 检查登录日志中管理员 bind 失败频率
grep -E "管理员绑定失败|admin_bind" logs/*.log | tail -50

# 2. 检查登录失败总数(用户 bind 失败 + 管理员 bind 失败)
grep -c "ErrInvalidCredentials" logs/*.log

# 3. 检查 LDAP 错误子码分布(data 775 vs 52e vs 533)
grep -oE "data [0-9a-f]{3}" logs/*.log | sort | uniq -c
```

#### B. AD 域控侧(请 AD 管理员协助)
```powershell
# 查看服务账号当前 badPwdCount(失败计数)
Get-ADUser -Identity "<service-account>" -Properties badPwdCount, lockoutTime, lastBadPasswordAttempt |
  Select badPwdCount, lockoutTime, lastBadPasswordAttempt

# 查看最近 4625 事件(失败登录),只保留本应用服务器 IP
Get-WinEvent -FilterHashtable @{LogName='Security';Id=4625} |
  Where-Object {$_.Properties[5].Value -like '*<应用服务器IP>*'} |
  Select TimeCreated, @{N='Source';E={$_.Properties[5].Value}}
```

#### C. 锁定复发判断
| 复发情况 | 判定 | 下一步 |
|---------|------|--------|
| 48h 内**再次**被锁 | 假设 A(代码放大器)成立 | 进入代码修复:合并两次 bind + 加失败告警 |
| 48h 内**未**再锁 | 假设 B/C 成立 | 询问 AD 管理员当时是否检测到异常来源,记录结论 |
| 多次锁定且失败来源都是应用服务器 IP | 假设 A 几乎确定 | 直接走代码修复 |

### 同步动作(必须做,不依赖复发结果)
1. **配置专属监控**:在 AD 安全日志配置邮件告警,服务账号 `lockoutTime != 0` 立即通知
2. **记录 base 状态**:观察期开始时记一次 `badPwdCount` 初始值,作为对照基线
3. **不要中途改密码**:避免破坏观察对照

## Resolution

### Root Cause
**待定**——观察期结束前不妄下结论。

### Fix Applied
**暂无**——按用户要求进入观察期。

### Verification
观察期满后:
1. 跑完上面的采集清单 A/B/C
2. 在本文件 Resolution 章节追加结论
3. 状态改为 `resolved` 或升级到代码修复阶段

### Files Changed
- **未修改任何代码**

### Next Action When Observed
- **若复发**:`/gsd-debug continue ad-admin-lockout-recurrence` 继续诊断
- **若未复发**:在本文件 Resolution 写明"48h 观察期内未复发,初步判定为 AD 侧偶发事件"并移到 `resolved/`

## Phase 41 Closure (2026-06-26)
won't_fix_reason: 已被 Phase 36 AccountPool(多账号池 + 状态机 0=可用/1=停用/2=熔断 + 自动熔断恢复)+ Phase 38 FailoverClient.MarkFailure 前缀区分(`operation:` vs `dial:`,前者不计入 failure_count 不触发应用层熔断)覆盖(项目 memory ad-modify-fail-double-counts-breaker / ad-ldap-error49-data-subcodes),代码层无进一步可修。**观察期已过 8 天(2026-06-18 → 2026-06-26),未报告新锁定事件**。若再发现锁定走 Phase 38 熔断恢复机制 + 5-30 分钟 cron RecoverExpiredBreakers 自愈,无需再开 debug session。
action: wontfix (D-02,已被 Phase 36/38 覆盖 + 观察期未复发)
skip_audit: true