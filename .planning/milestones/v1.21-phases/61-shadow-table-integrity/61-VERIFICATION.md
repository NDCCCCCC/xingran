---
phase: 61-shadow-table-integrity
verified: 2026-10-20T23:53:36Z
updated: 2026-10-22T16:20:00+08:00
status: passed
score: 8/8 must-haves verified
covered_files: [".planning/phases/61-shadow-table-integrity/61-01-PLAN.md", ".planning/phases/61-shadow-table-integrity/61-01-SUMMARY.md", ".planning/phases/61-shadow-table-integrity/61-02-PLAN.md", ".planning/phases/61-shadow-table-integrity/61-02-SUMMARY.md", ".planning/phases/61-shadow-table-integrity/61-03-PLAN.md", ".planning/phases/61-shadow-table-integrity/61-03-SUMMARY.md", "internal/services/addomain/computer.go", "internal/services/addomain/computer_78_06_test.go", "internal/services/addomain/group_sync_service.go", "internal/services/addomain/shadow_cleanup_service.go", "internal/services/addomain/shadow_cleanup_test.go", "internal/services/addomain/sync.go", "internal/services/addomain/sync_78_05_test.go"]
covered_digest: "v3:sha256:15ac699db24e4aa3a75608e3b78fce1c3fe69a1682093cc73683086346030bf4"
behavior_unverified: 0
overrides_applied: 0
human_verification:
  - test: "生产 PG 真并发空窗观察：组成员重放执行期间，从第二会话查询 sys_ad_group_member"
    expected: "外部会话要么看到旧成员集、要么看到新成员集，观察不到空成员窗口（PG READ COMMITTED + 单事务原子性）"
    why_human: "sqlite 进程内测试无法建立真正的跨连接并发观察者；单测只覆盖了原子性的进程内可证伪代理（TestSyncGroupMembers_CreateFailRollsBackDelete）"
    resolution: "PASS（2026-10-22 08:05 轮复测）：根因经 20:05 轮观测 + 最小复现实验闭合为部署滞后（10.62.10.39 二进制未含 cb1382c3 单事务实现，V 形与旧两步提交逐特征自洽）。39 重新发版后 08:05 轮（sync_log 08:05:00.011 dur=9s / 08:05:00.013 dur=6s，67 组）采样器 v5 连续 3025 点 member_count 全程 2815、DISTINCT 59 零偏差；重放窗口 08:05:04.882-05.769（0.887s）内 26 个样本逐条 2815|59，与失败轮同等密度下无 V 形 → 空窗不可观测，expected 达成。证据在 61-UAT.md Test 1"
  - test: "首轮批量清理规模真实观察：部署后核对 SyncResult 三计数（deletedUserCount/deletedOUCount/deletedComputerCount）与 sys_oper_log 清理日志"
    expected: "清理计数与 AD 端实际消失对象数一致；若拟删 >50% 被闸门拒绝属预期保护行为，运维不得误报为故障"
    why_human: "清理规模依赖生产 PG 存量与真实 AD 数据对比，无法在本地 sqlite fixture 复现"
    resolution: "PASS（2026-10-21 14:05 生产首跑）：OU 清理 977（与 sync_log ou_count 对账一致）、电脑 13、用户拟删 4643/8820 被闸门拦截（零误删）——三类计数与 AD 端数据自洽，WARN 无 DN 明文，整轮 success；证据在 61-UAT.md Test 2"
  - test: "闸门真触发演练：测试域控删除 >50% 对象后跑全量同步"
    expected: "清理被 Gate 2 拒绝，日志含「拟删除 x/y 个...(>50%)」WARN（不含 DN 明文），整轮同步仍 success"
    why_human: "需人为构造 AD 端大规模删除的真实域控环境"
    resolution: "PASS（2026-10-21 14:05 生产真实触发，等价覆盖）：LDAP 返回不完整使差集虚大 52.6% → Gate 2 拒绝 + WARN 无 DN 明文 + 轮次 success，与人为删对象走同一代码分支（shadow_cleanup_service.go:90-95）；测试域控真删演练转为可选项；证据在 61-UAT.md Test 3"
  - test: "生产 PG 唯一约束核对（FLAGGED ASSUMPTION A1）：启动冒烟后执行 \\d sys_ad_user / sys_ad_ou / sys_ad_computer / sys_ad_group"
    expected: "各表存在 (ad_config_id, *_dn/name) 唯一约束；若缺失，软删行复现将表现为重复行而非约束冲突（复活路径设计仍正确）"
    why_human: "本机无 PG 实例，sqlite 单测无法覆盖生产约束形态"
    resolution: "PASS（2026-10-21 16:33 实测）：user/ou/group 为真 UNIQUE 约束 (ad_config_id, *_dn)；computer 无约束行但有唯一索引 uni_sys_ad_computer_config_name (ad_config_id, computer_name)，与 computer.go:676 ON CONFLICT 目标完全对齐——A1 成立，证据在 61-UAT.md Test 4"
