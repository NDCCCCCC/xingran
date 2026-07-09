---
phase: 56-vlan-v1-20-1-0-5-plans-initiated-2026-07-09
plan: 03
subsystem: api
tags: [portwrite, vlan, port-binding, handler, router, permission, operlog]

# Dependency graph
requires:
  - phase: 56-01
    provides: vendor template ActionSetAccessVLAN + ActionPortBinding constants + per-vendor command renderers
  - phase: 56-02
    provides: PortWriteService SetAccessVlan + PortBinding methods + 4 sentinel errors + PortResult.Extra map
provides:
  - 2 new HTTP endpoints POST /network/ports/write/set-access-vlan + POST /network/ports/write/port-binding
  - 2 new handler methods (SetAccessVlan + PortBinding) wired via execSinglePort DRY
  - 4 new sentinel→HTTP-400 translations in execSinglePort
  - 2 new route-permission registry rows in pkg/permission/config.go reusing NetworkPortWrite
  - buildAfterValue signature extension: now reads *PortResult.Extra to populate after_value JSON for v1.20.1 actions
affects: [56-04, 56-05]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "bind-outside-DRY: callers bind their own request struct then pass portID+description into execSinglePort (fixes v1.19 double-BindJSON EOF trap exposed by v1.20.1 pre-binding)"
    - "Extra-map audit carrier consumed at HTTP layer via buildAfterValue(action, *PortResult) — after_value JSON now populated from service-injected Extra"
    - "OperType branch at handler pre-DRY: op=add → OperTypeCreate, op=remove → OperTypeDelete (one handler, two operTypes)"
    - "Permission registry row = discoverability only (group-level middleware is the RBAC enforcement point)"

key-files:
  created:
    - pkg/permission/config_v1201_test.go
  modified:
    - internal/api/v1/network/port_write_handler.go
    - internal/api/v1/network/port_write_handler_test.go
    - internal/api/v1/network/port_write_router.go
    - internal/api/v1/network/port_write_router_test.go
    - pkg/permission/config.go

key-decisions:
  - "execSinglePort refactored: signature gains portID + description params; binding moved to all 7 callers (5 v1.19 + 2 v1.20.1). Necessary because v1.20.1 handlers pre-bind their own struct per PATTERNS.md Option A, and the v1.19 internal ShouldBindJSON(&PortWriteRequest{}) would EOF on the already-consumed body. The v1.19 5 handlers were updated to bind PortWriteRequest then forward portID/description into execSinglePort — zero behavior change for v1.19."
  - "buildAfterValue(action, pr *PortResult) signature extended: pr may be nil/Extra-nil for v1.19 actions (backward compat); for ActionSetAccessVLAN/ActionPortBinding it reads pr.Extra[\"vlanId\"/\"ipAddress\"/\"macAddress\"/\"bindOp\"] populated by W2 service. Plan's 'AVOID: changing buildAfterValue signature' line contradicted its own step-4 instructions; followed step-4 (the substantive instruction) since PortResult.Extra was already implemented in W2."
  - "operlog.Record count is 2 statement invocations, not 7 as plan's verification step claimed. The v1.19 DRY pattern consolidates all 5 single-port handlers + 2 v1.20.1 handlers through 1 operlog.Record call in execSinglePort (chokepoint) + 1 in BatchWrite = 2 total. The '7 count' in the plan assumed each handler had its own operlog call. Plan's intent (operlog coverage for 2 new handlers) is satisfied — they flow through execSinglePort."
  - "OperType mapping per design.md §6 locked: set_access_vlan=Update(2), port_binding add=Create(1), port_binding remove=Delete(3). Branching happens at PortBinding handler BEFORE execSinglePort (one handler, two operTypes)."
  - "IP/MAC are NOT in operlog's 11-keyword sensitive list — plain operlog.Record(WithOperParam) used, NOT RecordWithBody. Path C lock preserved (no WithOperID / WithJsonResult)."

patterns-established:
  - "Handler pre-binds action-specific struct (SetAccessVlanRequest / PortBindingRequest), branches operType if needed, then calls execSinglePort(portID, desc, serviceCallClosure) — binding OUTSIDE the DRY"
  - "Mock service in handler tests: add result/err fields per new method to allow per-test customization (W2 zero-value stub extended in W3)"

requirements-completed:
  - VLAN-01
  - BIND-01
  - BIND-02
  - INFRA-02
  - INFRA-03
  - INFRA-04

# Metrics
duration: ~18min
completed: 2026-07-09
---

# Phase 56-03: Port-Write Handler + Router + Permission Wiring Summary

**Wired 2 new kebab HTTP endpoints (set-access-vlan + port-binding) through v1.19's execSinglePort DRY, refactored execSinglePort to receive bound portID/description (fixes double-BindJSON EOF trap), extended buildAfterValue to read PortResult.Extra for v1.20.1 audit after_value, and added 12 new subtests + 3 permission tests — zero v1.19 regression.**

