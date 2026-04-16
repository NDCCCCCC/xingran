# Roadmap

## Project: 工位导入部门/用户关联

**Created**: 2026-04-16
**Total Phases**: 2
**Total Requirements**: 7 (6 v1, 0 v2 deferred)

---

## Phase 1: 后端导入配置

**Goal**: 在 Excel 导入配置中添加部门和用户列，实现名称→ID 自动匹配

**Requirements**: IMPORT-01, IMPORT-02, IMPORT-03
**UI hint**: no

**Changes**:
- `internal/services/operations/excel_config.go` — workstation 配置添加部门和用户列

**Success Criteria**:
1. 导入含部门名称和用户名的 Excel 文件后，`sys_workstation` 表的 `dept_id` 和 `user_id` 字段正确存储 UUID
2. 导入不含部门/用户信息的 Excel 文件后，`dept_id` 和 `user_id` 为空，不报错
3. 部门名称或用户名在数据库中不存在时，对应字段留空，导入不中断

---

## Phase 2: 验证与导出

**Goal**: 确保导入后前端正确展示，导出包含部门/用户列，模板可下载

**Requirements**: IMPORT-04, TMPL-01, VAL-01, VAL-02
**UI hint**: yes

**Changes**:
- 检查/更新导入模板下载逻辑
- 检查/更新导出配置（如导出功能存在）
- 端到端验证：导入→列表展示

**Success Criteria**:
1. 工位列表"所属部门"列显示正确的部门名称
2. 工位列表"所属用户"列显示正确的用户姓名
3. 下载的导入模板包含"所属部门"和"所属用户"列头
4. 导出的 Excel 包含部门和用户信息

---

## Requirement Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| IMPORT-01 | Phase 1 | pending |
| IMPORT-02 | Phase 1 | pending |
| IMPORT-03 | Phase 1 | pending |
| IMPORT-04 | Phase 2 | pending |
| TMPL-01 | Phase 2 | pending |
| VAL-01 | Phase 2 | pending |
| VAL-02 | Phase 2 | pending |

---

## Coverage Check

- [x] All v1 requirements mapped to phases
- [x] No orphan requirements
- [x] Phase dependencies respected (Phase 2 depends on Phase 1)
- [x] Success criteria are observable behaviors
