---
phase: 62-hygiene-cleanup
verified: 2026-10-22T14:58:00+08:00
updated: 2026-10-22T14:58:00+08:00
status: passed
score: 18/18 must-haves verified
covered_files:
  - ".planning/phases/62-hygiene-cleanup/62-01-PLAN.md"
  - ".planning/phases/62-hygiene-cleanup/62-01-SUMMARY.md"
  - ".planning/phases/62-hygiene-cleanup/62-02-PLAN.md"
  - ".planning/phases/62-hygiene-cleanup/62-02-SUMMARY.md"
  - ".planning/phases/62-hygiene-cleanup/62-03-PLAN.md"
  - ".planning/phases/62-hygiene-cleanup/62-03-SUMMARY.md"
  - "internal/api/v1/auth/auth_handler_test.go"
  - "internal/api/v1/system/ou_mapping_handler.go"
  - "internal/api/v1/system/ou_mapping_handler_test.go"
  - "internal/services/addomain/dept_ou_mapper.go"
  - "internal/services/addomain/dept_ou_mapper_atomic_test.go"
  - "internal/services/addomain/dept_sync_conflict_test.go"
  - "internal/services/addomain/dept_sync_service.go"
  - "internal/services/addomain/group_member_filter.go"
  - "internal/services/addomain/group_member_filter_test.go"
  - "internal/services/addomain/group_sync_service.go"
  - "internal/services/addomain/ldap_iface.go"
  - "internal/services/addomain/sync.go"
  - "internal/services/addomain/test_fixtures_test.go"
  - "internal/services/addomain/user_ad_sync_service.go"
covered_digest: "v3:sha256:b73782ff456f573575bc431d1a7b908ca93fcc28b98ed40af046ec8b32361c7c"
behavior_unverified: 0
overrides_applied: 0
human_verification:
  - test: "HYGIENE-03：AD 登录与部门映射功能未受死代码删除影响"
    expected: "AD 账号可正常登录，用户管理部门字段正确，部门对账视图正常加载并呈现不一致项"
    why_human: "真实域控 + 生产部署环境的端到端登录链路，sqlite 单测无法覆盖"
    resolution: "PASS（2026-10-22 人工 UAT Test 1）：用户登录后在部门对账视图确认正常，名片与不一致项列表（chenxinyi-014 / yuyanbo / c_xiaochenfeng 等 12 项，属 Phase 59 GUARD-01 既有数据）呈现正常；证据在 62-UAT.md Test 1"
  - test: "HYGIENE-04：部署后组同步重放自动清除嵌套组污染"
    expected: "sys_ad_group_member 中 user_dn 命中 sys_ad_group.group_dn 的行数 8 → 0，且该轮 sync_status=success"
    why_human: "依赖生产 PG 存量污染数据与真实 AD 重放，本地 fixture 只能证伪过滤逻辑本身"
    resolution: "PASS（2026-10-22 14:35 生产轮）：06:35:00 UTC full/success（ou=3590 group=67 user=4177 error=0 dur=9s）；nested_pollution 8→7（14:05 失败轮后）→0；total_members 2815→2808→2807；not_in_ad_user 仍 18（已离开 sys_ad_user 的普通账号，flagged_assumption 明确不在 HYGIENE-04 范围）；证据在 62-UAT.md Test 2"
  - test: "HYGIENE-05：同名兄弟部门 OU DN 冲突显式告警不静默覆盖"
    expected: "syncDeptTree 遇冲突时 FailedDepts++ 且 Errorf 同时含两个部门名，incoming 部门映射不落库、incumbent 映射保留"
    why_human: "生产部门同步无 API/UI 入口（ad_dept_sync_handler 未注册路由），无法生产触发"
    resolution: "PASS（自动化验证）：TestSyncDeptTree_OUConflictCountedAsFailed 断言 FailedDepts==1 / Errors[0].DeptID==dept-B / SuccessDepts==0 / dept-A 映射行保留，日志含双部门名；另发现生产已存在 4 组同名兄弟部门（业务科/个人营销业务销售部/综合管理科），每组仅 01-29 部门有映射、06-24 部门无映射——旧静默覆盖痕迹，本改动后此类冲突将显式失败；证据在 62-UAT.md Test 3"
  - test: "HYGIENE-05：手工 OU 重绑 API 的显式 Force 语义与操作留痕"
    expected: "重绑成功返回 200 且该 OU 仅一行（新部门），预检与 Force 各留一条 WARN，operlog 照旧记录"
    why_human: "真实操作员重绑行为与操作日志落库"
    resolution: "PASS（自动化验证，本轮补齐 handler 测试）：新增 internal/api/v1/system/ou_mapping_handler_test.go —— TestUpdateOUDeptMapping_ForceRebindReplacesExisting（200/关联成功/单行 dept-B/两条 WARN 逐字匹配）、TestUpdateOUDeptMapping_MissingDeptRejected（400 且原映射不变）；recordOperLog 调用未改动且 helper.go:39 有 core==nil 守卫；证据在 62-UAT.md Test 4"
