# Phase 104: handler层架构收敛（wire契约+样板去重）- Research

**Researched:** 2026-09-08
**Domain:** Go handler layer architecture convergence, error response wire contract unification, code duplication elimination
**Confidence:** HIGH

## Summary

Phase 104 addresses three distinct technical debt items in the handler layer: (1) wire contract bifurcation between `operations/base_handler.go` local helpers and `pkg/response/handler_helpers.go`, (2) 14 near-identical CRUD handler templates in the operations module, and (3) duplicate log handlers in the monitor module. The unified approach uses `apperrors.AppError` as the single error carrier, rewrites `pkg/response/handler_helpers.go` as the sole authority, converges the 14 handler templates via small function families, and deduplicates monitor handlers via a generic `MonitorLogHandler[T]`.

**Primary recommendation:** Execute in three waves — wire contract first (all callers migrate to new helpers), then handler template convergence, then monitor deduplication. The BusinessError migration and deletion must happen in the same commit as the new helper to avoid a broken intermediate state.

---

## User Constraints (from 104-CONTEXT.md)

### Locked Decisions

- **D-104-1:** Direction = C (重写统一权威) — `pkg/response/handler_helpers.go` rewritten as sole authority; `BusinessError` eliminated
- **D-104-2:** 3 BusinessError construction sites in `config_restore_task_service.go:76,93,112` → `apperrors.WrapWithHTTPStatus`; `pkg/response/business_error.go` deleted in same commit
- **D-104-3:** Binding error message = `err.Error()` (includes gin validator field names)
- **D-104-4:** `Response.code` = business code (1001/1500/1009...); `HTTPStatus` = semantically derived (400/409/500)
- **D-104-5:** Unified helper message strategy: binding errors pass through `err.Error()`, apperrors use business text, bare internal errors use `operation + "失败"` (no `err.Error()` leaked to client)
- **D-104-6:** Convergence mechanism = four-step pipeline small function family (`bind → service → err → operlog → success`); no new files created
- **D-104-7:** 14 handler files preserved; swagger comments retained; personalized methods (Statistics/Geocode/SearchOptions) untouched
- **D-104-8:** Monitor deduplication via generic `MonitorLogHandler[T]`
- **D-104-9:** `OperLog.Clean` and `LoginLog.Clean` retained per file; `LoginLog.UnlockUser` untouched (Phase 107 TODO-03)
- **D-104-10:** httptest contract tests assert `(HTTPStatus, code, message)` triple; 200+ existing tests need code assertion value upgrade (1001→400 / 1500→500)

### Deferred Ideas (OUT OF SCOPE)

- `UnlockUser` implementation (login_log_handler.go:188) — Phase 107 TODO-03
- Generic handler extension for future log types
- Frontend business-code branching logic

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Error wire contract | API/Backend (`pkg/response`) | Handler | `handler_helpers.go` lives in pkg, called by all handlers |
| Operations CRUD handlers | API/Backend (`operations/`) | — | 14 handlers in `internal/api/v1/operations/` |
| Monitor log handlers | API/Backend (`monitor/`) | — | `oper_log_handler.go`, `login_log_handler.go` |
| BusinessError migration | API/Backend (`config_restore_task_service`) | Service | 3 call sites in service layer |
| Handler tests | API/Backend (tests) | — | `*_handler_test.go` files |

---

## Standard Stack

No new packages required. This is a pure refactoring phase using existing project infrastructure.

### Files Modified (by phase wave)

| Wave | Files | Action |
|------|-------|--------|
| 1 | `pkg/response/handler_helpers.go` | Rewrite `HandleJSONBinding` + `HandleServiceError` |
| 1 | `internal/api/v1/operations/base_handler.go` | Delete (merged into pkg) |
| 1 | `internal/services/config_restore_task_service.go` | Migrate 3 BusinessError → `WrapWithHTTPStatus` |
| 1 | `pkg/response/business_error.go` | Delete |
| 1 | `internal/services/config_restore_task_service_97_03_test.go` | Upgrade type assertions |
| 2 | `internal/api/v1/operations/{wall,server_room,floor,door,building,room_device,floor_plan_text,dedicated_line,location_alias,infopoint,workstation,workstation_device,asset,asset_component}_handler.go` | Switch to `response.HandleJSONBinding` + `response.HandleServiceError` |
| 3 | `internal/api/v1/monitor/oper_log_handler.go` | Consolidate into `MonitorLogHandler[T]` |
| 3 | `internal/api/v1/monitor/login_log_handler.go` | Consolidate into `MonitorLogHandler[T]` |
| 3 | `internal/api/v1/monitor/log_handler.go` (new) | Generic `MonitorLogHandler[T]` struct |

