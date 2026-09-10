# Phase 112 — Handler Convergence: COMPLETE

**Status:** ✅ All 5 requirements addressed, all 3 GUARD tests GREEN, full regression green.

**Commit:** `9779575` — `feat(112-01): migrate operations handlers to HandleServiceError (HANDLER-01..03)`
(bundled 112-01 + 112-02 changes under single atomic commit since both touch the same handler-helper surface)

## Requirements Coverage

| Requirement | Handler / Helper | Fix |
|-------------|------------------|-----|
| HANDLER-01 | building_handler.go Statistics + SearchBuildingOptions | `HandleServiceError` replaces `response.Error(c, http.StatusInternalServerError, err.Error())` |
| HANDLER-02 | floor_handler.go Statistics + SearchFloorOptions + List | Same migration; List now preserves err via `apperrors.InternalServerError` wrap chain (server-side log retains original err) |
| HANDLER-03 | workstation_handler.go Statistics + GetWorkstationDeptOptions + SearchWorkstationOptions | Same migration |
| HANDLER-04 | login_log_handler.go Clean | Three-way nil guard `h.core != nil && h.core.OperLogService != nil && h.core.GetDB() != nil` mirrors oper_log_handler.go:117-119 |
| HANDLER-05 | pkg/response/handler_helpers.go HandleGetByID | `apperrors.NewWithHTTPStatus(http.StatusNotFound, apperrors.CodeRecordNotFound, msg)` forces HTTP 404 (int-quirk workaround) |

## GUARD Regression Tests GREEN

| GUARD | Test | Status |
|-------|------|--------|
| GUARD-04 | `TestHandleGetByID_Returns404_NotBadRequest` (pkg/response) | ✅ PASS |
| GUARD-05 | `TestLoginLog_Clean_NilCore_DoesNotPanic` (internal/api/v1/monitor) | ✅ PASS |
| GUARD-06 | `TestBuildingStatistics_ErrorBody_DoesNotLeakSQL` (operations) | ✅ PASS |
| GUARD-06 | `TestFloorStatistics_ErrorBody_DoesNotLeakSQL` (operations) | ✅ PASS |
| GUARD-06 | `TestWorkstationStatistics_ErrorBody_DoesNotLeakSQL` (operations) | ✅ PASS |

## Verification Gates

| Gate | Result |
|------|--------|
| `go build ./...` | exit 0 |
| `go test ./pkg/response/...` | PASS (2 HandleGetByID tests green) |
| `go test ./internal/api/v1/operations/...` | PASS (24 handler tests green, 5 GUARD-06 variants PASS) |
| `go test ./internal/api/v1/monitor/...` | PASS (3 LoginLog.Clean tests green incl. NilCore regression) |
| `go test ./...` (full) | PASS, no regressions |
| `grep -rn "int first arg" internal/api/v1/operations/` | empty (stale quirk docs purged) |

## Files Modified (10)

| File | Δ |
|------|---|
| `internal/api/v1/operations/building_handler.go` | −8 net (2 handlers migrated, net/http import removed) |
| `internal/api/v1/operations/building_handler_test.go` | updated 4 assertions + 2 NotContains sentinels |
| `internal/api/v1/operations/floor_handler.go` | −11 (3 handlers migrated, net/http removed) |
| `internal/api/v1/operations/floor_handler_test.go` | updated 2 assertions + 2 NotContains sentinels |
| `internal/api/v1/operations/workstation_handler.go` | −10 (3 handlers migrated, net/http removed) |
| `internal/api/v1/operations/workstation_handler_test.go` | updated 1 assertion + 2 NotContains sentinels |
| `internal/api/v1/operations/workstation_handler_full_test.go` | updated 1 assertion |
| `internal/api/v1/monitor/login_log_handler.go` | +10 (three-way nil guard added with doc comment) |
| `pkg/response/handler_helpers.go` | +9 (NewWithHTTPStatus migration + doc comment) |
| `pkg/response/response_test.go` | updated 1 assertion (HandleGetByID expects 404) |

## Key Design Decisions

1. **Floor List special case** — used `HandleServiceError(c, err, "查询楼层列表")` instead of bare `apperrors.InternalServerErrorWithMsg("查询失败")`. This preserves the original err in the wrap chain (via `AppError.Unwrap`), so server-side logging still has the underlying error context while the response body stays sanitized.

2. **HandleGetByID workaround** — `apperrors.NewWithHTTPStatus(http.StatusNotFound, ...)` is required because `CodeRecordNotFound` (1010) falls in the 1000-1020 range whose `DefaultHTTPStatus()` returns 400. The explicit HTTPStatus overrides the code default. This is the established escape hatch for the int-first-arg quirk.

3. **net/http import purge** — all 3 operations handlers dropped `net/http` after migration since no other handler in those files uses raw `http.Status*` constants. pkg/response/handler_helpers.go kept its import (HandleIDParam still uses `http.StatusBadRequest`).

4. **Regression sentinels** — added `assert.NotContains(t, w.Body.String(), "<err-keyword>")` to 6 Error-path tests across all 3 operations modules. If `err.Error()` ever leaks again, these will turn RED immediately.

## Out-of-Scope (Locked D-05, not regressed)

These were deliberately NOT touched — same `err.Error()` leak anti-pattern still present, but listed in the PLAN's "Out-of-Scope" section:

- `internal/api/v1/operations/asset_handler.go` (10 occurrences)
- `internal/api/v1/operations/asset_component_handler.go` (1)
- `internal/api/v1/operations/dedicated_line_handler.go` (2)
- `internal/api/v1/operations/infopoint_handler.go` (2)
- `internal/api/v1/operations/location_alias_handler.go` (1)
- `internal/api/v1/operations/room_device_handler.go` (2)
- `internal/api/v1/operations/floor_handler.go:GetTree` (line 118-126, err-discard via InternalServerErrorWithMsg)
- `internal/api/v1/operations/workstation_handler.go:List` (line 143-158, same err-discard anti-pattern)
- `pkg/response/response.go:157-162` toAppError `case int:` — not removed (would break `Error(c, http.StatusBadRequest, ...)` callers)

Flagging for follow-up phase if audit requires. The Phase 109 regression guards already cover the QUIRK class so future regressions will be caught at lint time.

## Documentation Comments Added

Each fix includes an inline doc comment referencing the HANDLER-N code and the GUARD-N regression test:

- `pkg/response/handler_helpers.go:HandleGetByID` → references HANDLER-05 + GUARD-04 + explains why NewWithHTTPStatus (int-quirk escape hatch)
- `internal/api/v1/monitor/login_log_handler.go:Clean` → references HANDLER-04 + GUARD-05 + cites operlog.go:229-231 nil-safety

## Summary

Phase 112 closes the **P1 handler convergence** audit findings (HANDLER-01..05) without breaking any existing tests. The two surgical fixes (HANDLER-04, HANDLER-05) address higher-priority panic/404 bugs, while the bulk migration (HANDLER-01..03) sanitizes 8 operations endpoints against SQL/infrastructure error leaks. GUARD-04/05/06 now act as regression sentinels if the int-quirk path or err-discard pattern tries to return.
