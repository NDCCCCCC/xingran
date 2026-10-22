---
phase: 62
phase_name: 卫生收尾与死代码清理
status: complete
started: 2026-10-22T12:35:00+08:00
updated: 2026-10-22T14:55:00+08:00
tests_total: 4
tests_passed: 4
tests_failed: 0
tests_pending: 0
---

# Phase 62 UAT — 卫生收尾与死代码清理（HYGIENE-03/04/05）

## Context

Phase 62 三 plan 已执行完毕并提交（390ae9eb / 30f58068 / 26276291 等），新 Linux 二进制已于 2026-10-22 11:55 构建并部署到 10.62.10.39。

- **HYGIENE-03**：删除零生产调用的 `user_ou_service.go`（7 方法）与 `BatchMoveUsersToNewOU`，修订 4 处过期注释，共享测试 const 迁入 test_fixtures_test.go
- **HYGIENE-04**：组成员重放入口按本轮 group DN 集合过滤嵌套组 DN（零 DDL），重放自愈清除生产存量 8 行污染
- **HYGIENE-05**：UpsertMapping 遇同名兄弟部门争用同一 OU DN 时返回 `ErrOUDNConflict` 不静默覆盖；dept_sync 计 FailedDepts + Errorf 双部门名；手工 API 走 `UpsertMappingForce` + 预检告警

自动化证据：`go build ./...` exit 0；`go vet ./...` 仅存量 `internal/core/security/ad_authenticator_test.go:159` unreachable；`go test ./internal/services/addomain/ ./internal/api/v1/... ./internal/scheduler/ -count=1` 全 ok；HYGIENE-03 全仓 grep 死代码符号零残留。

环境：无浏览器 MCP，全人工验证。

## Current Test

[testing complete]

## Tests

### 1. AD 登录与部门映射功能未受死代码删除影响（HYGIENE-03 / SC1）

result: [pass]
automated: 部分（go build + 全量回归 + grep 零残留已通过）
evidence: 2026-10-22 12:57 用户实测 —— AD 部门对账视图（配置列表/对账 Tab）正常加载，「配置无法归属」的「用户 OU」不一致项正常呈现（chenxinyi-014 / yuyanbo / c_xiaochenfeng / caojun-024 等，属 Phase 59 GUARD-01 既有数据而非本次改动引起）；无 500 报错、无渲染异常。死代码删除侧自动化证据：go build ./... exit 0 + go test ./internal/services/addomain/ ./internal/api/v1/... ./internal/scheduler/ -count=1 全 ok + 全仓 grep -rnE "UserOUService|NewUserOUService|HandleUserLoginAD|GetUserDeptByADOU|BatchMoveUsersToNewOU|createDeptFromOUDN|generateUniqueDeptCode|restoreDeletedUserWithADInfo|setupUserOUSvcDB" internal/ cmd/ 零残留。
- 背景：`user_ou_service.go` 的 `HandleUserLoginAD` / `GetUserDeptByADOU` 等零生产调用删除后，登录路径实际走 `internal/services/system/user_sync_service.go`。需确认登录与部门映射不受影响。

steps:
1. 用 AD 域账号登录系统（如 d_chenchao 或其他域账号）
2. 确认登录成功且能进入系统
3. 进入「用户管理」，查看该用户的部门字段是否正确显示
4. 进入「AD 部门对账视图」，确认能正常加载且无报错

expected:登录成功、部门正确、对账视图正常加载。

### 2. 组成员列表不再混入嵌套组、total 不再虚高（HYGIENE-04 / SC2）

result: [pass]
automated: 部分（单元 + 集成测试已覆盖嵌套组不落库与全嵌套组边界）
evidence: 2026-10-22 14:35（UTC 06:35）轮全量同步 `sync_status=success`、`user_count=4177`、`group_count=67`、`error_count=0`、`duration=9s` —— GAP-62-01 修复版（14:22:12 构建）部署后首轮即通过。验证 SQL 结果：**nested_pollution 8 → 0**（部署前 8；14:05 失败轮后 7；14:35 成功轮后 0）；total_members 2815 → 2807（净减 8，与污染行数一致）；not_in_ad_user 仍有 18（已离开 sys_ad_user 的普通用户/计算机账号，属 PLAN flagged_assumptions 明确不在 HYGIENE-04 范围）。时间线：14:05 轮首次失败 → 14:22 重建二进制 → 14:32 部署 → 14:35 轮成功。
- 补充观察：06:05–06:30（14:05–14:30）连续 6 轮 failed，间隔恰好 5 分钟 —— 属既有「失败即重试」行为：`checkAndSyncADConfigs` 每 5 分钟 tick 一次（`0 */5 * * * *`），失败不更新 `last_sync_at`（仍为 03:05 上次成功时刻），故每 tick 都判定「已超 3 小时间隔」而重试；06:35 成功后 `last_sync_at=06:35:09` 恢复正常 3 小时周期。非本次改动引入。

