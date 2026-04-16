---
wave: 1
depends_on: []
files_modified:
  - internal/services/operations/excel_config.go
requirements:
  - IMPORT-01
  - IMPORT-02
  - IMPORT-03
autonomous: true
---

# Plan 01: 添加工位导入部门/用户列配置

<objective>
在 excel_config.go 的 workstation 配置中添加"所属部门"和"所属用户"两个可选列，复用现有 ReferenceResolver 实现名称→UUID 自动匹配。
</objective>

## Task 1: 添加 Excel 导入列配置

<read_first>
- internal/services/operations/excel_config.go（当前 workstation 列配置在第 108-124 行）
- internal/services/operations/reference_resolver.go（了解 Reference 解析机制）
- internal/models/workstation.go（确认模型字段）
</read_first>

<action>
在 `excel_config.go` 第 122 行（`remark` 列之前）添加两个 ExcelColumn 配置：

```go
{Field: "deptName", Header: "所属部门", Reference: "sys_dept.dept_name", DBField: "dept_id"},
{Field: "userName", Header: "所属用户", Reference: "sys_user.user_name", DBField: "user_id"},
```

- `Reference: "sys_dept.dept_name"` 让 ReferenceResolver 通过部门名称查询 sys_dept 表获取 dept_id
- `Reference: "sys_user.user_name"` 让 ReferenceResolver 通过用户名查询 sys_user 表获取 user_id
- 两个字段均无 `Required: true`，匹配失败时自动留空，不阻断导入
- `DBField` 指定写入 workstation 表的目标列名（dept_id / user_id）
</action>

<acceptance_criteria>
- `excel_config.go` workstation 配置的 Columns 数组包含 `deptName` 和 `userName` 两个条目
- `grep -c "deptName.*所属部门" internal/services/operations/excel_config.go` 返回 1
- `grep -c "userName.*所属用户" internal/services/operations/excel_config.go` 返回 1
- `go build ./...` 编译通过
</acceptance_criteria>

## Task 2: 验证编译和基本逻辑

<read_first>
- internal/services/operations/excel_config.go（确认修改正确）
</read_first>

<action>
1. 运行 `go build ./...` 确认编译通过
2. 检查 ReferenceResolver 是否支持 `sys_dept` 和 `sys_user` 表（它支持任意 `table.field` 格式，无需额外配置）
3. 确认 workstation 模型已有 `DeptID` 和 `UserID` 字段（无需数据库迁移）
4. 确认 workstation_service.go 的 List 查询已 JOIN sys_dept 和 sys_user（无需修改）
</action>

<acceptance_criteria>
- `go build ./...` 返回 exit code 0
- `grep "sys_dept.dept_name" internal/services/operations/excel_config.go` 有匹配
- `grep "sys_user.user_name" internal/services/operations/excel_config.go` 有匹配
- `grep "DeptID" internal/models/workstation.go` 有匹配（确认模型已支持）
- `grep "UserID" internal/models/workstation.go` 有匹配（确认模型已支持）
</acceptance_criteria>

<must_haves>
- workstation Excel 导入配置包含"所属部门"列（IMPORT-01）
- workstation Excel 导入配置包含"所属用户"列（IMPORT-02）
- 两个字段均为可选，匹配失败不阻断导入（IMPORT-03）
- 编译通过，无新错误
</must_haves>
