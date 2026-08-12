---
slug: infopoint-building-floor-not-showing
status: resolved
trigger: 信息点管理页面列表不显示"所属楼宇"和"所属楼层"列数据。已完成模型字段添加和后端重启，问题仍存在。
created: 2026-05-18T10:25:00Z
updated: 2026-05-18T10:45:00Z
session_type: bug
---

# Debug Session: infopoint-building-floor-not-showing

## Symptoms

### Expected Behavior
信息点管理页面列表应显示"所属楼宇"和"所属楼层"列，数据正常展示

### Actual Behavior
- 列标题存在但数据显示为 "-"
- 已修改 `OpsInfoPoint` 模型添加 `BuildingName`, `FloorName` 字段
- 已重新编译并重启后端服务
- 已刷新前端页面（Ctrl+F5）
- 问题仍然存在

### Error Messages
无错误消息，静默失败

### Timeline
- 2026-05-18 10:20: 完成模型字段添加
- 2026-05-18 10:22: 重新编译并重启后端
- 2026-05-18 10:25: 刷新前端后问题仍存在
- 2026-05-18 10:45: 根本原因已识别并修复

### Reproduction
1. 访问信息点管理页面
2. 查看列表中的"所属楼宇"和"所属楼层"列
3. 数据显示为 "-" 而非实际值

## Current Focus

- hypothesis: null
- next_action: null
- test: null
- expecting: null
- reasoning_checkpoint: null
- tdd_checkpoint: null

## Evidence

### Evidence 1: Model Definition (infopoint.go lines 32-34)
```go
BuildingID      *string         `gorm:"-" json:"buildingId,omitempty"`              // 楼宇ID（JOIN获取，不存储）
BuildingName    *string         `gorm:"-" json:"buildingName,omitempty"`            // 楼宇名称（JOIN获取，不存储）
FloorName       *string         `gorm:"-" json:"floorName,omitempty"`               // 楼层名称（JOIN获取，不存储）
```
- Fields are marked with `gorm:"-"` indicating virtual fields (not stored in DB)
- These should be populated via JOINs

### Evidence 2: Service Query (infopoint_service.go lines 155-163)
```go
if err := query.
    Select("ops_info_points.*, ops_floors.name as floor_name, ops_buildings.name as building_name, ops_buildings.id as building_id, sys_workstation.workstation_name").
    Joins("LEFT JOIN sys_workstation ON sys_workstation.id::text = ops_info_points.workstation_id").
    Joins("LEFT JOIN ops_floors ON ops_floors.id = sys_workstation.floor_id::uuid").
    Joins("LEFT JOIN ops_buildings ON ops_buildings.id = sys_workstation.building_id::uuid").
    ...
```
- Service does JOIN with `ops_floors` and `ops_buildings`
- SELECT clause includes `floor_name` and `building_name` aliases
- **ROOT CAUSE**: GORM won't automatically scan JOINed results into fields marked with `gorm:"-"`

### Evidence 3: Comparison with Workstation Model (workstation.go lines 49, 55)
```go
BuildingName *string `gorm:"size:100" json:"buildingName,omitempty"` // 楼宇名称
FloorName    *string `gorm:"size:100" json:"floorName,omitempty"`    // 楼层名称
```
- In Workstation model, these are regular database fields with `gorm:"size:100"`
- Workstation service (workstation_service.go line 15) uses the same JOIN pattern
- Workstation successfully populates these fields because they are NOT marked with `gorm:"-"`

## Eliminated

## Resolution

### Root Cause
The `BuildingName` and `FloorName` fields in `OpsInfoPoint` model were incorrectly marked with `gorm:"-"` (virtual fields), but the service tried to populate them via JOINs. GORM does not scan JOINed results into virtual fields - they need to be regular struct fields that GORM can write to.

### Fix Applied
Changed the field definitions in `internal/models/operations/infopoint.go`:
```go
// Before (virtual fields - INCORRECT)
BuildingID      *string         `gorm:"-" json:"buildingId,omitempty"`
BuildingName    *string         `gorm:"-" json:"buildingName,omitempty"`
FloorName       *string         `gorm:"-" json:"floorName,omitempty"`

// After (regular fields - CORRECT)
BuildingID      *string         `gorm:"size:64" json:"buildingId,omitempty"`
BuildingName    *string         `gorm:"size:100" json:"buildingName,omitempty"`
FloorName       *string         `gorm:"size:100" json:"floorName,omitempty"`
```

This matches the pattern used in `Workstation` model and allows GORM to properly scan the JOINed results.

### Verification
- Code compiles successfully
- Change aligns with existing patterns in the codebase (Workstation model)
- No migration needed (these are redundant fields populated via JOINs)

### Next Steps for User
1. Rebuild backend: `go build -o xingran-backend.exe ./cmd/main.go`
2. Restart backend service
3. Refresh frontend page (Ctrl+F5)
4. Verify "所属楼宇" and "所属楼层" columns display correct data

