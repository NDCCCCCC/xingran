---
phase: 60-soft-delete-recovery-safety
verified: 2026-10-20T09:20:34Z
updated: 2026-10-22
status: human_needed
score: 12/12 must-haves verified
covered_files:
  - ".planning/phases/60-soft-delete-recovery-safety/60-01-PLAN.md"
  - ".planning/phases/60-soft-delete-recovery-safety/60-01-SUMMARY.md"
  - ".planning/phases/60-soft-delete-recovery-safety/60-02-PLAN.md"
  - ".planning/phases/60-soft-delete-recovery-safety/60-02-SUMMARY.md"
  - "internal/api/v1/auth.go"
  - "internal/api/v1/auth_80_03_test.go"
  - "internal/core/security/ad_authenticator.go"
  - "internal/core/security/ad_authenticator_phase60_test.go"
  - "internal/core/security/authenticator.go"
  - "internal/services/system/user_sync_service.go"
  - "internal/services/system/user_sync_service_phase60_test.go"
covered_digest: "v3:sha256:8d37669b66ac996932c4498ef3e83ac0513d4992ec568581d0bbca55a7426adb"
behavior_unverified: 0
overrides_applied: 0
advisory:
  - finding: "ROADMAP.md 里程碑阶段清单第 44 行 Phase 60 仍为 `- [ ]`，而 Progress Table 已写 `2/2 | Complete | 2026-10-20` —— 同一文件内自相矛盾"
    category: other
    reason: "60-02 Task 2 的验收口径只要求 Progress Table 行与两个 plan 复选框回填，未覆盖里程碑清单行；属文档卫生，不影响 SAFE-01/SAFE-02 的代码达成"
    evidence_status: "none provided"
  - finding: "60-REVIEW-DISPOSITION.md 中 13 条 review 发现全部为 `open`，无一条完成 triage（CR-01~CR-03 尤其未定去向）"
    category: other
    reason: "phase 结束时审查发现未被分流；其中 CR-01 直接关系到 SAFE-01 的安全成效（见「开放风险与升级项」），建议在 phase 收尾前补 triage"
    evidence_status: "none provided"
human_verification:
  - test: "真实 AD 环境下制造 DN 漂移场景登录（软删用户存量 ad_dn 与 AD 当前 DN 不一致），观察 HTTP 响应体 message 与后端日志"
    expected: "响应 message 为『该账号存在 AD 身份不一致，已被安全拦截，请联系系统管理员』（不含 CN=/OU=/DC= 片段）；后端 applogger 出现 `[SECURITY] 软删用户恢复被拒绝` WARN，字段含 path=login / userID / username / storedAdDN / currentAdDN；sys_user 行保持 deleted_at 非空且 ad_dn 逐字不变；不签发 JWT"
    why_human: "60-VALIDATION.md Manual-Only 表保留项；需要真实 AD 域控与国密（SM2/SM4）加密登录请求，端到端 HTTP 响应无法由 sqlite 单元测试覆盖（服务层原因码映射与文案映射已分别由 TestPhase60_SyncErrorReasonFor / TestAuth8003_SyncErrorReasonMessage 锁定）"
    resolution: "已转留 60-UAT.md Test 1（用户选择跳过；真实 AD + 国密登录下的 DN 漂移拒绝端到端仍待实测，结果记录在该文件）"
  - test: "核对本环境 AD 账号的 sAMAccountName 与 userPrincipalName 本地部分是否一致（抽查若干在职账号）"
    expected: "两者一致；若存在 sAMAccountName≠UPN本地部分 的账号，需按「开放风险与升级项」CR-01 的口径评估闸门比对对象是否正确"
    why_human: "AD 侧账号属性形态属部署环境事实，代码库不可见；这是 SAFE-01 安全成效的前置条件"
    resolution: "已在 60-UAT.md Test 2 实测通过（2026-10-20，抽查 sAMAccountName 与 userPrincipalName 本地部分一致）"
  - test: "核对 sys_ad_user 表的 user_dn 与 AD 实时 DN 的一致性（抽查已改名/移动过的账号）"
    expected: "一致；若存在陈旧行，需按「开放风险与升级项」WR-03 的口径确认批量路径的比对基准"
    why_human: "批量路径的 DN 取自 sys_ad_user 缓存表，其新鲜度是生产数据事实，sqlite fixture 无法覆盖"
    resolution: "已转留 60-UAT.md Test 3（用户选择跳过；sys_ad_user.user_dn 与 AD 实时 DN 的一致性仍待核对，结果记录在该文件）"
---

# Phase 60: 软删恢复安全 Verification Report

**Phase Goal:** 软删用户恢复不再盲目接管 AD 身份——AD 当前 DN 与存量 ad_dn 校验一致才恢复；登录路径与 batch-sync 对同一恢复场景行为一致。
**Verified:** 2026-10-20
**Status:** human_needed
**Re-verification:** Yes — 2026-10-22（staleness refresh：covered_files 变更后重跑验证）