---

# Phase 62: 卫生收尾与死代码清理（HYGIENE-03/04/05）Verification Report

**Phase Goal:** F-22 旧路径死代码在 SAFE-01 落地后安全移除；嵌套组不污染用户成员关系；同名兄弟部门 OU DN 冲突显式告警不静默覆盖。
**Verified:** 2026-10-22T14:58:00+08:00
**Status:** passed（18/18 must-haves verified；4 项人工/自动化 UAT 全 PASS；UAT 期间发现并修复 1 项生产回归 GAP-62-01）
**Re-verification:** Yes — 首轮验证发现 GAP-62-01（生产回归），修复并重新发版后复验通过

## Goal Achievement

### Observable Truths（18 条 = 62-01 五条 + 62-02 六条 + 62-03 七条）

| # | Plan | Truth | Status | Evidence |
|---|------|-------|--------|----------|
| 1 | 62-01 | user_ou_service.go 及测试文件从仓库消失，死符号全仓零命中 | ✓ VERIFIED | `grep -rnE "UserOUService|NewUserOUService|HandleUserLoginAD|GetUserDeptByADOU|createDeptFromOUDN|generateUniqueDeptCode|restoreDeletedUserWithADInfo|updateUserDeptAndADInfo|setupUserOUSvcDB" internal/ cmd/ --include="*.go"` → **ZERO_HITS**（含注释在内完全零残留，auth_handler_test.go 6 处注释亦已改写） |
| 2 | 62-01 | BatchMoveUsersToNewOU 已删除；moveSingleUserToOU 保留且 F-A 载体用例通过 | ✓ VERIFIED | `grep -c "func (s *UserADSyncService) moveSingleUserToOU"` = 1；BatchMoveUsersToNewOU 全仓仅剩 internal/services/addomain/dept_ou_mapper.go:74 描述"已删除"的注释（非代码）；`TestPhase58_MoveSingleUserToOU_UpdatesAdDn`（ad_push_queue_drain_test.go）在全包回归中通过 |
| 3 | 62-01 | 三个共享测试 const 逐字节存在于 test_fixtures_test.go，复用方零改动可编译 | ✓ VERIFIED | 程序化从 user_ou_service_test.go 抽取 slice(11,61) + slice(202,215) 原样写入 test_fixtures_test.go（createDeptOUMappingTable / createUserTable / createDeptTable）；dept_ou_mapper_test.go:379、ad_push_queue_drain_test.go:83-85 复用点未改动，`go test ./internal/services/addomain/` 编译并全绿 |
| 4 | 62-01 | DeptOUmapper 活方法保留、签名零变化 | ✓ VERIFIED | dept_ou_mapper.go：FindDeptByOUDN:33 / FindOUDNByDeptID:51 / FindOUDNByDeptIDForConfig:75 / UpsertMapping:120 全在且签名未变（另 UpsertMappingForce:127 与 GetMappingByOU:203 为 62-03 新增） |
| 5 | 62-01 | go build ./... 通过；addomain + api/v1/auth 测试全绿 | ✓ VERIFIED | `go build ./...` exit 0；`go test ./internal/services/addomain/ ./internal/api/v1/auth/ -count=1` → ok 8.052s / 0.360s |
| 6 | 62-02 | 生产实锤的 8 个嵌套组 DN 不再写入 sys_ad_group_member | ✓ VERIFIED | 本地：`TestSyncGroupMembers_NestedGroupMemberNotPersisted`、`TestFilterNestedGroupMemberDNs_{Basic,CaseInsensitive,NilSetKeepsAll,EmptyMember}` 全绿；生产：14:35 轮后 nested_pollution **8→0**（SQL 见 62-UAT.md Test 2） |
| 7 | 62-02 | 过滤基于"本轮已知 group DN 集合"（每轮一次查询），比较大小写不敏感 | ✓ VERIFIED | group_member_filter.go `loadKnownGroupDNSet`：`Model(&models.ADGroup{}).Where("ad_config_id = ? AND deleted_at IS NULL", configID).Pluck("group_dn", &dns)` 每轮一次，键经 `strings.ToLower(strings.TrimSpace(dn))` 归一；`TestFilterNestedGroupMemberDNs_CaseInsensitive` 锁死大小写不敏感。**实现偏差披露**：plan truth 写 strings.EqualFold，实际用"小写归一 + map 查表"，语义等价且更快（每次比较不再重复 ToLower） |
| 8 | 62-02 | 集合 nil 时不过滤；三条生产调用路径全部显式加载，无默认放行路径 | ✓ VERIFIED | syncGroupMembers 内 nil→`applogger.Warnf("未加载组 DN 集合…不过滤")` 防御；三调用点均显式加载：sync.go syncGroups（建组后、成员循环前，错误上抛）、group_sync_service.go SyncSingleGroup（错误上抛）、syncGroupEntries（失败 Warnf 降级沿该路径既有非致命语义）；`TestSyncGroupMembers_NilSetKeepsAll` 覆盖 nil 分支 |
| 9 | 62-02 | 既有三条 syncGroupMembers 用例语义不变、仍全绿 | ✓ VERIFIED | `TestSync78_SyncGroupMembers_FullDiff` / `TestSyncGroupMembers_CreateFailRollsBackDelete` / `TestSyncGroupMembers_ReplayIdempotent` 全 PASS（6 处直调仅补第 5 参 nil，断言零改动） |
| 10 | 62-02 | ADGroup.MemberCount 保持 LDAP 原始 member 数 | ✓ VERIFIED | 未触碰 MemberCount 赋值（group_sync_service.go:143 / sync.go:367 保持 `len(members)`）；过滤仅作用于 sys_ad_group_member 落库，不改组计数语义（flagged_assumption 已记录） |
| 11 | 62-02 | 零 DDL：无 migration、无新列 | ✓ VERIFIED | `git status --porcelain internal/core/db/migrations/` 为空；grep is_known_group / member_type 零命中 |
| 12 | 62-03 | UpsertMapping 冲突返回 *OUDNConflictError 且不删除，第一部门映射不再被静默吞掉 | ✓ VERIFIED | dept_ou_mapper.go 冲突分支改为 `return &OUDNConflictError{...}`（无 Delete）；`TestUpsertMapping_DifferentOUTriggersDeleteAndInsert` 已翻转：断言 `errors.As(err, &conflict)` + incumbent dept-A 行原样 + incoming dept-B 无行 |
| 13 | 62-03 | ErrOUDNConflict 哨兵可 errors.Is / errors.As 识别，Unwrap 返回哨兵 | ✓ VERIFIED | `var ErrOUDNConflict = errors.New("OU DN 已映射到其他部门")` + `func (e *OUDNConflictError) Unwrap() error { return ErrOUDNConflict }`；用例同时断言 errors.Is 与 errors.As |
| 14 | 62-03 | UpsertMappingForce 保留原"告警 + 删旧 + 建新"语义，仅供手工重绑 | ✓ VERIFIED | `UpsertMappingForce` 委托 `upsert(allowRebind=true)`，保留 `applogger.Warnf("OU DN %s 已映射到部门 %s，现在重新映射到部门 %s（显式 Force 覆盖）")`；`TestUpsertMapping_ForceRebindReplacesOld` PASS；`TestUpsertMapping_AtomicDeleteInsert`（P1 事务回滚语义）改走 Force 以保持原覆盖度；生产调用方仅 ou_mapping_handler.go |
| 15 | 62-03 | dept_sync 冲突 FailedDepts++ 并追加 DeptSyncError，Errorf 含双部门名，SuccessDepts 不自增 | ✓ VERIFIED | dept_sync_service.go syncDeptTree 内 errors.As 分流：Errorf「部门 OU DN 冲突: 部门 [%s] 与部门 [%s] 争用同一 OU DN %s，映射保留给 [%s]…」+ FailedDepts++ + Errors append；`TestSyncDeptTree_OUConflictCountedAsFailed` 断言三项计数与日志逐字吻合；其余错误保持原 Warnf 非致命 |
| 16 | 62-03 | ou_mapping_handler.UpdateOUDeptMapping 走 Force + 冲突前 Warnf 记录 incumbent + operlog 照旧 | ✓ VERIFIED | handler 内先 `GetMappingByOU` 预检（异 dept → Warnf「手工重绑 OU %s: 原部门 %s → 新部门 %s（操作员显式覆盖…）」）再 `UpsertMappingForce`；`recordOperLog(c, h.core, "OU部门映射", OperTypeUpdate)` 未改动；新增 handler 测试两用例 PASS |
| 17 | 62-03 | dept_ou_mapper_atomic_test.go 翻转断言 | ✓ VERIFIED | 该用例断言改为"冲突返回错误 + 旧行原样 + 新行不存在"（注释说明翻转依据）；同文件 `TestUpsertMapping_TransactionSuccess` / `AtomicDeleteInsert` 仍全绿 |
| 18 | 62-03 | go build ./...；addomain + api/v1/... 测试全绿 | ✓ VERIFIED | `go build ./...` exit 0；`go test ./internal/api/v1/system/ ./internal/services/addomain/ ./internal/scheduler/ -count=1` → ok 3.97s / 7.88s / 9.89s；`go vet ./...` 仅 1 处存量告警（internal/core/security/ad_authenticator_test.go:159 unreachable code，非本 phase 影响面，62 之前即存在） |

