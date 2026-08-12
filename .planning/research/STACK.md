# Stack Research — v1.19 网络设备写命令 (Network Device Write Operations)

**Project:** XingRan-Next (xingran-go-backend)
**Milestone:** v1.19 — SSH write operations on network devices (Huawei/H3C/锐捷)
**Researched:** 2026-07-06
**Confidence:** HIGH

## Executive Summary

Critical finding: **the SSH write infrastructure is already in place**. This milestone is overwhelmingly a *composition* problem, not a *new infrastructure* problem. The v1.18 read path installed `scrapli/scrapligo v1.3.3` and built `ScrapliWrapper.SendConfig()`, the `ConfigExecutionService` already drives it against `sys_config_execution` and writes `sys_config_execution_detail`, the connection pool already serializes per-device access with ref-counting, the `DeviceInfoCollectionService.Enqueue(deviceID)` interface is already used by v1.18 to trigger re-collection after writes, the operlog helper at `internal/utils/operlog` already has `Record` (普通) and `RecordWithBody` (含敏感字段) variants, and the permission namespace `network:port:query` is already declared in `pkg/permission/config.go` (v1.19 needs to add a sibling `network:port:write`).

What v1.19 must add is a *port-focused write layer* (not template, not free-form command): a `PortWriteService` that takes `{deviceId, portId, action, params}`, looks up the vendor-specific command template, calls `ScrapliWrapper.SendConfig`, persists the result to `sys_config_execution(_detail)` for audit, enqueues a port re-collection via the existing interface, and emits an operlog row. The frontend adds a per-row "shutdown / undo shutdown / description / dot1x enable-disable" action set. A batch mode iterates devices serially with fail-fast semantics.

Recommendation: **no new Go module additions**. All required primitives are already on the v1.18 wire. Only the standard `go.mod` cleanup and an upgrade of `scrapli/scrapligo` to v1.4.0+ (to pick up the `GetPrompt → (string, error)` signature change and the panic-mitigation patch) are recommended.

## Recommended Stack Additions

### Core Technologies (existing — no additions required)

| Technology | Version | Purpose | Why Recommended (current state) |
|------------|---------|---------|---------------------------------|
| `scrapli/scrapligo` | v1.3.3 (upgrade to v1.4.x) | SSH/Telnet transport with config-mode write | ALREADY INSTALLED and used for read (`SendCommand`); `SendConfig` exists at `internal/device/scrapli_wrapper.go:567` and is exercised by `ConfigExecutionService` at `internal/services/config_execution_service.go:277`. v1.4.0 closes the queue-panic window (see `internal/utils/scrapligo-queue-panic-windows.md`) |
| `internal/device.ScrapliWrapper.SendConfig(string) (*Response, error)` | — | Send config-mode string block to device | Already implemented, acquires opMu RLock, handles panic recovery, returns `Response{Result, Started, Finished, Failed}` |
| `internal/services.ConfigExecutionService` | — | Persists execution record + per-device detail | ALREADY WIRED: `ExecuteByTemplate` creates `sys_config_execution` (status 0→1→2/3) and `sys_config_execution_detail` rows with `CommandSent`, `OutputReceived`, `ErrorMessage`, `Duration`. Can be reused for port-write |
| `internal/services.DeviceInfoCollectionService.Enqueue(deviceID string) error` | — | Async re-collection trigger | ALREADY WIRED in v1.18. Dedupes pending/running tasks. v1.19 calls this *after* successful port write to refresh the port_status row |
| `internal/services.DeviceCredentialHelper` | — | Resolves device → credential with default fallback | ALREADY USED by `portcollection/service.go`. Reuse — no need to re-resolve credentials in the write path |
| `internal/utils/operlog` | — | Audit log | ALREADY EXPORTED with `Record(c, svc, db, "模块", OperTypeXxx)` and `RecordWithBody(...)` (auto-masks password/pwd/secret/key/etc) |
| `pkg/permission/config.go` PermissionCode | — | RBAC | ALREADY HAS `NetworkPortQuery = "network:port:query"` and `NetworkCommandExecute = "network:command:execute"`. v1.19 adds `NetworkPortWrite` to this file |
| `internal/services/portcollection` | — | Port data layer (read + write semantics) | ALREADY OWNS the port domain. v1.19's `PortWriteService` lives here or as a sibling service |

