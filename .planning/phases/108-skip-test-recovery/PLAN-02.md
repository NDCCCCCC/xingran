---
phase: 108
plan: "02"
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/core/security/authenticator_test.go
  - internal/core/security/local_authenticator_test.go
  - internal/services/system/user_sync_service_test.go
autonomous: true
requirements_addressed: [SKIP-02]
---

<objective>
Restore 13 skipped tests using embedded sqlite :memory: database. Part A of SKIP-02 covers local_authenticator_test.go (7 tests) and user_sync_service_test.go (6 tests). The key insight: `setupSecurityTestDB` and `createTestUser` already exist in `integration_test.go` — the fix is redirecting `setupTestDB`/`setupTestDBForSync` to use them.
</objective>

<tasks>

## Context

In `internal/core/security/integration_test.go`:
- `setupSecurityTestDB(t *testing.T) *gorm.DB` creates a sqlite :memory: database with `sys_user`, `sys_config`, and `sys_ad_config` tables
- `createTestUser(t *testing.T, db *gorm.DB, username, plainPassword string, status models.UserStatus) *models.User` creates a test user with SM3-hashed password

Both are in the `security` package and accessible to all test files in the same package.

## Task 1: Fix setupTestDB in authenticator_test.go

**File:** `internal/core/security/authenticator_test.go`

Replace the stub `setupTestDB` function (lines 32-38) with a call to `setupSecurityTestDB`:

```go
// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
    return setupSecurityTestDB(t)
}
```

This single line replaces the TODO/Skip stub. `setupSecurityTestDB` already creates the `sys_user` table with all required columns (username, password, salt, status, etc.) that `LocalAuthenticator.Authenticate` queries.

The `createTestUser` helper is already defined in `integration_test.go` and is accessible from `local_authenticator_test.go` (same package).

## Task 2: Remove t.Skip from local_authenticator_test.go

**File:** `internal/core/security/local_authenticator_test.go`

For each of the 7 tests, remove the `t.Skip(...)` block. All tests already call `setupTestDB(t)` and check for nil before proceeding — once `setupTestDB` returns a real db pointer, the `if db == nil { t.Skip(...) }` guard is no longer triggered.

Tests to update (remove Skip, no other changes needed):

| Line | Test Function |
|------|---------------|
| 17 | `TestLocalAuthenticator_Authenticate_Success` |
| 45 | `TestLocalAuthenticator_Authenticate_UserNotFound` |
| 65 | `TestLocalAuthenticator_Authenticate_InvalidPassword` |
| 88 | `TestLocalAuthenticator_Authenticate_UserDisabled` |
| 111 | `TestLocalAuthenticator_Authenticate_SM3PasswordVerification` |
| 151 | `TestLocalAuthenticator_Name` |
| 164 | `TestLocalAuthenticator_TableDrivenTests` |

Each test already has proper assertions — just removing the Skip and nil-check guard is sufficient.

## Task 3: Implement setupTestDBForSync in user_sync_service_test.go

**File:** `internal/services/system/user_sync_service_test.go`

Replace the stub `setupTestDBForSync` (lines 13-17) with a sqlite :memory: setup matching the pattern in `integration_test.go`. The `user_sync_service_test.go` needs `sys_user`, `sys_dept`, `sys_role`, `sys_user_role` tables for the sync tests.

