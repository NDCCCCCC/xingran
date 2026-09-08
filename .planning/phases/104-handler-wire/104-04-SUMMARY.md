# Phase 104 Plan 04 Summary: Handler Test Assertions

**Committed:** `aa09fab`

## Changes

### Test Assertion Updates

Wire contract change: `HandleJSONBinding` now uses `apperrors.Wrap(err, apperrors.CodeParamError, err.Error())` → `Response.code = 1001` (not 400). `HandleServiceError` for plain service errors now returns HTTPStatus=500 + code=500 + message="operation失败" (no err detail leaked).

| File | Fixes |
|------|-------|
| `pkg/response/response_test.go` | `BusinessError` → `apperrors.NewWithHTTPStatus`; code assertions updated (400→400001, 409→409001); HTTPStatus quirk fixed |
| `internal/api/v1/operations/building_handler_test.go` | binding message assertion updated ("Address" not "参数错误") |
| `internal/api/v1/network/backup_handler_test.go` | 12 binding code assertions: 400→1001 |
| `internal/api/v1/network/command_execution_handler_test.go` | 5 binding code assertions: 400→1001 |
| `internal/api/v1/network/credential_handler_test.go` | 3 binding code assertions: 400→1001 |
| `internal/api/v1/network/device_handler_test.go` | 6 binding code assertions: 400→1001 |
| `internal/api/v1/network/discovery_handler_test.go` | 4 binding code assertions: 400→1001 |
| `internal/api/v1/network/mac_history_heatmap_handler_test.go` | 9 binding code assertions: 400→1001 |
| `internal/api/v1/network/mac_port_handler_test.go` | 6 binding code assertions: 400→1001 |
| `internal/api/v1/network/template_handler_test.go` | 15 assertions: binding code 400→1001, service error HTTPStatus 400→500, message → generic "Xxx操作失败" |

### Notes
- Monitor handler tests (oper_log_handler_test.go, login_log_handler_test.go) do not assert `resp.Code` — only HTTP status, no changes needed
- `operations/asset_handler_test.go` did not assert `resp.Code` in the affected patterns — no changes needed
- Monitor tests verified green: `./internal/api/v1/monitor/...`
- Operations tests verified green: `./internal/api/v1/operations/...`
- Network tests verified green: `./internal/api/v1/network/...`
