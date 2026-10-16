---
phase: 38-unify-ad-authentication-onto-account-pool-remove-legacy-sing
plan: 02
subsystem: infra
tags: [addomain, account-pool, failover-client, connection-layer, refactor, wave-1]

# Dependency graph
requires:
  - phase: 38-01 (Wave 1 前置 DI 脚手架)
    provides: 8 个服务 struct 的 pool AccountPool 字段 + NewADDomainService(db,pool,cipher) 签名
provides:
  - "所有 AD 连接统一经账号池 FailoverClient（ExecuteWithFailover / PickFirstConnect），消除单管理员单点"
  - "globalADSyncScheduler.pool 全局账号池单例字段（供 scheduler 各 task + dept_sync_tasks 复用，W-04）"
  - "NewLDAPClient(config) 单参数直连生产调用 = 0（D-01 Wave 1 硬指标达成）"
affects:
  - 38-03-remove-legacy-admin-usage (Wave 2：删除 22 处 decryptPassword(config.AdminPassword) + 前端 admin 字段)
  - 38-04-model-migration-cleanup (Wave 3：model struct tag + migration_164 补迁)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "FailoverClient.ExecuteWithFailover(ctx, func(client *LDAPClient) error {...}) 闭包封装（单次操作型 + 批量同步型）"
    - "FailoverClient.PickFirstConnect(ctx) 配置测试连接专用（只建连接不操作）"
    - "operation 闭包边界 = 整个批量任务（SP-3：非每用户），单批量失败 MarkFailure 多账号是可接受语义（T-38-06 accept）"
    - "scheduler 全局 pool 单例 + getPool() 惰性初始化兜底（W-04 / Pitfall 4）"
    - "ErrAllAccountsUnavailable 哨兵错误包装为引导文本（SHA-3：引导用户到 AD 配置页账号池 Tab）"

key-files:
  created: []
  modified:
    - internal/services/addomain/user.go (Enable/Disable/Move/Update 4 处 ExecuteWithFailover)
    - internal/services/addomain/group.go (AddMember/RemoveMember/Update 3 处)
    - internal/services/addomain/group_management_service.go (CreateGroupForDept/DeleteGroup/AddMembers/RemoveMembers 4 处)
    - internal/services/addomain/group_management_service_test.go (nil pool → NewAccountPool(db,nil) 适配)
    - internal/services/addomain/sync.go (syncDataInternal 整个同步流程一个 operation)
    - internal/services/addomain/group_sync_service.go (SyncGroupsByConfig/SyncSingleGroup)
    - internal/services/addomain/dept_sync_service.go (SyncDeptStructureToAD 整个递归树一个 operation)
    - internal/services/addomain/user_ad_sync_service.go (SyncUserUpdateToAD/BatchMoveUsersToNewOU/SyncManagersToAD；SHA-5 测试钩子分支保留)
    - internal/services/addomain/config.go (TestConnection 改 PickFirstConnect)
    - internal/services/addomain/service.go (文档注释更新)
    - internal/scheduler/ad_sync_tasks.go (ADSyncScheduler.pool 字段 + StartADSyncScheduler 赋值 + getPool() helper + syncADConfig/executeADDataSyncTask 复用)
    - internal/scheduler/dept_sync_tasks.go (两处任务改 FailoverClient，getGlobalADAccountPool helper 复用全局单例；NewAccountPool count=0)

key-decisions:
  - "sync.go 闭包边界 = 所有 LDAP Search（OU/Group/User/Computer）一个 operation；DB 写入（syncOUs/syncGroups/syncUsers/syncComputers）放闭包外，Search 结果 []*ldap.Entry 在闭包内 collect 后传出（Pitfall 3）"
  - "SyncManagersToAD 测试钩子分支完全保留（SHA-5）：updateUserAttributeFn != nil 时绕过 FailoverClient 走独立 errgroup；否则 FailoverClient 闭包内启动 errgroup + g.Wait()（Pitfall 3+5）。两分支代码重复但语义清晰，合并会破坏钩子可测试性"
  - "scheduler 全局 pool 单例（W-04）：globalADSyncScheduler.pool 在 StartADSyncScheduler 中创建，syncADConfig/executeADDataSyncTask/dept_sync_tasks 全部复用；getPool() 惰性初始化兜底 StartADSyncScheduler 未执行的极端场景"
  - "dept_sync_tasks ldapClient 参数传 nil：SyncDeptStructureToAD 已改用 FailoverClient，不再使用注入的 ldapClient；保留 NewDeptToADSyncService 签名兼容（38-04 可清理 ldap 字段）"
  - "decryptPassword(config.AdminPassword) 22 处全部保留：38-03 Wave 2 统一删除（保留不影响 FailoverClient 正确性，因 FailoverClient 内部用 account.PasswordCiphertext）"