### Supporting Libraries (existing — no additions required)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `golang.org/x/sync/errgroup` | v0.19.0 | Parallelism with concurrency limit | Batch port write (parallel) — but MVP locks strategy to **serial + fail-fast** per v1.19 init decisions, so errgroup.SetLimit is only used if a future phase opts into parallel |
| `gorm.io/gorm` | v1.30.5 | ORM | `sys_config_execution` and `sys_config_execution_detail` persistence |
| `sirupsen/logrus` + `natefinch/lumberjack` | v1.9.3 / v2.2.1 | Structured logging | Log SSH session start/end per device, with vendor, port, action, duration |
| `google/uuid` | v1.6.0 | Execution ID generation | Use for `ConfigExecution.ID` and per-port `ConfigExecutionDetail.ID` |
| `swaggo/swag` + `gin-swagger` | v1.16.4 / v1.6.0 | Swagger UI | Auto-doc the new `POST /network/ports/write` and `/batch-write` endpoints |

### Development Tools (existing)

| Tool | Purpose | Notes |
|------|---------|-------|
| `scripts/generate_swagger.sh` | Regenerate Swagger docs | Run after adding the new port-write handler comments |
| `go build ./...` | Build check | MANDATORY before any commit (per `CLAUDE.md`) |
| `go test ./...` | Unit tests | The new `PortWriteService` should have table-driven tests per `command_statistics_test.go` pattern |

## Architecture Integration Points

```
Client (Ant Design Table row action / batch dialog)
    │
    ▼
Gin Router  POST /network/ports/write       (per-row)
            POST /network/ports/batch-write (batch, serial+fail-fast)
    │
    ▼
PortWriteHandler (internal/api/v1/network/port_write_handler.go)   ← NEW
    │
    ├── operlog.RecordWithBody  (with body for action+portId+params; masks nothing here)
    │
    ▼
PortWriteService (internal/services/portcollection/port_write_service.go)   ← NEW
    │
    ├── 1. Resolve device + credential (DeviceCredentialHelper)
    ├── 2. Resolve vendor command template (VendorPortTemplate[vendor][action][port])
    ├── 3. Render command string with params (port name, description text, etc.)
    ├── 4. Create sys_config_execution row (ExecutionTypeCommand, status=Running)
    ├── 5. Get connection from pool, SendConfig (vendor, command)
    ├── 6. Save sys_config_execution_detail row with response.Result
    ├── 7. Update sys_config_execution row to status=Success/Failed
    ├── 8. On success: deviceInfoCollectionService.Enqueue(deviceID)   ← reuse v1.18
    ├── 9. Return result to handler → response.Success/Error
    │
    ▼
Device connection pool  (internal/device/connection_pool.go)
    │
    ▼
ScrapliWrapper.SendConfig  (internal/device/scrapli_wrapper.go:567)
    │
    ▼
Huawei / H3C / 锐捷 SSH session
```

## Vendor Command Template (Hardcoded Map)

Locked decision: "命令模板: 硬编码 vendor→template map（落地为先，后续 phase 抽象为数据库）".

This is a **Go map in a single file**, not a database table. Lives at `internal/services/portcollection/vendor_port_template.go`:

