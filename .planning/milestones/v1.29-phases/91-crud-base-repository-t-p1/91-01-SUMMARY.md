---
phase: 91-crud-base-repository-t-p1
plan: 01
subsystem: api
tags: [gorm, generic-repository, scope-based, type-alias, contract-tests, sqlite]

# Dependency graph
requires:
  - phase: 80-05 (base.GORMRepository 原始落地)
    provides: GORMRepository[T] 骨架 + WrapError/IsNotFound/IsDuplicate + sqlite 测试脚手架
  - phase: 89 (PAGINATION 常量集中化)
    provides: pkg/constants 分页常量（pagination_helper 既有 clamp 语义的底座）
provides:
  - base.Scope type alias（gorm 官方 scope 函数签名）+ PageParams struct
  - scope 函数式 GORMRepository[T] 六方法（Create/Update/Delete/GetByID/List/BatchDelete），纯 struct 无 interface
  - GetByID 变长 scopes + Tabler 断言表限定 id 过滤
  - BatchDelete 空 ids → nil（P1 语义反转）
  - SortScope（B 型排他默认）/ SortScopeWithTail（A 型复合尾随）排序 helper（复用 ApplySort 白名单）
  - base.PageResult 全项目唯一定义 + operations.PageResult type alias
  - TestBase91_* 六契约锁值测试（GORM 链语义回归防线）
affects: [91-02 workstation pilot, 91-03 building/floor/asset, 91-04 批量迁移收尾, 92 缓存层统一]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "scope 函数式泛型仓储：service 构造 Scope 闭包传 repo，repo 只承接管道不解释业务"
    - "type alias 合并双定义：type PageResult = base.PageResult 保 76 处引用零改动"
    - "Query 回调捕获 SQL 断言（Callback().Query().After(\"gorm:query\")）锁 GORM 链语义"

key-files:
  created:
    - internal/services/base/service_test.go
  modified:
    - internal/services/base/service.go
    - internal/services/base/base_80_05_test.go
    - internal/services/operations/pagination_helper.go

key-decisions:
  - "D-01/D-02 落地：删 Repository[T] interface 与 Query/WhereCondition DSL（零生产消费者），六方法 scope 化，service 直接组合 struct"
  - "GetByID 表限定采用「存在非 nil scope 即 Tabler 断言」判定；断言失败退回裸 id = ?"
  - "BatchDelete 空 ids 语义反转返回 nil（P1）：与 11 个 operations service 现状一致，双向锁值"
  - "PageResult 用 type alias 而非硬删除（F4）：floor_cache_impl.go（Phase 92 文件）与 76 处测试引用零改动"

patterns-established:
  - "排序双型 helper：B 型 SortScope（排他默认）+ A 型 SortScopeWithTail（复合尾随），ORDER BY 只能来自白名单 map value"
  - "契约锁值模式：泛型仓储的 GORM 链语义（Count 替换/恢复 Select、ORDER BY 剥离恢复、值切片）由 TestBase91_* 固化"

requirements-completed: [CRUD-REUSE-01, CRUD-REUSE-07]

# Metrics
duration: 15min
completed: 2026-09-04
---

# Phase 91 Plan 01: base.GORMRepository scope 化地基 Summary

**base.GORMRepository[T] 改造为 gorm scope 函数式六方法纯 struct 仓储（删 DSL/interface、BatchDelete 空 ids 反转 nil、双型排序 helper），PageResult 收敛为 base 单一定义 + operations alias，6 个 TestBase91_* 契约测试锁死 GORM 链语义**

## Performance

- **Duration:** 15 min
- **Started:** 2026-09-04T09:48:43Z
- **Completed:** 2026-09-04T10:03:48Z
- **Tasks:** 3/3
- **Files modified:** 4（1 新建 + 3 修改）

## Accomplishments

- base 包只剩 scope 函数式 GORMRepository[T]：`type Repository[` / `type Query struct` / `type WhereCondition struct` 计数归零（D-01/D-02），Create/Update/Delete/PageResult/WrapError/IsNotFound/IsDuplicate 逐字保留
- GetByID 支持变长 scopes：空 scope → `WHERE id = ?`；非空 scope → Tabler 断言 `<TableName()>.id = ?` 表限定（join 场景防 id 列 ambiguous，契约 5 行为锁证明）
- List 重写为 scope 管道：Model(new(T)) → scopes → Count → Offset/Limit/Find，PageResult.List 为值切片 []T（P4）；repo 不 clamp 分页、不加默认排序（P5 红线）
- BatchDelete 空 ids 从 BadRequest 反转为返回 nil（P1），nil 与 []string{} 双向锁值
- SortScope（B 型排他默认）+ SortScopeWithTail（A 型复合尾随）就位，底座复用 ApplySort/ResolveSort 白名单机制
- operations.PageResult 变为 base.PageResult 的 type alias（D-03/F4），operations 包 76 处测试引用零改动编译；6 个 extract/clamp/calculate helper 全部保留（F1）
- 6 个 TestBase91_* 契约测试全绿：值切片、Joins+Select 下 Count 正确且 Find 保留 Select 列、复合排序、软删过滤、表限定 GetByID、双型排序 SQL 断言（Query 回调捕获）

