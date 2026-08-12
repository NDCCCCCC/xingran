---
phase: 23-ad-group-sync
plan: 10
title: "Create DeptGroupMappingService for Department-to-Group Mappings"
status: complete
completed: 2025-01-26T12:00:00Z
duration_minutes: 15
---

# Phase 23 Plan 10: DeptGroupMappingService Implementation Summary

## Objective

Create the DeptGroupMappingService to manage department-to-group mappings, including CRUD operations, automatic group discovery by naming convention (`cxhub-{dept}`), and mapping validation.

## One-Liner

Implemented a comprehensive department-to-AD-group mapping service with CRUD operations, automatic `cxhub-{dept}` group discovery, and batch processing capabilities.

## Deviations from Plan

### Auto-fixed Issues

**None - plan executed exactly as written.**

## Key Files Created/Modified

### Created
- `internal/services/addomain/dept_group_mapping_service.go` (356 lines)
  - DeptGroupMappingService interface with 9 methods
  - Service implementation with validation and business logic
  - Request/response DTOs for all operations

### Modified
- `internal/services/addomain/service.go`
  - Added `DeptGroupMapping DeptGroupMappingService` field to ADDomainService
  - Initialized service in constructor: `DeptGroupMapping: NewDeptGroupMappingService(db)`

## Service Architecture

### Interface Methods
```go
type DeptGroupMappingService interface {
    CreateMapping(ctx, *CreateMappingRequest) (*models.DeptGroupMapping, error)
    UpdateMapping(ctx, id, *UpdateMappingRequest) error
    DeleteMapping(ctx, id) error
    GetMapping(ctx, id) (*models.DeptGroupMapping, error)
    ListMappings(ctx, *ListMappingsRequest) (*ListMappingsResponse, error)
    AutoMapDepartment(ctx, deptID, configID) (*models.DeptGroupMapping, error)
    AutoMapAllDepartments(ctx, configID) (*AutoMapResult, error)
    GetMappingByDept(ctx, deptID) (*models.DeptGroupMapping, error)
}
```

### Request/Response DTOs
- **CreateMappingRequest**: deptId, adGroupId, adConfigId, mappingType, syncEnabled, createdBy
- **UpdateMappingRequest**: mappingStatus, syncEnabled, updatedBy (all optional)
- **ListMappingsRequest**: adConfigId, deptId, mappingType, mappingStatus, groupName, current, pageSize
- **ListMappingsResponse**: total, list
- **AutoMapResult**: totalDepts, mappedCount, failedCount, failedDepts

## Core Features

### 1. CRUD Operations
- **CreateMapping**: Validates dept/group/config existence, checks duplicate mappings, creates with defaults
- **UpdateMapping**: Partial updates for status and sync_enabled flag
- **DeleteMapping**: Soft delete using GORM
- **GetMapping**: Preloads Dept and ADGroup associations
- **ListMappings**: Filter by config/dept/type/status/groupname with pagination

### 2. Auto-Discovery Logic
```go
// Naming convention: cxhub-{deptName without "部" suffix}
groupName := fmt.Sprintf("cxhub-%s", strings.TrimSuffix(dept.DeptName, "部"))
// Example: 科技创新部 -> cxhub-科技创新
```

**AutoMapDepartment**:
1. Fetches department by ID
2. Constructs expected group name: `cxhub-{dept}`
3. Queries ADGroup table for match by configId and groupName
4. Returns existing mapping if found, or creates new auto-mapping

**AutoMapAllDepartments**:
1. Queries all second-level departments (parent_id IS NOT NULL AND status = 0)
2. Iterates through each department, calling AutoMapDepartment
3. Collects successes and failures with descriptive error messages
4. Returns aggregate result with counts and failed department list

### 3. Validation Layer
- Department existence check before creating mapping
- ADGroup existence check before creating mapping
- ADConfig existence check before creating mapping
- Duplicate mapping detection (one dept → one group)
- GORM constraints enforced via uniqueIndex

## Integration Points

### Database Tables
- **sys_dept_group_mapping**: Main mapping storage
  - Unique constraint on (dept_id, deleted_at) ensures one-to-one mapping
  - Foreign keys to sys_dept, sys_ad_group, sys_ad_config
- **sys_dept**: Department data
- **sys_ad_group**: AD group data with group_name field

### Service Registration
```go
type ADDomainService struct {
    Config          *ConfigService
    Sync            *SyncService
    OU              *OUService
    User            *UserService
    Group           *GroupService
    GroupSync       *GroupSyncService
    DeptGroupMapping DeptGroupMappingService  // NEW
    Log             *LogService
    Computer        *ComputerService
}
```

## Threat Mitigation

| Threat | Mitigation |
|--------|------------|
| T-23-10-01: Unauthorized mapping creation | Service layer ready for handler permission checks (admin role required) |
| T-23-10-02: SQL injection in filters | All queries use GORM parameterized queries with automatic binding |
| T-23-10-03: Race condition in AutoMapDepartment | Unique constraint on (dept_id, deleted_at) prevents duplicates; duplicate check is pre-flight validation |
| T-23-10-04: Information disclosure | Generic error messages ("部门不存在") returned to caller; detailed errors logged internally |

## Next Steps

**Plan 23-11** will create API handlers for this service:
1. `CreateMappingHandler` - POST /mappings
2. `UpdateMappingHandler` - POST /mappings/:id/update
3. `DeleteMappingHandler` - POST /mappings/:id/delete
4. `GetMappingHandler` - POST /mappings/:id
5. `ListMappingsHandler` - POST /mappings/list
6. `AutoMapDepartmentHandler` - POST /mappings/auto-map/:deptId
7. `AutoMapAllDepartmentsHandler` - POST /mappings/auto-map-all

## Testing Considerations

**Recommended test scenarios**:
1. Create mapping with valid dept/group/config IDs
2. Attempt duplicate mapping creation (should fail)
3. Update mapping status and sync_enabled flag
4. List mappings with various filter combinations
5. AutoMapDepartment with matching cxhub-* group
6. AutoMapDepartment with non-existent group (should fail gracefully)
7. AutoMapAllDepartments with mixed success/failure departments
8. Soft delete mapping and verify it's excluded from queries

## Known Limitations

1. **Naming convention assumption**: AutoMapDepartment assumes groups follow `cxhub-{dept}` pattern. Groups with different naming will not be auto-discovered.
2. **Manual mapping required**: For groups not following the naming convention, users must manually create mappings via API.
3. **No group creation**: Service only discovers existing groups; it does not create new AD groups.
4. **Single mapping per dept**: One department can map to only one AD group (enforced by unique constraint).

## Commit Details

**Commit**: `75dbacf` - feat(23-10): create DeptGroupMappingService for department-to-group mappings

**Files**: 2 changed, 356 insertions(+), 16 deletions(-)

**Impact**: Added complete service layer for department-to-group mapping management with auto-discovery capabilities.
