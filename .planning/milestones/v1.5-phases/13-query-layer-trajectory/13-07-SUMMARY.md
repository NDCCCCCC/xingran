---
phase: 13-query-layer-trajectory
plan: 07
subsystem: backend-contract
tags: [gap-closure, contract-fix, cr-03, w4-vendor, camelCase]
dependency_graph:
  requires:
    - phase: 13
      plan: 01
      reason: TrajectoryNode struct definition and aggregateTrajectory foundation
    - phase: 13
      plan: 03
      reason: GetVendor handler and GetVendorResponse definition
  provides:
    - "TrajectoryNode.MACAddress json:\"mac\" field for tooltip display"
    - "GetVendorResponse camelCase vendorName field (project-wide convention)"
    - "TestAggregateTrajectory_MACAddressPropagation coverage"
  affects:
    - internal/services/mac_history_query_service.go
    - internal/api/v1/network/mac_history_handler.go
    - internal/services/mac_history_query_service_test.go
tech-stack:
  added: []
  patterns:
    - "Wire-format tag rename only (vendor_name → vendorName) — no logic change"
    - "Per-node MAC propagation from rawEvent (invariant: WHERE mac_address=? filter)"
key-files:
  created: []
  modified:
    - internal/services/mac_history_query_service.go
    - internal/api/v1/network/mac_history_handler.go
    - internal/services/mac_history_query_service_test.go
decisions:
  - "MACAddress JSON tag uses 'mac' (single word) to disambiguate from MACTrajectoryResult.macAddress top-level field — top-level is query input echo, per-node is trajectory payload"
  - "Same-location merge branch keeps MACAddress unchanged (WHERE mac_address=? SQL filter guarantees invariant across same-MAC events)"
  - "GetVendorResponse field renamed to camelCase to align with macAddress/deviceName/vlanId convention"
metrics:
  duration_seconds: 667
  completed_date: 2026-06-26
  tasks_completed: 2
  files_modified: 3
  commits: 2
---

# Phase 13 Plan 07: Backend Contract Defects (CR-03 + W4) Summary

**One-liner:** Add per-node MAC propagation to TrajectoryNode (CR-03) and rename `GetVendorResponse.VendorName` to camelCase `vendorName` (W4).

## Objective Recap

Closed two backend response-contract gaps surfaced by `13-VERIFICATION.md`:

| Gap | Severity | Description |
|-----|----------|-------------|
| CR-03 | BLOCKER | TrajectoryNode lacked `MACAddress` field — front-end tooltip could only show top-level `MACTrajectoryResult.MACAddress`, not per-node |
| W4 | partial | `GetVendorResponse.VendorName` used `vendor_name` (snake_case), inconsistent with project's camelCase convention (`macAddress`/`deviceName`/`vlanId`) |

## Tasks Executed

### Task 1 — TrajectoryNode.MACAddress field + aggregateTrajectory population + test coverage

**Commit:** `e3871e25` — `fix(13-07): add TrajectoryNode.MACAddress field + populate in aggregateTrajectory`

Changes:
- `internal/services/mac_history_query_service.go`:
  - Added `MACAddress string \`json:"mac"\`` field to `TrajectoryNode` struct (line 91, after `VLANID`).
  - Populated `MACAddress: evt.MACAddress` at both `current = &TrajectoryNode{...}` initialization sites (lines 935 and 957).
  - Same-location merge branch unchanged — `WHERE mac_address = ?` SQL filter guarantees MAC invariant across merged events.
- `internal/services/mac_history_query_service_test.go`:
  - Extended `TestAggregateTrajectory` to assert `assert.Equal(t, "AABBCCDDEEFF", firstNode.MACAddress)` and `secondNode.MACAddress`.
  - Added `TestAggregateTrajectory_MACAddressPropagation` subtest with 3 rawEvents (2 same-location merged + 1 different-location), asserting both nodes' `MACAddress` is non-empty and equals `expectedMAC`.

**Verification:**
- `go build ./...` → exit 0
- `go test -v -run TestAggregateTrajectory ./internal/services/` → 2/2 PASS
- `grep -n 'json:"mac"' internal/services/mac_history_query_service.go` → 1 hit at line 91
- `grep -n "MACAddress: evt.MACAddress" internal/services/mac_history_query_service.go` → 2 hits (lines 935, 957)

### Task 2 — GetVendorResponse camelCase rename (vendor_name → vendorName)

**Commit:** `bf057ca6` — `refactor(13-07): rename GetVendorResponse.VendorName json tag to vendorName (camelCase)`

Changes:
- `internal/api/v1/network/mac_history_handler.go` line 135:
  - Before: `VendorName string \`json:"vendor_name"\``
  - After:  `VendorName string \`json:"vendorName"\``
