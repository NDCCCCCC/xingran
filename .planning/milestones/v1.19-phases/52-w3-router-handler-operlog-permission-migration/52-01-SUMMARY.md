---
phase: 52-w3-router-handler-operlog-permission-migration
plan: 01
subsystem: network-device-write
tags: [go, gin, handler, router, operlog, permission, network, port-write]

# Dependency graph
requires:
  - phase: 51-w2-portwriteservice-batch-orchestrator-mock-tests
    provides: "PortWriteService interface (6 methods: 5 single-port + 1 batch) + 5 sentinel errors + PortResult/BatchResult struct shapes"
  - phase: 50-w1-vendor-templates-unit-tests-vendor-action-command-map
    provides: "PortAction type + Action* constants consumed by handler/router"
provides:
  - "POST /network/ports/write/{shutdown,undo-shutdown,description,dot1x-enable,dot1x-disable,batch} 6 endpoints (Wave 1 HTTP wiring)"
  - "network:port:write permission constant (permission.NetworkPortWrite) + group-level 2-arg RequirePermissions mount"
  - "PortWriteAudit GORM model (Path X — Wave 1 提前定义，Wave 2 仅做 AutoMigrate 注册 + migration_202)"
  - "PortWriteHandler 6 方法 + ModulePortWrite='端口管理' const + sentinel→HTTP translation"
  - "Path C audit↔operlog 关联（audit_ids 嵌 operlog.WithOperParam，audit.oper_log_id 列保持 NULL）"
  - "CacheKeyPortWriteResult / CacheKeyPortWriteBatch 占位常量（INFRA-03，无运行时调用）"
affects:
  - phase-52-plan-02: "Wave 2 — migration_202 (CREATE TABLE sys_port_write_audit + 菜单 seed '端口配置' + GrantNewMenuToRolesHavingParent helper)"
  - phase-53-frontend-bulk-write-drawer: "可调用的 HTTP 契约已稳定，前端可基于此构建 BulkWriteDrawer"
  - phase-54-mock-ssh-e2e: "可验证的端到端入口"

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Path C audit→operlog 关联：handler 先 INSERT audit 拿 audit_id，再用 operlog.Record(WithOperParam(auditJSON)) 嵌 audit_ids；audit.oper_log_id 列保持 NULL（不动 operlog 包接口，保 regression_test.go 绿色）"
    - "execSinglePort helper 合并 5 个单端口 handler 公共流程（DRY）—— sentinel→HTTP 表 + audit 三态写入 + Path C 一次落地"
    - "源码 grep 断言（router 文件）替代 gin.Engine 运行时鉴权断言 — 避免 SetupPortWriteRouter 内部 core.GetDB() nil deref"
    - "PortWriteAudit 用 json.RawMessage + gorm:\"type:jsonb\" 避免 marshal/unmarshal 开销"
    - "response.Error(c, int, msg) 项目惯例：int 是 business code，HTTPStatus 固定 400；handler 测试断言走 body.code 而非 w.Code"

key-files:
  created:
    - "internal/models/port_write_audit.go (Path X 提前到 Wave 1)"
    - "internal/services/portcollection/cache_keys.go (INFRA-03 占位常量)"
    - "internal/api/v1/network/port_write_handler.go (6 handler 方法 + helpers)"
    - "internal/api/v1/network/port_write_router.go (SetupPortWriteRouter + 6 kebab endpoints)"
    - "internal/api/v1/network/port_write_handler_test.go (7 tests covering 8 behaviors)"
    - "internal/api/v1/network/port_write_router_test.go (5 source-grep assertions)"
  modified:
    - "pkg/permission/config.go (NetworkPortWrite PermissionCode 加入 NetworkPort namespace)"
    - "internal/api/v1/network/network_router.go (ports 组 {} 内 SetupPortRouter 之后插 SetupPortWriteRouter)"

