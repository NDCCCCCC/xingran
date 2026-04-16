# Research: Architecture

## 影响范围分析

### 需修改的文件（预计 2-3 个）

1. **`internal/services/operations/excel_config.go`** — 添加 workstation 的部门/用户列配置
2. **`internal/services/operations/excel_service.go`** — 可能需要处理导出时部门/用户名称的导出（如果导出也需要）
3. **`ruoyi-react-frontend/src/pages/operations/workstations/`** — 前端导入模板下载（如果模板是动态生成的）

### 不需要修改的文件

| 文件 | 原因 |
|------|------|
| `internal/models/workstation.go` | 已有 `DeptID`/`DeptName`/`UserID`/`UserName` 字段 |
| `workstation_service.go` | List/GetByID 已 JOIN 返回部门名称和用户姓名 |
| `reference_resolver.go` | 通用解析器，支持任意 `table.field` 格式 |
| `batch_upserter.go` | 通用批量写入器，根据 config 自动处理 |
| `columns.tsx` | 前端已有"所属部门"和"所属用户"列 |

## 数据流

```
Excel 文件上传
    ↓
excel_handler.go 接收文件
    ↓
excel_service.go 解析 Excel
    ↓
excel_config.go 定义列映射 (需添加部门/用户列)
    ↓
reference_resolver.go 名称→UUID (自动处理 Reference 配置)
    sys_dept.dept_name → dept_id
    sys_user.user_name → user_id
    ↓
batch_upserter.go 批量写入 sys_workstation
    写入 dept_id, user_id 字段
    ↓
列表查询时 JOIN sys_dept/sys_user 自动返回名称
    ↓
前端展示 (已有列)
```

## 构建顺序

1. **后端**: 修改 `excel_config.go`（核心变更，约 2 行代码）
2. **验证**: 导入测试 → 确认列表展示
3. **前端**: 检查导入模板下载是否需要更新
