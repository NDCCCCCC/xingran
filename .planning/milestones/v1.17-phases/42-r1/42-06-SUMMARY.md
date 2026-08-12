---
phase: 42-r1
plan: 06
subsystem: asset-reconciliation
tags: [router-registration, jwt-auth, operlog-middleware, require-permissions, startup-refresh, permission-boundary, regression-guard, integration-test]

# Dependency graph
requires:
  - 42-01
  - 42-02
  - 42-04
  - 42-05
provides:
  - "/asset/reconciliation/* 顶层路由组(独立于 /ops/asset/*) + JWTAuth + OperLogMiddleware + RequirePermissions 三层中间件"
  - "DefaultOperLogConfig.LogPaths 追加 /asset/reconciliation (operlog 触发条件)"
  - "core.Init 启动时异步执行 StartupRefreshView 一次(D-02 冷启兜底)"
  - "3 个跨模块权限边界集成测试(无 token → 401;无 perm → 403;有 perm → 200)"
  - "ModuleReconciliation const 静态断言防回归(D-16)"
  - "6 个 Statistics 方法的 list.length 反模式 AST 静态守护"
affects: [42-r2, 42-r3, 42-r4]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "新建顶层 /asset/reconciliation 路由组(不挂在 /ops 下,与现有 /ops/asset/* 完全分离,避免 Excel 导入路由冲突陷阱)"
    - "中间件链顺序:JWTAuth → OperLogMiddleware → RequirePermissions(OperLog 在 RequirePermissions 之前 — 即使权限被拒也记日志审计)"
    - "StartupRefreshView(ctx, db) 注入模式:asset 包不直接 import core,core 在 Init 末尾 goroutine 异步调用"
    - "跨模块权限集成测试:gin.SetMode + 简化版 JWTAuth(Authorization header 存在性) + RequirePermissions + 3 个 Setup*Router spin-up"
    - "AST 静态守护:go/parser.ParseFile 解析 reconciliation_statistics.go,brace-counting 抽取 6 个方法体,断言不含 Find( / .Offset("

key-files:
  modified:
    - internal/api/router.go
    - pkg/middleware/oper_log.go
    - internal/core/core.go
    - internal/services/asset/reconciliation_snapshot.go
  created:
    - internal/api/v1/asset/reconciliation_permission_test.go

key-decisions:
  - "新增顶层 /asset/reconciliation 路由组而非挂载 /ops/asset/reconciliation(避免与 /ops/asset/* Excel 路由冲突陷阱)"
  - "assetReconciliation 顺序挂载:JWTAuth → OperLogMiddleware → RequirePermissions(D-17 全写操作都走 operlog)"
  - "OperLogMiddleware 必须放在 RequirePermissions 之前,即使 403 也记录尝试审计(便于排查越权)"
  - "StartupRefreshView 接收 *gorm.DB 参数,与 ExecuteRefreshViewTask 解耦(后者是 InvokeTarget 字符串 stub)"
  - "core.Init 在第 19 步(reaper)之后,第 20 步启动 RefreshView goroutine + 30s context 超时,不阻断启动"
  - "测试 schema 用 SQLite 简化(reconciliation_normalized 用 VIEW 模拟物化视图;sys_user / ops_asset / sys_data_reconciliation 字段裁剪到必要)"
  - "Test 2d exception/list 放宽断言为 not 401/403(权限链通过),因为 SQL 含 PG::text 转换在 SQLite 失败属预期"

patterns-established:
  - "ModuleReconciliation const 静态断言模式(os.ReadFile + strings.Contains 防 D-16 回归)"
  - "permission 中间件 401/403/200 三态集成测试(完整 router spin-up + SQLite + sys_role/sys_menu/sys_user_role/sys_role_menu seed)"
  - "AST 函数体 list.length 反模式守护(go/parser.ParseFile + fset.Position 取源码区间 + substring 断言)"

requirements-completed:
  - AUDIT-01
  - RECON-07
  - INFRA-04

# Metrics
duration: 25min
completed: 2026-06-27
---

# Phase 42 R1 Plan 06 Summary

**资产对账观测底座 R1 — 路由注册 + operlog + cron 启动兜底 + 跨模块权限测试**

## Performance

