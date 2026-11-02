---
phase: 60-soft-delete-recovery-safety
reviewed: 2026-10-20T08:45:00Z
depth: standard
files_reviewed: 7
files_reviewed_list:
  - internal/api/v1/auth.go
  - internal/api/v1/auth_80_03_test.go
  - internal/core/security/ad_authenticator.go
  - internal/core/security/ad_authenticator_phase60_test.go
  - internal/core/security/authenticator.go
  - internal/services/system/user_sync_service.go
  - internal/services/system/user_sync_service_phase60_test.go
findings:
  critical: 3
  warning: 5
  info: 5
  total: 13
status: issues_found
---

# Phase 60: Code Review Report

**Reviewed:** 2026-10-20T08:45:00Z
**Depth:** standard
**Files Reviewed:** 7
**Status:** issues_found

## Summary

Phase 60's stated deliverables are implemented as specified:

- `checkSoftDeletedRestore` is a single read-only predicate, shared by the login path (`restoreSoftDeletedUser`, before password hashing) and the batch path (`BatchSyncADUsers`, after classification and before hashing / stage-5c writes).
- Refusal is a short-circuit policy result: login returns a sentinel mapped to `SyncReasonSoftDeleteMismatch` and a fixed DN-free Chinese message; batch increments `Skipped`, appends a DN-free `Errors` entry, and never enters `runInBatchedTx`.
- `go test ./internal/services/system/ -run TestPhase60 -count=1` passes; all refusal tests assert the row stays soft-deleted and `ad_dn` is byte-for-byte unchanged.

The defects below are mostly about what the gate *trusts*, plus two independently verified pre-existing bugs in `auth.go` that the submitted file set exposes. The accepted A4 TOCTOU window (classification snapshot vs stage-5c write) and the D-01 "empty stored DN → allow" policy were **not** re-reported as findings.

Cross-checked supporting files: `internal/services/addomain/dn_util.go`, `internal/services/addomain/user_ou_service.go`, `internal/api/v1/system/ad_domain_user_sync_handler.go`, `internal/core/security/local_authenticator.go`, `internal/core/security/hybrid_authenticator.go`, `pkg/middleware/auth.go`.

## Critical Issues

### CR-01: AD bind identity (UPN) and DN-gate identity (sAMAccountName) are resolved by different attributes — SAFE-01 can be bypassed

**File:** `internal/core/security/ad_authenticator.go:82-87`, `internal/core/security/ad_authenticator.go:110-111`, `internal/core/security/ad_authenticator.go:254-263` (gate consumes the result at `internal/services/system/user_sync_service.go:272-297`)

**Issue:** The user is authenticated with the UPN form `username@<config.DomainName>` (line 83), but the DN that Phase 60 compares against `sys_user.ad_dn` comes from a *separate* search keyed on `(sAMAccountName=<username>)` (line 260). AD's `userPrincipalName` and `sAMAccountName` are independent attributes and need not select the same object. Concretely, an AD object whose UPN is `victim@corp.com` but whose `sAMAccountName` is not `victim` can bind successfully; the subsequent search returns the *victim's* AD object, hence the victim's DN. `checkSoftDeletedRestore` then sees `storedAdDn == currentAdDn`, restores the soft-deleted local `victim` row and the handler issues a JWT for that local account — the exact account-takeover shape SAFE-01 exists to prevent. Even absent malice, UPN/sAMAccountName divergence makes the system sync and authorize a different AD object than the one that authenticated (wrong attributes, wrong `ad_dn` written on the next update).

The phase threat model assumes "AD 当前 DN" is authoritative for the authenticated principal; it is not, because it is fetched by a different identifier.

**Fix:** Make the authenticated object and the gated object identical. Resolve the entry first through the already-required admin connection, then bind the user against `entry.DN`:

```go
// admin search first (already supported by the existing account pool)
adminClient, err := a.bindAdminWithFailover(ctx, config)
if err != nil { /* ... */ }
defer adminClient.Close()
adUserInfo, err := a.searchUserInfo(adminClient.Conn(), config, req.Username)
if err != nil { /* ... */ }

userConn, err := a.dialConnection(config, address)
if err != nil { return nil, fmt.Errorf("%w: %v", ErrADConnectionFailed, err) }
defer userConn.Close()
if err := userConn.Bind(adUserInfo.UserDN, req.Password); err != nil {
    return nil, ErrInvalidCredentials
}
```