## Performance

- **Tasks:** 3/3 complete (Task 3 is acceptance gate only — no commit)
- **Files modified:** 5 (+1 new test file)
- **Test functions added:** 8 (5 handler + 1 router + 2 permission test files with 5 subtests)

## Accomplishments
- **2 new HTTP endpoints registered**: `POST /network/ports/write/set-access-vlan` and `POST /network/ports/write/port-binding` mounted on the v1.19 `write` group — both inherit `RequirePermissions([network:port:write], core)` automatically (no new perm constant).
- **2 new handler methods**: `SetAccessVlan` (OperType=Update=2) and `PortBinding` (OperType branches: add→Create=1, remove→Delete=3). Both bind their action-specific struct OUTSIDE execSinglePort then forward portID + description to the DRY helper.
- **4 new sentinel→HTTP-400 translations** wired in execSinglePort: `ErrVlanIdOutOfRange` / `ErrBindOpInvalid` / `ErrIPAddressInvalid` / `ErrMACAddressInvalid`. Generic Chinese messages, no internal detail leakage.
- **execSinglePort refactored**: signature now receives `portID` + `description` as explicit params (was binding internally). All 7 callers updated (5 v1.19 + 2 v1.20.1). This fixes the double-BindJSON EOF trap that the v1.20.1 pre-binding pattern exposed.
- **buildAfterValue signature extended** to `buildAfterValue(action portcollection.PortAction, pr *portwrite.PortResult)`. Reads `pr.Extra["vlanId"]` for set_access_vlan and `pr.Extra["ipAddress"/"macAddress"/"bindOp"]` for port_binding. v1.19 actions unchanged (pr nil-safe).
- **2 new route-permission registry rows** in `pkg/permission/config.go` reuse the existing `NetworkPortWrite` constant (discoverability for `GetRoutePermissions()`; enforcement is at group-level middleware).
- **12 new handler subtests** (5 funcs): binding rejection (vlanId range + op oneof), service sentinel 400 (4 sentinels), success path with Extra-populated after_value, OperType branching (add/remove/invalid), source-grep assertions.
- **3 new permission tests** (1 new file): v1.20.1 rows present + use NetworkPortWrite; v1.19 6 rows intact; `GetPermissionByPath` lookups return NetworkPortWrite for both new routes.
- **1 new router test**: explicit v1.20.1 endpoint registration assertion.

## Task Commits

Each task was committed atomically:

1. **Task feat: handlers + sentinels + execSinglePort refactor + tests** — `e40e4362`
2. **Task feat: kebab routes + permission registry rows + tests** — `6887e5c1`
3. **Task 3: acceptance gate only (no file changes — no commit)**

## Files Created/Modified
- `internal/api/v1/network/port_write_handler.go` (+194/-35) — 2 new handlers + 2 request structs + 4 sentinel translations + execSinglePort refactor + buildAfterValue signature extension
- `internal/api/v1/network/port_write_handler_test.go` (+213/-12) — 5 new test funcs (12 subtests) + mock service extended with setAccessVlanOut/Err + portBindingOut/Err fields
- `internal/api/v1/network/port_write_router.go` (+15/-9) — 2 new write.POST lines + comment refreshed to 8-endpoint inventory
- `internal/api/v1/network/port_write_router_test.go` (+14) — `TestSetupPortWriteRouter_RegistersV1201KebabEndpoints`
- `pkg/permission/config.go` (+3) — 2 new route-permission registry rows (set-access-vlan + port-binding)
- `pkg/permission/config_v1201_test.go` (NEW, +88) — 3 tests: v1.20.1 rows present + v1.19 intact + `GetPermissionByPath` lookups

