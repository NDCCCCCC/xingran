---
phase: 39-workstation-dept-location-alias
plan: 04
subsystem: backend-api-router
tags: [router, location-alias, cache-invalidation, operlog, permissions]
requires:
  - 39-02-PLAN (LocationAliasHandler + LocationAliasService 产出)
  - 39-01-PLAN (sys_menu perms seed: ops:location:alias:list/add/edit/delete)
provides:
  - "POST /api/v1/ops/location-alias/list 受 ops:location:alias:list 保护"
  - "POST /api/v1/ops/location-alias 受 ops:location:alias:add 保护"
  - "POST /api/v1/ops/location-alias/:id/update 受 ops:location:alias:edit 保护"
  - "POST /api/v1/ops/location-alias/:id/delete 受 ops:location:alias:delete 保护"
  - "alias 写操作副作用 → DepartmentService.InvalidateDeptCache (D-03 决策)"
affects:
  - internal/api/router.go (插入 location-alias 路由组)
  - internal/api/v1/operations/location_alias_handler.go (注入 DepartmentService + 缓存失效)
tech-stack:
  added: []
  patterns:
    - "链式 WithXxx setter 依赖注入 (与 workstation_handler.go 风格一致)"
    - "路由组级 RequirePermissions middleware (4 perms 严格, 无 RequirePermissionsWithQuery 放宽)"
    - "缓存失效失败仅 warn 不阻断响应 (与 Phase 28 cache_invalidator.go 一致)"
key-files:
  created: []
  modified:
    - internal/api/router.go
    - internal/api/v1/operations/location_alias_handler.go
decisions:
  - "D-03 落地:alias 写操作触发 DepartmentService.InvalidateDeptCache(ctx),失败仅 warn"
  - "DeptService 通过链式 WithDeptCacheInvalidator 注入而非 core 字段(core 不直接持有 DepartmentService)"
metrics:
  duration: PT15M
  completed: 2026-06-25
---

# Phase 39 Plan 04: location-alias 路由注册 + dept 缓存失效 Summary

注册 4 个 alias CRUD 端点到 `/api/v1/ops/location-alias`,挂 4 perms 严格权限中间件,并在 3 个写方法中触发 `DepartmentService.InvalidateDeptCache` (D-03 决策),完成 Phase 39 后端 API 层。

## What Was Built

### Task 1: SetupLocationAliasRouter 路由注册块 (commit a5b9184)

在 `internal/api/router.go` 中 `workstations` 路由组之后、`workstationDevices` 之前,插入 `locationAlias` 路由组:

- `locationAlias.Use(middleware.RequirePermissions([4 perms], core))` — 4 perms 严格,无 `RequirePermissionsWithQuery` 放宽(alias 没有"只读路径放宽"诉求)
- 路由顺序严格:`/list` `/:id/update` `/:id/delete` 在前,`""` 在最后兜底(与 workstation 路由风格一致,避免 `:id` 兜底误匹配)
- service 注入 `core.DB.GetDB()`,handler 链式 `WithCore(core)`(operlog 所需)
- 不修改 ops group 外任何代码,不修改其他路由组

### Task 2: dept 缓存失效 (commit 973ebdb)

`internal/api/v1/operations/location_alias_handler.go` 中:

- 新增字段 `deptCacheInvalidator systemServices.DepartmentService` + 链式 setter `WithDeptCacheInvalidator(deptSvc)`
- 抽出辅助方法 `invalidateDeptCache(c)` — nil 守护 + `Warnf` 不阻断响应
- Create/Update/Delete 三方法在 service 成功后、`operlog.Record` 之前调用 `h.invalidateDeptCache(c)`
- router.go 现场构造 `DepartmentService`(`NewDepartmentServiceWithCache` / `NewDepartmentService` 二选一,与 `department_router.go` 模式完全一致)
- List 读方法不触发失效(无副作用需求)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `core.DeptService` 路径不存在**

- **Found during:** Task 2
- **Issue:** 计划原文给出 `h.core.DeptService.InvalidateDeptCache(ctx)`,但 `core.Core` 结构体并不直接持有 `DepartmentService` 字段(该字段位于私有 `warmUpServices` 结构,仅初始化期间用于缓存预热)。直接按计划写法会编译失败:`h.core.DeptService undefined (type *core.Core has no field or method DeptService)`
- **Fix:** 改为链式依赖注入 — `LocationAliasHandler` 新增 `deptCacheInvalidator` 字段 + `WithDeptCacheInvalidator` setter;`router.go` 现场构造 `DepartmentService`(沿用 `department_router.go` 的 `NewCacheProvider` + `NewDepartmentServiceWithCache`/`NewDepartmentService` 二选一模式)并注入 handler。三处 `h.core.DeptService.InvalidateDeptCache(...)` 调用统一收敛到 `h.invalidateDeptCache(c)` 辅助方法,nil 守护 + warn 不阻断。
- **Files modified:** `internal/api/v1/operations/location_alias_handler.go`、`internal/api/router.go`
- **Commit:** 973ebdb
- **Rationale:** 这是项目既有惯例(`DepartmentHandler` 也是路由组现场构造 + 链式 setter),并非新架构。计划中 `core.DeptService` 引用是基于 `warmUpServices` 字段的误读 — 该结构体在 `core.go:48-54` 是私有 init-only,核心 `Core` 通过 `*CoreInfra` + `*CoreServices` 嵌入暴露的只有 `DataCacheService / CacheConfigService` 等通用层,不含领域 service。

None other — 计划其余部分(路由顺序、4 perms 字面量、operlog 模块名"工位管理"、3 OperType 常量引用、缓存失效失败仅 warn)按原文执行。

## Verification

- `go build ./...` 退出码 0
- `go vet ./internal/api/... ./internal/api/v1/operations/... ./internal/utils/operlog/...` 0 警告
- `go test ./internal/utils/operlog/ -run "TestOperTypeCountEquals25|TestOperTypeConstantStability" -v` 全绿(25 OperType 常量值稳定)
- `grep "ops:location:alias" internal/api/router.go` 显示 4 条权限字符串字面量
- `grep "OperTypeCreate\|OperTypeUpdate\|OperTypeDelete" internal/api/v1/operations/location_alias_handler.go` 显示 3 处引用
- `grep "invalidateDeptCache" internal/api/v1/operations/location_alias_handler.go` 显示 4 处(1 定义 + 3 调用)

## Known Stubs

None — handler 全部 4 方法有真实业务数据落地(service 层 CRUD 在 Plan 39-02 实现),无 mock/placeholder。

## Threat Flags

None — 本计划仅注册路由 + 复用既有 DepartmentService 缓存失效路径,未引入新的网络端点形态、auth 路径、文件访问模式或信任边界 schema 变更。4 perms 已在 Plan 39-01 `migration_165` seed 入库。

## Self-Check: PASSED

- FOUND: internal/api/router.go(包含 `locationAlias := ops.Group("/location-alias")`)
- FOUND: internal/api/v1/operations/location_alias_handler.go(包含 4 handler 方法 + invalidateDeptCache 辅助方法)
- FOUND: commit a5b9184 (feat(39-04): 注册 location-alias 4 端点)
- FOUND: commit 973ebdb (feat(39-04): alias 写操作触发 dept 缓存失效)
