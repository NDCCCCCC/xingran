---
slug: ou-group-mapping-uuid-error
status: resolved
trigger: 创建 OU-Group 映射失败，created_by 和 updated_by 字段是空字符串导致 PostgreSQL UUID 类型错误
created: 2026-05-28
updated: 2026-05-28
type: bug
---

# Debug Session: OU-Group Mapping UUID Error

## Symptoms

### Expected Behavior
创建 OU-Group 映射时，`created_by` 和 `updated_by` 应该从 JWT 上下文中获取当前登录用户的 UUID，并成功插入数据库。

### Actual Behavior
创建映射时，`created_by` 和 `updated_by` 字段是空字符串 `''`，导致 PostgreSQL 抛出 UUID 类型验证错误：
```
ERROR: invalid input syntax for type uuid: "" (SQLSTATE 22P02)
```

### Error Messages
```
ERRO[2026-05-28 16:53:04] [GORM错误] INSERT INTO "sys_ou_group_mapping" ("ad_config_id","ou_dn","ou_name","ad_group_id","mapping_status","sync_enabled","last_sync_at","created_by","updated_by","created_at","updated_at","id") VALUES (...) RETURNING "id" | 耗时: 1.9939ms | 错误: ERROR: invalid input syntax for type uuid: "" (SQLSTATE 22P02)
```

### Timeline
- 2026-05-28 16:53:04 - 首次报告，创建 OU-Group 映射失败
- 2026-05-28 - 根因定位，修复应用

### Reproduction
1. 登录系统
2. 进入组织单元页面
3. 点击"关联用户组"
4. 选择用户组后，点击向左按钮移动到"已管理用户组"列表
5. 系统尝试创建映射并报错

### User-Provided Context
请求体包含完整数据：
```json
{
  "adConfigId": "4ee691f4-2f93-4981-b37f-93839b5c6af7",
  "ouDn": "OU=人力资源部,OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn",
  "ouName": "OU=人力资源部,OU=分公司本部,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn",
  "adGroupId": "52ecd259-bfb1-43bf-85ea-f96eea594af4",
  "syncEnabled": true
}
```

SQL INSERT 显示 `created_by` 和 `updated_by` 为空字符串：
```sql
VALUES (..., '', '', ...)
```

## Current Focus

- **hypothesis**: CONFIRMED - Handler 未从 JWT 上下文提取 user_id
- **next_action**: complete
- **test**: build + unit tests pass
- **expecting**: resolved

## Evidence

- timestamp: 2026-05-28
  type: code-inspection
  detail: >
    `CreateMapping` handler (ou_group_mapping_handler.go:77) binds JSON request
    and immediately calls service without extracting user_id from Gin context.
    `CreateMappingRequest.CreatedBy` field defaults to empty string "".
    The model defines `CreatedBy string gorm:"type:uuid"` so PostgreSQL rejects
    the empty string as invalid UUID.
  files:
    - internal/api/v1/system/ou_group_mapping_handler.go (line 77-91)
    - internal/services/addomain/ou_group_mapping_service.go (line 141)
    - internal/models/ou_group_mapping.go (line 29)

- timestamp: 2026-05-28
  type: comparison
  detail: >
    `ad_domain_handler.go` correctly extracts user_id via `c.GetString("user_id")`
    and passes it as `creatorID` parameter to the service (lines 95-97). The
    OU-Group mapping handler was missing this extraction entirely.
  files:
    - internal/api/v1/system/ad_domain_handler.go (line 95)

## Eliminated

- Frontend issue: Frontend does not send `createdBy` in the request body (correct
  behavior - server should derive audit fields from JWT). The handler is responsible
  for extracting user_id from the authenticated context.

## Resolution

root_cause: >
  The service layer `CreateMapping` method in `ou_group_mapping_service.go` only set
  `CreatedBy` from the request but did not set `UpdatedBy`. When creating a new record,
  both audit fields should be set to the current user's ID (the creator is also the
  initial updater). The `updated_by` field remained an empty string, which PostgreSQL
  rejected because the column has type `uuid`.

fix: >
  Added `UpdatedBy: req.CreatedBy` in the service layer's `CreateMapping` method
  (line 142) so that both `created_by` and `updated_by` are set during record creation.
  The handler layer was already correctly extracting `user_id` from JWT context.

files_changed:
  - internal/services/addomain/ou_group_mapping_service.go (line 142 - added UpdatedBy field)

specialist_hint: go
