---
phase: 91-crud-base-repository-t-p1
reviewed: 2026-09-04T00:00:00Z
depth: standard
files_reviewed: 26
files_reviewed_list:
  - internal/api/v1/operations/requests/workstation_requests.go
  - internal/api/v1/operations/workstation_handler.go
  - internal/api/v1/operations/workstation_handler_full_test.go
  - internal/api/v1/operations/workstation_handler_test.go
  - internal/services/base/base_80_05_test.go
  - internal/services/base/service.go
  - internal/services/base/service_test.go
  - internal/services/operations/asset_building_roomdevice_test.go
  - internal/services/operations/asset_listfilter_test.go
  - internal/services/operations/asset_service.go
  - internal/services/operations/building_service.go
  - internal/services/operations/dedicated_line_service.go
  - internal/services/operations/door_service.go
  - internal/services/operations/floor_plan_text_crud_test.go
  - internal/services/operations/floor_plan_text_service.go
  - internal/services/operations/floor_service.go
  - internal/services/operations/floor_service_crud_test.go
  - internal/services/operations/infopoint_service.go
  - internal/services/operations/pagination_helper.go
  - internal/services/operations/pagination_helper_test.go
  - internal/services/operations/room_device_service.go
  - internal/services/operations/server_room_service.go
  - internal/services/operations/wall_service.go
  - internal/services/operations/workstation_floor_code_77_03_test.go
  - internal/services/operations/workstation_service.go
  - internal/services/operations/workstation_update_createdat_test.go
findings:
  critical: 0
  warning: 2
  info: 6
  total: 8
status: issues_found
---

# Phase 91: Code Review Report

**Reviewed:** 2026-09-04
**Depth:** standard
**Files Reviewed:** 26
**Status:** issues_found

## Summary

Phase 91 是纯重构：11 个 operations CRUD service 迁移到重建后的 scope 函数式 `base.GORMRepository[T]`。本次审查将 26 个变更文件逐一与 `0ab266e` 基线 diff 对照，重点核验：scope 组合顺序（P8）、软删语义一致性、分页边界、`PageResult` type alias 合并、测试改写保真度。

**核心结论：未发现 BLOCKER 级问题。** 行为保持不变量的关键论证经独立核实成立：

1. **GORM 链语义根基已实证**：直接读取 module cache 中 `gorm.io/gorm@v1.30.5/finisher_api.go` 的 `Count` 实现确认——SELECT 子句临时替换为 count(*) 并 defer 恢复、ORDER BY 在无 GROUP BY 时剥离并在返回后恢复、复杂多列 Select（含逗号）落入 `count(*)` 分支。这与 `service.go` 注释声明及 `TestBase91` 契约测试一致。"join scope 前置于 Count 不改 Total"（全部为按主键 N:1 LEFT JOIN 无行增殖）的论证成立。
2. **scope 顺序逐字核对**：server_room（joinScope → filterScope → orgId → Sort，P8 顺序敏感）、room_device（joinScope 前置供 orgId 引用 b.org_id）、workstation/floor/infopoint/asset/building 的 filter/join/sort 相对顺序均与迁移前等价（GORM Select/Order 为独立 clause，应用次序不影响最终 SQL）。
3. **排序 A/B 型语义逐字核对**：door/wall 迁移前 `ApplySort + Order("") no-op + fetchRecords 恒追加 created_at DESC` = `SortScopeWithTail` 复合尾随；其余 9 服务迁移前 `ApplySort + if empty Order(default)` = `SortScope` 排他默认。三态（命中/空/非法）逐分支比对一致。
4. **两处 checkpoint 批准的行为变更按描述落地**：workstation typed `GetPagination`（pageSize 上限 10000→100、current<1 回退 1）；building/asset 由 `.Table()` 起链改 `Model(new(T))` 起链后 List Total 收紧为不含软删行。均与 review context 声明一致，不作为缺陷计。
5. **floor Update 死代码修复正确**：`oldBuildingID` 改从库中读取修改前值，`TestFloor91_Update_SyncsWorkstation` 以 Eventually 锁定异步同步（符合项目异步测试规约）。
6. **`PageResult = base.PageResult` type alias**：JSON 契约（list/total/current/pageSize tag）逐字节相同，`TestBase91_List_ValueSlice` 锁定 `[]T` 值切片形态。
7. **floor_cache_impl.go 白盒构造**（`&floorService{db: db}`，repo 为 nil）由 `listRepo()` 惰性兜底，且惰性构造不写回字段、无数据竞争；全仓扫描确认其余 10 个 service 无同类白盒生产构造。
8. **编译与测试全绿**：`go build ./...`、`go test ./internal/services/base/ ./internal/api/v1/operations/ ./internal/services/operations/` 全部通过。被删除的 typesafe handler/service 在基线 commit 中即无 router 注册（死代码删除安全），相关测试同步删除。

