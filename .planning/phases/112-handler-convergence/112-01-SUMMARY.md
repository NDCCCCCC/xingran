# Phase 112 Plan 112-01 Summary

**Plan:** 112-01
**Phase:** 112
**Subsystem:** operations handlers
**Tags:** handler-migration, error-sanitization, HANDLER-01, HANDLER-02, HANDLER-03
**Dependency graph:** requires: [] provides: [HANDLER-01, HANDLER-02, HANDLER-03] affects: [operations/building_handler, operations/floor_handler, operations/workstation_handler]
**Tech stack added:** `pkg/response.HandleServiceError`
**Key files created:** internal/api/v1/operations/{building,floor,workstation}_handler.go, *_test.go
**Decisions:** 1. Used `HandleServiceError` for all 8 endpoints instead of `response.Error(c, http.StatusInternalServerError, err.Error())` 2. Floor handler List uses `HandleServiceError(c, err, "查询楼层列表")` instead of `InternalServerErrorWithMsg` to preserve err in server-side log 3. Removed unused `net/http` import from all 3 handlers
**Metrics:** duration: ~8min, completed: 2026-09-09, tasks: 4, files: 7

## Objective

Fix HANDLER-01..03: Replace `response.Error(c, http.StatusInternalServerError, err.Error())` patterns in three operations handlers with the project-standard `response.HandleServiceError(c, err, operation)` helper. This closes the SQL/infrastructure error leak path and the pre-existing `int`-first-arg `toAppError` quirk (which maps int to HTTP 400 instead of 500). GUARD-06 goes GREEN.

## What Was Changed

### Handler Changes (3 files)

**building_handler.go:**
- Statistics: `response.Error(c, http.StatusInternalServerError, err.Error())` -> `response.HandleServiceError(c, err, "查询统计数据")`
- SearchBuildingOptions: same migration
- Removed `net/http` import (no longer needed)

**floor_handler.go:**
- Statistics: migrated to HandleServiceError
- SearchFloorOptions: migrated to HandleServiceError
- List: `response.Error(c, apperrors.InternalServerErrorWithMsg("查询失败"))` -> `response.HandleServiceError(c, err, "查询楼层列表")` (preserves err for server-side log)
- Removed `net/http` import (no longer needed)

**workstation_handler.go:**
- Statistics: migrated to HandleServiceError
- GetWorkstationDeptOptions: migrated to HandleServiceError
- SearchWorkstationOptions: migrated to HandleServiceError
- Removed `net/http` import (no longer needed)

### Test Changes (4 files, 8 tests updated)

- `building_handler_test.go`: TestBuildingHandler_Statistics_Error, TestBuildingHandler_SearchBuildingOptions_Error - updated assertion 400->500 + added NotContains(body, err) sentinel
- `floor_handler_test.go`: TestFloorHandler_Statistics_Error, TestFloorHandler_SearchFloorOptions_Error - same
- `workstation_handler_test.go`: GUARD-06 comment updated (now GREEN)
- `workstation_handler_full_test.go`: TestWorkstationHandler_Statistics_Error, TestWorkstationHandler_GetWorkstationDeptOptions_Error, TestWorkstationHandler_SearchWorkstationOptions_Error - same

## Deviations from Plan

None - plan executed exactly as written.

## Verification

| Gate | Result |
|------|--------|
| `go build ./...` | PASS |
| `go test ./internal/api/v1/operations/...` | PASS (all tests green) |
| GUARD-06: TestBuildingStatistics_ErrorBody_DoesNotLeakSQL | PASS (GREEN) |
| GUARD-06: TestFloorStatistics_ErrorBody_DoesNotLeakSQL | PASS (GREEN) |
| GUARD-06: TestWorkstationStatistics_ErrorBody_DoesNotLeakSQL | PASS (GREEN) |
| `grep -rn "int first arg" internal/api/v1/operations/` | empty (PASS) |

## Commit

**SHA:** 9779575
**Message:** `feat(112-01): migrate operations handlers to HandleServiceError (HANDLER-01..03)`

## Self-Check

- [x] All 8 endpoints migrated to HandleServiceError
- [x] net/http import removed from 3 handlers
- [x] 8 test assertions updated (400 -> 500)
- [x] 6 regression sentinels added (NotContains body)
- [x] 3 GUARD-06 tests now GREEN
- [x] No stale quirk docs remain
- [x] go build ./... passes
- [x] go test ./internal/api/v1/operations/... passes