---

# Phase 61: 影子表完整性（删除清理 + 组成员事务化）Verification Report

**Phase Goal:** AD 端已消失的对象不再永久滞留 `sys_ad_user/ou/computer` 影子表（静默数据丢失消除）；组成员重放消除空成员可见窗口；清理与全量同步不并发。
**Verified:** 2026-10-20T23:53:36Z
**Status:** passed（自动化证据全绿；4 项真环境人工项均已 PASS，见 frontmatter resolution；2026-10-22 为 staleness 复验）
**Re-verification:** Yes — 2026-10-22（staleness refresh：covered_files 变更后重跑验证）

## Re-verification (2026-10-22 — staleness refresh)

**触发原因**：`init progress` 报告 `verification_status=stale`。covered_files 中 6 个文件在本报告出具后被改动：

| 文件 | 最后改动提交 | 性质 |
|---|---|---|
| `internal/services/addomain/sync.go` | `78cb857a` fix(addomain): GAP-62-01 全嵌套组成员导致 GORM ErrEmptySlice 中止整轮同步 | Phase 62 的 HYGIENE-04 过滤 + 过滤后空切片早返回 |
| `internal/services/addomain/group_sync_service.go` | `390ae9eb` fix(addomain): HYGIENE-03/04/05 … | Phase 62 的组成员过滤接线 |
| `internal/services/addomain/sync_78_05_test.go` | `30f58068` test(addomain): Phase 62 三 plan 测试用例 | 6 处直调补第 5 参（语义不变） |
| `internal/services/addomain/computer.go`、`computer_78_06_test.go` | `3bcd1c84` feat(61-02): resurrect soft-deleted computer and group rows on upsert | 61 自身后续提交（原验证快照早于该提交） |
| `internal/services/addomain/shadow_cleanup_test.go` | `bdc8c452` test(61-02): add failing resurrect tests … | 同上 |

**复核动作与结论**：

1. 核心真相「组成员重放消除空成员可见窗口」——`sync.go:679 syncGroupMembers` 仍是**每组单事务**：`s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {`（`:680`）内 `tx.Unscoped().Where("ad_config_id = ? AND group_dn = ?", …).Delete(...)`（`:682`）+ 批量 Create。Phase 62 的改动只在**该事务内部**追加嵌套组 DN 过滤与「过滤后为空则早返回」，**未触碰 delete/insert 的事务边界** → 原子性不变。
2. 回滚语义回归仍绿：`TestSyncGroupMembers_CreateFailRollsBackDelete`（FORCED_FAIL CHECK 注入 → 旧成员 Unscoped 原样保留）仍 PASS，证明事务边界未被 Phase 62 破坏。
3. 生产端复核（Phase 62 部署后）：2026-10-22 14:35 轮（UTC 06:35）全量同步 `success`（ou_count=3590 / group_count=67 / user_count=4177 / error_count=0 / duration=9s），其中嵌套组污染行由 8 → 0 —— 61 的「重放无空窗 + 影子表清理」链路在新代码下整轮正常完成。
4. `computer.go` 的复活语义与 `shadow_cleanup_service.go` 的两道闸门（用户 `:90` / OU `:146`，判式 `len(vanishedDNs)*2 > len(stale*)` 严格大于、恰 50% 放行）自 61 自身提交后未再改动。
5. 行为测试复跑：`go test ./internal/services/addomain/ ./internal/scheduler/ -count=1` → ok 8.3s / 12.5s（含 `TestSync78_*`、`TestPhase58_*` 及 62 阶段新增用例）。