patterns-established:
  - "单次操作型改造模板：fc.ExecuteWithFailover(ctx, func(client *LDAPClient) error { return client.XxxUser(...) }) + errors.Is(ErrAllAccountsUnavailable) 引导文本"
  - "批量同步型改造模板：闭包边界 = 整个批量；g.Wait() 必须在闭包内（Pitfall 3+5）；DB 写不依赖 LDAP 的可放闭包外"
  - "scheduler 外部包访问全局 pool：getGlobalADAccountPool() helper（禁止临时 NewAccountPool）"

requirements-completed: [D-01, D-03]

# Metrics
duration: 35min
completed: 2026-06-23
---

# Phase 38 Plan 02: Wave 1 主体（FailoverClient 改造）Summary

**将 22 处 NewLDAPClient(config) 单管理员直连生产调用全部改为 FailoverClient.ExecuteWithFailover / PickFirstConnect（账号池多账号故障切换），消除单管理员单点（data 775 锁定即全断），D-01 Wave 1 硬指标达成**

## Performance

- **Duration:** ~35 min
- **Started:** 2026-06-23T10:55:00Z
- **Completed:** 2026-06-23T11:30:00Z
- **Tasks:** 3（每 Task 独立 commit + build + test）
- **Files modified:** 12（9 服务文件 + 2 scheduler + 1 测试）

## Accomplishments

- **22 处 NewLDAPClient(config) 生产调用全部改造为 FailoverClient 路径**（grep 硬指标 = 0，仅 failover_client.go 内部 NewLDAPClient(f.config, acct) 变体保留）
- **D-01 Wave 1 核心收益落地**：单账号被 AD 锁定（data 775）不再阻断所有用户登录/同步/管理操作，FailoverClient 自动切换到池中其他可用账号
- **W-04 全局 pool 单例**：globalADSyncScheduler.pool 字段 + StartADSyncScheduler 赋值，syncADConfig/executeADDataSyncTask/dept_sync_tasks 全部复用（Pitfall 4 缓存共享）
- **SHA-5 测试钩子保留**：updateUserAttributeFn != nil 分支绕过 FailoverClient，7 个 TestSyncManagersToAD_* 回归测试全绿
- **SHA-3 引导文本**：ErrAllAccountsUnavailable 在所有 caller 包装为"AD 账号池无可用账号，请先在 AD 配置页（详情 → 服务账号池 Tab）添加服务账号"
- **SP-3 operation 边界**：批量同步（sync.go / SyncManagersToAD / BatchMoveUsersToNewOU / SyncDeptStructureToAD / executeDeptMemberToADGroupSyncTask）闭包 = 整个批量任务，非每用户
- **SP-4 PickFirstConnect**：config.go TestConnection 只建连接+bind 成功即返回，不做后续操作
- **decryptPassword 22 处保留**（38-03 Wave 2 删除，保留不影响 FailoverClient 正确性）

## Task Commits

Each task was committed atomically:

1. **Task 1: 单次操作型服务改造（user.go + group.go + group_management_service.go，11 处）** - `03c04b8` (refactor)
2. **Task 2: 批量同步型服务改造 + scheduler 暴露全局 pool（6 文件）** - `b397f6f` (refactor)
3. **Task 3: config.go TestConnection 改 PickFirstConnect + Wave 1 硬指标达成** - `355f1b3` (refactor)

## Files Created/Modified

