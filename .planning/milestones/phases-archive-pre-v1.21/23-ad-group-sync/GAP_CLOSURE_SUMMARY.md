# Phase 23 Gap Closure Plans Summary

**Phase:** 23-ad-group-sync (Gap Closure Mode)
**Plans Created:** 6 (23-08 through 23-13)
**Date:** 2026-05-26

## Gap Closure Overview

Based on the verification report (23-VERIFICATION.md), Phase 23's implementation diverged significantly from its original plan. The implemented feature was an **incremental AD group reader** that syncs existing LDAP groups to the local database, while the plan called for a **department-to-group mapping system** with automatic group creation, member syncing, and exclusive membership enforcement.

## Verification Score: 1/6 Truths Verified

| # | Truth | Status | Gap Addressed By |
|---|-------|--------|------------------|
| 1 | Can specify "member OU DN" via configuration | FAILED | Plan 23-08 |
| 2 | System auto-creates AD groups for departments | FAILED | Plan 23-12 |
| 3 | Department members automatically become group members | FAILED | Plan 23-11 |
| 4 | Department changes trigger automatic group updates | FAILED | Plan 23-11 |
| 5 | API for manual sync trigger and sync status query | VERIFIED | ✅ Already implemented |
| 6 | Scheduled task auto-runs sync logic | PARTIAL | Plan 23-13 |

## Gap Closure Plans

### Plan 23-08: Add Member OU DN Configuration Field
**Gap Addressed:** No ADConfig.member_ou_dn field or sys_config for target group OU

**Implementation:**
- Add `MemberOUDN` field to ADConfig model (internal/models/ad_domain.go)
- Create migration 132_add_member_ou_dn_to_ad_config.sql
- Update ADConfigCreateRequest and ADConfigUpdateRequest to include MemberOUDN

**Deliverables:**
- ADConfig model with MemberOUDN field
- Database migration adding member_ou_dn column
- API request structs supporting the new field

**Wave:** 1 (no dependencies)

---

### Plan 23-09: Create Department-Group Mapping Data Model
**Gap Addressed:** No DeptGroupMappingService or sys_dept_group_mapping table

**Implementation:**
- Create DeptGroupMapping model (internal/models/dept_group_mapping.go)
- Create DeptGroupMappingSyncLog model for tracking sync history
- Create migration 133_create_dept_group_mapping_table.sql

**Deliverables:**
- DeptGroupMapping model with foreign keys to sys_dept and sys_ad_group
- DeptGroupMappingSyncLog model for audit trail
- Database migrations creating both tables with proper constraints

**Wave:** 1 (no dependencies)

---

### Plan 23-10: Implement DeptGroupMappingService
**Gap Addressed:** No DeptGroupMappingService to manage department-group relationships

**Implementation:**
- Create DeptGroupMappingService interface and implementation
- Implement CRUD operations (CreateMapping, UpdateMapping, DeleteMapping, GetMapping, ListMappings)
- Implement AutoMapDepartment (finds cxhub-{dept} groups automatically)
- Implement AutoMapAllDepartments (batch auto-mapping)
- Register service in ADDomainService

**Deliverables:**
- DeptGroupMappingService with full CRUD
- Automatic group discovery by naming convention
- Service integration into ADDomainService

**Dependencies:** 23-08 (MemberOUDN field), 23-09 (mapping table)
**Wave:** 2

---

### Plan 23-11: Implement MemberSyncService
**Gap Addressed:** No MemberSyncService to push dept members to AD groups, no exclusive membership enforcement

**Implementation:**
- Create MemberSyncService interface and implementation
- Implement SyncDeptMembers (compares current vs target members, adds/removes deltas)
- Implement SyncAllMembers (batch sync for all mapped departments)
- Implement HandleDeptChange (remove from old group, add to new group)
- Implement EnsureExclusiveMembership (enforces one-group-per-user policy)
- Register service in ADDomainService

**Deliverables:**
- MemberSyncService with incremental sync logic
- Department change handling
- Exclusive membership enforcement
- Sync log recording to DeptGroupMappingSyncLog

**Dependencies:** 23-09 (mapping table), 23-10 (DeptGroupMappingService)
**Wave:** 2

---

### Plan 23-12: Implement AD Group Management Service
**Gap Addressed:** No CreateGroup logic (only reads existing groups)

**Implementation:**
- Extend LDAPClient with CreateGroup, DeleteGroup, AddGroupMembers, RemoveGroupMembers
- Create GroupManagementService interface and implementation
- Implement CreateGroupForDept (cxhub-{dept} naming convention)
- Implement DeleteGroup (with safety checks: no members, no mappings)
- Implement AddMembers/RemoveMembers (batch operations)
- Implement BulkCreateGroupsForDepts (batch group creation)
- Register service in ADDomainService

