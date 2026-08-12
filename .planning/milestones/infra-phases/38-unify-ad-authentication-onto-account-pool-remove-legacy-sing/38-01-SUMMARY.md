---
phase: 38-unify-ad-authentication-onto-account-pool-remove-legacy-sing
plan: 01
subsystem: infra
tags: [addomain, account-pool, dependency-injection, failover-client, refactor, di-scaffolding]

# Dependency graph
requires:
  - phase: 36-ad-account-pool-failover
    provides: AccountPool 接口 / FailoverClient / sys_ad_service_accounts 表（本 plan 注入目标）
provides:
  - "8 个 addomain 服务 struct 的 pool AccountPool 字段（Sync/User/Group/GroupSync/GroupManagement/DeptToADSync/UserADSync/Config）"
  - "NewADDomainService(db, pool, cipher...) 聚合根构造函数（透传 pool 给所有 sub-service）"
  - "router/scheduler 复用同一 AccountPool 实例（Pitfall 4 缓解）"
  - "编译期 DI 脚手架就绪，38-02 可直接通过 s.pool 访问账号池改造 FailoverClient 闭包"
affects:
  - 38-02-unify-ad-connection-layer (Wave 1 连接层统一)
  - 38-03-remove-legacy-admin-usage (Wave 2 Go 清理 + 前端)
  - 38-04-model-migration-cleanup (Wave 3 model + migration)

# Tech tracking
tech-stack:
  added: [] # 本 plan 纯重构，无新增依赖
  patterns:
    - "AccountPool struct 字段注入（pool AccountPool 紧跟 db 字段后）"
    - "router 单次 NewAccountPool 后由 service+handler 共享（Pitfall 4 反模式规避）"
    - "构造函数透传：NewADDomainService → 8 个 sub-service 全部接收 pool"

key-files:
  created: []
  modified:
    - internal/services/addomain/service.go (NewADDomainService 签名 + 透传)
    - internal/services/addomain/sync.go (SyncService.pool)
    - internal/services/addomain/user.go (UserService.pool)
    - internal/services/addomain/group.go (GroupService.pool)
    - internal/services/addomain/group_sync_service.go (GroupSyncService.pool + 透传 NewSyncService)
    - internal/services/addomain/group_management_service.go (groupManagementService.pool)
    - internal/services/addomain/dept_sync_service.go (DeptToADSyncService.pool)
    - internal/services/addomain/user_ad_sync_service.go (UserADSyncService.pool, 钩子保留)
    - internal/services/addomain/config.go (ConfigService.pool)
    - internal/api/v1/system/ad_domain_router.go (accountPool 单例共享)
    - internal/api/v1/system/user_router.go (NewUserADSyncService 补 nil pool)
    - internal/scheduler/ad_sync_tasks.go (2 处 NewADDomainService 透传 pool)
    - internal/scheduler/dept_sync_tasks.go (NewDeptToADSyncService 补 nil pool)
    - internal/services/addomain/manager_sync_test.go (11 处 caller 适配新签名)
    - internal/services/addomain/group_management_service_test.go (5 处 caller 适配新签名)

key-decisions:
  - "本 plan 仅做编译期 DI 脚手架，业务逻辑（NewLDAPClient(config) / decryptPassword）暂未改动，留待 38-02 改造"
  - "user_router.go / dept_sync_tasks.go 暂传 nil pool（pool 字段在 38-02 才被业务方法使用）；38-02 改造时改为复用 core 共享的 AccountPool 实例（Pitfall 4）"
  - "scheduler 按 per-task NewAccountPool 现有模式（保持代码风格一致性，不引入 scheduler 全局 pool 字段）"
  - "SHA-5 严格遵守：user_ad_sync_service.go 的 updateUserAttributeFn 测试钩子字段保留不变（7 个 TestSyncManagersToAD_* 回归测试依赖）"

patterns-established:
  - "AccountPool 注入模式：struct 字段 + 构造函数参数，紧跟 db 字段后；pool 字段加注释标明 Phase 38 Wave 1 DI 脚手架用途"
  - "聚合根透传：NewADDomainService 接收 pool 后透传给所有需走账号池的 sub-service，不需 pool 的（OU/OUGroupMapping/Log/Computer）保持原样"

requirements-completed: [D-01]

# Metrics
duration: 18min
completed: 2026-06-23
---

# Phase 38 Plan 01: Wave 1 前置 DI 脚手架 Summary

**为 8 个 addomain 服务 struct 注入 pool AccountPool 字段并统一改构造函数签名，消除 38-02 改造的编译阻塞（21+ 处 NewLDAPClient(config) caller 无法直接 NewFailoverClient(s.pool, config)）**

## Performance

- **Duration:** ~18 min
- **Started:** 2026-06-23T10:25:00Z
- **Completed:** 2026-06-23T10:43:00Z
- **Tasks:** 2
- **Files modified:** 15（8 服务 + service.go + router + scheduler + 2 测试 + user_router/dept_sync_tasks）

