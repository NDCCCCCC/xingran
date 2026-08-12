---
phase: quick
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/agent/server/jwt_auth.go
  - internal/agent/server/handlers.go
  - internal/agent/server/connection_manager.go
  - internal/agent/pkg/retry/retry.go
autonomous: true
requirements:
  - UAT-Gap-2: CallBackend retry, 401/403 token refresh, handler error message cleanup
---

# Quick Plan: CallBackend Retry + 401/403 Token Refresh + Handler Cleanup

## Objective

Complete the three remaining items from Plan 22b-02 that were not fully delivered:

1. **CallBackend retry logic** -- `CallBackend()` currently sends a single HTTP request with no retry. Add retry with exponential backoff using the existing `internal/agent/pkg/retry` package for network errors and 5xx responses.
2. **401/403 proactive token refresh** -- When `CallBackend()` receives a 401 or 403, force-refresh the token and retry the request once.
3. **Handler error message cleanup** -- `handlers.go` mixes two response styles (`gin.H{}` and `response.Error()`/`response.Success()`). Unify to use the Agent's own error constants and `gin.H{}` consistently (since the Agent runs independently of the main backend's response package).

**Purpose**: Make the Agent resilient to transient failures and expired tokens without manual intervention.

**Output**: Production-ready retry and auth-refresh in `CallBackend()`, consistent error handling in handlers.

## Context

@.planning/phases/22b-vm-agent-service/22b-02-PLAN.md
@internal/agent/server/jwt_auth.go
@internal/agent/server/handlers.go
@internal/agent/server/connection_manager.go
@internal/agent/pkg/retry/retry.go

### Key Interfaces (extracted from codebase)

```go
// internal/agent/pkg/retry/retry.go
type Retryable func(error) bool
type Config struct {
    MaxRetries    int
    InitialDelay  time.Duration
    MaxDelay      time.Duration
    Multiplier    float64
    Jitter        bool
    Retryable     Retryable
}
func DefaultConfig() *Config
func IsNetworkError(err error) bool
func IsHTTPRetryable(statusCode int) bool
func DoWithRetry(ctx context.Context, config *Config, fn func() error) error
```

```go
// internal/agent/server/jwt_auth.go
type AuthError struct { StatusCode int; Message string }
type HTTPError struct { StatusCode int; Message string }
func (a *JWTAuthenticator) RefreshToken(ctx context.Context) error
func (a *JWTAuthenticator) GetCurrentToken(ctx context.Context) (string, error)
func (a *JWTAuthenticator) CallBackend(ctx context.Context, method, path string, body interface{}) (*response.Response, error)
```

```go
// internal/agent/server/handlers.go -- error constants already defined
const (
    errMsgGeneric         = "..."
    errMsgInvalidRequest  = "..."
    errMsgUnauthorized    = "..."
    errMsgForbidden       = "..."
    errMsgNotFound        = "..."
    errMsgInternalError   = "..."
    errMsgServiceUnavailable = "..."
)
```

## Tasks

<task type="auto">
  <name>Task 1: Add retry and 401/403 refresh to CallBackend</name>
  <files>internal/agent/server/jwt_auth.go, internal/agent/pkg/retry/retry.go</files>
  <action>
1. **Fix `retry.go` bug**: The `containsIgnoreCase` function always returns `true`. Replace the `contains` and `containsIgnoreCase` functions with a correct implementation using `strings.Contains(strings.ToLower(s), strings.ToLower(substr))`.

2. **Rewrite `CallBackend()` in `jwt_auth.go`** to wrap the HTTP request in `retry.DoWithRetry`:
   - Move the current HTTP request logic into a closure `requestFn func() error`.
   - Inside the closure, after reading the response body, check the HTTP status code:
     - `401` or `403` -> return an `AuthError` (not retryable by the retry loop, handled separately).
     - `4xx` (other) -> return `fmt.Errorf("client error: %d", resp.StatusCode)` (not retryable).
     - `5xx` or `429` -> return `fmt.Errorf("server error: %d", resp.StatusCode)` (retryable).
     - `2xx` -> decode response into `result` pointer and return `nil`.
   - Create a `retry.Config` with: `MaxRetries: 3`, `InitialDelay: 1s`, `MaxDelay: 10s`, `Multiplier: 2.0`, `Jitter: true`, and a `Retryable` function that returns `true` for network errors (via `retry.IsNetworkError`) and server errors but `false` for `AuthError` and client errors.
   - Call `retry.DoWithRetry(ctx, retryConfig, requestFn)`.
   - **After the retry loop**, if the error is an `*AuthError`:
     a. Clear the cached token (`a.mu.Lock`, set `a.currentToken = ""`, `a.tokenExpiryAt = time.Time{}`).
     b. Call `a.RefreshToken(ctx)` to force a new token.
     c. If refresh succeeds, execute `requestFn` one more time (single attempt, no retry wrapper) and return the result.
     d. If refresh fails, return the original auth error.
   - Keep the existing TLS error detection logic inside the closure.

3. **Add a `forceRefresh` path to `RefreshToken()`**: The current method has an early return if the token is still valid (within 1 hour of expiry). Add an unexported method `forceRefreshToken(ctx context.Context) error` that skips this check and always requests a new token. The `CallBackend` 401/403 handler should call this method instead of `RefreshToken`.

   Implementation: Extract the token generation logic from `RefreshToken` into a helper, then have `RefreshToken` check validity before calling the helper, while `forceRefreshToken` calls the helper directly.