At minimum, request `sAMAccountName` and `userPrincipalName` in the search attributes and reject when the returned entry's `sAMAccountName` does not equal `req.Username`, and document the UPN/sAMAccountName assumption (it is currently implicit).

### CR-02: Locally disabled users can still log in through the AD and hybrid paths (token issued without status check)

**File:** `internal/api/v1/auth.go:211-264` (AD success path); related `internal/core/security/ad_authenticator.go:127-166`, `internal/core/security/hybrid_authenticator.go:28-48`

**Issue:** `LocalAuthenticator` enforces `UserStatusEnabled` (`local_authenticator.go:38-41`) and `loginLocalDirect` enforces it (`auth.go:281-284`), but the factory path never checks status. For AD, `updateExistingUser` leaves the local `status` untouched, `ADAuthenticator` returns `AuthResult.User` regardless of status, and `login` re-reads the row at line 219 only to build the response. Consequence: an administrator disabling a user (or a user disabled in `sys_user`) can still authenticate with AD credentials and receive a valid access/refresh token. The hybrid authenticator makes this unconditional: local auth fails with `ErrUserDisabled`, then AD auth succeeds and is returned. `AuthResult.User.Status` is computed (`ad_authenticator.go:159`) and then ignored.

**Fix:** Enforce the local status before issuing tokens, in the common login path:

```go
var user models.User
if err := core.DB.GetDB().Where("id = ?", authResult.User.ID).First(&user).Error; err != nil { /* ... */ }
if user.Status != models.UserStatusEnabled {
    response.Error(c, response.ErrForbidden, "用户已被禁用")
    recordLoginLog(c, core, req.Username, user.Nickname, 1, "用户已被禁用")
    return
}
```

Place it before `ClearLoginFailure` / `recordLoginLog` / `GenerateTokenPair`. Consider also re-checking status in the JWT middleware if immediate revocation on disable is required (currently neither JWT middleware nor the handler checks it).

### CR-03: `refreshToken` re-issues a full token pair for disabled users

**File:** `internal/api/v1/auth.go:416-437`

**Issue:** The refresh handler loads the user by ID and immediately generates a new access+refresh pair. It never checks `user.Status`, so a user disabled after login can keep rotating tokens indefinitely (the refresh token lives `RefreshKeyExpire` = 604800s). This defeats the same account-disable control that the local login path enforces.

**Fix:** After the `First(&user)` call at line 418:

```go
if user.Status != models.UserStatusEnabled {
    response.Error(c, response.ErrForbidden, "用户已被禁用")
    return
}
```

## Warnings

### WR-01: `getAuthConfig` reuses the same struct for two `First()` calls — `defaultMode` is never read when the AD-enabled row exists (verified)

**File:** `internal/api/v1/auth.go:473-483`

**Issue:** Both queries write into the same `var config models.Config`. After the first `First(&config)` succeeds, `config.ID` is non-zero; GORM's `First` adds the non-zero primary key as a condition to the second query. Verified from the actual SQL emitted by `TestAuth8003_GetAuthConfig`:

```
SELECT * FROM `sys_config` WHERE config_key = "sys.auth.default.mode"
  AND `sys_config`.`deleted_at` IS NULL
  AND `sys_config`.`id` = "b2954a0e-4d7a-414c-a024-74e6c05ab6ca" LIMIT 1
```

So whenever `sys.auth.ad.enabled` exists, `sys.auth.default.mode` always misses and `defaultMode` stays `"local"`. In an AD-enabled deployment the login page preselects local auth even if the configured default is `ad`/`hybrid`, which makes AD users fail unless they manually switch modes. The bug is already documented and *locked in as expected behavior* by `TestAuth8003_GetAuthConfig` case `AD启用_因QUIRK模式读不到` (`auth_80_03_test.go:448`).

**Fix:** Use independent destinations (and update the test case to expect `"ad"`):

