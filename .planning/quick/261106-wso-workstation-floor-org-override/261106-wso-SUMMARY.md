---
quick_id: 261106-wso
status: complete
date: 2026-11-06
commit: none (按项目约定，提交前须经用户确认)
---

# 完成：一栋楼跨机构办公 —— 楼层可覆盖所属机构（方案 A）

## 根因

`ops_buildings.org_id` 单值（浙商大厦 = 分公司本部），`OpsFloor` 无归属字段，楼层继承楼宇。工位模块 4 个消费点全读 `b.org_id`：List/Statistics/SearchOptions（workstation_service.go）、楼层 List/SearchFloorOptions（floor_service.go）、楼宇 List/SearchBuildingOptions（building_service.go）、前端编辑回显（index.tsx building.orgId）。故 1/3 楼无法显示武汉中支部门，左侧选武汉中支也筛不出 1/3 楼工位。

`sys_dept_location_alias`（Phase 39）只作用于「所属部门下拉」且方向是「部门→机构」、partial unique + location 必须外部机构，无法承担「楼层→机构」的空间归属，故未在本次复用。

## 实现

### 数据层
- `internal/models/operations/floor.go`：`OpsFloor` 加 `OrgID *string`（可空 = 继承楼宇 org_id）。
- `internal/core/db/migrations/migration_234_floor_org_id.go` + `_test.go`：`HasColumn` 幂等兜底，覆盖 `SKIP_AUTOMIGRATE` 旁路；**不做历史数据回填**（存量行为不变）。
- `internal/core/db/database.go`：PG / sqlite 两分支注册 234。

### 后端筛选（统一「楼层覆盖优先、NULL/空串继承楼宇」）
- `workstation_service.go`：新增 `workstationOrgColumn = COALESCE(NULLIF(f.org_id, ''), b.org_id)`，Statistics / List / SearchWorkstationOptions 三处 `BuildDeptRecursiveFilter` 改用该表达式。
- `floor_service.go`：List / SearchFloorOptions 的 orgId EXISTS 改 `COALESCE(NULLIF(ops_floors.org_id, ''), b.org_id) = ?`。
- `building_service.go`：`applyDeptFilter` / `SearchBuildingOptions` 的 orgId 筛选加 `OR EXISTS (楼层 org_id IN deptIDs)` —— 让浙商大厦出现在武汉中支的楼宇下拉（Cascader 第一级）。EXISTS 不引入重复行。
- 5 个 SQLite 测试 fixture 补 `org_id` 列（`AutoMigrate` 的除外）。

### 前端
- `types/operations.ts`：`Floor` 加 `orgId?`。
- `floors/components/FloorModal.tsx`：新增可选 Form.Item「楼层归属机构」（`DepartmentTreeSelect` + allowClear），**编辑态也可改**，与上方「所属机构」（楼宇下拉过滤器、编辑禁用）语义分离；标签 + extra 明确「留空继承楼宇」。
- `workstations/index.tsx`：
  - `setEditFormValues`：`orgId = floor?.orgId || building?.orgId` → 所属机构回显 + `useAliasByLocation` 部门下拉数据源。
  - `loadFloorsForCascader` 加可选 `orgId`；`handleCascaderLoadData` 用 `cascaderOrgIdRef`（新增流程过滤楼层、编辑回显不过滤），避免跨机构楼宇列出其它机构楼层。
- `shared/ExcelImport.tsx`：（上一任务 261106-wsd 的反推计数展示，同分支未提交）。

### Excel
- `excel_export_config.go`：floor 导出加「所属机构ID(覆盖)」列，供核对/回填。
- **导入配置刻意不加列**：floor 走 `standardUpsert`，`ON CONFLICT DO UPDATE` 会把记录里缺失的 `org_id` 写成 NULL，误清空覆盖值。批量设置走楼层表单或下方 SQL。

## 上线后必须做的「数据回填」（未自动执行）

生产 UUID 未知，需用户确认后执行。给 1/3 楼设置覆盖值（先 SELECT 核对 ID）：

```sql
-- 1) 核对部门 / 楼宇 / 楼层 ID
SELECT id, dept_name FROM sys_dept WHERE dept_name IN ('分公司本部','武汉中支');
SELECT id, name, org_id FROM ops_buildings WHERE name = '浙商大厦';
SELECT id, name, floor_no, org_id FROM ops_floors
 WHERE building_id = '<浙商大厦ID>' ORDER BY floor_no;

-- 2) 把 1、3 楼覆盖到武汉中支（按实际楼层号调整）
UPDATE ops_floors
   SET org_id = '<武汉中支部门ID>'
 WHERE building_id = '<浙商大厦ID>'
   AND floor_no IN ('1','3')
   AND deleted_at IS NULL;

-- 3) 其余楼层（4/5/6/47）保持 org_id IS NULL = 继承分公司本部，无需改动
```

也可在「楼层管理」页逐条编辑「楼层归属机构」字段。

## 验证

| 命令 | 结果 |
|---|---|
| `go build ./...` | PASS |
| `go vet ./internal/services/operations/ ./internal/core/db/migrations/ ./internal/models/operations/` | PASS |
| `go test ./internal/services/operations/ ./internal/core/db/migrations/ -count=1` | PASS |
| 新增 `floor_org_override_test.go` | 6/6 PASS |
| 新增 `migration_234_floor_org_id_test.go` | 4/4 PASS |
| `npm run type-check`（xingran-react-frontend） | PASS |
| 楼层/工位相关 vitest | 39/39 PASS |

回归用例：工位 List / Statistics / SearchOptions 按覆盖机构筛选、楼层 List/下拉、楼宇下拉因含覆盖楼层而命中、空串覆盖值等同继承、继承语义不被破坏。

## 未改动 / 边界

- 不拆楼宇、不改 alias 表结构、不改 `batch_upserter.go` upsert 语义。
- `floorService.Update` 的 `Save(floor)`：前端不传 `orgId` → `*string` 为 nil → 写 NULL = 继承楼宇，语义正确；编辑已有覆盖值时 `setFieldsValue(record)` 会回显，不丢。
- 楼层 List / SearchFloorOptions 的 orgId 仍为**精确等值**（既有与工位递归口径不一致未改，避免扩大回归面）；若后续发现「工位列表能筛到、楼层下拉却选不到」，再统一为递归 helper。
- 未碰无关的 `internal/services/system/asset_columns_schema.json`（用户既有未提交改动）。
- **未自动 commit**：项目 CLAUDE.md 要求提交前询问用户。
