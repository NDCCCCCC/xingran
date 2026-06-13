---
phase: 32
slug: v1-14-p1-p2
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-13
---

# Phase 32 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go standard `testing` + `github.com/stretchr/testify/assert` |
| **Config file** | None (Go's built-in test discovery) |
| **Quick run command** | `go test -count=1 -run "<TestName>" ./<package>/` |
| **Full suite command** | `go test -count=1 ./...` |
| **Estimated runtime** | ~120 seconds (full suite incl. AD/operations modules) |

---

## Sampling Rate

- **After every task commit:** `go build ./... && go vet ./...`
- **After every plan wave:** `go test -count=1 ./...`
- **Before `/gsd:verify-work`:** Full suite green + coverage threshold met
- **Max feedback latency:** 60 seconds (build + vet + targeted package test)

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 32-01-01 | 01 | 1 | P1-S1 | T-01 (alg confusion) | Reject JWT with `alg=none` or `alg=HS256` confusion | unit | `go test -count=1 -run "TestSM2JWT" ./pkg/crypto/ -v` | ❌ W0 | ⬜ pending |
| 32-01-02 | 01 | 1 | P1-S2 | T-02 (replay) | Reject timestamp ±61s outside window | unit | `go test -count=1 -run "TestRequestEncryption" ./pkg/crypto/ -v` | ❌ W0 | ⬜ pending |
| 32-01-03 | 01 | 1 | P1-S3 | T-03 (memory leak) | CleanupExpiredNonces ticker runs every `maxTimeDiff`s | unit | `go test -count=1 -run "TestShardedNonceStorage" ./pkg/crypto/ -v` | ❌ W0 | ⬜ pending |
| 32-01-04 | 01 | 1 | P1-S4 | T-04 (privilege escalation) | Permission check rejects C-type child menus inheriting parent | unit | `go test -count=1 -run "TestPermission" ./pkg/middleware/ -v` | ❌ W0 | ⬜ pending |
| 32-01-05 | 01 | 1 | P1-S7 | T-05 (malicious upload) | Reject Excel file lacking `PK\x03\x04` magic | unit | `go test -count=1 -run "TestVerifyExcelMagic" ./internal/api/v1/operations/ -v` | ❌ W0 | ⬜ pending |
| 32-02-01 | 02 | 2 | P1-S5 | T-06 (weak KDF) | Default iterations ≥600000; legacy 100k hashes still verify | unit | `go test -count=1 -run "TestPasswordManager" ./internal/core/security/ -v` | ❌ W0 | ⬜ pending |
| 32-02-02 | 02 | 2 | P1-S6 | T-07 (modulo bias) | Generated passwords show no character-distribution bias | unit (statistical) | `go test -count=1 -run "TestGenerateRandomPassword" ./internal/core/security/ -v` | ❌ W0 | ⬜ pending |
| 32-03-01 | 03 | 3 | P1-C1 | T-08 (double-sync) | `singleflight.Group` dedupes 10 concurrent calls to 1 sync | unit (concurrent) | `go test -count=1 -run "TestSyncData" ./internal/services/addomain/ -v` | ❌ W0 | ⬜ pending |
| 32-03-02 | 03 | 3 | P1-C2 | T-09 (mass delete) | handleDeletedGroups rejects empty LDAP or delete ratio >50% | unit | `go test -count=1 -run "TestHandleDeletedGroups" ./internal/services/addomain/ -v` | ❌ W0 | ⬜ pending |
| 32-03-03 | 03 | 3 | P1-C3 | T-10 (partial state) | UpsertMapping atomic: delete+insert in single transaction | unit (mock DB failure) | `go test -count=1 -run "TestUpsertMapping_Atomic" ./internal/services/addomain/ -v` | ❌ W0 | ⬜ pending |
| 32-03-04 | 03 | 3 | P1-C4 | T-11 (connection pool) | Port collector batches 100 inserts per call | unit (SQL mock) | `go test -count=1 -run "TestPortCollector" ./internal/collectors/ -v` | ❌ W0 | ⬜ pending |
| 32-03-05 | 03 | 3 | P1-C5 | T-12 (zombie conn) | WebSocket readPump closes stale connection | unit (mock conn) | `go test -count=1 -run "TestClient_ReadPump" ./internal/websocket/ -v` | ❌ W0 | ⬜ pending |
| 32-03-06 | 03 | 3 | P1-C6 | T-13 (N+1) | validateUniqueness uses single IN query, not per-row | unit (SQL counter) | `go test -count=1 -run "TestValidateUniqueness" ./internal/services/operations/ -v` | ❌ W0 | ⬜ pending |
| 32-04-01 | 04 | 4 | P1-B1 | — | ConfigService.Update invalidates middleware encryption cache | unit | `go test -count=1 -run "TestConfigService" ./internal/services/system/ -v` | ❌ W0 | ⬜ pending |
| 32-04-02 | 04 | 4 | P1-B2 | — | buildDepartmentPaths called once in userService.List | code review | N/A (manual grep) | N/A | ⬜ pending |
| 32-05-01 | 05 | 5 | P2-A1 | — | Core split: CoreInfra + CoreServices preserve all field access | compile | `go build ./... && go vet ./...` | N/A | ⬜ pending |
| 32-05-02 | 05 | 5 | P2-A2 | — | Only one `cache_keys.go` source remains (deduped) | compile | `go vet ./... && find . -name "cache_keys*.go" | wc -l` = 1 | N/A | ⬜ pending |
| 32-05-03 | 05 | 5 | P2-A3 | — | user_service_optimized.go removed; router uses optimized | compile | `go build ./... && ! test -f internal/services/system/user_service_optimized.go` | N/A | ⬜ pending |
| 32-05-04 | 05 | 5 | P2-A5 | — | role_service uses apperrors (no fmt.Errorf for known errors) | unit | `go test -count=1 -run "TestRole" ./internal/services/system/ -v` | ❌ W0 | ⬜ pending |
| 32-06-01 | 06 | 6 | P2-A4 | — | Migration files renumbered: no conflicts in 027/028/029/030/031/036 | manual + runtime | Check filenames + verify auto-migrate runs in order | N/A | ⬜ pending |
| 32-06-02 | 06 | 6 | P2-A7 | — | Subprocess calls use process group + reaper | unit (subprocess test) | `go test -count=1 -run "TestSubprocess" ./internal/agent/server/ -v` | ❌ W0 | ⬜ pending |
| 32-06-03 | 06 | 6 | P2-A8 | — | Excel import transactional: rollback on sub-process failure | unit (rollback test) | `go test -count=1 -run "TestImportData" ./internal/services/operations/ -v` | ❌ W0 | ⬜ pending |
| 32-07-01 | 07 | 7 | P2-A6 | — | LDAP mock tests: Connect/Bind/Search/group_sync/user_ou_service | unit (mock) | `go test -count=1 -run "TestLDAP\|TestGroupSync\|TestUserOU" ./internal/services/addomain/ -v` | ❌ W0 | ⬜ pending |
| 32-07-02 | 07 | 7 | P2-A6 | — | Verify stripBaseDN_test.go and dept_ou_mapper_test.go are complete | unit | `go test -count=1 -run "TestStripBaseDN\|TestDeptOUMapper" ./internal/services/addomain/ -v` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

### Test Files to Create (missing per RESEARCH.md audit)

- [ ] `pkg/crypto/sm2_jwt_alg_test.go` — covers P1-S1 (alg=none, alg=HS256 confusion)
- [ ] `pkg/crypto/request_encryption_replay_test.go` — covers P1-S2 (±61s boundary)
- [ ] `pkg/crypto/nonce_storage_cleanup_test.go` — covers P1-S3 (ticker fires)
- [ ] `pkg/middleware/permission_child_inherit_test.go` — covers P1-S4 (C-type rejection)
- [ ] `internal/core/security/password_iterations_test.go` — covers P1-S5 (600k + 100k compat)
- [ ] `internal/core/security/random_password_bias_test.go` — covers P1-S6 (chi-square on charset)
- [ ] `internal/api/v1/operations/excel_handler_magic_test.go` — covers P1-S7 (PK\x03\x04 check)
- [ ] `internal/services/addomain/sync_singleflight_test.go` — covers P1-C1 (10 goroutines, 1 sync)
- [ ] `internal/services/addomain/group_sync_deleted_threshold_test.go` — covers P1-C2
- [ ] `internal/services/addomain/dept_ou_mapper_atomic_test.go` — covers P1-C3
- [ ] `internal/collectors/port_collector_batch_test.go` — covers P1-C4 (mock SQL, count INSERTs)
- [ ] `internal/websocket/notice_hub_readpump_test.go` — covers P1-C5
- [ ] `internal/services/operations/excel_service_uniqueness_test.go` — covers P1-C6
- [ ] `internal/services/system/config_service_cache_test.go` — covers P1-B1
- [ ] `internal/services/system/role_service_apperrors_test.go` — covers P2-A5
- [ ] `internal/agent/server/subprocess_reaper_test.go` — covers P2-A7
- [ ] `internal/services/operations/excel_service_transaction_test.go` — covers P2-A8
- [ ] `internal/services/addomain/ldap_client_mock_test.go` — covers P2-A6 (Connect/Bind/Search)
- [ ] `internal/services/addomain/group_sync_test.go` — covers P2-A6 (expand existing)
- [ ] `internal/services/addomain/user_ou_service_test.go` — covers P2-A6 (expand existing)

**Wave 0 complete when:** All 20 files exist and `go test -count=1 ./...` runs (with at least compile success).

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Migration files renumber | P2-A4 | Runtime ordering depends on filename vs DB state | Run `gorm-auto-migrate` twice (fresh + warm); verify applied versions match expected order |
| `buildDepartmentPaths` called once | P1-B2 | Already fixed; verification = no regression | `grep -n "buildDepartmentPaths" internal/services/system/user_service.go` returns single call site |
| Role service error format consistency | P2-A5 | Review subjective | `grep -rn "fmt.Errorf" internal/services/system/role_service.go` returns only wrapped external errors (DB, HTTP); known domain errors use `apperrors.New*` |

*Otherwise: All phase behaviors have automated verification.*

---

## Validation Sign-Off

- [ ] All tasks have automated verify or Wave 0 dependency
- [ ] Sampling continuity: every P1-S/P1-C/P1-B/P2-A item has at least one `go test` command
- [ ] Wave 0 covers all 20 MISSING test files
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s (build+vet+single-package test)
- [ ] Coverage threshold: `go test -coverprofile=coverage.out ./internal/services/addomain/ ./internal/services/system/ ./internal/services/operations/ ./internal/collectors/ ./internal/websocket/ ./pkg/crypto/ ./internal/core/security/ ./internal/agent/server/ && go tool cover -func=coverage.out` shows ≥70% for these packages

**Approval:** pending (will become approved after `/gsd:verify-work` reports green)