```go
package portcollection

// PortAction 端口写操作类型
type PortAction string

const (
    PortActionShutdown     PortAction = "shutdown"
    PortActionUndoShutdown PortAction = "undo_shutdown"
    PortActionDescription  PortAction = "description"
    PortActionDot1xEnable  PortAction = "dot1x_enable"
    PortActionDot1xDisable PortAction = "dot1x_disable"
)

// PortTemplateParams 模板参数
type PortTemplateParams struct {
    InterfaceName string // GE0/0/1
    Description   string // description 专用
}

// VendorPortTemplate 硬编码 vendor → action → template map
// MVP 范围:Huawei / H3C / 锐捷 (Cisco 后续 phase)
var VendorPortTemplate = map[models.DeviceVendor]map[PortAction]func(PortTemplateParams) string{
    models.VendorHuawei: {
        PortActionShutdown:     func(p PortTemplateParams) string { return fmt.Sprintf("interface %s\nshutdown", p.InterfaceName) },
        PortActionUndoShutdown: func(p PortTemplateParams) string { return fmt.Sprintf("interface %s\nundo shutdown", p.InterfaceName) },
        PortActionDescription:  func(p PortTemplateParams) string { return fmt.Sprintf("interface %s\ndescription %s", p.InterfaceName, p.Description) },
        PortActionDot1xEnable:  func(p PortTemplateParams) string { return fmt.Sprintf("interface %s\ndot1x enable", p.InterfaceName) },
        PortActionDot1xDisable: func(p PortTemplateParams) string { return fmt.Sprintf("interface %s\nundo dot1x enable", p.InterfaceName) },
    },
    models.VendorH3C: {
        // H3C uses nearly identical syntax to Huawei
        PortActionShutdown:     func(p PortTemplateParams) string { return fmt.Sprintf("interface %s\nshutdown", p.InterfaceName) },
        PortActionUndoShutdown: func(p PortTemplateParams) string { return fmt.Sprintf("interface %s\nundo shutdown", p.InterfaceName) },
        PortActionDescription:  func(p PortTemplateParams) string { return fmt.Sprintf("interface %s\ndescription %s", p.InterfaceName, p.Description) },
        PortActionDot1xEnable:  func(p PortTemplateParams) string { return fmt.Sprintf("interface %s\ndot1x enable", p.InterfaceName) },
        PortActionDot1xDisable: func(p PortTemplateParams) string { return fmt.Sprintf("interface %s\nundo dot1x enable", p.InterfaceName) },
    },
    models.VendorRuijie: {
        // Ruijie uses Cisco-style: no "undo" prefix for negate
        PortActionShutdown:     func(p PortTemplateParams) string { return fmt.Sprintf("interface %s\nshutdown", p.InterfaceName) },
        PortActionUndoShutdown: func(p PortTemplateParams) string { return fmt.Sprintf("interface %s\nno shutdown", p.InterfaceName) },
        PortActionDescription:  func(p PortTemplateParams) string { return fmt.Sprintf("interface %s\ndescription %s", p.InterfaceName, p.Description) },
        PortActionDot1xEnable:  func(p PortTemplateParams) string { return fmt.Sprintf("interface %s\ndot1x port-control auto", p.InterfaceName) },
        PortActionDot1xDisable: func(p PortTemplateParams) string { return fmt.Sprintf("interface %s\nno dot1x port-control", p.InterfaceName) },
    },
}
```

NOTE: The user (locked v1.19 decision) prefers Ruijie's "no shutdown" syntax. For dot1x, Ruijie S2960 uses `dot1x port-control auto` / `no dot1x port-control`. This must be confirmed against the real device templates that v1.18 used (see `internal/services/component_collector/cli_ruijie_collector.go`) and Phase 49's Ruijie chassis fixes.

## Concrete Service Shape (PortWriteService)