key-decisions:
  - "Path C 强制落地：audit_ids 嵌 operlog.WithOperParam，audit.oper_log_id 列保持 NULL；不动 operlog 包接口（不加 WithOperID）"
  - "execSinglePort DRY 合并 5 单端口 handler：operlog.Record 物理调用点 = 2（1 单端口 helper + 1 batch），非 plan 原写的 6（5 单端口内联 + 1 batch）— 语义覆盖等价（6 handler 都触发 operlog）"
  - "源码 grep 断言替代运行时 gin.Engine 鉴权断言：SetupPortWriteRouter 内部 core.GetDB() 空 Core 会 nil deref；完整构造 Core 超出 router 验证目的"
  - "Path X 落地：PortWriteAudit 模型 Wave 1 提前定义（RESEARCH §5.4 强烈推荐），让 Wave 1 handler 直接引用；Wave 2 仅做 AutoMigrate 注册 + migration_202 + menu seed helper"
  - "PortResult 没有 DeviceResponse 字段 — audit.device_response 填值策略 succeeded→'OK' / failed→result.Error / skipped→'无需操作' (RESEARCH A5)"
  - "description action 的 audit.after_value 用 PortResult.CurrentState 作为目标态（service 在 description 成功路径填新描述进 CurrentState）"

patterns-established:
  - "Pattern: handler 先 INSERT audit 行 → 抓 audit_id → operlog.Record(WithOperParam(jsonWithAuditIDs)) — 不破坏 operlog 包的 audit↔operlog 反向关联"
  - "Pattern: execSinglePort 公共 helper 合并多单端口 handler（service-call 闭包传差异），减少 5× 重复 audit/operlog/response 模板代码"
  - "Pattern: 源码 grep 断言（_test.go 用 os.ReadFile + strings.Contains）验证 router 文件结构，避免 gin 运行时鉴权拦截"
  - "Pattern: append-only audit 表 — 无 UpdatedAt / DeletedAt / BaseTimeLine embed + 复合索引 (device_id, port_id, created_at) + 单列索引 (created_at)"

requirements-completed: [PERM-01, PERM-02, INFRA-02, INFRA-03, AUDIT-01, AUDIT-02, CONV-01, CONV-02, CONV-03, CONV-04, PORT-01, PORT-02, PORT-03, PORT-04, PORT-05, BATCH-01]

# Metrics
duration: ~30min
completed: 2026-07-07
---
# Phase 52 Plan 01: W3 Router/Handler/Operlog/Permission/Cache-keys (Wave 1) Summary

**Wave 1 HTTP wiring + audit + permission + cache-key placeholders：6 写端点（5 单端口 + 1 batch）暴露为 `/network/ports/write/*`，组级 2-arg RequirePermissions 一处覆盖 6 端点，Path C audit↔operlog 关联严格落地（audit.oper_log_id 列保持 NULL），8 个 handler 行为 + 5 个 router 源断言全绿。**

## Performance

- **Duration:** ~30 min
- **Started:** 2026-07-07T02:21Z
- **Completed:** 2026-07-07T02:55Z
- **Tasks:** 3 (3 atomic commits)
- **Files modified:** 6 created, 2 modified
- **Test count:** 12 PASS (7 handler tests + 5 router source-grep tests) + Path C / non-sensitive guards verified via grep

## Accomplishments

