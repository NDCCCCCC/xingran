---
phase: 16-api-key-mgt
plan: 02
subsystem: api-key-management, backend-service
tags: [api-keys, service-layer, crud, crypto, gorm, handler-service-pattern]

# Dependency graph
requires: [16-01]
provides:
  - APIKeyService interface and implementation
  - API key generation (crypto/rand)
  - API key validation (format, expiration, status)
  - CRUD operations for API keys
  - Request/response models for API keys
affects: [16-03, 16-04, 16-05]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Handler-Service pattern with interface-based design
    - crypto/rand for secure key generation
    - GORM JSONB for array storage (scopes, ip_whitelist)
    - Async last_used_at updates (goroutine)
    - RFC3339 time parsing for expiration

key-files:
  created:
    - internal/models/system/requests/apikey_request.go
    - internal/services/system/apikey_service.go
  modified: []

key-decisions:
  - "No caching for security (always validate against database)"
  - "Full key only returned once during creation"
  - "Key format: rec_ + 64 hex chars (32 bytes random)"
  - "Max 100 keys per user (resource limit)"
  - "JSONB storage for scopes and ip_whitelist arrays"

patterns-established:
  - "Service interface pattern (APIKeyService)"
  - "Private implementation struct (apiKeyServiceImpl)"
  - "Constructor dependency injection (NewAPIKeyService)"
  - "Context propagation (db.WithContext)"
  - "Error handling with apperrors wrap"

requirements-completed: []

# Metrics
duration: 20min
completed: 2026-05-19
---

# Phase 16: Plan 02 Summary

**API密钥管理的服务层 - 基础CRUD功能，包含密钥生成、验证和CRUD操作**

## Performance

- **Duration:** 20 min
- **Started:** 2026-05-19T00:42:19Z
- **Completed:** 2026-05-19T01:02:00Z
- **Tasks:** 4
- **Files created:** 2
- **Commits:** 2

## Accomplishments

- Created `CreateAPIKeyRequest`, `UpdateAPIKeyRequest`, `ListAPIKeysParams` request models
- Created `APIKeyService` interface with 7 methods (Create, List, Get, Update, Delete, Toggle, Validate)
- Implemented `apiKeyServiceImpl` with GORM database operations
- Added secure key generation using `crypto/rand` (32 bytes → 64 hex chars)
- Implemented key format validation (`rec_` prefix + 64 hex characters)
- Implemented CRUD operations with user validation, scope validation, and JSONB handling
- Added async last_used_at updates using goroutine

## Task Commits

Each task was committed atomically:

1. **Tasks 1 & 2: Create request models and service interface** - `6eead31` (feat)
2. **Task 4: Implement CRUD operations** - `dc18a8d` (feat)

## Files Created/Modified

### Created

- `internal/models/system/requests/apikey_request.go` - Request/response models
  - `CreateAPIKeyRequest`: name, description, scopes, inheritPerms, ipWhitelist, expiresAt
  - `UpdateAPIKeyRequest`: updateable fields
  - `ListAPIKeysParams`: pagination, keyword, status, scope filters
  - `GetPagination()`, `DefaultListAPIKeysParams()` helper methods

- `internal/services/system/apikey_service.go` - Service implementation (374 lines)
  - `APIKeyService` interface: 7 methods
  - `apiKeyServiceImpl` struct with db dependency
  - `NewAPIKeyService()` constructor
  - `CreateAPIKey()`: validate user, check limit, generate key, store
  - `ListAPIKeys()`: filter by keyword/status/scope, pagination
  - `GetAPIKey()`: retrieve with User preload
  - `UpdateAPIKey()`: update fields, validate scopes
  - `DeleteAPIKey()`: soft delete
  - `ToggleAPIKeyStatus()`: flip is_active boolean
  - `ValidateAPIKey()`: format check, expiration check, async update
  - Helper functions: `generateKey()`, `isKeyExpired()`, `isValidKeyFormat()`, `validateScopes()`

### Modified

- None

## Decisions Made

1. **No Caching for Security (T-16-08 mitigation)**
   - API key validation always queries database
   - Prevents stale key data from being used after revocation
   - Performance acceptable for authentication path