```go
var adCfg models.Config
if err := core.DB.GetDB().Where("config_key = ?", "sys.auth.ad.enabled").First(&adCfg).Error; err == nil {
    adEnabled = adCfg.ConfigValue == "true"
}
var modeCfg models.Config
if err := core.DB.GetDB().Where("config_key = ?", "sys.auth.default.mode").First(&modeCfg).Error; err == nil {
    if modeCfg.ConfigValue == "local" || modeCfg.ConfigValue == "ad" || modeCfg.ConfigValue == "hybrid" {
        defaultMode = modeCfg.ConfigValue
    }
}
```

### WR-02: AD connection errors are wrapped but compared with `==`, so `ErrADConnectionFailed` never matches

**File:** `internal/api/v1/auth.go:187-196`; producer `internal/core/security/ad_authenticator.go:76-79`

**Issue:** `dialConnection` failure returns `fmt.Errorf("%w: %v", ErrADConnectionFailed, err)`, but the handler tests identity with `err == security.ErrADConnectionFailed`. The comparison is always false, so an AD outage falls into the generic branch and is reported to the user as `用户名或密码错误` (HTTP 401 / code 1013) with login-log text `认证失败`, instead of `认证服务暂时不可用`. This actively misleads operators during an AD failure and hides infrastructure problems behind credential errors. (`RecordLoginFailure` runs before classification, so AD outages also increment per-user failure counters and can contribute to lockouts — see the `ad-update-no-such-object-vs-lockout` incident history.)

**Fix:** Use `errors.Is` for all sentinel comparisons in this block (raw sentinels still match):

```go
switch {
case errors.Is(err, security.ErrUserNotFound), errors.Is(err, security.ErrInvalidCredentials):
    ...
case errors.Is(err, security.ErrUserDisabled):
    ...
case errors.Is(err, security.ErrADConfigNotFound), errors.Is(err, security.ErrADConnectionFailed):
    ...
}
```

### WR-03: The batch safety gate compares against the stale `sys_ad_user` cache, not the live AD DN

**File:** `internal/services/system/user_sync_service.go:591-605`; input source `internal/api/v1/system/ad_domain_user_sync_handler.go:44-71`

**Issue:** The login path feeds `checkSoftDeletedRestore` a DN taken from a live LDAP search (`entry.DN`). The batch path feeds it `ADUserInfoForSync.UserDN`, which the handler populates from the `sys_ad_user` table (`getADUsersByDNs`, lines 44-71). If that row is stale — AD object renamed/moved, or a prior sync failed — the gate compares the *old* stored `ad_dn` with the *old* cached DN and passes, restoring a local account to an AD identity that is no longer current; it then writes the stale DN into `sys_user.ad_dn`. Conversely, a legitimate move can produce an unnecessary refusal. This is a different issue from the accepted A4 TOCTOU window (which concerns the snapshot between classification and stage-5c write, not the trustworthiness of the DN source).

**Fix:** For selected DNs, fetch the current DN from LDAP (or run `sys_ad_user` sync for those DNs) before invoking `BatchSyncADUsers`, or fail closed when `sys_ad_user.last_sync_at`/`when_changed` is stale relative to an acceptable threshold. Add a test where `sys_ad_user.user_dn` and the live DN disagree.

### WR-04: `NormalizeDN` splits on commas without honoring RFC4514 escapes — equivalent DNs can be judged different, locking legitimate users out

**File:** `internal/services/addomain/dn_util.go:42-55` (dependency of `checkSoftDeletedRestore`, `user_sync_service.go:277-287`)

**Issue:** `strings.Split(trimmed, ",")` fragments escaped commas (`CN=Doe\, John,...`) and quoted values (`CN="Doe, John",...`). Hex-escaped representations (`\,` vs `\2C`) are not canonicalized either. Two semantically identical DNs can therefore normalize differently and be refused, leaving a legitimate soft-deleted user unable to log in until an administrator manually rewrites `ad_dn`. The code comment deliberately avoids `ldap.ParseDN` for legacy *unescaped* `sys_dept_ou_mapping` values, but `sys_user.ad_dn` is written from AD `entry.DN` (properly escaped), so that caveat does not apply to this gate.