## Re-verification (2026-10-22 — staleness refresh)

**触发原因**：`init progress` 报告 `verification_status=stale`。covered_files 中 `internal/api/v1/auth.go` 在本报告出具后被 `ff699b9b`（fix: 登录与刷新令牌路径补本地账号状态检查（60-REVIEW CR-02/CR-03 critical））改动。该修复来自本会话的缺陷盘点，与 SAFE-01/SAFE-02 属**不同相位**的安全要求，必须验证二者不互相掩盖。

**关键交互核查（CR-02 × SAFE-01）**：

1. `auth.go` 登录链路顺序：步骤 5 `Authenticate` → 步骤 6 `if authResult.User == nil` → **步骤 6.5 `user.Status != models.UserStatusEnabled` → 403「用户已被禁用」**（`:229-237`）→ 步骤 9 发 token。
2. SAFE-01 的**拒绝侧**发生在步骤 5 内部：`ad_authenticator.go:131 SyncADUser` 返回 `ErrSoftDeleteIdentityMismatch` → `:136-137` 记 `[SECURITY] 软删恢复被安全策略拒绝` WARN → `:141-147` 返回 `AuthResult{User: nil, SyncErrorReason: syncErrorReasonFor(err)}` → **在步骤 6 即被 `syncErrorReasonMessage` 转成「该账号存在 AD 身份不一致，已被安全拦截，请联系系统管理员」返回，早于 6.5** → SAFE-01 的拒绝语义**不会被「用户已被禁用」掩盖**。
3. SAFE-01 的**放行侧**（DN 一致 → 恢复）：`user_sync_service.go:346-358 applyRestoreSoftDeleted` 的更新集含 `"status": models.UserStatusEnabled`（`:351`）→ 恢复后账号为启用态；`auth.go` 步骤 6 又从 DB 重读用户（`:219`）→ 6.5 读到的即启用态 → 放行，SAFE-01「DN 一致才恢复并可登录」语义完整保留。
4. SAFE-02（批量路径）：`user_sync_service.go:598` 的「AD 身份不一致，拒绝恢复（需管理员核对 ad_dn）」仍在；60-02 的批量过滤只作用于批量路径，不改登录路径。
5. CR-03（`refreshToken` `:439` 同款状态检查）是**新增收紧**，不改变 SAFE-01/SAFE-02 的判定路径。

**复核动作**：行为测试复跑 `go test ./internal/api/v1/ ./internal/core/security/ ./internal/services/system/ -count=1` → 全部 ok（13.5s / 33.2s / 11.5s），含 `ad_authenticator_phase60_test.go`、`user_sync_service_phase60_test.go`、`auth_80_03_test.go`（`TestAuth8003_*`、`TestPhase60_*`）。

**状态结论**：12/12 must-haves 仍 VERIFIED，CR-02/CR-03 与 SAFE-01/SAFE-02 无冲突。`status` 维持 `human_needed` —— 剩余 human_verification 中「真实 AD + 国密登录下的 DN 漂移拒绝端到端」（60-UAT.md Test 1）与「sys_ad_user.user_dn 与 AD 实时 DN 一致性抽查」（Test 3）仍待实测，属环境约束、不可由本地测试替代。本次变化仅为重算 covered_digest（`v3:sha256:8d37669b…`）以消除 stale。

## 目标达成判定

两个目标分量均在代码库真实成立，且均由本验证独立复跑的行为测试锁定：

- **「AD 当前 DN 与存量 ad_dn 校验一致才恢复」**——`internal/services/system/user_sync_service.go:272-297` 的包内私有入口 `checkSoftDeletedRestore` 是唯一比对点（`addomain.NormalizeDN(` 非注释调用恰 2 处，均在函数体内；`restoreSoftDeletedUser` 体内 0 处）。登录路径在 `:312` 于 `HashPassword("123456")`（`:317`）之前短路；拒绝返回 `security.ErrSoftDeleteIdentityMismatch`（`%w` 包装），事务内零写入。
- **「登录路径与 batch-sync 对同一恢复场景行为一致」**——批量路径在 `BatchSyncADUsers` 阶段 2 分类后、阶段 3 `hashPasswordsConcurrent`（`:616`）之前调用**同一个**入口（`s.checkSoftDeletedRestore(` 共 2 处调用点），被拒项 `result.Skipped++`（全文件仅 1 处）并短路移除，**不抛 error**，因此不进入 `runInBatchedTx` 的整批降级管道。

无 FAILED 真话、无 MISSING/STUB 工件、无 NOT_WIRED 关键链路。唯一未能自动证明的部分是**真实 AD/国密登录环境下的端到端呈现**（60-VALIDATION.md Manual-Only 保留项）与两项**部署环境事实**（AD 属性口径、sys_ad_user 新鲜度）——均已结构化为人工验证项，故状态为 `human_needed` 而非 `passed`。

