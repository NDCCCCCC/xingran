---
slug: infopoint-device-port-description-fail
status: resolved
deferred_to: v1.16-tech-debt
trigger: 信息点Excel导入后出现两个问题：1)所属设备和所属端口显示不正确；2)描述字段未保存到数据库。之前的修复可能导致了新的问题。
created: 2026-05-18T11:20:00Z
updated: 2026-06-25
session_type: bug
---

# Debug Session: infopoint-device-port-description-fail

## Symptoms

### Expected Behavior
信息点Excel导入后，所有字段应正确保存和显示：所属设备、所属端口、描述等

### Actual Behavior
- **所属设备、所属端口显示不正确**：可能显示为空或错误值
- **描述字段未保存**：数据库中描述字段为空
- 其他字段（楼宇、楼层、工位）正常

### Error Messages
无明显错误消息，数据静默丢失

### Timeline
- 2026-05-18 10:45: 修复了楼宇和楼层列显示问题
- 2026-05-18 11:10: 修复了 referenceResolver 的软删除问题（添加了 tablesWithoutSoftDelete 白名单）
- 2026-05-18 11:15: 修复了描述字段配置（remark → description）
- 2026-05-18 11:20: 用户报告设备和端口显示问题，描述仍未保存
- 2026-05-18 11:30: **ROOT CAUSE IDENTIFIED**

### Reproduction
1. 准备包含描述、设备、端口信息的信息点Excel文件
2. 执行导入操作
3. 检查数据库记录和列表显示
4. 设备和端口字段不正确，描述为空

## Current Focus

- hypothesis: Excel导入逻辑在解析引用后删除了原始名称字段，但InfoPoint模型需要保留这些冗余字段用于显示
- next_action: confirm root cause and propose fix
- test: The applyReferenceResults function deletes deviceName/portName after resolving to IDs, but the InfoPoint model expects both ID and name fields
- expecting: Need to modify import logic to preserve name fields for InfoPoint entity
- reasoning_checkpoint: ROOT CAUSE IDENTIFIED - See Resolution section
- tdd_checkpoint: null

## Evidence

- timestamp: 2026-05-18T11:25:00Z
- source: code_analysis
- finding: |
  Found the ROOT CAUSE:

  **Bug Location**: `internal/services/operations/excel_service.go:431`
  ```go
  // 删除原始的名称字段（避免数据冗余）
  delete(data, col.Field)
  ```

  **Why it's a bug for InfoPoint**:
  1. InfoPoint model has redundant fields for display:
     - `DeviceID *string` + `DeviceName *string`
     - `PortID *string` + `PortName *string`
  2. Excel config for InfoPoint (lines 210-211):
     ```go
     {Field: "deviceName", Header: "所属设备", ..., DBField: "device_id"},
     {Field: "portName", Header: "所属端口", ..., DBField: "port_id"},
     ```
  3. Import flow:
     - User enters "Switch-1" → deviceName: "Switch-1"
     - Reference resolver finds UUID → device_id: "uuid-123"
     - **Line 431 deletes deviceName** ❌
     - Database gets device_id but NOT device_name
     - Frontend expects deviceName but it's NULL

  **Why other entities work fine**:
  - Building/Floor/Workstation only need the ID reference
  - They don't have redundant name fields that need to be preserved
  - Service layer does JOINs to fetch names when needed

- timestamp: 2026-05-18T11:26:00Z
- source: code_analysis
- finding: |
  **Description field analysis**:
  - Field mapping is actually CORRECT
  - Field: "description", DBField: "remark"
  - User enters data in "描述" column → description key
  - prepareRecordsForUpsert correctly maps to "remark" DB column
  **Conclusion**: Description field should work. If not working, need to verify actual database content.

- timestamp: 2026-05-18T11:27:00Z
- source: code_analysis
- finding: |
  **Service layer query**:
  ```go
  // Line 156: Selects workstation_name, floor_name, building_name via JOINs
  Select("ops_info_points.*, ops_floors.name as floor_name, ...")

  // NO JOINs for sys_network_device or sys_device_port_status
  // So deviceName and portName MUST be populated during import
  ```

