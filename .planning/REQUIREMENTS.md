# Requirements

## v1 Requirements

### 导入配置
- [ ] **IMPORT-01**: Excel 导入配置添加"所属部门"列，通过部门名称匹配 `sys_dept.dept_name`，写入 `dept_id`
- [ ] **IMPORT-02**: Excel 导入配置添加"所属用户"列，通过用户名匹配 `sys_user.user_name`，写入 `user_id`
- [ ] **IMPORT-03**: 两个字段均为可选，匹配失败时留空不阻断导入
- [ ] **IMPORT-04**: Excel 导出包含"所属部门"和"所属用户"列（如导出功能存在）

### 导入模板
- [ ] **TMPL-01**: 导入模板下载包含"所属部门"和"所属用户"列头

### 验证
- [ ] **VAL-01**: 导入含部门/用户名的 Excel 后，工位列表正确显示部门名称和用户姓名
- [ ] **VAL-02**: 导入不含部门/用户名的 Excel 后，工位列表对应列为空

## v2 Requirements (Deferred)

- 按-orgId 级联匹配部门（解决同名部门问题）
- 导入时显示匹配失败的警告信息
- 批量修改已有工位的部门/用户关联

## Out of Scope

- 导入时自动创建不存在的部门或用户 — 超出导入功能职责
- 工位与用户多对多关系 — 当前只需一对一
- 部门树级联选择匹配 — 复杂度高，用户未要求

## Traceability

| Phase | Requirements | Status |
|-------|-------------|--------|
| — | — | — |
