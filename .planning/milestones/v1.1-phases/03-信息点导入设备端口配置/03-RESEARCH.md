# Phase 3: 信息点导入设备端口配置 - Research

**Researched:** 2026-04-16
**Domain:** Excel import configuration, name-to-ID reference resolution
**Confidence:** HIGH

## Summary

This phase adds two optional Excel columns ("所属设备" and "所属端口") to the infoPoint import configuration in `excel_config.go`. The implementation is a direct application of the existing `ExcelColumn.Reference` pattern, already proven by the workstation config's `deptName`/`userName` columns (Phase 1-2). No model changes, no migrations, no frontend changes, and no new files are required. The entire change is mechanical: add two `ExcelColumn` entries to the `infoPoint` config block at lines 197-213 of `excel_config.go`.

The `ReferenceResolver` already supports the `"table.field"` format for batch name-to-ID resolution. The `OpsInfoPoint` model already has `DeviceID`/`DeviceName`/`PortID`/`PortName` fields. The batch upserter's `partialUpdate` mode (enabled for infoPoint) will skip these fields when they are empty, ensuring backward compatibility.

**Primary recommendation:** Add two `ExcelColumn` entries to the `infoPoint` config in `excel_config.go` using `Reference: "sys_network_device.device_name"` for device and `Reference: "sys_device_port_status.interface_name"` for port. No other files need modification.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** 设备名称精确匹配 `sys_network_device.device_name`（与 workstation 中 deptName/userName 的 Reference 模式一致）
- **D-02:** 端口名称精确匹配 `sys_device_port_status.interface_name`
- **D-03:** 两个字段均为可选（Required: false），匹配失败留空不阻断导入
- **D-04:** 不做端口与设备的级联验证（Out of Scope）

### Claude's Discretion
- Reference 配置格式、列顺序、Header 命名等细节由 Claude 决定
- 端口匹配使用全局查找（不限设备），因 Reference 模式不支持 DependsOn 级联到 device_id -> port_id

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| IMPORT-05 | Excel import config add "所属设备" column, match device name to `sys_network_device.name`, write `device_id` | Reference pattern `"sys_network_device.device_name"` with `DBField: "device_id"` -- verified model has `DeviceName` field at `network_device.go:36` |
| IMPORT-06 | Excel import config add "所属端口" column, match port name to `sys_device_port_status.port_name`, write `port_id` | Reference pattern `"sys_device_port_status.interface_name"` with `DBField: "port_id"` -- verified model has `InterfaceName` field at `device_port_status.go:33` |
| IMPORT-07 | Both fields optional, match failure leaves empty without blocking import | Proven by workstation's `deptName`/`userName` pattern -- `Required: false` + `PartialUpdate: true` in batch_upserter skips empty fields |
| VAL-03 | After import, infoPoint list correctly shows device name and port name | `infopoint_service.go:62-76` `populateRedundantFields()` already fills `DeviceName` and `PortName` from `DeviceID`/`PortID` -- no changes needed |
| VAL-04 | Export includes device and port info (verify no regression) | `excel_export_config.go:338-345` already has export columns for `deviceId`/`deviceName`/`portId`/`portName` -- no changes needed |
</phase_requirements>

## Standard Stack

### Core (No installation needed -- all already in project)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| excelize/v2 | v2.10.0 | Excel read/write | Project's standard for all import/export |
| GORM | v1.30.5 | ORM | Reference resolver uses GORM for batch queries |
| gorm/postgres | v1.5.9 | PostgreSQL driver | Reference resolver queries via GORM |

### No new dependencies required

This phase modifies only configuration data (Go struct literal). No new libraries or packages are needed.

## Architecture Patterns

### Pattern 1: ExcelColumn Reference Pattern (VERIFIED -- existing pattern)

**What:** Define an `ExcelColumn` with `Reference: "table.field"` and `DBField: "target_column"` to enable automatic name-to-ID resolution during import.

**When to use:** When an Excel column contains a human-readable name that must be resolved to a database UUID.

**How it works (verified from code):**

1. `excel_service.go:276` -- `extractReferenceRequests()` collects all columns with `Reference != ""` and non-empty values into `ReferenceRequest` objects.
2. `excel_service.go:284-296` -- Independent references (no `DependsOn`) are resolved via `ResolveBatch()`, which groups by reference type and executes one SQL query per group.
3. `excel_service.go:379-403` -- `applyReferenceResults()` replaces the name value with the resolved ID using the `DBField` as the target key, then deletes the original name key.
4. `excel_service.go:635-685` -- `prepareRecordsForUpsert()` only includes fields with `DBField != ""`, and for reference fields, uses the resolved ID (not the original name).
5. `batch_upserter.go:466-477` -- `shouldSkipFromUpdate()` skips reference columns without a valid `DBField`, ensuring only the ID field is written.

