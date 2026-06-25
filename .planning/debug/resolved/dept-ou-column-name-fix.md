---
slug: "dept-ou-column-name-fix"
status: fixing
trigger: "部门AD同步列名错误"
created: "2026-05-25T07:50:00.000Z"
updated: "2026-05-25T07:50:00.000Z"
tdd_mode: "false"
---

# Debug Session: GORM列名映射错误

## Trigger

部门AD同步时数据库列名不匹配错误

**错误信息**:
```
ERROR: column "oudn" of relation "sys_dept_ou_mapping" does not exist (SQLSTATE 42703)
```

## Symptoms

### Expected behavior
- GORM应该生成正确的SQL，使用数据库列名 `ou_dn` 和 `parent_ou_dn`

### Actual behavior
- GORM生成的INSERT语句使用 `oudn`，但数据库列名是 `ou_dn`
- 错误发生在部门OU映射关系创建/更新时

### Error messages
```
INSERT INTO "sys_dept_ou_mapping" ("dept_id","ad_config_id","oudn",...
ERROR: column "oudn" of relation "sys_dept_ou_mapping" does not exist
```

### Timeline
- 在修复了parent_id UUID问题后，部门AD同步能查询根部门
- 但在创建/更新OU映射关系时失败

### Reproduction
- 执行部门到AD同步任务
- 到达映射关系更新步骤时报错

## Current Focus

- **hypothesis**: `DeptOUMapping` 模型中缺少 GORM 列名标签
- **test**: 检查 `internal/models/dept_ou_mapping.go` 中的字段定义
- **expecting**: `OUDN` 字段需要添加 `gorm:"column:ou_dn"` 标签
- **next_action**: 修复模型定义，添加正确的列名映射
- **reasoning_checkpoint**: GORM默认将字段名转为小写，但数据库使用snake_case命名

## Evidence

### 2026-05-25T07:50:00.000Z
- **source**: code_inspection
- **evidence**: 
  - 文件: `internal/models/dept_ou_mapping.go`
  - 第16行: `OUDN string` 缺少 `gorm:"column:ou_dn"` 标签
  - 第18行: `ParentOUDN *string` 缺少 `gorm:"column:parent_ou_dn"` 标签
  - GORM默认将 `OUDN` 转为 `oudn`，但数据库列名是 `ou_dn`

## Eliminated

None - direct issue found

## Resolution

- **root_cause**: GORM字段名 `OUDN` 转为小写 `oudn`，但数据库列名使用 `ou_dn` (snake_case)
- **fix**: 在字段标签中添加明确的列名映射
  - `OUDN` 字段添加 `gorm:"column:ou_dn"`
  - `OUName` 字段添加 `gorm:"column:ou_name"`
  - `ParentOUDN` 字段添加 `gorm:"column:parent_ou_dn"`
- **files_changed**: `internal/models/dept_ou_mapping.go`
- **verification**: ✅ 编译通过 (`go build ./internal/models/...`)
- **status**: fixed

## Verification Steps

1. ✅ 模型定义已修复
2. ✅ 代码编译通过
3. ⏳ 需要用户重新测试部门AD同步功能
4. ⏳ 验证OU映射关系正确创建

## TDD Checkpoint

Not applicable - straightforward model fix