background:生产 `sys_ad_group_member` 2815 行中 8 行为嵌套组 DN（CN=CXHUBinternet-VIP、CN=CXHUB-财务部(FGS) 等），导致 GetMembers total 虚高。部署后首次组同步重放应自动清除。

steps:
1. 等下一轮组同步（3 小时周期：02:05/05:05/08:05/11:05/14:05/17:05/20:05/23:05 +08）
2. 执行验证 SQL：

```sql
SELECT count(*) AS nested_pollution
FROM sys_ad_group_member m
WHERE EXISTS (
  SELECT 1 FROM sys_ad_group g
  WHERE g.ad_config_id = m.ad_config_id
  AND lower(g.group_dn) = lower(m.user_dn)
);
```

3. 进入「AD 域管理 → 组列表」，点开一个已知含嵌套组的组（如 CXHUBinternet-VIP 的父组），查看成员列表与总数

expected:
- SQL 返回 0（部署前为 8）
- 组成员列表不再出现组套组的组名，total 与实际成员数一致

### 3. 同名兄弟部门 OU DN 冲突显式告警（HYGIENE-05 / SC3）

result: [pass]
automated: 是（生产不可触发，见下）
evidence: 2026-10-22 14:50 核实 —— **部门同步在生产上无触发入口**：`ADDeptSyncHandler` 的三个方法（SyncDeptToAD / GetDeptSyncStatus / TriggerDeptSync）**未注册在任何 router**（internal/api/v1/system/ad_dept_sync_handler.go:31 注释原文「该 handler 当前未注册在任何 router（仅 test 引用）」；`grep -rn "ADDeptSyncHandler|SyncDeptToAD|dept-sync" internal/api` 仅命中该源文件与 ad_dept_sync_handler_test.go）。因此 `syncDeptTree` 的冲突分支无法通过生产 UI/API 触发。
  验证方式改为自动化：`TestSyncDeptTree_OUConflictCountedAsFailed`（dept_sync_conflict_test.go）以 mockLDAPClient + 真实 sqlite 映射表直调 `syncDeptTree`，断言冲突时 `FailedDepts==1`、`Errors[0].DeptID=="dept-B"`、`SuccessDepts==0`、原部门（dept-A）映射行原样保留 —— 已 PASS，日志输出「部门 OU DN 冲突: 部门 [人资部] 与部门 [财务部] 争用同一 OU DN OU=人资部,DC=corp，映射保留给 [财务部]，本次不同步该部门」。
  **生产实况补记（比原判断更值得关注）**：查询发现生产**已存在 4 组同名兄弟部门**（共 8 个部门，2026-01-29 与 2026-06-24 各建一套）：业务科×2（父 473f9890 / f0ab0d55）、个人营销业务销售部×2、综合管理科×2。每组**只有较早（01-29）那个有 OU 映射，较新（06-24）那个无映射** —— 正是旧代码「静默覆盖 / 只成功一次」留下的痕迹（当前 (ad_config_id, ou_dn) 重复=0 是因为旧逻辑删旧建新）。若将来注册部门同步入口，新代码会把这 4 个孤儿部门显式报为冲突失败而非再次静默覆盖。

steps:
1. 在「部门管理」创建一个部门，名称与某现有二级部门完全相同
2. 触发一次部门同步到 AD（或等待定时轮次）
3. 查看后端日志，确认出现「部门 OU DN 冲突: 部门 [X] 与部门 [Y] 争用同一 OU DN ...」的 ERRO 记录
4. 查看部门同步结果，确认该部门计入「失败」而非「成功」

expected:日志有双部门名冲突告警，同步结果中该部门计失败，原有部门的 OU 映射保留不被静默覆盖。

### 4. 手工 OU 重绑走 Force 且有预检告警（HYGIENE-05 / SC3 补充）

result: [pass]
automated: 是（本轮补齐 handler 层测试，消除原「手工 API 路径无测试覆盖」空白）
evidence: 2026-10-22 14:50 新增 internal/api/v1/system/ou_mapping_handler_test.go（sqlite 建 sys_dept / sys_dept_ou_mapping / sys_ad_config 三表 + 两部门 + 启用配置 + 预置 OU=TestOU→dept-A 映射），两个用例全 PASS：
  - `TestUpdateOUDeptMapping_ForceRebindReplacesExisting`：把 OU 重绑到 dept-B → HTTP 200 + 「关联成功」，DB 中该 OU **仍只有 1 行**且 `dept_id=dept-B`（Force 覆盖生效）。用例日志逐条出现预检 WARN「手工重绑 OU OU=TestOU,DC=corp: 原部门 dept-A → 新部门 dept-B（操作员显式覆盖，原映射将被替换）」与 Force WARN「OU DN ... 已映射到部门 dept-A，现在重新映射到部门 dept-B（显式 Force 覆盖）」，即留痕路径被真实执行。
  - `TestUpdateOUDeptMapping_MissingDeptRejected`：不存在的 deptId → HTTP 400，且原映射 dept-A 未被改动（校验失败不落库）。
  回归：`go build ./...` + `go test ./internal/api/v1/system/ ./internal/services/addomain/ ./internal/scheduler/ -count=1` → ok 3.97s / 7.88s / 9.89s。
