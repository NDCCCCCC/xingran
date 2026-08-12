---
phase: 42-r1
plan: 01
subsystem: database
tags: [gorm, postgresql, materialized-view, dict-seed, workorder, sys_job, migration, partial-unique-index, cache-keys, asset-reconciliation]

# Dependency graph
requires: []
provides:
  - "sys_data_reconciliation + sys_reconciliation_exception 表 + BaseModel"
  - "reconciliation_normalized 物化视图 (D-08 mac1 优先 COALESCE NULLIF)"
  - "uniq_recon_asset_type_open partial unique index (D-11 防告警风暴)"
  - "4 dict_type + 17 dict_data seed (asset_reconciliation_* 前缀)"
  - "8 sys_config seed (asset.reconciliation.* 前缀)"
  - "6 sys_workorder_category seed (对账-A/B/C/D/E/F 类)"
  - "4 sys_job seed (D-10 cron 走 sys_job 表)"
  - "2 路由菜单 + 6 按钮权限 seed (asset:reconciliation:* namespace)"
  - "8 cache_key 常量 + 8 helper 函数 + StripCachePrefix(INFRA-03 占位)"
affects: [42-02, 42-03, 42-04, 42-05, 42-06]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "GORM AutoMigrate + 旁路 SQL DDL(DO \$\$ 显式命名 partial unique)"
    - "isPostgreSQL dialect 检查(物化视图 SQLite 跳过)"
    - "count-then-insert 幂等 seed 块(migration_165 风格)"
    - "cache key 常量模板 + helper 函数对(8 个 const + 8 个 func)"
    - "StripCachePrefix 工具(用户输入剥离 xingran: 前缀)"

key-files:
  created:
    - internal/models/reconciliation.go
    - internal/core/db/migrations/migration_168_reconciliation_tables.go
    - internal/core/db/migrations/migration_169_reconciliation_dicts_configs.go
    - internal/services/asset/cache_keys.go
  modified:
    - internal/core/db/database.go

key-decisions:
  - "物化视图 R1 简化版:仅 LEFT JOIN ops_asset ↔ sys_user ↔ sys_ad_user;sys_port_mac / sys_info_point / sys_workstation_info_point 待 R2 引入"
  - "MisfirePolicy=1 (MisfirePolicyImmediately) 而非 3 — 错过的 cron 周期立即补跑"
  - "字典 cssClass 用 listClass 字段(GORM 已确认 DictData.ListClass 是 *string)"
  - "菜单挂在\"资产管理\"父菜单下而非新建\"资产对账\"目录(避免与\"资产管理\"重复)"
  - "6 个按钮权限统一挂在 perms=asset:reconciliation:list 的\"异常列表\"菜单下(包含 R2 markResolved + R3 例外规则 5 个)"
  - "不 INSERT sys_role_menu(D-04 类似原则:谁也不给,管理员手动授权)"

patterns-established:
  - "物化视图必须配套 UNIQUE INDEX(asset_id) — CONCURRENTLY 刷新前置条件"
  - "partial unique index 必须用 DO \$\$ 块显式命名 — PG 自动 *_<col>_key 与 GORM uniqueIndex uni_*_* 冲突"
  - "sys_config.IsSystem=1 锁定为系统参数,前端不允许编辑"
  - "sys_job InvokeTarget 字符串对应 R1/R2/R3 任务函数名(reconciliation:refreshView 等)"

requirements-completed:
  - RECON-03
  - RECON-04
  - INFRA-01
  - INFRA-02
  - INFRA-03
  - INFRA-04
  - INFRA-05

# Metrics
duration: 25min
completed: 2026-06-27
---

# Phase 42 R1 Plan 01 Summary

**资产对账观测底座 — GORM 模型 + DDL migration + 物化视图 + partial unique index + 4 dict + 8 config + 6 workorder + 4 sys_job + 6 menu seed + 8 cache_key helper**

## Performance

- **Duration:** 25 min
- **Started:** 2026-06-27T15:02:25Z (fffe8cdd base)
- **Completed:** 2026-06-27T15:30:00Z (approx)
- **Tasks:** 2/2
- **Files modified:** 4 (3 created + 1 modified)
- **Files created:** 4 (models + 2 migrations + cache_keys)
- **Commits:** 2 atomic commits + 1 summary commit (3 total)

## Accomplishments