### Key APIs Used

| API | Location | Purpose |
|-----|----------|---------|
| `apperrors.NewWithHTTPStatus(code, httpStatus, msg)` | `pkg/errors/errors.go:78` | Create app error with custom HTTP status |
| `apperrors.WrapWithHTTPStatus(err, code, httpStatus, msg)` | `pkg/errors/errors.go:100` | Wrap existing error with HTTP status |
| `response.HandleJSONBinding(c, obj)` | `pkg/response/handler_helpers.go` (new) | Unified binding helper |
| `response.HandleServiceError(c, err, operation)` | `pkg/response/handler_helpers.go` (new) | Unified service error handler |
| `response.Success(c, data)` | `pkg/response/response.go:74` | Success response |
| `response.Error(c, err, msg...)` | `pkg/response/response.go:87` | Error response |
| `response.Page(c, list, total, current, pageSize)` | `pkg/response/response.go:123` | Paginated response |

---

## Architecture Patterns

### Pattern 1: Unified Wire Contract (D-104-1 through D-104-5)

**What:** Single authoritative error handling in `pkg/response/handler_helpers.go`. All handlers across all modules use the same two helpers.

**When to use:** Every handler method that binds JSON and calls a service.

**New `HandleJSONBinding` implementation:**
```go
// Unified: pkg/response/handler_helpers.go
func HandleJSONBinding(c *gin.Context, obj interface{}) bool {
    if err := c.ShouldBindJSON(obj); err != nil {
        // D-104-3: Pass through err.Error() (contains field name + gin validator message)
        // D-104-5: Binding error → operation + "失败" prefix NOT applied here
        Error(c, apperrors.Wrap(err, apperrors.CodeParamError, err.Error()))
        return false
    }
    return true
}
```

**New `HandleServiceError` implementation:**
```go
// Unified: pkg/response/handler_helpers.go
func HandleServiceError(c *gin.Context, err error, operation string) bool {
    if err == nil {
        return true
    }
    // D-104-4: apperrors.AppError carries business code in .Code and HTTPStatus in .HTTPStatus
    // response.Error() reads both via toAppError() conversion
    Error(c, err, operation+"失败")
    return false
}
```

**Message strategy (D-104-5) breakdown:**
- Binding error: `err.Error()` (gin validator output like `"user_name: cannot be blank"`)
- Apperrors business error: `e.Message` (business text from apperrors constructors)
- Bare internal error: `operation + "失败"` — `err.Error()` NOT leaked to client; err detail goes to server log via request_id correlation

**Why this pattern:** `response.Error()` already has `toAppError()` that converts `*apperrors.AppError` to `*response.AppError` extracting `int(appErr.Code)` and `appErr.GetHTTPStatus()`. The new helpers simply ensure all errors flow through this single conversion path.

### Pattern 2: Handler Template Convergence (D-104-6 through D-104-7)

**What:** Extract `bind → service → err → operlog → success` into small functions within each handler file. No new files.

**When to use:** For each of the 14 operations CRUD handlers.

**Example refactoring (building_handler.go Create method):**

**Before:**
```go
func (h *BuildingHandler) Create(c *gin.Context) {
    var building operations.OpsBuilding
    if !handleJSONBinding(c, &building) {  // local helper
        return
    }
    if !handleServiceError(c, h.service.Create(c.Request.Context(), &building), "创建") {
        return
    }
    operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "楼宇管理", operlog.OperTypeCreate)
    response.Success(c, building)
}
```

