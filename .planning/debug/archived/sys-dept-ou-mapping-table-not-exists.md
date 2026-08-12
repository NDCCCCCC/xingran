---
slug: sys-dept-ou-mapping-table-not-exists
status: resolved
trigger: 后端报错：ERRO[2026-05-25 13:16:27] [GORM错误] SELECT * FROM "sys_dept_ou_mapping" WHERE ou_dn = 'OU=基础运维科,OU=科技创新部,OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn' AND sync_enabled = true ORDER BY "sys_dept_ou_mapping"."id" LIMIT 1 | 耗时: 0s | 错误: ERROR: relation "sys_dept_ou_mapping" does not exist (SQLSTATE 42P01)
WARN[2026-05-25 13:16:27] 用户 chenchao-076 的AD OU OU=基础运维科,OU=科技创新部,OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn 未找到对应部门: 查询OU DN映射失败: ERROR: relation "sys_dept_ou_mapping" does not exist (SQLSTATE 42P01)
部门里明明有分公司本部-科技创新部-基础运维科，为什么这里系统提示未找到？
created: 2026-05-25T13:16:27+08:00
updated: 2026-05-25T13:16:27+08:00
tdd_checkpoint:
tdd_mode: false
---

## Symptoms

### Expected Behavior
AD 用户登录时，系统应该通过用户的 OU DN 在 `sys_dept_ou_mapping` 表中找到对应的部门 ID，从而建立 AD 用户与系统部门的关联关系。

### Actual Behavior
每次 AD 用户登录时，系统查询 `sys_dept_ou_mapping` 表失败，提示该表不存在。导致无法将 AD 用户映射到部门，即使部门"分公司本部-科技创新部-基础运维科"在 `sys_dept` 表中存在。

### Error Messages
```
ERRO[2026-05-25 13:16:27] [GORM错误] SELECT * FROM "sys_dept_ou_mapping" WHERE ou_dn = 'OU=基础运维科,OU=科技创新部,OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn' AND sync_enabled = true ORDER BY "sys_dept_ou_mapping"."id" LIMIT 1 | 耗时: 0s | 错误: ERROR: relation "sys_dept_ou_mapping" does not exist (SQLSTATE 42P01)
WARN[2026-05-25 13:16:27] 用户 chenchao-076 的AD OU OU=基础运维科,OU=科技创新部,OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn 未找到对应部门: 查询OU DN映射失败: ERROR: relation "sys_dept_ou_mapping" does not exist (SQLSTATE 42P01)
```

### Timeline
- **一直有问题，从未工作过**：该功能从部署开始就失败了

### Reproduction Steps
1. 配置 AD/LDAP 连接
2. AD 用户（如 chenchao-076）尝试登录
3. 系统查询 `sys_dept_ou_mapping` 表
4. 查询失败，返回错误 `relation "sys_dept_ou_mapping" does not exist`
5. 用户无法被正确映射到部门

**复现稳定性**：每次 AD 用户登录都会触发，问题稳定可复现

## Current Focus

### Hypothesis
代码期望 `sys_dept_ou_mapping` 表存在，但该表的数据库迁移文件可能缺失、未执行，或者模型定义有问题导致表未被自动创建。

### Next Action
gather initial evidence - 搜索 `sys_dept_ou_mapping` 相关的模型定义、迁移文件和使用位置

### Test
N/A

### Expecting
找到模型定义和迁移文件，确认表结构是否正确

### Reasoning Checkpoint
N/A

## Evidence

## Eliminated

## Resolution

### Root Cause
N/A

### Fix
N/A

### Verification
N/A

### Files Changed
N/A

## Evidence

- timestamp: 2026-05-25T16:30:00+08:00
- Found `DeptOUMapping` model definition in `internal/models/dept_ou_mapping.go`
- Found SQL migration file `internal/core/db/migrations/126_create_dept_ou_mapping_table.sql`
- Found manual migration script `scripts/migrate/126_create_dept_ou_mapping_table.go`
- **CRITICAL**: `DeptOUMapping` is NOT included in `database.go`'s `AutoMigrate()` list (lines 210-288)
- Table is actively used in `internal/services/addomain/dept_ou_mapper.go` for AD login
- Error occurs when AD users try to login and system queries `sys_dept_ou_mapping` table

- timestamp: 2026-05-25T16:31:00+08:00
- Root cause identified: The `sys_dept_ou_mapping` table was never created
- The migration exists as a manual Go script but was never executed
- The model exists but is not in AutoMigrate, so GORM won't create it automatically



## Resolution

### Root Cause
The `sys_dept_ou_mapping` table was never created because the `DeptOUMapping` model was not included in the `AutoMigrate()` call in `internal/core/db/database.go`. While a manual migration script existed (`scripts/migrate/126_create_dept_ou_mapping_table.go`), it was never executed, and the model was missing from the automated migration list.

### Fix
Added `&models.DeptOUMapping{}` to the `AutoMigrate()` list in `internal/core/db/database.go` line 273, right after `&models.ADSyncLog{}` and before the operations models comment.

### Files Changed
- `internal/core/db/database.go`: Added `&models.DeptOUMapping{}` to AutoMigrate list (line 273)

### Verification
- Code compiles successfully
- Next time the application starts, GORM will automatically create the `sys_dept_ou_mapping` table
- AD users will be able to login and map to their departments correctly

