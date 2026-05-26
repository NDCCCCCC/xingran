---
phase: 23-ad-group-sync
plan: 12
title: "Implement AD Group Management Service"
author: "Claude Opus 4.7"
created: "2026-05-26T00:00:00Z"
status: complete
duration: "PT15M"
subsystem: "AD域管理"
tags: ["ad", "group-management", "ldap", "cxhub-naming"]
tech-stack:
  added:
    - "LDAPClient: CreateGroup, DeleteGroup, AddGroupMembers, RemoveGroupMembers"
    - "GroupManagementService: department-based group CRUD operations"
  patterns:
    - "Handler-Service pattern延续"
    - "批量操作错误收集模式"
    - "安全删除前置检查模式"
key-files:
  created:
    - "internal/services/addomain/group_management_service.go"
  modified:
    - "internal/services/addomain/ldap_client.go"
    - "internal/services/addomain/service.go"
requirements: []
decisions:
  - "使用cxhub-{dept}命名规则，自动去除'部'后缀"
  - "DeleteGroup实施安全检查：成员数>0或存在映射关系时拒绝删除"
  - "批量操作使用错误收集模式，不中断整个流程"
metrics:
  tasks_completed: 3
  files_created: 1
  files_modified: 2
  lines_added: 369
  commits: 3
---

# Phase 23 Plan 12: AD Group Management Service Summary

## Objective

Create the GroupManagementService to manage AD groups (create, delete, add/remove members), implementing the `cxhub-{dept}` naming convention for department-based groups. This service extends the existing read-only group operations with write capabilities.

**Purpose**: Enable automatic creation of AD groups for departments, with proper naming convention and member management.

## Implementation Summary

### Tasks Completed

#### Task 1: Extend LDAPClient with Group Creation/Deletion Methods
**Commit**: `92f190c`

Extended `internal/services/addomain/ldap_client.go` with four new methods:

1. **CreateGroup(groupDN, groupName, description, groupType)** - Creates AD groups with proper objectClass attributes
   - Uses `objectClass: ["top", "group"]`
   - Sets `sAMAccountName` and `cn` to group name
   - Supports optional description
   - Default groupType: `-2147483646` (Global Security Group)

2. **DeleteGroup(groupDN)** - Removes AD groups via LDAP delete operation

3. **AddGroupMembers(groupDN, userDNs)** - Batch adds members to a group
   - Single LDAP modify operation with multiple member DNs
   - Empty array guard clause

4. **RemoveGroupMembers(groupDN, userDNs)** - Batch removes members from a group
   - Single LDAP modify operation with multiple member deletions
   - Empty array guard clause

**Key Design Decision**: All methods use go-ldap library's parameterized requests (AddRequest, ModifyRequest, DelRequest), which automatically escape values and prevent LDAP injection.

#### Task 2: Create GroupManagementService Interface and Implementation
**Commit**: `7be17c9`

Created `internal/services/addomain/group_management_service.go` (317 lines):

**Interface Methods**:
```go
type GroupManagementService interface {
    CreateGroupForDept(ctx, deptID, configID, parentOUDN) (*models.ADGroup, error)
    DeleteGroup(ctx, groupID) error
    AddMembers(ctx, groupID, userIDs) (*MemberChangeResult, error)
    RemoveMembers(ctx, groupID, userIDs) (*MemberChangeResult, error)
    BulkCreateGroupsForDepts(ctx, deptIDs, configID, parentOUDN) (*BulkCreateResult, error)
}
```

**Implementation Highlights**:

1. **CreateGroupForDept**: Implements `cxhub-{dept}` naming convention
   ```go
   groupName := fmt.Sprintf("cxhub-%s", strings.TrimSuffix(dept.DeptName, "部"))
   ```
   - Removes "部" suffix from department names (e.g., "科技创新部" → "cxhub-科技创新")
   - Uses `config.MemberOUDN` if provided, otherwise falls back to `config.BaseDN`
   - Creates both LDAP group entry and local `sys_ad_group` database record
   - Detects "already exists" errors (LDAP error code 68)

2. **DeleteGroup**: Multi-layer safety checks
   - Checks `MemberCount > 0` (cannot delete non-empty groups)
   - Checks for existing `DeptGroupMapping` records
   - Performs soft-delete on local database record after LDAP deletion

3. **AddMembers/RemoveMembers**: Batch member operations
   - Filters users to only those with valid `ad_dn` values
   - Updates local `member_count` field after successful LDAP operations
   - Returns `MemberChangeResult` with added/removed/failed counts
   - Failed count = users without valid AD DNs (skipped, not errors)

4. **BulkCreateGroupsForDepts**: Error collection pattern
   - Continues processing even if individual department groups fail
   - Collects error messages in `FailedDepts` array
   - Returns summary statistics (total/success/failed counts)

**Result Types**:
```go
type MemberChangeResult struct {
    GroupID      string
    GroupName    string
    AddedCount   int
    RemovedCount int
    FailedCount  int  // Users without valid AD DNs
}

type BulkCreateResult struct {
    TotalCount   int
    SuccessCount int
    FailedCount  int
    FailedDepts  []string  // "部门名: 错误原因"
}
```

