---
phase: 62-ai-internal-core-db
plan: 03
subsystem: internal/core/db
tags: [C2, C5, OC-M-MENUSEED, CDX-M-USERROLE, TDD, seed-hardening, gorm-errors]

# Dependency graph
requires:
  - phase: 62-ai-internal-core-db (62-REVIEWS cross-AI review)
    provides: C2/C5/OC-M-MENUSEED/CDX-M-USERROLE consensus findings
provides:
  - env-overridable admin bootstrap password (SYS_ADMIN_BOOTSTRAP_PASSWORD)
  - loud WARN alert on admin123 fallback (no password value in logs)
  - cleared dead Salt literal in sys_user seed
  - per-subtree dept seed idempotency via ensureDept helper
  - DeptCode populated with stable unique values (ROOT/SHENZHEN/CHANGSHA/RD/MARKET/TEST)
  - errors.Is(err, ErrRecordNotFound) for menu seed loops (page + button)
  - parentMenuID empty-string guard in button loop
  - GORM-based sys_user_role creation (no hardcoded table-name SQL)
affects: [internal/core/db/init_data, internal/core/db/init_data_test]

tech-stack:
  added: []
  patterns: [env-driven seed bootstrap, errors.Is gorm.ErrRecordNotFound pattern, per-subtree idempotency]

key-files:
  created:
    - internal/core/db/init_data_test.go
  modified:
    - internal/core/db/init_data.go

key-decisions:
  - "D-62-03-01: C2 deferred to env override + WARN rather than first-login forced change (full scheme needs handler+frontend changes outside this phase)"
  - "D-62-03-02: C5 chose fine-grained per-subtree idempotency over full db.Transaction wrap (full wrap conflicts with core.go initDBAndData 'failure only warn' policy)"
  - "D-62-03-03: DeptCode populated with stable unique values to satisfy uniqueIndex;not null constraint (seed had latent UNIQUE collision bug pre-fix)"
  - "D-62-03-04: Test DB DSNs use cache=private (NOT shared) — shared cache caused cross-test row pollution that masked fallback-path assertion failures"

patterns-established:
  - "ensureDept helper pattern: query by natural key (dept_name + parent_id), write back existing row ID, create if missing, propagate real errors"
  - "errors.Is(err, gorm.ErrRecordNotFound) three-branch pattern: nil→exists, Is(ErrRecordNotFound)→create, else→return wrapped error"
  - "Always-populate-all-required-unique-fields-on-seed pattern: never assume non-constrained fields will stay empty on real DB"

requirements-completed: [C2, C5, OC-M-MENUSEED, CDX-M-USERROLE]

# Metrics
duration: 18min
completed: 2026-08-14
---

# Phase 62 Plan 03: Init Data Seed Hardening Summary

**Hardened `internal/core/db/init_data.go` seed layer: env-overridable admin bootstrap, per-subtree dept idempotency, errors.Is-aware menu loops, and ORM-based sys_user_role creation**

## Performance

- **Duration:** 18 min
- **Started:** 2026-08-14T10:53:36Z
- **Completed:** 2026-08-14T11:11:58Z
- **Tasks:** 3 (all TDD: RED → GREEN per task)
- **Files modified:** 2 (`init_data.go`, new `init_data_test.go`)

## Accomplishments

- C2: `createDefaultUser` now reads `SYS_ADMIN_BOOTSTRAP_PASSWORD` env var, falls back to `admin123` only with a multi-line `applogger.Warnf` alert covering 3 recovery points; dead `Salt: "default"` literal cleared with comment explaining why `User.Salt` is a dead field (real salt is embedded in `$sm3$iterations$salt$hash`)
- C5: `createDefaultDept` rewritten around new `ensureDept(db, dept, parentID)` helper — eliminates `count > 0` short-circuit, supports partial-state recovery (top pre-existing → fills 5 missing sub-depts), seeded `DeptCode` with stable unique values so first-boot seed no longer crashes on `uniqueIndex;not null`
- OC-M-MENUSEED: page and button menu loops now use `errors.Is(err, gorm.ErrRecordNotFound)` three-branch pattern; button loop added `parentMenuID == ""` guard against empty-UUID inserts from upstream page failure
- CDX-M-USERROLE: `createUserRoleRelations` now uses `db.Create(&models.UserRole{...})` instead of `db.Exec("INSERT INTO sys_user_role ...")` — survives `UserRole.TableName` renames

## Task Commits

Each task was committed atomically (TDD: RED→GREEN, single commit per task):

1. **Task 1: admin seed hardening (C2)** - `bb2329e` (feat)
2. **Task 2: per-subtree dept idempotency (C5)** - `68119aa` (feat)
3. **Task 3: menu seed error paths + UserRole ORM create** - `d93f024` (feat)

## Files Created/Modified

- `internal/core/db/init_data.go` — `createDefaultUser` (env override + WARN + dead salt), new `ensureDept` helper, rewritten `createDefaultDept` (3-level chain), `createOperationsManagementMenus` (errors.Is + parent guard), `createUserRoleRelations` (db.Create)
- `internal/core/db/init_data_test.go` — new file with `freshUserDB`/`freshDeptDB`/`freshUserRoleDB` helpers (all using `cache=private`), 4 admin tests, 3 dept tests, 1 UserRole sqlite test, 1 source-assertion test for menu error paths

## Decisions Made