**After (small function family within same file):**
```go
// At file level, after type definition:
func (h *BuildingHandler) createWithOperLog(c *gin.Context, building *operations.OpsBuilding, createFn func() error) bool {
    if !response.HandleJSONBinding(c, building) {
        return false
    }
    if !response.HandleServiceError(c, createFn(), "创建") {
        return false
    }
    operlog.Record(c, h.core.OperLogService, h.core.GetDB(), "楼宇管理", operlog.OperTypeCreate)
    response.Success(c, building)
    return true
}

func (h *BuildingHandler) Create(c *gin.Context) {
    var building operations.OpsBuilding
    h.createWithOperLog(c, &building, func() error {
        return h.service.Create(c.Request.Context(), &building)
    })
}
```

**Key insight from D-104-6:** Go higher-order functions are the standard approach for "homogeneous flow + localized variation." Generics are NOT appropriate here because service interface signatures vary significantly across the 14 handlers (List params differ, entity-specific methods are numerous).

**Personalized methods (Statistics/Geocode/SearchOptions) unchanged per D-104-7.**

**server_room drift fix (List:101-103):**
```go
// server_room_handler.go:101 — currently:
response.Error(c, apperrors.InternalServerErrorWithMsg("查询失败"))
// Should be:
response.HandleServiceError(c, err, "查询")
```

### Pattern 3: Monitor Generic Handler (D-104-8 through D-104-9)

**What:** `MonitorLogHandler[T]` generic struct holding the 5 duplicate methods.

**When to use:** When `OperLogService` and `LoginLogService` have identical method signatures.

**Interface constraint (T upper bound):**
```go
// Both services implement this interface identically
type MonitorLogService interface {
    List(ctx context.Context, params interface{}) (*PageResult, error)
    GetByID(ctx context.Context, id string) (interface{}, error)
    Delete(ctx context.Context, id string) error
    BatchDelete(ctx context.Context, ids []string) error
    Clean(ctx context.Context) error
}
```

**Generic struct:**
```go
// internal/api/v1/monitor/log_handler.go (new file)
type MonitorLogHandler[T any] struct {
    service MonitorLogService
    core    *core.Core
    moduleName string  // "操作日志" or "登录日志"
}

func NewMonitorLogHandler[T any](service MonitorLogService, moduleName string) *MonitorLogHandler[T] {
    return &MonitorLogHandler[T]{service: service, moduleName: moduleName}
}
```

**5 generic methods:**
```go
func (h *MonitorLogHandler[T]) List(c *gin.Context) { ... }
func (h *MonitorLogHandler[T]) GetByID(c *gin.Context) { ... }
func (h *MonitorLogHandler[T]) Delete(c *gin.Context) { ... }
func (h *MonitorLogHandler[T]) BatchDelete(c *gin.Context) { ... }
```

**Non-generic methods kept in original files:**
- `OperLogHandler.Clean` — synchronous audit transaction + post-clean verification (200-line chicken-and-egg defense)
- `LoginLogHandler.Clean` — simple async operlog.Record
- `LoginLogHandler.UnlockUser` — Phase 107 TODO-03 placeholder, untouched

**Router registration:**
```go
// oper_log_router.go
operLogHandler := monitor.NewMonitorLogHandler[models.OperLog](operLogService, "操作日志")

// login_log_router.go  
loginLogHandler := monitor.NewMonitorLogHandler[models.LoginLog](loginLogService, "登录日志")
```

### Pattern 4: BusinessError Migration (D-104-2)

**What:** Replace 3 `&response.BusinessError{...}` constructions with `apperrors.WrapWithHTTPStatus(nil, code, httpStatus, msg)`.

**Locations and mappings:**

| Location | Old | New |
|----------|-----|-----|
| config_restore_task_service.go:76 | `&response.BusinessError{HTTPStatus:400, Code:400001, Message:"备份不属于目标设备..."}` | `apperrors.WrapWithHTTPStatus(nil, 400001, 400, "备份不属于目标设备，仅限恢复到备份源设备")` |
| config_restore_task_service.go:93 | `&response.BusinessError{HTTPStatus:409, Code:409001, Message:"该设备存在进行中的恢复任务"}` | `apperrors.WrapWithHTTPStatus(nil, 409001, 409, "该设备存在进行中的恢复任务")` |
| config_restore_task_service.go:112 | `&response.BusinessError{HTTPStatus:409, Code:409001, Message:"该设备存在进行中的恢复任务"}` | `apperrors.WrapWithHTTPStatus(nil, 409001, 409, "该设备存在进行中的恢复任务")` |