**Score:** 18/18 truths verified（0 present-behavior-unverified；全部行为依赖型真相均有通过的命名行为测试或生产数据佐证）

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| internal/services/addomain/group_member_filter.go | 纯函数 + 集合加载器 | ✓ VERIFIED | 新建；filterNestedGroupMemberDNs（kept/filtered 双返回、nil 全保留）+ loadKnownGroupDNSet（Pluck + 小写归一 + 软删排除） |
| internal/services/addomain/test_fixtures_test.go | 承接三个共享测试 const | ✓ VERIFIED | 新建；createDeptOUMappingTable / createUserTable / createDeptTable 逐字节迁入（DDL 内容零改动，未加 UNIQUE(ad_config_id, ou_dn)） |
| internal/services/addomain/group_member_filter_test.go | 纯函数单测 + 集成用例 | ✓ VERIFIED | 新建；5 条纯函数用例 + 4 条集成用例（含 GAP-62-01 回归用例 TestSyncGroupMembers_AllMembersNestedNoEmptySliceError） |
| internal/services/addomain/dept_sync_conflict_test.go | syncDeptTree 冲突计失败用例 | ✓ VERIFIED | 新建；sqlite + mockLDAPClient 驱动，断言 FailedDepts/Errors/SuccessDepts 与 incumbent 映射留存 |
| internal/api/v1/system/ou_mapping_handler_test.go | 手工重绑 Force 用例 | ✓ VERIFIED | 新建（该 handler 此前零测试）；2 用例覆盖 200 重绑成功与 400 部门不存在 |
| internal/services/addomain/dept_ou_mapper.go | OUDNConflictError + Force 变体 | ✓ VERIFIED | ErrOUDNConflict 哨兵、OUDNConflictError（含 OUDN/ADConfigID/双部门 ID+Name）、Unwrap、UpsertMapping 严格化、UpsertMappingForce；四个既有活方法签名零变化 |
| internal/services/addomain/sync.go | 组成员过滤接线 + 全嵌套组边界 | ✓ VERIFIED | syncGroupMembers 第 5 参 + 过滤 + 过滤后早返回（GAP-62-01 修复）；syncGroups 每轮加载集合 |
| internal/services/addomain/dept_sync_service.go | 冲突计入 FailedDepts | ✓ VERIFIED | errors.As 分流，Errorf 双部门名，树不中断 |
| internal/api/v1/system/ou_mapping_handler.go | 手工 Force + 预检 | ✓ VERIFIED | GetMappingByOU 预检 Warnf + UpsertMappingForce；响应形态与 operlog 未变 |
| internal/services/addomain/user_ou_service.go / _test.go | 整文件删除 | ✓ VERIFIED | 两文件已从工作区与 git 索引移除 |

