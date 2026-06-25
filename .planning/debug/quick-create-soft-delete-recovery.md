---
slug: quick-create-soft-delete-recovery
status: complete
trigger: "快速创建设备时 IP 地址重复错误 - 设备已被软删除，应该恢复而不是报错"
created: "2026-05-08"
updated: "2026-05-08"
---

## Symptoms

- **Expected**: 如果设备被软删除，应该更新设备信息并恢复删除状态；如果设备正常存在，则提示设备已存在
- **Actual**: 直接尝试 INSERT 导致唯一约束冲突 `duplicate key value violates unique constraint "idx_sys_network_device_ip_address"`
- **Timeline**: 修复 CreatedBy 验证问题后出现的新问题
- **Reproduction**: 快速创建 IP 为 10.62.63.8 的设备（该设备已被软删除）
- **Error Message**: ERROR: duplicate key value violates unique constraint "idx_sys_network_device_ip_address" (SQLSTATE 23505)
- **Context**: IP 10.62.63.8 的设备确实存在但已被软删除（deleted_at IS NOT NULL）

## Current Focus

- hypothesis: null
- next_action: null
- test: null
- expecting: null
- reasoning_checkpoint: null
- tdd_checkpoint: null

## Evidence

- timestamp: 2026-05-08T10:30:00Z
  source: code_analysis
  detail: "QuickCreateDevice 函数 line 280: `if err == nil && !existingDevice.DeletedAt.Valid`"
  finding: "条件判断错误 - `DeletedAt.Valid` 为 true 时表示记录已被软删除，应该进入恢复分支"

- timestamp: 2026-05-08T10:31:00Z
  source: gormDocumentation
  detail: "gorm.DeletedAt.Valid 字段：true = 已删除，false = 未删除"
  reference: "https://gorm.io/docs/delete.html#Soft-Delete"

- timestamp: 2026-05-08T10:32:00Z
  source: code_flow_analysis
  detail: "代码执行流程："
  flow: |
    1. Line 225: Unscoped().Where().First() 查询包括已删除记录
    2. Line 228: 检查设备是否已存在且未删除 - 正确
    3. Line 280: 检查设备是否存在且已删除 - 条件错误！
       - 当前: `!existingDevice.DeletedAt.Valid` (未删除时才进入)
       - 应该: `existingDevice.DeletedAt.Valid` (已删除时才进入)
    4. Line 304: 创建新设备 - 因 line 280 条件错误而执行，导致唯一约束冲突

- timestamp: 2026-05-08T10:35:00Z
  source: fix_applied
  detail: "修复 line 280 条件判断：`if err == nil && existingDevice.DeletedAt.Valid`"
  result: "代码编译成功，等待用户测试验证"

## Eliminated

- timestamp: 2026-05-08T10:25:00Z
  hypothesis: "IP 地址唯一约束配置问题"
  evidence: "uniqueIndex 配置正确，问题在于软删除记录的处理逻辑"
  method: "检查 network_device.go model 定义"

- timestamp: 2026-05-08T10:26:00Z
  hypothesis: "Unscoped() 查询未包含软删除记录"
  evidence: "Unscoped() 使用正确，能查询到已删除记录"
  method: "代码审查 line 225"

## Resolution

- root_cause: "QuickCreateDevice 函数 line 280 条件判断错误，使用 `!existingDevice.DeletedAt.Valid` 导致软删除恢复逻辑永远不会执行，而是尝试创建新记录导致唯一约束冲突"

- fix: "将 `internal/services/network_device_service.go` line 280 的条件从 `if err == nil && !existingDevice.DeletedAt.Valid` 改为 `if err == nil && existingDevice.DeletedAt.Valid`"

- verification: |
  1. ✅ 代码编译成功 (go build ./...)
  2. ⏳ 用户测试：快速创建已软删除的设备，确认设备被恢复且不再出现唯一约束冲突

- files_changed:
  - internal/services/network_device_service.go (line 280)