**Note:** `WrapWithHTTPStatus` returns `nil` when `err == nil` (line 101-103 of errors.go). Since these are pure business errors with no underlying error, passing `nil` is correct. The function checks `if err == nil { return nil }` — but this is for the case where the caller passes nil; our construction with `nil` err and non-empty message returns a valid `*AppError` because the early return only triggers when BOTH err is nil AND the function was called to wrap (not to construct).

Wait — looking at `WrapWithHTTPStatus` line 101-103:
```go
func WrapWithHTTPStatus(err error, code ErrorCode, httpStatus int, message string) *AppError {
    if err == nil {
        return nil  // This means if err is nil, returns nil AppError!
    }
    ...
}
```

This is a problem. The 3 call sites need to create a NEW error without underlying err. `WrapWithHTTPStatus` returns `nil` when `err == nil`, which would be wrong.

Looking more carefully at line 99-110:
```go
func WrapWithHTTPStatus(err error, code ErrorCode, httpStatus int, message string) *AppError {
    if err == nil {
        return nil  // Line 101-102
    }
    return &AppError{
        Code:       code,
        Message:    message,
        HTTPStatus: httpStatus,
        Err:        err,
    }
}
```

This is indeed a problem for pure business error construction. The D-104-2 decision says to use `WrapWithHTTPStatus` for the migration, but it returns `nil` when `err == nil`.

However, `NewWithHTTPStatus(code, httpStatus, message)` (line 78) does NOT have this restriction:
```go
func NewWithHTTPStatus(code ErrorCode, httpStatus int, message string) *AppError {
    return &AppError{
        Code:       code,
        Message:    message,
        HTTPStatus: httpStatus,
    }
}
```

**Correction to D-104-2 migration:**
- Use `apperrors.NewWithHTTPStatus(code, httpStatus, msg)` NOT `WrapWithHTTPStatus(nil, ...)` because the latter returns nil.

This is an important implementation detail the planner must capture.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Error response wire contract | Parallel BusinessError + apperrors systems | Single `pkg/response/handler_helpers.go` authority | Wire contract bifurcation causes inconsistent (HTTPStatus, code) tuples across modules |
| Monitor log handler duplication | Copy-paste 5 methods per log type | `MonitorLogHandler[T]` generic | 10+ nearly identical methods across 2 files |
| Binding error message construction | Hard-coded "参数错误" + appending | `err.Error()` directly | gin validator provides field-level messages already localized |

---

## Common Pitfalls

### Pitfall 1: WrapWithHTTPStatus returns nil for nil error

**What goes wrong:** Using `WrapWithHTTPStatus(nil, code, status, msg)` produces `nil` instead of an `*AppError`, causing nil pointer panics downstream.

**Why it happens:** `WrapWithHTTPStatus` is designed to wrap existing errors; it returns `nil` when `err == nil` as a convenience to avoid nil-checking at call sites.

**How to avoid:** Use `NewWithHTTPStatus(code, status, msg)` for pure business errors (no underlying error to wrap).

### Pitfall 2: server_room List handler drift not caught

**What goes wrong:** `server_room_handler.go:101` uses `response.Error(c, apperrors.InternalServerErrorWithMsg("查询失败"))` which bypasses the standard service error flow. This discards the actual error detail.

**How to avoid:** During handler migration, grep for all direct `response.Error` calls with `apperrors` argument and verify each goes through `HandleServiceError`.

### Pitfall 3: Test assertion values hardcoded after wire change

**What goes wrong:** 200+ handler tests currently assert `code: 0` or check HTTP status only. After the wire contract change, `Response.code` becomes business code (e.g., 1001 for param error, 1500 for server error), not HTTP status.

**How to avoid (D-104-10):** Upgrade test assertions in the same commit:
- Binding error: `code: 1001`, `httpStatus: 400`
- Service error: `code: 1500`, `httpStatus: 500`
- Not-found: `code: 1010`, `httpStatus: 404`

### Pitfall 4: response.Error() message override

**What goes wrong:** `response.Error(c, err, message...)` accepts an optional message that OVERRIDES `appErr.Message`. The new `HandleServiceError` passes `operation+"失败"` which overrides the apperrors.Message.

**Why this is correct for D-104-5:** The D-104-5 decision says apperrors business errors use their business text. But the new `HandleServiceError` passes `operation+"失败"` as override. This is a conflict — the override always wins.

