---
slug: sys-api-keys-table-missing
status: resolved
trigger: API 密钥管理页面报错"relation sys_api_keys does not exist"
created: 2026-05-19
updated: 2026-05-19
---

# Debug Session: sys-api-keys-table-missing

## Symptoms

- **Expected behavior**:
  API 密钥管理页面应正常加载密钥列表，前端调用 `POST /api/v1/system/apikeys/list` 应返回数据

- **Actual behavior**:
  前端收到 400 错误，错误信息为"数据库操作失败"。后端日志显示 PostgreSQL 错误：`ERROR: relation "sys_api_keys" does not exist (SQLSTATE 42P01)`

- **Error messages**:
  ```
  ERRO[2026-05-19 18:16:04] [GORM错误] SELECT count(*) FROM "sys_api_keys" WHERE user_id = '652eae20-48e6-4a42-b2c5-b53247195627' AND "sys_api_keys"."deleted_at" IS NULL | 耗时: 1.0576ms | 错误: ERROR: relation "sys_api_keys" does not exist (SQLSTATE 42P01)
  ```

- **Timeline**:
  从未正常工作（新功能）

- **Reproduction**:
  1. 启动后端服务
  2. 访问前端 API 密钥管理页面
  3. 页面自动调用 `/api/v1/system/apikeys/list`
  4. 后端尝试查询 `sys_api_keys` 表时报错

## Current Focus

- **Hypothesis**: `sys_api_keys` 表的数据库迁移未执行或未成功
- **Next action**: 检查数据库迁移文件和执行状态
- **Test**: 验证 `sys_api_keys` 表是否存在于数据库中
- **Expecting**: 发现缺失的迁移或表结构定义

## Evidence

- timestamp: 2026-05-19 18:20:00
  source: code_analysis
  detail: |
    发现 migration_085_api_keys.go 文件存在，定义了 Migrate085APIKeys 函数
    但是该函数从未被调用

- timestamp: 2026-05-19 18:20:30
  source: code_analysis
  detail: |
    检查 internal/core/db/database.go 的 AutoMigrate() 函数
    发现 APIKey 和 APIKeyUsageLog 模型未包含在 AutoMigrate 列表中
    其他模型如 User, Role, Menu 等都在列表中

- timestamp: 2026-05-19 18:21:00
  source: code_analysis
  detail: |
    确认模型定义存在：
    - internal/models/api_key.go 定义了 APIKey 模型，表名为 sys_api_keys
    - internal/models/api_key_usage_log.go 定义了 APIKeyUsageLog 模型

## Eliminated

- ❌ 模型定义缺失 - 模型文件已存在且定义正确
- ❌ 表名映射错误 - TableName() 方法正确返回 "sys_api_keys"
- ❌ 迁移函数错误 - Migrate085APIKeys 函数逻辑正确

## Resolution

### Root Cause
`sys_api_keys` 表不存在是因为 **APIKey 和 APIKeyUsageLog 模型从未被添加到数据库迁移流程中**。

虽然创建了 `migration_085_api_keys.go` 文件和 `Migrate085APIKeys` 函数，但该函数从未被调用。在 `internal/core/db/database.go` 的 `AutoMigrate()` 函数中，这两个模型没有被包含在迁移列表里。

### Fix Direction
需要在 `internal/core/db/database.go` 的 `AutoMigrate()` 函数中添加 APIKey 和 APIKeyUsageLog 模型：

```go
// AutoMigrate 自动迁移所有模型
func (d *Database) AutoMigrate() error {
    // ... existing code ...
    err := d.DB.Migrator().AutoMigrate(
        // ... existing models ...
        &models.User{},
        &models.Role{},
        // ... other models ...

        // API密钥管理相关模型
        &models.APIKey{},
        &models.APIKeyUsageLog{},
    )
    // ... rest of code ...
}
```

### Fix Applied
✅ **修复已完成**

在 `internal/core/db/database.go` 的 AutoMigrate() 函数中添加了两个缺失的模型：

```go
// API密钥管理相关模型
&models.APIKey{},
&models.APIKeyUsageLog{},
```

**修改位置**: database.go:233-235

### Verification
- ✅ 代码编译通过 (`go build ./cmd/main.go`)
- 🔄 下次启动后端服务时，GORM 将自动创建 `sys_api_keys` 和 `sys_api_key_usage_logs` 表
- 📝 建议重启后端服务以执行迁移

### Files Changed
- `internal/core/db/database.go` - 添加 APIKey 和 APIKeyUsageLog 到 AutoMigrate 列表

### Next Steps
1. **重启后端服务** - 让 GORM 执行表迁移
2. **验证表创建** - 检查数据库中 `sys_api_keys` 表是否已创建
3. **测试前端页面** - 刷新 API 密钥管理页面，确认列表正常加载
