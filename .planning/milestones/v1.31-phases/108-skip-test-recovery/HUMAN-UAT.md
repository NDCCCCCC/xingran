# Phase 108 HUMAN-UAT: AD Integration Tests

## Background

Phase 108 (SKIP-02) restored all possible tests using embedded infrastructure:
- sqlite :memory: for database-dependent tests (13 tests)
- Fake LDAP server for LDAP-client tests (4 tests)

However, 2 tests explicitly require a **real Active Directory environment** and cannot be restored with embedded infrastructure.

## Tests Requiring Real AD

### Test: `TestADAuthenticator_IntegrationTest` (Line 244)

**File:** `internal/core/security/ad_authenticator_test.go`
**Skip reason:** `"集成测试需要真实AD环境配置"`

**What it tests:**
- Full end-to-end AD authentication flow
- Requires: real AD server, valid bind credentials, test user accounts
- Verifies: Bind, Search, and authentication against real LDAP

**Setup requirements:**
- AD server (Windows Server or Samba AD)
- Test organizational unit (OU) with test users
- Bind DN with sufficient permissions to search and authenticate

---

### Test: `TestADAuthenticator_IntegrationTest` (Line 253)

**File:** `internal/core/security/ad_authenticator_test.go`
**Skip reason:** `"集成测试需要真实AD环境配置"`

**What it tests:**
- Same as above — appears to be a duplicate or variant
- Both tests need the same real AD environment

---

## Manual Test Procedure

When real AD environment is available:

1. Configure AD connection in `configs/config.yaml`:
```yaml
ad:
  server: ldaps://ad.example.com:636
  bind_dn: CN=admin,CN=Users,DC=example,DC=com
  bind_password: <password>
  base_dn: DC=example,DC=com
```

2. Run the tests:
```bash
go test ./internal/core/security/ -run "TestADAuthenticator_IntegrationTest" -v
```

3. Verify:
- Test connects to AD successfully
- Bind authentication works
- User search returns expected results

---

## Sign-Off

| Test | Environment | Status | Date |
|------|-------------|--------|------|
| TestADAuthenticator_IntegrationTest (line 244) | Real AD | Pending | - |
| TestADAuthenticator_IntegrationTest (line 253) | Real AD | Pending | - |