**Resolution:** `HandleServiceError` should NOT pass a message override when the error is already an apperrors with a good message. The `operation+"失败"` prefix should only apply to bare errors (non-apperrors).

```go
// Corrected HandleServiceError:
func HandleServiceError(c *gin.Context, err error, operation string) bool {
    if err == nil {
        return true
    }
    // Only apply operation prefix for non-apperrors; apperrors already has business message
    if !apperrors.IsAppError(err) {
        Error(c, err, operation+"失败")  // override with operation prefix
    } else {
        Error(c, err)  // apperrors.Message is already business-appropriate
    }
    return false
}
```

### Pitfall 5: operations/base_handler.go deleted before all callers migrated

**What goes wrong:** If `base_handler.go` is deleted before all 14 handlers are updated to use `pkg/response.HandleJSONBinding` and `pkg/response.HandleServiceError`, compilation fails.

**How to avoid:** Migrate handlers in wave 2 BEFORE deleting base_handler.go in wave 1 (or keep base_handler.go as thin aliases during transition).

---

## Code Examples

### New unified HandleServiceError (pkg/response/handler_helpers.go)

```go
// Source: derived from D-104-3, D-104-4, D-104-5
func HandleServiceError(c *gin.Context, err error, operation string) bool {
    if err == nil {
        return true
    }
    // D-104-4: apperrors.AppError carries business code in .Code field
    // D-104-5: For non-apperrors, apply operation prefix; apperrors already has business message
    if !apperrors.IsAppError(err) {
        Error(c, err, operation+"失败")
    } else {
        Error(c, err)
    }
    return false
}
```

### New unified HandleJSONBinding (pkg/response/handler_helpers.go)

```go
// Source: derived from D-104-3, D-104-5
func HandleJSONBinding(c *gin.Context, obj interface{}) bool {
    if err := c.ShouldBindJSON(obj); err != nil {
        // D-104-3: Pass through err.Error() — gin validator includes field name
        // Example: "user_name: cannot be blank" or "email: must be valid email"
        Error(c, apperrors.Wrap(err, apperrors.CodeParamError, err.Error()))
        return false
    }
    return true
}
```

### BusinessError migration (config_restore_task_service.go)

**Before:**
```go
return nil, &response.BusinessError{
    HTTPStatus: 400,
    Code:       400001,
    Message:    "备份不属于目标设备，仅限恢复到备份源设备",
}
```

**After:**
```go
return nil, apperrors.NewWithHTTPStatus(400001, 400, "备份不属于目标设备，仅限恢复到备份源设备")
```

### Generic MonitorLogHandler (internal/api/v1/monitor/log_handler.go)

```go
// Source: derived from D-104-8, D-104-9
type MonitorLogHandler[T any] struct {
    service     MonitorLogService
    core        *core.Core
    moduleName  string
}

func NewMonitorLogHandler[T any](svc MonitorLogService, moduleName string) *MonitorLogHandler[T] {
    return &MonitorLogHandler[T]{service: svc, moduleName: moduleName}
}

func (h *MonitorLogHandler[T]) List(c *gin.Context) {
    var params map[string]interface{}
    if !response.HandleJSONBinding(c, &params) {
        return
    }
    result, err := h.service.List(c.Request.Context(), params)
    if !response.HandleServiceError(c, err, "查询") {
        return
    }
    response.Success(c, result)
}
```

---

## Runtime State Inventory

> Not applicable — this is a pure refactoring phase with no rename/rebrand/migration scope.

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `operations/base_handler.go` local helpers | `pkg/response/handler_helpers.go` single authority | Phase 104 | All 14 ops handlers + 6 other modules unified |
| `response.BusinessError` parallel error type | `apperrors.AppError` only | Phase 104 | Wire contract simplified; bizCode removed from response data |
| `Response.code = HTTPStatus` | `Response.code = business code (1001/1500/1009...)` | Phase 104 | Frontend/api.ts interceptor needs no change (only checks `code === 0`) |
| Monitor OperLogHandler + LoginLogHandler (copy-paste) | `MonitorLogHandler[T]` generic | Phase 104 | 5 duplicate methods eliminated |