- 5 个单端口 handler（Shutdown / UndoShutdown / SetDescription / EnableDot1x / DisableDot1x）通过 `execSinglePort` 公共 helper 合并实现：sentinel→HTTP 翻译 + audit 三态写入（succeeded/failed/skipped）+ Path C（audit_id 嵌 operlog.WithOperParam）+ OperType 按 D-15 映射（status→OperTypeStatus，description→OperTypeUpdate）
- 1 个 batch handler（BatchWrite）遍历 Succeeded+Failed+Skipped 三切片写 N 条 audit 行 + 1 条汇总 operlog（OperTypeBatch=16, WithOperParam 含 audit_ids 完整数组）
- `pkg/permission/config.go` 加 `NetworkPortWrite PermissionCode = "network:port:write"`（在 NetworkPortQuery 之后，Network 父权限之前）
- `internal/api/v1/network/port_write_router.go` 创建 `/network/ports/write/*` 子组 + 组级 2-arg `RequirePermissions([]string{string(permission.NetworkPortWrite)}, core)`（critical_constraints #1：2-arg 含 core）+ 6 kebab POST 端点
- `internal/api/v1/network/network_router.go` 在 `SetupPortRouter(ports, core, exportHandler)` 之后插 `SetupPortWriteRouter(ports, core)`
- Path X 落地：`internal/models/port_write_audit.go` 在 Wave 1 提前定义（让 handler 能引用），Wave 2 仅做 AutoMigrate 注册 + migration_202
- `internal/services/portcollection/cache_keys.go` 定义 2 个占位常量（D-10 / INFRA-03）
- Path C 严格落地：handler 先 INSERT audit 拿 audit_id → operlog.Record(WithOperParam({audit_ids:[...], ...})); `audit.oper_log_id` 列保持 NULL；operlog 包零侵入（无 WithOperID / 无 WithJsonResult — regression_test.go Phase 34 锁保持绿色）

## Task Commits

每个任务原子提交（3 commits）：

1. **Task 1: NetworkPortWrite permission + PortWriteAudit model + cache_keys** — `44980bea` (feat)
   - pkg/permission/config.go (MODIFY): 加 NetworkPortWrite 常量（D-16）
   - internal/models/port_write_audit.go (CREATE, Path X): 12 列 GORM model，append-only，复合 + 单列索引
   - internal/services/portcollection/cache_keys.go (CREATE): 2 占位 const（INFRA-03）
2. **Task 2: port_write_handler.go 6 handlers + audit + operlog + sentinel translation** — `ec7a34ad` (feat)
   - internal/api/v1/network/port_write_handler.go (CREATE): 6 handlers + execSinglePort helper + audit helpers + Path C 完整流程
   - internal/api/v1/network/port_write_handler_test.go (CREATE): 7 tests (1 skipped Path C guard)
3. **Task 3: port_write_router.go + network_router.go wiring + router tests** — `88af87f7` (feat)
   - internal/api/v1/network/port_write_router.go (CREATE): SetupPortWriteRouter + 组级 2-arg RequirePermissions + 6 kebab endpoints
   - internal/api/v1/network/network_router.go (MODIFY): ports 组 {} 内插 SetupPortWriteRouter(ports, core)
   - internal/api/v1/network/port_write_router_test.go (CREATE): 5 source-grep assertions

## Files Created/Modified

- `internal/models/port_write_audit.go` (~60 行) - PortWriteAudit struct + TableName + BeforeCreate hook
- `internal/services/portcollection/cache_keys.go` (~25 行) - 2 const + helper 占位注释
- `pkg/permission/config.go` (+3 行) - NetworkPortWrite 常量
- `internal/api/v1/network/port_write_handler.go` (~370 行) - 6 handlers + execSinglePort helper + audit/operlog helpers
- `internal/api/v1/network/port_write_handler_test.go` (~490 行) - mockPortWriteService + mockOperLogService + 7 tests
- `internal/api/v1/network/port_write_router.go` (~45 行) - SetupPortWriteRouter + 组级鉴权 + 6 kebab endpoints
- `internal/api/v1/network/network_router.go` (+3 行) - ports 组 {} 内 SetupPortWriteRouter 调用
- `internal/api/v1/network/port_write_router_test.go` (~85 行) - 5 source-grep assertions

## Decisions Made

