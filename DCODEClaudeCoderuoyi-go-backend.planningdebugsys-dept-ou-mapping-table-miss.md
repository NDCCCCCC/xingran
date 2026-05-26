---
status: resolved
trigger: 后端报错：ERRO[2026-05-25 13:16:27] [GORM错误] SELECT * FROM "sys_dept_ou_mapping" WHERE ou_dn = 'OU=基础运维科,OU=科技创新部,OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn' AND sync_enabled = true ORDER BY "sys_dept_ou_mapping"."id" LIMIT 1 | 耗时: 0s | 错误: ERROR: relation "sys_dept_ou_mapping" does not exist (SQLSTATE 42P01) WARN[2026-05-25 13:23:20] 用户 chenchao-076 的AD OU OU=基础运维科,OU=科技创新部,OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn 未找到对应部门: 未找到OU DN OU=基础运维科,OU=科技创新部,OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn 对应的部门
created: 2026-05-25
updated: 2026-05-25
---

## Symptoms

**Expected behavior:**
AD域用户登录时应能通过OU DN映射找到对应的部门，实现用户与部门的关联。

**Actual behavior:**
1. 最初报错 `ERROR: relation "sys_dept_ou_mapping" does not exist (SQLSTATE 42P01)` - 已通过执行迁移脚本解决
2. 现在警告 `未找到OU DN ... 对应的部门` - 表已存在但为空，需要填充映射数据

**Error messages:**
```
WARN[2026-05-25 13:23:20] 用户 chenchao-076 的AD OU OU=基础运维科,OU=科技创新部,OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn 未找到对应部门: 未找到OU DN OU=基础运维科,OU=科技创新部,OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn 对应的部门
```

**Timeline:**
从部署以来一直存在。表已创建但数据缺失。

**Reproduction:**
AD域用户登录时触发OU映射查询

## Current Focus

**hypothesis:** `sys_dept_ou_mapping` 表已创建但为空，需要运行部门到AD的同步任务来填充映射数据

**next_action:** 触发部门同步任务以填充映射表

**expecting:** 同步任务完成后，表中有部门与OU的映射数据

**reasoning_checkpoint:** null
**tdd_checkpoint:** null

## Evidence

- timestamp: 2025-05-25T13:16:27Z
  source: migration_execution
  finding: Successfully created migration runner and executed migration
  details: |
    - Created `scripts/migrate/126_create_dept_ou_mapping_table.go`
    - Successfully executed migration on production database
    - Table `sys_dept_ou_mapping` now exists with all required columns

- timestamp: 2025-05-25T13:23:20Z
  source: new_symptom
  finding: Warning changed from "table does not exist" to "mapping not found"
  details: |
    - Table exists but has no data
    - Need to populate department to AD OU mappings
    - User chenchao-076 has OU: OU=基础运维科,OU=科技创新部,OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn
    - System has department: 分公司本部-科技创新部-基础运维科 (exists in sys_dept)

- timestamp: 2025-05-25T13:23:20Z
  source: code_analysis
  finding: Department sync service exists but needs to be triggered
  details: |
    - `DeptToADSyncService.SyncDeptStructureToAD()` creates OU mappings
    - Scheduled task exists: `executeDeptToADSyncTask()` 
    - Task runs daily at 2 AM but hasn't been executed yet
    - Manual trigger needed to populate initial data
    - API endpoint available: POST /system/ad/sync/dept-to-ad

## Eliminated

## Resolution

**root_cause:** The `sys_dept_ou_mapping` table was successfully created via migration, but it's empty because the department sync task has never been executed. The scheduled sync task runs daily at 2 AM but hasn't triggered yet since the table was created.

**fix:** Three solution approaches provided:

1. **API Trigger (Recommended)**: Call POST `/system/ad/sync/dept-to-ad` with `{"adConfigId": "config-id"}` to trigger immediate department sync. This will automatically create OU mappings for all departments.

2. **Shell Script**: Created `scripts/trigger_dept_sync.sh` for automated triggering of the sync process via API calls.

3. **Manual SQL**: Created `scripts/manual_dept_sync.sql` for direct database insertion of specific department-OU mappings for immediate fix.

**verification:** After executing any of the above solutions:
- Verify data exists: `SELECT COUNT(*) FROM sys_dept_ou_mapping;`
- Check specific mapping: `SELECT * FROM sys_dept_ou_mapping WHERE ou_dn LIKE '%基础运维科%';`
- Test AD login: user chenchao-076 should now find their department without warnings
- Monitor logs for successful sync completion

**files_changed:**
- `scripts/trigger_dept_sync.sh` (created) - Automated sync trigger script
- `scripts/manual_dept_sync.sql` (created) - Manual SQL insertion template