以下 2 个 WARNING 均为**迁移前已存在的潜在缺陷**（本次重构按零行为变更底线原样保留），非 Phase 91 引入的回归；建议作为后续硬化项跟进。6 个 Info 为死代码与测试质量问题。

## Warnings

### WR-01: 8 个 operations service 的 Update 仍会把 created_at/created_by 清零（created_at 归零 bug 类未收敛）

**File:** `internal/services/operations/floor_service.go:161`、`server_room_service.go:94`、`dedicated_line_service.go:116`、`room_device_service.go:101`、`infopoint_service.go:95`、`door_service.go:57`、`wall_service.go:50`、`floor_plan_text_service.go:55`
**Issue:** 这 8 个 handler 均为 fresh-bind（`var x operationsmodels.Xxx; handleJSONBinding(c, &x); x.ID = id`）后直调 service.Update → `repo.Update`（即 GORM `Save`，全量 Select "*" 更新）。`BaseModel.CreatedAt` 是非指针 `time.Time`（见 `internal/models/base.go:13`），零值 0001-01-01 会全量覆盖库中原值——这正是项目已确认并修复过的 workstation bug 类（`.planning/debug/workstation-update-createdat-zeroed.md` + 回归测试 `workstation_update_createdat_test.go`）。目前仅 3/11 受保护：workstation（First 回填 CreatedAt/CreatedBy，`workstation_service.go:197-209`）、asset/building（`Omit("CreatedAt")`，`asset_service.go:146`、`building_service.go:113`）。客户端 payload 不回传 createdAt 时（workstation 当年即此场景）数据即被清零。
**说明:** 迁移前这些方法同样是裸 `Save`，Phase 91 按行为保持原则未扩大修复面，不构成回归；但同族缺陷跨 8 服务存续且修复范式已有现成模板，建议在 Phase 92+ 统一收敛。
**Fix:** 二选一，与 workstation/asset 现有范式对齐：
```go
// 方案 A（回填式，workstation_service.go:197 同款）：Update 前 First 查 existing，
// 回填 CreatedAt/CreatedBy 再 repo.Update。
// 方案 B（Omit 式，asset_service.go:146 同款）：
//   repo.Update 无法携带 Omit → 改为 s.db.WithContext(ctx).Omit("CreatedAt").Save(entity)
```

### WR-02: floor GetTree 的 LEFT JOIN 未过滤软删楼层，已删楼层会在楼层树中复活

**File:** `internal/services/operations/floor_service.go:299-310`
**Issue:** `GetTree` 原生 SQL 仅 `WHERE b.deleted_at IS NULL` 过滤楼宇，`LEFT JOIN ops_floors f ON b.id = f.building_id` 未加 `f.deleted_at IS NULL`，`FloorID != ""` 即追加 children——软删除的楼层仍出现在树中。与同文件 `updateBuildingFloorCount`（`:395-401`，经 Model 计数已排除软删）及 `List`（repo 化后软删行已过滤）口径不一致。迁移前行为即如此，非本次引入。
**Fix:**
```sql
FROM ops_buildings b
LEFT JOIN ops_floors f
       ON b.id = f.building_id
      AND f.deleted_at IS NULL   -- 新增
WHERE b.deleted_at IS NULL
```

