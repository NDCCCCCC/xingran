---
phase: 56-vlan-v1-20-1-0-5-plans-initiated-2026-07-09
plan: 02
subsystem: api
tags: [portwrite, vlan, port-binding, scrapligo, validation, gorm]

# Dependency graph
requires:
  - phase: 56-01
    provides: vendor template `set_access_vlan` + `port_binding` action rendering for 3 vendors (Huawei/H3C/Ruijie) with normalized MAC formats
provides:
  - PortWriteService interface extended with SetAccessVlan + PortBinding methods
  - 4 new sentinel errors (ErrVlanIdOutOfRange / ErrBindOpInvalid / ErrIPAddressInvalid / ErrMACAddressInvalid) for HTTP-400 translation in W3
  - PortResult.Extra map as audit after_value carrier (vlanId / bindOp / ipAddress / macAddress)
  - BatchWriteRequest extended with VLANID/BindOp/IPAddress/MACAddress fields
  - checkPreState vlan_match NoOp + port_binding Pitfall-6 skip
affects: [56-03, 56-04, 56-05]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "sentinel-error → HTTP-400 translation contract (execSinglePort consumer in W3)"
    - "PortResult.Extra map pattern for non-status audit payload"
    - "checkPreState NoOp via CurrentState sentinel string (e.g. vlan_match)"

key-files:
  created: []
  modified:
    - internal/services/portwrite/port_write_service.go
    - internal/services/portwrite/pre_state_check.go
    - internal/services/portwrite/batch_orchestrator.go
    - internal/services/portwrite/port_write_service_test.go
    - internal/api/v1/network/port_write_handler_test.go

key-decisions:
  - "IPv4 regex intentionally allows 0.0.0.0 / 255.255.255.255 (RFC-legal); its job is shell-injection rejection (;/|/space/non-digit/extra-dots), not segment range — device protocol layer self-rejects out-of-range segments"
  - "checkPreState returns nil (not NoOp) for ActionPortBinding — Pitfall 6: binding has no idempotent pre-state, must always SSH execute (3-5s round-trip acceptable to avoid stale binding)"
  - "checkPreState returns NoOp CurrentState='vlan_match' for SetAccessVLAN when port.VLAN == target — uniform across 3 vendors"
  - "5 v1.19 methods (Shutdown/UndoShutdown/Description/Dot1x/MacMove) forward zero tail params (0,'','','') — zero behavior change"
  - "Null MAC (00:00:00:00:00:00) rejected via ErrMACAddressInvalid; empty MAC allowed (optional for IP-only binding)"

patterns-established:
  - "Extra-map audit carrier: write Extra in new action methods, leave empty for v1.19 methods"
  - "writeAndRefresh/writeSinglePort/executeWrite carry 4 tail params (vlanId/bindOp/ipAddr/macAddr) — only ActionPortBinding/ActionSetAccessVLAN consume them"

requirements-completed:
  - VLAN-01
  - VLAN-03
  - VLAN-05
  - VLAN-06
  - BIND-01
  - BIND-05
  - BIND-07
  - INFRA-01
  - TEST-02

# Metrics
duration: ~15min
completed: 2026-07-09
---

# Phase 56-02: PortWriteService VLAN + Port Binding Extension Summary

**Extended v1.19 PortWriteService with SetAccessVlan + PortBinding methods, 4 validator sentinels, Extra-map audit carrier, and checkPreState NoOp/skip logic — 31 new subtests, zero v1.19 regression**

## Performance

- **Tasks:** 5/5 complete (collapsed into 3 atomic commits by executor)
- **Files modified:** 5

## Accomplishments
- SetAccessVlan: VLAN ID 1-4094 validation (ErrVlanIdOutOfRange), DB-cached VLAN pre-state check with vlan_match NoOp
- PortBinding: op/IP/MAC validators (3 sentinels), NormalizeMACAddress for null-MAC reject, add/remove paths
- PortResult.Extra map introduced as the audit after_value carrier (fixes v1.20.1 BLOCKER-1 from plan revision 1)
- Batch orchestrator forwards req.{VLANID,BindOp,IPAddress,MACAddress} to per-port executeWrite (fixes BLOCKER-2 batch path)
- 4 tail params threaded through writeAndRefresh/writeSinglePort/executeWrite; 5 v1.19 methods forward zeros

## Task Commits

Each task was committed atomically:

1. **Task feat: service + sentinels + pre-state sig** - `ed28903c` (feat)
2. **Task test: 6+ validator + service tests (31 subtests)** - `159fad39` (test)
3. **Task fix: mock handler test service stubs** - `d255e8c9` (fix)

> NOTE: SUMMARY.md authored by orchestrator (not a separate commit) — executor agent was interrupted by a transient API quota error (429) immediately after committing task 3 and before writing this file. All 3 code commits are intact and verified.

## Files Created/Modified
- `internal/services/portwrite/port_write_service.go` (+158) — SetAccessVlan + PortBinding methods, 4 sentinels, ipv4Pattern, Extra map, BatchWriteRequest fields, interface extension
- `internal/services/portwrite/pre_state_check.go` (+52/-) — checkPreState signature extended; ActionSetAccessVLAN vlan_match NoOp + ActionPortBinding nil (Pitfall 6)
- `internal/services/portwrite/batch_orchestrator.go` (+6/-) — forwards 4 new req fields to per-port executeWrite
- `internal/services/portwrite/port_write_service_test.go` (+335) — 8 new test funcs, 31 subtests (validation + success + NoOp + BindSkip)
- `internal/api/v1/network/port_write_handler_test.go` (+12) — compensatory mock stubs for 2 new interface methods (handler wiring is W3 responsibility)

## Decisions Made
See key-decisions frontmatter above — primarily the IPv4-regex philosophy (injection-rejection vs range-validation) and Pitfall-6 binding pre-state skip.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] mockPortWriteService in port_write_handler_test.go missing 2 new interface methods**
- **Found during:** Task 3 verification (`go test ./internal/api/v1/network/...`)
- **Issue:** Phase 52 W3 mock service struct didn't implement SetAccessVlan + PortBinding, breaking compilation of the whole network api test package
- **Fix:** Added zero-value stubs returning `&PortResult{Status: succeeded}` to the mock (handler execution-path coverage deferred to W3)
- **Files modified:** internal/api/v1/network/port_write_handler_test.go
- **Verification:** `go test ./internal/api/v1/network/...` compiles + passes
- **Committed in:** d255e8c9

---

**Total deviations:** 1 auto-fixed (Rule 1 - Bug)
**Impact on plan:** Necessary compilation fix. No scope creep.

## Issues Encountered
- Executor agent hit a transient API quota error (429) after committing all 3 tasks but before writing SUMMARY.md. Orchestrator recovered by merging the worktree branch (3 intact commits), verifying build + tests, and authoring this SUMMARY.md centrally.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Service layer complete; ready for W3 handler/router/permission wiring
- 4 sentinel errors are contract-locked; W3 execSinglePort maps them to HTTP 400
- PortResult.Extra map ready for W3 buildAfterValue consumption (after_value audit)
- BLOCKER-1 (after_value) + BLOCKER-2 (batch path) from plan revision 1 resolved

---
*Phase: 56-vlan-v1-20-1*
*Completed: 2026-07-09*