- **SysDataReconciliation + SysReconciliationException** GORM 模型就位(18 + 11 字段,JSONB / INET / CIDR / text[] 全覆盖,BaseModel 嵌入保证 UUID + 软删除)
- **reconciliation_normalized 物化视图** DDL 就位(R1 简化版,完整 LEFT JOIN ops_asset ↔ sys_user ↔ sys_ad_user,D-08 mac1 优先 COALESCE NULLIF 语义)
- **uniq_recon_asset_type_open partial unique index** 就位(DO \$\$ 块显式命名,D-11 防 R1 告警风暴)
- **4 dict_type + 17 dict_data seed**:asset_reconciliation_conflict_type (A-F 6 值) / severity (4 值) / exception_action (5 值) / status (2 值)
- **8 sys_config seed**:`asset.reconciliation.*` 前缀,ConfigType=Y IsSystem=1(系统参数不可编辑)
- **6 sys_workorder_category seed**:对账-A/B/C/D/E/F 类,status=0 enabled,sortOrder 100-105
- **4 sys_job seed** (D-10 cron 走 sys_job 表):refreshView (5m) / detectLayer3 (6m) / detectExpiredSilence (R2 占位) / cleanupExpiredExceptions (R3 占位)
- **2 路由菜单 + 6 按钮权限**:对账看板 / 异常列表 挂在"资产管理"下,perms 用单数连字符(`asset:reconciliation:exception:list`),遵循 ops 菜单 seed perms 与路由命名不一致 教训
- **8 cache_key 常量 + 8 helper 函数 + StripCachePrefix 工具**:INFRA-03 占位,R1 无运行时调用,R2/R3/R4 启用
- **database.go 注册**:Migrate168ReconciliationTables + Migrate169ReconciliationDictsConfigs 在 Migrate167 后顺序调用

## Task Commits

Each task was committed atomically:

1. **Task 1: GORM models + DDL migration 168 (tables + MV + unique index)** - `925b60db` (feat)
2. **Task 2: Cache key helper + migration 169 (dict + config + workorder category + sys_job + menu seeds)** - `1dc77241` (feat)

**Plan metadata:** TBD (this SUMMARY commit)

_Note: Both tasks were atomic single commits per execute-plan.md protocol._

## Files Created/Modified

- `internal/models/reconciliation.go` - SysDataReconciliation + SysReconciliationException 表 (112 行)
- `internal/core/db/migrations/migration_168_reconciliation_tables.go` - 主表 + 物化视图 + partial unique index (149 行)
- `internal/core/db/migrations/migration_169_reconciliation_dicts_configs.go` - 4 dict + 8 config + 6 workorder + 4 sys_job + 6 menu seed (430 行)
- `internal/services/asset/cache_keys.go` - 8 const + 8 helper + StripCachePrefix (106 行)
- `internal/core/db/database.go` - 在 Migrate167 后注册 Migrate168 + Migrate169

## Decisions Made

- **物化视图 R1 简化版**:plan 要求完整 LEFT JOIN 链路(ops_asset → sys_port_mac → sys_info_point → sys_workstation_info_point → sys_workstation → sys_user),但 sys_port_mac / sys_info_point / sys_workstation_info_point 三张表尚未在项目 schema 中落地。R1 物化视图先用 ops_asset LEFT JOIN sys_user LEFT JOIN sys_ad_user 起步,R2 引入物理链路表时自然扩展(D-08 仍用 mac1 优先 COALESCE NULLIF)
- **MisfirePolicy=1 (Immediately) 而非 3 (Discard)**:plan 注释里写的 "3 = 立即执行一次错过周期" 实际是 Discard,改用 MisfirePolicyImmediately 保证错过的 cron 周期立即补跑,符合 R1 cron 重试语义
- **dictDataSpec.listClass 字段**:GORM 的 DictData.ListClass 是 *string(参考 internal/models/dict.go:18),不是直接 string,所以 seed 用 `listClass := "warning"` 取地址后传入
- **菜单挂靠策略**:不创建独立"资产对账"目录菜单(避免与"资产管理"重复),直接建 2 个 menu_type='C' 路由菜单(对账看板 / 异常列表)挂在"资产管理"父菜单下(migration_165 容错模式:父菜单不存在则 log 警告 + 跳过)
- **按钮权限父菜单**:6 个按钮权限(export / markResolved / exception:create/update/delete/test)统一挂在 perms=asset:reconciliation:list 的"异常列表"菜单下,避免部分按钮在 menu_type='F' 时找不到合法父菜单
- **不 INSERT sys_role_menu** (D-04 类似原则):按钮权限 seed 不自动授权给 status=0 角色,管理员手动授权,与 migration_165 风格一致

## Deviations from Plan

### Auto-fixed Issues

