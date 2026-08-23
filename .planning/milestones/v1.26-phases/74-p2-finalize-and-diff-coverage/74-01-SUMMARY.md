---
phase: 74
plan: 01
subsystem: operations-handler-coverage
tags: [coverage, handler-tests, mock-services, p2-finalize]
dependency_graph:
  requires: [phase-72-p0-core-supplement, phase-73-ratchet]
  provides: [operations-handler-tests-suite]
  affects: [internal/api/v1/operations]
tech_stack:
  added:
    - gin test mode + httptest (already in stack)
  patterns:
    - mock-service with per-method *Func fields
    - stubRecorder for operlog interface satisfaction
    - shared handlers_test_helpers_test.go
key-files:
  created:
    - internal/api/v1/operations/handlers_test_helpers_test.go
    - internal/api/v1/operations/building_handler_test.go
    - internal/api/v1/operations/floor_handler_test.go
    - internal/api/v1/operations/workstation_handler_full_test.go
    - internal/api/v1/operations/serverroom_handler_test.go
    - internal/api/v1/operations/infopoint_handler_test.go
    - internal/api/v1/operations/dedicated_line_handler_test.go
    - internal/api/v1/operations/asset_handler_test.go
    - internal/api/v1/operations/location_alias_room_device_handler_test.go
    - internal/api/v1/operations/floor_plan_text_wall_handler_test.go
    - internal/api/v1/operations/workstation_device_handler_test.go
    - internal/api/v1/operations/building_handler_typesafe_test.go
    - internal/api/v1/operations/router_test.go
  modified: []
decisions:
  - id: D-12-STRICT
    summary: Zero business code changes; only *_test.go files in this commit
    rationale: Phase 74 D-12 P2 plan explicitly forbids touching non-test files; coverage ratchet purely via tests
  - id: D-02-D-08-MOCK-PATTERN
    summary: mockXxxService struct with *Func fields (per Phase 73-01 duty_handler_test.go pattern)
    rationale: Compile-time interface satisfaction, zero external test framework dependency
  - id: D-03-OPERLOG-COVERAGE
    summary: stubRecorder satisfying both operlog.Recorder and services.OperLogService interfaces
    rationale: operlog.Record must not panic when core.OperLogService is nil; stub Recorder no-ops the call
  - id: D-15-P2-FLOOR
    summary: Per-package coverage ≥70%
    rationale: Phase 74 P2 floor enforcement; achieved 72.3% from 3.0% baseline
metrics:
  duration_minutes: ~50
  completed_date: 2026-08-21
  baseline_coverage: 3.0
  final_coverage: 72.3
  coverage_delta: 69.3
  test_files_added: 13
  tests_added: 331
---

# Phase 74 Plan 01: Operations Handler Tests Summary

**One-liner:** Comprehensively tested all 13 sub-handlers in `internal/api/v1/operations` (1285 stmts), raising per-package coverage from 3.0% to 72.3% — exceeding the Phase 74 D-15 P2 package floor of ≥70%.

## Coverage Progression

| Milestone | Coverage | Trigger |
|-----------|----------|---------|
| Baseline | 3.0% | Pre-plan state (Phase 73-04 handoff) |
| After 4 handler files (Building/Floor/Workstation/ServerRoom) | 19.8% | First batch committed |
| After InfoPoint + DedicatedLine + Door | 38.4% | Second batch |
| After Asset + LocationAlias + RoomDevice | 53.7% | Third batch |
| After FloorPlanText + Wall | 61.0% | Fourth batch |
| After WorkstationDevice | 66.8% | Fifth batch |
| **After BuildingHandlerTypeSafe + RouterTest** | **72.3%** | Final batch |

## Files Created

| File | Lines | Purpose |
|------|-------|---------|
| `handlers_test_helpers_test.go` | ~120 | Shared `mountRouter`, `routeMount`, `httpDo`, `newTestCore`, `stubRecorder` |
| `building_handler_test.go` | ~370 | All 9 BuildingHandler methods (Statistics/Search/Create/List/GetByID/Update/Delete/Batch/Geocode) |
| `floor_handler_test.go` | ~440 | All 10 FloorHandler methods (incl. GetTree) |
| `workstation_handler_full_test.go` | ~485 | All 13 WorkstationHandler methods (incl. BatchUpdatePositions, DeptOptions) |
| `serverroom_handler_test.go` | ~330 | All 9 ServerRoomHandler methods |
| `infopoint_handler_test.go` | ~390 | All 9 InfoPointHandler methods + `validateDevicePortConsistency` |
| `dedicated_line_handler_test.go` | ~475 | DedicatedLine + Door handler tests |
| `asset_handler_test.go` | ~340 | All 11 AssetHandler methods (incl. SearchBySerial, Statistics, GetDeviceTypes/Categories/StatusValues) |
| `location_alias_room_device_handler_test.go` | ~445 | LocationAlias + RoomDevice handler tests |
| `floor_plan_text_wall_handler_test.go` | ~395 | FloorPlanText + Wall handler tests |
| `workstation_device_handler_test.go` | ~370 | All 11 WorkstationDeviceHandler methods |
| `building_handler_typesafe_test.go` | ~230 | All 7 BuildingHandlerTypeSafe methods |
| `router_test.go` | ~225 | Smoke test verifying every handler can be mounted and returns valid HTTP status |