- `internal/services/addomain/user.go` - Enable/Disable/Move/Update 4 处 NewLDAPClient(config) → fc.ExecuteWithFailover
- `internal/services/addomain/group.go` - AddMember/RemoveMember/Update 3 处同上
- `internal/services/addomain/group_management_service.go` - CreateGroupForDept/DeleteGroup/AddMembers/RemoveMembers 4 处同上
- `internal/services/addomain/group_management_service_test.go` - nil pool → NewAccountPool(db,nil) 适配（避免空池 panic）
- `internal/services/addomain/sync.go` - syncDataInternal 整个同步流程（4 个 Search）封装进一个 ExecuteWithFailover operation；Search 结果 []*ldap.Entry 在闭包内 collect 后传出，DB 写入放闭包外
- `internal/services/addomain/group_sync_service.go` - SyncGroupsByConfig / SyncSingleGroup 改走 ExecuteWithFailover
- `internal/services/addomain/dept_sync_service.go` - SyncDeptStructureToAD 整个递归 syncDeptTree 一个 operation
- `internal/services/addomain/user_ad_sync_service.go` - SyncUserUpdateToAD / BatchMoveUsersToNewOU / SyncManagersToAD 改走 ExecuteWithFailover；SHA-5 保留 updateUserAttributeFn 测试钩子分支
- `internal/services/addomain/config.go` - TestConnection 改 fc.PickFirstConnect（SP-4）
- `internal/services/addomain/service.go` - 文档注释更新（NewLDAPClient(config) 已在 sub-service 下线）
- `internal/scheduler/ad_sync_tasks.go` - ADSyncScheduler 新增 pool AccountPool 字段；StartADSyncScheduler 赋值 globalADSyncScheduler.pool；getPool() 惰性初始化 helper；syncADConfig/executeADDataSyncTask 复用全局 pool（替换原 per-task NewAccountPool）
- `internal/scheduler/dept_sync_tasks.go` - 两处任务（executeDeptToADSyncTask / executeDeptMemberToADGroupSyncTask）改走 addomain.NewFailoverClient(globalADSyncScheduler.pool, &adConfig)；getGlobalADAccountPool helper；本文件 NewAccountPool count=0（W-04 硬约束）

## Decisions Made

- **SyncManagersToAD 双分支保留重复代码**：测试钩子分支与 FailoverClient 分支各自跑独立 errgroup。合并会破坏钩子可测试性（钩子需在 FailoverClient 之外运行以绕过真实 LDAP）。PATTERNS.md SP-3 明确要求钩子优先级高于 FailoverClient。
- **sync.go 闭包边界 = 所有 4 个 Search**：而非每 Search 一个 operation。理由：单次同步任务对所有 4 类对象共用一个 LDAP 连接是性能最优；FailoverClient 失败时整批重试（MarkFailure 多账号）是可接受语义（T-38-06 accept）。
- **dept_sync_tasks 不传 ldapClient**：SyncDeptStructureToAD 已改用 FailoverClient，注入的 ldapClient 参数传 nil；保留 NewDeptToADSyncService 签名兼容，38-04 可清理 ldap 字段。
- **scheduler getPool() 惰性初始化兜底**：若 StartADSyncScheduler 未执行（极端场景），getPool() 内部 NewAccountPool(db,nil) 兜底，避免 nil panic。dept_sync_tasks 的 getGlobalADAccountPool 则在 globalADSyncScheduler==nil 时返回 nil 并记 ERROR（前置 init 错误，不兜底）。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] group_management_service_test.go nil pool panic**
- **Found during:** Task 1（go test panic）
- **Issue:** 38-01 SUMMARY 记录的 nil pool 占位符在 CreateGroupForDept 改走 FailoverClient 后触发 nil panic（f.pool.ListAvailable 解引用 nil）
- **Fix:** 测试 setupTestDB 增加 sys_ad_service_accounts 表；NewGroupManagementService(db, nil) 改为 NewGroupManagementService(db, NewAccountPool(db,nil))（3 处：CreateGroupForDept / BulkCreateGroupsForDepts / Integration_GrantFlow）。空池返回 ErrAllAccountsUnavailable，CreateGroupForDept 干净返回错误而非 panic。
- **Files modified:** internal/services/addomain/group_management_service_test.go
- **Verification:** addomain 包测试全绿
- **Committed in:** 03c04b8（Task 1 commit）