- 背景：`POST /system/ou/:ouDn/dept-mapping` 现在先预检（GetMappingByOU）并 Warnf 留痕，再走 `UpsertMappingForce` 覆盖。

steps:
1. 找一个已有 OU 映射的 OU，记录其当前部门
2. 在「AD 域管理 → OU 部门映射」，把该 OU 重绑到另一个部门并提交
3. 查看后端日志，确认出现「手工重绑 OU %s: 原部门 %s → 新部门 %s（操作员显式覆盖，原映射将被替换）」的 WARN
4. 确认接口返回成功且映射已切换到新部门

expected:预检告警可见（留痕），操作成功，原映射被替换。

## Gaps

### GAP-62-01（CRITICAL，已修复待部署复测）：全嵌套成员的组导致 GORM "empty slice found"，整轮同步失败

**发现**：2026-10-22 14:05 轮（UTC 06:05）全量同步 `sync_status=failed`、`error_count=1`、`user_count=0`，`error_message` = `批量添加组成员失败: empty slice found`。

**影响**：HYGIENE-04 过滤引入的生产回归。`syncGroupMembers` 在 Delete 后过滤嵌套组 DN，若某组成员**全部**是嵌套组（生产实例：`CN=CXHUB-人资条线,OU=外网权限,OU=MgtGrp,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn`，唯一成员本身是组 DN），过滤后 `kept` 为空 → `tx.Create(&[])` 返回 GORM `ErrEmptySlice` → 事务回滚 → 错误沿 `syncGroups`（sync.go:408 上抛）中止整轮同步。14:05 轮因此 user_count=0（该轮用户同步未执行）。

**根因定位**：`internal/services/addomain/sync.go:704-715` —— `len(memberDNs) == 0` 的早返回只存在于过滤**之前**（:688），过滤后未再判空。

**修复**：过滤块之后补早返回（Delete 已在事务内生效，直接提交）：

```go
// HYGIENE-04 边界:组成员可能"全部"是嵌套组(生产实例 CN=CXHUB-人资条线),
// 过滤后为空时 Create 会返回 GORM ErrEmptySlice("empty slice found"),
// 事务回滚并让整轮 syncGroups 中止。Delete 已在事务内生效,直接提交即可。
if len(memberDNs) == 0 {
    return nil
}
```

**回归用例**：`TestSyncGroupMembers_AllMembersNestedNoEmptySliceError`（group_member_filter_test.go）—— 预置唯一成员为嵌套组 DN + 历史行，断言不返回错误且该组零行。**RED→GREEN 已验证**：临时移出修复后该用例报出与生产完全一致的 `批量添加组成员失败: empty slice found`，恢复后全绿。

**验证**：`go build ./...` exit 0；`go test ./internal/services/addomain/ ./internal/scheduler/ -count=1` → ok 8.1s / 10.2s。

**收口（2026-10-22 14:36）**：14:22:12 构建的二进制已由用户部署到 10.62.10.39；14:35（UTC 06:35）轮即 `success`（user_count=4177、error_count=0），嵌套组污染 `nested_pollution` 8 → 0。**修复已生产验证。**

## Summary

**4/4 pass**（2026-10-22 14:55）。

| # | 测试 | 需求 | 结果 | 验证方式 |
|---|------|------|------|----------|
| 1 | AD 登录与部门映射未受死代码删除影响 | HYGIENE-03 / SC1 | pass | 人工（对账视图）+ 自动化（build/回归/grep 零残留） |
| 2 | 组成员列表不再混入嵌套组、total 不虚高 | HYGIENE-04 / SC2 | pass | 生产验证（14:35 轮 success，污染 8→0） |
| 3 | 同名兄弟部门 OU DN 冲突显式告警 | HYGIENE-05 / SC3 | pass | 自动化（生产无触发入口） |
| 4 | 手工 OU 重绑走 Force 且有预检告警 | HYGIENE-05 / SC3 补充 | pass | 自动化（本轮补齐 handler 测试） |

**测试期间发现并修复的生产缺陷 1 项**：GAP-62-01（全嵌套成员组触发 GORM ErrEmptySlice → 整轮同步失败），已修复、已生产验证（14:35 轮 success，污染 8→0）。