**Fix:** For the `ad_dn` comparison, prefer `ldap.ParseDN` and a canonical serialization, with the current string path as a fallback; or implement an escape-aware split (respect `\` and quoted values) and canonicalize hex escapes. Add predicate tests with `CN=Doe\, John,OU=IT,DC=corp` and `CN=Doe\2C John,OU=IT,DC=corp`.

### WR-05: An unguarded legacy soft-delete restore helper remains in the tree and would bypass the shared gate if re-wired

**File:** `internal/services/addomain/user_ou_service.go:97-158` (`restoreDeletedUserWithADInfo`)

**Issue:** This function clears `deleted_at` by `username` and rewrites `ad_dn` with no identity comparison, directly contradicting the Phase 60 invariant "登录路径与批量路径都必须经由本函数判定". Today it has no production callers (`HandleUserLoginAD` is also uncalled; the fixed login path refuses soft-deleted users at lines 34-49), so this is latent rather than live. It is only scheduled for removal in a later phase. Anyone re-wiring `UserOUService` restores the pre-Phase-60 takeover behavior without any test failing.

**Fix:** Delete `restoreDeletedUserWithADInfo` (and the dead `UserOUService` surface) now, or route it through `checkSoftDeletedRestore` before clearing `deleted_at`. Add a compile-time/`grep` regression or a unit test asserting no `Unscoped().Update("deleted_at", nil)` path exists outside the validator.

## Info

### IN-01: Public debug endpoint `GET /test-sm2` is registered unconditionally

**File:** `internal/api/v1/auth.go:65-66`, `internal/api/v1/auth.go:539-576`

**Issue:** The SM2 self-test endpoint is exposed without authentication in all environments. It does not leak the private key, but it is an unnecessary debug surface that performs a full encrypt/decrypt cycle per request.
**Fix:** Register it only when `config.Server.Mode == "debug"`.

### IN-02: A single refusal emits three WARN records

**File:** `internal/services/system/user_sync_service.go:292-295`, `:453-457`; `internal/core/security/ad_authenticator.go:136-140`

**Issue:** `checkSoftDeletedRestore` logs the audit WARN, then `SyncADUser` logs the same refusal, then `ADAuthenticator.Authenticate` logs it again. One security event produces three near-duplicate lines, and the comment in `checkSoftDeletedRestore` calling it the "唯一一条安全审计 WARN" is misleading.
**Fix:** Keep exactly one audit WARN (the one in `checkSoftDeletedRestore`, which has the richest fields) and downgrade the upper layers to DEBUG or remove them.

### IN-03: Policy refusal is returned as HTTP 500

**File:** `internal/api/v1/auth.go:211-215`

**Issue:** `response.ErrServerError` (code/HTTP 500) is used for `SyncReasonSoftDeleteMismatch`, although the refusal is a deliberate authorization decision, not a server fault. This can trigger 5xx alerting/retry logic and contradicts the phase's goal of distinguishing policy refusal from infrastructure failure. Note: the plan (60-01-PLAN line 138) explicitly said to keep this branch unchanged, so this is recorded as debt rather than a spec deviation.
**Fix:** Return `response.ErrForbidden` for `SyncReasonSoftDeleteMismatch` (403 + fixed message), leaving 500 for `admin_*`, `user_search`, `user_sync`, `no_syncer`.

### IN-04: No `sys_login_log` entry is written for a policy refusal

**File:** `internal/api/v1/auth.go:211-215`

**Issue:** The `authResult.User == nil` branch returns before `recordLoginLog`, so the only audit trace is the service WARN. A refused login attempt (AD bind succeeded, local identity check failed) is exactly the event an operator may need to audit later.
**Fix:** Call `recordLoginLog(c, core, req.Username, nil, 1, "AD 身份不一致，安全拦截")` before returning (message must stay DN-free).

### IN-05: The config-read bug is encoded as expected behavior in the test suite

**File:** `internal/api/v1/auth_80_03_test.go:427-475` (case `AD启用_因QUIRK模式读不到`)

**Issue:** The table deliberately asserts `defaultMode == "local"` when `sys.auth.default.mode = "ad"` and the AD-enabled row exists. Once WR-01 is fixed this test must change; until then it gives false confidence that the handler is correct.
**Fix:** Fix WR-01 and update this case to `wantDefaultMode: "ad"`; keep a separate regression case proving the second query no longer carries residual conditions.

---

_Reviewed: 2026-10-20T08:45:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
