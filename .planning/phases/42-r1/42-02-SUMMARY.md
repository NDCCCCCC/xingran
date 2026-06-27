---
phase: 42-r1
plan: 02
subsystem: asset-reconciliation
tags: [gorm, postgresql, materialized-view, layer3-detection, cron, snapshot, service-layer, handler-router, operlog, unique-violation]

# Dependency graph
requires:
  - 42-01
provides:
  - "ReconciliationService interface (ListExceptions + GetByID) with JOIN chain"
  - "ReconciliationExceptionService interface (List + GetByID) R1 skeleton"
  - "ReconciliationSnapshot service (RefreshView + LastRefreshAt) + 4 cron InvokeTarget global functions"
  - "ReconciliationDetection engine (Layer 3 Type A-F classification + confidence + severity + D-11 unique violation catch)"
  - "5 unit tests (ClassifyType, ComputeConfidence, ComputeSeverity, DetectLayer3 Type A, DetectLayer3 duplicate skip)"
  - "4 read POST endpoints (handler + router) with ModuleReconciliation constant"
affects: [42-03, 42-04, 42-05, 42-06]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Handler-Service pattern with interface + private impl + WithCore DI (from location_alias_service / asset_handler)"
    - "JOIN chain: ops_asset LEFT JOIN sys_user (responsible) LEFT JOIN reconciliation_normalized (physical/ad)"
    - "Layer 3 detect: 5-factor signals → 6 type mapping (A=healthy, B/C/D/E/F = 5 abnormal classes)"
    - "confidence formula: physical*0.5 + declared*0.3 + ad*0.2"
    - "severity mapping: B/C=high, D/F=medium, E=low"
    - "D-11 unique violation catch: pgconn 23505 / SQLite UNIQUE fallback via isReconciliationDuplicate"
    - "D-09 Type A skip: 物理链路/declared/AD 三路一致不入 sys_data_reconciliation"
    - "D-02 RefreshView 失败仅 log + 返回 error(调用方写 job_log)"
    - "D-18 R1 无 mark-resolved handler(读端点全跳 operlog.Record)"
    - "4 cron InvokeTarget 全局函数(reconciliation:refreshView / detectLayer3 / detectExpiredSilence / cleanupExpiredExceptions)"

key-files:
  created:
    - internal/services/asset/reconciliation_service.go
    - internal/services/asset/reconciliation_exception.go
    - internal/services/asset/reconciliation_snapshot.go
    - internal/services/asset/reconciliation_detection.go
    - internal/services/asset/reconciliation_test.go
    - internal/api/v1/asset/reconciliation_handler.go
    - internal/api/v1/asset/reconciliation_router.go
    - internal/api/v1/asset/reconciliation_exception_handler.go
    - internal/api/v1/asset/reconciliation_exception_router.go
  modified: []

key-decisions:
  - "ExceptionListItem 用 AssetIPDisplay 字段名(避开 SysDataReconciliation.AssetIP 字段冲突)"
  - "responsible_username 通过 sys_user JOIN(非 reconciliation_normalized,因 MV 暴露的是 asset_username 与 responsible_username 同源)"
  - "DetectLayer3 跳过的 Type A 计入 skipped(而非 inserted),便于 D-09 验证"
  - "RefreshView 失败:log.Warn + return err(给 cron 写 job_log)"
  - "LastRefreshAt 解析 RFC3339 / SQL datetime 两种格式,失败返回 nil"
  - "4 个 cron InvokeTarget 全局函数是 stub(无 core 依赖避免循环),42-04/42-06 接入 service binding"
  - "测试用 sqlite in-memory(共享 cache + view 模拟 MV),unique index 不带 WHERE 子句(SQLite 限制),D-11 通过 catch UNIQUE constraint failed 字符串兜底"

patterns-established:
  - "Layer 3 检测引擎 6 类语义(Type A 健康 / B 物理无责任人 / C 物理不匹配 / D 仅责任人 / E 无用户 / F AD 不一致)"
  - "operlog module 常量 ModuleReconciliation = \"资产对账\" 仅 R1 1 个(R2 扩展)"
  - "读端点全跳 operlog.Record(对标 operations/asset_handler.go Statistics 模式)"