> 注：60-REVIEW.md 的 13 条发现（含 3 条 critical）**未被本 phase 修复**，全部在 60-REVIEW-DISPOSITION.md 记为 `open`。经逐条对本次 must_haves 核验，**没有一条使本 phase 的 must-have 失效**（详见文末「开放风险与升级项」）；CR-01 削弱的是 SAFE-01 的**安全成效上限**而非本 phase 承诺的**判定语义**，故记为需跟进的高优风险，不记为 phase 目标缺口。

## 本次独立执行的命令与实际输出

| 命令 | 实际输出 |
|------|---------|
| `go build ./...` | 无输出，EXIT=0 |
| `go test -count=1 ./internal/services/system/ ./internal/core/security/ ./internal/api/v1/` | 三包全 `ok`（10.7s / 34.4s / 13.6s） |
| `go test -count=1 ./internal/services/system/ -run 'TestPhase60_' -v` | 10 个 `--- PASS`，0 FAIL（含 Predicate 的 8 个子用例） |
| `go test -count=1 ./internal/core/security/ -run 'TestPhase60_' -v` | `TestPhase60_SyncErrorReasonFor` PASS |
| `go test -count=1 ./internal/api/v1/ -run 'TestAuth8003_SyncErrorReasonMessage' -v` | PASS |
| `go test -count=1 ./internal/services/system/ -run 'TestPhase57\|TestPhase59'` | `ok`（Phase 57/59 哨兵零回归） |
| `go test -count=1 ./internal/services/addomain/` | `ok`（Phase 58 哨兵零回归） |
| `go test ./... -count=1` | 全仓 EXIT=0，无 FAIL 行（已知 SNMP flake 本轮亦未触发） |

## Observable Truths

60-01 与 60-02 两个 plan 的 must_haves.truths 共 12 条，逐条核验如下（**未**采信 SUMMARY 的自述，全部以源码与本次实跑测试为准）。

### 60-01（登录路径 / SAFE-01 / SAFE-02）

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | 存量 ad_dn 非空且与 AD 当前 DN 归一化后一致时，登录路径恢复成功：deleted_at 清空、用户可被普通作用域查询到 | ✓ VERIFIED | `user_sync_service.go:312` 校验放行 → `:321` `applyRestoreSoftDeleted` 写 `deleted_at: nil`（`:347`）；`TestPhase60_Login_MatchingDN_Restores` PASS（存 `CN=u,OU=IT,DC=corp,DC=com` / 当前 `cn=U,ou=it,dc=CORP,dc=COM`，断言 `db.First` 命中且 `DeletedAt.Valid == false`、`*AdDn == current`） |
| 2 | DN 不一致时，登录路径拒绝恢复：返回可被 `errors.Is` 识别的拒绝信号，行保持 deleted_at 非空，ad_dn 逐字不变 | ✓ VERIFIED | `user_sync_service.go:313` `fmt.Errorf("%w: username=%s", security.ErrSoftDeleteIdentityMismatch, ...)`；`SyncUserFromAD:105` 的 `%w` 包装不破坏 `errors.Is`；`TestPhase60_Login_MismatchedDN_Refuses` PASS（连续两次调用均拒、`errors.Is` 为 true、`db.Unscoped()` 读回 `DeletedAt.Valid == true` 且 `*AdDn` 逐字相等） |
| 3 | 存量 ad_dn 为 NULL、空串或纯空白时放行恢复，行为与改造前一致 | ✓ VERIFIED | `user_sync_service.go:273-282`：nil 取空串 → `NormalizeDN` 后为空即 `return true` 且不写日志；`TestPhase60_CheckSoftDeletedRestore_Predicate` 的 nil/空串/纯空白三子用例 PASS + `TestPhase60_Login_EmptyStoredDN_Restores` PASS。放行后的写库路径 `applyRestoreSoftDeleted` 本 phase 未改动（git diff 仅新增校验块） |
| 4 | 拒绝的审计详情仅由共享入口写一条 WARN（username/userID/存量 ad_dn/AD 当前 DN/path=login）；上层只用 WARN 级别区分且不携带 DN 字段；不写 sys_oper_log、不建表 | ✓ VERIFIED | 唯一审计点 `:292-295`（字段顺序 path/userID/username/storedAdDN/currentAdDN/normalizedStored/normalizedCurrent）；上层 `user_sync_service.go:453-454` 与 `ad_authenticator.go:136-137` 的 WARN 只带 username + err（err 文本为 `软删用户恢复被拒绝: AD 身份不一致: username=…`，无 DN）；`git diff 454f02ee..3b945aaa -- internal/core/db/` 为空（零迁移零建表），全仓 `operlog.Record` 未新增；`TestPhase60_Login_RefusalAuditWarn` PASS（断言含 userID/username/两侧 DN/path=login，且不含用户文案）；Predicate 用例断言放行组不产生 WARN（`strings.Count == 2`，恰为两条拒绝用例） |
| 5 | 登录拒绝映射为 `SyncErrorReason=soft_delete_mismatch`，用户文案固定且不含任何 DN 片段 | ✓ VERIFIED | `ad_authenticator.go:298-303` 纯函数 `syncErrorReasonFor`（`errors.Is` 命中 sentinel → `SyncReasonSoftDeleteMismatch`）；`auth.go:663-665` 映射为固定中文；`TestPhase60_SyncErrorReasonFor` PASS（普通 error → `user_sync`、单层/双层 `%w` 包装均命中）+ `TestAuth8003_SyncErrorReasonMessage` PASS；`grep -c 'CN=' internal/api/v1/auth.go` = 0 |
| 6 | 校验与审计只存在于一个包内私有共享入口，`restoreSoftDeletedUser` 不自行实现 DN 比对 | ✓ VERIFIED | 非注释 `addomain.NormalizeDN(` 恰 2 处，均在 `checkSoftDeletedRestore` 内（`:277`、`:285`）；`s.checkSoftDeletedRestore(` 共 2 处调用点（登录 `:312`、批量 `:594`）；awk 断言「`applyRestoreSoftDeleted` 函数体内无 `checkSoftDeletedRestore`」EXIT=0（校验未下沉） |