**Example (from workstation config, verified at `excel_config.go:122-123`):**
```go
// Source: excel_config.go lines 122-123
{Field: "deptName", Header: "所属部门", Reference: "sys_dept.dept_name", DBField: "dept_id"},
{Field: "userName", Header: "所属用户", Reference: "sys_user.username", DBField: "user_id"},
```

**What we need to add (direct analog):**
```go
{Field: "deviceName", Header: "所属设备", Reference: "sys_network_device.device_name", DBField: "device_id"},
{Field: "portName", Header: "所属端口", Reference: "sys_device_port_status.interface_name", DBField: "port_id"},
```

### Pattern 2: Optional Reference Field (VERIFIED)

**What:** Reference columns without `Required: true` will not block import when the value is empty or the name is not found in the database.

**When to use:** For optional associations that should not block the import process.

**How it works:**
1. Empty cells: `validateAndParseRow()` at `excel_service.go:769-778` skips empty values for non-required fields.
2. No reference requests: `extractReferenceRequests()` at `excel_service.go:367` only creates requests for non-empty values.
3. Match failure: `applyReferenceResults()` at `excel_service.go:398-399` logs a debug message but does not create an error when a reference key is not found in results.
4. Validation: `validateReferenceFields()` at `excel_service.go:711-712` only checks `col.Required` references, so optional ones pass through.
5. Partial update: The infoPoint config has `PartialUpdate: true`, so `buildUpdateData()` at `batch_upserter.go:225` only includes fields with non-nil values.

### Pattern 3: Model Already Has Fields (VERIFIED)

**What:** The `OpsInfoPoint` model already has the target fields.

**Verified at `internal/models/operations/infopoint.go:32-35`:**
```go
DeviceID   *string  `gorm:"size:64" json:"deviceId,omitempty"`
DeviceName *string  `gorm:"size:100" json:"deviceName,omitempty"`
PortID     *string  `gorm:"size:64" json:"portId,omitempty"`
PortName   *string  `gorm:"size:100" json:"portName,omitempty"`
```

All four fields use `*string` (pointer), meaning they can be nil in the database. This matches the "optional" requirement perfectly.

### Pattern 4: Redundant Field Population (VERIFIED -- already works)

**What:** After import, `infopoint_service.go` already populates `DeviceName` and `PortName` from `DeviceID` and `PortID`.

**Verified at `infopoint_service.go:62-76`:**
```go
// populateRedundantFields fills DeviceName and PortName from DeviceID/PortID
func (s *infoPointService) populateRedundantFields(ctx context.Context, infoPoint *operations.OpsInfoPoint) error {
    // ... fills WorkstationName
    if infoPoint.DeviceID != nil && *infoPoint.DeviceID != "" && infoPoint.DeviceName == nil {
        var device models.NetworkDevice
        if err := s.db.WithContext(ctx).Select("device_name").Where("id = ?", *infoPoint.DeviceID).First(&device).Error; err == nil {
            infoPoint.DeviceName = &device.DeviceName
        }
    }
    if infoPoint.PortID != nil && *infoPoint.PortID != "" && infoPoint.PortName == nil {
        var portStatus models.DevicePortStatus
        if err := s.db.WithContext(ctx).Select("interface_name").Where("id = ?", *infoPoint.PortID).First(&portStatus).Error; err == nil {
            infoPoint.PortName = &portStatus.InterfaceName
        }
    }
    return nil
}
```

This runs on Create and Update, so imported records will automatically have their redundant name fields populated.

### Anti-Patterns to Avoid

- **Do NOT use `DependsOn` for port:** The CONTEXT.md explicitly states ports use global lookup (not scoped to device). Using `DependsOn: "deviceName"` would cause the resolver to try filtering ports by `device_id`, which is not the desired behavior per D-04.
- **Do NOT modify the model:** The `OpsInfoPoint` model already has all needed fields. No migration required.
- **Do NOT modify the export config:** `excel_export_config.go` already has `deviceId`/`deviceName`/`portId`/`portName` export columns (lines 338-345). They work via JOIN configuration.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Name-to-ID resolution | Custom SQL queries | `ReferenceResolver.ResolveBatch()` | Already handles batch grouping, soft-delete filtering, error recovery |
| Reference column config | Custom import logic | `ExcelColumn.Reference` field | The entire import pipeline (extract -> resolve -> apply -> validate -> upsert) already handles Reference columns |
| Template example values | Manual string mapping | `getExampleValue()` switch | Add new cases to the existing switch for `deviceName`/`portName` fields |

**Key insight:** The entire import pipeline was designed to be configuration-driven. Adding new reference columns is a data-only change.

