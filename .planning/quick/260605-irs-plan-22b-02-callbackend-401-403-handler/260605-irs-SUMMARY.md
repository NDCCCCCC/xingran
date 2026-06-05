---
phase: quick
plan: 01
subsystem: agent
tags: [retry, auth-refresh, structured-logging, error-handling]
dependency_graph:
  requires: [retry-package, jwt-authenticator, logger]
  provides: [callbackend-retry, token-refresh-on-401-403, unified-handler-responses]
  affects: [internal/agent/server/jwt_auth.go, internal/agent/server/handlers.go, internal/agent/server/connection_manager.go, internal/agent/pkg/retry/retry.go]
tech_stack:
  added: [retry.DoWithRetry, errors.As for AuthError detection, logrus structured logging]
  patterns: [retry-with-exponential-backoff, forced-token-refresh, gin.H-response-unification]
key_files:
  created:
    - internal/agent/pkg/retry/retry.go
    - internal/agent/server/connection_manager.go
  modified:
    - internal/agent/server/jwt_auth.go
    - internal/agent/server/handlers.go
decisions:
  - Used errors.As for AuthError detection to distinguish auth failures from retryable errors
  - Extracted generateAndStoreToken() helper to share logic between RefreshToken and forceRefreshToken
  - Used io.ReadAll instead of json.NewDecoder for response body to enable status-code-based branching
metrics:
  duration: 5m
  tasks_completed: 2
  completed_at: 2026-06-05T05:42:39Z
---

# Quick Plan 01: CallBackend Retry + 401/403 Token Refresh + Handler Cleanup Summary

## One-liner

Add retry with exponential backoff (3 retries, 1s-10s, jitter) to CallBackend, forced token refresh on 401/403, fix retry.go always-true string bug, unify handler responses to gin.H{}, and replace log.Printf with structured logrus logging.

## Tasks Completed

### Task 1: Add retry and 401/403 refresh to CallBackend
- **Commit:** 6f66383
- **Files:** `internal/agent/pkg/retry/retry.go`, `internal/agent/server/jwt_auth.go`

Fixed the critical `containsIgnoreCase` bug that always returned `true`, making all errors match all patterns. Rewrote `CallBackend()` to wrap HTTP requests in `retry.DoWithRetry` with exponential backoff (3 retries, 1s initial, 10s max, jitter). Network errors and 5xx/429 responses trigger retries. On 401/403, the cached token is cleared, a forced refresh is performed (skipping validity check), and the request is retried once. Added `forceRefreshToken()` and `generateAndStoreToken()` helpers.

### Task 2: Unify handler error responses and add structured logging
- **Commit:** 11f97be
- **Files:** `internal/agent/server/handlers.go`, `internal/agent/server/connection_manager.go`

Replaced all `response.Error()`/`response.Success()` calls in handlers.go with `gin.H{}` responses using existing error message constants. Removed the `pkg/response` import. Added structured logging via `WithFields()`/`WithRequestID()` to all handler methods, including request ID extraction from `X-Request-ID` header. Replaced `log.Printf` in `connection_manager.go` with structured logrus logging including reconnect attempt counts.

## Deviations from Plan

None - plan executed exactly as written.

## Threat Model Compliance

| Threat ID | Mitigation | Status |
|-----------|------------|--------|
| T-quick-01 | Max 3 retries, exponential backoff with jitter, max delay 10s | Implemented |
| T-quick-02 | Force refresh clears cached token, single retry after refresh | Implemented |
| T-quick-03 | sanitizeError() strips sensitive patterns from client-facing messages | Preserved (switched to structured logging) |

## Self-Check: PASSED

- [x] FOUND: internal/agent/pkg/retry/retry.go
- [x] FOUND: internal/agent/server/jwt_auth.go
- [x] FOUND: internal/agent/server/handlers.go
- [x] FOUND: internal/agent/server/connection_manager.go
- [x] FOUND: commit 6f66383 (Task 1)
- [x] FOUND: commit 11f97be (Task 2)
