---
phase: 99
phase_name: operations口径统一
plan: 99-06
subsystem: operations + constants
tags: [refactor, gap-closure, orgId-filter, constants-merge, V130R-08, V130R-09]
dependency_graph:
  requires: [dept_filter.go helper (99-03), pkg/query.NormalizePagination authority (99-04)]
  provides: [single-authority constants package (pkg/constants), converged BuildDeptRecursiveFilter call sites]
  affects: [workstation_service, infopoint_service, building_service, asset_service, server_room_service, pkg/constants, base/service.go]
tech_stack:
  added: []
  patterns:
    - gorm sub-DB via Session(NewDB) + EXISTS (?) arg for scope-in-subquery composition
    - CAST(... AS TEXT) dual-dialect uuid/varchar comparison inside shared helper
    - single leaf-const package (pkg/constants) with AST pinning tests
key_files:
  created:
    - pkg/constants/cache.go
    - pkg/constants/time.go
    - pkg/constants/scheduler.go
    - pkg/constants/uuid.go
    - pkg/constants/doc.go
    - pkg/constants/example_test.go
  modified:
    - internal/services/operations/dept_filter.go
    - internal/services/operations/workstation_service.go
    - internal/services/operations/infopoint_service.go
    - internal/services/operations/building_service.go
    - internal/services/operations/asset_service.go
    - internal/services/operations/server_room_service.go
    - pkg/constants/pagination.go
    - pkg/constants/pagination_test.go
    - internal/services/base/service.go
    - CLAUDE.md
    - ".planning/phases/99-operations口径统一/99-CONTEXT.md"
  deleted:
    - internal/constants/ (7 files)
decisions:
  - id: D-03-10
    decision: internal/constants merged wholesale into pkg/constants; DefaultCurrent/DefaultPageSize keep pkg definitions (same values 1/10, no same-name-different-value collision found)
metrics:
  duration_minutes: 40
  completed_date: 2026-09-07
  tasks_completed: 3
  files_changed: 41
---

# Phase 99 Plan 99-06: V130R-08/09 缺口收口 Summary

**BuildDeptRecursiveFilter 收敛 9 处调用（workstation×4/infopoint/building×2/asset/server_room）+ internal/constants 整体并入 pkg/constants（D-03-10）+ base/service.go 过时注释清零**

## What Was Built

### Task 1: 8+1 处内联 orgId 筛选收敛到 BuildDeptRecursiveFilter（V130R-08 收敛断链关闭）

- **workstation ×4**：Statistics / filterScope / SearchWorkstationOptions 的三处同型
  `EXISTS (... JOIN sys_dept d ... 四条件 ...)` 重建为 gorm 子查询——
  `workstationOrgExistsSub(db, link)`（Session(NewDB) 派生全新 statement，不污染外层链）
  + `Where("EXISTS (?)", BuildDeptRecursiveFilter(orgID, "b.org_id")(sub))`；
  原 `JOIN sys_dept d` 由 helper 的 IN 子查询承接后删除。
  GetWorkstationDeptOptions 的 UNION ALL 左支由 Raw 拼接条件改为
  `Table("sys_dept").Scopes(helper(orgId,"id"))` 子 DB，经 `(?)` 实参内联进 Raw
  （GORM 渲染子查询并合并绑定参数），alias 右支与 location_id 参数不变。
- **infopoint ×1**：filterScope 的 w→f→b 链 EXISTS 同款迁移；w/f/b 软删过滤逐字保留。
- **building ×2 / asset ×1 / server_room ×1**：`sys_dept` Pluck 站点改为
  `Table("sys_dept").Scopes(BuildDeptRecursiveFilter(deptID, "id"))`。
- **helper 加固（Rule 1）**：`sys_dept.id` 是 uuid 而 `ops_*` org 列是 varchar，
  PG 无隐式互转（42883）——helper 的 idColumn 与子查询输出统一 `CAST(... AS TEXT)`
  （sqlite 下 no-op，99-03 的 6 个回归测试原样保持绿）。
- **server_room 第 9 处为 verifier 扫描遗漏**（99-VERIFICATION 只列 8 处），
  计划自身的 grep 验收（operations *_service.go 中 `ancestors LIKE` 清零）要求一并迁移。