## Info

### IN-01: typesafe 楼宇服务删除后遗留孤儿请求结构体

**File:** `internal/api/v1/operations/requests/building_requests.go:4-14`
**Issue:** 本阶段删除了 `BuildingServiceTypeSafe`/`BuildingHandlerTypeSafe` 及其测试后，`BuildingListRequest`（含内嵌 `BuildingBatchOperationRequest`）在全仓已无任何生产/测试消费者，成为死代码。
**Fix:** 删除 `building_requests.go` 中 `BuildingListRequest` 与 `BuildingBatchOperationRequest`（先全仓 grep 复核）。

### IN-02: WorkstationBatchOperationRequest 为既有死代码

**File:** `internal/api/v1/operations/requests/workstation_requests.go:24-26`
**Issue:** handler 的 BatchOperation 用的是函数内匿名 struct（`workstation_handler.go:286-289`），`WorkstationBatchOperationRequest` 无任何消费者（迁移前即如此，本阶段改写时保留）。
**Fix:** 删除该结构体，或在批量端点改用它以消除双定义。

### IN-03: workstation typed bind 后，单个字段类型畸形会连带丢弃全部过滤条件

**File:** `internal/api/v1/operations/workstation_handler.go:147-150`
**Issue:** D-05 typed 化后，JSON 中 `status`/`type` 若为非整数值（如 `"1"` 字符串、`1.5` 浮点），`ShouldBindJSON` 整体报错 → 走零值请求降级 → 全部过滤条件静默失效。迁移前 map 路径 `extractIntParam` 逐字段容错（畸形字段回退默认值，其余过滤仍生效）。仅影响畸形输入的健壮性，合法 payload 无差异。
**Fix:** 若需完全等价可改用 `StatusRequest{Status *int}` 之外的 json.Number/自定义 Unmarshal；或接受现状并在接口文档注明数值字段必须为整数。

### IN-04: handler 全量测试文件存在 import 压制 hack 与冗余双重 core 构造

**File:** `internal/api/v1/operations/workstation_handler_full_test.go:112, 491-492`
**Issue:** 文件尾 `var _ = strings.HasPrefix` / `var _ = opsModels.OpsBuilding{}` 为压 unused import 的 hack（应直接删 import）；`newWorkstationHandler` 内部已 `WithCore(newTestCore(&testing.T{}))`，各测试又再次 `.WithCore(newTestCore(t))`，每次用例重复构造两份 core（后值覆盖，功能无碍但浪费且 `&testing.T{}` 绕过测试生命周期）。
**Fix:** 删除文件尾两行压制 hack 及对应未用 import；`newWorkstationHandler` 去掉内部 WithCore，仅保留调用侧注入。

### IN-05: dept 子树 Pluck 的 WithContext 使用不一致

**File:** `internal/services/operations/asset_service.go:270`、`building_service.go:187`
**Issue:** `applyDeptFilter` 的 `s.db.Table("sys_dept")...Pluck` 未带 `WithContext(ctx)`，而 serverRoom 同型 helper `getDeptAndChildDeptIDs`（`server_room_service.go:236`）带了。缺 WithContext 的子查询不随请求 ctx 取消/追踪（迁移前即如此，非本次引入）。
**Fix:** `s.db.WithContext(ctx).Table("sys_dept")...`（applyDeptFilter 需增加 ctx 参数或从闭包捕获）。

### IN-06: BatchUpdatePositions 中 `_ = v` 死赋值与 valuer 重复求值

**File:** `internal/services/operations/workstation_service.go:374-376`
**Issue:** `appendOptionalClause` 第一趟循环 `if v, ok := valuer(item); ok { _ = v; ... }` 中 `v` 未使用（死赋值），第二趟 args 循环再次调用 valuer 重复求值。功能正确（valuer 为纯函数），可读性欠佳（既有代码，随文件纳入审查范围）。
**Fix:** 第一趟改 `if _, ok := valuer(item); ok`；或一趟循环同时收集 WHEN 子句与 args。

---

_Reviewed: 2026-09-04_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