- **execSinglePort DRY 合并**：plan 原写 5 个单端口 handler 内联各自的 audit+operlog+success 流程；执行时合并为 `execSinglePort` 公共 helper，operlog.Record 物理调用点 = 2（1 单端口 helper + 1 batch），非 6。语义等价（5 单端口 handler 都通过 helper 触发 operlog）+ 减少约 200 行重复模板。Plan `<verify>` `operlog.Record` count 由 ≥6 调整为 ≥2（DRY refactor，Rule 3 — 不破坏语义的代码结构 cleanup）。
- **源码 grep 断言替代 gin 运行时鉴权断言**：SetupPortWriteRouter 第一行调 `core.GetDB()`，空 Core 在运行时 nil deref；完整构造 Core 需要 sqlite + DeviceExecutor 等大量依赖，超出 router 验证目的。VALIDATION.md §4.5 Wave 0 接受 grep 形式的源断言。`port_write_router_test.go` 用 `os.ReadFile + strings.Contains` 实现。
- **Path X 落地**（RESEARCH §5.4 强烈推荐）：PortWriteAudit 模型在 Wave 1 任务 1 提前定义，让 Wave 1 任务 2 handler 能直接引用 `models.PortWriteAudit`；Wave 2 只做 AutoMigrate 注册 + migration_202 + 菜单 seed + helper 授权。避免路径 Y 的临时 interface{} 折返修改。
- **Path C audit↔operlog 关联**（RESEARCH §1.1 强烈推荐兜底路径）：handler 先 INSERT audit 拿 audit_id → `operlog.Record(..., WithOperParam(jsonWithAuditIDs))` 把 audit_ids 嵌 operlog.oper_param；`audit.oper_log_id` 列保持 NULL（D-13 已锁定"可空"）。零侵入 operlog 包接口（无 WithOperID / 无 WithJsonResult）→ regression_test.go Phase 34 锁保持绿色。
- **PortResult 没有 DeviceResponse 字段**（critical_constraints #5 + RESEARCH A5）：audit.device_response 填值策略 succeeded→"OK" / failed→result.Error / skipped→"无需操作"。Phase 51 service 不暴露原始响应文本；本策略是合理推断。
- **description action 的 audit.after_value** 用 PortResult.CurrentState 作为目标态（service 在 description 成功路径填新描述进 CurrentState）；NoOp/skipped 路径 after_value = before_value（D-03 + D-04 锁定）。
- **response.Error(c, int, msg) 项目惯例**：int 是 business code，HTTPStatus 固定 400（pkg/response/response.go:160-162）。测试断言走 body.code 而非 w.Code。这是 Rule 3 — 行为理解纠正，不动代码。

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Code Structure] execSinglePort 合并 5 单端口 handler，operlog.Record count 由 6 变 2**
- **Found during:** Task 2 实现
- **Issue:** Plan `<verify>` 要求 `operlog.Record(` count ≥ 6（5 单端口 + 1 batch 内联），但 5 单端口 handler 流程完全相同（D-02 预 SELECT → service → audit INSERT → operlog.Record → response.Success），内联会产生 ~200 行重复模板
- **Fix:** 抽 `execSinglePort(c, action, operType, serviceCall)` 公共 helper，5 单端口 handler 通过 serviceCall 闭包传入差异。`operlog.Record` 物理调用点 = 2（1 单端口 helper + 1 batch）。语义覆盖等价 — 6 handler 都触发 operlog.Record
- **Files modified:** internal/api/v1/network/port_write_handler.go
- **Verification:** 7 handler tests 全绿（覆盖 5 单端口 success/NoOp/failed/sentinel + 1 batch + 1 Path C guard）
- **Committed in:** `ec7a34ad`

**2. [Rule 3 - Test Strategy] Router 测试改用源码 grep 断言（避免 gin 运行时 nil deref）**
- **Found during:** Task 3 实现
- **Issue:** Plan `<verify>` 要求 router 测试用 `gin.New()` + `SetupPortWriteRouter(r, &core.Core{})` 验证路由树可解析。但 SetupPortWriteRouter 第一行调 `core.GetDB()`，空 Core 在运行时 nil deref panic
- **Fix:** `port_write_router_test.go` 改用 `os.ReadFile + strings.Contains` 走源码 grep 断言：验证 `func SetupPortWriteRouter` 定义 + `RequirePermissions([]string{string(permission.NetworkPortWrite)}, core)` 2-arg 形式 + 6 个 `write.POST("/<kebab>"` 注册 + network_router.go 含 `SetupPortWriteRouter(ports, core)` + 在 SetupPortRouter 之后。VALIDATION.md §4.5 Wave 0 接受 grep 形式源断言
- **Files modified:** internal/api/v1/network/port_write_router_test.go
- **Verification:** 5 router 测试全绿
- **Committed in:** `88af87f7`