### Task 2: internal/constants 合并进 pkg/constants（V130R-09 双包合并落地，D-03-10）

- **冲突清点**：两包逐名比对，唯一交集 `DefaultCurrent`/`DefaultPageSize`（同值 1/10），
  保留 pkg 既有定义；**无同名异值**（time.go 17 个时长常量 vs timeouts.go 10 个、
  cache.go 3 个键格式、scheduler.go 1 个、uuid.go 1 个 var，均无冲突）。
- **迁移**：新建 pkg/constants/{cache,time,scheduler,uuid,doc,example_test}.go；
  MinPageSize/MaxListPageSize/MaxOptionsPageSize 并入 pkg pagination.go
  （常量 3→6，锁值/计数测试按「先改测试再改常量」纪律同步更新）。
- **import 重写**：29 个 Go 文件改写为 pkg/constants 路径；ad_sync_tasks.go 双包
  同 import 去重（保留 pkgconstants 别名，internal import 删除）。
- **删除** internal/constants/ 全部 7 文件；**CLAUDE.md** 分页惯例段落同步 6 常量 +
  追加合并注记；**99-CONTEXT.md** 追加 D-03-10 决策 + Constants canonical refs 更新。

### Task 3: base/service.go 过时注释处置

- PageParams doc 不再声称「三种 clamp 并存」，改为 service 层统一经
  `pkg/query.NormalizePagination / NormalizePaginationWithMax` 归一化（V130R-09）；
  List 设计红线（repo 直信 PageParams、不 clamp）原样保留——设计未变。
- 纯注释变更，`grep 三种 clamp` base/ 清零。

## Verification

- `go build ./...` 0 错误
- `go test ./internal/services/operations/... ./pkg/constants/... ./internal/services/... ./internal/models/...` 26/26 包 ok、0 FAIL
- `BuildDeptRecursiveFilter(` 生产引用 11 处 ≥ 8（9 个真实调用站点 + 注释引用）
- `grep -rn "xingran-go-backend/internal/constants" --include="*.go"` 0 命中
- `ancestors LIKE` 仅存于 dept_filter.go（helper 实现）与 dept_filter_test.go（测试）
- 附带运行：`go test ./internal/api/v1/operations/...`（handler 层）、`./internal/utils/...`、
  `./internal/scheduler/...` 全绿

## Deviations from Plan

**[Rule 1 - Bug] helper 的 IN 子查询在 PG 会因 uuid/varchar 无隐式互转报 42883**
- **Found during:** Task 1 站点核对（sys_dept.id 为 `type:uuid`，ops_buildings.org_id 为 varchar(64)）
- **Fix:** helper 生成 SQL 对 idColumn 与子查询输出统一 `CAST(... AS TEXT)`；
  sqlite no-op，既有 6 个回归测试零改动保持绿
- **Files:** internal/services/operations/dept_filter.go
- **Commit:** c1b35be

**[Rule 1 - 扫描遗漏] server_room_service.go 第 9 处内联实现**
- **Found during:** Task 1 就近重定位（grep `ancestors LIKE` 命中 9 个站点，verifier 仅列 8）
- **Fix:** getDeptAndChildDeptIDs 一并迁移到 helper（计划验收 grep 本身要求清零）
- **Files:** internal/services/operations/server_room_service.go
- **Commit:** c1b35be

**[计划修正] `.Where(scopeFunc)` 不可行**
- gorm v1.30.5 `BuildCondition` 不接受 `func(*gorm.DB)*gorm.DB`（计划原文称「可直接作
  Where 实参」）——改用 gorm 一等公民形态 `.Scopes(...)`（base.Scope 文档即此语义）；
  EXISTS 站点以 `Where("EXISTS (?)", helper(...)(sub))` 直套 scope。

## Known Stubs

None — 本计划为纯收敛重构，无占位实现。

## Self-Check: PASSED

- 9 个迁移站点 + helper CAST 加固：commit c1b35be（6 files, +79/-59）
- constants 合并 41 文件：commit 3cd5b25
- base/service.go 注释：commit 83782a5
- internal/constants/ 已不存在（ls 确认）；全仓 import 路径零残留（grep 确认）
