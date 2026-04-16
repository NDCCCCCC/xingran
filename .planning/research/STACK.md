# Research: Stack

## 现状分析

本次开发仅需修改 Excel 导入配置，无需引入新技术栈。

### 已有技术组件（可直接复用）

| 组件 | 位置 | 用途 |
|------|------|------|
| `ReferenceResolver` | `internal/services/operations/reference_resolver.go` | 名称→UUID 批量解析，支持 `table.field` 格式 |
| `ExcelColumn.Reference` | `internal/services/operations/excel_config.go` | 列配置中声明引用关系，如 `"sys_dept.dept_name"` |
| `BatchUpserter` | `internal/services/operations/batch_upserter.go` | 批量写入，自动处理 Reference 字段的 ID 解析 |
| `Workstation` 模型 | `internal/models/workstation.go` | 已有 `DeptID`/`UserID` 字段 |
| List JOIN 查询 | `workstation_service.go:15-16` | 已 JOIN `sys_dept`/`sys_user` 返回名称 |

### 关键技术点

1. **Reference 解析机制**：`ExcelColumn.Reference = "table.field"` 格式，`ReferenceResolver` 自动将 Excel 中的名称值通过 `SELECT id FROM table WHERE field = value AND deleted_at IS NULL` 解析为 UUID。

2. **DBField 映射**：`ExcelColumn.DBField` 指定写入数据库的目标字段名。例如 `Reference: "sys_dept.dept_name", DBField: "dept_id"` 表示用部门名称查 `sys_dept`，结果写入 `dept_id`。

3. **可选字段**：`ExcelColumn.Required` 默认 false，不填则跳过，匹配失败不会阻断导入。

### 需确认的引用字段

| Excel 列名 | Reference | DBField | 说明 |
|------------|-----------|---------|------|
| 所属部门 | `sys_dept.dept_name` | `dept_id` | 部门名称→部门ID |
| 所属用户 | `sys_user.user_name` | `user_id` | 用户名→用户ID |

注意：`sys_user` 表的用户名字段是 `user_name`，不是 `nickname`。前端显示用的是 `nickname`（通过 JOIN `sys_user.nickname as user_name`），但导入匹配应该用 `user_name`（唯一且稳定）。

## 结论

无需新增依赖。仅需在 `excel_config.go` 的 workstation 配置中添加两个 `ExcelColumn` 条目。