### Key Link Verification

| From | To | Via | Status |
| ---- | -- | --- | ------ |
| sync.go syncGroups | group_member_filter.go loadKnownGroupDNSet | `s.loadKnownGroupDNSet(ctx, config.ID)`（成员循环前一次） | ✓ WIRED |
| group_sync_service.go SyncSingleGroup / syncGroupEntries | loadKnownGroupDNSet | `s.syncService.loadKnownGroupDNSet(...)` | ✓ WIRED |
| dept_sync_service.go syncDeptTree | dept_ou_mapper.go ErrOUDNConflict | `errors.As(err, &conflict)` | ✓ WIRED |
| ou_mapping_handler.UpdateOUDeptMapping | UpsertMappingForce | 直接调用（替换原 UpsertMapping） | ✓ WIRED |
| ROADMAP Phase 62 SC1/SC2/SC3 | 三个 PLAN 的 requirements | HYGIENE-03/04/05 全覆盖 | ✓ WIRED |

### Behavioral Tests（本轮独立复跑）

| Test | Result |
| ---- | ------ |
| TestFilterNestedGroupMemberDNs_{Basic,CaseInsensitive,NilSetKeepsAll,EmptyMember} | PASS |
| TestSyncGroupMembers_{NestedGroupMemberNotPersisted,ReplayPurgesLegacyNestedRow,NilSetKeepsAll,AllMembersNestedNoEmptySliceError} | PASS |
| TestLoadKnownGroupDNSet_ExcludesSoftDeleted | PASS |
| TestSync78_SyncGroupMembers_FullDiff / TestSyncGroupMembers_CreateFailRollsBackDelete / TestSyncGroupMembers_ReplayIdempotent | PASS（语义零漂移） |
| TestUpsertMapping_{DifferentOUTriggersDeleteAndInsert,ForceRebindReplacesOld,ConflictCarriesDeptNames,SameDeptUpdateStillWorks,AtomicDeleteInsert,TransactionSuccess} | PASS |
| TestSyncDeptTree_OUConflictCountedAsFailed | PASS |
| TestUpdateOUDeptMapping_{ForceRebindReplacesExisting,MissingDeptRejected} | PASS |
| TestPhase58_MoveSingleUserToOU_UpdatesAdDn（死代码删除后 F-A 载体仍在） | PASS |
| 全量：`go test ./internal/api/v1/system/ ./internal/services/addomain/ ./internal/scheduler/ -count=1` | ok 3.97s / 7.88s / 9.89s |
| 构建：`go build ./...` / Linux 发版 `scripts/build-linux.bat` | exit 0 / 14:22:12 · 62,603,448 B |