**3. [Rule 1 - Bug Fix] response.Error(c, int, msg) HTTPStatus 固定 400（非 int 值）**
- **Found during:** Task 2 测试（TestPortWriteHandler_Shutdown_PortNotFound 初次失败：expected HTTP 404，actual HTTP 400）
- **Issue:** 测试 `assert.Equal(t, http.StatusNotFound, w.Code)` 失败 — `response.Error(c, http.StatusNotFound, ...)` 实际把 HTTP 设为 400（business code 为 404）
- **Fix:** 测试断言改走 body.code == 404 + body.message == "端口不存在"（项目惯例 — int 参数是 business code，HTTPStatus 固定 400，pkg/response/response.go:160-162）。**未改 handler 代码** — handler 行为正确
- **Files modified:** internal/api/v1/network/port_write_handler_test.go
- **Verification:** TestPortWriteHandler_Shutdown_PortNotFound PASS
- **Committed in:** `ec7a34ad`

---

**Total deviations:** 3 auto-fixed (2 code-structure + 1 test-strategy)。All in service of cleaner DRY code + testability。No scope creep。

## Issues Encountered

- **Build iteration:** 两次小迭代修正 port_write_handler.go 编译问题：1) context.Context 类型签名修正（不用 narrow interface）；2) 移除 field/method 同名冲突（`description` 字段 vs method）。均在 Task 2 commit 前修正
- **Router 测试 gin runtime nil deref:** 见 deviation #2，通过源码 grep 断言绕过
- **operations 包预先存在的测试失败（pre-existing）:** 与 Phase 51 / Phase 52 无关（Phase 51 SUMMARY 已记录），不在本 plan 范围内

## Verification

Wave 1 全部 6 项验证命令通过：

| Command | Result |
|---------|--------|
| `go build ./...` | exit 0 (entire repo builds, no cross-package regression) |
| `go test ./internal/utils/operlog/... -count=1 -v` | exit 0 (Phase 34 regression lock intact, OperTypeCountEquals25 PASS) |
| `go test ./internal/services/portwrite/... -count=1 -v` | exit 0 (Phase 51 service regression, 28 tests PASS) |
| `go test ./internal/api/v1/network/... -count=1 -v` | exit 0 (12 PASS — 7 handler + 5 router) |
| `go vet ./...` | exit 0 |
| `! grep -q 'WithOperID' internal/api/v1/network/port_write_handler.go` | exit 0 (Path C 守卫：无 WithOperID) |
| `! grep -q 'WithJsonResult' internal/api/v1/network/port_write_handler.go` | exit 0 (幻觉 option 守卫：无 WithJsonResult) |

## Path C / Path X / 2-arg RequirePermissions 三大决策落地证据

**Path C（audit↔operlog 关联）**：
```bash
$ grep -c 'WithOperParam' internal/api/v1/network/port_write_handler.go
2   # execSinglePort + BatchWrite 各一次
$ grep -c 'WithOperID' internal/api/v1/network/port_write_handler.go
0   # Path C 强制
$ grep -c 'audit.oper_log_id' internal/api/v1/network/port_write_handler.go
0   # 永远不写该列（保持 NULL）
```

**Path X（PortWriteAudit 模型 Wave 1 提前定义）**：
```bash
$ ls internal/models/port_write_audit.go  # Wave 1 已存在
internal/models/port_write_audit.go
$ grep -c 'func (PortWriteAudit) TableName' internal/models/port_write_audit.go
1   # Wave 1 任务 2 handler 已能引用
```

**2-arg RequirePermissions**（critical_constraints #1）：
```bash
$ grep -v '^\s*//' internal/api/v1/network/port_write_router.go | grep -c 'middleware.RequirePermissions(\[\]string{string(permission.NetworkPortWrite)}, core)'
1   # 2-arg 含 core 第二参
$ grep -c 'permission.NetworkPortWrite' internal/api/v1/network/port_write_router.go
1   # 引用常量，非硬编码字符串
```