### 60-02（批量路径 / SAFE-01 / SAFE-02）

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 7 | 批量路径在相同的不一致场景下拒绝恢复，被拒项计入 `result.Skipped` 与 `Errors` 明细，`result.Failed` 保持 0 | ✓ VERIFIED | `user_sync_service.go:591-605`：`Skipped++` + `SyncError{Username, Error:"AD 身份不一致，拒绝恢复（需管理员核对 ad_dn）"}`；`TestPhase60_Batch_MismatchedDN_Skipped` PASS（Total=1/Skipped=1/Failed=0/Success=0/len(Errors)=1，且 `Errors[0].Error` 不含 `CN=`） |
| 8 | 批量拒绝发生在阶段 2 分类之后、阶段 3 并发哈希之前；被拒用户不执行密码哈希、不进入 `runInBatchedTx` | ✓ VERIFIED | 过滤块 `:591-605` 早于 `:616` `hashPasswordsConcurrent`（awk 行号断言 EXIT=0）；`plansByCat[catSoftDeleted] = allowed`（`:604`）在哈希前完成；被拒项已从切片移除 → 阶段 5c（`:725`）拿不到；`TestPhase60_Batch_MismatchedDN_Skipped` 断言日志不含「降级逐个处理」，且全仓 `result.Skipped++` 仅 1 处（不在 `runInBatchedTx` 内） |
| 9 | 批量路径 DN 一致时恢复成功，计入 Success，用户 deleted_at 清空 | ✓ VERIFIED | `TestPhase60_Batch_MatchingDN_Restores` PASS（大小写变体；Success=1/Skipped=0/Failed=0；`db.First` 命中、`DeletedAt.Valid == false`、`*AdDn == current`） |
| 10 | 成功 + 失败 + 跳过恒等于总数，非法拒项不会让 Success 被错误递增 | ✓ VERIFIED | `result := &BatchSyncResult{Total: len(users)}`（`:558`）；分类互斥穷尽四类，被拒项在 `:604` 从 `catSoftDeleted` 移除 → 既不进 `runInBatchedTx` 的 `Success += len(batch)`（`:821`）也不进 `Failed++`（`:831`）；`TestPhase60_Batch_MixedUsers_Conservation` PASS（Total=2 = Success1+Failed0+Skipped1） |
| 11 | 返回前端的 `SyncError.Error` 文本不含 DN 细节，日志 WARN 含 `path=batch` 与两侧 DN | ✓ VERIFIED | 响应文本为固定中文常量（`:598`，不含 `CN=`/`OU=`/`DC=`）；WARN 文本由共享入口产生，含 `path=batch` 与 storedAdDN/currentAdDN（`TestPhase60_Batch_MismatchedDN_Skipped` 断言 `path=batch` 与 `[SECURITY]`） |
| 12 | 同一输入场景下登录路径与批量路径的允许/拒绝判定一致，且最终 DB 状态一致 | ✓ VERIFIED | 两路径调用同一入口、同一 `NormalizeDN` 口径；`TestPhase60_PathConsistency_MismatchRefusedBoth` PASS（两套独立 fixture 同输入：登录 `errors.Is` 命中 sentinel / 批量 Skipped=1 / 两库均 `DeletedAt.Valid` 且 `ad_dn` 原值相同）；`TestPhase60_PathConsistency_MatchingRestoredBoth` PASS（两路径均恢复、`ad_dn` 均写入同一当前 DN）。⚠ 边界：两路径的 DN **来源**不同（登录取 LDAP `entry.DN`，批量取 `sys_ad_user` 缓存）——「同一输入」成立，真实环境可能分叉，见 WR-03 |

**Score:** 12/12 truths verified（0 present-but-behavior-unverified）

