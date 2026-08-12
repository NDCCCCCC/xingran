# VDI服务器表缺少last_sync_time字段修复

## 概述

VDI同步时出现数据库错误，提示 `last_sync_time` 字段不存在。需要添加该字段到VDIServer模型并创建数据库迁移。

## 问题描述

```
ERROR: column "last_sync_time" of relation "sys_vdi_server" does not exist (SQLSTATE 42703)
```

**影响**: VDI同步功能无法正常更新同步时间

## 修复步骤

### 1. 检查VDIServer模型定义

**文件**: `internal/models/vdi.go`

检查VDIServer结构体是否包含LastSyncTime字段。

### 2. 添加LastSyncTime字段（如果缺失）

在VDIServer结构体中添加：
```go
LastSyncTime *time.Time `gorm:"column:last_sync_time" json:"lastSyncTime,omitempty"`
```

### 3. 创建数据库迁移

**目录**: `internal/core/db/migrations/`

创建新迁移文件（格式：`085_add_vdi_last_sync_time.up.sql`）：
```sql
-- 添加 last_sync_time 字段到 sys_vdi_server 表
ALTER TABLE sys_vdi_server
ADD COLUMN last_sync_time TIMESTAMP NULL;
```

创建对应的down迁移：
```sql
-- 移除 last_sync_time 字段
ALTER TABLE sys_vdi_server
DROP COLUMN IF EXISTS last_sync_time;
```

### 4. 验证修复

- 运行 `go build ./...` 确保编译通过
- 重启服务，触发自动迁移
- 手动触发VDI同步，验证不再出现错误
- 检查数据库表中last_sync_time字段已正确创建

## 预期效果

- ✅ VDI同步能正常更新 last_sync_time
- ✅ 不再出现数据库字段不存在的错误
- ✅ 数据库迁移成功执行
- ✅ VDI同步日志不再显示错误