**状态结论**：8/8 must-haves 仍 VERIFIED，4 项 human_verification 均为 PASS（结论见 frontmatter resolution），`status` 维持 `passed`。本次变化仅为重算 covered_digest（`v3:sha256:15ac699d…`）以消除 stale。

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | 全量同步末尾，AD 端已消失的 `sys_ad_user/ou/computer` 行被软删除（SC1 / SHADOW-01） | ✓ VERIFIED | `shadow_cleanup_service.go` 三类清理（`cleanupVanishedUsers/OUs/Computers`）均为 GORM 默认作用域 `Delete`（:99-101/:155-157/:214-216），差集键分别为 UserDN/OUN/DistinguishedName，管理集口径与对应 sync 函数一致（user 仅 sAMAccountName 非空 :68、computer 仅 cn 非空 :184、OU 全部 entry :127）；`TestShadowCleanup_{Users,OUs,Computers}_SoftDeletesVanishedRows` + `FiltersUnmanagedEntries` + `CategoriesIndependent` 全部 PASS（本轮独立复跑 12/12） |
| 2 | 两道闸门生效：LDAP 0 条拒绝；拟删\*2 > 存量拒绝（恰 50% 放行，判式严格大于）（SC2 / SHADOW-01） | ✓ VERIFIED | 三类清理各含 Gate 1（`len(entries)==0` → WARN + nil）与 Gate 2（`len(vanished)*2 > len(stale)` :90/:146/:205，与 `group_sync_service.go` handleDeletedGroups 判式同构）；`RejectsEmptyLDAP`×3 / `RejectsOverThreshold`×2 / `AtExactThreshold` 全部 PASS——恰 50% 边界有行为断言 |
| 3 | `sys_ad_group_member` 重放在单事务内完成，Create 失败时 Delete 回滚、重放幂等（SC3 / SHADOW-02） | ✓ VERIFIED | `sync.go:670` `syncGroupMembers` 整体包进 `s.db.WithContext(ctx).Transaction` 闭包，Delete/Create 一律用 `tx`，`Unscoped()` 保留，错误包装前缀保留；`TestSyncGroupMembers_CreateFailRollsBackDelete` 以 fixture 表级 `CHECK (user_dn != 'FORCED_FAIL')`（sync_78_05_test.go:188）注入 Create 失败，断言 Unscoped 旧 2 条原样 + 返回 error——PASS；`TestSyncGroupMembers_ReplayIdempotent` PASS |
| 4 | 删除清理挂 `syncDataInternal` 内执行（复用 singleflight），与全量同步互斥不并发（SC4 / SHADOW-03） | ✓ VERIFIED | `sync.go:173` 恰一处 `s.cleanupAfterFullSync(...)` 调用（grep 全包计数==1），位于 syncComputers 成功块（:163-166）之后、last_sync_at 更新（:178）之前；`syncDataInternal` 由 `s.syncGroup.Do(key, ...)`（:70-72）包裹；`cleanupAfterFullSync` 首行 `models.ADSyncTypeFull` 常量守卫（shadow_cleanup_service.go:38）；`TestShadowCleanup_FullSyncOnly`（增量 no-op vs full 清理）PASS；清理失败仅 `applogger.Warnf` 非阻断（:174，无 return err） |
| 5 | 软删行同 DN 在 AD 复现 → 四类 upsert（OU/Group/User/Computer）复活该行而非撞唯一约束（SHADOW-01 配套，61-02） | ✓ VERIFIED | 三处 existing 查询 Unscoped 化（sync.go:244/:351/:505）+ computer.go:528/:549（后者显式 `deleted_at IS NULL` 谓词已移除）+ group_sync_service.go:244；三处+两处+一处 OnConflict AssignmentColumns 含 `"deleted_at"`（sync.go:329/:460/:647、group_sync_service.go:157/:339、computer.go:686）；六处命中行 `DeletedAt` 清零（sync.go:246/:353/:507、computer.go:532/:553、group_sync_service.go:246）；`TestSync_Resurrect{OU,Group,Computer_SameDN,Computer_SameNameNewDN,Group_IncrementalPath}` 5/5 PASS（同 DN 复活、同名换 DN 经 name-map 复活、增量路径复活） |
| 6 | D-78-05c「软删行 + 同 DN → UNIQUE 冲突」既有断言已按复活语义翻转，且既有套件零回归 | ✓ VERIFIED | `TestSync78_GetExistingOUs`（现断言软删行被命中 2 条）与 `TestSync78_SyncUsers_RestoreSoftDeleted`（现断言 `require.NoError` + 默认作用域可见 + Unscoped 计数 1）翻转落盘（sync_78_05_test.go:321-328 注释记翻转依据）；`go test ./internal/services/addomain/ -count=1` 全包 ok（8.175s，本轮独立复跑），含 threshold 四用例、F-01/F-02、FullDiff 等既有锚点 |
| 7 | `SyncResult` 新增 `deletedUserCount` / `deletedOUCount` / `deletedComputerCount` 三个 JSON 计数字段 | ✓ VERIFIED | sync.go:47-50 三字段落盘，JSON tag 与 plan 逐字一致（ComputerCount 带 nolint 注释）；成功路径赋值于 shadow_cleanup_service.go:105/:161/:220 |
| 8 | 空成员集语义与 Unscoped 硬删语义保持；三个调用方零改动（SHADOW-02 prohibition 面） | ✓ VERIFIED | `len(memberDNs)==0` 提前 return 保留在事务内（sync.go:680-682）；`TestSync78_SyncGroupMembers_FullDiff` 既有断言在全包回归中保持绿；调用方恰三处（sync.go:408、group_sync_service.go:164/:349），`git diff 3bcd1c84..cb1382c3 -- group_sync_service.go` 为空（61-03 未触碰该文件） |