**Deliverables:**
- LDAP client extensions for group write operations
- GroupManagementService with full CRUD
- cxhub-{dept} naming convention implementation

**Dependencies:** 23-08 (MemberOUDN field)
**Wave:** 2

---

### Plan 23-13: Register Periodic Group Sync Cron Task
**Gap Addressed:** Group sync not registered as periodic cron job

**Implementation:**
- Modify ADSyncScheduler.Start to register second cron job for group sync (15-minute interval)
- Implement checkAndSyncADGroups (queries active mappings, triggers sync)
- Modify syncADGroups to call MemberSyncService.SyncAllMembers
- Register ad_group_member_sync task in main Scheduler
- Update GetADSyncStatus to show both full sync and group sync schedules

**Deliverables:**
- Dual cron jobs (full sync: 5min, group sync: 15min)
- Smart execution (only triggers for configs with active mappings)
- Enhanced status reporting

**Dependencies:** 23-09 (mapping table), 23-10 (DeptGroupMappingService), 23-11 (MemberSyncService)
**Wave:** 3

## Wave Structure

| Wave | Plans | Autonomous | Description |
|------|-------|------------|-------------|
| 1 | 23-08, 23-09 | yes, yes | Data model and configuration foundation |
| 2 | 23-10, 23-11, 23-12 | yes, yes, yes | Service layer implementation |
| 3 | 23-13 | yes | Cron registration and scheduler integration |

## Key Implementation Patterns

### Naming Convention
- **Group names:** `cxhub-{dept}` (e.g., "科技创新部" → "cxhub-科技创新")
- **Implementation:** GroupManagementService.CreateGroupForDept

### Exclusive Membership
- **Rule:** Each member belongs to exactly one department group
- **Implementation:** MemberSyncService.EnsureExclusiveMembership

### Sync Flow
1. **DeptGroupMappingService** identifies which groups exist for departments
2. **MemberSyncService** compares current members vs target members (dept users)
3. **GroupManagementService** creates/deletes groups via LDAP
4. **ADSyncScheduler** triggers periodic sync every 15 minutes

### Error Handling
- Continue processing individual items on failure (batch operations)
- Log errors with context (which dept, which user, which operation)
- Use soft deletes to preserve audit trail

## Integration Points

### Existing Code (Reuse, Don't Modify)
- `internal/services/addomain/group_sync_service.go` - Keep as-is (reads existing groups)
- `internal/services/addomain/sync.go` - Keep as-is (fixed memberOf parsing)
- `internal/api/v1/system/ad_domain_handler.go` - Extend with new endpoints
- `internal/api/v1/system/ad_domain_router.go` - Add new routes

### New Code (Create from Scratch)
- `internal/models/dept_group_mapping.go` - NEW: Mapping models
- `internal/services/addomain/dept_group_mapping_service.go` - NEW: Mapping CRUD
- `internal/services/addomain/member_sync_service.go` - NEW: Member sync logic
- `internal/services/addomain/group_management_service.go` - NEW: Group write operations

### Extensions (Modify Existing)
- `internal/models/ad_domain.go` - ADD: MemberOUDN field
- `internal/services/addomain/ldap_client.go` - ADD: CreateGroup, DeleteGroup methods
- `internal/services/addomain/service.go` - ADD: Register new services
- `internal/scheduler/ad_sync_tasks.go` - MODIFY: Add second cron job
- `internal/scheduler/scheduler.go` - ADD: Register ad_group_member_sync task

## Verification Strategy

After all plans are executed, re-run verification to check:

1. **Configuration:** Can set member_ou_dn via API and persist to database
2. **Group creation:** Can create AD group with cxhub-{dept} naming
3. **Member sync:** Department members are added to corresponding groups
4. **Department changes:** User removed from old group and added to new group
5. **Exclusive membership:** Each user belongs to only one department group
6. **Cron execution:** Group sync runs automatically every 15 minutes

## Next Steps

1. **Execute plans in wave order:**
   - Start with Wave 1 (23-08, 23-09) - data model foundation
   - Proceed to Wave 2 (23-10, 23-11, 23-12) - service implementation
   - Complete with Wave 3 (23-13) - scheduler integration

2. **Create API handlers** (not in scope for gap closure, but needed for frontend):
   - DeptGroupMapping CRUD endpoints
   - Member sync trigger endpoints
   - Group management endpoints (create/delete)

3. **Frontend integration** (not in scope for gap closure):
   - Department-group mapping management UI
   - Member sync status display
   - Manual sync trigger buttons

4. **Re-verification:** After all plans complete, run verification again to confirm all 6 truths now pass

---

*Gap Closure Plans Generated: 2026-05-26*
*Plans: 23-08 through 23-13*
*Total: 6 plans closing all verification gaps*