#### Task 3: Register GroupManagementService in ADDomainService
**Commit**: `81ef252`

Updated `internal/services/addomain/service.go`:

1. Added field to struct:
```go
type ADDomainService struct {
    // ... existing fields
    GroupMgmt       GroupManagementService
    // ... existing fields
}
```

2. Initialized in constructor:
```go
func NewADDomainService(db *gorm.DB) *ADDomainService {
    return &ADDomainService{
        // ... existing initializations
        GroupMgmt:       NewGroupManagementService(db),
        // ... existing initializations
    }
}
```

This follows the established pattern where all sub-services are registered in the main ADDomainService for unified access.

## Deviations from Plan

**None** - Plan executed exactly as written.

## Threat Model Compliance

| Threat ID | Category | Mitigation Status |
|-----------|----------|-------------------|
| T-23-12-01 | Spoofing (Unauthorized group creation) | ⚠️ PENDING - Permission checks must be implemented in handler layer (require admin role) |
| T-23-12-02 | Tampering (LDAP operation failure) | ✅ MITIGATED - Service creates local record only after successful LDAP operation |
| T-23-12-03 | Information Disclosure (LDAP injection) | ✅ MITIGATED - Uses go-ldap library's parameterized requests (automatic escaping) |
| T-23-12-04 | Denial (Group deletion with members) | ✅ MITIGATED - DeleteGroup checks MemberCount > 0 before proceeding |

## Verification Results

### Automated Checks
✅ LDAP client methods exist:
```
298:func (c *LDAPClient) CreateGroup(groupDN, groupName, description string, groupType int) error
318:func (c *LDAPClient) DeleteGroup(groupDN string) error
324:func (c *LDAPClient) AddGroupMembers(groupDN string, userDNs []string) error
336:func (c *LDAPClient) RemoveGroupMembers(groupDN string, userDNs []string) error
```

✅ GroupManagementService structure verified:
```
15:type GroupManagementService interface
60:func (s *groupManagementService) CreateGroupForDept(...)
74:groupName := fmt.Sprintf("cxhub-%s", strings.TrimSuffix(dept.DeptName, "部"))
126:func (s *groupManagementService) DeleteGroup(...)
```

✅ Service registration verified:
```
112:  GroupMgmt       GroupManagementService
127:    GroupMgmt:       NewGroupManagementService(db),
```

### Compilation Check
✅ `go build ./internal/services/addomain/` - No errors

### Success Criteria Met
- [x] LDAPClient extended with CreateGroup, DeleteGroup, AddGroupMembers, RemoveGroupMembers
- [x] GroupManagementService implements all required methods
- [x] CreateGroupForDept uses cxhub-{dept} naming convention
- [x] DeleteGroup has safety checks (no members, no mappings)
- [x] AddMembers/RemoveMembers support batch operations
- [x] Service registered in ADDomainService with proper initialization
- [x] No compilation errors after integration

## Architecture Patterns Used

1. **Handler-Service Pattern Continuation**: GroupManagementService follows the established pattern with interface definition, private implementation, and constructor function.

2. **Batch Operation Error Collection**: `BulkCreateGroupsForDepts` continues processing on individual failures, collecting errors for aggregate reporting.

3. **Safety-First Deletion**: `DeleteGroup` implements pre-deletion checks to prevent accidental data loss (members, mappings).

4. **Dual-State Sync**: Service maintains both LDAP state (via operations) and local database state (sys_ad_group records).

## Integration Points

### Existing Components Used
- **LDAPClient**: Extended for group write operations
- **ADConfig model**: Uses MemberOUDN field for group placement
- **Department model**: Reads dept_name for group naming
- **User model**: Reads ad_dn field for member operations
- **DeptGroupMapping model**: Referenced in delete safety checks

### Data Flow
```
Handler → ADDomainService.GroupMgmt → GroupManagementService
                                        ↓
                                  LDAPClient (operations)
                                        ↓
                                  AD Server (group CRUD)
                                        ↓
                                  Local DB (sys_ad_group records)
```

## Next Steps

This plan completes the backend service layer for AD group management. Next plans should implement:

1. **API Handlers** (Plan 23-13): Create HTTP handlers for group management operations
   - POST /api/v1/ad/groups/create - Create group for department
   - DELETE /api/v1/ad/groups/:id - Delete group
   - POST /api/v1/ad/groups/:id/members/add - Add members
   - POST /api/v1/ad/groups/:id/members/remove - Remove members

2. **Permission Integration**: Add `ops:ad:group:manage` permission to protect group management endpoints

3. **Frontend Integration**: Add group management UI to AD domain management pages

4. **Member Synchronization** (Plan 23-14): Implement automatic member sync from sys_dept to AD groups

## Files Summary

### Created
- `internal/services/addomain/group_management_service.go` (317 lines)

### Modified
- `internal/services/addomain/ldap_client.go` (+50 lines)
- `internal/services/addomain/service.go` (+2 lines)

### Total Impact
- **Lines Added**: 369
- **Commits**: 3
- **Build Status**: ✅ Passing