```go
// internal/services/portcollection/port_write_service.go (NEW)
package portcollection

type PortWriteRequest struct {
    DeviceID      string
    PortID        string  // sys_device_port_status.id (UUID)
    InterfaceName string  // GE0/0/1 — passed alongside PortID for vendor template
    Action        PortAction
    Description   string  // required iff Action == PortActionDescription
    Timeout       int     // seconds; default 30, max 120
}

type PortWriteResult struct {
    ExecutionID    string
    DeviceID       string
    PortID         string
    Action         PortAction
    Status         models.ExecutionStatus
    CommandSent    string
    OutputReceived string
    ErrorMessage   string
    Duration       int
    EnqueuedReCollect bool
}

type PortWriteService struct {
    db                          *gorm.DB
    deviceExecutor              *device.DeviceExecutor
    deviceInfoCollectionService *services.DeviceInfoCollectionService
    credentialHelper            *services.DeviceCredentialHelper
}

func (s *PortWriteService) WritePort(ctx context.Context, req *PortWriteRequest) (*PortWriteResult, error) {
    // 1. Validate action + required params
    // 2. Load device, load credential (DeviceCredentialHelper)
    // 3. Render vendor template command string
    // 4. Create sys_config_execution + sys_config_execution_detail rows
    // 5. Get connection from pool; wrapper.SendConfig
    // 6. Update rows with response
    // 7. On success: Enqueue(deviceID) — async re-collect
    // 8. Return result
}

func (s *PortWriteService) BatchWritePorts(ctx context.Context, reqs []PortWriteRequest) ([]*PortWriteResult, error) {
    // Serial + fail-fast: iterate, on first failure return immediately
    // (locked decision: "批量执行：串行失败即停")
}
```

## Handler Shape (PortWriteHandler)

```go
// internal/api/v1/network/port_write_handler.go (NEW)
package network

type PortWriteHandler struct {
    core *core.Core
}

func (h *PortWriteHandler) Write(c *gin.Context) {
    // bind PortWriteRequest
    // service.WritePort
    // operlog.Record(c, ..., "端口管理", operlog.OperTypeUpdate)  // shutdown/undo → OperTypeStatus
    // response.Success / Error
}

func (h *PortWriteHandler) BatchWrite(c *gin.Context) {
    // bind {ports: [PortWriteRequest]}
    // service.BatchWritePorts
    // operlog.Record(c, ..., "端口管理", operlog.OperTypeBatch)
}
```

OperType mapping per action:

| Action | OperType |
|--------|----------|
| shutdown / undo_shutdown | `OperTypeStatus` (启用/停用) |
| description | `OperTypeUpdate` (修改) |
| dot1x_enable / dot1x_disable | `OperTypeStatus` |
| batch | `OperTypeBatch` |

`Record` is used (NOT `RecordWithBody`) because the request body contains no password/key/secret — only `deviceId`, `portId`, `action`, `description`. The SSH credentials used for the device are stored server-side in `sys_auth_credential` and never appear in the request.

## Router Registration

Append to `internal/api/v1/network/network_router.go` (line 211, after `SetupPortRouter`):

```go
// ==================== 端口写命令路由（v1.19 新增）====================
portWrites := r.Group("/ports")
portWrites.Use(middleware.RequirePermissions([]string{
    "network:port:write",  // 新增权限常量
    "network:port:query",  // 读权限（兼容读后写）
}, core))
{
    portWrites.POST("/write", portWriteHandler.Write)
    portWrites.POST("/batch-write", portWriteHandler.BatchWrite)
}
```

NOTE: This creates a route overlap on `/ports/*`. The existing `/ports/list` group uses `RequirePermissionsWithQuery` to also accept ops read perms. The new write group uses `RequirePermissions` (strict). Gin resolves POST `/ports/write` to the new group; POST `/ports/list` to the existing one. No conflict because the path patterns are different. **Verify with `go build ./...` after wiring**.

## Permission Constant Addition

In `pkg/permission/config.go`, after line 186 (`NetworkPortQuery`):

```go
// 端口写命令权限 (v1.19 新增)
NetworkPortWrite PermissionCode = "network:port:write"
```

In the `GetRoutePermissions()` table (around line 360+), add the two new routes mapping to `NetworkPortWrite`.

## Frontend Stack (existing — no additions)

| Library | Version | Purpose | Use |
|---------|---------|---------|-----|
| `antd` Table actions | 6.1 | Per-row action dropdown | Add "端口操作" menu: 启用/停用/修改描述/dot1x 启用-停用 |
| `antd` Modal | 6.1 | Single-port dialog | Form with `description` input + submit |
| `antd` Modal | 6.1 | Batch dialog | Multi-select ports + action picker |
| `@/lib/api` `post` | — | Wrapped POST | Use `post('/network/ports/write', req)` and `post('/network/ports/batch-write', {ports: [...]})` |
| React Query mutation | 5.90.12 | Optimistic update + rollback | `useMutation` with `onSuccess` refetch port list |