- **D-62-03-01 (C2 scope):** Full first-login forced password reset requires handler + frontend changes; deferred. This plan ships env-override + WARN alert — enough to make the bad default auditable, not enough to force reset (documented as deferred).
- **D-62-03-02 (C5 strategy):** Per-subtree `ensureDept` chosen over full `db.Transaction` wrap because the latter conflicts with `core.go initDBAndData`'s "InitData failure only warn" policy (would block startup on partial seed).
- **D-62-03-03 (DeptCode population):** The original seed left `dept_code` empty, which silently violated the `uniqueIndex;not null` constraint added by migration 080 — first-boot seed would always crash. Populating with stable values (`ROOT`, `SHENZHEN`, `CHANGSHA`, `RD`, `MARKET`, `TEST`) was necessary to make the seed actually work and is consistent with the plan's scope (rewriting the helper anyway).
- **D-62-03-04 (Test DSN cache mode):** `file::memory:?cache=shared` (the existing convention in `menu_grant_helpers_test.go`) caused cross-test row pollution — `TestCreateDefaultUser_EnvOverride`'s admin row leaked into `TestCreateDefaultUser_FallbackDefault`, making the latter assert against a `Test@2026`-hashed row while verifying `admin123`. Switched to `cache=private` for init_data_test.go helpers. Pre-existing menu_grant_helpers_test.go unaffected (its tests don't depend on cross-test isolation).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Populated DeptCode with stable unique values**

- **Found during:** Task 2 (RED-phase `TestCreateDefaultDept_FullSeed`)
- **Issue:** Original seed had `dept_code: ""` on all 6 dept rows, but the model has `uniqueIndex;not null` (added by migration `080_add_dept_code_field.sql`). First-boot seed crashed with `UNIQUE constraint failed: sys_dept.dept_code` — the seed literally never worked since 080 was applied.
- **Fix:** Populated `DeptCode` with stable semantic values (`ROOT`, `SHENZHEN`, `CHANGSHA`, `RD`, `MARKET`, `TEST`). Minimal change consistent with the dept model requirements; brand names left as-is per plan ("品牌名/leader/电话/邮箱等种子值保持原样").
- **Files modified:** `internal/core/db/init_data.go`
- **Verification:** `TestCreateDefaultDept_FullSeed` (6 rows), `TestCreateDefaultDept_PartialRecovers`, `TestCreateDefaultDept_FullyIdempotent` all PASS; build clean
- **Committed in:** `68119aa` (Task 2 commit)

**2. [Rule 3 - Blocking] Fixed shared SQLite cache causing cross-test pollution**

- **Found during:** Task 1 GREEN — `TestCreateDefaultUser_FallbackDefault` failed when run after `TestCreateDefaultUser_EnvOverride` despite correct env handling
- **Issue:** `file::memory:?cache=shared` lets multiple `gorm.Open(sqlite.Open(...))` calls share the same in-memory database across test invocations. The admin row created by EnvOverride (with `Test@2026` hash) persisted into FallbackDefault, causing `createDefaultUser` to short-circuit on `count > 0` and return the prior hashed row — `VerifyPassword("admin123")` then legitimately failed.
- **Fix:** Changed all 3 test DB helpers in `init_data_test.go` to `cache=private`. Pre-existing `menu_grant_helpers_test.go` left untouched (its `freshSQLiteDB` uses `cache=shared` but its tests don't rely on cross-test isolation — verified by running them).
- **Files modified:** `internal/core/db/init_data_test.go`
- **Verification:** `TestCreateDefaultUser_EnvOverride`, `TestCreateDefaultUser_FallbackDefault`, `TestCreateDefaultUser_NoDefaultLiteralSalt` (env_path + fallback_path subtests), `TestCreateDefaultUser_Idempotent` all PASS when run together
- **Committed in:** `bb2329e` (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (1 missing-critical, 1 blocking)
**Impact on plan:** Both auto-fixes essential for the seed to actually work and for the test suite to be reliable. No scope creep; both directly serve the plan's C2/C5 intent.

## Issues Encountered

- Initial test failure (`TestCreateDefaultUser_FallbackDefault` after `EnvOverride`) required diagnostic work to identify the shared-cache row pollution — see Rule 3 deviation above.
- `TestSourceAssertions_MenuSeedErrorPaths` initially matched `"INSERT INTO sys_user_role"` in the docstring comment. Rephrased the comment to use prose description (`db.Exec 拼字符串 + 硬编码 sys_user_role 表名`) rather than embedding the literal SQL — preserves documentation value without tripping the regression assertion.
- `commitlint` body line-length (100 chars) hook rejected first Task 2 commit attempt — split bullets across more lines, retry succeeded.

## User Setup Required

None — no external service configuration required. The new env var `SYS_ADMIN_BOOTSTRAP_PASSWORD` is optional (fallback path still works for dev), but ops should set it in production per the WARN alert.

## Next Phase Readiness

- Seed layer hardened against the four consensus findings (C2/C5/OC-M-MENUSEED/CDX-M-USERROLE)
- Deferred items (first-login forced reset, full transactional seed wrap) documented for future planning
- Phase 62 plans 04/05 can proceed; remaining open items from the review (C1 Migrate176 schema-version check, C3 advisory lock, C4 was Phase 62-02) are tracked separately

---
*Phase: 62-ai-internal-core-db*
*Completed: 2026-08-14*