## 需求覆盖（SAFE-01 / SAFE-02）

| 需求 | 声明来源 | 描述 | 状态 | 证据 |
|------|---------|------|------|------|
| SAFE-01 | 60-01-PLAN / 60-02-PLAN frontmatter；REQUIREMENTS.md:24 | `catSoftDeleted` 恢复前校验 AD 当前 DN 与存量 `ad_dn` 一致，不一致拒绝恢复 + 安全审计日志 | ✓ SATISFIED | 真话 1-4、7-11 全部 VERIFIED；REQUIREMENTS.md:24 已勾选 `- [x]`，:71 Traceability 行记 `Complete (2026-10-20, Phase 60 — …)` 且证据行与本次实跑的用例名一致 |
| SAFE-02 | 60-01-PLAN / 60-02-PLAN frontmatter；REQUIREMENTS.md:25 | 登录路径与 batch-sync 共用分类逻辑与校验，行为一致 | ✓ SATISFIED | 真话 6、12 VERIFIED（单一入口 + 双路径成对用例）；REQUIREMENTS.md:25 已勾选 `- [x]`，:72 Traceability 行记 `Complete (2026-10-20, Phase 60 — …)` |

**ID 结算**：两个 plan 声明的 requirement 集合完全相同且均为 {SAFE-01, SAFE-02}，REQUIREMENTS.md 中 Phase 60 名下亦仅此两项 —— 无遗漏 ID、无孤儿 ID、无重复声明。

## Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/core/security/authenticator.go` | 软删恢复拒绝 sentinel 与专用原因码常量 | ✓ VERIFIED | `ErrSoftDeleteIdentityMismatch`（`:95`）、`SyncReasonSoftDeleteMismatch = "soft_delete_mismatch"`（`:101`）；`grep -c 'ErrSoftDeleteIdentityMismatch'` = 1；`AuthResult.SyncErrorReason` 注释已补原因码 |
| `internal/core/security/ad_authenticator.go` | 错误→`SyncErrorReason` 纯函数映射 | ✓ VERIFIED | `syncErrorReasonFor`（`:298-303`）；`grep -c 'syncErrorReasonFor(err)'` = 1；`SyncErrorReason: "user_sync"` 内联写法 = 0 |
| `internal/api/v1/auth.go` | 专用原因码的中文用户文案 | ✓ VERIFIED | `case security.SyncReasonSoftDeleteMismatch:`（`:663`）→ 固定文案「该账号存在 AD 身份不一致，已被安全拦截，请联系系统管理员」；未新增 403 分支（与 60-01 Task 1 步骤 3 约定一致） |
| `internal/services/system/user_sync_service.go` | 两路径共用的校验入口 + 登录短路 + 批量过滤 | ✓ VERIFIED | `checkSoftDeletedRestore`（`:272-297`，只读，awk 守卫 EXIT=0）；登录短路 `:312` 早于哈希 `:317`；批量过滤 `:591-605` 早于哈希 `:616` |
| `internal/services/system/user_sync_service_phase60_test.go` | 判定表 / 登录 / 批量 / 双路径一致性用例 | ✓ VERIFIED | 364 行新增；10 个 `TestPhase60_*` 全 PASS；未重复定义既有 helper（`captureLogs`/`strPtrSyncTest`/`setupTestDBForSync` 重定义计数 = 0） |
| `internal/core/security/ad_authenticator_phase60_test.go` | 原因码映射纯函数用例 | ✓ VERIFIED | `TestPhase60_SyncErrorReasonFor` PASS（含多层 `%w` 包装） |
| `internal/api/v1/auth_80_03_test.go` | 原因码→中文文案表驱动期望行 | ✓ VERIFIED | 新增一行 `{security.SyncReasonSoftDeleteMismatch, "该账号存在 AD 身份不一致，已被安全拦截，请联系系统管理员"}`，引用常量而非字面量；`TestAuth8003_SyncErrorReasonMessage` PASS |

## Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `user_sync_service.go` | `addomain/dn_util.go` | `addomain.NormalizeDN`（归一化判空 + 归一化比较） | ✓ WIRED | 非注释调用恰 2 处，均在 `checkSoftDeletedRestore` 内；`dn_util.go:36-56` 语义读核（TrimSpace → 按 "," split → attr 名与值双侧 ToLower → 无空格拼接；空输入返回空串，永不失败） |
| `user_sync_service.go` | `core/security/authenticator.go` | `fmt.Errorf("%w: security.ErrSoftDeleteIdentityMismatch")` | ✓ WIRED | `:313`；`errors.Is` 链完整（`SyncUserFromAD:105` → `SyncADUser:451` → `ad_authenticator.go:136/299`） |
| `ad_authenticator.go` | `api/v1/auth.go` | `SyncErrorReason` → `syncErrorReasonMessage` | ✓ WIRED | `auth.go:211-215` 的 `authResult.User == nil` 分支消费 `authResult.SyncErrorReason`；文案映射分支存在且逐字匹配 |
| `user_sync_service.go` | `user_sync_service.go`（批量） | 阶段 2 过滤 → 阶段 3 哈希 → 阶段 5c 写库 | ✓ WIRED | `:591-605` → `:616` → `:725`；被拒项在 5c 前已移出切片 |
| `user_sync_service_phase60_test.go` | `user_sync_service.go` | 同一 fixture 造型分别经两路径 | ✓ WIRED | 两个 `TestPhase60_PathConsistency_*` 用例 |

