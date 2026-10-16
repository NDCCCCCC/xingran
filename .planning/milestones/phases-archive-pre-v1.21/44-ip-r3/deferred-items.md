# Phase 44 Deferred Items (out-of-scope discoveries)

## Pre-existing test failures (NOT caused by Phase 44)

### TestValidator_ValidateFloor / ValidateWall / ValidateDoor

- **Discovered during:** Phase 44 Plan 02 Task 4 (broader test sweep)
- **Tests:** `TestValidator_ValidateFloor/存在的楼层`, `TestValidator_ValidateWall/存在的墙体`, `TestValidator_ValidateDoor/存在的门`
- **File:** `internal/services/operations/validation_helper_test.go`
- **Failure:** `ValidateFloor() error code = 1500, wantErrorCode 0` (and Wall/Door equivalents)
- **Verified pre-existing:** `git stash` (revert Phase 44-02 changes) → failures still occur on previous HEAD (`3f2e205a`).
- **Scope:** Floor plan 3D editor validation helper. Completely unrelated to asset reconciliation / Excel / exception rules.
- **Action:** Logged for future cleanup. Do NOT fix in Phase 44 (scope boundary per executor deviation rules).
