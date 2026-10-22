---
phase: "60"
slug: "soft-delete-recovery-safety"
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: "2026-10-20"
---

# Phase 60 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| AD 登录请求 → 本地同步 | 外部已通过 bind 的用户名与 LDAP 返回的 DN 进入本地恢复决策 | 用户名 + AD DN（不可信外部值） |
| sys_user 存量行 → 恢复判定 | 软删行中的 ad_dn 是不可信历史值，必须与 AD 当前 DN 比对 | 存量 ad_dn vs AD UserDN |
| 服务层拒绝 → 登录/批量响应 | 拒绝细节（DN）不得越过该边界进入用户可见文案 | 固定中文文案（DN-free） |
| 批量结果 → 前端结果面板 | `Skipped` 与 `Errors` 明细直接返回浏览器，不能携带 DN | 批量记账（DN-free） |
| 策略拒绝 → 批次事务管道 | 拒绝若进入 error 通道会被降级重试并计入 Failed | 拒绝短路（不进 error 通道） |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-60-01 | Elevation of Privilege | `restoreSoftDeletedUser` / `checkSoftDeletedRestore` | high | mitigate | `user_sync_service.go:272` 共享入口 `NormalizeDN` 比对存量 ad_dn 与 AD 当前 DN，不一致拒绝；拒绝后 `authResult.User=nil` 不签发 JWT（`TestPhase60_Login_MismatchedDN_Refuses` 等锁定） | closed |
| T-60-02 | Tampering | 拒绝分支写库路径 | high | mitigate | 校验函数只读且位于 `HashPassword`（:317）与任何 Update 之前；`TestPhase60_Login_RefusalAuditWarn` 断言拒绝后 `deleted_at` 非空且 `ad_dn` 逐字不变、二次登录不能变空 DN 绕过 | closed |
| T-60-03 | Information Disclosure | `auth.go syncErrorReasonMessage` / `SyncError.Error` | medium | mitigate | 用户文案固定中文（『该账号存在 AD 身份不一致，已被安全拦截，请联系系统管理员』）且不含 DN；`TestAuth8003_SyncErrorReasonMessage` 锁定；DN 细节只进 applogger WARN | closed |
| T-60-04 | Repudiation | 登录失败日志级别 | low | mitigate | `ad_authenticator.go:130-144` 策略拒绝用 `Warnf("[SECURITY] ...")`，基础设施故障用 `Errorf`，`errors.Is(ErrSoftDeleteIdentityMismatch)` 区分 | closed |
| T-60-05 | Elevation of Privilege | `BatchSyncADUsers` 阶段 2 | high | mitigate | 与登录路径调用同一 `checkSoftDeletedRestore`（:594）；不一致项从恢复分类移除，永不进入恢复写库事务（`TestPhase60_Batch_MismatchedDN_Skipped`） | closed |
| T-60-06 | Tampering | 批量结果记账 | medium | mitigate | 被拒项 `result.Skipped++`（全文件唯一）并短路，不抛 error 不入 `runInBatchedTx` 降级管道；`TestPhase60_Batch_MixedUsers_Conservation` 断言 `Total == Success+Failed+Skipped` | closed |
| T-60-07 | Information Disclosure | `SyncError.Error` 响应字段 | medium | mitigate | 批量错误文本固定为不含 DN 的中文；DN 细节只进 `applogger.Warnf`（path=batch），批量用例断言 | closed |
| T-60-SC | Tampering | npm/pip/cargo installs | low | accept | 本 phase 零新增依赖、不改 go.mod/go.sum（commits 仅触碰 4 个实现文件 + 测试 + planning）；无供应链安装面 | closed |

*Status: closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| RISK-60-SC | T-60-SC | 本 phase 零新增依赖，无供应链安装面 | planner (60-01/60-02 threat_model) | 2026-10-20 |
| RISK-60-A4 | T-60-02 (adjacent) | 阶段 2 分类快照与阶段 5c 写库之间的窄 TOCTOU 窗口（RESEARCH A4）：本 phase 不引入事务内重读，窗口内的并发 AD 对象改名不在此闸门防护范围 | verifier 确认为 documented accepted risk | 2026-10-20 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-10-20 | 8 | 8 | 0 | orchestrator (ASVS L1 grep-depth; evidence: verifier 独立复跑 + 静态守卫) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-10-20
