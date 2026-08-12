---
phase: 19-ad-auth-login
plan: 05
subsystem: authentication
tags: [strategy-pattern, auth-config, login-integration, sys_config, ad-auth, react, conditional-ui]

# Dependency graph
requires:
  - phase: 19-02
    provides: AuthStrategyFactory, Authenticator interface, three authenticators
  - phase: 19-03
    provides: UserSyncService, migration 100 auth_source fields, UserSyncer interface
provides:
  - AD auth config migration (124) with 5 sys_config parameters
  - Login handler integrated with AuthStrategyFactory
  - Core.AuthFactory singleton initialized with UserSyncService
  - Public GET /system/auth/config endpoint for frontend
  - Conditional auth mode selector on login page based on config
  - AuthMode type in frontend types
affects: [auth, login, config-management, frontend-login]

# Tech tracking
tech-stack:
  added: []
  patterns: [Config-driven auth mode selection, Public config endpoint pattern, Graceful factory fallback]

key-files:
  created:
    - internal/core/db/migrations/124_add_auth_config.sql
  modified:
    - internal/core/db/database.go
    - internal/api/v1/auth.go
    - internal/core/core.go
    - xingran-react-frontend/src/types/auth.ts
    - xingran-react-frontend/src/store/authStore.ts
    - xingran-react-frontend/src/pages/login/index.tsx

key-decisions:
  - "Migration numbered 124 (plan said 086) since 086 and 100 were already used"
  - "Login handler falls back to direct local auth when AuthFactory is nil (backward compatibility)"
  - "Public /system/auth/config endpoint added to unauthenticated auth routes for pre-login config"
  - "Auth mode selector hidden by default; only shown when sys.auth.ad.enabled=true"

patterns-established:
  - "Config-driven feature toggle: sys_config values control UI visibility and auth behavior"
  - "Public config endpoint pattern: unauthenticated GET endpoint for login page configuration"

requirements-completed: [AUTH-05]

# Metrics
duration: 19min
completed: 2026-05-22
---

# Phase 19 Plan 05: Auth Config and Login Integration Summary

**Login handler integrated with strategy pattern auth factory, config-driven AD auth toggle, and conditional frontend auth mode selector**

## Performance

- **Duration:** 19 min
- **Started:** 2026-05-21T16:44:38Z
- **Completed:** 2026-05-21T17:03:36Z
- **Tasks:** 5
- **Files modified:** 7

## Accomplishments
- Database migration adds 5 AD auth config parameters to sys_config (ad.enabled, default.mode, ad.config_id, default_role_id, default_dept_id)
- Login handler fully integrated with AuthStrategyFactory supporting local/ad/hybrid modes
- Core singleton AuthFactory initialized with UserSyncService for automatic AD user sync
- Frontend login page conditionally shows auth mode selector based on server config
- Public auth config endpoint enables pre-login configuration discovery

## Task Commits

Each task was committed atomically:

1. **Task 1: AD auth config migration** - `8d9c20b` (feat)
2. **Task 2+3: Login integration + Core factory init** - `800a341` (feat)
3. **Task 4: AuthMode type and store pass-through** - `97983c9` (feat)
4. **Task 5: Auth config endpoint + conditional selector** - `ea93855` (feat)

## Files Created/Modified
- `internal/core/db/migrations/124_add_auth_config.sql` - SQL migration for 5 AD auth config params
- `internal/core/db/database.go` - Added createADAuthConfig() to initData flow
- `internal/api/v1/auth.go` - Refactored login() to use strategy pattern; added AuthMode to LoginRequest; added getAuthConfig public endpoint; added loginLocalDirect fallback
- `internal/core/core.go` - Added AuthFactory field, initAuthFactory(), GetAuthFactory(); factory initialized with UserSyncService
- `xingran-react-frontend/src/types/auth.ts` - Added AuthMode type, authMode to LoginRequest
- `xingran-react-frontend/src/store/authStore.ts` - Passes authMode from credentials to login API
- `xingran-react-frontend/src/pages/login/index.tsx` - Loads auth config on mount; conditionally shows auth mode selector based on adEnabled

## Decisions Made
- Migration numbered 124 instead of plan's 086 since 086 and 100 were already occupied
- Combined Tasks 2 and 3 into single commit since auth.go changes depend on Core.AuthFactory which must exist simultaneously
- Added loginLocalDirect() fallback function for when AuthFactory is nil, ensuring backward compatibility if factory initialization fails
- Public GET /system/auth/config endpoint added to unauthenticated routes so login page can fetch config before authentication
- Auth mode selector hidden by default (adEnabled=false); only visible when admin enables AD auth in config

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Migration number 086 already used**
- **Found during:** Task 1
- **Issue:** Plan specified 086_add_auth_config.sql but 086_remove_gateway_column.sql already exists; 100_add_auth_source_fields.sql also exists
- **Fix:** Used migration number 124 (next available)
- **Files modified:** internal/core/db/migrations/124_add_auth_config.sql
- **Committed in:** 8d9c20b

**2. [Rule 3 - Blocking] SQL migration files are reference-only; actual seeding via Go code**
- **Found during:** Task 1
- **Issue:** SQL migration files in this project are documentation only; actual config seeding happens through Go code in database.go initData
- **Fix:** Added createADAuthConfig() Go function following the existing createRequestEncryptionToggleConfig pattern, and called it from initData
- **Files modified:** internal/core/db/database.go
- **Committed in:** 8d9c20b

**3. [Rule 2 - Security] Config endpoint must be public (unauthenticated)**
- **Found during:** Task 5
- **Issue:** Plan suggested creating services/config.ts with getAuthConfig calling /system/auth/config, but no such endpoint existed and the config routes are behind auth middleware
- **Fix:** Added GET /system/auth/config to the unauthenticated auth router group (same level as /public-key and /login). Only exposes adEnabled (boolean) and defaultMode (string) - no sensitive AD config details (threat model T-19-17)
- **Files modified:** internal/api/v1/auth.go
- **Committed in:** ea93855

---

**Total deviations:** 3 auto-fixed (2 blocking, 1 security)
**Impact on plan:** All auto-fixes necessary for correctness and security. No scope creep.

## Issues Encountered
- Pre-existing apikey_service.go compilation error (unrelated, out of scope) - did not block our changes
- Tab/space indentation inconsistencies across files required careful sed-based edits

## User Setup Required
None - no external service configuration required.

## Known Stubs
None - all functionality has full implementation.

## Threat Flags
None - new public endpoint only exposes adEnabled (boolean) and defaultMode (string), aligned with threat model T-19-17.

## Self-Check: PASSED

All 7 files verified present. All 4 commits verified in git log.

---
*Phase: 19-ad-auth-login*
*Completed: 2026-05-22*