## Common Pitfalls

### Pitfall 1: Using wrong column name for port table

**What goes wrong:** REQUIREMENTS.md says `sys_device_port_status.port_name`, but the actual model column is `interface_name`.

**Why it happens:** The requirements were written before verifying the model schema. The `DevicePortStatus` model uses `InterfaceName` as the field name, which maps to `interface_name` in the database.

**How to avoid:** Use `sys_device_port_status.interface_name` in the Reference configuration, NOT `port_name`. This is confirmed by `device_port_status.go:33`:
```go
InterfaceName string `gorm:"size:100;not null" json:"interfaceName"`
```

**Warning signs:** If the Reference resolver returns "no records found" for valid port names, the field name in the Reference string is likely wrong.

### Pitfall 2: Export config has a pre-existing bug with port_name

**What goes wrong:** The export config at `excel_export_config.go:345` uses `SelectField: "port_name"`, but the actual database column is `interface_name`.

**Why it happens:** The export config was likely written assuming a `port_name` column that doesn't exist.

**How to avoid:** This is a pre-existing issue, NOT in scope for this phase. Do NOT fix it during this phase. The import config must use the correct column name (`interface_name`).

### Pitfall 3: Adding getExampleValue cases

**What goes wrong:** If the `getExampleValue()` switch in `excel_service.go` doesn't have cases for `deviceName` and `portName`, the template will show blank example cells for these columns.

**Why it happens:** The function has explicit cases for known field names, and falls through to returning empty string for unrecognized optional fields.

**How to avoid:** Add two cases to the `getExampleValue` switch:
```go
case "deviceName":
    return "核心交换机A"
case "portName":
    return "Gi0/1"
```

Note: `sourceDeviceName` and `destDeviceName` already have examples ("核心交换机A"), and `sourcePort`/`destPort` have "Gi0/1". Use the same values for consistency.

### Pitfall 4: Column order in the config

**What goes wrong:** Adding columns at the wrong position in the `Columns` array will misalign Excel headers with data cells.

**Why it happens:** Column order in the config determines the column order in the Excel template.

**How to avoid:** Add the two new columns after the existing columns but before `status` and `remark`. Follow the workstation pattern where `deptName` and `userName` appear after the core fields but before `remark`.

## Code Examples

### Verified pattern: Adding two Reference columns to infoPoint config

**Target file:** `internal/services/operations/excel_config.go`, lines 206-212

**Current infoPoint columns:**
```go
Columns: []ExcelColumn{
    {Field: "name", Header: "信息点名称", Required: true, MaxLength: 100, DBField: "name"},
    {Field: "infoPointType", Header: "信息点类型", Options: map[interface{}]string{...}, DBField: "info_point_type"},
    {Field: "workstationName", Header: "关联工位名称", Required: true, MaxLength: 100, Reference: "sys_workstation.workstation_name", DBField: "workstation_id"},
    {Field: "status", Header: "状态", Options: map[interface{}]string{...}, DBField: "status"},
    {Field: "remark", Header: "备注", MaxLength: 500, DBField: "remark"},
},
```

**After modification (add two columns between workstationName and status):**
```go
Columns: []ExcelColumn{
    {Field: "name", Header: "信息点名称", Required: true, MaxLength: 100, DBField: "name"},
    {Field: "infoPointType", Header: "信息点类型", Options: map[interface{}]string{...}, DBField: "info_point_type"},
    {Field: "workstationName", Header: "关联工位名称", Required: true, MaxLength: 100, Reference: "sys_workstation.workstation_name", DBField: "workstation_id"},
    {Field: "deviceName", Header: "所属设备", MaxLength: 100, Reference: "sys_network_device.device_name", DBField: "device_id"},
    {Field: "portName", Header: "所属端口", MaxLength: 100, Reference: "sys_device_port_status.interface_name", DBField: "port_id"},
    {Field: "status", Header: "状态", Options: map[interface{}]string{...}, DBField: "status"},
    {Field: "remark", Header: "备注", MaxLength: 500, DBField: "remark"},
},
```

### Verified pattern: Template example values

**Target file:** `internal/services/operations/excel_service.go`, `getExampleValue()` function