requirements-completed:
  - RECON-01
  - RECON-02
  - RECON-05
  - RECON-06
  - RECON-07
  - INFRA-04

# Metrics
duration: 35min
completed: 2026-06-27
---

# Phase 42 R1 Plan 02 Summary

**资产对账观测底座 R1 — ReconciliationService 核心 (Layer 3 检测引擎 + 物化视图刷新 + 4 端点)**

## Performance

- **Duration:** 35 min (parallel agent execution)
- **Started:** 2026-06-27T16:00:00Z (approx, after 42-01 merge)
- **Completed:** 2026-06-27T16:35:00Z (approx)
- **Tasks:** 4/4
- **Files modified:** 9 (9 created, 0 modified)
- **Lines added:** ~1,400 (services + handlers + tests)
- **Commits:** 4 atomic commits + 1 summary commit (5 total)

## Accomplishments

- **ReconciliationService** 接口就位 (ListExceptions + GetByID),含完整 JOIN 链:
  - `ops_asset` LEFT JOIN 取 `asset_code / asset_ip`
  - `sys_user` LEFT JOIN 取 `responsible_username`(R1 物理链路未填时,与 asset_username 同源)
  - `reconciliation_normalized` MV LEFT JOIN 取 `physical_username / ad_username`(R1 物理字段恒为 NULL,AD 字段由 MV 维护)
  - `reconAllowedSortFields` 白名单:`detectedAt / conflictType / severity / confidenceScore` 4 字段
- **ReconciliationExceptionService** R1 skeleton (List + GetByID),R3 接入 Create/Update/Delete/Enable/Disable
- **ReconciliationSnapshot** 物化视图刷新服务:
  - `RefreshView` 走 `REFRESH MATERIALIZED VIEW CONCURRENTLY reconciliation_normalized`
  - 失败 log.WithError + log.Warn + 返回 error(调用方决定 handler 500 / cron 写 job_log)
  - `LastRefreshAt` 从 `sys_config.config_key='asset.reconciliation.view.last_refresh_at'` 读 ISO8601
  - 4 个 cron InvokeTarget 全局函数就位:`reconciliation:refreshView / detectLayer3 / detectExpiredSilence / cleanupExpiredExceptions`
- **ReconciliationDetection** Layer 3 引擎:
  - `ClassifySignals` 5 因子信号(`HasPhysical / HasDeclared / HasAD / PhysicalMatchDeclared / PhysicalMatchAD`)
  - `ClassifyType` A-F 分类(规则见 D-09)
  - `ComputeConfidence` `physical*0.5 + declared*0.3 + ad*0.2`,截断 2 位小数
  - `ComputeSeverity` `B/C=high, D/F=medium, E=low, A=low`
  - `DetectLayer3` 遍历 MV 写入 sys_data_reconciliation,Type A 跳过(D-09),unique violation catch 静默跳过(D-11)
  - `isReconciliationDuplicate` 识别 `pgconn 23505` + SQLite `UNIQUE constraint failed` 字符串兜底
- **5 单元测试**(全部 PASS):
  - `TestClassifyType` — 7 子测试覆盖 A/B/C/D/E/F + F 健康不一致子例
  - `TestComputeConfidence` — 5 子测试覆盖全 0 到 1.0 各种组合
  - `TestComputeSeverity` — 7 子测试覆盖 A/B/C/D/E/F/X
  - `TestDetectLayer3_TypeA_NotInserted` — D-09 验证(健康资产不入主表)
  - `TestDetectLayer3_DuplicateViolation_Skipped` — D-11 验证(预插入 + 重跑不抛错,总数 1)
- **4 读端点** (handler + router 4 文件):
  - `POST /asset/reconciliation/exception/list` → ListExceptions
  - `POST /asset/reconciliation/exception/:id` → GetExceptionByID
  - `POST /asset/reconciliation/exception-rule/list` → ListRules(R1 skeleton)
  - `POST /asset/reconciliation/exception-rule/:id` → GetRuleByID(R1 skeleton)
  - 全部读端点不调 operlog.Record
  - `ModuleReconciliation = "资产对账"` 常量在 reconciliation_handler.go 顶部(D-16)

