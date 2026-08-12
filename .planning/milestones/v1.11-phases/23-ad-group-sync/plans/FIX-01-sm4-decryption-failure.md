# FIX-01: SM4 Password Decryption Failure

**Status**: `diagnosed` → `planned`
**Severity**: Blocker
**Affects**: Tests 5,6,8,9,10 (Sync operations)
**Root Cause**: AD config passwords stored as plaintext or encrypted with different key/algorithm. `decryptPassword()` silently fails and returns ciphertext instead of propagating error.

## Problem Statement

When attempting AD group synchronization operations, the system fails with `cipher: message authentication failed` error. This occurs because:

1. **Historical passwords**: Some AD configs have plaintext passwords (stored before SM4 encryption was implemented)
2. **Silent failure**: `decryptPassword()` in `utils.go` returns ciphertext on decryption failure instead of error
3. **No backward compatibility**: System tries to decrypt plaintext as SM4-GCM, causing authentication failure
4. **Missing cipher initialization**: `encryptPassword()` returns plaintext when SM4 cipher is nil

## Objectives

1. ✅ Fix `decryptPassword()` to properly propagate errors
2. ✅ Add backward compatibility for plaintext passwords
3. ✅ Update all callers to handle decryption errors gracefully
4. ✅ Add data migration to detect and re-encrypt plaintext passwords
5. ✅ Ensure SM4 cipher is initialized before AD operations

## Implementation Plan

### Step 1: Fix decryptPassword() Error Handling

**File**: `internal/services/addomain/utils.go`

**Current Code** (lines 48-51):
```go
func decryptPassword(ciphertext string) string {
    plaintext, err := sm4.Decrypt(ciphertext)
    if err != nil {
        // Silent failure - returns ciphertext
        return ciphertext
    }
    return plaintext
}
```

**Target Code**:
```go
func decryptPassword(ciphertext string) (string, error) {
    plaintext, err := sm4.Decrypt(ciphertext)
    if err != nil {
        return "", fmt.Errorf("failed to decrypt AD password: %w", err)
    }
    return plaintext, nil
}

// looksLikeSM4Ciphertext checks if a string appears to be SM4-GCM encrypted
func looksLikeSM4Ciphertext(s string) bool {
    // SM4-GCM base64 output is typically longer than 32 chars and contains specific patterns
    if len(s) < 32 {
        return false
    }
    // Check if it's valid base64
    _, err := base64.StdEncoding.DecodeString(s)
    return err == nil
}

// decryptPasswordWithFallback tries SM4 decryption, falls back to plaintext
func decryptPasswordWithFallback(ciphertext string) (string, error) {
    // If it looks like SM4 ciphertext, try decryption
    if looksLikeSM4Ciphertext(ciphertext) {
        plaintext, err := sm4.Decrypt(ciphertext)
        if err == nil {
            return plaintext, nil
        }
    }
    // Fallback: assume plaintext (for historical data)
    return ciphertext, nil
}
```

### Step 2: Update encryptPassword() to Ensure Cipher Initialization

**File**: `internal/services/addomain/utils.go`

**Current Code** (lines 22-27):
```go
func encryptPassword(plaintext string) string {
    cipher := sm4.GetCipher()
    if cipher == nil {
        return plaintext // Returns plaintext if cipher not initialized
    }
    return sm4.Encrypt(plaintext)
}
```

**Target Code**:
```go
func encryptPassword(plaintext string) (string, error) {
    cipher := sm4.GetCipher()
    if cipher == nil {
        return "", fmt.Errorf("SM4 cipher not initialized. Call SetADSM4Cipher first")
    }
    return sm4.Encrypt(plaintext), nil
}
```

### Step 3: Update All Callers to Handle Errors

**File**: `internal/services/addomain/config.go`

**Update TestConnection**:
```go
// Before
password := decryptPassword(config.Password)

// After
password, err := decryptPasswordWithFallback(config.Password)
if err != nil {
    return fmt.Errorf("failed to decrypt AD config password: %w", err)
}
```

**File**: `internal/services/addomain/group_sync_service.go`

**Update all sync operations**:
```go
// Before
password := decryptPassword(config.Password)

// After
password, err := decryptPasswordWithFallback(config.Password)
if err != nil {
    return fmt.Errorf("failed to decrypt password for sync: %w", err)
}
```

### Step 4: Add Data Migration Script

**File**: `internal/core/db/migrations/135_fix_ad_password_encryption.sql`

```sql
-- Migration to detect and flag plaintext AD passwords
-- This adds a temporary column to track which passwords need re-encryption

ALTER TABLE sys_ad_config
ADD COLUMN password_needs_reencryption BOOLEAN DEFAULT FALSE;

-- Flag passwords that appear to be plaintext (shorter than typical SM4 ciphertext)
UPDATE sys_ad_config
SET password_needs_reencryption = TRUE
WHERE LENGTH(password) < 32;

-- Add comment for documentation
COMMENT ON COLUMN sys_ad_config.password_needs_reencryption IS 
'Temporary flag for migration 135: TRUE if password needs re-encryption with SM4';
```

