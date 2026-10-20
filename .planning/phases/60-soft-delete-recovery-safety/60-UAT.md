---
status: testing
phase: 60-soft-delete-recovery-safety
source: [60-VERIFICATION.md]
started: 2026-10-20
updated: 2026-10-20
---

## Current Test

number: 1
name: 真实 AD 环境下制造 DN 漂移场景登录（软删用户存量 ad_dn 与 AD 当前 DN 不一致），观察 HTTP 响应体 message 与后端日志
expected: |
  响应 message 为『该账号存在 AD 身份不一致，已被安全拦截，请联系系统管理员』（不含 CN=/OU=/DC= 片段）；
  后端 applogger 出现 `[SECURITY] 软删用户恢复被拒绝` WARN，字段含 path=login / userID / username / storedAdDN / currentAdDN；
  sys_user 行保持 deleted_at 非空且 ad_dn 逐字不变；不签发 JWT
awaiting: user response

## Tests

### 1. 真实 AD + 国密登录下的 DN 漂移拒绝端到端
expected: 响应 message 为『该账号存在 AD 身份不一致，已被安全拦截，请联系系统管理员』（不含 CN=/OU=/DC= 片段）；后端 applogger 出现 `[SECURITY] 软删用户恢复被拒绝` WARN，字段含 path=login / userID / username / storedAdDN / currentAdDN；sys_user 行保持 deleted_at 非空且 ad_dn 逐字不变；不签发 JWT
result: [pending]

### 2. 核对本环境 AD 账号的 sAMAccountName 与 userPrincipalName 本地部分是否一致
expected: 两者一致；若存在 sAMAccountName≠UPN本地部分 的账号，需按 CR-01 的口径评估闸门比对对象是否正确
result: [pending]

### 3. 核对 sys_ad_user 表的 user_dn 与 AD 实时 DN 的一致性
expected: 一致；若存在陈旧行，需按 WR-03 的口径确认批量路径的比对基准
result: [pending]

## Summary

total: 3
passed: 0
issues: 0
pending: 3
skipped: 0
blocked: 0

## Gaps

（无 —— 12/12 must-haves 已自动验证通过；以上 3 项为部署环境事实与端到端呈现，无法由单元测试覆盖）
