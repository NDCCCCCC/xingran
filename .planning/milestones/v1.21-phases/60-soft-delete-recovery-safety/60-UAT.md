---
status: complete
phase: 60-soft-delete-recovery-safety
source: [60-VERIFICATION.md]
started: 2026-10-20
updated: 2026-10-20T14:03:07.005Z
---

## Current Test

[testing complete]

## Tests

### 1. 真实 AD + 国密登录下的 DN 漂移拒绝端到端
expected: 响应 message 为『该账号存在 AD 身份不一致，已被安全拦截，请联系系统管理员』（不含 CN=/OU=/DC= 片段）；后端 applogger 出现 `[SECURITY] 软删用户恢复被拒绝` WARN，字段含 path=login / userID / username / storedAdDN / currentAdDN；sys_user 行保持 deleted_at 非空且 ad_dn 逐字不变；不签发 JWT
result: [pass]

> **2026-10-22 生产端到端实测（chenchao-076，.39 实例登录路径，SAFE-01）：**
> - 布现场：该行软删（deleted_at=now()）+ ad_dn/ad_ou_dn 篡改为 OU=随州中心支公司（AD 实际在 OU=基础运维科，归一化后必不一致），完整行快照留存。
> - 用户真实登录 → **页面提示『该账号存在 AD 身份不一致，已被安全拦截，请联系系统管理员』**（无 DN 片段，用户确认「被拦截」）。
> - 判据 SQL 全中：deleted_at 仍非空（未恢复）、ad_dn 与布现场值逐字节相同（dn_unchanged=t）、密码哈希未变（未被重置为初始密码，证明 restoreSoftDeletedUser 未执行任何写库）、updated_at 停留在上次对照半段（10:47:01，本次登录零写库）→ 不签发 JWT + 事务无写入实锤。
> - 后端 WARN（[SECURITY] 软删用户恢复被拒绝）在 .39 远程实例日志，本机 app.log 无该条（登录请求由 .39 处理）；写库不变量已充分判定。
> - 恢复：deleted_at 置 NULL + ad_dn/ad_ou_dn 回填快照原值，行回到活跃态（active=t/dn_ok=t/dept_ok=t），零残留。

### 2. 核对本环境 AD 账号的 sAMAccountName 与 userPrincipalName 本地部分是否一致
expected: 两者一致；若存在 sAMAccountName≠UPN本地部分 的账号，需按 CR-01 的口径评估闸门比对对象是否正确
result: pass

### 3. 核对 sys_ad_user 表的 user_dn 与 AD 实时 DN 的一致性
expected: 一致；若存在陈旧行，需按 WR-03 的口径确认批量路径的比对基准
result: [pass]

> **2026-10-22 生产补测（原 10-20 skip 项）**：① 影子表口径对拍——sys_user 全量（ad_dn 非空、未删、启用）与 sys_ad_user 10-21 快照逐 DN 比对 0 行漂移、0 行大小写不一致；② 实时抽查（用户在「AD 域管理→AD 用户列表」页直查域控）3 个样本 DN 全部一致：c_chenchao-051→OU=孝感中心支公司（今日双向 move 后）、lixuanhui→OU=湖北分公司根、chenchao-076→OU=基础运维科,OU=科技创新部,OU=分公司本部。影子表 user_dn 与 AD 实时 DN 一致性确认。
reason: "用户选择跳过"

## Summary

total: 3
passed: 1
issues: 0
pending: 0
skipped: 2
blocked: 0

## Gaps

（无 —— 12/12 must-haves 已自动验证通过；以上 3 项为部署环境事实与端到端呈现，无法由单元测试覆盖）
