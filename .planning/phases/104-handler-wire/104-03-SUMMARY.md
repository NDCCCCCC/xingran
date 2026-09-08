# Phase 104 Plan 03 Summary: MonitorLogHandler[T] Generic

**Committed:** `1512d15`

## Changes

### New file: `internal/api/v1/monitor/log_handler.go`

Created `MonitorLogHandler[T]` generic struct with shared delete methods:

```go
type MonitorLogHandler[T any] struct {
    svc            logMutator
    moduleName     string
    operTypeDelete operlog.OperType
    operTypeBatch  operlog.OperType
    needOperlog   bool
    core          *core.Core
}
```

Shared methods: `Delete`, `BatchDelete`
Interface: `logMutator` = `Delete` + `BatchDelete` (context.Context variants)

### Refactored: `oper_log_handler.go`

- Embeds `*MonitorLogHandler[models.OperLog]`
- Retains custom methods: `List`, `GetByID`, `Clean`
- Constructor initializes embedded generic with `needOperlog=false`

### Refactored: `login_log_handler.go`

- Embeds `*MonitorLogHandler[models.LoginLog]`
- Retains custom methods: `List`, `GetByID`, `Clean`, `UnlockUser`
- Constructor initializes embedded generic with `needOperlog=true`

### Test fixes
- `oper_log_handler_test.go`: `TestOperLog_WithCore` — use proper constructor
- `login_log_handler_test.go`: `TestLoginLog_WithCore` — use proper constructor

### Dead code removed
- `GetByUsername` — not in router, not in service interface

### Design Notes
- `GetByUsername` was counted as a shared method in the original PLAN but is LoginLog-only and not actually implemented in the service
-泛型 T constraint uses `logMutator` interface (Delete + BatchDelete only)

### Verification
- `go build ./internal/api/v1/monitor/...` ✓
- `go test ./internal/api/v1/monitor/...` ✓ (all pass)
