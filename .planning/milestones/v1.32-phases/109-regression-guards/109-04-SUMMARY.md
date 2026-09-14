# Phase 109 Plan 04: GUARD-04 HandleGetByID 404 Regression Test — Summary

## Execution Summary

**Plan:** 109-04
**GUARD ID:** GUARD-04
**Test:** `TestHandleGetByID_Returns404_NotBadRequest`
**Location:** `pkg/response/handler_helpers_test.go`
**Commit:** `1375a87`

## Test Behavior

| State | Condition | Result |
|-------|-----------|--------|
| **RED (Phase 109)** | `handler_helpers.go:62` passes `http.StatusNotFound` (int) to `Error()`, `toAppError` `case int:` maps it to HTTP 400 | **FAIL** (expected) |
| **GREEN (Phase 112)** | Phase 112 HANDLER-05 uses `apperrors.New(404, ...)` so `toAppError` returns HTTP 404 | **PASS** |

## Bug Root Cause

```go
// handler_helpers.go:60-63
entity, err := getter(id)
if err != nil {
    Error(c, http.StatusNotFound, notFoundMessage)  // passes int → toAppError treats as code
    return false
}

// response.go:157-162 (toAppError int case):
case int:
    return &AppError{
        Code:       e,                          // e = 404 (StatusNotFound) treated as code
        Message:    "操作失败",
        HTTPStatus: http.StatusBadRequest,      // int falls through to 400!
    }
```

## Test Strategy

1. Create a Gin test context with an `id` URL param
2. Define a getter that returns an error
3. Call `HandleGetByID` with a custom `notFoundMessage` ("实体不存在")
4. Assert `w.Code == http.StatusNotFound` (404), not `http.StatusBadRequest` (400)
5. Assert the response body contains the `notFoundMessage`

## Current Test Output (RED)

```
=== RUN   TestHandleGetByID_Returns404_NotBadRequest
    handler_helpers_test.go:42:
        Error:      Not equal:
                     expected: 404
                     actual  : 400
        Test:       TestHandleGetByID_Returns404_NotBadRequest
        Messages:   HandleGetByID should return HTTP 404 when getter returns error, got 400
--- FAIL: TestHandleGetByID_Returns404_NotBadRequest (0.00s)
```

## Phase 112 Fix Required

In `handler_helpers.go:62`, replace:
```go
Error(c, http.StatusNotFound, notFoundMessage)
```

With the apperrors-based approach (HANDLER-05) so `toAppError` returns HTTP 404 instead of 400.

## Self-Check: PASSED

- Test file exists: `pkg/response/handler_helpers_test.go`
- Commit exists: `1375a87`
- Test FAILS (RED confirmed): exit code 1, got 400 expected 404
