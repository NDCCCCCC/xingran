---
phase: 99
phase_name: operations 口径统一
phase_slug: 99-operations口径统一
milestone: v1.30
status: planned
planned_at: 2026-09-06
requirements:
  - V130R-06
  - V130R-07
  - V130R-08
  - V130R-09
plans:
  - plan_id: 99-01
    title: V130R-06 Total 口径收紧（软删过滤）
    type: fix
    gap_closure: false
  - plan_id: 99-02
    title: V130R-07 换楼同步有序化
    type: fix
    gap_closure: false
  - plan_id: 99-03
    title: V130R-08 orgId 子部门筛选共享 helper
    type: refactor
    gap_closure: false
  - plan_id: 99-04
    title: V130R-09 分页 clamp 单一权威收敛（Go 口径核心）
    type: refactor
    gap_closure: false
  - plan_id: 99-05
    title: V130R-09 CAD 全集专用端点 + 前端消费者迁移
    type: feature
    gap_closure: false
success_criteria:
  - V130R-06: asset/building List Total 不再计入软删记录
  - V130R-07: floor 换楼同步不再乱序，First 失败不再被静默吞掉
  - V130R-08: orgId 子部门筛选四条件口径统一，workstation/infopoint 三条件漏匹配修复
  - V130R-09: 分页 clamp 三口径收敛为单一权威路径（discuss 后解锁）
  - go build ./... && go test ./... 0 failure
  - 七 gate 不倒退
---

# Phase 99 Plan: operations 口径统一

## Context

Phase 99 修复 operations 域四处口径/语义缺陷，涉及 building/asset/floor/workstation/infopoint 五个 service 文件。部分依赖 Phase 91 的 `base.GORMRepository[T]` 和 Phase 89 的 `pkg/constants` 基线。

### V130R-06 根因（Total 软删虚高）

`building_service.go:231` 用 `.Table("ops_buildings")` 起链导致 Count 查询不过滤软删除行（Total 虚高）。`asset_service` 有同类问题。

修复路径：repo 化后统一用 `db.Model(new(T))` 或 `s.repo.List(...).Count()` 口径。

### V130R-07 根因（换楼乱序）

`floor_service.go` 换楼同步（`buildingId` 变更）时，更新语句顺序依赖数据库执行顺序。若前一条更新触发子查询（工位数统计），后一条 `UPDATE` 可能因外键约束或时序问题乱序或失败。

### V130R-08 根因（orgId 筛选不一致）

`building_service`、`workstation_service`、`infopoint_service` 各自实现「部门+全部子部门」筛选，逻辑相同但代码重复。`workstation`/`infopoint` 使用三条件形式（`id = X OR ancestors LIKE X%`），漏匹配 ancestors 中段部门。

### V130R-09 根因（分页 clamp 三口径）

| 路径 | 来源 | 上限 |
|------|------|------|
| `operations/pagination_helper.go` `clampPageSize` | `internal/constants.MaxOptionsPageSize` | 10000 |
| `pkg/query.NormalizePagination` | `pkg/constants.MaxPageSize` | 200 |
| 多个 service 内联 clamp | 硬编码 | 100 |

D-03 设计决策项已经 `/gsd:discuss-phase 99` 敲定（方案 B 单一权威出口），详见 `99-CONTEXT.md`。

---

## Plan 99-01: V130R-06 Total 口径收紧

### Steps

**Step 1: 审查 building_service.go List Total 口径（10 min）**

已有注释（`building_service.go:132`）说明问题：`Table("ops_buildings")` 起链导致 Count 不过滤软删除行。

**Step 2: 审查 asset_service 同类问题（10 min）**

**Step 3: 修复 building_service（15 min）**

```go
// BEFORE（软删漏过滤）
query := s.db.WithContext(ctx).Table("ops_buildings").Where(...).Count(&total)

// AFTER（统一 Model 起链，gorm 自动附上 deleted_at IS NULL scope）
query := s.db.WithContext(ctx).Model(&models.Building{}).Where(...).Count(&total)
```

**Step 4: 修复 asset_service（15 min）**

**Step 5: 写回归测试（10 min）**

创建 `internal/services/operations/building_total_softdelete_test.go`：
- 软删 3 条记录，List Total 应返回 0

### Files to Modify
- `internal/services/operations/building_service.go`
- `internal/services/operations/asset_service.go`（确认）
- 新增测试文件

### Verification
```bash
go test -v -run TestBuilding.*Total ./internal/services/operations/
go build ./...
```

---

## Plan 99-02: V130R-07 换楼同步有序化

### Steps

**Step 1: 审查 floor_service.go 换楼逻辑（15 min）**

查找 `UpdateFloor` 或 `MoveFloor` 中 `building_id` 变更处理。

**Step 2: 确认乱序场景（10 min）**

换楼涉及：
1. 更新 floor.building_id
2. 清除旧楼工位关联
3. 建立新楼工位关联

