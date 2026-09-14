# Phase 108 Plan 03 Summary: AD Authenticator Tests Restored

## Status: COMPLETE

## Tests Restored

| Test | Line | Status |
|------|------|--------|
| `TestADAuthenticator_Authenticate_Success` | 67 | PASS |
| `TestADAuthenticator_Authenticate_ConfigNotFound` | 108 | PASS |
| `TestADAuthenticator_Authenticate_TableDrivenTests` | 133 | PASS (3 subcases) |
| `TestADAuthenticator_NeedsSyncFlag` | 214 | PASS |

## Tests Left Skipped (as required)

| Test | Line | Reason |
|------|------|--------|
| `TestADAuthenticator_IntegrationTest` | 253 | Requires real AD environment |

## Key Changes

### `internal/core/security/ad_authenticator_test.go`

1. **Added `mockAccountPool`** - Full implementation of `addomain.AccountPool` interface (14 methods) for testing. This was necessary because Phase 38 removed the single-admin fallback in `bindAdminWithFailover`, requiring an account pool to be injected.

2. **Added `setupFakeLDAP`** - Minimal fake LDAP server that accepts connections and returns Bind success responses. Reuses the Phase 78 pattern from `ldap_fake_server_78_07_test.go`.

3. **Added `parsePort`** - Helper to extract port from address string.

4. **Restored `TestADAuthenticator_Authenticate_Success`** - Uses in-memory SQLite with ADConfig + fake LDAP + mock account pool. User bind succeeds (fake LDAP), admin bind fails (password mismatch), flow completes with `NeedsSync=true`.

5. **Restored `TestADAuthenticator_Authenticate_ConfigNotFound`** - Uses in-memory SQLite without ADConfig. Verifies `ErrADConfigNotFound` is returned.

6. **Restored `TestADAuthenticator_Authenticate_TableDrivenTests`** - Three subcases:
   - "配置不存在": Verifies config not found error
   - "空用户名": With fake LDAP, user bind succeeds but admin bind fails → `NeedsSync=true`
   - "空密码": LDAP library client-side rejects empty password → `ErrInvalidCredentials`

7. **Restored `TestADAuthenticator_NeedsSyncFlag`** - Verifies `NeedsSync=true` when no userSyncer is set.

8. **Removed unused `mockADDomainService` and `mockLDAPClient`** - These were no longer needed.

## Test Results

```
=== RUN   TestADAuthenticator_Authenticate_Success        --- PASS
=== RUN   TestADAuthenticator_Authenticate_ConfigNotFound --- PASS
=== RUN   TestADAuthenticator_Name                        --- PASS
=== RUN   TestADAuthenticator_Authenticate_TableDrivenTests
    --- PASS: 配置不存在                                  --- PASS
    --- PASS: 空用户名                                    --- PASS
    --- PASS: 空密码                                      --- PASS
=== RUN   TestADAuthenticator_NeedsSyncFlag              --- PASS
=== RUN   TestADAuthenticator_IntegrationTest             --- SKIP
=== RUN   TestADAuthenticator_Authenticate_DialFailure   --- PASS
=== RUN   TestADAuthenticator_BindAdminNoPool            --- PASS
=== RUN   TestADAuthenticator_Setters                   --- PASS
=== RUN   TestADAuthenticator_GetDefaultRoleID           --- PASS
PASS
```

## Notes

- The fake LDAP server only handles Bind operations. Search operations fail, causing `NeedsSync=true` to be returned even on successful user authentication. This is acceptable for unit testing purposes.
- The `mockAccountPool` implements all 14 methods of the `AccountPool` interface including `InvalidateCache` and `StartHotReload` added in later phases.
- `setupSecurityTestDB` in `integration_test.go` already had `sys_ad_config` table definition, so no migration changes were needed.
