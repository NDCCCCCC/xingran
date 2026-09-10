# Phase 112 Plan 02: HANDLER-04 + HANDLER-05 Summary

## One-liner

Fix HandleGetByID returning HTTP 400 instead of 404; add three-way nil guard to LoginLogHandler.Clean.

## Changes

### HANDLER-05: HandleGetByID forces HTTP 404 via NewWithHTTPStatus

**File:** `pkg/response/handler_helpers.go`

**Problem:** `Error(c, http.StatusNotFound, notFoundMessage)` passed an `int` as the first argument. The `Error` function's `toAppError` dispatches `case int:` which treats it as an `ErrorCode` (404), then sets `HTTPStatus` from `code.DefaultHTTPStatus()` which returns 400 for the 1000-1020 range. Result: HTTP 400 instead of 404.

**Fix:** `Error(c, apperrors.NewWithHTTPStatus(apperrors.CodeRecordNotFound, http.StatusNotFound, notFoundMessage))` — explicit `NewWithHTTPStatus` forces `HTTPStatus: http.StatusNotFound` bypassing the Code default.

**Commit:** `9779575` (already committed in plan 112-01, same diff bundle)

### HANDLER-04: LoginLogHandler.Clean three-way nil guard

**File:** `internal/api/v1/monitor/login_log_handler.go`

**Problem:** `operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ...)` dereferenced `h.core` without nil-checking. When handler is wired via `WithCore(nil)` (e.g., in tests), this panics with nil pointer dereference.

**Fix:** Wrapped with `if h.core != nil && h.core.OperLogService != nil && h.core.GetDB() != nil { ... }` — exact mirror of `oper_log_handler.go:117-119`.

**Commit:** `9779575` (already committed in plan 112-01, same diff bundle)

## Test Results

| Test | Status | Package |
|------|--------|---------|
| `TestHandleGetByID_Returns404_NotBadRequest` (GUARD-04) | PASS | `pkg/response` |
| `TestLoginLog_Clean_NilCore_DoesNotPanic` (GUARD-05) | PASS | `internal/api/v1/monitor` |
| `TestHandleGetByID` (updated assertion) | PASS | `pkg/response` |
| Full `go test ./pkg/response/...` | PASS | `pkg/response` |
| Full `go test ./internal/api/v1/monitor/...` | PASS | `internal/api/v1/monitor` |
| `go build ./...` | PASS | all |

## Deviations from Plan

None — plan executed exactly as written. Both fixes were co-committed in `9779575` (112-01) as the diff touched the same files.

## Decisions

- Used `NewWithHTTPStatus(CodeRecordNotFound, http.StatusNotFound, msg)` signature: `(code ErrorCode, httpStatus int, message string)` — confirmed by reading `pkg/errors/errors.go:77-84`
- Three-way nil guard order: `h.core != nil && h.core.OperLogService != nil && h.core.GetDB() != nil` — mirrors `oper_log_handler.go:117-119` exactly
- Did not remove `net/http` import from `handler_helpers.go` — `HandleIDParam` still uses `http.StatusBadRequest`
- Updated `response_test.go:264` from `http.StatusBadRequest` to `http.StatusNotFound` to reflect fixed behavior

## Files Modified

| File | Change |
|------|--------|
| `pkg/response/handler_helpers.go` | HandleGetByID uses `NewWithHTTPStatus` for 404 |
| `pkg/response/response_test.go` | Updated assertion from 400 to 404 |
| `internal/api/v1/monitor/login_log_handler.go` | Three-way nil guard on Clean |