**Score:** 8/8 truths verified（0 present-behavior-unverified；所有行为依赖型真相均有通过的命名行为测试）

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `internal/services/addomain/shadow_cleanup_service.go` | cleanupAfterFullSync 编排 + 三类清理方法 | ✓ VERIFIED | 225 行，实质完整：三类五段结构、双闸门、默认作用域软删、syncType 常量守卫、无 Unscoped 硬删 |
| `internal/services/addomain/shadow_cleanup_test.go` | TestShadowCleanup 全套 + TestSync_Resurrect 全套 | ✓ VERIFIED | 600 行，17 个测试函数，全部经本轮独立复跑 PASS |
| `internal/services/addomain/sync.go` | SyncResult 三字段 + 单点挂载 + 复活化 + syncGroupMembers 事务化 | ✓ VERIFIED | 全部落地且接线（挂载点唯一、非阻断 WARN、Unscoped+清零+AssignmentColumns） |
| `internal/services/addomain/computer.go` | queryExistingComputers/queryAllComputerNames Unscoped 化 + batchUpdate 补 deleted_at | ✓ VERIFIED | :528/:549/:686 落地；读路径 :94/:151 保持默认作用域 |
| `internal/services/addomain/group_sync_service.go` | syncGroupEntries Unscoped 化 + 两处 OnConflict 补 deleted_at | ✓ VERIFIED | :244/:157/:339 落地；读路径 :189/:197/:211/:378 保持默认作用域 |
| `internal/services/addomain/sync_78_05_test.go` | 两用例翻转 + CHECK 注入 fixture + 2 条事务新用例 | ✓ VERIFIED | :188 CHECK 注入、:321 注释记翻转依据、:1309/:1349 两条新用例 |
| `internal/services/addomain/computer_78_06_test.go` | 既有 name-map 断言按复活语义改写 | ✓ VERIFIED | 61-02 落地，全包回归中保持绿 |
| 回填工件（VALIDATION/REQUIREMENTS/ROADMAP/STATE） | 收口回填 | ✓ VERIFIED | 61-VALIDATION.md `status: validated` + `nyquist_compliant: true` + Task Map 按实际编号回填（61-01-01~61-03-02 全 ✅ green）；REQUIREMENTS.md SHADOW-01/02/03 三行 traceability 均为 Complete 且附证据；ROADMAP 三 plan 条目已勾选附日期 |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | --- | --- | ------ | ------- |
| syncDataInternal (sync.go:173) | cleanupAfterFullSync → 三类 cleanupVanished* | 成功路径单点调用 | ✓ WIRED | 调用恰一处，位于 syncComputers 后、last_sync_at 前，singleflight.Do 闭包内 |
| cleanupVanished* | sys_ad_user/ou/computer 软删 | 默认作用域 Delete（ad_config_id + *_dn IN） | ✓ WIRED | 三类条件列与模型列名逐字对齐 |
| existing 查询 Unscoped | 复活行进入 update 分支 | Unscoped().Find + DeletedAt 清零 | ✓ WIRED | 裸 Create（batchCreateOUs/createGroupsInBatches/usersToCreate）不再收到复活行——由 5 条 TestSync_Resurrect 行为证明 |
| OnConflict AssignmentColumns + deleted_at | SET deleted_at = EXCLUDED（NULL）→ 复活本体 | 三文件六处 | ✓ WIRED | 均含 `"deleted_at"` 列 + 机制注释 |
| fixture CHECK (user_dn != 'FORCED_FAIL') | Create 半途失败注入 → 回滚断言 | setupSync78DB DDL :188 | ✓ WIRED | TestSyncGroupMembers_CreateFailRollsBackDelete 消费该注入点并 PASS |
| 读路径（user.go:50/88、ou.go:43/78、computer.go:94/151） | 默认作用域保持 | deleted_at IS NULL 谓词 | ✓ WIRED | 复活改造未触碰任何 List/Get 读路径（grep 实证） |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| cleanupVanishedUsers/OUs/Computers | staleUsers/staleOUs/staleComputers | `s.db.WithContext(ctx).Where(...).Find(...)` 真实 DB 查询 | 是 | ✓ FLOWING |
| cleanupVanished* managedXxxDNs | LDAP entries | `syncDataInternal` 内 `client.Search*` 实采结果透传（ous/users/computers 形参） | 是 | ✓ FLOWING |
| SyncResult.Deleted*Count | 软删差集长度 | 实际执行软删的 `len(vanishedDNs)` | 是 | ✓ FLOWING |
| syncGroupMembers members | LDAP member 属性 | 调用方解析的 memberDNs 形参（显式收参，无自持 db） | 是 | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| 全仓编译 | `go build ./...` | EXIT=0 | ✓ PASS |
| 清理 12 条闸门/软删/独立性/增量 no-op | `go test ./internal/services/addomain/ -run "TestShadowCleanup" -count=1 -v` | 12/12 PASS (ok 1.947s) | ✓ PASS |
| 四类复活 + 事务回滚/幂等 7 条 | `go test ./internal/services/addomain/ -run "TestSync_Resurrect\|TestSyncGroupMembers" -count=1 -v` | 7/7 PASS (ok 1.870s) | ✓ PASS |
| 全包零回归（含翻转用例与既有锚点） | `go test ./internal/services/addomain/ -count=1` | ok 8.175s | ✓ PASS |

