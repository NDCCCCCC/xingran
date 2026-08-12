# Phase 19 Plan 03: AD User Sync Service and Database Migration Summary

---
phase: 19
plan: 03
subsystem: authentication
tags: [user-sync, database-migration, ad-auth, auto-sync]
dependency_graph:
  requires: [19-02]
  provides: [UserSyncService, migration 100 auth_source fields, UserSyncer interface]
  affects: [internal/models/user.go, internal/core/security/ad_authenticator.go, internal/core/security/auth_strategy_factory.go]
tech_stack:
  added: []
  patterns: [Adapter Pattern for cross-package interface, Graceful Degradation]
key_files:
  created:
    - internal/core/db/migrations/100_add_auth_source_fields.sql
    - internal/services/system/user_sync_service.go
  modified:
    - internal/models/user.go
    - internal/core/security/authenticator.go
    - internal/core/security/ad_authenticator.go
    - internal/core/security/auth_strategy_factory.go
decisions:
  - Migration numbered 100 instead of 085 since 085 was already used for dedicated_line_ip_fields
  - Used UserSyncer interface in security package to avoid circular import between core/security and services/system
  - UserSyncService.SyncADUser adapts security.ADUserInfo to internal ADUserInfoForSync type
  - ADAuthenticator sync failure does not block authentication (graceful degradation)
  - userSyncer is optional (nil-safe) so ADAuthenticator works without sync service configured
metrics:
  duration: 8m
  completed: 2026-05-21
  tasks_total: 4
  tasks_completed: 4
  files_created: 2
  files_modified: 4
  commit_count: 4
---

## One-liner

AD user sync service with database migration (auth_source, ad_username, ad_dn fields), UserSyncer interface for cross-package integration, and graceful sync-on-auth pattern.

## Completed Tasks

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Create database migration script | 05b2cdc | internal/core/db/migrations/100_add_auth_source_fields.sql |
| 2 | Update User model | e56add5 | internal/models/user.go |
| 3 | Implement user sync service | bb86e4c | internal/services/system/user_sync_service.go |
| 4 | Integrate user sync into AD auth flow | 240b9f8 | internal/core/security/authenticator.go, ad_authenticator.go, auth_strategy_factory.go, user_sync_service.go |

## What Was Built

### Database Migration (`100_add_auth_source_fields.sql`)
- `auth_source` column: VARCHAR(10), default 'local', NOT NULL -- distinguishes local vs AD users
- `ad_username` column: VARCHAR(100) -- stores AD account name for lookup
- `ad_dn` column: TEXT -- stores AD distinguished name for LDAP operations
- Indexes on `auth_source` and `ad_username` for query performance
- Partial unique index on `ad_username` (WHERE NOT NULL) prevents duplicate AD user mapping
- Existing users set to 'local' auth_source automatically

### User Model Update (`models/user.go`)
- Added `AuthSource string` field with GORM tag `size:10;default:'local';not null`
- Added `ADUsername *string` field with GORM tag `size:100`
- Added `ADDN *string` field with GORM tag `type:text`

### UserSyncer Interface (`security/authenticator.go`)
- `UserSyncer` interface with `SyncADUser(ctx, *ADUserInfo, defaultRoleID) (*SyncedUser, error)` method
- `SyncedUser` struct: mirrors UserResult fields for cross-package data transfer
- Enables dependency inversion: security package depends on abstraction, not concrete service

### UserSyncService (`services/system/user_sync_service.go`)
- `ADUserInfoForSync` struct: internal representation for sync operations
- `SyncUserFromAD(ctx, *ADUserInfoForSync, defaultRoleID) (*User, error)`: core sync logic
- `createNewUser`: creates sys_user with AuthSource="ad", InitFlag=true (forces password change), default password "123456", optional default dept/role
- `updateExistingUser`: updates email/phone/ad_dn from AD, preserves locally modified nickname
- `assignRole`: uses `INSERT ... ON CONFLICT DO NOTHING` for idempotent role assignment
- `SyncADUser`: adapter method implementing `security.UserSyncer` interface, converts types
- All database operations wrapped in transactions for atomicity

### AD Authenticator Integration (`ad_authenticator.go`)
- Added `userSyncer UserSyncer` field (optional, nil-safe)
- `SetUserSyncer(syncer)` method for dependency injection
- After successful LDAP auth + admin search, calls `SyncADUser` if syncer is set
- On sync failure: gracefully degrades to returning NeedsSync=true (auth still succeeds)
- On sync success: returns full UserResult with NeedsSync=false
- Added `getDefaultRoleID()` helper reading `sys.auth.ad.default_role_id` from sys_config

### Factory Integration (`auth_strategy_factory.go`)
- Added `userSyncer UserSyncer` field and `SetUserSyncer()` setter
- Both "ad" and "hybrid" modes now call `ad.SetUserSyncer(f.userSyncer)` before returning

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Migration number 085 already used**
- **Found during:** Task 1
- **Issue:** Plan specified `085_add_auth_source_fields.sql` but `085_split_dedicated_line_ip_fields.sql` already exists
- **Fix:** Used migration number 100 (next available after 099)
- **Files modified:** internal/core/db/migrations/100_add_auth_source_fields.sql
- **Commit:** 05b2cdc

**2. [Rule 3 - Blocking] Cross-package circular dependency between security and system**
- **Found during:** Task 4
- **Issue:** Plan had ADAuthenticator directly importing and using `system.UserSyncService`, but `security` package cannot import `services/system` without creating a circular or inverted dependency
- **Fix:** Defined `UserSyncer` interface in the `security` package. `UserSyncService.SyncADUser` implements this interface via Go duck typing. Added `SyncedUser` type for cross-package data transfer.
- **Files modified:** internal/core/security/authenticator.go, internal/services/system/user_sync_service.go
- **Commit:** 240b9f8

**3. [Rule 2 - Security] Graceful degradation on sync failure**
- **Found during:** Task 4
- **Issue:** Threat model T-19-06 requires transaction safety. If sync fails after successful AD authentication, the user should still be able to authenticate.
- **Fix:** When sync fails, ADAuthenticator returns NeedsSync=true instead of erroring. The auth handler can retry sync later or the user gets a limited session.
- **Files modified:** internal/core/security/ad_authenticator.go
- **Commit:** 240b9f8

## Threat Model Compliance

| Threat ID | Mitigation | Status |
|-----------|-----------|--------|
| T-19-06 | Database transactions for sync operations | Implemented: SyncUserFromAD uses tx.Transaction() |
| T-19-07 | Default password protection | Implemented: InitFlag=true forces password change on first login |
| T-19-08 | Default role permission control | Implemented: configurable via sys.auth.ad.default_role_id |
| T-19-09 | Duplicate user prevention | Implemented: partial unique index on ad_username WHERE NOT NULL |

## Known Stubs

None - all sync operations have full implementation logic.

## Threat Flags

None - all new surface (sync service, database columns) is covered by the plan's threat model.

## Self-Check: PASSED

All 7 files verified present. All 4 commits verified in git log.

---

**Execution completed:** 2026-05-21
**Duration:** 8 minutes
**Executor:** Claude (GSD Execute Phase - Parallel Worktree)
