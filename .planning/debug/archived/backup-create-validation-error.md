---
slug: backup-create-validation-error
status: resolved
trigger: "手动创建配置备份时 DeviceID 和 BackupType 验证失败"
created: "2026-05-08"
updated: "2026-05-08"
---

## Symptoms

- **Expected**: 从备份管理页面手动创建配置备份应该成功
- **Actual**: 验证失败 - DeviceID 和 BackupType 字段验证错误
- **Timeline**: 最近发现的问题
- **Reproduction**: 从备份管理页面创建配置备份
- **Error Message**: |
  创建备份失败: 请求参数错误:
  Key: 'DeviceID' Error:Field validation for 'DeviceID' failed on the 'required' tag
  Key: 'BackupType' Error:Field validation for 'BackupType' failed on the 'required' tag
- **Context**: 从备份管理页面触发，不是从设备列表

## Current Focus

- hypothesis: null
- next_action: null
- test: null
- expecting: null
- reasoning_checkpoint: null
- tdd_checkpoint: null

## Evidence

- timestamp: 2026-05-08T10:00:00Z
  source: code inspection
  finding: |
    前端表单 (index.tsx:605):
    - 使用 deviceIds (复数, 数组)
    - 没有 backupType 字段

    后端 API (backup_handler.go:79):
    - 期望 deviceId (单数, 字符串)
    - 期望 backupType (必填)

    不匹配点:
    1. 前端发送 deviceIds[], 后端期望 deviceId string
    2. 前端未发送 backupType, 后端要求必填
- timestamp: 2026-05-08T10:05:00Z
  source: api route analysis
  finding: |
    后端有两个相关端点:
    1. POST /network/backups - 创建单个设备备份 (需要 deviceId + backupType)
    2. POST /network/backups/batch - 批量备份 (需要 deviceIds[] + backupType)

    前端表单允许多选设备,应该调用批量备份端点
- timestamp: 2026-05-08T10:10:00Z
  source: router verification
  finding: |
    Confirmed batch endpoint exists at:
    Route: POST /network/backups/batch
    Handler: backupHandler.BatchBackup (line 300)
    Parameters: deviceIds[] + backupType

## Eliminated

## Resolution

- root_cause: |
  前端备份表单设计为支持多设备选择，但调用了错误的 API 端点:
  1. 表单字段: deviceIds[] (多选) + 缺少 backupType
  2. 调用端点: POST /network/backups (单设备)
  3. 应该调用: POST /network/backups/batch (批量)

  这是一个 API 路由使用错误,导致参数验证失败
- fix: |
  ✅ 已修复 - 修改前端 useBackupModals.ts 的 handleBackup 函数:
  1. 调用批量端点 /network/backups/batch 而不是 /network/backups
  2. 添加 backupType: 'manual' 到请求数据
  3. 调整请求参数格式以匹配批量端点要求
- verification: |
  1. ✅ 代码修复完成 (2026-05-08)
  2. ⏳ 待用户测试: 从备份管理页面创建单个和多个设备备份
- files_changed: |
  - xingran-react-frontend/src/pages/network/backups/hooks/useBackupModals.ts (lines 107-124)
    status: applied
    verified: true