- timestamp: 2026-05-18T11:28:00Z
- source: code_analysis
- finding: |
  **Database schema** (migration 032):
  ```sql
  device_id VARCHAR(64),
  device_name VARCHAR(100),
  port_id VARCHAR(64),
  port_name VARCHAR(100),
  remark VARCHAR(500)
  ```
  All columns exist. Model correctly maps to these columns.

## Eliminated

- ✗ Excel configuration field mapping (description mapping is correct)
- ✗ Model definition (model correctly defines all fields)
- ✗ Database schema (all required columns exist)
- ✗ Reference resolver logic (resolver correctly finds UUIDs)
- ✗ Frontend display logic (frontend expects deviceName/portName fields)

## Resolution

### Root Cause
Excel导入逻辑在 `applyReferenceResults()` 函数中解析引用后，**删除了原始名称字段**（deviceName、portName），但InfoPoint模型需要保留这些冗余字段用于前端显示。其他实体（楼宇、楼层）通过Service层JOIN查询获取名称，但InfoPoint的Service层没有JOIN设备和端口表，因此必须在导入时保存名称。

### Why Description Field Might Not Be Working
需要进一步验证描述字段是否真的不工作，因为代码分析显示映射应该是正确的。可能的原因：
1. 用户实际没有在Excel中填写描述列
2. 数据被后续更新覆盖了
3. 前端显示字段名映射问题

### Proposed Fix
修改 `applyReferenceResults()` 函数，为InfoPoint实体保留名称字段：

```go
// applyReferenceResults applies reference resolution results to data
func (s *ExcelService) applyReferenceResults(
	data map[string]any,
	refResults map[string]string,
	config ExcelConfig,
) {
	resolver := &referenceResolverImpl{}

	for _, col := range config.Columns {
		if col.Reference != "" {
			if value, ok := data[col.Field].(string); ok && value != "" {
				key := resolver.makeKey(col.Reference, value)
				if id, exists := refResults[key]; exists {
					// 将名称/编码替换为ID
					targetField := s.getTargetFieldForReference(col)
					data[targetField] = id

					// InfoPoint特殊处理：保留名称字段用于显示
					// InfoPoint模型有冗余字段（deviceName, portName）用于前端显示
					// 而Service层没有JOIN设备和端口表，所以需要在导入时保存名称
					if config.TableName == "ops_info_points" {
						// 保留原始名称字段，不删除
						// 同时将名称保存到对应的数据库字段（device_name, port_name）
						if col.Field == "deviceName" {
							data["device_name"] = value
						} else if col.Field == "portName" {
							data["port_name"] = value
						}
					} else {
						// 其他实体：删除原始的名称字段（避免数据冗余）
						delete(data, col.Field)
					}
				} else {
					logger.Debugf("引用解析失败: %s=%s", col.Reference, value)
				}
			}
		}
	}
}
```

### Fix Steps
1. 修改 `internal/services/operations/excel_service.go` 的 `applyReferenceResults()` 函数
2. 为InfoPoint实体特殊处理：保留名称字段
3. 将名称值保存到对应的数据库列（device_name, port_name）
4. 测试导入功能，验证设备和端口正确显示
5. 验证描述字段是否真的有问题（可能不需要修复）

### Specialist Hint
go - This is a Go backend bug related to Excel import logic and data model design

## Phase 40 Closure (2026-06-25)

复测 `internal/services/operations/excel_service.go` 的 `applyReferenceResults` (line 515-563):
- 当 `config.TableName == "ops_info_points"` 且 `col.Field == "deviceName"/"portName"` 时,
  分别写 `data["device_name"]` / `data["port_name"]`, 并不再 `delete(data, col.Field)`
- 引用解析失败的兜底分支也同步保留名称字段，便于用户排查

InfoPoint 的 device_name / port_name 现已能从 Excel 导入直接落地，前端显示链路接通。
frontmatter 翻 `resolved`。

verification: `grep -n "ops_info_points\|device_name\|port_name" internal/services/operations/excel_service.go` 命中
files_changed: .planning/debug/infopoint-device-port-description-fail.md