## Documented Quirks (D-12 STRICT — no business code changed)

| # | Quirk | Where | Observed behavior |
|---|-------|-------|-------------------|
| 1 | `response.Error(c, int, string)` int-first-arg | `pkg/response/response.go` | Returns 400 regardless of int value (because `toAppError case int` hardcodes HTTPStatus to 400) |
| 2 | `apperrors.XxxNotFound()` codes 3010/3040/etc | `pkg/errors/codes.go` `DefaultHTTPStatus` | Codes in range 2000-8999 map to 400 BadRequest, not 404 |
| 3 | `BuildingHandler.Geocode` empty-address branch | `building_handler.go` | Unreachable via valid JSON due to `binding:"required"` |
| 4 | `handleJSONBinding` vs fallback `params = make(map[string]interface{})` divergence | Across handlers | Building/Floor/Workstation differ in List bind-error fallback semantics |

Per D-12 (zero business code changes), these quirks are documented for awareness but NOT fixed.

## Mock Service Pattern (per Phase 73-01)

All 13 sub-handlers use a consistent mock pattern:

```go
type mockXxxService struct {
    MethodAFunc func(ctx context.Context, args...) (return...)
    MethodBFunc func(ctx context.Context, args...) (return...)
    ...
}

func (m *mockXxxService) MethodA(ctx context.Context, args...) (return...) {
    if m.MethodAFunc != nil {
        return m.MethodAFunc(ctx, args...)
    }
    return errNotImplemented  // forces test to set the field
}
```

This pattern:
- Compiles against the actual service interface (compile-time safety)
- Forces each test to explicitly set the mock behavior it needs
- No external mocking framework dependency

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Field name corrections in test data**
- **Found during:** Writing asset_handler_test.go (Asset.DeviceTypeCount.Type → Value)
- **Issue:** Service struct field names didn't match my initial guesses
- **Fix:** Verified actual struct definitions and updated test data; no business code touched
- **Files modified:** `asset_handler_test.go` (3 lines)

**2. [Rule 3 - Blocking] WorkstationDeviceService interface completeness**
- **Found during:** Writing `workstation_device_handler_test.go`
- **Issue:** `opsServices.WorkstationDeviceService` interface has 12 methods including `GetADDevicesByUser`, `GetAssetDevicesByUser`, `GetADDevicesByWorkstations`, `GetAssetDevicesByWorkstations`, `GetPhysicalDevicesByWorkstations`, `SetPrimaryAndSaveBySerial(workstationID, serial, req)` — added stub methods on mock to satisfy interface
- **Fix:** Added 6 stub methods returning `nil`/empty; no business code touched
- **Files modified:** `workstation_device_handler_test.go` (added stub methods)

### Stub Methods Required for Interface Compliance

The `mockWorkstationDeviceService` had to implement these methods (not exercised by tests but required for interface satisfaction):
- `GetADDevicesByWorkstations` → returns empty map
- `GetAssetDevicesByWorkstations` → returns empty map
- `GetPhysicalDevicesByWorkstations` → returns empty map
- `GetADDevicesByUser` → returns nil
- `GetAssetDevicesByUser` → returns nil
- `SetPrimaryAndSaveBySerial` → returns nil

### Quirks Discovery

Per D-12, the following pre-existing business code quirks were DISCOVERED but NOT fixed (out of scope; documenting for future plan):

1. **`response.Error(c, int, string)` returns 400 instead of int** — affects all 11 handlers using `http.StatusInternalServerError` as first arg (asset_handler.go, room_device_handler.go, etc.). Tests adapted to expect 400 with comments.
2. **`apperrors.XxxNotFound()` codes in 2000-8999 range return 400** — affects building/floor/wall/info-point/room-device "NotFound" tests.
3. **`BuildingHandler.Geocode` empty-address branch is unreachable** — `binding:"required"` on Address makes the manual check dead code.

## Self-Check: PASSED

- All 13 `*_test.go` files compile
- All 331 tests pass
- Coverage 72.3% (≥70% target met)
- No business code modified