## Task Commits

Each task was committed atomically:

1. **Task 1: ReconciliationService + ExceptionService (read paths only)** - `dafb8aed` (feat)
2. **Task 2: ReconciliationSnapshot (MV refresh) + 4 cron global functions** - `735aa93a` (feat)
3. **Task 3: ReconciliationDetection (Layer 3 Type A-F + confidence + unique violation catch) + 5 tests** - `ce617b0a` (feat)
4. **Task 4: Handler + Router (R1 read endpoints + exception skeleton)** - `14427efc` (feat)

**Plan metadata:** TBD (this SUMMARY commit)

_Note: All tasks were atomic single commits per execute-plan.md protocol._

## Files Created/Modified

- `internal/services/asset/reconciliation_service.go` - ReconciliationService 接口 + impl(JOIN chain)
- `internal/services/asset/reconciliation_exception.go` - ReconciliationExceptionService skeleton
- `internal/services/asset/reconciliation_snapshot.go` - Snapshot service + 4 cron global functions
- `internal/services/asset/reconciliation_detection.go` - Layer 3 engine
- `internal/services/asset/reconciliation_test.go` - 5 unit tests
- `internal/api/v1/asset/reconciliation_handler.go` - 异常 handler(2 endpoint)
- `internal/api/v1/asset/reconciliation_router.go` - 异常路由注册
- `internal/api/v1/asset/reconciliation_exception_handler.go` - 例外规则 handler(R1 skeleton)
- `internal/api/v1/asset/reconciliation_exception_router.go` - 例外规则路由注册

## Decisions Made

- **AssetIPDisplay 字段名**:避开 SysDataReconciliation.AssetIP 字段名冲突,GORM tag `column:asset_ip` 路由到新字段
- **responsible_username JOIN 路径**:通过 `sys_user LEFT JOIN ops_asset.user_id` 直接取(因 MV 暴露的 asset_username 与 responsible_username 同源,R1 物理链路未填,简化 JOIN 链)
- **DetectLayer3 跳过 Type A 计入 skipped**:与 inserted 分开统计,便于 D-09 验证和监控数据健康率
- **RefreshView 失败处理**:返回 error 而非 nil(让调用方决定 500/job_log),但同时 log.Warn(D-02 要求"仅 log"是给 cron 调用方的指示,本方法返回 err 是契约透明)
- **LastRefreshAt 多格式兼容**:尝试 RFC3339 + `"2006-01-02 15:04:05"` 两种格式,失败返回 nil(避免前端显示错)
- **cron InvokeTarget stub 设计**:4 个全局函数不直接 import core 避免循环依赖,42-04/42-06 由 service 注入层在 cron 调度时 binding service
- **测试用 SQLite 共享 cache + view 模拟 MV**:SQLite 不支持物化视图,用 `CREATE VIEW` 模拟,验证 SQL 兼容性
- **D-11 partial unique index SQLite 妥协**:SQLite unique index 不支持 WHERE 子句,降级为非 partial unique,测试通过 DetectLayer3 不抛错 + 计数 0 验证 D-11 语义

## Deviations from Plan

### Auto-fixed Issues

**1. [Scope - JOIN chain simplification] responsible_username 取自 sys_user 而非 MV**
- **Found during:** Task 1 (writing reconciliation_service.go)
- **Issue:** plan 规定 responsible_username 来自 reconciliation_normalized MV,但 MV (migration_168) 实际只暴露 `asset_username`(即 `sys_user.username` JOIN via `ops_asset.user_id`),未显式提供 `responsible_username` 列(因 R1 物理链路与责任人同源,无需冗余)
- **Fix:** service 层直接通过 `LEFT JOIN sys_user ru ON ru.id = a.user_id` 取 `responsible_username`,语义与 plan 一致
- **Files modified:** `internal/services/asset/reconciliation_service.go`
- **Verification:** `go build ./internal/services/asset/...` 通过;5 字段 SELECT 全部命中 GORM 标签
- **Committed in:** `dafb8aed` (Task 1 commit)
- **后续影响:** R2 引入独立 `responsible_user_id` 字段时,JOIN 改用该字段即可,无需改 model