## Accomplishments
- Sync/User/Group/GroupSync/GroupManagement/DeptToADSync/UserADSync/Config 8 个服务 struct 全部含 pool AccountPool 字段，构造函数全部接收 pool 参数
- NewADDomainService 聚合根签名变更为 `(db, pool, cipher...)`，透传 pool 到 6 个 sub-service（OU/OUGroupMapping/Log/Computer 不需 pool 保持原样）
- ad_domain_router.go 中 service 与 accountHandler 共享同一 accountPool 实例（Pitfall 4 缓解：不重复 New，避免熔断后账号仍被选中）
- 全项目 `go build ./...` 通过；addomain 包测试全绿（7 个 TestSyncManagersToAD_* + TestAccountPoolPasswordRoundTrip + TestFailoverClient + TestDecryptPassword_* + 3 个 TestResolveLeader*）
- user_ad_sync_service.go 测试钩子 updateUserAttributeFn 字段保留不变（SHA-5，4 处引用全部保留）
- 业务逻辑零改动：22 处 NewLDAPClient(config) 与 17 处 decryptPassword 调用数量不变（留待 38-02/38-03）

## Task Commits

Each task was committed atomically:

1. **Task 1: 为 8 个服务 struct 增加 pool AccountPool 字段并改构造函数签名** - `abdd3e0` (refactor)
2. **Task 2: 更新 NewADDomainService 签名 + 所有 caller 透传同一 AccountPool 实例** - `21ebb33` (refactor)

_Note: Task 1 同时更新 service.go 的 NewADDomainService 内部透传（addomain 包必须整体编译，service.go 在包内调用 6 个 sub-service 构造函数）；Task 2 处理包外 caller（router/scheduler/test）。两个 commit 紧邻，中间无中间态（main 始终可达到全项目 build 通过的状态）。_

## Files Created/Modified
- `internal/services/addomain/service.go` - NewADDomainService 签名 `(db, pool, cipher...)` + 透传 pool 到 6 个 sub-service
- `internal/services/addomain/sync.go` - SyncService.pool 字段 + NewSyncService(db, pool)
- `internal/services/addomain/user.go` - UserService.pool 字段 + NewUserService(db, pool)
- `internal/services/addomain/group.go` - GroupService.pool 字段 + NewGroupService(db, pool)
- `internal/services/addomain/group_sync_service.go` - GroupSyncService.pool 字段 + 透传给内嵌 NewSyncService(db, pool)
- `internal/services/addomain/group_management_service.go` - groupManagementService.pool 字段 + NewGroupManagementService(db, pool)
- `internal/services/addomain/dept_sync_service.go` - DeptToADSyncService.pool 字段 + NewDeptToADSyncService(db, pool, ldap, mapper)
- `internal/services/addomain/user_ad_sync_service.go` - UserADSyncService.pool 字段 + NewUserADSyncService(db, pool, ldap, mapper)（updateUserAttributeFn 钩子保留）
- `internal/services/addomain/config.go` - ConfigService.pool 字段 + NewConfigService(db, pool)
- `internal/api/v1/system/ad_domain_router.go` - accountPool 单次创建后由 service+handler 共享（Pitfall 4）
- `internal/api/v1/system/user_router.go` - NewUserADSyncService 补 nil pool（38-02 接入共享实例）
- `internal/scheduler/ad_sync_tasks.go` - 2 处 NewADDomainService 透传 pool
- `internal/scheduler/dept_sync_tasks.go` - NewDeptToADSyncService 补 nil pool（38-02 接入共享实例）
- `internal/services/addomain/manager_sync_test.go` - 11 处 NewUserADSyncService(db, nil, nil, nil) 适配新签名（钩子逻辑不变）
- `internal/services/addomain/group_management_service_test.go` - 5 处 NewGroupManagementService(db, nil) 适配新签名

## Decisions Made
- **service.go NewADDomainService 内部透传与 Task 1 同 commit**：addomain 包整体编译，service.go 在包内调用 6 个 sub-service 构造函数，必须同步更新否则包内 build 失败。Task 1/Task 2 边界以"包内（addomain 包）/包外（router/scheduler/test）"划分，而非 plan 原本的"8 服务/service.go+caller"划分，避免半改状态。
- **scheduler 按 per-task NewAccountPool 现有模式**：scheduler 现有代码即 per-task 创建 pool（ad_sync_tasks.go:105 已是此模式），本 plan 保持一致不引入 scheduler 全局 pool 字段。38-02 若需 Pitfall 4 严格共享再评估。
- **user_router.go / dept_sync_tasks.go 暂传 nil pool**：这两个 caller 不在 plan files_modified 列表但必须适配新签名（否则编译失败）。pool 字段在 38-02 才被业务方法使用，nil 不影响当前行为；38-02 接入共享实例。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] 测试文件 caller 同步适配新签名**
- **Found during:** Task 2（addomain 包测试编译失败）
- **Issue:** Task 1 签名变更后，manager_sync_test.go（11 处 NewUserADSyncService）与 group_management_service_test.go（5 处 NewGroupManagementService）使用旧签名导致 addomain 包测试 build failed，无法跑回归守护
- **Fix:** 测试调用点补 nil 占位参数（NewUserADSyncService(db, nil, nil, nil) / NewGroupManagementService(db, nil)）；测试行为不变（仅签名适配，钩子逻辑保留）
- **Files modified:** internal/services/addomain/manager_sync_test.go, internal/services/addomain/group_management_service_test.go
- **Verification:** 7 个 TestSyncManagersToAD_* + TestAccountPoolPasswordRoundTrip + TestFailoverClient + TestDecryptPassword_* + 3 个 TestResolveLeader* 全绿
- **Committed in:** 21ebb33（Task 2 commit）