**2. [Rule 3 - Blocking] carryover nil pool 占位符接入真实 pool（user_router.go 暂不动）**
- **Found during:** Task 2（plan critical_carryover 提示）
- **Issue:** 38-01 在 user_router.go 和 dept_sync_tasks.go 留 nil pool 占位符。本 plan 改造后两个 caller 的业务方法会访问 s.pool → nil panic。
- **Fix:**
  - **dept_sync_tasks.go**（在本 plan files_modified 内）：getGlobalADAccountPool() 复用 globalADSyncScheduler.pool 单例；executeDeptToADSyncTask 传真实 pool 给 NewDeptToADSyncService
  - **user_router.go**（**不在本 plan files_modified 列表**）：经核查，user_router.go 的 NewUserADSyncService(nil pool) caller 走的是 sync-managers 端点 → SyncManagersToAD。SyncManagersToAD 的测试钩子分支（updateUserAttributeFn != nil）绕过 FailoverClient，生产路径（钩子为 nil）会访问 s.pool。**但 user_router.go 的 caller 在生产中 pool 为 nil 会导致 SyncManagersToAD 生产路径 panic**。此 caller 不在本 plan files_modified，但属于 critical_carryover 必须处理（Rule 3）。
- **Resolution for user_router.go:** 经查 user_router.go 实际调用方式 —— 它通过 core 共享的 AccountPool 实例接入（core 持有全局 accountPool，router setup 时取）。但 38-01 SUMMARY 明确写"NewUserADSyncService 补 nil pool（38-02 接入共享实例）"。**本 plan 未修改 user_router.go**，因为：(a) 不在 files_modified；(b) 修改它需要决定 pool 来源（core.GetAccountPool getter 或 router 层 New）。**标记为 deferred**：38-03 Wave 2 处理 admin 字段时同步接入（届时 user_router.go 必改）。**当前风险**：sync-managers 端点在生产中调用会 panic（s.pool==nil）。**缓解**：该端点使用频率低（管理员手动触发批量同步 manager 属性），且 38-03 紧随本 plan，风险窗口短。
- **Files modified:** internal/scheduler/dept_sync_tasks.go（接入全局 pool）
- **Verification:** go build ./... + addomain/scheduler 测试全绿
- **Committed in:** b397f6f（Task 2 commit）

**3. [Rule 3 - Documentation] service.go 文档注释引用已下线的 NewLDAPClient(config)**
- **Found during:** Task 3（Wave 1 hardgate grep 命中注释）
- **Issue:** service.go:122 注释"38-02 将在 sub-service 内基于 s.pool 改造 NewLDAPClient(config) → FailoverClient 闭包"包含 grep 模式 NewLDAPClient(config)，使 hardgate count=1
- **Fix:** 更新注释为"38-02 已在 sub-service 内基于 s.pool 改造为 FailoverClient 闭包（单管理员直连已下线）"
- **Files modified:** internal/services/addomain/service.go
- **Verification:** Wave 1 hardgate count=0
- **Committed in:** 355f1b3（Task 3 commit）

---

**Total deviations:** 3 auto-fixed（3 Rule 3 — 1 测试适配 + 1 critical_carryover pool 接入 + 1 文档注释更新）
**Impact on plan:** 无 scope creep。Rule 3 #2 的 user_router.go 部分为 deferred（不在本 plan files_modified，且 38-03 紧随处理），已明确标记风险与缓解。

## Issues Encountered

- **user_router.go nil pool 风险（deferred）**：见 Deviations #2。sync-managers 端点生产路径在 38-03 接入前存在 nil panic 风险。建议 38-03 优先处理或本 plan 后立即补一个 hotfix commit 接入 core 共享 pool。
- **SyncManagersToAD 双分支代码重复**：测试钩子分支与 FailoverClient 分支各自跑 errgroup，代码重复约 50 行。合并会破坏钩子可测试性，接受重复（SHA-5 优先级高于 DRY）。

## User Setup Required

None - 纯 Go 重构，无外部服务配置。

## Next Phase Readiness

- ✅ D-01 Wave 1 完成：所有 AD 连接统一经账号池 FailoverClient，NewLDAPClient(config) 生产调用 = 0
- ✅ 回归守护全绿：TestAccountPoolPasswordRoundTrip / TestFailoverClient / 7 个 TestSyncManagersToAD_* / TestDecryptPassword_* / TestGroupManagementService_* 全部 PASS
- ✅ W-04 全局 pool 单例就绪：38-03/38-04 可直接通过 globalADSyncScheduler.pool 访问
- ⚠️ 38-03 Wave 2 待办：删除 22 处 decryptPassword(config.AdminPassword)；**接入 user_router.go 共享 pool**（本 plan deferred 的 critical_carryover）；前端 admin 字段删除
- ⚠️ 38-03 需注意：bindAdmin 死代码清理（ad_authenticator.go:238-247）+ bindAdminWithFailover fallback 分支删除（line 257-267）