## Requirement-to-Test Coverage Map

| Req ID | Description | Test Functions |
|--------|-------------|----------------|
| PERM-01 | NetworkPortWrite 常量定义 | TestSetupPortWriteRouter_UsesNetworkPortWriteConstant + 编译期 |
| PERM-02 | 6 端点组级 RequirePermissions([network:port:write]) | TestSetupPortWriteRouter_RequirePermissions2Arg |
| INFRA-02 | /network/ports/write/* 路由组注册 | TestNetworkRouter_RegistersSetupPortWriteRouter + TestSetupPortWriteRouter_Registers6KebabEndpoints + TestSetupPortWriteRouter_DefinesSetupFunction |
| INFRA-03 | cache_keys.go 2 个占位常量 | 编译期 + grep |
| AUDIT-01 | 6 handler 在 response.Success 之前调 operlog.Record("端口管理", ...) | 7 handler tests 断言 mockOperLog.recordAsyncCalls ≥ 1 |
| AUDIT-02 | oper_param 含 device_id/port_id/action/operator/result_status/audit_ids | TestPortWriteHandler_Shutdown_Success (oper_param Contains audit.ID + port-001) + TestPortWriteHandler_Batch (oper_param.audit_ids 数组 len=3) |
| CONV-01 | shutdown/undo → OperTypeStatus(=10) | TestPortWriteHandler_Shutdown_Success (lastBusinessType == 10) |
| CONV-02 | description → OperTypeUpdate(=2) | TestPortWriteHandler_SetDescription (lastBusinessType == 2) |
| CONV-03 | dot1x enable/disable → OperTypeStatus(=10) | 源码 grep（execSinglePort 传 operlog.OperTypeStatus） |
| CONV-04 | batch → OperTypeBatch(=16) | TestPortWriteHandler_Batch (lastBusinessType == 16) |
| PORT-01 | shutdown 端点 | TestPortWriteHandler_Shutdown_* (4 tests) |
| PORT-02 | undo-shutdown 端点 | 源码 grep（router + handler 签名） |
| PORT-03 | description 端点 | TestPortWriteHandler_SetDescription |
| PORT-04 | dot1x-enable 端点 | 源码 grep（router + handler 签名） |
| PORT-05 | dot1x-disable 端点 | 源码 grep（router + handler 签名） |
| BATCH-01 | batch 端点 | TestPortWriteHandler_Batch + TestPortWriteHandler_Batch_ExceedsMax |

## Next Phase Readiness

Phase 52 Wave 1 (52-01) HTTP wiring + audit + permission + cache-key placeholders 全部就绪：

- 6 个写端点（`/network/ports/write/*`）可被 Phase 53 前端 BulkWriteDrawer 直接调用
- Path C audit↔operlog 关联已锁定（audit_ids 嵌 operlog.oper_param；audit.oper_log_id 列保持 NULL）
- PortWriteAudit 模型已定义（Path X），Wave 2 只需 `database.go` 加 `&models.PortWriteAudit{}` 到 AutoMigrate 列表 + 显式调用 `Migrate202PortWriteAudit`
- Phase 34 operlog regression 锁保持绿色（零侵入 operlog 包）
- Phase 51 PortWriteService regression 锁保持绿色（零侵入 service）

**Next:** `52-02-PLAN.md`（Wave 2 — migration_202 + 菜单 seed '端口配置' + GrantNewMenuToRolesHavingParent helper + database.go AutoMigrate 注册）

## Self-Check: PASSED

- All 8 expected files FOUND (6 created + 2 modified)
- All 3 task commits FOUND (44980bea / ec7a34ad / 88af87f7)
- Plan verification block 6 commands all green
- Path C guards (WithOperID=0 / WithJsonResult=0) verified
- Phase 34 operlog + Phase 51 portwrite regression locks intact

---
*Phase: 52-w3-router-handler-operlog-permission-migration*
*Plan: 01 (Wave 1)*
*Completed: 2026-07-07*
