---
phase: 60
review: 60-REVIEW.md
titles: json
findings:
  - id: CR-01
    severity: critical
    disposition: deferred
    title: "AD bind identity (UPN) and DN-gate identity (sAMAccountName) are resolved by different attributes — SAFE-01 can be bypassed"
  - id: CR-02
    severity: critical
    disposition: deferred
    title: "Locally disabled users can still log in through the AD and hybrid paths (token issued without status check)"
  - id: CR-03
    severity: critical
    disposition: deferred
    title: "`refreshToken` re-issues a full token pair for disabled users"
  - id: WR-01
    severity: warning
    disposition: open
    title: "`getAuthConfig` reuses the same struct for two `First()` calls — `defaultMode` is never read when the AD-enabled row exists (verified)"
  - id: WR-02
    severity: warning
    disposition: open
    title: "AD connection errors are wrapped but compared with `==`, so `ErrADConnectionFailed` never matches"
  - id: WR-03
    severity: warning
    disposition: deferred
    title: "The batch safety gate compares against the stale `sys_ad_user` cache, not the live AD DN"
  - id: WR-04
    severity: warning
    disposition: open
    title: "`NormalizeDN` splits on commas without honoring RFC4514 escapes — equivalent DNs can be judged different, locking legitimate users out"
  - id: WR-05
    severity: warning
    disposition: deferred
    title: "An unguarded legacy soft-delete restore helper remains in the tree and would bypass the shared gate if re-wired"
  - id: IN-01
    severity: info
    disposition: deferred
    title: "Public debug endpoint `GET /test-sm2` is registered unconditionally"
  - id: IN-02
    severity: info
    disposition: deferred
    title: "A single refusal emits three WARN records"
  - id: IN-03
    severity: info
    disposition: deferred
    title: "Policy refusal is returned as HTTP 500"
  - id: IN-04
    severity: info
    disposition: deferred
    title: "No `sys_login_log` entry is written for a policy refusal"
  - id: IN-05
    severity: info
    disposition: open
    title: "The config-read bug is encoded as expected behavior in the test suite"
open: 4
total: 13
recorded: 2026-10-20T09:55:00.000Z
---

# Phase 60: Code Review Disposition

| Finding | Severity | Disposition | Source |
|---------|----------|-------------|--------|
| CR-01 | critical | deferred | 上游身份解析缺陷：bind 用 `username@domain`（ad_authenticator.go:83）、search 用 `(sAMAccountName=username)`（:260），两者可指向不同 AD 对象；本 phase 对该段零改动（判定语义已实现并被 11 条断言锁定），但削弱 SAFE-01 的安全成效上限。去向：新建跟进需求；处置前需先完成 60-UAT.md 人工项 2（本环境 AD 账号 sAMAccountName 与 UPN 本地部分是否一致） |
| CR-02 | critical | deferred | 早于本 phase 的 auth 层缺陷：AD/hybrid 登录不查 `user.Status`（local 路径已查）。Phase 57 CONTEXT 已登记 `internal/api/v1/auth.go` AD 登录路径不查 status 为 Deferred 项。与 SAFE-01 无因果 |
| CR-03 | critical | deferred | 同 CR-02：`refreshToken` 为禁用用户重签完整 token 对（auth.go:416-437），属预存缺陷，与 SAFE-01 无因果 |
| WR-01 | warning | open | - |
| WR-02 | warning | open | - |
| WR-03 | warning | deferred | 本 phase 范围内但 SAFE-02 措辞为「同一输入场景」，成对用例（`TestPhase60_PathConsistency_*`）已证伪；真实环境中 AD 对象移动后登录会拒、批量会放行。去向：跟进项（批量路径改用实时 AD DN 为比对基准）；前置人工项 3（`sys_ad_user.user_dn` 与 AD 实时 DN 一致性） |
| WR-04 | warning | open | - |
| WR-05 | warning | deferred | 遗留 `restoreDeletedUserWithADInfo` 未接共享闸门，但当前无调用路径；若重新接线会绕过闸门。去向：跟进项（删除该死代码或接入 `checkSoftDeletedRestore`） |
| IN-01 | info | deferred | 预存 debug 端点 `GET /test-sm2` 无条件注册，早于本 phase，与 SAFE-01/02 无关。去向：Phase 62 卫生收尾 |
| IN-02 | info | deferred | 单次拒绝产出三条 WARN（`checkSoftDeletedRestore` + `SyncADUser` + `ADAuthenticator`），本 phase 新增了其中一条。去向：跟进项（保留唯一一条最完整的审计 WARN，上层降为 DEBUG） |
| IN-03 | info | deferred | 策略拒绝走 `response.ErrServerError`（500），与「策略拒绝 vs 基础设施故障」分级意图相悖；60-01-PLAN 明确要求该分支保持不变，故记为债务而非规格偏差。去向：跟进项（策略拒绝改 403） |
| IN-04 | info | deferred | 策略拒绝在 `recordLoginLog` 之前返回，缺 `sys_login_log` 审计痕迹。去向：跟进项（拒绝前补记登录日志，文案保持 DN-free） |
| IN-05 | info | open | - |

Dispositions: `open` (recorded, not yet triaged), `fixed`, `skipped`, `deferred`.
Set `deferred` by hand and put the reason in the Source cell; both are preserved.
A `|` in the reason is kept as prose and escaped on the next run.
