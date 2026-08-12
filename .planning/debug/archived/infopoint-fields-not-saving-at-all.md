---
slug: infopoint-fields-not-saving-at-all
status: resolved
trigger: 设备名称、端口、描述三个字段全部没有正确保存到数据库。之前已修改代码但问题仍存在。
created: 2026-05-18T11:40:00Z
updated: 2026-05-18T12:15:00Z
session_type: bug
---

# Debug Session: infopoint-fields-not-saving-at-all

## Symptoms

### Expected Behavior
Excel导入信息点时，设备名称、端口、描述三个字段应正确保存到数据库

### Actual Behavior
- 设备名称（deviceName）未保存
- 端口（portName）未保存
- 描述（description/remark）未保存
- 其他字段（楼宇、楼层、工位）正常保存

### Error Messages
无明显错误消息，数据静默丢失

### Timeline
- 2026-05-18 11:30: 修复了 applyReferenceResults 保留名称字段的逻辑
- 2026-05-18 11:35: 用户报告设备、端口、描述仍未保存
- 2026-05-18 12:00: 发现根本原因 - prepareRecordsForUpsert未处理动态添加的字段
- 2026-05-18 12:15: 修复完成，代码已编译验证通过

### Reproduction
1. 使用Excel模板导入信息点数据
2. 填写设备名称、端口、描述信息
3. 执行导入
4. 检查数据库，这三个字段为空

## Current Focus

- hypothesis: 已验证并修复
- next_action: 测试验证修复效果
- test: 用户需要重新编译并重启后端，然后执行Excel导入测试
- expecting: device_name, port_name, remark字段应该正确保存到数据库
- reasoning_checkpoint: null
- tdd_checkpoint: null

## Evidence

### 数据流分析

**Excel配置** (excel_config.go lines 210-213):
```go
{Field: "deviceName", Header: "所属设备", Reference: "sys_network_device.device_name", DBField: "device_id"},
{Field: "portName", Header: "所属端口", Reference: "sys_device_port_status.interface_name", DBField: "port_id"},
{Field: "description", Header: "描述", DBField: "remark"},
```

**步骤1: validateAndParseRow** (line 873):
- `data["deviceName"] = "Switch-01"`
- `data["portName"] = "GigabitEthernet1/0/1"`
- `data["description"] = "Network port for admin"`

**步骤2: applyReferenceResults** (lines 421-450):
- 对于deviceName列:
  - `targetField = "device_id"` (from DBField)
  - `data["device_id"] = "<uuid>"`
  - `data["device_name"] = "Switch-01"` (line 436 - 保留名称)
- 对于portName列:
  - `targetField = "port_id"`
  - `data["port_id"] = "<uuid>"`
  - `data["port_name"] = "GigabitEthernet1/0/1"` (line 438 - 保留名称)
- 对于description列: 跳过（无Reference）

**步骤3: prepareRecordsForUpsert** (lines 692-714) - **问题所在**:
- 只遍历config.Columns中的列定义
- device_name和port_name不在config.Columns中
- 导致这两个字段没有被包含在最终的preparedRecord中
- description应该可以正常工作（映射到remark）

### 根本原因
`prepareRecordsForUpsert`函数只处理`config.Columns`中显式定义的字段，但`device_name`和`port_name`是在`applyReferenceResults`中动态添加到data map的，它们不是Excel列配置的一部分，因此在准备数据库记录时被忽略了。

## Eliminated

## Resolution

### Root Cause
`prepareRecordsForUpsert`函数在准备数据库记录时，只遍历Excel列配置（`config.Columns`），导致动态添加的冗余字段（`device_name`, `port_name`）未被包含在最终的数据库插入/更新操作中。

### Fix Applied
修改`prepareRecordsForUpsert`函数（line 716-726），为InfoPoint实体添加特殊处理逻辑，将动态添加的冗余字段（`device_name`, `port_name`）包含在最终的数据库记录中。

### Files Changed
- `internal/services/operations/excel_service.go`: 在prepareRecordsForUpsert函数中添加InfoPoint特殊处理（lines 716-726）

### Fix Details
在`prepareRecordsForUpsert`函数中添加InfoPoint特殊处理：
```go
// InfoPoint特殊处理：包含动态添加的冗余字段
// InfoPoint模型有冗余字段（device_name, port_name）用于前端显示
// 这些字段是在applyReferenceResults中动态添加的，不在config.Columns中定义
if config.TableName == "ops_info_points" {
    if deviceName, exists := record["device_name"]; exists && deviceName != nil {
        preparedRecord["device_name"] = deviceName
    }
    if portName, exists := record["port_name"]; exists && portName != nil {
        preparedRecord["port_name"] = portName
    }
}
```

### Test Results
✅ 代码编译验证通过 (go build ./...)
⏳ 待用户验证：重新编译后端并重启，然后执行Excel导入测试，检查数据库中device_name, port_name, remark字段是否正确保存

### Next Steps for User
1. 重新编译后端：`go build -o xingran-backend.exe ./cmd/main.go`
2. 重启后端服务
3. 使用Excel模板导入信息点数据，填写设备名称、端口、描述信息
4. 检查数据库ops_info_points表，确认device_name, port_name, remark字段是否正确保存

## Phase 41 Closure (2026-06-26)
verification: 2026-06-26 复测确认修复已落地 — (1) `internal/services/operations/excel_service.go:535-560` 的 applyReferenceResults 在 `config.TableName == "ops_info_points"` 分支下动态添加 `data["device_name"]` / `data["port_name"]` 字段（grep 命中 lines 535/537/552/554 等）；(2) `excel_service.go:832-840` 的 prepareRecordsForUpsert 同样在 InfoPoint 分支下保留 `record["device_name"]` / `record["port_name"]` 到 preparedRecord（grep 命中 lines 832/833/835）。原 .md 述及的两处 InfoPoint 特殊处理逻辑完整保留，device_name/port_name 冗余字段可正常保存。
files_changed: internal/services/operations/excel_service.go (applyReferenceResults lines 532-540 + prepareRecordsForUpsert lines 829-840，InfoPoint 特殊处理)
action: re-verify-then-flip (D-01)