**2. [Test infrastructure - SQLite view 写入问题] Test 4 写入 view 失败**
- **Found during:** Task 3 (writing reconciliation_test.go)
- **Issue:** SQLite view 不可 INSERT,test 4/5 直接 INSERT INTO reconciliation_normalized 失败;同时 `file::memory:?cache=shared` 跨 test 共享 state 导致污染
- **Fix:** test schema 拆为 3 张表(ops_asset_physical / ops_asset_declared / ops_asset_ad),reconciliation_normalized 改为基于这 3 张表做 LEFT JOIN 的 view;每个 test 用独立 DSN cache name(`file:test_type_a?...` / `file:test_duplicate?...`)
- **Files modified:** `internal/services/asset/reconciliation_test.go`
- **Verification:** 5 tests 全部 PASS
- **Committed in:** `ce617b0a` (Task 3 commit)

**3. [Test infrastructure - BaseModel 字段] 漏掉 created_by/updated_by/version 列**
- **Found during:** Task 3 (running TestDetectLayer3)
- **Issue:** GORM Create() 报 `no column named created_by`(BaseModel 字段),SQLite 建表时漏
- **Fix:** setupTestDB 增加 created_by / updated_by / version 三列
- **Files modified:** `internal/services/asset/reconciliation_test.go`
- **Verification:** `go test -v ./internal/services/asset/...` 全部 PASS
- **Committed in:** `ce617b0a` (Task 3 commit)

**Total deviations:** 3 auto-fixed (1 JOIN scope + 2 test infra)
**Impact on plan:** All auto-fixes necessary for correctness/test stability. No scope creep beyond what plan specified.

## Issues Encountered

- **Test 5 logrus debug 日志**:第二次 DetectLayer3 触发 SQLite UNIQUE constraint failed,isReconciliationDuplicate 识别并 log.Debug,test 仍 PASS(D-11 语义正确)。属预期行为,非问题。

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

**Ready:**
- 42-03 plan 可以直接消费 ReconciliationService.ListExceptions 端点
- 42-04 plan 可以直接消费 ReconciliationSnapshot.RefreshView + 4 cron global functions
- 42-05/42-06 可以在 main router.go 注册两个 Setup*Router(已留 interface 入口)
- Layer 3 检测引擎可在 cron 触发后写入 sys_data_reconciliation,Type A 跳过
- D-11 unique violation catch 路径已 unit test 验证
- ModuleReconciliation 常量已定义,R2 扩展时仅需在 reconciliation_handler.go 加新 const

**Blockers / Concerns:**
- 4 个 cron InvokeTarget 全局函数当前是 stub,无 core 依赖 → 实际生产路径需要 service binding 注入,42-04/42-06 计划中需补
- ReconciliationException 是 R1 skeleton,无 Create/Update/Delete handler,R3 接入时需扩展
- refreshView 失败仅 log + 返回 err 的策略:实际 cron 调度时调用方(ExecuteRefreshViewTask)会收到 err,需确认 cron framework 写 job_log 的路径与本方法契约一致(目前 stub 未做实际 job_log 写入,42-04 落实)

**42-06 路由注册建议**(per CONTEXT.md D-21 / router.go):
```go
// 在 internal/api/router.go 主路由注册
assetReconciliation := r.Group("/asset/reconciliation")
assetReconciliation.Use(middleware.JWTAuth(core.JWTManager))
assetReconciliation.Use(middleware.OperLogMiddleware(core.OperLogService, core))
assetReconciliation.Use(middleware.RequirePermissions([]string{
    "asset:reconciliation:list",
    "asset:reconciliation:dashboard",
    "asset:reconciliation:export",
}, core))
{
    asset.SetupReconciliationRouter(assetReconciliation, core)
    asset.SetupReconciliationExceptionRouter(assetReconciliation, core)
}
```

---
*Phase: 42-r1-资产对账观测底座 (R1)*
*Plan: 02 — Service layer (read + detect) + cron stubs + handler/router*
*Completed: 2026-06-27*