若步骤 2 的 `WHERE floor_id = ?` 条件中 `floor_id` 是旧值，可能导致乱序。

**Step 3: 修复方案——乐观条件或事务串行化（20 min）**

方案 A（推荐）：在 UPDATE 语句中直接使用 `WHERE id = ? AND building_id = ?` 乐观条件，确保只有楼层确实在旧楼时才更新。

**Step 4: 写回归测试（10 min）**

### Files to Modify
- `internal/services/operations/floor_service.go`
- 新增测试文件

### Verification
```bash
go test -v -run TestFloor.*Move ./internal/services/operations/
go build ./...
```

---

## Plan 99-03: V130R-08 orgId 子部门筛选共享 helper

### Steps

**Step 1: 审查四服务 orgId 筛选实现（20 min）**

确认以下位置：
- `building_service.go` — `applyDeptFilter`
- `workstation_service.go` — orgId 筛选
- `infopoint_service.go` — orgId 筛选
- 其他使用 `RecursiveDeptID` 的位置

**Step 2: 创建共享 helper（20 min）**

在 `internal/services/operations/` 创建 `dept_filter.go`：

```go
// BuildDeptRecursiveFilter 返回"部门+所有子部门"筛选条件 scope
// deptID: 根部门 ID
// idColumn: 表的 dept_id 列名（支持 "d.dept_id" 形式）
func BuildDeptRecursiveFilter(deptID string, idColumn string) base.Scope {
    return func(db *gorm.DB) *gorm.DB {
        return db.Where(
            db.Where(idColumn+" = ?", deptID).
                Or(idColumn+" IN (SELECT dept_id FROM sys_dept WHERE ancestors LIKE ?)", "%"+deptID+"%"),
        )
    }
}
```

**Step 3: 替换四服务中的重复实现（15 min）**

**Step 4: 验证 workstation/infopoint 三条件漏匹配修复（10 min）**

原有三条件形式：
```sql
(id = ? OR ancestors LIKE ?%)
```
当 `deptId = "d2"`，ancestors 存 `d1/d2/d3` 时，`d2` 在中间位置，`LIKE '%d2%'` 匹配但 `%d2%` 不匹配起点。修复后：
```sql
(id = ? OR ancestors LIKE ? OR ancestors LIKE ?)
```

### Files to Modify
- `internal/services/operations/dept_filter.go`（新建）
- `internal/services/operations/building_service.go`
- `internal/services/operations/workstation_service.go`
- `internal/services/operations/infopoint_service.go`
- 新增测试文件

### Verification
```bash
go test -v -run TestDept.*Recursive ./internal/services/operations/
go build ./...
```

---

## Plan 99-04 / 99-05: V130R-09 分页 clamp 单一权威收敛（D-03 已敲定）

D-03 讨论已完成（方案 B 单一权威出口），按 checker 反馈拆分为两个计划，完整任务分解见：

- **99-04（wave 1）**：`99-04-PLAN.md` — Go 分页口径核心：`pkg/query.NormalizePaginationWithMax` 变体、`pagination_helper.go` @deprecated、workstation List cap=200（D-03-8）、三个 extractPagination/clampPageSize 生产调用方迁移（D-03-3）、vdi 内联 100 清理（D-03-4 重定靶：真实内联 clamp 在 `vdi_server_service_impl.go:70`，dict/post 仅过时注释）
- **99-05（wave 2，depends_on 99-04）**：`99-05-PLAN.md` — CAD 全集专用端点 `GET /ops/workstation/:floorId/workstations-all`（workstation 组注册，D-03-6/7）+ 6 个前端消费者迁移 + 前后端测试同步（D-03-9；assets/index.tsx 为 Excel 导出路径，排除）

---

## Wave Analysis

**Wave 1（Plan 99-01, 99-02, 99-03, 99-04）：**
- 99-01: building/asset service 修改（已完成）
- 99-02: floor service 修改（已完成）
- 99-03: 新 dept_filter.go + 四服务替换（已完成）
- 99-04: Go 分页口径核心（touches workstation_service.go List）

**Wave 2（Plan 99-05，depends_on 99-04）：**
- CAD 端点 + 前端消费者迁移（同样 touches workstation_service.go，共享文件强制后排）

---

## Regression Guards

1. V130R-06 回归测试 — 软删 Total 正确过滤
2. V130R-07 回归测试 — 换楼顺序正确
3. V130R-08 回归测试 — orgId 递归筛选正确
4. V130R-09 回归测试 — NormalizePaginationWithMax 边界 + CAD 端点 + 前端消费者断言
5. `go build ./...` — 无编译错误
6. `go test ./...` — 0 failure
7. 七 gate 全程不倒退

---

## Notes

- Phase 91 `base.GORMRepository[T]` 是 building/asset/floor list 的基础，修复时注意不要破坏 repo 化成果
- V130R-08 的共享 helper 应放在 `internal/services/operations/` 目录，与现有 `pagination_helper.go` 同级
- V130R-07 换楼问题可能涉及外键约束，需要事务保护