## Observability Patterns (existing — extend, don't add)

| Pattern | Location | v1.19 Application |
|---------|----------|-------------------|
| Logrus + Lumberjack | `pkg/logger` (via `applogger`) | Log each write: `applogger.Infof("[port-write] device=%s port=%s action=%s status=%d duration=%dms", ...)` |
| Logrus Debug-level command echo | `internal/device/executor.go:200,203,223,236,244,248,254` | Already logs raw command + response; reuse |
| Operlog (audit table) | `internal/utils/operlog` | Mandatory: every successful write must emit a row before `response.Success` |
| Connection pool ref-counting | `internal/device/connection_pool.go:266-273` (F-14 Phase 31 fix) | Use `defer conn.ReleaseRef()` after `SendConfig` — matches the exact pattern at `config_execution_service.go:274` |
| Scrapli opMu RLock | `internal/device/scrapli_wrapper.go:567` | `SendConfig` already takes opMu RLock; nothing extra needed |
| GetPrompt serialization | `internal/device/scrapli_wrapper.go:277` | Defense-in-depth; v1.19 inherits it automatically because SendConfig is gated by acquireOp |

## Concurrent-Write Concerns (analysis of the existing pool)

`internal/device/connection_pool.go` and `connection_pool_test.go` (F-14 fix, 2026-07-06 复查):

1. **Per-device serialization**: `GetConnection(ctx, deviceID)` returns a `PooledConnection` whose internal `opMu RLock` on the wrapper serializes all `SendConfig`/`SendCommand` calls for that single device. Two parallel writes to the same device are safe but will execute serially in scrapli.

2. **Ref-counting**: `GetConnection` already does `refCount +1`; the v1.18 fix mandates `defer conn.ReleaseRef()` (NOT `Acquire+Release`, which would double-count). The template execution path at `config_execution_service.go:262-274` shows the canonical pattern.

3. **TOCTOU state check**: `acquireOp()` at `scrapli_wrapper.go:245-264` re-checks `getState() == StateReady` after acquiring the RLock. This is the only protection against a connection closing mid-write. v1.19 inherits this.

4. **Batch write strategy**: v1.19's locked "serial + fail-fast" decision is conservative and **sidesteps** several concurrent concerns:
   - No errgroup.SetLimit for the batch
   - No parallel device writes
   - No risk of multiple devices hitting their own `deviceID`'s pool concurrently (which would be fine anyway because pools are per-device)
   - Tradeoff: 50 ports × 1 device = slower than parallel but predictable

5. **Serial fail-fast semantics**: When a `BatchWritePorts` call hits a failed device, the service must:
   - Mark the failed port's detail row as `ExecutionStatusFailed`
   - Update the parent `sys_config_execution` row to `ExecutionStatusFailed` (overwriting partial-success counts)
   - Return the partial results to the caller; the handler surfaces them in the response
   - The operlog is still emitted (with the partial result data)

## Sensitive Data Flow (operlog masking)

The write path carries no sensitive data in the request body:
- `deviceId` — UUID
- `portId` — UUID
- `interfaceName` — string like `GE0/0/1`
- `action` — enum
- `description` — user-supplied free text (NOT a secret)

Therefore `operlog.Record(...)` is correct. `RecordWithBody` would be the right choice if the action were something like "rotate device SSH key" or "change credential", but port configuration actions are not sensitive-keyword-matching.

