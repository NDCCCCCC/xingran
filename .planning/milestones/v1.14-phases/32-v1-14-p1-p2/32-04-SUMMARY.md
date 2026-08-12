---
phase: 32-v1-14-p1-p2
plan: 04
subsystem: business-logic
tags: [config, encryption, gorm, sqlite, middleware-cache, regression-test]

# Dependency graph
requires:
  - phase: 32-v1-14-p1-p2
    plan: 03
    provides: "Wave 3 P2/P1 implementation work — config invalidation was already wired (commit 0bcac33) and buildDepartmentPaths was already deduplicated (commit ab0a279); wave 4 adds regression coverage and grep verification"
provides:
  - "Regression test (config_invalidation_test.go) proving OnEncryptionConfigChanged callback fires once on encryption flag update and zero times on non-encryption updates"
  - "Nil-callback safe-path test (TestConfigService_UpdateEncryptionFlag_NilCallbackSafe) ensuring Update does not panic when callback is unset"
  - "Grep-based verification that buildDepartmentPaths is called exactly once in userService.List (P1-B2)"
  - "Closure of P1-B1 and P1-B2 — all 15 P1 items now resolved"
affects: [config-service, user-service, encryption-middleware, future-cache-invalidation-patterns]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "In-memory SQLite per-test isolation via unique DSN (file:<testName>?mode=memory&cache=private) to avoid UNIQUE-constraint cross-contamination from shared in-memory caches"
    - "Package-level callback registration (OnEncryptionConfigChanged) with t.Cleanup-based restore for test isolation"
    - "Seed-via-raw-SQL helper (seedConfig) bypassing GORM hooks to keep tests deterministic"

key-files:
  created:
    - internal/services/system/config_invalidation_test.go
  modified: []

key-decisions:
  - "Use unique-per-test SQLite cache (file:<testName>?mode=memory&cache=private) instead of shared 'file::memory:?cache=shared' to prevent UNIQUE constraint leakage between sub-tests (apikey_service_test.go uses shared mode and has cross-test pollution — pre-existing issue, not in scope)"
  - "Wrap OnEncryptionConfigChanged save/restore in t.Cleanup so a panic in one test still restores the package variable for subsequent tests"
  - "Insert seed rows via raw SQL (seedConfig) rather than GORM Create to avoid BaseModel.BeforeCreate UUID generation overhead and ensure deterministic test IDs"

patterns-established:
  - "Pattern: callback-driven cache invalidation — service layer exposes package-level func var, caller (core.Init) injects implementation at startup, service test swaps the var in/out via t.Cleanup"
  - "Pattern: nil-callback safety — services must not panic when callback is unset (testing, standalone build); guards with explicit nil check before invocation"

requirements-completed: [P1-B1, P1-B2]

# Metrics
duration: 8min
completed: 2026-06-13
---

# Phase 32 Plan 04: Wave 4 Business Logic Closeout Summary

**P1-B1 regression-tested config cache invalidation + P1-B2 verified buildDepartmentPaths single-call, closing all 15 P1 items**

## Performance

- **Duration:** 8 min
- **Started:** 2026-06-13T12:53:00Z
- **Completed:** 2026-06-13T13:01:00Z
- **Tasks:** 1
- **Files modified:** 1 (created, 0 production edits)

## Accomplishments

- Created `config_invalidation_test.go` with 4 sub-tests covering the encryption-flag update path, non-encryption isolation, and nil-callback safety
- Verified `buildDepartmentPaths` is called exactly once in `userService.List` (line 398); function definition at line 409 and a doc comment at line 408 account for the other 2 grep matches
- P1-B1 + P1-B2 closed — all 15 P1 items now resolved (per `32-RESEARCH.md` P1 fix tracking)
- `go build ./...` and `go vet ./...` both exit 0; targeted test passes 100%

## Task Commits

Each task was committed atomically:

1. **Task 1: Add ConfigService invalidation test + verify buildDepartmentPaths single call (P1-B1 + P1-B2)** - `d4adb50` (test)

**Plan metadata:** _(SUMMARY.md commit to follow)_

## Files Created/Modified

