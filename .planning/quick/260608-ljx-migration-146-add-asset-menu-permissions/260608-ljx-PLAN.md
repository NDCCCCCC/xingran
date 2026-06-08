---
quick_id: "260608-ljx"
description: "修复 migration_146_add_asset_menu_permissions.go 编译错误"
created: "2026-06-08"
status: "in_progress"
---

# 计划：修复 migration_146_add_asset_menu_permissions.go 编译错误

## 问题分析

编译错误原因：
1. `models.Menu` 结构体已移除 `IsFrame` 和 `IsCache` 字段，现在使用 `Meta` JSONB 字段
2. `assetMenu.ID` 是 `string` 类型，但 `ParentID` 字段需要 `*string` 类型

## 任务

### 任务 1：修复 Menu 结构体字段引用

**文件**: `internal/core/db/migrations/migration_146_add_asset_menu_permissions.go`

**操作**:
1. 删除第 33-34 行的 `IsFrame: 0,` 和 `IsCache: 0,` 字段
2. 删除第 71-72 行的 `IsFrame: 1,` 和 `IsCache: 0,` 字段
3. 修复第 68 行：将 `ParentID: assetMenu.ID,` 改为 `ParentID: &assetMenu.ID,`

**验证**:
- 运行 `go build ./...` 确保没有编译错误
- 运行 `go run .\cmd\main.go` 确保应用程序可以正常启动

**完成标准**:
- 编译成功，无错误
- 迁移文件逻辑保持不变