The device's SSH password is decrypted at runtime inside the connection pool (F-14 layer), used by scrapli, and never appears in the request body or operlog.

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| Extend `ConfigExecutionService` with port-specific methods | Build a separate `PortWriteService` | Decision: **separate service** because (a) port write has a vendor-template layer the template service doesn't model, (b) port write must enqueue re-collection, which `ConfigExecutionService` doesn't know about, (c) audit granularity is per-port not per-device |
| Hardcoded Go map `VendorPortTemplate` | Persist to `sys_command_template` table (XingRan standard pattern) | Decision: **hardcoded first** (v1.19 locked). Database-driven templates can come in a later phase after the MVP is UAT-validated. The `sys_config_template` table is *template-based init/config*, not port-action enums |
| Reuse `network:command:execute` permission | Add new `network:port:write` permission | Decision: **new permission** because (a) port write is a higher-risk surface than free-form command, (b) operators who can run `display` should not necessarily be able to `shutdown` a port, (c) audit semantics are different (Status/Update vs Other) |
| `SendConfig` (single multi-line string) | `SendConfigs` ([]string, one per command) | Decision: **`SendConfig`** — the vendor templates produce a single multi-line block per port (`interface X\nshutdown`). `SendConfigs` is for "one config per device" scenarios |
| `defer conn.ReleaseRef()` | `defer conn.Release()` | Decision: **`ReleaseRef`** (F-14 fixed; calling `Release` after `GetConnection` would double-decrement and break pool accounting) |
| `OperTypeStatus` for shutdown/undo | `OperTypeEnable` / `OperTypeDisable` | Decision: **`OperTypeStatus`** (10) — semantically "状态变更（启用/停用）" matches the audit log convention; `Enable`/`Disable` are also valid but `Status` is the more general bucket covering both directions |
| Serial batch (fail-fast) | Parallel batch with per-device workers | Decision: **serial** (v1.19 locked). Config drift risk: if 5 of 10 ports fail in parallel, recovery is harder. Serial is the conservative default for first UAT |

## What NOT to Add

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| New SSH library (golang.org/x/crypto/ssh, survey, wish, etc.) | Scrapli is already the abstraction; introducing a second SSH client doubles the credential-decryption surface and breaks the panic-recovery patterns at `scrapli_wrapper.go:516-521,572-578` | Reuse `ScrapliWrapper.SendConfig` |
| New connection pool | The existing pool is F-14 fixed, ref-counted, and used by the template execution path; introducing a parallel pool would split the connection lifecycle | Reuse `pool.GetConnection(ctx, deviceID)` + `defer conn.ReleaseRef()` |
| New config template model | `sys_config_template` exists but is designed for full-device init/config, not port-action enums; creating a new table for the 5 port actions is over-engineering for MVP | Hardcoded `VendorPortTemplate` map |
| Netconf/YANG | The 3 target vendors (Huawei/H3C/锐捷) all support SSH CLI write; Netconf adds parser + cert complexity disproportionate to MVP scope | SSH CLI write (`SendConfig`) |
| `expect`-style interactive prompting | All 5 actions are non-interactive (no `[Y/N]` confirmations on `shutdown`/`description`/`dot1x`) | Plain multi-line config block |
| Async batch (background job) | v1.19 locked "MVP 仅'失败即停 + 回传失败点'"; async complicates the response contract (caller must poll `sys_config_execution` for status) | Synchronous batch with per-port detail rows |
| Auto-rollback on partial failure | v1.19 explicitly out of scope; auto-rollback requires per-command undo planning, vendor-state assumptions, and a "what if rollback also fails" failure mode | Fail-fast + return partial result; manual rollback by operator |
| Free-form CLI command in port write request | Would let any caller run `reboot` or `delete flash:`; massively increases audit blast radius and turns the endpoint into a backdoor around `network:command:execute` | Enum-gated `PortAction` with hardcoded templates |

## Stack Patterns by Variant

**If port = trunk port (no shutdown allowed for protection):**
- SendConfig still proceeds but response.Result will contain Huawei's `Error: This interface cannot be shut down.` The service captures the output in `sys_config_execution_detail.output_received` and the operlog row reflects the failure. The handler returns `ExecutionStatusFailed` with the captured error message.

**If device is offline (status=1):**
- `GetConnection` will fail at TCP dial timeout (10s). Service returns the error, marks the row failed, no operlog success emission.

**If user has `network:port:query` but not `network:port:write`:**
- The new middleware rejects with 403. The row action menu on frontend should be hidden via `useAuthz('network:port:write')`.