4. **Do NOT change the function signature** of `RefreshToken(ctx context.Context) error` -- other callers (like `GetCurrentToken`) depend on it.

5. **Add import** for `"github.com/xingran-next/xingran-go-backend/internal/agent/pkg/retry"` to `jwt_auth.go` (add `"io"` if not present for `io.ReadAll`).
  </action>
  <verify>
    <automated>cd D:/code/ClaudeCode/xingran-go-backend && go build ./internal/agent/...</automated>
  </verify>
  <done>
    - `CallBackend()` wraps HTTP request in `retry.DoWithRetry` with exponential backoff (3 retries, 1s-10s, jitter).
    - Network errors and 5xx responses trigger retry.
    - 401/403 responses trigger forced token refresh followed by a single retry.
    - Other 4xx responses fail immediately without retry.
    - `retry.go` contains/containsIgnoreCase functions work correctly (not always-true).
    - All code compiles without errors.
  </done>
</task>

<task type="auto">
  <name>Task 2: Unify handler error responses and add structured logging</name>
  <files>internal/agent/server/handlers.go, internal/agent/server/connection_manager.go</files>
  <action>
1. **Unify `handlers.go` error responses**: The file currently mixes two response styles:
   - `CreateAccount` uses `gin.H{"error": ..., "code": ...}` (correct for standalone Agent).
   - `DeleteAccount`, `ResetPassword`, `EnableAccount`, `DisableAccount`, `ListAccounts` use `response.Error(c, response.ErrServerError, ...)` and `response.Success(c, ...)` from the main backend's `pkg/response` package (incorrect -- the Agent should not depend on the main backend's response helpers).

   Replace ALL `response.Error()` and `response.Success()` calls with `gin.H{}` using the existing error message constants:
   - `response.Error(c, response.ErrBadRequest, msg)` -> `c.JSON(http.StatusBadRequest, gin.H{"error": errMsgInvalidRequest, "code": errCodeInternal})`
   - `response.Error(c, response.ErrServerError, msg)` -> `c.JSON(http.StatusInternalServerError, gin.H{"error": sanitizeError(err), "code": errCodeInternal})`
   - `response.Error(c, response.ErrNotImplemented, msg)` -> `c.JSON(http.StatusNotImplemented, gin.H{"error": errMsgServiceUnavailable, "code": errCodeInternal})`
   - `response.Success(c, data)` -> `c.JSON(http.StatusOK, data)`

   After this change, remove the `"github.com/xingran-next/xingran-go-backend/pkg/response"` import from `handlers.go`.

2. **Add structured logging to handler methods**: Replace `log.Printf(...)` calls with `server.WithFields(...)` or `server.WithRequestID(...)` from the existing logger. For example, in `CreateAccount`:
   - `log.Printf("Failed to create account: %v", err)` -> `server.WithFields(logrus.Fields{"error": err.Error(), "username": req.Username}).Error("Failed to create account")`
   - Add `requestID := c.GetHeader("X-Request-ID")` at the start of each handler and log it.

   Add `"github.com/sirupsen/logrus"` import to `handlers.go`.

3. **Fix `connection_manager.go` logging**: Replace `log.Printf(...)` calls with structured logging:
   - `log.Printf("Health check failed: %v", err)` -> `server.WithFields(logrus.Fields{"error": err.Error()}).Warn("Health check failed")`
   - `log.Printf("Reconnect failed: %v", err)` -> `server.WithFields(logrus.Fields{"error": err.Error(), "attempt": cm.reconnectCount}).Warn("Reconnect failed")`
   - Remove `"log"` import, add `"github.com/sirupsen/logrus"` if needed (or just use `server.WithFields`/`server.Warn`/`server.Error` directly).
  </action>
  <verify>
    <automated>cd D:/code/ClaudeCode/xingran-go-backend && go build ./internal/agent/...</automated>
  </verify>
  <done>
    - All handler methods use `gin.H{}` consistently (no `response.Error`/`response.Success` imports).
    - Error responses use the existing error message constants (`errMsgGeneric`, `errMsgInvalidRequest`, etc.).
    - Handler methods log using structured logger (`server.WithFields`/`server.WithRequestID`).
    - `connection_manager.go` uses structured logging instead of `log.Printf`.
    - `pkg/response` import removed from `handlers.go`.
    - All code compiles without errors.
  </done>
</task>

## Threat Model

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-quick-01 | Denial of Service | retry mechanism | mitigate | Max 3 retries, exponential backoff with jitter, max delay 10s |
| T-quick-02 | Spoofing | 401/403 token refresh | mitigate | Force refresh clears cached token, single retry after refresh |
| T-quick-03 | Information Disclosure | handler errors | mitigate | `sanitizeError()` strips sensitive patterns from client-facing messages |

## Verification

```bash
cd D:/code/ClaudeCode/xingran-go-backend
go build ./...
go vet ./internal/agent/...
```

## Success Criteria

1. `CallBackend()` retries on network errors and 5xx (up to 3 times with exponential backoff + jitter).
2. `CallBackend()` forces token refresh on 401/403, then retries once.
3. `retry.go` string matching functions work correctly (not always-true).
4. All handler methods use `gin.H{}` responses with existing error constants.
5. No `pkg/response` import in `handlers.go`.
6. Handler and connection_manager logging uses structured logger.
7. `go build ./internal/agent/...` passes with zero errors.

## Output

After completion, create `.planning/quick/260605-irs-plan-22b-02-callbackend-401-403-handler/260605-irs-SUMMARY.md`