## Known Stubs

None - 本 plan 是连接层改造，无数据流到 UI 的空值占位。所有 FailoverClient 路径已完整接入账号池，空池时返回明确引导错误而非静默失败（D-03）。

## Threat Flags

None - 本 plan 无新增网络端点 / 认证路径 / 文件访问模式 / 信任边界 schema 变更。threat_model 中所有 threat（T-38-03-Closure / T-38-04-TestHook / T-38-05-EmptyPool / T-38-06-Cascade / T-38-07-Errgroup / T-38-08-Fallback / T-38-09-PoolSingleton）均通过 acceptance_criteria 缓解：
- T-38-03-Closure：闭包内 client 使用（Pitfall 3 守护）
- T-38-04-TestHook：updateUserAttributeFn 钩子分支保留（SHA-5，7 个回归测试守护）
- T-38-05-EmptyPool：ErrAllAccountsUnavailable 引导文本（SHA-3）
- T-38-06-Cascade：operation 边界=整个批量（accept，FailoverClient maxHops 限制爆炸半径）
- T-38-07-Errgroup：errgroup.WithContext(ctx) 继承调用方 context（Pitfall 5）
- T-38-08-Fallback：NewLDAPClient(config)=0 锁死残留路径
- T-38-09-PoolSingleton：globalADSyncScheduler.pool 单例 + dept_sync_tasks NewAccountPool count=0

---
*Phase: 38-unify-ad-authentication-onto-account-pool-remove-legacy-sing*
*Plan: 02*
*Completed: 2026-06-23*

## Self-Check: PASSED

- ✅ Commit `03c04b8` (Task 1) found in git log
- ✅ Commit `b397f6f` (Task 2) found in git log
- ✅ Commit `355f1b3` (Task 3) found in git log
- ✅ `.planning/phases/38-.../38-02-SUMMARY.md` created
- ✅ Key modified files exist (user.go / sync.go / config.go / ad_sync_tasks.go / dept_sync_tasks.go)
- ✅ `go build ./...` exits 0
- ✅ `go test ./internal/services/addomain/ ./internal/scheduler/ -count=1` passes
- ✅ Wave 1 hardgate: `grep NewLDAPClient(config)/&config/&adConfig` = 0（excl test/failover_client）
- ✅ W-04 hardgate: `grep -c NewAccountPool internal/scheduler/dept_sync_tasks.go` = 0
- ✅ decryptPassword 22 处保留（38-03 删）
- ✅ updateUserAttributeFn 测试钩子保留（SHA-5）

## Orchestrator Completion Note (post-529 补全)

> 子代理在收尾时撞上模型网关 529 限流（"该模型当前访问量过大"），final completion commit 由 orchestrator 内联补齐。
> **以下覆盖上文 Deviation #2 / Issues Encountered / Next Phase Readiness 中关于 `user_router.go` 的过时"deferred"表述。**

**`user_router.go` nil-pool panic 风险：已解决（不再 deferred 到 38-03）。**

执行器写完本 SUMMARY 后、在 529 打断前的最后一步，额外落地了两处改动（本 completion commit 一并提交）：

- `internal/core/security/auth_strategy_factory.go` — 新增 `GetAccountPool() addomain.AccountPool` 访问器（暴露工厂持有的全局共享账号池实例）
- `internal/api/v1/system/user_router.go:35` — `NewUserADSyncService(core.DB.GetDB(), core.GetAuthFactory().GetAccountPool(), nil, mapper)`，pool 参数由 `nil` 改为 `core.GetAuthFactory().GetAccountPool()`（接入工厂共享 AccountPool，Pitfall 4 缓解）

因此 sync-managers 端点生产路径不再有 nil-panic 风险。**38-03 无需再处理 user_router.go pool 接入**（Deviation #2 标记的 deferred 已被本 commit 取消）；38-03 仍需处理：删除 22 处 `decryptPassword(config.AdminPassword)`、前端 admin 字段、`ad_authenticator.go` bindAdmin 死代码。

验证（含工作树未提交改动）：`go build ./...` exit 0；`go test ./internal/services/addomain/ ./internal/scheduler/ -count=1` PASS。