---

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `NewWithHTTPStatus` (not `WrapWithHTTPStatus`) is the correct migration target for the 3 BusinessError construction sites | D-104-2 implementation | If `WrapWithHTTPStatus(nil, ...)` actually works (returns valid AppError), the correction is unnecessary |
| A2 | The 5 monitor service methods (List/GetByID/Delete/BatchDelete/Clean) have compatible enough signatures for a common interface | D-104-8 | If signatures differ in ways not visible from the handler files, generics may not be viable |

---

## Open Questions

1. **MonitorLogService interface boundary**
   - What we know: `OperLogService.List` takes `OperLogListParams`, `LoginLogService.List` takes `LoginLogListParams`
   - What's unclear: Whether a single `MonitorLogService` interface with `interface{}` params would satisfy the type system, or if we need to parameterize the params type too
   - Recommendation: Use `any` for params in the shared interface; the concrete handler implements type-safe wrappers

2. **Generic handler constructor naming**
   - D-104-8 says `NewOperLogHandler` / `NewLoginLogHandler` but doesn't specify where they live
   - Recommendation: `internal/api/v1/monitor/log_handler.go` with `NewMonitorLogHandler[T any](svc, moduleName)` factory

---

## Environment Availability

Step 2.6: SKIPPED (no external dependencies — pure Go refactoring using existing codebase only)

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` + `github.com/stretchr/testify` |
| Config file | none — standard Go test |
| Quick run command | `go test ./internal/api/v1/operations/... -run TestBuilding -v` |
| Full suite command | `go test ./internal/api/v1/... -count=1` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Command |
|--------|----------|-----------|---------|
| WIRE-01 | Handler helpers unify error response contract | Integration | `go test ./internal/api/v1/operations/... -run TestHandlerWire` |
| WIRE-01 | BusinessError migration to apperrors | Unit | `go test ./internal/services/... -run TestV130R03 -v` |
| HANDLER-01 | 14 handler templates use new helpers | Compile + Integration | `go build ./...` (verify no references to base_handler.go) |
| HANDLER-02 | Monitor handlers deduplicated via generics | Compile + Integration | `go build ./internal/api/v1/monitor/...` |

### Sampling Rate
- **Per task commit:** `go test ./internal/api/v1/operations/... -count=1`
- **Per wave merge:** `go test ./internal/api/v1/... ./internal/services/... -count=1`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/api/v1/monitor/log_handler.go` — generic `MonitorLogHandler[T]`
- [ ] `internal/api/v1/monitor/log_handler_test.go` — tests for generic handler
- [ ] Upgrade 200+ handler test assertions for new `Response.code` values

---

## Security Domain

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V5 Input Validation | yes | gin validator + `HandleJSONBinding` error propagation |
| V4 Access Control | no | Not modified by this phase |

No security-sensitive changes. This is a pure refactoring phase that preserves existing security guarantees.

---

## Sources

### Primary (HIGH confidence)
- `pkg/response/handler_helpers.go` — current helpers with BusinessError special case (line 29-45)
- `pkg/errors/errors.go:78,100` — `NewWithHTTPStatus` / `WrapWithHTTPStatus` signatures
- `pkg/response/business_error.go` — BusinessError type marked for deletion
- `internal/api/v1/operations/base_handler.go` — local helpers marked for deletion
- `internal/services/config_restore_task_service.go:76,93,112` — 3 BusinessError call sites

### Secondary (MEDIUM confidence)
- `internal/api/v1/operations/{building,server_room,wall}_handler.go` — handler template patterns
- `internal/api/v1/monitor/{oper_log,login_log}_handler.go` — monitor handler patterns
- `internal/services/monitor/{oper_log,login_log}_service.go` — service interface definitions

### Tertiary (LOW confidence)
- D-104-6 assumption that Go higher-order functions are better than generics for handler template convergence — this is a reasoned architectural decision based on the diverse service interface signatures across the 14 handlers

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries are existing project dependencies; no new packages
- Architecture: HIGH — patterns derived from locked decisions in 104-CONTEXT.md plus code analysis
- Pitfalls: MEDIUM — some implementation details (WrapWithHTTPStatus nil behavior) required inference

**Research date:** 2026-09-08
**Valid until:** 2026-10-08 (30 days for stable refactoring domain)