## Data-Flow Trace

| Artifact | 数据变量 | 来源 | 产出真实数据 | 状态 |
|----------|---------|------|-------------|------|
| `checkSoftDeletedRestore` | `storedDN` | `sys_user.ad_dn`（`indexExistingUsers` 的 `Unscoped()` 软删行预查询，零额外查询） | ✓ FLOWING | 预查询 `Find(&deleted)` 载入全字段，含 `AdDn *string` |
| `checkSoftDeletedRestore` | `currentUserDN`（登录） | LDAP `entry.DN`（`searchUserInfo`） | ✓ FLOWING | 真实 AD 对象 DN（来源口径风险见 CR-01） |
| `checkSoftDeletedRestore` | `currentUserDN`（批量） | `sys_ad_user.user_dn`（`ad_domain_user_sync_handler.go:44-71`） | ⚠️ STATIC | 缓存快照而非实时 AD（新鲜度风险见 WR-03） |
| 拒绝明细 | `SyncError.Error` | 常量中文串 | ✓ FLOWING | 不含 DN；DN 只进 `applogger.Warnf` |

## Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| 判定表 8 组 | `go test ./internal/services/system/ -run 'TestPhase60_CheckSoftDeletedRestore_Predicate' -count=1 -v` | 8 子用例 PASS | ✓ PASS |
| 登录恢复/拒绝/空 DN/审计 | `go test ./internal/services/system/ -run 'TestPhase60_Login_' -count=1 -v` | 4 用例 PASS | ✓ PASS |
| 批量允许/拒绝/守恒 | `go test ./internal/services/system/ -run 'TestPhase60_Batch_' -count=1 -v` | 3 用例 PASS | ✓ PASS |
| 双路径一致性 | `go test ./internal/services/system/ -run 'TestPhase60_PathConsistency_' -count=1 -v` | 2 用例 PASS | ✓ PASS |
| 原因码映射 | `go test ./internal/core/security/ -run TestPhase60_SyncErrorReasonFor -count=1 -v` | PASS | ✓ PASS |
| 文案映射 | `go test ./internal/api/v1/ -run TestAuth8003_SyncErrorReasonMessage -count=1 -v` | PASS | ✓ PASS |
| 静态守卫（单一入口 / 只读 / 顺序 / 未下沉） | 见下方「静态守卫核对」 | 全部 EXIT=0 | ✓ PASS |

### 静态守卫核对（PLAN acceptance_criteria 逐条实测）

| 守卫 | 期望 | 实测 |
|------|------|------|
| `grep -c 'ErrSoftDeleteIdentityMismatch' internal/core/security/authenticator.go` | 1 | 1 |
| `grep -rn '"soft_delete_mismatch"' internal/core/security internal/api internal/services/system --include=*.go \| wc -l` | 1 | 1 |
| `grep -c 'syncErrorReasonFor(err)' internal/core/security/ad_authenticator.go` | 1 | 1 |
| `grep -c 'SyncErrorReason: "user_sync"' internal/core/security/ad_authenticator.go` | 0 | 0 |
| `grep -c 'security.SyncReasonSoftDeleteMismatch' internal/api/v1/auth.go` | ≥1 | 1 |
| 非注释 `addomain.NormalizeDN(` 计数 | 2 | 2 |
| `s.checkSoftDeletedRestore(` 计数 | 2 | 2 |
| `result.Skipped++` 计数 | 1 | 1 |
| awk：校验函数体内零写库 | EXIT=0 | EXIT=0 |
| awk：校验早于 `HashPassword("123456")` | EXIT=0 | EXIT=0 |
| awk：批量过滤块早于 `hashPasswordsConcurrent` | EXIT=0 | EXIT=0 |
| awk：`applyRestoreSoftDeleted` 内无校验调用 | EXIT=0 | EXIT=0 |
| 测试文件未重复定义既有 helper | 0 | 0 |

## Probe Execution

无。本 phase 未声明探针，仓库 `scripts/*/tests/probe-*.sh` 模式不适用（纯 Go 单元测试 phase）。

## 范围护栏核查（prohibitions）