### Requirements Coverage

| Requirement | Plan | Status | Evidence |
| ----------- | ---- | ------ | -------- |
| HYGIENE-03 死代码清理 | 62-01 | ✓ SATISFIED | 两文件删除 + BatchMoveUsersToNewOU 删除 + 3 处过期注释改写 + 全仓死符号 ZERO_HITS |
| HYGIENE-04 嵌套组 DN 标记/过滤 | 62-02 | ✓ SATISFIED | 零 DDL 过滤 + 生产 nested_pollution 8→0 + GAP-62-01 边界修复 |
| HYGIENE-05 OU DN 冲突显式告警 | 62-03 | ✓ SATISFIED | typed error + FailedDepts 计数 + 手工 Force 语义 + 原子测试翻转 |

## Deviations & Findings

### GAP-62-01（生产回归，UAT 期间发现并已修复）

- **现象**：14:05 轮全量同步 failed，error_message=`批量添加组成员失败: empty slice found`，连续 6 轮（06:05–06:30 UTC）复现；nested_pollution 8→7 后停滞。
- **根因**：62-02 的过滤使某组成员"全部"为嵌套组（生产实例 `CN=CXHUB-人资条线,OU=外网权限,OU=MgtGrp,…`，成员数 1 且为组 DN）时过滤后为空；原 `len(memberDNs)==0` 早返回只在过滤**之前**，过滤后 `tx.Create(&[])` 返回 GORM ErrEmptySlice → 事务回滚 → 错误上抛中止整轮 syncGroups。
- **修复**：sync.go 过滤块之后补早返回（Delete 已在事务内生效）：
  ```go
  // HYGIENE-04 边界:组成员可能"全部"是嵌套组(生产实例 CN=CXHUB-人资条线),
  // 过滤后为空时 Create 会返回 GORM ErrEmptySlice("empty slice found"),
  // 事务回滚并让整轮 syncGroups 中止。Delete 已在事务内生效,直接提交即可。
  if len(memberDNs) == 0 {
      return nil
  }
  ```
