---
quick_id: 261106-wso
slug: workstation-floor-org-override
status: planned
date: 2026-11-06
---

# 一栋楼跨机构：楼层可覆盖所属机构（ops_floors.org_id）

## 用户决策（已锁定）

- **方案 A 直接做**，不做临时拆楼宇方案。
- 痛点「两者都有」：编辑工位时所属部门下拉要有武汉中支部门；左侧选武汉中支要能看到 1/3 楼工位。即**全链路**都要修。
- 不自动提交（项目 CLAUDE.md：提交前询问用户）。

## 根因

`ops_buildings.org_id` 单值 = 分公司本部；`OpsFloor` 无归属字段，楼层继承楼宇。工位模块 4 个消费点全部读 `b.org_id`：

- `workstation_service.go:92 / :325 / :536`Statistics / List / SearchOptions → `workstationOrgExistsSub` 的 `BuildDeptRecursiveFilter(orgId, "b.org_id")`
- `floor_service.go:277 / :412` List / SearchFloorOptions → `b.org_id = ?`（精确等值）
- `building_service.go:256 / :351` 楼宇 List / SearchBuildingOptions → `org_id IN deptIDs`（Cascader 第一级数据源，**不改则选武汉中支时根本拿不到浙商大厦**）
- 前端 `index.tsx:530-543` `watchedOrgId = building.orgId` → 部门下拉 union / `[映射]` 注入

`sys_dept_location_alias`（Phase 39）只作用于「所属部门下拉」，方向是「部门→机构」，且 partial unique `(dept_id, scope)` + location 必须 `is_external_org=1`，**不能**承担「楼层→机构」的空间归属。

## 数据模型

`OpsFloor` 加 `OrgID *string`（`gorm:"column:org_id;size:64"` / `json:"orgId,omitempty"`），**可空 = 继承楼宇 `ops_buildings.org_id`**。

- `MigrateModelList` 已注册 `&operations.OpsFloor{}`（database.go:759），AutoMigrate 自动 ADD COLUMN。
- 另加 migration 234（`HasColumn` 幂等兜底），覆盖 `SKIP_AUTOMIGRATE` 旁路；database.go 两个分支（PG :1191 后 / sqlite :1312 后）注册。
- 不做历史数据回填（生产 UUID 未知）：给用户在 SUMMARY 提供按名称回填的 SQL 模板，由用户确认后执行。

## 后端：统一 `COALESCE`

新增常量/小 helper 表达「楼层有效机构」：`COALESCE(f.org_id, b.org_id)`（工位侧，表别名 f/b）；楼层侧用 `COALESCE(ops_floors.org_id, b.org_id)`。

| 文件 | 改动 |
|---|---|
| `workstation_service.go` | `workstationOrgExistsSub` 三处调用点 `:92 / :325 / :536` 的 `BuildDeptRecursiveFilter(orgId, "b.org_id")` → `"COALESCE(f.org_id, b.org_id)"`。helper 内 `CAST(<expr> AS TEXT)` 对复合表达式合法，PG/SQLite 双方言 OK。 |
| `floor_service.go:277 / :412` | `b.org_id = ?` → `COALESCE(ops_floors.org_id, b.org_id) = ?`。**保持原有精确等值语义**（既有与工位递归口径不一致不在本次范围，避免回归）。 |
| `building_service.go:280 / :363` | 楼宇 orgId 筛选加 `OR EXISTS (SELECT 1 FROM ops_floors f WHERE CAST(f.building_id AS TEXT) = CAST(ops_buildings.id AS TEXT) AND f.org_id IN ? AND f.deleted_at IS NULL)`，使「含覆盖楼层的楼宇」也能被该机构选中。 |

`GetWorkstationDeptOptions` 后端**不改**（入参语义是「该工位实际归属机构」，由前端传对值即可）。

## 前端

| 文件 | 改动 |
|---|---|
| `types/operations.ts` | `Floor` 加 `orgId?: string` |
| `pages/operations/floors/components/FloorModal.tsx` | 新增可选 Form.Item `name="orgId"`「楼层归属机构（留空继承楼宇）」，用现有 `DepartmentTreeSelect` + `departments` prop；**编辑态也可编辑**（现有「所属机构」disabled 是楼宇下拉过滤器，语义不同，标签明确区分） |
| `pages/operations/floors/index.tsx` | `openModal` 编辑时：`floorForm.setFieldsValue(record)` 已带 `orgId` → 自动回显；`modalDeptId` 仍取 `building.orgId`（楼宇下拉过滤，不变）；`handleSave` 的 `values` 自然含 `orgId`，`floorApi.create/update` 直传 |
| `pages/operations/workstations/index.tsx` | `setEditFormValues`：`orgId = floor?.orgId ?? building?.orgId`，用于表单回显 + `setWatchedOrgId` → 部门下拉 union |

## Excel

- `excel_export_config.go` floor 导出加 `{Field: "orgId", Header: "所属机构ID", DBField: "org_id"}`（只读，便于核对）。
- **导入配置不加列**：`floor` 走 `standardUpsert`，`ON CONFLICT DO UPDATE` 会把记录里缺失的 `org_id` 写成 NULL，用户不填机构列重导会把 1/3 楼覆盖值清空。导出文件带该列但导入不识别（表头未匹配 → 跳过），不会误清空。
- 批量设置走楼层表单，或按 SUMMARY 的 SQL 模板回填。

## 回归测试（新 `floor_org_override_test.go`）

| 用例 | 断言 |
|---|---|
| 楼层 org_id 覆盖后，工位 List/Statistics 按该机构可筛到 | 覆盖楼层工位出现，继承楼层不出现 |
| 楼层 List / SearchFloorOptions 按覆盖机构可筛到 | 同上 |
| 楼宇 SearchBuildingOptions 因含覆盖楼层而命中 | 浙商大厦出现在武汉中支下 |
| 继承语义：org_id 为 NULL 时按 building.org_id | 选分公司本部仍能筛到所有楼层 |
| 编辑回显数据源（unit 级，前端由 type-check + 现有 render 测试覆盖） | — |

## 验证

`go build ./...` / `go vet ./internal/services/operations/` / `go test ./internal/services/operations/ -count=1` / `go test ./internal/core/db/migrations/ -count=1` / 前端 `npm run type-check` + 相关 vitest。

## 不做

- 不拆楼宇、不改 alias 表结构、不改 `batch_upserter.go` 的 upsert 语义。
- 不碰 `internal/services/system/asset_columns_schema.json`（用户既有未提交改动）。
- 不擅自回填生产数据。
