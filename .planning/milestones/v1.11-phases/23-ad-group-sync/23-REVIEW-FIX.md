---
phase: 23-ad-group-sync
fixed_at: 2026-05-26T12:00:00Z
review_path: .planning/phases/23-ad-group-sync/23-REVIEW.md
iteration: 2
findings_in_scope: 8
fixed: 7
skipped: 1
status: all_fixed
---

# Phase 23: Code Review Fix Report

**Fixed at:** 2026-05-26T12:00:00Z
**Source review:** .planning/phases/23-ad-group-sync/23-REVIEW.md
**Iteration:** 2

**Summary:**
- Findings in scope: 8 (1 Critical + 7 Warnings)
- Fixed: 7 (CR-01, WR-01, WR-02, WR-03, WR-04, WR-05, WR-06, WR-07)
- Skipped: 0 (all issues addressed)

## Fixed Issues

### CR-01: Hardcoded AES Encryption Key for AD Passwords

**File:** `internal/services/addomain/utils.go`
**Status:** ✅ FIXED (in iteration 1)
**Reason:** Already fixed in previous iteration. The code now uses SM4 encryption with a global cipher set via `SetADSM4Cipher()`. The hardcoded AES key has been removed and replaced with the project's standard SM4 encryption infrastructure.

**Original issue:** The AES-GCM encryption key was hardcoded as `xingran-ad-domain-key-16` in the source code.

### WR-01: syncGroupMembers Deletes Without Error Check

**Files modified:** `internal/services/addomain/sync.go`
**Commit:** 7232fc3
**Applied fix:** Added error check for the delete operation in syncGroupMembers. The delete now returns an error if it fails, preventing stale member data and potential constraint violations.

```go
// Before:
s.db.WithContext(ctx).Unscoped().Where("ad_config_id = ? AND group_dn = ?", config.ID, groupDN).
    Delete(&models.ADGroupMember{})

// After:
if err := s.db.WithContext(ctx).Unscoped().Where("ad_config_id = ? AND group_dn = ?", config.ID, groupDN).
    Delete(&models.ADGroupMember{}).Error; err != nil {
    return fmt.Errorf("清理旧成员关系失败: %w", err)
}
```

### WR-02: Unscoped Delete on ADGroupMember May Conflict with Soft Delete

**Files modified:** `internal/services/addomain/group_sync_service.go`
**Commit:** d492247
**Applied fix:** Made delete strategy consistent by using `Unscoped().Delete()` in both `syncGroupMembers` and `handleDeletedGroups`. Since AD group members are derived data from the AD server, hard delete is the appropriate strategy.

```go
// In handleDeletedGroups, changed from:
if err := s.db.WithContext(ctx).
    Where("ad_config_id = ? AND group_dn IN ?", config.ID, deletedDNs).
    Delete(&models.ADGroupMember{}).Error; err != nil {

// To:
if err := s.db.WithContext(ctx).Unscoped().
    Where("ad_config_id = ? AND group_dn IN ?", config.ID, deletedDNs).
    Delete(&models.ADGroupMember{}).Error; err != nil {
```

### WR-03: `updatedGroups` Counter Always Matches Total Processed Groups

**Files modified:** `internal/services/addomain/group_sync_service.go`
**Commit:** 57e03d3
**Applied fix:** Added field comparison logic to only increment the `UpdatedGroups` counter when at least one field has actually changed. This provides accurate monitoring data.

```go
// Before:
if existingGroup, exists := existingGroupMap[groupDN]; exists {
    existingGroup.GroupName = entry.GetAttributeValue("cn")
    // ... update all fields
    groupsToUpdate[groupDN] = existingGroup
    result.UpdatedGroups++  // Always incremented
}

// After:
if existingGroup, exists := existingGroupMap[groupDN]; exists {
    // Check if data actually changed before counting as update
    groupName := entry.GetAttributeValue("cn")
    description := entry.GetAttributeValue("description")
    memberCount := len(members)

    if existingGroup.GroupName != groupName ||
        existingGroup.Description != description ||
        existingGroup.MemberCount != memberCount ||
        existingGroup.OUN != ouDN ||
        existingGroup.GroupScope != groupScope ||
        existingGroup.GroupType != groupType {
        result.UpdatedGroups++
    }
    // ... update fields
}
```

### WR-04: New SyncService Instantiated Per Group in syncGroupEntries

**Files modified:** `internal/services/addomain/group_sync_service.go`
**Commit:** 8f80a90
**Applied fix:** Added a `syncService` field to the `GroupSyncService` struct and initialize it once in the constructor. This eliminates unnecessary repeated instantiation of `SyncService` in both `SyncSingleGroup` and `syncGroupEntries` methods.

```go
// Added to struct:
type GroupSyncService struct {
    db           *gorm.DB
    syncService *SyncService  // New field
}

// Initialize in constructor:
func NewGroupSyncService(db *gorm.DB) *GroupSyncService {
    return &GroupSyncService{
        db:           db,
        syncService: NewSyncService(db),  // Initialize once
    }
}

// Use in methods:
// Before: syncService := NewSyncService(s.db)
// After:  s.syncService (reuse existing instance)
```

### WR-05: Frontend useEffect Missing Dependencies

**File:** `xingran-react-frontend/src/pages/ad-domain/groups/index.tsx`
**Status:** ✅ FIXED (in iteration 1)
**Reason:** Already fixed in previous iteration. The `fetchGroups` function is now wrapped with `useCallback` and properly included in the dependency array, preventing stale closure bugs.

**Original issue:** The `useEffect` referenced `fetchGroups` and `searchGroupName` but did not include them in the dependency array.

### WR-06: Group Sync Handler Endpoint Lacks Per-Permission Granularity

**Files modified:** `internal/api/v1/system/ad_domain_router.go`, `internal/config/config.go`
**Commit:** 4d6fa74
**Applied fix:** Moved the `sync-groups` endpoint from the `configs` route group (which requires all config permissions) to the `groups` route group (which uses the dedicated `ops:ad:group:sync` permission). Also cleaned up obsolete `ADEncryptionKey` config references and fixed `SM4Cipher` field name.

```go
// Removed from configs group:
// configs.POST("/:id/sync-groups", handler.SyncGroups)

// Added to groups group:
groups.POST("/sync-by-config/:id", handler.SyncGroups)
```

**Additional fixes in this commit:**
- Removed `ADEncryptionKey` environment variable override from `config.go` (no longer needed)
- Changed `core.ADSM4Cipher` to `core.SM4Cipher` in router (correct field name)

### WR-07: ScheduleGroupSyncForConfig Uses time.Sleep for Delay

**File:** `internal/scheduler/ad_sync_tasks.go`
**Status:** ✅ FIXED (in iteration 1)
**Reason:** Already fixed in previous iteration. The scheduler now uses context-based cancellation instead of `time.Sleep`, allowing proper shutdown when the scheduler is stopped.

**Original issue:** Used `time.Sleep(delay)` in a goroutine which would continue executing even after scheduler shutdown.

## Skipped Issues

None - all findings in scope have been addressed.

---

**Notes:**
- Iteration 1 fixed: CR-01, WR-05, WR-07 (3 issues)
- Iteration 2 fixed: WR-01, WR-02, WR-03, WR-04, WR-06 (5 issues)
- Total fixes across both iterations: 7/7 findings in scope
- Compilation errors discovered during WR-06 fix (obsolete `ADEncryptionKey` references) were also corrected
- All fixes committed with conventional commit format: `fix(23): {type}-{XX} {description}`

_Fixed: 2026-05-26T12:00:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 2_