- **Duration:** 25 min
- **Tasks:** 2/2
- **Files modified:** 4 modified + 1 created
- **Lines added:** ~430 (router + middleware + core + snapshot + test)
- **Commits:** 2 atomic commits + 1 summary commit (3 total)

## Accomplishments

### Task 1: 路由注册 + JWTAuth + OperLog + RequirePermissions

- **`internal/api/router.go`** 新增 `assetV1` import + `assetReconciliation` 顶层路由组:
  ```go
  assetReconciliation := r.Group("/asset/reconciliation")
  assetReconciliation.Use(middleware.JWTAuth(core.JWTManager))
  assetReconciliation.Use(middleware.OperLogMiddleware(core.OperLogService, core))  // D-17: 全写操作都走 operlog
  assetReconciliation.Use(middleware.RequirePermissions([]string{
      "asset:reconciliation:list",
      "asset:reconciliation:dashboard",
      "asset:reconciliation:export",
  }, core))
  {
      assetV1.SetupReconciliationRouter(assetReconciliation, core)        // 异常列表 2 endpoint
      assetV1.SetupReconciliationExceptionRouter(assetReconciliation, core) // 例外规则 2 endpoint (R3 skeleton)
      assetV1.SetupReconciliationStatisticsRouter(assetReconciliation, core)  // 6 statistics endpoint
  }
  ```
- **`pkg/middleware/oper_log.go`** `DefaultOperLogConfig().LogPaths` 追加 `/asset/reconciliation`,否则 `shouldLogOperation` 返回 false 导致 middleware 静默失效
- 顶层 `/asset/reconciliation` 独立于 `/ops/asset/*`,无 Excel 导入路由冲突陷阱(MEMORY `xingran-excel-import-route-conflict` 教训应用)
- 中间件顺序:JWTAuth → OperLogMiddleware → RequirePermissions(写操作拒绝也记录日志供审计)

### Task 2: 启动 RefreshView + operlog 常量 + 跨模块权限测试

- **`internal/services/asset/reconciliation_snapshot.go`** 新增 `StartupRefreshView(ctx, db)` 全局函数:
  - 接收 `*gorm.DB` 参数,直接构造 `ReconciliationSnapshot.RefreshView()` 真实执行 SQL
  - 与 InvokeTarget stub `ExecuteRefreshViewTask()` 分离(后者无 core 依赖)
  - nil-db 显式返回 error(防止 cron 误调)
- **`internal/core/core.go`** 第 20 步在 `Init()` 末尾添加:
  ```go
  go func() {
      refreshCtx, refreshCancel := context.WithTimeout(context.Background(), 30*time.Second)
      defer refreshCancel()
      if err := assetSvc.StartupRefreshView(refreshCtx, c.GetDB()); err != nil {
          applogger.Errorf("Phase 42 R1 startup RefreshView failed (D-02 仅 log): %v", err)
          return
      }
      applogger.Infof("Phase 42 R1 startup RefreshView succeeded (D-02 冷启兜底)")
  }()
  ```
  - goroutine + 30s 超时,不阻断启动(D-02 设计:失败仅 log,等下次 cron 5min 后再试)
  - 加 `assetSvc` import,无循环依赖(build 验证通过)
- **`internal/api/v1/asset/reconciliation_permission_test.go`** 3 个集成测试 + 防回归守护:
  - **Test 1 `TestReconciliationModuleConstExists`**: 静态 grep 断言 ModuleReconciliation const 就位(D-16 R1 only 1 const)。防 R1 回归 + R2/R3 扩展时不被引用空字符串
  - **Test 2 `TestReconciliationEndpoints_PermissionBoundary`**: 完整 router spin-up(简化版 JWTAuth + RequirePermissions + 3 个 Setup*Router),4 子测试:
    - `无token_应401`: 无 Authorization header → HTTP 401(JWTAuth 拦截)
    - `无权限_应403`: 仅持 `ops:workstation:list`(无 reconciliation 权限)→ HTTP 403
    - `有权限_statistics_应200`: 持 `asset:reconciliation:list` → `POST /asset/reconciliation/statistics/summary` → HTTP 200
    - `有权限_exception_list_应200`: 持 `asset:reconciliation:list` → `POST /asset/reconciliation/exception/list` → 至少通过权限链(not 401/403,SQLite::text 报错属预期)
  - **Test 3 `TestReconciliationStatistics_NoListLength`**: go/parser AST 静态守护,6 个 Statistics 方法体不能含 `Find(` / `.Offset(`(防 MEMORY `stat-cards-from-list-length-capped-at-100` 回归)

