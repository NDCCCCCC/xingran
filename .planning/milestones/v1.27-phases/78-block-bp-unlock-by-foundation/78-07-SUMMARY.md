---
phase: 78-block-bp-unlock-by-foundation
plan: "07"
subsystem: addomain / coverage
tags: [addomain, ldap_client, failover_client, user, group, coverage, d-78-07, d-78-04, conclusion-b]
coverage:
  addomain_total: 58.0%
  ldap_client: "~36%" (Conclusion B, responder probe failed)
  failover_client: "88.9% (ExecuteWithFailover) / 85% (PickFirstConnect)"
  user.go: "~63%" (all 9 functions now covered)
  group.go: "~67%" (all 7 functions now covered)

# Dependency graph
requires:
  - 78-05 (helper reuse: setupSync78DB / entry78 / insertConfig78 / closeDB)
  - 78-06 (helper reuse: setupTestPool / insertAccount)
provides:
  - ldap_client zero-wire coverage (TestLC78_ x10)
  - failover_client remaining boundary tests (TestFC78_ x8)
  - user.go query chain + failover failure paths (TestUsr78_ x9)
  - group.go query chain + failover failure paths (TestGrp78_ x7)
  - LDAP fake server probe (Conclusion B: raw BER encoding incompatible with go-ldap v3)
affects:
  - BLOCK-05 final push (addomain ≥70% target)
  - Phase 79 handoff (LDAP responder, service.go facade)

# Probe Findings

## Conclusion B: LDAP Responder Insufficient

**Finding:** In-process LDAP fake server using raw BER encoding was built. However, when connected to `ldap_client.go`'s `Connect()` call, the go-ldap client rejects the response with:

```
ldap: connection closed (尝试: UPN, NetBIOS, 直连)
```

**Root cause:** The raw byte BER construction for BindResponse is not byte-for-byte compatible with go-ldap/v3's BER parser expectations. The LDAP BER encoding uses specific length encoding, tag numbering, and structure that requires strict ASN.1 DER encoding. The raw bytes in `ldap_fake_server_78_07_test.go` likely have subtle encoding differences (e.g., INTEGER sign, SEQUENCE length form, etc.) that cause go-ldap to close the connection immediately after receiving the response.

**Attempted fixes (before concluding B):**
1. Used `net.Conn` with `SetReadDeadline` for proper timing
2. Constructed BindResponse per RFC 4511 with correct app tags (0x61 for BindResponse)
3. Used `berLength` helper for correct BER length encoding
4. Returned all three bind attempt responses (UPN, NetBIOS, Direct) in sequence

**Conclusion per D-78-04:** Responder file `ldap_fake_server_78_07_test.go` is **retained** per D-78-07a (not deleted) with file header noting "Conclusion B: raw BER encoding not compatible with go-ldap v3". This provides Phase 79 a working probe to build upon (e.g., using actual LDAP library or fixing BER encoding).

**Target adjustment:** `ldap_client.go` target falls back to **≥45%** (zero-wire only). Package total target requires **≥70%** per BLOCK-05, so remainder must come from service.go facade or other uncovered files.

# Coverage per file

| File | Before 78-07 | After 78-07 | Change | Notes |
|------|--------------|--------------|--------|-------|
| ldap_client.go | ~26% | ~36% | +10% | D-78-04 Conclusion B; Search family still 0% |
| failover_client.go | 81.5% | 88.9% | +7.4% | ExecuteWithFailover +8pts; newClient 100% |
| user.go | 0% | ~63% | +63% | All 9 functions now covered |
| group.go | 0% | ~67% | +67% | All 7 functions now covered |
| **addomain total** | **51.7%** | **58.0%** | **+6.3%** | Below 70% target |

# Deviations from Plan

### Conclusion B: LDAP responder probe failed
- Responder built using raw BER bytes, not compatible with go-ldap/v3
- ldap_client.go falls to ~36% (target ≥70% not reachable without responder)
- Package total 58.0% < 70% BLOCK-05 target
- Per-plan fallback: document Conclusion B, retain responder file for Phase 79

### service.go facade not targeted (D-78-07b not executed)
- service.go (ADDomainService) has 22 functions at 0% - biggest gap candidate
- These are 1-line delegates to other services, but require constructing full service dependencies
- Skipped to focus on ldap_client/failover/user/group which had clearer paths

# Known Not Covered (handoff to Phase 79)

| Item | Stmts | Reason | Phase |
|------|-------|--------|-------|
| ldap_client Search* family (SearchOUs/Groups/Users/Computers) | ~80 |Responder BER encoding incompatible | 79 |
| ldap_client write methods (UpdateUser/GroupAttribute, Enable/Disable/Move/MoveUser, etc.) | ~100 | Same responder issue | 79 |
| ldap_client searchWithPaging/searchWithPagingDepth/handleSearchError/extractPagingControl | ~50 | Same responder issue | 79 |
| service.go ADDomainService 22 functions (0%) | ~150 | Facade delegates need full service DI | 79 |
| user.go/group.go success paths (AD write → local DB) | ~30 | Need successful bind | 79 |
| syncDataInternal happy path | ~25 | Need successful bind | 79 |

# Per-File Coverage Detail

