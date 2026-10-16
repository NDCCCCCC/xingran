# Deferred Items — Phase 52 W3 (52-02)

Out-of-scope issues discovered during 52-02 plan verification. **Not fixed** by
this plan — owner should triage separately.

## Pre-existing `tests/integration/login_encryption_test.go` failures (3 tests)

**Discovered during:** Task 3 full `go test ./... -count=1` run
(plan verification command 5).

**Failing tests** (all in `tests/integration/login_encryption_test.go`, last
modified at commit `139ed845` — well before Phase 52):

1. `TestPublicKeyEndpoint` (line 80 + 91)
   - Expected: HTTP 200 + JSON `{code, message, data: {publicKey}}`
   - Actual: HTTP 404 (endpoint `/auth/public-key` not registered in
     `setupMinimalTestServer`)
   - Cause: `setupMinimalTestServer` constructs a minimal gin engine without
     the SM2 public-key endpoint that the encryption test expects.
2. `TestResponseHeaders` (line 222)
   - Expected: Content-Type contains `application/json`
   - Actual: Content-Type = `text/plain`
   - Cause: Same minimal-server setup gap.
3. `TestRequestMethodValidation` (line 235)
   - Expected: GET request → 200
   - Actual: HTTP 404
   - Cause: Same minimal-server setup gap.

**Phase 52 W3 impact:** None. These tests touch `/auth/public-key` + SM2/SM4
encryption + middleware chain. Phase 52 W3 only adds:
- `models.PortWriteAudit` AutoMigrate registration
- `migrations.Migrate202PortWriteAudit` explicit call
- `migrations.GrantNewMenuToRolesHavingParent` helper
- `migration_202_port_write_audit.go` menu seed

None of these affect `/auth/*` routes or the encryption middleware setup.

**Pre-existing baseline:** The integration test file was authored at
commit `139ed845` (login encryption test suite) and the minimal-server
setup has been broken since then. Phase 51 SUMMARY also recorded
pre-existing operations test failures. Not in 52-02 scope per CLAUDE.md
"Scope Constrainment" + dev rule "fix only reported issue, don't touch
other files".

**Recommended owner:** integration test infra hardening pass (separate
plan). Likely needs `setupMinimalTestServer` to register SM2 pubkey
endpoint + middleware chain from `cmd/main.go`.

**Tracking:** Logged here for Phase 52 visibility. Next phase planning
should consider adding an integration-test hygiene task to the backlog.