## Decisions Made
See key-decisions frontmatter above — primarily the execSinglePort refactor (necessary for pre-binding pattern), the buildAfterValue signature extension (resolved plan's internal contradiction), and the operlog.Record count clarification.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] execSinglePort double-BindJSON EOF on v1.20.1 handlers**
- **Found during:** Task 1 verification (v1.20.1 handler tests returned 400 "请求参数错误: EOF")
- **Issue:** Plan instructed SetAccessVlan/PortBinding handlers to bind their own request struct OUTSIDE execSinglePort (PATTERNS.md Option A). But v1.19 execSinglePort internally called `c.ShouldBindJSON(&PortWriteRequest{})` — when called from the new handlers, the request body had already been consumed by the handler's own bind, causing the internal bind to read EOF and reject with 400.
- **Fix:** Refactored execSinglePort to receive `portID` + `description` as parameters (binding responsibility moved to all 7 callers). The 5 v1.19 handlers now bind `PortWriteRequest` at their top then forward `req.PortID, req.Description` into execSinglePort. v1.19 behavior unchanged (same binding tag, same field semantics); v1.20.1 handlers bind their own struct then forward portID + empty description.
- **Files modified:** internal/api/v1/network/port_write_handler.go (signature change + 7 call-site updates)
- **Verification:** All v1.19 + v1.20.1 handler tests pass; go build + go vet clean
- **Committed in:** e40e4362

**2. [Rule 1 - Documentation] operlog.Record count discrepancy in plan verification step**
- **Found during:** Task 1 acceptance grep verification
- **Issue:** Plan's Task 1 acceptance criterion said `operlog.Record invocation count in the file is 7 (5 v1.19 single-port + 1 batch + 2 new)`. Actual count is 2 statement invocations. The v1.19 DRY pattern routes all 5 single-port handlers through 1 shared `operlog.Record` call inside execSinglePort (the chokepoint) — not 5 separate calls. The 2 new v1.20.1 handlers inherit the same call by flowing through execSinglePort. Plan author miscounted by assuming per-handler operlog calls.
- **Fix:** No code change — the 2 new handlers ARE covered by operlog (via execSinglePort's existing call). Documented here as a deviation to prevent future confusion. Plan's intent (operlog coverage for the 2 new handlers) is satisfied.
- **Files modified:** none (documentation only)
- **Verification:** grep shows 2 `operlog.Record(` statement invocations; both new handlers flow through execSinglePort which contains one of them
- **Committed in:** (documented in this SUMMARY)

**3. [Rule 2 - Critical] Added v1.20.1 handler tests + permission config tests not explicitly listed in plan**
- **Found during:** Task 1 implementation
- **Issue:** Plan's success criteria mentioned "4+ new v1.20.1 handler tests" but didn't enumerate them; without tests the wiring contract (binding tags / sentinel translations / Extra-populated after_value / OperType branching) would be unlocked and silent regressions possible.
- **Fix:** Added 5 handler test funcs (12 subtests) + 1 router test + 3 permission config tests. Tests cover: binding rejection (vlanId 0/4095/10000 + op="DELETE"), 4 service sentinels (each → 400 + no audit + no operlog), success path with Extra-populated after_value, OperType branching (add=1/remove=3/invalid=400), source-grep assertions.
- **Files modified:** port_write_handler_test.go, port_write_router_test.go, pkg/permission/config_v1201_test.go (new)
- **Verification:** All tests pass; 12 new subtests + 3 new permission tests = 15 new test cases
- **Committed in:** e40e4362 + 6887e5c1

---

**Total deviations:** 3 (1 Rule 1 bug + 1 Rule 1 doc + 1 Rule 2 critical)
**Impact on plan:** All auto-fixes were necessary for correctness and contract-locking. Zero scope creep — every change is in service of the plan's stated objective ("Lock the HTTP contract for the 2 new actions").

## Issues Encountered
None beyond the deviations above.

## User Setup Required
None — no external service configuration required. The 2 new endpoints are reachable via the existing v1.19 RBAC group; no menu seed or permission grant changes (group-level middleware auto-covers).

## Next Phase Readiness
- W3 HTTP contract locked; W4 frontend can wire the 2 new endpoints via kebab URLs (`/network/ports/write/set-access-vlan` + `/network/ports/write/port-binding`)
- JSON request bodies match v1.20.1 spec: `SetAccessVlanRequest{portId, vlanId, reason}` and `PortBindingRequest{portId, op, ipAddress, macAddress, reason}`
- OperType mapping is locked per design.md §6; frontend's `showAuditLinkToast` can branch on Create/Update/Delete text
- `PortResult.Extra` → `after_value` JSON pipeline is end-to-end tested (service injection → handler audit row → DB)
- Zero new external dependencies (no go.mod diff)
- Phase 34 operlog regression intact (25 OperType + 11 keywords + 5-param Record)

## Self-Check: PASSED

**Created/modified files exist:**
- FOUND: internal/api/v1/network/port_write_handler.go
- FOUND: internal/api/v1/network/port_write_handler_test.go
- FOUND: internal/api/v1/network/port_write_router.go
- FOUND: internal/api/v1/network/port_write_router_test.go
- FOUND: pkg/permission/config.go
- FOUND: pkg/permission/config_v1201_test.go

**Commits exist:**
- FOUND: e40e4362 (feat 56-03 Task 1)
- FOUND: 6887e5c1 (feat 56-03 Task 2)

**Acceptance gate green:**
- `go build ./...` exits 0
- `go vet ./internal/api/v1/network/...` exits 0
- `go test ./internal/api/v1/network/...` exits 0
- `go test ./internal/services/portwrite/... ./internal/services/portcollection/... ./internal/utils/operlog/...` exits 0
- 2 new endpoints registered (grep confirms 8 `write.POST` calls)
- 2 new permission rows present (grep confirms 9 NetworkPortWrite references)
- 4 new sentinel→HTTP-400 translations wired
- operlog regression intact

---
*Phase: 56-vlan-v1-20-1*
*Completed: 2026-07-09*
