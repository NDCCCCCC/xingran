# Phase 99: operations 口径统一 - Context

**Gathered:** 2026-09-06
**Status:** Ready for planning

## Phase Boundary

统一 operations 域四处口径/语义缺陷（V130R-06/07/08/09）。Plan 99-01/02/03 已完成，本次讨论仅针对 Plan 99-04（V130R-09 分页口径收敛）。

## Implementation Decisions

### V130R-09 分页口径收敛（Plan 99-04）

- **D-03-1:** 采纳方案 B——单一权威出口。`pkg/query.NormalizePagination` 为唯一分页归一化函数。
- **D-03-2:** Go 最佳实现：`NormalizePagination` 保持 2 参数（cap=200，向后兼容）；新增 `NormalizePaginationWithMax(current, pageSize, maxPageSize int)` 3 参数版。
  ```go
  // pkg/query/pagination.go
  func NormalizePagination(current, pageSize int) (int, int) {
      return NormalizePaginationWithMax(current, pageSize, MaxPageSize)
  }
  func NormalizePaginationWithMax(current, pageSize, maxPageSize int) (int, int) {
      if current <= 0 { current = DefaultCurrent }
      if pageSize <= 0 { pageSize = DefaultPageSize }
      if pageSize > maxPageSize { pageSize = maxPageSize }
      return current, pageSize
  }
  ```
- **D-03-3:**（2026-09-06 用户长程标准修订）ops services 改用 `query.NormalizePaginationWithMax(c, s, 10000)`；原「clampPageSize/extractPagination @deprecated 保留」条款被用户指令「不考虑向后兼容」覆盖——迁移后零调用方即直接删除（含对应死测试）。
- **D-03-4:** 内联硬编码 `100`（dict/post 等老 service）全部替换为 `query.NormalizePagination(c, s, 100)` 或 `query.NormalizePagination(c, s, constants.MaxListPageSize)`。

### CAD 楼层视图反模式修复

- **D-03-5:** 消除 CAD/3D 楼层视图滥用 `pageSize=10000` 分页做全集下拉的反模式。
- **D-03-6:** 新增专用端点 `GET /ops/floors/{floorId}/workstations-all`，无分页参数，返回该楼层全部工位。
- **D-03-7:** 新端点返回格式同 List：`{list: [...], total: N}`。
- **D-03-8:** `workstation_service.go` 删除 `GetPaginationWithMax(constants.MaxOptionsPageSize)`，改用 `query.NormalizePagination` cap=200。
- **D-03-9:** 前后端同步改造，包括前后端测试。
- **D-03-10:**（2026-09-07 orchestrator 补充裁决）internal/constants 整体合并进 pkg/constants（Phase 89/90 leaf-const 唯一权威）；DefaultCurrent/DefaultPageSize 保留 pkg 既有定义。

### 99-01/02/03 继承决策（已在 SUMMARY 中记录）

- **V130R-06-1:** building/asset List Total 用 `Model()` 起链而非 `Table()`，gorm 自动附上 `deleted_at IS NULL` scope 过滤软删除。
- **V130R-07-1:** floor 换楼使用乐观锁 `WHERE id=? AND building_id=?`，`RowsAffected==0` 时返回 `ErrFloorNotInExpectedBuilding`。
- **V130R-08-1:** 提取共享 `BuildDeptRecursiveFilter` helper 到 `internal/services/operations/dept_filter.go`。
- **V130R-08-2:** 修复 3 条件到 4 条件 ancestor 匹配，修复 deptId 出现在路径中间位置漏匹配问题。

## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Pagination Architecture
- `pkg/query/pagination.go` — NormalizePagination/NormalizePaginationWithMax 现状
- `internal/services/operations/pagination_helper.go` — 待废弃的 clampPageSize/extractPagination
- `internal/api/v1/operations/requests/common.go` — GetPaginationWithMax 现状

### Workstation Service
- `internal/services/operations/workstation_service.go` — GetPaginationWithMax 使用位置
- `internal/services/operations/dept_filter.go` — Plan 99-03 新增 helper

### Constants
- `pkg/constants/pagination.go` — DefaultCurrent/DefaultPageSize/MaxPageSize/MinPageSize/MaxListPageSize/MaxOptionsPageSize（V130R-09 D-03-10 后唯一常量权威，internal/constants 已删除）

### Phase 99 Prior Plans
- `.planning/phases/99-operations口径统一/PLAN.md` — 99-01/02/03/04 完整计划
- `.planning/phases/99-operations口径统一/99-02-SUMMARY.md` — 99-02 执行结果
- `.planning/phases/99-operations口径统一/99-03-SUMMARY.md` — 99-03 执行结果

### Regression Guards
- `internal/services/operations/pagination_helper_test.go` — clampPageSize 测试待迁移/删除
- `internal/api/v1/operations/requests/pagination_cap_test.go` — GetPaginationWithMax 测试待迁移/删除

## Existing Code Insights

### Reusable Assets
- `pkg/query.NormalizePaginationWithMax` — 即将成为唯一分页权威，现有 NormalizePagination 调用方无需改动
- `operations/requests.GetPaginationWithMax` — handler 层便捷封装，内部调用 query，可保留

### Established Patterns
- ops services 用 `map[string]interface{}` bind 参数，通过 `extractIntParam` 提取
- `building_service`/`asset_service` 已 repo 化，List 经 `base.GORMRepository.List`
- `dept_filter.go`（99-03）是 ops services 共享 helper 的范例

### Integration Points
- `ops_router.go` — 新端点注册位置
- `workstation_service.go` — CAD 端点新增 service 层
- 前端 `xingran-react-frontend/src/lib/opsApi.ts` — 需要新增 `getFloorWorkstationsAll` 方法

## Specific Ideas

- CAD 楼层视图前端组件需要确定具体是哪个（Three.js floor plan renderer）
- 新端点路径 `/ops/floors/{floorId}/workstations-all` vs `/ops/floor/{id}/workstations-all` — 按现有路由风格选择

## Deferred Ideas

None — CAD 反模式已在此次 scope 内通过专用端点方案解决。

---

*Phase: 99-operations口径统一*
*Context gathered: 2026-09-06*