| 禁止项 | 核查方式 | 结果 |
|--------|---------|------|
| 拒绝路径绝不清空/改写 `ad_dn`，绝不解除软删 | `TestPhase60_Login_MismatchedDN_Refuses`（两次调用后 `Unscoped` 读回断言）+ `TestPhase60_Batch_MismatchedDN_Skipped` + 校验函数零写库 awk | ✓ 守住 |
| 用户可见文案与批量 Errors 明细绝不包含 DN | `grep -c 'CN=' internal/api/v1/auth.go` = 0；批量错误文本为常量中文；测试断言 `NotContains "CN="` | ✓ 守住 |
| 两路径不得各自实现 DN 比对 | `NormalizeDN(` 非注释计数 = 2，均在共享入口内 | ✓ 守住 |
| 不得修改 `user_ou_service.go` / `GetUserDeptByADOU` / `BatchMoveUsersToNewOU` | `git diff --stat 454f02ee 3b945aaa -- internal/services/addomain/user_ou_service.go` → 空 | ✓ 守住（Phase 62 HYGIENE-03 对照载体完整） |
| 不写 sys_oper_log、不建表 | `git diff --name-only 454f02ee 3b945aaa -- internal/core/db/` → 空 | ✓ 守住 |
| 批量拒绝不得以 error 形式进入 `runInBatchedTx` | 过滤块短路 `continue`，不 return error；测试断言日志无「降级逐个处理」 | ✓ 守住 |
| 被拒项不得留在 `plansByCat[catSoftDeleted]` | `:604` 覆盖回写 `allowed` | ✓ 守住 |

## Anti-Patterns

| 文件 | 位置 | 模式 | 严重度 | 影响 |
|------|------|------|--------|------|
| — | — | 未发现 `TBD` / `FIXME` / `XXX` / `TODO`（对本 phase 改动的 7 个文件全文扫描） | — | 无 |
| — | — | 未发现硬编码空返回 / `return null` 桩 / 空 catch | — | 无 |

## 回归检查（前序 phase）

| 检查项 | 结果 |
|--------|------|
| Phase 57 哨兵（`TestPhase57*`，`internal/services/system`） | ok，零回归 |
| Phase 59 哨兵（`TestPhase59*`，`internal/services/system`） | ok，零回归 |
| Phase 58 包（`internal/services/addomain`） | ok，零回归 |
| 全仓 `go test ./... -count=1` | EXIT=0，无 FAIL（已知 SNMP flake 本轮未触发） |
| `user_ou_service.go` diff | 空（Phase 62 载体未被触碰） |

## Human Verification Required

1. **真实 AD 登录拒绝的用户可见文案（60-VALIDATION Manual-Only 保留项）**
   - 做法：在真实 AD 环境制造 DN 漂移（软删用户存量 `ad_dn` 与 AD 当前 DN 不一致）后走一次完整登录（国密 SM2/SM4 加密请求）。
   - 预期：响应 message = 「该账号存在 AD 身份不一致，已被安全拦截，请联系系统管理员」，不含 `CN=`/`OU=`/`DC=`；后端出现 `[SECURITY] 软删用户恢复被拒绝` WARN（path=login + userID + username + 两侧 DN）；`sys_user` 行保持软删且 `ad_dn` 逐字不变；不签发 token。
   - 为何需人工：需真实域控 + 国密请求端到端；服务层原因码与文案映射已分别被单测锁定。

2. **AD 身份属性口径核对（CR-01 前置条件）**
   - 做法：抽查本环境若干在职账号的 `sAMAccountName` 与 `userPrincipalName` 本地部分。
   - 预期：两者一致；若不一致，闸门比对的对象与被认证对象可能不是同一个 AD 对象（见开放风险 CR-01）。
   - 为何需人工：AD 账号属性形态是部署事实，代码库不可见。

3. **批量路径 DN 来源新鲜度核对（WR-03）**
   - 做法：抽查 `sys_ad_user.user_dn` 与 AD 实时 DN（尤其改名/移动过的账号）。
   - 预期：一致；若存在陈旧行，批量路径的比对基准与登录路径会分叉。
   - 为何需人工：生产数据事实，sqlite fixture 不可覆盖。

## 开放风险与升级项（来自 60-REVIEW.md，本 phase 未修复、未 triage）

按「是否使本 phase 的 must-have 失效」逐条判定：

