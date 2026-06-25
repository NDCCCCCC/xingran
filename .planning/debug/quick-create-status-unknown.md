---
slug: quick-create-status-unknown
status: resolved
trigger: "快速创建设备成功后，设备状态显示为'未知' (status=2)，期望是正常 (status=0)"
created: "2026-05-08"
updated: "2026-05-08"
---

## Symptoms

- **Expected**: 快速创建成功后，设备状态应为 `0`（正常/在线）
- **Actual**: 设备状态显示为 `2`（未知）
- **Timeline**: 快速创建功能修复后发现的问题
- **Reproduction**: 快速创建设备 → 创建成功 → 查看设备状态为"未知"
- **Context**: 设备功能正常，未知状态不影响使用
- **Status Convention**: 0=在线, 1=离线, 2=未知

## Current Focus

- hypothesis: null
- next_action: null
- test: null
- expecting: null
- reasoning_checkpoint: null
- tdd_checkpoint: null

## Evidence

- timestamp: 2026-05-08T10:30:00Z
  source: code_review
  finding: |
    在 internal/services/network_device_service.go 的 QuickCreateDevice 函数中发现：
    - 第293行（恢复已删除设备）: existingDevice.Status = models.DeviceStatusUnknown
    - 第320行（创建新设备）: Status: models.DeviceStatusUnknown
    应该改为: models.DeviceStatusOnline (值为0)

- timestamp: 2026-05-08T10:31:00Z
  source: model_constants
  finding: |
    DeviceStatus 常量定义（internal/models/network_device.go）:
    - DeviceStatusOnline = 0  // 在线
    - DeviceStatusOffline = 1 // 离线
    - DeviceStatusUnknown = 2 // 未知

- timestamp: 2026-05-08T10:32:00Z
  source: build_verification
  finding: |
    修复后编译验证通过：go build ./internal/services/...
    无编译错误

## Eliminated

## Resolution

- root_cause: QuickCreateDevice 函数中硬编码了 Status = models.DeviceStatusUnknown (值为2)，而不是 DeviceStatusOnline (值为0)
- fix: 将 internal/services/network_device_service.go 第293行和第320行的 models.DeviceStatusUnknown 改为 models.DeviceStatusOnline
- verification: |
  1. 代码修复：两处状态赋值已从 DeviceStatusUnknown 改为 DeviceStatusOnline
  2. 编译验证：go build ./internal/services/... 通过，无错误
  3. 待测试验证：实际执行快速创建操作，确认设备状态显示为"在线"(0)
- files_changed: internal/services/network_device_service.go
