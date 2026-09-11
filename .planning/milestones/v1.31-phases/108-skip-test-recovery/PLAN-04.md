---
phase: 108
plan: "04"
type: execute
wave: 2
depends_on: []
files_modified:
  - .planning/phases/108-skip-test-recovery/HUMAN-UAT.md
autonomous: true
requirements_addressed: [SKIP-02]
---

<objective>
Create HUMAN-UAT.md documenting the 2 integration tests that require a real Active Directory environment and cannot be automated in the current test infrastructure.
</objective>

<tasks>

## Task 1: Create HUMAN-UAT.md

**File:** `.planning/phases/108-skip-test-recovery/HUMAN-UAT.md`

Document the 2 skipped integration tests that need real AD environment:

```markdown
# Phase 108 HUMAN-UAT: AD Authenticator Integration Tests

## Overview

Two integration tests in `internal/core/security/ad_authenticator_test.go` are skipped because they require a **real Active Directory environment** with:
- A live LDAP/AD server (Windows Server AD or equivalent)
- Valid test credentials with known usernames and passwords
- Network connectivity to the AD server

These tests verify end-to-end AD authentication flows that cannot be mocked at the protocol level.

## Skipped Tests

### Test 1: TestADAuthenticator_IntegrationTest (line 244)

**File:** `internal/core/security/ad_authenticator_test.go`
**Function:** `TestADAuthenticator_IntegrationTest`
**Line:** 244

```go
func TestADAuthenticator_IntegrationTest(t *testing.T) {
    if testing.Short() {
        t.Skip("跳过集成测试（使用 -short 标志）")
    }
    t.Skip("集成测试需要真实AD环境配置")
}
```

**What it tests:**
- Full AD bind with real credentials
- User search after successful bind
- AD user info retrieval and mapping
- End-to-end authentication flow with real LDAP responses

### Test 2: TestADAuthenticator_IntegrationTest (line 253)

**File:** `internal/core/security/ad_authenticator_test.go`
**Function:** `TestADAuthenticator_IntegrationTest` (second instance)
**Line:** 253

```go
func TestADAuthenticator_IntegrationTest(t *testing.T) {
    // ... same Skip ...
}
```

**What it tests:**
- Same as above — appears to be a duplicate test case or alternative scenario

## Prerequisites for Manual Testing

To run these tests manually:

1. **AD Environment Setup:**
   - Windows Server with AD DS role enabled
   - Test organizational unit (OU) created for testing
   - Test user accounts with known passwords

2. **Configuration:**
   - Set environment variables or test configuration:
     - `AD_SERVER_ADDRESS` — AD server hostname/IP
     - `AD_ADMIN_USERNAME` — Bind DN or UPN for admin operations
     - `AD_ADMIN_PASSWORD` — Admin password (consider using a service account)

3. **Test Configuration in Database:**
   - Insert AD config into `sys_ad_config` table:
     ```sql
     INSERT INTO sys_ad_config (id, config_name, server_address, server_port,
         domain_name, base_dn, admin_username, admin_password, status)
     VALUES ('test-integration', 'Integration Test AD',
         '<AD_SERVER>', 389, '<DOMAIN>', '<BASE_DN>',
         '<ADMIN_USER>', '<ADMIN_PASSWORD>', 0);
     ```

4. **Run the tests:**
   ```bash
   # Remove the t.Skip() calls temporarily
   go test -v ./internal/core/security/ -run "TestADAuthenticator_IntegrationTest"

   # Or run with short flag (will also skip):
   go test -v -short ./internal/core/security/ -run "TestADAuthenticator_IntegrationTest"
   ```

## Expected Behavior When Real AD is Available

1. **Bind Success Path:**
   - Test connects to AD server
   - Admin bind succeeds with configured credentials
   - User bind succeeds with test user credentials
   - Search returns user attributes (mail, displayName, department, etc.)
   - `AuthResult` is returned with `AuthSource="ad"` and `NeedsSync=true`

2. **Bind Failure Path:**
   - Wrong password returns `ErrInvalidCredentials`
   - Non-existent user returns `ErrUserNotFound`
   - Disabled user returns appropriate error

## Coverage Limitations

These tests are the only way to verify:
- Real LDAP protocol behavior with actual AD server
- Edge cases in AD response parsing
- Timeout and connection failure handling with real network
- SSL/TLS LDAPS connections (if configured)

Automated tests using fake LDAP server (Phase 78 pattern) cover ~80% of the code paths but cannot cover:
- Real SSL/TLS certificate validation
- Actual AD attribute schema variations
- Real-world network timeout behavior
- AD-specific error codes and messages

## Recommendations

1. **Keep these tests skipped in CI** — they require external AD infrastructure
2. **Run manually before AD-related changes** — when modifying AD authentication logic
3. **Consider AD simulation container** — for more reproducible integration testing (out of scope for Phase 108)
4. **Document in release notes** — when AD authentication behavior changes

## Related Files

- `internal/core/security/ad_authenticator.go` — AD authenticator implementation
- `internal/core/security/ad_authenticator_test.go` — test file with skipped tests
- `internal/services/addomain/ldap_fake_server_78_07_test.go` — Phase 78 fake LDAP server (automated testing)
- `.planning/phases/108-skip-test-recovery/RESEARCH.md` — Phase 108 research findings
```

</tasks>

<success_criteria>
- HUMAN-UAT.md created at `.planning/phases/108-skip-test-recovery/HUMAN-UAT.md`
- Documents both skipped IntegrationTest functions with line numbers
- Includes prerequisites, expected behavior, and manual run instructions
</success_criteria>

<output>
No output file required — documentation only
</output>