### Probe Execution

无 phase 声明探针（`scripts/*/tests/probe-*.sh` 不适用本 Go 服务 phase）——以定向命名测试 + 全包回归替代，见上表。

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| SHADOW-01 | 61-01, 61-02 | 全量同步末尾对 AD 端已消失的 `sys_ad_user/ou/computer` 软删除，复用两道闸门 | ✓ SATISFIED | Truth 1/2/5/6；REQUIREMENTS.md traceability 已标 Complete；真环境人工项见 61-VALIDATION Manual-Only |
| SHADOW-02 | 61-03 | `sys_ad_group_member` 重放改为事务包裹，消除空成员可见窗口 | ✓ SATISFIED | Truth 3/8；CHECK 注入回滚用例 + 重放幂等 PASS；真并发观察列人工项 |
| SHADOW-03 | 61-01 | 删除清理挂 `syncDataInternal`（复用 singleflight），不与全量同步并发 | ✓ SATISFIED | Truth 4；挂载点源断言 + TestShadowCleanup_FullSyncOnly |

无孤儿需求：REQUIREMENTS.md SHADOW 组恰含这三项，三个 plan 的 `requirements` 声明（[SHADOW-01, SHADOW-03] / [SHADOW-01] / [SHADOW-02]）完整覆盖，无未被 plan 认领的 phase 需求。