| Finding | 严重度 | 是否击穿本 phase must-have | 判定依据与处置 |
|---------|--------|----------------------------|------------|
| **CR-01**（bind 用 UPN、闸门用 sAMAccountName 查回的 DN，二者可能不是同一 AD 对象） | critical | **否**（但削弱 SAFE-01 的安全成效上限） | 本 phase 的 must-have 是「判定语义」：给定 AD 当前 DN，与存量 `ad_dn` 归一化比较，不一致即拒。该语义 100% 实现并被 11 条行为断言锁定。CR-01 的根因在**身份解析层**（`ad_authenticator.go:83` 用 `username@domain` 绑定，`:260` 用 `(sAMAccountName=username)` 搜索），该代码本 phase **完全未改**（`git diff` 可见 bind/search 段零改动）。后果是「被认证主体 ≠ 被比对主体」，即在 UPN 与 sAMAccountName 可分叉/可碰撞的 AD 形态下，闸门可能放行一次身份错配的恢复。**建议**：新建跟进需求（身份解析必须与被认证对象同源，或搜索时同时取 `sAMAccountName` 与 `userPrincipalName` 并校验一致），并在 60-REVIEW-DISPOSITION.md 中把 CR-01 由 `open` 置为 `deferred` 并写明去向；部署前建议先做人工项 2。 |
| CR-02（AD/hybrid 路径不查本地 `status`，被禁用用户仍可登录拿 token） | critical | 否 | 纯 auth 层既有缺陷，早于本 phase（Phase 57 Deferred 已登记「auth.go AD 登录不查 user.Status」为独立 bug）；与 SAFE-01 的 DN 闸门无因果关系。本 phase 未触碰该分支。**建议**：独立 phase 修复。 |
| CR-03（`refreshToken` 为被禁用用户重签 token 对） | critical | 否 | 同上，既有 auth 层缺陷，与本 phase 无交集。 |
| WR-01 / WR-02 | warning | 否 | 既有 auth 层缺陷（config 读取复用 struct、AD 连接错误用 `==` 比较），不涉及恢复闸门。**注**：WR-02 与本 phase 的 sentinel 传播形成对照——本 phase 用 `errors.Is` 正确判定，未复制该反模式。 |
| WR-03（批量闸门比对的是 `sys_ad_user` 缓存 DN 而非实时 AD DN） | warning | 否（但削弱 SAFE-02 的真实世界一致性） | SAFE-02 的 must-have 措辞为「**同一输入场景**下两路径判定一致」，已被成对用例证伪；批量路径的 DN 来源是既有端点设计（`ad_domain_user_sync_handler.go:44-71` 从 `sys_ad_user` 取数），本 phase 未改该数据流。真实环境中 AD 对象移动后，登录（实时 DN）会拒、批量（陈旧 DN）会放行 → 实际分叉。**建议**：跟进需求（批量比对前取实时 DN，或对 `sys_ad_user` 新鲜度 fail-closed）。 |
| WR-04（`NormalizeDN` 不处理 RFC4514 转义逗号，等价 DN 可能被判不等） | warning | 否 | 属 Phase 58 锁定的 `NormalizeDN` 既有语义（D-02 单一口径），本 phase 未改该函数。表现为 fail-closed 误拒合法用户；RESEARCH A6 已登记为已接受风险。**建议**：与 WR-03 一并纳入后续 DN 语义 phase。 |
| WR-05（`user_ou_service.go` 的 `restoreDeletedUserWithADInfo` 未过闸门） | warning | 否（当前潜伏） | 已核实该函数与其唯一入口 `HandleUserLoginAD` **无任何生产调用方**（grep 全仓仅命中定义与 `user_ou_service_test.go` / `auth_handler_test.go` 注释）；且本 phase 承诺不修改该文件（护栏已守住）。**建议**：随 Phase 62 HYGIENE-03 一并删除。 |
| IN-01 / IN-02 / IN-03 / IN-04 / IN-05 | info | 否 | IN-03（策略拒绝返回 HTTP 500）是 60-01 Task 1 步骤 3 **显式决定**（保持既有 `User==nil` 分支包装不变），非偏差；IN-02 的「一条拒绝三条 WARN」中仅共享入口那一条带审计详情，上层两条无 DN 字段，不违反 must-have #4。 |

## Gaps Summary

**无 phase 目标级缺口。** 12 条 must-have 全部 VERIFIED，12 个工件全部存在且实质，5 条关键链路全部 WIRED，SAFE-01/SAFE-02 两个需求 ID 全部结算并已在 REQUIREMENTS.md 标记 Complete；零 stub、零未接线、零反模式、零范围越界（`user_ou_service.go` diff 为空、零迁移）。

状态为 `human_needed` 而非 `passed` 的唯一原因是存在 3 项需真实环境/人工判定的验证项（端到端拒绝文案、AD 属性口径、批量 DN 新鲜度），其中后两项直接关系到 SAFE-01/SAFE-02 在生产环境中的实际成效。

**待办（非本 phase 缺口，建议路由）**
1. CR-01 → 新建跟进需求：身份解析与被认证对象同源（或校验 `sAMAccountName == UPN 本地部分`）；并在 `60-REVIEW-DISPOSITION.md` 置 `deferred` + 写明去向（当前 13 条全为 `open`，无 triage）。
2. WR-03 → 新建跟进需求：批量比对取实时 DN 或对 `sys_ad_user` 新鲜度 fail-closed。
3. CR-02 / CR-03 → 独立 phase：AD/hybrid 登录与 `refreshToken` 的 `status` 校验。
4. 文档卫生：ROADMAP.md 里程碑清单第 44 行 Phase 60 复选框未随 Progress Table 同步为 `[x]`。

---

_Verified: 2026-10-20_
_Verifier: Claude (gsd-verifier)_
