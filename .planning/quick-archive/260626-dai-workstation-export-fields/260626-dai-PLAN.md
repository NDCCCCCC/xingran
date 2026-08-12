# Quick Task 260626-dai: 工位导出补充所属部门/人员名称 + 设备序列号等

**Created:** 2026-06-26
**Status:** Ready for execution

## 任务

工位管理 Excel 导出缺少「所属部门名称」「所属人员名称」「设备序列号」等字段，需补全。

## 根因（已调研确认）

导出链路：`exportData` handler → `ExcelService.ExportData("workstation", ...)` → 命中 `GetExportConfig("workstation")` → `excelExportServiceImpl.ExportData` → `DefaultQueryBuilder` + `resolveAssociations` + `formatExportCellValue`。

- `DefaultQueryBuilder` 只做 `db.Table("sys_workstation").Where("deleted_at IS NULL")`，**无任何 JOIN**，结果 map 只含 `sys_workstation` 自身列。
- `sys_workstation.dept_name` / `user_name` 存储列在导入/创建路径只写 `dept_id` / `user_id`（见 `workstation_service.go` 的 JOIN select `sys_dept.dept_name as dept_name, sys_user.nickname as user_name` 才有名称），所以这两列在表里基本为空。
- 当前 config 里 `deptName`/`userName` 用 `DBField:"dept_name"`/`"user_name"` 直读存储列 → 导出为空。而 `floorName`/`buildingName` 用 `Join` 配置走 `resolveAssociations` 才有值。
- `resolveAssociations` 用 `Where(join.RightField+" IN ?", ids)` 把 varchar 的 `floor_id`/`building_id`/`dept_id`/`user_id` 直接 IN 到 uuid 列，PG 下 `uuid = text` 有类型坑（`floor_id::uuid` 转换只在生产 List 查询里做，resolveAssociations 没做）。
- 设备序列号在 `ops_workstation_device.device_serial`（按 `workstation_id` 关联，`is_primary` 标主设备），config 里根本没有该列。

## 方案（镜像生产 List 查询 `workstationJoinClause`，最稳健）

新增 `WorkstationQueryBuilder`，用显式 LEFT JOIN + `::uuid`/`::text` 转换一次性带出 楼层/楼宇/部门/人员 名称，并用相关子查询取主设备的 序列号/名称/型号。彻底绕开 `resolveAssociations` 的类型坑（config 不再带 Join → resolveAssociations 提前返回）。

### 改动 1：`internal/services/operations/excel_query_builder.go`

1. 新增 `WorkstationQueryBuilder` 结构 + `NewWorkstationQueryBuilder()`。
2. `BuildQuery`：
   - `db.Table("sys_workstation")`
   - `Select` 显式列出需要的列 + JOIN 别名 + 设备子查询别名（别名必须与 config `DBField` 对齐）：
     - `sys_workstation.id, workstation_name, workstation_type, status, location, description, dept_id, user_id, floor_id, building_id, created_at, updated_at`
     - `ops_floors.name AS floor_name`
     - `ops_buildings.name AS building_name`
     - `sys_dept.dept_name AS dept_name`
     - `sys_user.nickname AS user_name`
     - `(SELECT device_serial FROM ops_workstation_device WHERE workstation_id = sys_workstation.id AND deleted_at IS NULL ORDER BY (CASE WHEN is_primary = true THEN 0 ELSE 1 END), priority DESC, created_at ASC LIMIT 1) AS device_serial`
     - 同结构 `device_name`、`device_model` 两个子查询
   - `Joins`：
     - `LEFT JOIN ops_floors ON ops_floors.id = sys_workstation.floor_id::uuid`
     - `LEFT JOIN ops_buildings ON ops_buildings.id = sys_workstation.building_id::uuid`
     - `LEFT JOIN sys_dept ON sys_dept.id::text = sys_workstation.dept_id`
     - `LEFT JOIN sys_user ON sys_user.id::text = sys_workstation.user_id`
   - `Where("sys_workstation.deleted_at IS NULL")`
   - 复用 `config.FilterMapping` 应用筛选（`name→workstation_name, status, floorId→floor_id, deptId→dept_id, userId→user_id`），字段无 `.` 前缀时加 `sys_workstation.` 表别名；类型 switch 与 `DefaultQueryBuilder` 一致（string→LIKE，int/int64/float64/bool→=，[]interface{}→IN）。需要 import `strings`。
3. `GetQueryBuilder` 增加 `case "workstation": return NewWorkstationQueryBuilder(), true`。

> SQLite 兼容：`::uuid`/`::text` 为 PG 专有，但 `workstation_service.go` 生产 List 查询已大量使用同款转换，本改动与之同源、不引入新不一致；不在本任务范围解决双 DB 方言。

### 改动 2：`internal/services/operations/excel_export_config.go`

`workstation` 配置项修改：
- `QueryBuilder: "default"` → `QueryBuilder: "workstation"`
- `floorName` 列：去掉 `Join`，改为 `DBField: "floor_name"`（值由 builder JOIN 提供）
- `buildingName` 列：去掉 `Join`，改为 `DBField: "building_name"`
- `deptName`（已是 `DBField:"dept_name"`）、`userName`（已是 `DBField:"user_name"`）保持不变，现由 builder JOIN 提供真实值
- 在 `userName` 之后、`workstationType` 之前新增三列：
  - `{Field: "deviceSerial", Header: "设备序列号", DBField: "device_serial"}`
  - `{Field: "deviceName", Header: "设备名称", DBField: "device_name"}`
  - `{Field: "deviceModel", Header: "设备型号", DBField: "device_model"}`
- 其余列（workstationName/floorId/buildingId/deptId/userId/workstationType/location/status/remark/createdAt/updatedAt）不变。

## 不改动的部分

- 导入流程、List 查询、前端（前端列定义 `deptName`/`userName` 已是期望字段名，导出对齐即可）。
- `excel_export_service.go`（resolveAssociations / formatExportCellValue 逻辑无需改：无 Join 时 resolveAssociations 提前返回，formatExportCellValue 按 DBField 读列）。

## 验收（must_haves）

1. `go build ./...` 通过。
2. `go test ./internal/services/operations/` 现有测试通过（不新增依赖 SQLite 的 builder 测试，避免 `::uuid` 方言冲突）。
3. 导出 workbook「工位列表」sheet 含新表头：所属部门 / 所属人员 / 设备序列号 / 设备名称 / 设备型号，且部门/人员名称非空（由 JOIN 实时取值）。
4. floor/building 名称仍正常（由 builder JOIN 提供，行为不回归）。
5. operlog：导出走既有 `exportData` handler，已记录 `OperTypeExport`，无需新增。

## 关键文件

- `internal/services/operations/excel_query_builder.go`（改）
- `internal/services/operations/excel_export_config.go`（改）
- 参考：`internal/services/operations/workstation_service.go`（JOIN 同源）、`internal/services/operations/excel_export_service.go`（formatExportCellValue/resolveAssociations）