2. **Key Format Design**
   - Prefix `rec_` for easy identification
   - 64 hex characters (32 bytes) for sufficient entropy
   - Total length 68 chars for validation

3. **User Validation on Create (T-16-10 mitigation)**
   - Check user exists before creating key
   - Prevents orphaned keys referencing non-existent users

4. **Max Keys Limit (T-16-09 mitigation)**
   - 100 keys per user prevents resource exhaustion
   - Enforced before key generation

5. **Async Last Used Update**
   - goroutine updates `last_used_at` after validation
   - Doesn't block authentication request

6. **JSONB for Array Fields**
   - scopes stored as JSONB array
   - ip_whitelist stored as JSONB array
   - Allows PostgreSQL queries with `@>` operator

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed PageResult redeclaration error**
- **Found during:** Task 2 (Service interface creation)
- **Issue:** PageResult struct declared in both user_service.go and apikey_service.go
- **Fix:** Removed PageResult from apikey_service.go, added comment referencing user_service.go
- **Files modified:** internal/services/system/apikey_service.go
- **Verification:** `go build ./internal/services/system/...` passes
- **Committed in:** `6eead31` (Task 2 commit)

**2. [Rule 3 - Blocking] Fixed CodeNotFound undefined error**
- **Found during:** Task 4 (CRUD implementation)
- **Issue:** Used `apperrors.CodeNotFound` which doesn't exist in pkg/errors
- **Fix:** Replaced with `apperrors.CodeParamError` (follows existing pattern)
- **Files modified:** internal/services/system/apikey_service.go (4 occurrences)
- **Verification:** `go build ./internal/services/system/...` passes
- **Committed in:** `dc18a8d` (Task 4 commit)

**3. [Rule 3 - Blocking] Added encoding/json import**
- **Found during:** Task 4 (CRUD implementation)
- **Issue:** Needed json.Marshal for scopes and ipWhitelist JSONB fields
- **Fix:** Added "encoding/json" to imports
- **Files modified:** internal/services/system/apikey_service.go
- **Verification:** Build passes after import added
- **Committed in:** `dc18a8d` (Task 4 commit)

---

**Total deviations:** 3 auto-fixed (all Rule 3 - blocking issues)
**Impact on plan:** All fixes necessary for code to compile. No scope creep.

## Issues Encountered

1. **PageResult struct redeclaration**
   - Go doesn't allow duplicate type names in same package
   - Solution: Reference existing definition in user_service.go

2. **Missing error code constant**
   - `CodeNotFound` doesn't exist in error package
   - Solution: Use `CodeParamError` instead (follows existing user_service pattern)

3. **Missing import for JSON marshaling**
   - Need to serialize scopes and ipWhitelist to JSONB
   - Solution: Added encoding/json import

All issues resolved without changing scope or requirements.

## Known Stubs

**None** - All CRUD methods are fully implemented with real database operations. No placeholder functionality.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: tampering | internal/services/system/apikey_service.go | Key generation uses crypto/rand - T-16-07 mitigated |
| threat_flag: disclosure | internal/services/system/apikey_service.go | Full key only returned once on creation - T-16-08 mitigated |
| threat_flag: dos | internal/services/system/apikey_service.go | MaxKeysPerUser=100 prevents resource exhaustion - T-16-09 mitigated |
| threat_flag: spoofing | internal/services/system/apikey_service.go | User validation before key creation - T-16-10 mitigated |

**All threat mitigations from plan implemented successfully.**

## Next Phase Readiness

**Ready for Phase 16-03 (API密钥管理的处理器层 - HTTP接口实现):**
- ✅ APIKeyService interface complete
- ✅ All CRUD methods implemented
- ✅ Request/response models defined
- ✅ Build verification passes

**Dependencies for next phase:**
- 16-03 needs apikey_service.go for handler integration
- 16-03 needs apikey_request.go models for request binding

**Recommendations:**
- Verify database migration (16-01) runs before testing CRUD operations
- Test key generation produces valid format (rec_ + 64 hex)
- Test validation rejects invalid formats and expired keys
- Verify user validation prevents orphaned keys

---
*Phase: 16-api-key-mgt*
*Completed: 2026-05-19*
