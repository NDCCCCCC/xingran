# Phase 19 Plan 02: Strategy Pattern Authentication System Summary

---
phase: 19
plan: 02
subsystem: authentication
tags: [strategy-pattern, authenticator, ldap, sm3, local-auth, ad-auth, hybrid-auth]
dependency_graph:
  requires: [19-01]
  provides: [Authenticator interface, LocalAuthenticator, ADAuthenticator, HybridAuthenticator, AuthStrategyFactory]
  affects: [internal/core/security, internal/services/addomain]
tech_stack:
  added: [go-ldap/ldap/v3 (existing), gorm.io/gorm (existing)]
  patterns: [Strategy Pattern, Factory Pattern]
key_files:
  created:
    - internal/core/security/authenticator.go
    - internal/core/security/local_authenticator.go
    - internal/core/security/ad_authenticator.go
    - internal/core/security/hybrid_authenticator.go
    - internal/core/security/auth_strategy_factory.go
  modified:
    - internal/services/addomain/utils.go
decisions:
  - ADAuthenticator uses direct LDAP dial+bind instead of addomain.LDAPClient to avoid admin-bind-first requirement
  - Exported DecryptPassword from addomain package as public API for reuse
  - HybridAuthenticator returns local auth error when both fail (more generic, prevents info leakage)
  - AD auth gracefully degrades to basic info if admin search fails after user bind succeeds
metrics:
  duration: 6m
  completed: 2026-05-21
  tasks_total: 5
  tasks_completed: 5
  files_created: 5
  files_modified: 1
  commit_count: 5
---

## One-liner

Strategy pattern authentication system with three authenticators (local SM3, AD LDAP bind, hybrid fallback) and a config-driven factory for runtime mode selection.

## Completed Tasks

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Authenticator interface definition | 44bed1c | internal/core/security/authenticator.go |
| 2 | Local authenticator with SM3 | fe4a8df | internal/core/security/local_authenticator.go |
| 3 | AD authenticator with LDAP bind | 33bd97f | internal/core/security/ad_authenticator.go, internal/services/addomain/utils.go |
| 4 | Hybrid authenticator with fallback | e6f3c12 | internal/core/security/hybrid_authenticator.go |
| 5 | Auth strategy factory | 205e318 | internal/core/security/auth_strategy_factory.go |

## What Was Built

### Authenticator Interface (`authenticator.go`)
- `Authenticator` interface with `Authenticate(ctx, req)` and `Name()` methods
- `AuthRequest` struct: Username, Password, IP
- `AuthResult` struct: User (UserResult), AuthSource, ADUserInfo, NeedsSync flag
- `UserResult`: simplified user info avoiding circular dependencies
- `ADUserInfo`: AD user attributes for sync to sys_user
- Standard error vars: `ErrUserNotFound`, `ErrInvalidCredentials`, `ErrUserDisabled`, `ErrADConfigNotFound`, `ErrADConnectionFailed`

### LocalAuthenticator (`local_authenticator.go`)
- Queries `sys_user` table by username
- Checks user status (0=enabled, 1=disabled)
- Verifies password using SM3-PBKDF2 via `PasswordManager.VerifyPassword()`
- Returns `UserResult` with user details, AuthSource="local", NeedsSync=false

### ADAuthenticator (`ad_authenticator.go`)
- Reads AD config from `sys_ad_config` table (by configID, status=0)
- Dials LDAP connection (supports LDAPS/LDAP+StartTLS/plain)
- Authenticates user via UPN bind: `username@domain.com`
- After successful user bind, connects as admin to search user attributes
- Graceful degradation: returns basic ADUserInfo if admin search fails
- Uses `ldap.EscapeFilter()` to prevent LDAP injection (T-19-01 mitigation)

### HybridAuthenticator (`hybrid_authenticator.go`)
- Tries `LocalAuthenticator` first (better performance)
- Logs local auth failure reason at DEBUG level
- Falls back to `ADAuthenticator` on local failure
- Forces `NeedsSync=true` when AD auth succeeds
- Returns local auth error when both fail (generic, prevents info leakage per T-19-03)

### AuthStrategyFactory (`auth_strategy_factory.go`)
- Creates authenticators by mode string: "local", "ad", "hybrid"
- Validates mode parameter (only accepts known values, T-19-02 mitigation)
- `GetDefaultAuthMode()`: reads `sys.auth.default.mode` from `sys_config`, defaults to "local"
- `getADConfigID()`: reads `sys.auth.ad.config_id` from `sys_config`, falls back to first enabled AD config
- Returns error if no AD config available when requesting "ad" or "hybrid" mode

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] ADAuthenticator does not use addomain.LDAPClient directly**
- **Found during:** Task 3
- **Issue:** Plan used `addomain.LDAPClient.GetConn()` which does not exist. LDAPClient.Connect() binds with admin credentials, making it unsuitable for user credential verification.
- **Fix:** ADAuthenticator creates its own LDAP connections directly using `ldap.DialURL`. User bind uses a separate connection from admin search. This is the standard LDAP authentication pattern.
- **Files modified:** internal/core/security/ad_authenticator.go
- **Commit:** 33bd97f

**2. [Rule 3 - Blocking] decryptPassword is unexported in addomain package**
- **Found during:** Task 3
- **Issue:** ADAuthenticator needs to decrypt admin password from `sys_ad_config`, but `decryptPassword` in `addomain/utils.go` was unexported (lowercase).
- **Fix:** Added exported wrapper `DecryptPassword` that calls the internal `decryptPassword`, maintaining backward compatibility.
- **Files modified:** internal/services/addomain/utils.go
- **Commit:** 33bd97f

**3. [Rule 2 - Security] LDAP injection prevention**
- **Found during:** Task 3
- **Issue:** Threat model T-19-01 requires mitigating LDAP injection. Plan did not explicitly escape filter values.
- **Fix:** Used `ldap.EscapeFilter(username)` in the search filter construction to prevent LDAP injection attacks.
- **Files modified:** internal/core/security/ad_authenticator.go
- **Commit:** 33bd97f

**4. [Rule 2 - Security] Generic error messages to prevent info disclosure**
- **Found during:** Task 3, Task 4
- **Issue:** Threat model T-19-03 requires unified error returns to avoid leaking user existence information.
- **Fix:** ADAuthenticator returns `ErrInvalidCredentials` (not `ErrUserNotFound`) on bind failure. HybridAuthenticator returns local auth error when both fail (more generic).
- **Files modified:** internal/core/security/ad_authenticator.go, internal/core/security/hybrid_authenticator.go
- **Commit:** 33bd97f, e6f3c12

## Threat Model Compliance

| Threat ID | Mitigation | Status |
|-----------|-----------|--------|
| T-19-01 | LDAP bind verification for user identity | Implemented: UPN bind + ldap.EscapeFilter |
| T-19-02 | Validate mode parameter | Implemented: switch with default error |
| T-19-03 | Unified error returns | Implemented: ErrInvalidCredentials for all auth failures |
| T-19-04 | LDAP connection management | Partial: connections closed after use, no pool yet |
| T-19-05 | Hybrid fallback still validates | Implemented: AD credentials verified on fallback |

## Known Stubs

None - all authenticators have full implementation logic.

## Threat Flags

None - all new surface (LDAP connections, auth mode parameter) is covered by the plan's threat model.

## Self-Check

All files and commits verified below.

---

**Execution completed:** 2026-05-21
**Duration:** 6 minutes
**Executor:** Claude (GSD Execute Phase - Parallel Worktree)

## Self-Check: PASSED

All 7 files verified present. All 5 commits verified in git log.