**File**: `internal/services/addomain/migration.go` (new file)

```go
package addomain

import (
    "fmt"
    "gorm.io/gorm"
)

// MigratePasswords re-encrypts plaintext AD passwords with SM4
func MigratePasswords(db *gorm.DB) error {
    // Find all configs that need re-encryption
    var configs []ADConfig
    if err := db.Where("password_needs_reencryption = ?", true).Find(&configs).Error; err != nil {
        return fmt.Errorf("failed to find configs needing migration: %w", err)
    }

    for _, config := range configs {
        // Check if password is already SM4 encrypted
        if looksLikeSM4Ciphertext(config.Password) {
            // Already encrypted, just clear the flag
            db.Model(&config).Update("password_needs_reencryption", false)
            continue
        }

        // Re-encrypt with SM4
        encrypted, err := encryptPassword(config.Password)
        if err != nil {
            return fmt.Errorf("failed to encrypt password for config %s: %w", config.ID, err)
        }

        // Update the password
        if err := db.Model(&config).Updates(map[string]interface{}{
            "password": encrypted,
            "password_needs_reencryption": false,
        }).Error; err != nil {
            return fmt.Errorf("failed to update config %s: %w", config.ID, err)
        }
    }

    return nil
}
```

### Step 5: Ensure SM4 Cipher Initialization

**File**: `internal/core/core.go`

**Update Core initialization**:
```go
// In Initialize() function, ensure SM4 cipher is set early
func (c *Core) Initialize() error {
    // ... existing initialization code ...

    // Initialize SM4 cipher for AD password encryption
    if c.Config.Security.SM4.Key != "" {
        sm4.SetADSM4Cipher(c.Config.Security.SM4.Key)
        logrus.Info("SM4 cipher initialized for AD password encryption")
    } else {
        logrus.Warn("SM4 key not configured - AD password encryption disabled")
    }

    // ... rest of initialization ...
}
```

### Step 6: Add Migration Helper Endpoint

**File**: `internal/api/v1/addomain/migration_handler.go` (new file)

```go
package addomain

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/xingran-next/xingran-go-backend/pkg/response"
)

type MigrationHandler struct {
    db *gorm.DB
}

func NewMigrationHandler(db *gorm.DB) *MigrationHandler {
    return &MigrationHandler{db: db}
}

// TriggerPasswordMigration triggers the password re-encryption migration
func (h *MigrationHandler) TriggerPasswordMigration(c *gin.Context) {
    err := addomain.MigratePasswords(h.db)
    if err != nil {
        response.Error(c, http.StatusInternalServerError, err.Error())
        return
    }
    response.Success(c, gin.H{
        "message": "Password migration completed successfully",
    })
}
```

**File**: `internal/api/v1/addomain/group_sync_router.go`

**Add migration endpoint**:
```go
// SetupGroupSyncRouter - add this endpoint
r.POST("/migrate-passwords", migrationHandler.TriggerPasswordMigration)
```

## Verification Criteria

1. ✅ `decryptPassword()` returns `(string, error)` instead of `(string)`
2. ✅ `decryptPasswordWithFallback()` can handle both plaintext and SM4-encrypted passwords
3. ✅ All callers updated to handle decryption errors
4. ✅ Migration script runs successfully and re-encrypts plaintext passwords
5. ✅ SM4 cipher is initialized on application startup
6. ✅ AD group synchronization operations complete without `cipher: message authentication failed` error
7. ✅ All UAT tests 5,6,8,9,10 pass

## Testing Plan

1. **Unit Test**: Test `decryptPasswordWithFallback()` with both plaintext and SM4-encrypted passwords
2. **Integration Test**: Test `MigratePasswords()` with a database containing mixed password formats
3. **E2E Test**: Create AD config, trigger sync, verify no authentication failures
4. **UAT Verification**: Run UAT tests 5,6,8,9,10 to confirm blocker is resolved

## Rollback Plan

If issues occur:
1. Keep migration 135 reversible (can drop `password_needs_reencryption` column)
2. Maintain both old and new decryption functions temporarily
3. Can revert to old `decryptPassword()` signature if backward compatibility breaks

## Dependencies

- None (fix is self-contained within addomain service)
- Requires database migration 135 to be applied

## Estimated Effort

- **Step 1**: 30 minutes (update utils.go)
- **Step 2**: 15 minutes (update encryptPassword)
- **Step 3**: 45 minutes (update all callers)
- **Step 4**: 1 hour (migration script)
- **Step 5**: 15 minutes (cipher initialization)
- **Step 6**: 30 minutes (migration endpoint)
- **Testing**: 1 hour
- **Total**: ~4 hours

## Success Metrics

- All AD group sync operations succeed without SM4 decryption errors
- Historical AD configs with plaintext passwords are successfully migrated
- No regression in existing AD authentication functionality
- UAT completion rate increases from 1/10 to at least 8/10