### Prohibitions Check（ADR-550 判定面）

| Prohibition | Tier | Status | Evidence |
| ----------- | ---- | ------ | -------- |
| 不得硬删影子表行（保持 Unscoped 可审计） | test | ✓ VERIFIED | 三类清理均默认作用域 Delete；SoftDeletesVanishedRows 用例断言 Unscoped 计数不变 |
| 闸门拒绝不得使整轮同步转 failed | test | ✓ VERIFIED | RejectsEmptyLDAP/RejectsOverThreshold 断言返回 nil 无错误传播；挂载点 WARN 非 return err |
| 清理不得脱离 syncDataInternal 单点挂载 | test | ✓ VERIFIED | grep 挂载计数==1；无 sys_job/独立 goroutine；TestShadowCleanup_FullSyncOnly 增量 no-op |
| 禁 partial unique index 路线（零 DDL） | judgment | ✓ VERIFIED | `git diff 96d2c3f4..HEAD -- internal/core/db/` 为空 |
| 禁 PG TRIGGER 路线（零 .sql） | judgment | ✓ VERIFIED | `git diff --name-only -- "*.sql"` 为空 |
| 注释不得引用 Phase 62 边界文件/符号 | judgment | ✓ VERIFIED | 六个触碰文件 grep `user_ou_service\|GetUserDeptByADOU\|BatchMoveUsersToNewOU` 计数全 0 |
| 不得移除 Delete 的 Unscoped() | test | ✓ VERIFIED | sync.go:675 `tx.Unscoped()` 保留；FullDiff 计数断言保持绿 |
| 不得合并整轮为大事务 | judgment | ✓ VERIFIED | syncGroupMembers 签名不变、仍逐组调用；三调用方零改动（61-03 diff 为空） |
| 空成员集语义不得改变 | test | ✓ VERIFIED | `len(memberDNs)==0` 分支保留事务内；FullDiff 保持绿 |

无 flagged/unverified prohibition 残留——全部条目有源断言或测试证据，无静默通过项。

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| internal/services/addomain/sync.go | 224 | 注释中 `gorm:"size:XXX"` 标签写法引用命中 XXX 关键词 | ℹ️ Info | 非债务标记——是对既有模型 tag 语法的文档引用，无 follow-up 缺口 |

无 TBD/FIXME/PLACEHOLDER/空实现/stub 返工痕迹；无硬编码空数据；phase 触碰文件零债务标记。

### Gaps Summary

无 gaps。三条 Success Criteria 与三个需求（SHADOW-01/02/03）的全部自动化可证伪断言在本轮独立复跑中全绿（`go build ./...` EXIT=0；TestShadowCleanup 12/12；TestSync_Resurrect + TestSyncGroupMembers 7/7；addomain 全包 ok）。

**状态为 human_needed 的唯一原因**：四项真环境人工项（生产 PG 并发空窗观察、首轮清理规模核对、闸门真触发演练、生产唯一约束核对）依赖真实 AD 域控与生产 PG 实例，超出 sqlite 单测能力面——均已结构化记录于 61-VALIDATION.md Manual-Only 表与本报告 human_verification 节，随 verify-work 人工项流程执行。这不构成任何真相失败：代码已存在、已接线、行为有通过的进程内测试锁定。

**部署整体性提醒（承 61-02 SUMMARY）**：61-01/02/03 三个 plan 的产物必须整体部署——清理 writer（61-01）与复活路径（61-02）分隔部署会使软删行复现撞唯一约束。

---

_Verified: 2026-10-20T23:53:36Z_
_Verifier: Claude (gsd-verifier)_