- **RED→GREEN**：临时移出修复 → `TestSyncGroupMembers_AllMembersNestedNoEmptySliceError` 报出与生产逐字一致的错误；恢复后全绿。
- **生产验证**：14:22 重建 / 14:32 部署 / **14:35 轮（06:35 UTC）full + success**（ou=3590 group=67 user=4177 error=0 dur=9s），nested_pollution **→0**。

### 其他偏差（均已在 SUMMARY 记录）

1. **62-02 truth 措辞**：plan 写 strings.EqualFold，实现为小写归一 + map 查表（语义等价，性能更优，有 CaseInsensitive 用例锁定）。
2. **62-03 额外改动**：`TestUpsertMapping_AtomicDeleteInsert` 改走 UpsertMappingForce —— 若不同步改，该用例将不再覆盖 P1"Delete 后 Create 失败整体回滚"语义（严格化后异 dept 分支不再删除）。属保持覆盖度的必要调整。
3. **62-01 附带**：test_fixtures_test.go 注释原本含 buildAncestors / generateUniqueDeptCode 字样（会污染 grep 零命中判定），改写为"由 user_ou_service_test.go 迁入"。
4. **62 未新增 handler 入口**：部门同步 handler（ad_dept_sync_handler.go）仍未注册任何路由，HYGIENE-05 的同步路径在生产只能由程序内调用触发；手工 API 路径（ou_mapping 重绑）已有 UI 入口且经 handler 测试覆盖。

## Assumptions（沿用 62-02 PLAN flagged_assumptions）

- **A1**：sys_ad_group_member 中另有 18 行 user_dn 不在 sys_ad_user 且非组 DN（已离开 AD 的普通用户/计算机账号）——不属 HYGIENE-04 范围，本轮未处理，重放会在这些对象仍存在于 LDAP 时保留；若 AD 侧已删则下一轮自然清除。
- **A2**：ADGroup.MemberCount 仍记 LDAP 原始 member 数（含嵌套组），组列表的 memberCount 与可查成员数存在差额 —— 有意保留（ROADMAP SC2 只要求"不混入用户成员关系数据"）。
- **A3**：生产 Sys_dept_ou_mapping 当前 (ad_config_id, ou_dn) 重复=0，但已存在 4 组同名兄弟部门，其中 06-24 那批无映射 —— 本改动后若再跑部门同步，这些会显式计失败（而非静默覆盖），需运维决定是否合并同名部门。

## 结论

Phase 62 三条 Success Criteria 全部达成：死代码归零（SC1）、嵌套组不再污染成员关系且生产存量已自愈归零（SC2）、同名兄弟部门冲突由静默覆盖转为显式失败告警（SC3）。UAT 4/4 PASS，期间捕获并修复 1 项生产回归（GAP-62-01），修复经生产轮次验证。**status: passed**。