**If batch is interrupted (handler context cancel):**
- The remaining devices are skipped (fail-fast). The already-written per-device detail rows are preserved. The parent `sys_config_execution` row is marked failed with the cancel reason.

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| `scrapli/scrapligo` v1.3.3 (current) | `go1.24` toolchain go1.24.5 | Build clean per `CLAUDE.md` |
| `scrapli/scrapligo` v1.4.x (recommended upgrade) | `go1.24` | v1.4.0+ returns `GetPrompt() (string, error)` — code at `scrapli_wrapper.go:279` already takes the v1.4 signature. **Confirm by reading `go.sum` and changelog before upgrading** |
| `golang.org/x/sync/errgroup` v0.19.0 | `go1.24` | OK; only used if a future phase opts into parallel batch |
| `gorm.io/gorm` v1.30.5 | `go1.24` | OK; auto-migration handles new rows |
| `antd 6.1` (frontend) | React 19.2 | Table actions, Modal, Form already in use across the ops module |

## Sources

- `internal/device/scrapli_wrapper.go` (verified) — `SendConfig`, `acquireOp`, `opMu` lifecycle, `PlatformName` vendor mapping
- `internal/services/config_execution_service.go` (verified) — canonical pattern: `pool.GetConnection` → `defer conn.ReleaseRef` → `wrapper.SendConfig` → persist detail + execution rows
- `internal/services/command_dispatch_service.go` (verified) — `ExecutionTypeCommand` + `ExecutionStrategy`/`Concurrency` fields
- `internal/services/device_info_collection_service.go:133` (verified) — `Enqueue(deviceID string) error` async trigger
- `internal/services/device_credential_helper.go:24-40` (verified) — device → credential resolver with default fallback
- `internal/models/config_execution.go` (verified) — `ConfigExecution` + `ConfigExecutionDetail` table names `sys_config_execution` / `sys_config_execution_detail`
- `internal/utils/operlog/operlog.go` (verified via `CLAUDE.md`) — `Record` and `RecordWithBody` helpers
- `pkg/permission/config.go:147-198` (verified) — `NetworkCommandExecute`, `NetworkPortQuery` existing; `NetworkPortWrite` to be added
- `internal/api/v1/network/network_router.go:206-214` (verified) — `/ports` group with `RequirePermissionsWithQuery`; v1.19's `/ports/write` + `/ports/batch-write` route pair
- `internal/api/v1/network/port_handler.go:96-111` (verified) — `PortCollectionRequest` + `operlog.Record` shape for the read-side; v1.19's write handler mirrors it
- `.planning/PROJECT.md` (verified) — v1.19 init decisions: device_id strategy, hardcoded templates, serial fail-fast, vendor scope
- `CLAUDE.md` (verified) — operlog convention with 25 OperType constants; middleware chain; status convention 0=normal/1=stopped

## Key Takeaway

**Zero new Go module dependencies** are required for v1.19. The SSH write path is composable from existing primitives: `ScrapliWrapper.SendConfig` + `ConfigExecutionService` persistence shape + `DeviceInfoCollectionService.Enqueue` + `operlog.Record` + a hardcoded vendor→template map. The single optional upgrade is `scrapli/scrapligo` v1.3.3 → v1.4.x to pick up the queue-panic mitigation, which is a low-risk version bump and can be deferred to a separate commit.

The work is concentrated in **three new files**:
1. `internal/services/portcollection/port_write_service.go` — service layer
2. `internal/services/portcollection/vendor_port_template.go` — hardcoded vendor map
3. `internal/api/v1/network/port_write_handler.go` — HTTP handler

Plus edits to:
- `internal/api/v1/network/network_router.go` (route registration)
- `pkg/permission/config.go` (add `NetworkPortWrite` constant + route map)
- `internal/api/v1/network/port_router.go` (call `portWriteHandler` if split)
- Frontend `src/pages/network/ports/` (Table actions + 2 dialogs)

This is a thin, high-leverage milestone. Estimated scope: 1 phase, 2-3 plans, 3 waves.