## Task Commits

1. **Task 1: 重写 base/service.go — scope 函数式六方法 + 双型排序 helper** - `d92097f` (refactor)
2. **Task 2: operations.PageResult 合并为 base.PageResult 的 type alias** - `5d0008b` (refactor)
3. **Task 3: 新建 base/service_test.go 泛型契约锁值 + 适配 base_80_05_test.go** - `76a262c` (test)

## Files Created/Modified

- `internal/services/base/service.go` - scope 函数式 GORMRepository[T]：Scope/PageParams/六方法 + SortScope/SortScopeWithTail；删 Repository[T]/Query/WhereCondition
- `internal/services/base/service_test.go` - 新建：TestBase91_* 六契约锁值（含双表 join 模型 base91Child + Query 回调 SQL 捕获脚手架）
- `internal/services/base/base_80_05_test.go` - 适配：TestBas8005_List_Paged 的 Query DSL 用例改写为等价 scope 调用；TestBas8005_BatchDelete 空切片期望反转为 require.NoError；脚手架与 ErrorHelpers 逐字保留
- `internal/services/operations/pagination_helper.go` - PageResult struct 定义替换为 `type PageResult = base.PageResult` alias

## Decisions Made

- **GetByID「scopes 非空」判定为「存在非 nil scope」**：nil scope 不产生 join，无需表限定；语义上比 `len(scopes) > 0` 更精确（与 interfaces 注释「依序应用非 nil scopes」一致）
- **契约测试 SQL 断言用 Query 回调捕获（真实库）而非 DryRun Session**：plan 允许「经 stmt.SQL.String() 或回调捕获断言」；真实库 + After("gorm:query") 回调对 Count/Find/First 全部生效且无 DryRun 链复用歧义
- **契约 2 的「LEFT JOIN 自表聚合列」落为 1:1 LEFT JOIN + 别名 Select 断言**：Count==2 证明自定义 Select 被 count(*) 临时替换且 join 不增殖行；子表 name AS name 覆盖父表列证明 Find 保留 Select
- **Task 3 tdd="true" 执行说明**：本 plan 为 `type: execute`，实现已在 Task 1 按排序落地，Task 3 是契约锁值套件（测试即终态验证），无 RED→GREEN 分相 commit；以 `test(91-01)` 单 commit 收口

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- commitlint hook 拒绝超 100 字符的 commit body 行 — 缩短行宽后重试通过（首次 Task 1 commit 尝试，无代码影响）
- 复合验证命令中 `grep -c` 对 SortScope 签名误报 0（shell 转义问题），单独复查确认签名在 service.go:159/175 — 非代码问题

## Verification Results

- `go build ./...` 退出码 0（CLAUDE.md 强制，每 task 收尾均执行）
- `go test ./internal/services/base/...` 全绿（6 TestBase91 + 4 TestBas8005 + 既有 list_request_test）
- `go test ./internal/services/operations/...` 全绿（alias 改动零回归）
- `go vet ./internal/services/base/` + `go vet ./internal/services/operations/` 退出码 0
- AST 锁值防线全绿：`go test ./internal/models/ ./internal/utils/operlog/ ./pkg/constants/`
- LOC（本 plan 基建净增 +308 行，含 275 行契约测试；净减 ≥800 目标由 91-02/03/04 服务迁移兑现）

## User Setup Required

None - 纯内部重构，无外部服务配置。

## Next Phase Readiness

- 91-02（workstation pilot）可直接消费：`base.Scope`/`base.PageParams`/`base.SortScope`/`base.SortScopeWithTail`/`*base.GORMRepository[T]` 签名已按 plan interfaces 节定稿并锁值
- TestBase91_* 契约测试是后续三个 plan「行为零变更」的基准防线：任何 repo 语义漂移立即失败
- 无阻塞项；A2/A3 两个行为语义 checkpoint 已按 plan 分配到 91-02 Task 3 与 91-03 Task 4

## Self-Check: PASSED

- 4 个任务文件 + SUMMARY.md 全部存在
- 4 个 commit（d92097f / 5d0008b / 76a262c / 6737ae6）全部在 git log 中
- 终验：`go build ./...` 0 错误；`go test ./internal/services/base/... ./internal/services/operations/...` 0 失败

---
*Phase: 91-crud-base-repository-t-p1*
*Completed: 2026-09-04*