- Handler implementation, route registration, Swagger annotations all unchanged — wire-format-only rename.
- GET→POST route discussion in `13-VERIFICATION.md` (line 144) is **out of scope** for this plan — VERIFICATION flagged it as WARNING, not BLOCKER; existing `POST /network/history/vendor` + `ShouldBindJSON` is semantically valid for backend service and 13-08 plan covers front-end integration.

**Verification:**
- `go build ./...` → exit 0
- `grep -n 'json:"vendorName"' internal/api/v1/network/mac_history_handler.go` → 1 hit at line 135
- `grep -c 'json:"vendor_name"' internal/api/v1/network/mac_history_handler.go` → 0 (fully removed)

## Deviations from Plan

None — plan executed exactly as written.

**Minor amendment:** Initial Task 2 commit message was mis-drafted (`test(13-07): ...`) since it contained only handler changes (not tests). Amended in-place to `refactor(13-07):` via `git commit --amend` — same tree, same hash-class intent, semantically correct.

## Test Results

Targeted MAC history tests:
```
=== RUN   TestValidateMACAddress           --- PASS (10/10 subtests)
=== RUN   TestExtractOUIPrefix             --- PASS (6/6 subtests)
=== RUN   TestAggregateTrajectory          --- PASS
=== RUN   TestAggregateTrajectory_MACAddressPropagation --- PASS (NEW)
=== RUN   TestSameLocation                 --- PASS (4/4 subtests)
=== RUN   TestGetVendor                    --- PASS (6/6 subtests)
PASS
ok  github.com/xingran-next/xingran-go-backend/internal/services  1.901s
```

**Note on full services suite:** Running `go test ./internal/services/` end-to-end timed out at 181s due to a pre-existing goroutine-leak in `rate_limiter_test.go:262` (`TestRateLimiter_Cleanup.func1`). This is unrelated to MAC history changes — file was not modified and the leak signature (`time.go:338 +0x167` in a tRunner goroutine) indicates a missing `t.Cleanup` or goroutine termination in the rate-limiter test. Out of scope per executor Rule 3 SCOPE BOUNDARY; logged for future plan.

## Acceptance Criteria Status

| Criterion | Status |
|-----------|--------|
| TrajectoryNode.MACAddress field with `json:"mac"` tag | ✓ DONE (line 91) |
| aggregateTrajectory populates MACAddress at both initialization sites | ✓ DONE (lines 935, 957) |
| TestAggregateTrajectory asserts node.MACAddress on at least one node | ✓ DONE (2 nodes) |
| `go build ./...` exit 0 | ✓ DONE |
| `go test -v -run TestAggregateTrajectory ./internal/services/` PASS | ✓ DONE (2/2) |
| No regression: 8 pre-existing TrajectoryNode JSON fields preserved | ✓ DONE (DeviceID/DeviceName/Interface/VLANID/EventType/StartTime/EndTime/Duration + new MACAddress) |
| GetVendorResponse.VendorName JSON tag = "vendorName" | ✓ DONE (line 135) |
| `grep vendor_name` = 0 hits | ✓ DONE |
| 2 atomic commits on `main` | ✓ DONE (`e3871e25`, `bf057ca6`) |
| SUMMARY.md created | ✓ DONE |

## Threat Model Disposition

| Threat ID | Mitigation Outcome |
|-----------|-------------------|
| T-13W7-01 (Tampering — MACAddress drift) | Mitigated: `TestAggregateTrajectory_MACAddressPropagation` locks `assert.Equal(expectedMAC, node.MACAddress)` across both same-location-merged and different-location nodes |
| T-13W7-02 (Repudiation — vendor_name stale code) | Mitigated: grep-zero confirmation in summary; front-end plan 13-08 will consume `vendorName` |
| T-13W7-SC (Slopsquat — npm install) | N/A: no new dependencies, only struct field additions and JSON tag rename |

## Self-Check

- `go build ./...` → exit 0 ✓
- `go test -v -run TestAggregateTrajectory ./internal/services/` → 2/2 PASS ✓
- `git log --oneline | grep "13-07"` → both commits present ✓
- SUMMARY.md exists at `.planning/phases/13-query-layer-trajectory/13-07-SUMMARY.md` ✓
- No modifications to `.planning/STATE.md` or `.planning/ROADMAP.md` (orchestrator-owned) ✓

## Files Modified

1. `internal/services/mac_history_query_service.go` — TrajectoryNode struct + aggregateTrajectory (2 init sites)
2. `internal/api/v1/network/mac_history_handler.go` — GetVendorResponse JSON tag rename
3. `internal/services/mac_history_query_service_test.go` — extended assertions + new subtest

## Commits

- `e3871e25` — `fix(13-07): add TrajectoryNode.MACAddress field + populate in aggregateTrajectory`
- `bf057ca6` — `refactor(13-07): rename GetVendorResponse.VendorName json tag to vendorName (camelCase)`