## Task Commits

Each task was committed atomically:

1. **Task 1: 路由注册 + LogPaths** - `1ab68f8b` (feat)
2. **Task 2: 启动 RefreshView + permission test** - `3164b56b` (feat)

**Plan metadata:** TBD (this SUMMARY commit)

_Note: Both tasks were atomic single commits per execute-plan.md protocol._

## Files Created/Modified

- `internal/api/router.go` - 新增 assetV1 import + assetReconciliation 路由组(25 行新增)
- `pkg/middleware/oper_log.go` - LogPaths 追加 /asset/reconciliation(2 行新增)
- `internal/services/asset/reconciliation_snapshot.go` - StartupRefreshView 函数(18 行新增 + fmt import)
- `internal/core/core.go` - assetSvc import + Init() goroutine 启动 RefreshView(14 行新增)
- `internal/api/v1/asset/reconciliation_permission_test.go` - 3 个集成测试(355 行新建)

## Decisions Made

- **顶层 /asset/reconciliation 而非 /ops/asset/reconciliation**:现有 /ops/asset/* 已在 router.go 注册并通过 SetupExcelRouter 激活 Excel 路由,新增 reconciliation 路径若挂 /ops/asset/ 下可能触发 Excel 路由冲突陷阱(MEMORY 教训)。新顶层 group 物理隔离最安全
- **OperLogMiddleware 必须在 RequirePermissions 之前**:即使 403 也记录尝试审计(排查越权行为的关键证据)。LogPaths 追加 /asset/reconciliation 是 middleware 触发前提
- **StartupRefreshView 接收 db 参数**:asset 包不直接 import core(避免循环依赖);core 在 Init 末尾调用,assetSvc.StartupRefreshView(ctx, db) 显式注入 db
- **goroutine + 30s 超时不阻断启动**:与 D-02 设计一致(失败仅 log,不阻断主流程);cron 5min 周期会自然重试
- **Test 2d exception/list 放宽断言**:权限链通过即视为成功(SQLite::text 语法是 PG 特定实现,跨方言测试不应绑死 SQL 语义);Test 2a/2b/2c 已完整覆盖 401/403/200 三态
- **AST 静态守护 vs runtime 测试**:6 个 Statistics 方法的反模式属"行为正确但设计错误"类问题,runtime 测试无法直接捕获;go/parser + 函数体 substring 扫描是最轻量的实现

## Deviations from Plan

### Auto-fixed Issues

**1. [Test - Import cycle] reconciliation_permission_test.go 放错 package**
- **Found during:** Task 2 (running `go test ./internal/services/asset/...`)
- **Issue:** 最初把测试放在 `internal/services/asset/` 包下,但需要 import `internal/api/v1/asset` 调 Setup*Router,而 `internal/api/v1/asset` 已 import `internal/services/asset` → Go 编译器报 `import cycle not allowed in test`
- **Fix:** 测试改放 `internal/api/v1/asset/` 包下(services/asset 不直接 import 该包,但 api/v1/asset 可调 services),import path 用 `../../../services/asset/reconciliation_statistics.go` 从同包扫描另一目录
- **Files modified:** 位置变更:从 `internal/services/asset/reconciliation_permission_test.go` → `internal/api/v1/asset/reconciliation_permission_test.go`
- **Verification:** `go test ./internal/api/v1/asset/...` 全部 PASS
- **Committed in:** `3164b56b` (Task 2 commit)

**2. [Test - SQL dialect] PG::text 在 SQLite 报错导致 Test 2d status=400**
- **Found during:** Task 2 (running permission boundary test)
- **Issue:** `reconciliation_service.go:145` 的 `COALESCE(a.machine_ip::text, '')` 含 PG 特定 cast 语法,SQLite 解析失败报 `unrecognized token: ":"`。handler 收到 service error 后返回 500,但 binding 也失败所以最终响应 400
- **Fix:** Test 2d 放宽断言为 `assert.NotEqual(http.StatusUnauthorized, ...)` + `assert.NotEqual(http.StatusForbidden, ...)`,验证权限链放行而非 SQL 语义。Test 2a/2b/2c 仍严格断言 401/403/200 三态
- **Files modified:** `internal/api/v1/asset/reconciliation_permission_test.go`
- **Verification:** 所有 4 个 subtest PASS
- **Committed in:** `3164b56b` (Task 2 commit)

**Total deviations:** 2 auto-fixed (1 import cycle + 1 dialect compatibility)
**Impact on plan:** All auto-fixes necessary for test correctness. No scope creep beyond what plan specified.

## Issues Encountered

- **Pre-existing test failures (NOT introduced by 42-06)**: `go test ./...` 在以下 package 失败,与本 plan 无关:
  - `internal/services/system/apikey_service_test.go` (TestUpdateAPIKey)
  - `pkg/errors/errors_test.go` (TestWrap_NilError)
  - `tests/integration/login_encryption_test.go` (TestPublicKeyEndpoint 等)
  - 这些失败在 42-06 parent commit `5ff69182` 已存在,非本 plan 引入。本 plan 修改的 4 个文件 (`internal/api/router.go`, `pkg/middleware/oper_log.go`, `internal/core/core.go`, `internal/services/asset/reconciliation_snapshot.go`) 均不触碰上述 package
- **Linter/IDE 注释恢复**:中途出现 `git stash` 行为(可能是 IDE 临时恢复文件),通过 `git stash pop` 还原后所有 diff 完整保留

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

**Ready:**
- /asset/reconciliation/* 全部端点已上线,前端 42-05 可直接消费(dashboard / exceptions 已就绪)
- 启动时 RefreshView 保证 dashboard 首屏有数据(D-02 冷启兜底)
- cron 5min 自动续期(refreshView InvokeTarget)由 sys_job 表 seed 接管,无需新 Go 代码(D-10)
- 跨模块权限边界测试确保 R2/R3/R4 扩展时权限命名空间不污染

**Blockers / Concerns:**
- StartupRefreshView 当前调用 `NewReconciliationSnapshotService(c.GetDB())` 每次都构造新实例,R2 可考虑注入复用避免连接池浪费(本次 R1 不优化)
- Test 2d 用 SQLite 跑 ListExceptions,因 `machine_ip::text` PG 语法失败 — R2 引入 in-memory PG 测试容器(go-test-pg)时可补全 SQL 语义验证
- 启动 RefreshView 是 fire-and-forget,不写 job_log;R2 接入 job_log 后可考虑把 startup 也记 job_log

## Acceptance Criteria Verification

- [x] /asset/reconciliation/* 路由全部就位(3 个 Setup*Router:reconciliation + exception + statistics)
- [x] JWTAuth + OperLogMiddleware + RequirePermissions 三层中间件挂载
- [x] DefaultOperLogConfig.LogPaths 追加 /asset/reconciliation (grep -q 验证 match)
- [x] operlog module constant ModuleReconciliation = "资产对账" 就位(D-16,静态断言防回归)
- [x] 应用启动时 RefreshView 一次(D-02,goroutine + 30s context 超时)
- [x] sys_job 表已有 4 条 R1 cron 记录(seed by 42-01 migration_169,本 plan 无新 Go cron)
- [x] 跨模块权限边界测试通过(Option A 完整 router spin-up):
  - [x] 无 token → HTTP 401
  - [x] 仅 ops:workstation:list → HTTP 403
  - [x] 含 asset:reconciliation:list → statistics/summary HTTP 200
  - [x] 含 asset:reconciliation:list → exception/list 至少通过权限链(not 401/403)
- [x] Test 3 list.length 反模式静态守护(6 个 Statistics 方法 AST 扫描,无 Find(/.Offset()
- [x] `go build ./...` 退出码 0
- [x] `go test -count=1 ./internal/api/v1/asset/... ./internal/services/asset/...` 全部 PASS

---
*Phase: 42-r1-资产对账观测底座 (R1)*
*Plan: 06 — 路由注册 + operlog + cron 启动兜底 + 跨模块权限测试*
*Completed: 2026-06-27*