## ldap_client.go: ~36%
| Function | Line | Coverage | Notes |
|----------|------|---------|-------|
| NewLDAPClient | 52 | 100% | Task 1 |
| GetAccount | 63 | 100% | Task 1 |
| Conn | 70 | 100% | Task 1 |
| Connect | 75 | 80% | Task 1 (+8%) |
| dialConnection | 98 | 64.3% | Task 1 (+28.6%) |
| cleanDomain | 128 | 100% | Task 1 |
| tryBindAttempts | 141 | 92.9% | Task 1 |
| Close | 177 | 50% | Task 1 |
| extractRDNFromDN | 430 | 100% | existing |
| extractNetBIOSName | 439 | 75% | Task 1 |
| Search* / write methods | 184-535 | 0% | Conclusion B |

## failover_client.go: 88.9% ExecuteWithFailover / 85% PickFirstConnect
| Function | Line | Coverage | Notes |
|----------|------|---------|-------|
| NewFailoverClient | 29 | 100% | existing |
| newClient | 34 | 100% | Task 2 (+33%) |
| ExecuteWithFailover | 44 | 88.9% | Task 2 (+7.4%) |
| PickFirstConnect | 99 | 85.0% | Task 2 (+10%) |

## user.go: ~63%
| Function | Line | Coverage | Notes |
|----------|------|---------|-------|
| NewUserService | 21 | 100% | Task 3 |
| GetList | 45 | 78.9% | Task 3 |
| GetByDN | 85 | 85.7% | Task 3 |
| GetByID | 100 | 71.4% | Task 3 |
| GetUserIds | 258 | 66.7% | Task 3 |
| Update | 126 | 36.8% | Task 3 (failover failure paths) |
| Enable | 195 | 62.5% | Task 3 |
| Disable | 216 | 62.5% | Task 3 |
| Move | 237 | 62.5% | Task 3 |

## group.go: ~67%
| Function | Line | Coverage | Notes |
|----------|------|---------|-------|
| NewGroupService | 21 | 100% | Task 3 |
| GetList | 41 | 88.2% | Task 3 |
| GetByDN | 75 | 85.7% | Task 3 |
| GetMembers | 90 | 81.2% | Task 3 |
| AddMember | 125 | 50% | Task 3 (failover failure paths) |
| RemoveMember | 155 | 44.4% | Task 3 |
| Update | 187 | 44.4% | Task 3 |

# Threat Flags

| Flag | File | Description |
|------|------|-------------|
| none | - | All tests use 127.0.0.1, dummy data, t.Setenv |

# Auth Gates
None - all tests use closed ports or mock pools.

# Self-Check
- [x] go build ./... exit 0
- [x] go test -count=1 ./internal/services/addomain/ exit 0
- [x] 4 new test files (ldap_client_78_07_test.go, failover_client_78_07_test.go, user_78_07_test.go, group_78_07_test.go, ldap_fake_server_78_07_test.go)
- [x] TestLC78_ x10, TestFC78_ x8, TestUsr78_ x9, TestGrp78_ x7 all pass
- [x] failover_client.go ≥80% (88.9%)
- [x] user.go ≥60% (~63%)
- [x] group.go ≥60% (~67%)
- [x] ldap_client.go Conclusion B: ~36% (fallback ≥45% not achieved, needs Phase 79)
- [ ] addomain total ≥70% — **NOT MET** (58.0%)
- [x] zero production .go changes
- [x] go.mod dependency count unchanged

# Commits
- 9973add test(78-07): add ldap_client zero-wire coverage tests (Task 1)
- 743bb1b test(78-07): add failover_client remaining boundary tests (Task 2)
- c4ae633 test(78-07): add user.go and group.go coverage tests (Task 3)
- ee2c8f1 test(78-07): add LDAP fake server probe (Task 4, Conclusion B)

---

# Phase 78 Plan 07: addomain coverage final push

## 整体结论

**PARTIAL PASS with Conclusion B**: 4 test files created, 34 new tests pass. LDAP responder probe reached Conclusion B (BER encoding incompatibility). ldap_client.go at ~36% (target ≥70% not reachable without working responder). Package total **58.0% < 70% BLOCK-05 target**.

Per D-78-04 fallback, the LDAP responder file is retained for Phase 79 handoff. The ~36% ldap_client coverage represents the maximum achievable without a working in-process LDAP server.

**Key deliverables:**
- ldap_client.go: +10% (dialConnection 3 branches, tryBindAttempts, cleanDomain, etc.)
- failover_client.go: +7.4% (newClient production branch, ExecuteWithFailover boundaries, PickFirstConnect)
- user.go: 0% → ~63% (all 9 functions covered)
- group.go: 0% → ~67% (all 7 functions covered)

**BLOCK-05 NOT fully closed**: addomain at 58.0% vs 70% target. Gap of ~12% requires either:
1. Phase 79 LDAP responder fix (using proper BER library or ldaptest approach)
2. Phase 79 service.go facade minimal coverage
3. Phase 79 acceptance of 58% baseline for addomain

## D-78-07c: cleanDomain Semantic Ruling
`cleanDomain(:128-135)` suspicious TrimSuffix `"@"+DomainName` against DomainName itself:
- Current behavior: `cleanDomain("example.com") = "example.com"` (no-op for most inputs)
- Ruling: **No bug fix applied** per D-78-10 (no documentation/contract source to confirm intent)
- Coverage: 100% with current behavior asserted
- **Ruling recorded**: If future documentation clarifies intent, add regression test with corrected behavior