Add to the switch statement:
```go
case "deviceName":
    return "核心交换机A"
case "portName":
    return "Gi0/1"
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| N/A | Configuration-driven Reference pattern | Pre-existing | Adding columns is data-only, no new logic |

**No deprecations** -- this phase uses only established, stable patterns.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Port matching uses global lookup (not scoped to device) -- per CONTEXT.md D-04 and Claude's Discretion | Architecture Patterns | If device-scoped port lookup is actually needed, `DependsOn` pattern would be required, increasing complexity |
| A2 | The pre-existing export bug (`SelectField: "port_name"` vs actual column `interface_name`) should NOT be fixed in this phase | Common Pitfalls | If the export bug blocks VAL-04 testing, it would need to be fixed, expanding scope |
| A3 | `MaxLength: 100` is appropriate for device and port name columns | Code Examples | If names exceed 100 chars, imports would fail validation |

**Note:** Claims A1 and A2 are based on explicit CONTEXT.md decisions. Only A3 is truly assumed.

## Open Questions

1. **Export config bug: `port_name` vs `interface_name`**
   - What we know: `excel_export_config.go:345` uses `SelectField: "port_name"` but the actual column is `interface_name`
   - What's unclear: Whether this export bug currently manifests as a problem (port name might show blank in exports)
   - Recommendation: Flag for awareness, do NOT fix in this phase. If VAL-04 testing reveals the bug is visible, address it as a separate fix.

2. **Device name uniqueness**
   - What we know: `device_name` is not marked as unique in the model. The resolver uses `WHERE device_name IN (?) AND deleted_at IS NULL` and returns the first match via the map.
   - What's unclear: If two devices share the same `device_name`, the resolver will return whichever the database returns first (non-deterministic).
   - Recommendation: This is acceptable per D-01 (exact match). Users should be aware that duplicate device names may cause unpredictable resolution. Not a blocker.

## Environment Availability

Step 2.6: SKIPPED (no external dependencies -- this phase is purely a Go source code configuration change, no new tools, services, or runtimes required beyond the existing project build toolchain).

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify (existing) |
| Config file | None -- standard Go test files |
| Quick run command | `go test ./internal/services/operations/ -v -run TestReference -count=1` |
| Full suite command | `go test ./internal/services/operations/ -v -count=1` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| IMPORT-05 | Device name resolves to device_id via Reference | unit | `go test ./internal/services/operations/ -v -run TestReference -count=1` | Yes -- `reference_resolver_test.go` |
| IMPORT-06 | Port name resolves to port_id via Reference | unit | `go test ./internal/services/operations/ -v -run TestReference -count=1` | Yes -- `reference_resolver_test.go` |
| IMPORT-07 | Empty/missing device/port does not block import | unit | `go test ./internal/services/operations/ -v -run TestBatch -count=1` | Yes -- `batch_upserter_test.go` |
| VAL-03 | InfoPoint list shows device/port names after import | manual | Build server + import Excel file with device/port data, verify API response | N/A |
| VAL-04 | Export includes device/port columns without regression | manual | Call export API, verify Excel file has device/port columns | N/A |

### Sampling Rate
- Per task commit: `go build ./...`
- Per wave merge: `go test ./internal/services/operations/ -v -count=1`
- Phase gate: Full build + existing test suite green

### Wave 0 Gaps
None -- existing test infrastructure (`reference_resolver_test.go`, `batch_upserter_test.go`) covers the core mechanisms being used. No new test files needed for this configuration-only change.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | Not modifying auth flows |
| V3 Session Management | no | Not modifying session handling |
| V4 Access Control | no | Using existing permission middleware |
| V5 Input Validation | yes | GORM parameterized queries in ReferenceResolver |
| V6 Cryptography | no | No crypto changes |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| SQL injection via device/port names | Tampering | GORM parameterized queries (`Where(field+" IN ?", values)`) -- verified in `reference_resolver.go:261` |
| Data corruption via duplicate device names | Spoofing | Acceptable per D-01 -- exact match returns first result |

## Sources

### Primary (HIGH confidence)
- `internal/services/operations/excel_config.go` -- infoPoint configuration block (lines 197-213)
- `internal/services/operations/reference_resolver.go` -- batch resolution logic
- `internal/services/operations/excel_service.go` -- import pipeline with Reference handling
- `internal/services/operations/batch_upserter.go` -- partial update logic
- `internal/models/operations/infopoint.go` -- OpsInfoPoint model with DeviceID/PortID fields
- `internal/models/network_device.go` -- NetworkDevice model with device_name field
- `internal/models/device_port_status.go` -- DevicePortStatus model with interface_name field
- `internal/services/operations/infopoint_service.go` -- populateRedundantFields logic
- `internal/services/operations/excel_export_config.go` -- existing export config for infoPoint

### Secondary (MEDIUM confidence)
- None needed -- all findings verified directly from source code

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- no new dependencies, all existing verified in codebase
- Architecture: HIGH -- direct application of proven Reference pattern from Phase 1-2
- Pitfalls: HIGH -- verified column names and model fields directly from source code

**Research date:** 2026-04-16
**Valid until:** 2026-05-16 (stable -- configuration-only change with no external dependency)