**2. [Rule 3 - Blocking] user_router.go 与 dept_sync_tasks.go 补 nil pool**
- **Found during:** Task 2（全项目 build 失败暴露 plan files_modified 遗漏的 caller）
- **Issue:** 这两个 caller 不在 plan files_modified 列表，但 NewUserADSyncService / NewDeptToADSyncService 签名变更后必须适配，否则全项目 build 失败
- **Fix:** 暂传 nil pool（pool 字段在 38-02 才被业务方法使用，nil 不影响当前行为），加注释标明 38-02 改造时接入共享实例
- **Files modified:** internal/api/v1/system/user_router.go, internal/scheduler/dept_sync_tasks.go
- **Verification:** go build ./... 通过；addomain 测试通过
- **Committed in:** 21ebb33（Task 2 commit）

---

**Total deviations:** 2 auto-fixed（2 Rule 3 blocking — 测试 + 额外 caller 适配新签名）
**Impact on plan:** 两个 auto-fix 都是 Task 1 签名变更的必然结果（caller 必须适配）。无 scope creep，行为零变化（仅 nil 占位 + 注释）。

## Issues Encountered
- **plan files_modified 列表遗漏 caller**：plan 只列了 ad_domain_router.go 和 ad_sync_tasks.go，但 user_router.go（NewUserADSyncService）与 dept_sync_tasks.go（NewDeptToADSyncService）也是签名变更的 caller，必须适配。已按 Rule 3 auto-fix，并在 key-decisions 与 key-files 中记录。
- **NewLDAPClient(config) 实际基线 = 22 而非 plan 估算的 21**：plan acceptance_criteria 写 `= 21`，但 RESEARCH 估算与实测有 1 处差异（grep 含 `NewLDAPClient(&adConfig)` 变体）。关键不变量是"数量不变"（baseline 22 → 改造后 22），符合 plan 意图"本 plan 不改 caller，数量不变"。38-02 改造后归零的硬指标不受影响。

## User Setup Required
None - 纯 Go 重构，无外部服务配置。

## Next Phase Readiness
- ✅ 编译期 DI 脚手架就绪：38-02 可直接通过 `s.pool` 访问账号池改造 FailoverClient 闭包，无需再动 struct/constructor
- ✅ 回归守护全绿：TestAccountPoolPasswordRoundTrip / TestFailoverClient / 7 个 TestSyncManagersToAD_* / TestDecryptPassword_InvalidCiphertext 保持绿色
- ⚠️ 38-02 改造时需评估：user_router.go / dept_sync_tasks.go 的 nil pool 是否需替换为共享实例（Pitfall 4 严格共享）；scheduler per-task pool 模式是否需收敛为全局单例
- ⚠️ 38-02 改造时需注意：22 处 NewLDAPClient(config) 调用全部改走 FailoverClient.ExecuteWithFailover；user_ad_sync_service.go 的 updateUserAttributeFn 钩子分支保留（SHA-5）

## Known Stubs
None - 本 plan 是 DI 脚手架，pool 字段在业务方法中暂未被使用（Task 1 完成时 go vet 可能报 field unused 警告，Task 2 后 caller 已全部适配但业务方法尚未使用 pool，38-02 改造后清零）。这是 plan 明确允许的"半成品"状态（plan acceptance_criteria：`go vet ... | grep -c "unused"` 可大于 0，Task 2 后清零）。pool 字段并非数据流到 UI 的空值占位，而是 38-02 的编译期前置依赖。

## Threat Flags
None - 本 plan 无新增网络端点 / 认证路径 / 文件访问模式 / 信任边界 schema 变更。T-38-01-DI（AccountPool 实例复用）已通过 acceptance criteria `addomainServices.NewAccountPool(core.GetDB()` count=1 缓解。

---
*Phase: 38-unify-ad-authentication-onto-account-pool-remove-legacy-sing*
*Plan: 01*
*Completed: 2026-06-23*

## Self-Check: PASSED

- ✅ Commit `abdd3e0` (Task 1) found in git log
- ✅ Commit `21ebb33` (Task 2) found in git log
- ✅ `.planning/phases/38-.../38-01-SUMMARY.md` created
- ✅ Key modified files exist (service.go / sync.go / ad_domain_router.go / ad_sync_tasks.go)
- ✅ `go build ./...` exits 0
- ✅ `go test ./internal/services/addomain/ -count=1` passes (regression guards all green)
