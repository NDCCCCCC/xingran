---
phase: 20-ad-ou-dept-mapping
plan: 04
subsystem: ad-sync
tags: [ad, ldap, user-sync, ou-move, graceful-degradation]
dependency_graph:
  requires: [20-01]
  provides: [user-ad-sync-service, handler-ad-sync-integration]
  affects: [user-handler, user-router]
tech_stack:
  added: []
  patterns: [async-goroutine, graceful-degradation, dependency-injection]
key_files:
  created:
    - internal/services/addomain/user_ad_sync_service.go
  modified:
    - internal/api/v1/system/user_handler.go
    - internal/api/v1/system/user_router.go
decisions:
  - Variadic parameter for UserADSyncService injection preserves backward compatibility
  - buildADSyncMap converts typed UserUpdateRequest to map for AD sync consumption
  - OU move failure does not abort attribute sync (partial success handling)
  - BatchMoveUsersToNewOU included in initial file for future batch operations
metrics:
  duration: 5m
  completed: 2026-05-22
  tasks: 4
  files: 3
---

# Phase 20 Plan 04: User AD Sync Service Summary

User AD sync service with async goroutine-based AD synchronization on user update, OU move on department change, attribute sync, and graceful degradation strategy.

## Completed Tasks

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | UserADSyncService implementation | c641c17 | internal/services/addomain/user_ad_sync_service.go (259 lines, new) |
| 2 | User handler AD sync integration | 8e1b04f | internal/api/v1/system/user_handler.go (+41 lines) |
| 3 | Router DI for UserADSyncService | b26b1f3 | internal/api/v1/system/user_router.go (+7 lines) |
| 4 | Batch user move support | c641c17 | included in Task 1 (user_ad_sync_service.go) |

## Key Implementation Details

### Task 1: UserADSyncService (259 lines)

Core methods:
- `SyncUserUpdateToAD(ctx, userID, updateReq)` - Main sync entry point called after system update
- `moveUserToNewOU(ctx, ldapClient, userID, newDeptID)` - Moves user to new OU via LDAP ModifyDN
- `syncUserAttributes(ctx, ldapClient, user, updateReq)` - Syncs displayName, mail, telephoneNumber, department to AD
- `updateSyncTimestamp(ctx, userID)` - Updates ad_synced_at column
- `BatchMoveUsersToNewOU(ctx, userIDs, newDeptID)` - Batch move with 10/batch rate limiting and 1s pause between batches
- `moveSingleUserToOU(ctx, ldapClient, userID, ouDN)` - Single user move helper for batch operations

Design decisions:
- LDAP client created per-sync-operation from AD config (no persistent connection)
- Checks user.ADDN before syncing; skips non-AD users gracefully
- OU move failure logs error but continues to attribute sync (partial success)
- AD config queried by `sync_enabled=true AND status=0`

### Task 2: User Handler Integration

- Added `userADSyncService *addomain.UserADSyncService` field to `UserHandler`
- `NewUserHandler` uses variadic parameter for backward compatibility: `userADSyncService ...*addomain.UserADSyncService`
- `Update` method triggers async AD sync via goroutine after successful system update
- `buildADSyncMap` converts typed `UserUpdateRequest` to `map[string]interface{}` mapping:
  - `nickname` -> `nickname`
  - `email` -> `email`
  - `phone` -> `phone`
  - `deptId` -> `deptId`

### Task 3: Router DI

- `SetupUserRouter` creates `DeptOUmapper` and `UserADSyncService`
- LDAP client passed as nil (created dynamically per sync)
- `UserADSyncService` injected into `UserHandler` via variadic constructor

### Task 4: Batch Move (included in Task 1)

- `BatchMoveUsersToNewOU` processes users in batches of 10
- 1 second pause between batches to avoid AD pressure
- Individual failures tracked but do not abort remaining users
- Returns error if any failures occurred with success/failure counts

## Attribute Mapping (System -> AD)

| System Field | AD Attribute | Notes |
|-------------|-------------|-------|
| nickname | displayName | Skip if empty string |
| email | mail | Direct mapping |
| phone | telephoneNumber | Direct mapping |
| deptId -> dept.DeptName | department | Resolved via DB lookup |

## Graceful Degradation Strategy

1. User without AD DN -> Skip sync entirely (INFO log)
2. AD config not found -> Return error (but system update already succeeded)
3. LDAP connection failure -> Return error (system update unaffected)
4. OU move failure -> Log error, continue to attribute sync
5. Attribute sync failure -> Return error (system update unaffected)
6. Sync timestamp update failure -> WARN log only
7. userADSyncService is nil -> Skip sync entirely (backward compatible)

## Data Flow

```
Admin updates user via POST /system/users/:id/update
  -> UserHandler.Update()
    -> service.Update() (system DB update)
    -> if userADSyncService != nil:
      -> goroutine (async, non-blocking):
        -> buildADSyncMap(req) -> map[string]interface{}
        -> SyncUserUpdateToAD(ctx, userID, map)
          -> Query user (check ADDN)
          -> Query AD config
          -> Connect LDAP
          -> If deptId changed: moveUserToNewOU()
            -> mapper.FindOUDNByDeptID()
            -> ldap.MoveUser(userDN, ouDN)
            -> Update user.ad_ou_dn
          -> syncUserAttributes()
            -> Build attribute map
            -> ldap.UpdateUserAttribute(userDN, attrs)
          -> updateSyncTimestamp()
  -> response.Success (already sent before AD sync completes)
```

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Adapted to actual UserHandler structure**
- **Found during:** Task 2
- **Issue:** Plan assumed `UpdateUser` method with `map[string]interface{}`, but actual code uses `Update` with typed `UserUpdateRequest`
- **Fix:** Used variadic parameter for DI, created `buildADSyncMap` helper to convert typed request to map
- **Files modified:** user_handler.go
- **Commit:** 8e1b04f

**2. [Rule 3 - Blocking] Adapted to actual User model field types**
- **Found during:** Task 1
- **Issue:** Plan assumed `user.DeptID` is `string`, but actual model uses `*string` (nullable pointer)
- **Fix:** Added nil check and dereference for DeptID comparison
- **Files modified:** user_ad_sync_service.go
- **Commit:** c641c17

**3. [Rule 3 - Blocking] Merged Task 4 into Task 1**
- **Found during:** Task 1
- **Issue:** Plan's Task 4 adds methods to the same file created in Task 1
- **Fix:** Included BatchMoveUsersToNewOU and moveSingleUserToOU in the initial file creation
- **Files modified:** user_ad_sync_service.go
- **Commit:** c641c17

### Key Design Adjustments

1. **Variadic constructor** instead of plan's three-parameter `NewUserHandler(userSvc, adSyncSvc, pwdMgr)` - preserves backward compatibility with existing callers that only pass `service`
2. **Field name `deptId`** in map instead of `dept_id` - matches the JSON tag in `UserUpdateRequest`
3. **AD config filter** uses `status=0` (enabled) in addition to `sync_enabled=true` for more precise config selection

## Verification

- All code compiles: `go build ./internal/services/addomain/...` and `go build ./internal/api/v1/system/...` and `go build ./cmd/...`
- No new untracked files from this plan
- No unexpected file deletions in any commit

## Self-Check: PASSED

- FOUND: internal/services/addomain/user_ad_sync_service.go
- FOUND: internal/api/v1/system/user_handler.go
- FOUND: internal/api/v1/system/user_router.go
- FOUND: c641c17 (Task 1 + Task 4)
- FOUND: 8e1b04f (Task 2)
- FOUND: b26b1f3 (Task 3)