- `internal/services/system/config_invalidation_test.go` (created) — Regression coverage for the `OnEncryptionConfigChanged` callback wired by commit `0bcac33`; sub-tests:
  - `encryption_false_to_true_invalidates_cache` — callback fires exactly once, DB value updates
  - `encryption_true_to_false_invalidates_cache` — callback fires exactly once, DB value updates
  - `non_encryption_flag_does_not_invalidate_cache` — callback fires zero times for `sys.user.init.password`
  - `NilCallbackSafe` — Update does not panic when callback is unset

## Decisions Made

- **Per-test SQLite DSN isolation**: Used `file:<testName>?mode=memory&cache=private` (test-name-derived) rather than the shared `file::memory:?cache=shared` pattern in `apikey_service_test.go`. Rationale: shared mode leaks `sys_config` rows between sub-tests and triggers UNIQUE-constraint failures; private mode gives each sub-test a fresh in-memory DB.
- **t.Cleanup for callback restore**: Wrap save/restore of `OnEncryptionConfigChanged` in `t.Cleanup` so the package-level variable is always reset, even on test panic.
- **Raw-SQL seeding**: `seedConfig` uses `db.Exec(INSERT ...)` directly to avoid GORM `BaseModel.BeforeCreate` UUID generation; tests use stable UUIDs for assertions on `id` and to keep setup cheap.
- **No production code changes**: Per `32-RESEARCH.md` and the plan directive, both P1-B1 (commit `0bcac33`) and P1-B2 (commit `ab0a279`) are already fixed; this plan adds regression coverage and verification only.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Test infrastructure per-test isolation**
- **Found during:** Task 1 (running the new test for the first time)
- **Issue:** Initial implementation used `file::memory:?cache=shared` (the pattern from `apikey_service_test.go`). The shared in-memory DB leaked `sys_config` rows between sub-tests, triggering `UNIQUE constraint failed: sys_config.config_key` on the second `seedConfig` for the same key.
- **Fix:** Switched `setupConfigTestDB` to derive a unique DSN from `t.Name()` (`file:<safeName>?mode=memory&cache=private`) so each sub-test gets a fresh in-memory DB.
- **Files modified:** `internal/services/system/config_invalidation_test.go`
- **Verification:** Re-ran `go test -count=1 -run TestConfigService_UpdateEncryptionFlag ./internal/services/system/ -v` — all 4 sub-tests pass.
- **Committed in:** `d4adb50` (part of task commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Auto-fix was essential for test correctness. No scope creep.

## Issues Encountered

- **Pre-existing apikey_service_test.go failures** (out of scope): When running the full `go test ./internal/services/system/` suite, the pre-existing `apikey_service_test.go` tests fail with cross-test data leakage and nil-pointer panics. Confirmed unrelated to this plan by:
  1. Running my new tests in isolation — all pass.
  2. Running `apikey_service_test.go` in isolation (`-run TestCreateAPIKey`) — still fails the same way.
  3. The failures occur because `apikey_service_test.go` uses `file::memory:?cache=shared` (shared in-memory cache) and assumes per-test isolation that the shared mode does not provide.
  
  Per SCOPE BOUNDARY rule ("Only auto-fix issues DIRECTLY caused by the current task's changes"), I logged this as out-of-scope and did not modify it. Recommendation: file as a separate quick-plan or include in a future plan.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All 15 P1 items closed
- Wave 4 (Business Logic Closeout) complete
- Ready for Wave 5 onward (per `32-RESEARCH.md` P2 architectural debt items, or any P3 follow-ups)

## Self-Check: PASSED

- `internal/services/system/config_invalidation_test.go` exists (211 lines, 4 sub-tests)
- `d4adb50` commit present in git log: `test(32-04): P1-B1 config invalidation regression coverage`
- `d3d9ad7` commit present in git log: `docs(32-04): complete Wave 4 Business Logic Closeout plan`
- `go build ./...` exits 0
- `go vet ./...` exits 0
- `go test -count=1 -run TestConfigService_UpdateEncryptionFlag ./internal/services/system/ -v` passes 4/4 sub-tests
- `grep -n "buildDepartmentPaths" internal/services/system/user_service.go` shows exactly 1 call site (line 398)

---

*Phase: 32-v1-14-p1-p2*
*Completed: 2026-06-13*