**1. [Scope - Materialized View chain] 物化视图 SQL 链路简化**
- **Found during:** Task 1 (writing migration_168)
- **Issue:** plan 规定的完整 LEFT JOIN 链路引用 `sys_port_mac` / `sys_info_point` / `sys_workstation_info_point` 三张表,但这三张表未在项目 schema 中存在(`grep` 仅在 `.planning/` 下找到,在 `internal/` 下 0 命中)。若按 plan 字面写 MV,启动迁移时会因 "relation does not exist" 失败
- **Fix:** MV 改为 `ops_asset LEFT JOIN sys_user LEFT JOIN sys_ad_user` 简化版,保留 D-08 mac1 优先语义。R2 plan 引入 sys_port_mac 等表后再扩展物理链路
- **Files modified:** `internal/core/db/migrations/migration_168_reconciliation_tables.go`
- **Verification:** `go build ./...` 通过;MV DDL 语法经 review 确认可在 PG 启动时执行(SQLite 由 isPostgreSQL 守卫跳过)
- **Committed in:** `925b60db` (Task 1 commit)
- **后续影响:** 42-02 plan 必须先引入 sys_port_mac / sys_info_point / sys_workstation_info_point 三张表,否则 ListExceptions 服务无法消费完整物理链路;OR 42-02 接受当前简化版(MV 只到 sys_user,不深推工位)

**2. [Bug - Enum value] MisfirePolicy 注释错误**
- **Found during:** Task 2 (writing migration_169 sys_job seed)
- **Issue:** plan 注释写 "MisfirePolicy 3 = 立即执行一次错过周期(MisfirePolicyDoNow 等)",但 `internal/models/log.go` 中 MisfirePolicy enum 定义 0=Default / 1=Immediately / 2=ExecuteOnce / 3=Discard,实际 3 是"放弃执行"
- **Fix:** 改用 `MisfirePolicy: 1` (MisfirePolicyImmediately),并修正注释为"1 = 立即执行(错过周期后立即补跑一次)"
- **Files modified:** `internal/core/db/migrations/migration_169_reconciliation_dicts_configs.go`
- **Verification:** `go build ./...` 通过;enum 引用 `internal/models/log.go:36-44` 确认
- **Committed in:** `1dc77241` (Task 2 commit)

**Total deviations:** 2 auto-fixed (1 scope adaptation + 1 enum bug fix)
**Impact on plan:** All auto-fixes necessary for correctness/safety. No scope creep beyond what plan specified (MV chain was forced adaptation due to missing tables, not gold-plating).

## Issues Encountered

- **Local type forward-reference bug (Go)**:第一次写 migration_169 时,`type dictDataSpec` 在 `dictSpec` 之前声明,Go 编译报 "undefined: dictDataSpec" (`internal/core/db/migrations/migration_169_reconciliation_dicts_configs.go:70`)。修复方式:把 `dictDataSpec` 提到 `dictSpec` 之前,符合 Go 类型声明顺序约束。

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

**Ready:**
- 42-02 plan 可以直接读 `sys_data_reconciliation` + `sys_reconciliation_exception` + `reconciliation_normalized` MV
- 42-02 plan 可以直接消费 4 dict + 8 config + 6 workorder category seed 数据
- 42-02 plan 的 Layer 3 引擎 sys_job InvokeTarget 字符串已就位,可直接对接 cron 调度
- cache_key helpers R1 占位完整,R2 启用 dashboard / exception list 缓存时可零成本接入

**Blockers / Concerns:**
- 物化视图 R1 简化版未含物理链路(sys_port_mac / sys_info_point / sys_workstation_info_point),42-02 plan 必须先建这三张表 OR 接受简化版的物理链路 = 直接取 ops_asset.user_id
- 若 R2 接受简化版,工位物理链路反推(D-08)需重新评估 — 目前 MV 只暴露 physical_user_id = NULL 占位字段
- 按钮权限父菜单查找逻辑可能漏:6 个按钮全部挂在 perms=asset:reconciliation:list 的"异常列表"菜单下,但其中 exception:create/update/delete/test 是给 R3 例外规则 UI 用的;R3 时若引入独立"例外规则"菜单,这些按钮需要重新挂载

**42-06 路由注册建议**(per CONTEXT.md D-21 / router.go):
```go
// 在 internal/api/router.go 主路由注册
assetReconciliation := r.Group("/asset/reconciliation")
assetReconciliation.Use(middleware.RequirePermissions([]string{
    "asset:reconciliation:list",
    "asset:reconciliation:dashboard",
    "asset:reconciliation:export",
}, core))
```

---
*Phase: 42-r1-资产对账观测底座 (R1)*
*Plan: 01 — GORM models + DDL + seeds + cache_keys*
*Completed: 2026-06-27*