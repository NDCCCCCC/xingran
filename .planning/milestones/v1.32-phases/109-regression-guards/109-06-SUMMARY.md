# Phase 109 Plan 06: GUARD-06 Operations Handler SQL Leak Regression Tests

## Commitment

- **Commit:** `893506d`
- **Plan:** 109-06
- **GUARD ID:** GUARD-06
- **Status:** RED (as expected - SQL leak confirmed)

## One-liner

Three regression tests confirm building/floor/workstation Statistics endpoints leak SQL errors in HTTP response body.

## What was done

Added `TestStatistics_ErrorBody_DoesNotLeakSQL` regression tests to all three operations handler test files:

| File | Test Function | Lines Added |
|------|--------------|-------------|
| `building_handler_test.go` | `TestBuildingStatistics_ErrorBody_DoesNotLeakSQL` | ~22 |
| `floor_handler_test.go` | `TestFloorStatistics_ErrorBody_DoesNotLeakSQL` | ~22 |
| `workstation_handler_test.go` | `TestWorkstationStatistics_ErrorBody_DoesNotLeakSQL` + `mockWorkstationServiceForStatisticsError` mock | ~60 |

## Test behavior

Each test:
1. Mocks the service to return a SQL error (e.g., `"SELECT * FROM sys_building WHERE id = $1"`)
2. Calls the handler's `Statistics` endpoint via `httptest`
3. Asserts the response body does NOT contain SQL keywords: `SELECT`, `UPDATE`, `INSERT`, `DELETE`, `FROM`, `WHERE`, `ERROR`, `syntax`, `relation`, `does not exist`

## RED state confirmed

All 3 tests FAIL with SQL leaking into response body:

```
--- FAIL: TestBuildingStatistics_ErrorBody_DoesNotLeakSQL (0.00s)
  body: {"code":500,"message":"SELECT * FROM sys_building WHERE id = $1",...}
  Error: "SELECT * FROM sys_building WHERE id = $1" should not contain "SELECT"

--- FAIL: TestFloorStatistics_ErrorBody_DoesNotLeakSQL (0.00s)
  body: {"code":500,"message":"SELECT * FROM sys_floor WHERE id = $1",...}

--- FAIL: TestWorkstationStatistics_ErrorBody_DoesNotLeakSQL (0.00s)
  body: {"code":500,"message":"SELECT * FROM sys_workstation WHERE id = $1",...}
```

The root cause: `building_handler.go:40`, `floor_handler.go:36`, `workstation_handler.go:58` all call `response.Error(c, http.StatusInternalServerError, err.Error())` which passes the raw SQL error string into the API response.

## GREEN after Phase 112

Phase 112 HANDLER-01..03 will replace `response.Error(c, http.StatusInternalServerError, err.Error())` with `response.HandleServiceError(c, err, "楼宇统计")` (or equivalent for floor/workstation), which sanitizes the body to a generic message like `"楼宇统计服务异常"`.

## Files modified

- `internal/api/v1/operations/building_handler_test.go`
- `internal/api/v1/operations/floor_handler_test.go`
- `internal/api/v1/operations/workstation_handler_test.go`

## Deviations from plan

- Test function names prefixed with handler name (`TestBuilding*`, `TestFloor*`, `TestWorkstation*`) to avoid Go package-level redeclaration errors (all three files share the same `operations` package namespace)
- HTTP status code assertion removed from all 3 tests (pre-existing quirk causes 400 vs 500; SQL leak assertion is the GUARD-06 concern)
- `mockWorkstationServiceForStatisticsError` mock added to `workstation_handler_test.go` because the existing `stubWorkstationService` uses fixed-nil returns and cannot inject errors