```go
// setupTestDBForSync creates an in-memory SQLite database for user sync testing.
func setupTestDBForSync(t *testing.T) *gorm.DB {
    t.Helper()
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
        DisableForeignKeyConstraintWhenMigrating: true,
    })
    if err != nil {
        t.Skipf("Failed to open test database: %v", err)
        return nil
    }

    // Create sys_user table (same schema as integration_test.go)
    err = db.Exec(`
        CREATE TABLE IF NOT EXISTS sys_user (
            id TEXT PRIMARY KEY,
            created_at DATETIME,
            updated_at DATETIME,
            deleted_at DATETIME,
            created_by TEXT,
            updated_by TEXT,
            version INTEGER DEFAULT 1,
            username TEXT NOT NULL UNIQUE,
            password TEXT NOT NULL,
            salt TEXT NOT NULL DEFAULT '',
            nickname TEXT,
            employee_no TEXT,
            email TEXT,
            phone TEXT,
            avatar TEXT,
            gender INTEGER DEFAULT 0,
            status INTEGER DEFAULT 0,
            dept_id TEXT,
            dept_name TEXT,
            login_ip TEXT,
            login_time DATETIME,
            pwd_update_time DATETIME,
            pwd_expire_days INTEGER DEFAULT 90,
            init_flag INTEGER DEFAULT 0,
            remark TEXT DEFAULT '',
            auth_source TEXT NOT NULL DEFAULT 'local',
            ad_username TEXT,
            ad_dn TEXT,
            ad_ou_dn TEXT,
            ad_synced_at DATETIME
        )
    `).Error
    if err != nil {
        t.Skipf("Failed to create sys_user table: %v", err)
        return nil
    }

    // Create sys_dept table
    err = db.Exec(`
        CREATE TABLE IF NOT EXISTS sys_dept (
            id TEXT PRIMARY KEY,
            created_at DATETIME,
            updated_at DATETIME,
            deleted_at DATETIME,
            created_by TEXT,
            updated_by TEXT,
            version INTEGER DEFAULT 1,
            parent_id TEXT,
            dept_name TEXT NOT NULL,
            dept_code TEXT,
            sort_order INTEGER DEFAULT 0,
            leader TEXT,
            phone TEXT,
            email TEXT,
            status INTEGER DEFAULT 0,
            remark TEXT DEFAULT ''
        )
    `).Error
    if err != nil {
        t.Skipf("Failed to create sys_dept table: %v", err)
        return nil
    }

    // Create sys_role table
    err = db.Exec(`
        CREATE TABLE IF NOT EXISTS sys_role (
            id TEXT PRIMARY KEY,
            created_at DATETIME,
            updated_at DATETIME,
            deleted_at DATETIME,
            created_by TEXT,
            updated_by TEXT,
            version INTEGER DEFAULT 1,
            role_name TEXT NOT NULL,
            role_code TEXT,
            sort_order INTEGER DEFAULT 0,
            status INTEGER DEFAULT 0,
            data_scope INTEGER DEFAULT 1,
            remark TEXT DEFAULT ''
        )
    `).Error
    if err != nil {
        t.Skipf("Failed to create sys_role table: %v", err)
        return nil
    }

    // Create sys_user_role table
    err = db.Exec(`
        CREATE TABLE IF NOT EXISTS sys_user_role (
            id TEXT PRIMARY KEY,
            created_at DATETIME,
            updated_at DATETIME,
            deleted_at DATETIME,
            created_by TEXT,
            updated_by TEXT,
            version INTEGER DEFAULT 1,
            user_id TEXT NOT NULL,
            role_id TEXT NOT NULL,
            UNIQUE(user_id, role_id)
        )
    `).Error
    if err != nil {
        t.Skipf("Failed to create sys_user_role table: %v", err)
        return nil
    }

    return db
}
```

Note: Also add the required import:
```go
import (
    "github.com/glebarez/sqlite"
)
```

If `sqlite` is not yet imported in that file, add it to the import block.

## Task 4: Remove t.Skip from user_sync_service_test.go

**File:** `internal/services/system/user_sync_service_test.go`

For each of the 6 tests, remove the `if db == nil { t.Skip(...) }` guard. The tests already call `setupTestDBForSync(t)` — once it returns a real db pointer, the guard is bypassed.

Tests to update:

| Line | Test Function |
|------|---------------|
| 23 | `TestUserSyncService_SyncUserFromAD_FirstTime` |
| 55 | `TestUserSyncService_SyncUserFromAD_UpdateExisting` |
| 101 | `TestUserSyncService_SyncUserFromAD_TransactionRollback` |
| 131 | `TestUserSyncService_SyncUserFromAD_RoleAssignment` |
| 171 | `TestUserSyncService_SyncUserFromAD_DepartmentAssignment` |
| 214 | `TestUserSyncService_SyncUserFromAD_TableDrivenTests` |

## Task 5: Verify

Run both test suites:

```bash
go test ./internal/core/security/ -run "TestLocalAuth" -v
go test ./internal/services/system/ -run "TestUserSync" -v
```

All 13 tests should pass without Skip.

</tasks>

<success_criteria>
- `go test ./internal/core/security/ -run "TestLocalAuth" -v` passes without Skip (7 tests)
- `go test ./internal/services/system/ -run "TestUserSync" -v` passes without Skip (6 tests)
- `TestLocalAuthenticator_Authenticate_Success` passes
- `TestUserSyncService_SyncUserFromAD_FirstTime` passes
</success_criteria>

<output>
Create SUMMARY.md on completion
</output>
