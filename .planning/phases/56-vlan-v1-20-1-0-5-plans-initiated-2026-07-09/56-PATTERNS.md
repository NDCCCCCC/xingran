# Phase 56: 网络设备 VLAN + 端口绑定 (v1.20.1) - Pattern Map

**Mapped:** 2026-07-09
**Files analyzed:** 14 (10 backend Go, 4 frontend TS/TSX, plus 1 testdata dir)
**Analogs found:** 14 / 14 (100% — v1.19 is structurally complete, Phase 56 is purely additive extension)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `internal/services/portcollection/vendor_port_template.go` (MODIFY) | service / template | table-driven render | same file (self-extension) | exact — add 2 actions × 3 vendors to existing `vendorPortTemplate` map |
| `internal/services/portcollection/vendor_port_template_test.go` (MODIFY) | test | table-driven | same file (self-extension) | exact — append 12 test rows to `TestRenderCommand_VendorActionMatrix` table |
| `internal/services/portwrite/pre_state_check.go` (MODIFY) | service | request-response | same file (self-extension) | exact — add 2 cases to `checkPreState` switch |
| `internal/services/portwrite/port_write_service.go` (MODIFY) | service | request-response | same file (self-extension) | exact — add `SetAccessVlan` / `PortBinding` methods + extend `PortWriteService` interface + extend `writeAndRefresh` signature |
| `internal/services/portwrite/port_write_service_test.go` (MODIFY) | test | unit | (create) | exact — mirror v1.19 unit test pattern |
| `internal/services/portwrite/port_write_e2e_test.go` (MODIFY) | test | e2e (scrapligo FileTransport) | same file (self-extension) | exact — append 10 e2e tests |
| `internal/services/portwrite/testdata/*.fixture` (CREATE 6+) | testdata | (fixture files) | n/a — Phase 56 creates the dir | n/a |
| `internal/api/v1/network/port_write_handler.go` (MODIFY) | handler | request-response | same file (`execSinglePort` DRY) | exact — add `SetAccessVlan` / `PortBinding` handlers calling existing `execSinglePort`; extend `buildAfterValue` switch |
| `internal/api/v1/network/port_write_router.go` (MODIFY) | router | request-response | same file | exact — append 2 kebab routes to existing `/write` group |
| `pkg/permission/config.go` (MODIFY) | config | n/a (registry) | same file (lines 268-273) | exact — append 2 permission registry rows |
| `xingran-react-frontend/src/types/network.ts` (MODIFY) | type | n/a | same file (PortWriteAction union) | exact — add 2 union members + 2 request interfaces |
| `xingran-react-frontend/src/lib/api/networkApi.ts` (MODIFY) | lib / api | request-response | same file (`writeShutdown` pattern) | exact — add 2 kebab-aligned wrappers |
| `xingran-react-frontend/src/components/network/port-write/constants.ts` (MODIFY) | component / constant | n/a | same file (`ACTION_TITLE` Record) | exact — extend Record by 2 keys + add 2 regex constants + BIND_OPS tuple |
| `xingran-react-frontend/src/components/network/port-write/SetAccessVlanModal.tsx` (CREATE) | component | request-response | `PortWriteModal.tsx` (full file) | exact — copy modal skeleton, replace description field with `vlanId` InputNumber |
| `xingran-react-frontend/src/components/network/port-write/PortBindingModal.tsx` (CREATE) | component | request-response | `PortWriteModal.tsx` (full file) | exact — copy modal skeleton, add op Radio.Group + ipInput + macInput |
| `xingran-react-frontend/src/pages/network/ports/index.tsx` (MODIFY) | page | request-response | same file (lines 343-352 ActionButtons array) | exact — append 2 menu items + 2 Modal mount points + 2 state vars |

## Pattern Assignments

### `internal/services/portcollection/vendor_port_template.go` (service / template, table-driven)

**Analog:** same file (self-extension). Pattern is to add new `PortAction` constants + new `PortTemplateParams` fields + new render closures in the existing `vendorPortTemplate` map.

**Action constants pattern** (lines 13-19 — add 2):
```go
const (
    ActionShutdown     PortAction = "shutdown"
    ActionUndoShutdown PortAction = "undo_shutdown"
    ActionDescription  PortAction = "description"
    ActionDot1xEnable  PortAction = "dot1x_enable"
    ActionDot1xDisable PortAction = "dot1x_disable"
)
```

**PortTemplateParams struct pattern** (lines 28-31 — extend with VLAN ID + binding tuple):
```go
type PortTemplateParams struct {
    InterfaceName string
    Description   string
    // v1.20.1 additions:
    VLANID     int    // ActionSetAccessVLAN only
    BindOp     string // "add"|"remove" — ActionPortBinding only
    IPAddress  string // ActionPortBinding only
    MACAddress string // ActionPortBinding only
}
```

**vendorPortTemplate map pattern** (lines 47-72 — add 2 entries × 3 vendors):
```go
var vendorPortTemplate = map[models.DeviceVendor]map[PortAction]func(PortTemplateParams) ([]string, error){
    models.VendorHuawei: {
        ActionShutdown:     func(p PortTemplateParams) ([]string, error) { return wrapInterface(p, "shutdown") },
        ActionUndoShutdown: func(p PortTemplateParams) ([]string, error) { return wrapInterface(p, "undo shutdown") },
        ActionDescription:  renderH3CDescription,
        ActionDot1xEnable:  func(p PortTemplateParams) ([]string, error) { return wrapInterface(p, "authentication-profile dot1x") },
        ActionDot1xDisable: func(p PortTemplateParams) ([]string, error) { return wrapInterface(p, "undo authentication-profile dot1x") },
        // v1.20.1: ActionSetAccessVLAN, ActionPortBinding
    },
    // ...
}
```

**Render function patterns to copy** — `renderH3CDescription` (lines 128-139) and `renderRuijieDot1xEnable` (lines 156-161) are the closest analogs. Both are vendor-specific closures that:
1. Validate params (return `ErrXxx` sentinel)
2. Return `[]string{ "interface <iface>", "<action-cmd>" }`

```go
// From vendor_port_template.go:128-139 (renderH3CDescription)
func renderH3CDescription(p PortTemplateParams) ([]string, error) {
    if p.Description == "" {
        return nil, ErrDescriptionEmpty
    }
    if len(p.Description) > 80 {
        return nil, fmt.Errorf("%w: %d > 80", ErrDescriptionTooLong, len(p.Description))
    }
    return []string{
        fmt.Sprintf("interface %s", p.InterfaceName),
        fmt.Sprintf("description %s", p.Description),
    }, nil
}
```

**Huawei `port link-type access` prefix** — RISK-03: always emit 3-step sequence `interface X | port link-type access | port default vlan Y` (universal prerequisite per S5700/S5735 community validation).

**H3C `user-bind` keyword** — RISK-01 resolved: H3C uses `user-bind ip-address X [mac-address Y]` (NO `static` keyword), per H3C Comware V7 manual. Huawei uses `user-bind static ip-address X [mac-address Y]`.

**Ruijie `switchport port-security binding` syntax** — RISK-02 resolved: full syntax `<mac-address> [vlan <vlan-id>] <ip-address>` per Ruijie port-security manual.

**MAC format conversion per vendor** — Pitfall 7:
- Huawei: `AA-BB-CC-DD-EE-FF` (replace `:` with `-`)
- H3C: same as Huawei
- Ruijie: `AABB.CCDD.EEFF` (replace `:` with `.`, strip dashes)
- Use `NormalizeMACAddress` from `internal/services/mac_normalize.go:32-56` to get canonical `AA:BB:CC:DD:EE:FF` first.

**wrapInterface helper** (lines 81-83) — used for single-line commands after `interface X`. For multi-line actions (Huawei `port link-type access | port default vlan Y`) build the slice directly rather than calling wrapInterface.

---

### `internal/services/portcollection/vendor_port_template_test.go` (test, table-driven)

**Analog:** same file. Pattern is `TestRenderCommand_VendorActionMatrix` table with `{name, vendor, action, params, expected}` rows.

**Test row pattern** (lines 22-56 — copy this structure):
```go
{
    name:     "huawei_description",
    vendor:   models.VendorHuawei,
    action:   ActionDescription,
    params:   PortTemplateParams{InterfaceName: "GE0/0/1", Description: "uplink"},
    expected: []string{"interface GE0/0/1", "description uplink"},
},
```

**Subtest naming convention** — `{vendor}_{action}` (snake_case), e.g. `huawei_set_access_vlan`, `h3c_port_binding_add`, `ruijie_port_binding_remove`.

**Phase 56 additions** — 12 test cases:
- `huawei_set_access_vlan`, `huawei_set_access_vlan_with_linktype` (RISK-03 prefix)
- `h3c_set_access_vlan`
- `ruijie_set_access_vlan` (switchport mode access + switchport access vlan)
- `huawei_port_binding_add`, `huawei_port_binding_add_with_mac`, `huawei_port_binding_remove`
- `h3c_port_binding_add` (user-bind, no static), `h3c_port_binding_remove`
- `ruijie_port_binding_add` (with MAC), `ruijie_port_binding_add_no_mac`, `ruijie_port_binding_remove`

---

### `internal/services/portwrite/pre_state_check.go` (service, request-response)

**Analog:** same file (lines 22-75). Pattern is `switch action` block returning `*PortResult` with `NoOp: true` + `Status: "skipped"` + descriptive `CurrentState`.

**Pre-state check pattern** (lines 64-73 — ActionDescription case):
```go
case portcollection.ActionDescription:
    if port.Description == desc {
        return &PortResult{
            PortID:       port.ID,
            Action:       action,
            Status:       "skipped",
            NoOp:         true,
            CurrentState: "description_match",
        }
    }
```

**Phase 56 additions** — extend switch with:
```go
case portcollection.ActionSetAccessVLAN:
    // Reuse port.VLAN field (DB-cached by portcollection cron)
    if port.VLAN != nil && *port.VLAN == vlanId {
        return &PortResult{
            PortID:       port.ID,
            Action:       action,
            Status:       "skipped",
            NoOp:         true,
            CurrentState: "vlan_match",
        }
    }
case portcollection.ActionPortBinding:
    // Per Pitfall 6 / Open Question #3: skip pre-state for port_binding in MVP.
    // Binding table has no DB model; SSH read adds 3-5s latency. Re-evaluate in v1.20.x.
    return nil
```

**Signature change required** — current signature `(port, action, desc)`. Extend to `(port, action, desc, vlanId int, bindOp, ipAddr, macAddr string)` to accommodate new action params. All 5 existing call sites (which pass `""` for new params) are updated in lockstep.

**Verify `DevicePortStatus.VLAN` field exists** — check `internal/models/device_port_status.go` (or similar). The TS type already has it (`types/network.ts:125 vlan?: number`); backend GORM model should have matching column. If not, add to model + migration.

---

### `internal/services/portwrite/port_write_service.go` (service, request-response)

**Analog:** same file. Pattern is: (1) extend `PortWriteService` interface, (2) add new method that calls `writeAndRefresh` with new action + payload.

**Interface pattern** (lines 88-96 — extend with 2 methods):
```go
type PortWriteService interface {
    Shutdown(ctx context.Context, portID string, operator string) (*PortResult, error)
    UndoShutdown(ctx context.Context, portID string, operator string) (*PortResult, error)
    SetDescription(ctx context.Context, portID string, desc string, operator string) (*PortResult, error)
    EnableDot1x(ctx context.Context, portID string, operator string) (*PortResult, error)
    DisableDot1x(ctx context.Context, portID string, operator string) (*PortResult, error)
    BatchWritePorts(ctx context.Context, req BatchWriteRequest, operator string) (*BatchResult, error)
    // v1.20.1 additions:
    SetAccessVlan(ctx context.Context, portID string, vlanId int, operator string) (*PortResult, error)
    PortBinding(ctx context.Context, portID string, op string, ipAddress string, macAddress string, operator string) (*PortResult, error)
}
```

**Method pattern** (lines 124-127 — Shutdown wraps `writeAndRefresh`):
```go
func (s *portWriteServiceImpl) Shutdown(ctx context.Context, portID string, operator string) (*PortResult, error) {
    return s.writeAndRefresh(ctx, portID, portcollection.ActionShutdown, "", operator)
}
```

**Phase 56 new methods**:
```go
func (s *portWriteServiceImpl) SetAccessVlan(ctx context.Context, portID string, vlanId int, operator string) (*PortResult, error) {
    if vlanId < 1 || vlanId > 4094 {
        return nil, fmt.Errorf("%w: %d (must be 1-4094)", ErrVlanIdOutOfRange, vlanId)
    }
    return s.writeAndRefresh(ctx, portID, portcollection.ActionSetAccessVLAN, "", operator)
}

func (s *portWriteServiceImpl) PortBinding(ctx context.Context, portID string, op string, ipAddress string, macAddress string, operator string) (*PortResult, error) {
    if op != "add" && op != "remove" {
        return nil, fmt.Errorf("%w: %q (must be add|remove)", ErrBindOpInvalid, op)
    }
    if !ipv4Pattern.MatchString(ipAddress) {
        return nil, fmt.Errorf("%w: %q", ErrIPAddressInvalid, ipAddress)
    }
    if macAddress != "" {
        normalized := maccollection.NormalizeMACAddress(macAddress)
        if normalized == "" || normalized == "00:00:00:00:00:00" {
            return nil, fmt.Errorf("%w: %q", ErrMACAddressInvalid, macAddress)
        }
    }
    return s.writeAndRefresh(ctx, portID, portcollection.ActionPortBinding, "", operator)
}
```

**Sentinel errors pattern** (lines 20-26 — append to existing `var ( ... )` block):
```go
var (
    ErrBatchTooLarge = errors.New("portwrite: batch exceeds max size of 50")
    ErrEmptyBatch    = errors.New("portwrite: batch is empty")
    ErrMixedDevices  = errors.New("portwrite: batch contains ports from different devices")
    ErrPortNotFound  = errors.New("portwrite: port not found")
    ErrDeviceNotFound = errors.New("portwrite: device not found")
)
```

**writeAndRefresh signature change** — current `(ctx, portID, action, desc, operator)`. Per Open Question in RESEARCH.md §Code Examples #1, recommendation is **Option (a) `payload any`** for the new params. But minimum-impact: pass empty `""` desc + carry VLAN/binding params through service method closure (same way `operator` is passed unused in v1.19). The `desc` field can carry the vlanId as string OR add a 6th param. **Pragmatic choice**: keep 5-param signature, store VLAN/binding in `PortTemplateParams` extension and pass through `executeWrite`.

**executeWrite param flow** (lines 224-242) — calls `portcollection.RenderCommand(dev.Vendor, action, portcollection.PortTemplateParams{InterfaceName: ..., Description: desc})`. Extend `PortTemplateParams` to carry VLANID/BindOp/IPAddress/MACAddress; pass them through to `RenderCommand`.

**Refresh pattern** (lines 365-378) — `refreshPortStatus(ctx, deviceID)` is automatic for any new action because `writeAndRefresh` calls it on success. No change needed for v1.20.1.

**Refresh backstop** — fire-and-forget goroutine + 30s detached context + log-only failure. Same as v1.19.

---

### `internal/services/portwrite/port_write_service_test.go` (test, unit)

**Analog:** v1.19 unit tests use `testify/assert` + `testify/mock` for the `portWriteExecutor` / `portWritePortCollectionSvc` interfaces. Tests assert on returned `PortResult.Status` / `Error` fields.

**Mock pattern** — implement `portWriteExecutor` and `portWritePortCollectionSvc` interfaces with no-op stubs. Set up expected `SendConfigs` invocations via mock.

**Validator unit tests** (NEW, 5 cases):
- `TestSetAccessVlan_Validation`: vlanId=0 → reject; vlanId=4095 → reject; vlanId=1 → accept; vlanId=4094 → accept; vlanId=4095 → reject
- `TestPortBinding_Validation`: op="add" → accept; op="remove" → accept; op="invalid" → reject; ip="10.62.25.5" → accept; ip="0.0.0.0" → reject (IPv4 regex blocks); mac="" → accept (optional); mac="00:00:00:00:00:00" → reject; mac="AA:BB:CC:DD:EE:FF" → accept

---

### `internal/services/portwrite/port_write_e2e_test.go` (test, e2e via scrapligo FileTransport)

**Analog:** same file (Phase 53 v1.19 e2e). Pattern is `TestE2E_X_VendorY_VariantZ` naming + scrapligo FileTransport for replaying fixture files from `testdata/`.

**E2E test pattern** — uses `device.DeviceExecutor` with FileTransport pointing to fixture file, asserts `PortResult.Status == "succeeded"` / `"failed"` / `"skipped"` per fixture output.

**Phase 56 additions** — 10 e2e tests:
- `TestE2E_SetAccessVlan_Huawei_Success`
- `TestE2E_SetAccessVlan_Huawei_NoOp` (pre-state match)
- `TestE2E_SetAccessVlan_Ruijie_Success`
- `TestE2E_PortBinding_Huawei_Add_Success`
- `TestE2E_PortBinding_Huawei_Add_WithMac_Success`
- `TestE2E_PortBinding_Huawei_Remove_Success`
- `TestE2E_PortBinding_Huawei_Remove_NoOp`
- `TestE2E_PortBinding_Ruijie_Add_Success`
- `TestE2E_Batch_SetAccessVlan` (mixed success/fail)
- `TestE2E_Batch_PortBinding_Add`

**Fixtures needed (6+ files in `testdata/`)** — scrapligo FileTransport format mimicking real device output for the commands in the templates. Naming: `huawei_set_access_vlan_success.fixture`, etc.

---

### `internal/api/v1/network/port_write_handler.go` (handler, request-response)

**Analog:** same file. **CRITICAL: Reuse `execSinglePort` DRY** — DO NOT bypass. The 8-step flow (lines 109-198) is locked for D-04 / D-13 / D-14 invariant guards.

**Handler pattern** (lines 67-73 — Shutdown):
```go
func (h *PortWriteHandler) Shutdown(c *gin.Context) {
    h.execSinglePort(c, portcollection.ActionShutdown, operlog.OperTypeDisable,
        func(ctx context.Context, portID, operator, desc string) (*portwrite.PortResult, error) {
            return h.service.Shutdown(ctx, portID, operator)
        })
}
```

**Phase 56 new handlers** (append after line 105, before `execSinglePort`):
```go
func (h *PortWriteHandler) SetAccessVlan(c *gin.Context) {
    h.execSinglePort(c, portcollection.ActionSetAccessVLAN, operlog.OperTypeUpdate,
        func(ctx context.Context, portID, operator, desc string) (*portwrite.PortResult, error) {
            // Body: { portId, vlanId, reason }
            var req struct {
                PortID string `json:"portId" binding:"required"`
                VLANID int    `json:"vlanId" binding:"required,min=1,max=4094"`
                Reason string `json:"reason,omitempty"`
            }
            if err := c.ShouldBindJSON(&req); err != nil {
                response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
                return nil, err
            }
            return h.service.SetAccessVlan(ctx, req.PortID, req.VLANID, operator)
        })
}
```

**WAIT — re-read `execSinglePort` body carefully.** The `serviceCall` signature is `(ctx, portID, operator, desc)`. If `SetAccessVlan` needs the request body's `vlanId`, the request binding MUST happen INSIDE the serviceCall closure (binding inside closure is valid in Go). But the binding error path also needs to return early from the handler — which means the closure must return a sentinel error that `execSinglePort` will translate to 400. **Two options:**

**Option A (recommended)**: Bind body OUTSIDE `execSinglePort`, then call `execSinglePort` with a pre-bound wrapper:
```go
func (h *PortWriteHandler) SetAccessVlan(c *gin.Context) {
    var req struct {
        PortID string `json:"portId" binding:"required"`
        VLANID int    `json:"vlanId" binding:"required,min=1,max=4094"`
        Reason string `json:"reason,omitempty"`
    }
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
        return
    }
    h.execSinglePort(c, portcollection.ActionSetAccessVLAN, operlog.OperTypeUpdate,
        func(ctx context.Context, portID, operator, desc string) (*portwrite.PortResult, error) {
            return h.service.SetAccessVlan(ctx, portID, req.VLANID, operator)
        })
}
```

The handler BINDING is still outside the DRY (only the audit+operlog+response flow is DRY), which is acceptable — see `BatchWrite` handler (lines 213-285) which does exactly this: binds outside, then calls service directly.

**buildAfterValue extension** (lines 317-331 — extend switch):
```go
func buildAfterValue(action portcollection.PortAction) json.RawMessage {
    switch action {
    case portcollection.ActionShutdown:
        return json.RawMessage([]byte(`{"admin_status":"down"}`))
    // ... existing 4 ...
    case portcollection.ActionSetAccessVLAN:
        // Caller in buildAuditRow will overwrite with actual vlanId from request
        return json.RawMessage([]byte(`{}`))
    case portcollection.ActionPortBinding:
        // Caller in buildAuditRow will overwrite with actual ip/mac from request
        return json.RawMessage([]byte(`{}`))
    default:
        return json.RawMessage([]byte(`{}`))
    }
}
```

The `buildAuditRow` function (lines 349-390) needs to be extended to fill in actual vlanId / ip / mac into `after_value` based on the request payload. Pattern is similar to how `ActionDescription` case is handled at line 354-364.

**ModulePortWrite constant** (line 25) — already `端口管理`. Reuse as-is. OperType mapping per design.md §6: `set_access_vlan=Update(2)`, `port_binding add=Create(1)`, `port_binding remove=Delete(3)`.

**OperType 25-constant stability** — Phase 34 lock. `OperTypeUpdate`, `OperTypeCreate`, `OperTypeDelete` all exist. Verified via `internal/utils/operlog/regression_test.go`.

---

### `internal/api/v1/network/port_write_router.go` (router, request-response)

**Analog:** same file (lines 36-54). Pattern is to add new routes to the existing `/write` group with group-level `RequirePermissions` (already covers all routes via the 2-arg signature).

**Route registration pattern** (lines 48-53):
```go
write.POST("/shutdown", handler.Shutdown)
write.POST("/undo-shutdown", handler.UndoShutdown)
write.POST("/description", handler.SetDescription)
write.POST("/dot1x-enable", handler.EnableDot1x)
write.POST("/dot1x-disable", handler.DisableDot1x)
write.POST("/batch", handler.BatchWrite)
```

**Phase 56 additions** (append 2 lines after line 53):
```go
write.POST("/set-access-vlan", handler.SetAccessVlan)
write.POST("/port-binding", handler.PortBinding)
```

**Permission** — group-level `write.Use(middleware.RequirePermissions([]string{string(permission.NetworkPortWrite)}, core))` (line 46) automatically covers new routes. No per-route permission needed.

**Constructor pattern** (lines 37-42) — `portwrite.NewPortWriteService(core.GetDB(), core.DeviceExecutor, portcollection.NewCollectionService(...))`. No change for v1.20.1.

---

### `pkg/permission/config.go` (config, registry)

**Analog:** same file (lines 268-273 — existing 6 port-write rows). Pattern is to append to the slice literal.

**Registry row pattern** (lines 268-273):
```go
{"/network/ports/write/shutdown", "POST", NetworkPortWrite, "关闭端口"},
{"/network/ports/write/undo-shutdown", "POST", NetworkPortWrite, "撤销关闭端口"},
{"/network/ports/write/description", "POST", NetworkPortWrite, "修改端口描述"},
{"/network/ports/write/dot1x-enable", "POST", NetworkPortWrite, "启用端口802.1X认证"},
{"/network/ports/write/dot1x-disable", "POST", NetworkPortWrite, "关闭端口802.1X认证"},
{"/network/ports/write/batch", "POST", NetworkPortWrite, "批量端口写操作"},
```

**Phase 56 additions** (append 2 rows):
```go
{"/network/ports/write/set-access-vlan", "POST", NetworkPortWrite, "修改端口 access VLAN"},
{"/network/ports/write/port-binding", "POST", NetworkPortWrite, "端口绑定（IP/MAC/Port 静态绑定）"},
```

**No new permission constant** — reuse `NetworkPortWrite = "network:port:write"`. Per design.md §5.

---

### `xingran-react-frontend/src/types/network.ts` (type, n/a)

**Analog:** same file (lines 282-287 PortWriteAction union). Pattern is to extend the union literal + add 2 request interfaces.

**PortWriteAction union pattern** (lines 282-287):
```typescript
export type PortWriteAction =
  | "shutdown"
  | "undo_shutdown"
  | "description"
  | "dot1x_enable"
  | "dot1x_disable";
```

**Phase 56 extension**:
```typescript
export type PortWriteAction =
  | "shutdown"
  | "undo_shutdown"
  | "description"
  | "dot1x_enable"
  | "dot1x_disable"
  | "set_access_vlan"   // v1.20.1 NEW
  | "port_binding";     // v1.20.1 NEW
```

**Request interface pattern** (lines 314-319 — BatchWriteRequest):
```typescript
export interface BatchWriteRequest {
    deviceId: string;
    action: PortWriteAction;
    portIds: string[];
    description?: string;
}
```

**Phase 56 new interfaces** (add after BatchWriteRequest):
```typescript
export interface SetAccessVlanRequest {
    portId: string;
    vlanId: number;     // 1-4094 (frontend InputNumber + backend service validates)
    reason: string;
}

export interface PortBindingRequest {
    portId: string;
    op: "add" | "remove";
    ipAddress: string;  // IPv4 regex
    macAddress?: string;// MAC regex (optional)
    reason: string;
}
```

**BatchWriteRequest extension** — may need optional `vlanId` / `op` / `ipAddress` / `macAddress` fields to support the batch path of new actions. Add as `?` (optional) fields to maintain backward compat with existing 5 actions.

---

### `xingran-react-frontend/src/lib/api/networkApi.ts` (lib / api, request-response)

**Analog:** same file (lines 263-272 — `writeShutdown`). Pattern is `post<PortResult>(url, body) + result.data!` with kebab-aligned URL.

**Wrapper pattern** (lines 263-272):
```typescript
export const writeShutdown = async (
    portId: string,
    reason: string
): Promise<PortResult> => {
    const result = await post<PortResult>("/network/ports/write/shutdown", {
        portId,
        reason,
    });
    return result.data!;
};
```

**LANDMINE #5 (locked)**: no try/catch in wrapper, no message.error — `post()` interceptor handles it.

**Phase 56 new wrappers** (append after `writeDot1xDisable`):
```typescript
export const writeSetAccessVlan = async (
    portId: string,
    vlanId: number,
    reason: string
): Promise<PortResult> => {
    const result = await post<PortResult>("/network/ports/write/set-access-vlan", {
        portId, vlanId, reason,
    });
    return result.data!;
};

export const writePortBinding = async (
    portId: string,
    op: "add" | "remove",
    ipAddress: string,
    macAddress: string | undefined,
    reason: string
): Promise<PortResult> => {
    const result = await post<PortResult>("/network/ports/write/port-binding", {
        portId, op, ipAddress, macAddress, reason,
    });
    return result.data!;
};
```

**Export list** (lines 352-364) — append 2 to the default export.

---

### `xingran-react-frontend/src/components/network/port-write/constants.ts` (component / constant, n/a)

**Analog:** same file (lines 41-54 `ACTION_TITLE` Record). Pattern is to extend the Record type union + Record literal.

**ACTION_TITLE pattern** (lines 41-54):
```typescript
export const ACTION_TITLE: Record<
    | "shutdown"
    | "undo_shutdown"
    | "description"
    | "dot1x_enable"
    | "dot1x_disable",
    string
> = {
    shutdown: "关闭端口",
    undo_shutdown: "启用端口",
    description: "修改描述",
    dot1x_enable: "启用 802.1X",
    dot1x_disable: "停用 802.1X",
};
```

**Phase 56 extension**:
```typescript
export const ACTION_TITLE: Record<
    | "shutdown"
    | "undo_shutdown"
    | "description"
    | "dot1x_enable"
    | "dot1x_disable"
    | "set_access_vlan"
    | "port_binding",
    string
> = {
    shutdown: "关闭端口",
    undo_shutdown: "启用端口",
    description: "修改描述",
    dot1x_enable: "启用 802.1X",
    dot1x_disable: "停用 802.1X",
    set_access_vlan: "修改 access VLAN",
    port_binding: "端口绑定",
};
```

**New regex constants** (append after REASON_CUSTOM_SENTINEL):
```typescript
// IPv4: 严格 4 段十进制 0-255, 拒 0.x.x.x / 255.x.x.x 范围
export const IPV4_REGEX = /^(([1-9]?\d|1\d\d|2[0-4]\d|25[0-5])\.){3}([1-9]?\d|1\d\d|2[0-4]\d|25[0-5])$/;

// MAC: 接受 3 种常见格式 (canonical colon / hyphen / no-separator)
export const MAC_REGEX = /^([0-9A-Fa-f]{2}[:\-]?){5}[0-9A-Fa-f]{2}$/;

// BIND_OPS: Radio.Group options for PortBindingModal
export const BIND_OPS = [
    { label: "新增绑定 (add)", value: "add" },
    { label: "删除绑定 (remove)", value: "remove" },
] as const;
```

---

### `xingran-react-frontend/src/components/network/port-write/SetAccessVlanModal.tsx` (component, request-response)

**Analog:** `PortWriteModal.tsx` (full file, 240 lines). Pattern is full file skeleton: imports → state → useEffect reset → handleOk → Form with action-specific field + reason.

**Imports pattern** (lines 18-40):
```typescript
import { useEffect, useState } from "react";
import { Modal, Form, Select, Input, App } from "antd";
import type { MessageInstance } from "antd/es/message/interface";
import { useNavigate } from "react-router-dom";
import type { DevicePortStatus } from "@/types/network";
import { writeSetAccessVlan } from "@/lib/api/networkApi";
import {
    PRESET_REASONS,
    REASON_MIN,
    REASON_MAX,
    REASON_CUSTOM_SENTINEL,
    composeReason,
    validateReasonRequired,
} from "./constants";
import { showAuditLinkToast } from "./PortWriteModal";
```

**Form pattern** (lines 193-235) — copy structure, replace `description` field with `vlanId` InputNumber.

**vlanId field pattern**:
```tsx
<Form.Item
    name="vlanId"
    label="VLAN ID"
    rules={[
        { required: true, message: "请输入 VLAN ID" },
        { type: "number", min: 1, max: 4094, message: "VLAN ID 必须在 1-4094 之间" },
    ]}
    extra="范围 1-4094 (VLAN 0/4095 保留)"
>
    <InputNumber min={1} max={4094} step={1} style={{ width: "100%" }} placeholder="请输入 1-4094 之间的 VLAN ID" />
</Form.Item>
```

**handleOk pattern** (lines 118-163) — copy verbatim, replace `wrapper` calls:
```typescript
if (!portRecord) return;
await writeSetAccessVlan(portRecord.id, values.vlanId as number, reason ?? "");
showAuditLinkToast(message, navigate);
form.resetFields();
onSuccess();
onClose();
```

**Modal props pattern** (lines 182-192) — same width 520, same `okText="确认执行"`, same `okButtonProps={{ loading: submitting }}`, `destroyOnHidden`.

**useEffect reset pattern** (lines 114-116):
```typescript
useEffect(() => {
    if (open) form.resetFields();
}, [open, form]);
```

**Initial values** — pre-populate with `portRecord.vlan` (DevicePortStatus has `vlan?: number` per `types/network.ts:125`):
```typescript
initialValues={{ vlanId: portRecord?.vlan ?? 1 }}
```

**Title pattern** — `${ACTION_TITLE["set_access_vlan"]} - ${portRecord?.interfaceName ?? ""}` (mirrors line 183).

---

### `xingran-react-frontend/src/components/network/port-write/PortBindingModal.tsx` (component, request-response)

**Analog:** `PortWriteModal.tsx` (full file). Same skeleton as SetAccessVlanModal, but Form has 3 fields: op Radio.Group + ipAddress Input + macAddress Input.

**Imports pattern** (additional Radio import):
```typescript
import { Modal, Form, Select, Input, Radio, App } from "antd";
```

**Form fields pattern**:
```tsx
{/* op: Radio.Group (mandatory) */}
<Form.Item name="op" label="操作" rules={[{ required: true, message: "请选择绑定操作" }]}>
    <Radio.Group buttonStyle="solid" options={BIND_OPS} />
</Form.Item>

{/* ipAddress: IPv4 regex */}
<Form.Item name="ipAddress" label="IP 地址" rules={[
    { required: true, message: "请输入 IP 地址" },
    { pattern: IPV4_REGEX, message: "请输入合法 IPv4 地址（如 10.62.25.5）" },
]}>
    <Input placeholder="例如 10.62.25.5" allowClear />
</Form.Item>

{/* macAddress: optional MAC regex */}
<Form.Item name="macAddress" label="MAC 地址（可选）" rules={[
    { pattern: MAC_REGEX, message: "请输入合法 MAC 地址（如 AA:BB:CC:DD:EE:FF）" },
]}>
    <Input placeholder="例如 AA:BB:CC:DD:EE:FF（不填则仅 IP 绑定）" allowClear />
</Form.Item>
```

**handleOk pattern**:
```typescript
await writePortBinding(
    portRecord.id,
    values.op as "add" | "remove",
    values.ipAddress as string,
    (values.macAddress as string) || undefined,
    reason ?? ""
);
```

**Title pattern** — `${ACTION_TITLE["port_binding"]} - ${portRecord?.interfaceName ?? ""}`.

---

### `xingran-react-frontend/src/pages/network/ports/index.tsx` (page, request-response)

**Analog:** same file (lines 343-352 ActionButtons array). Pattern is to append 2 menu items + 2 Modal mount points + 2 state vars.

**Menu item pattern** (lines 343-349):
```typescript
const actions: ActionButton[] = [
    { key: "shutdown", label: "关闭端口", onClick: () => openWriteModal("shutdown", record) },
    { key: "undo_shutdown", label: "启用端口", onClick: () => openWriteModal("undo_shutdown", record) },
    { key: "description", label: "修改描述", onClick: () => openWriteModal("description", record) },
    { key: "dot1x_enable", label: "启用 802.1X", onClick: () => openWriteModal("dot1x_enable", record) },
    { key: "dot1x_disable", label: "停用 802.1X", onClick: () => openWriteModal("dot1x_disable", record) },
];
```

**Phase 56 extension** (append 2 items + 2 menu items to a NEW separate array for the new modals):
```typescript
// v1.20.1: New modals are separate components (not PortWriteModal), so use dedicated openers
{ key: "set_access_vlan", label: "修改 access VLAN", onClick: () => openVlanModal(record) },
{ key: "port_binding", label: "端口绑定", onClick: () => openBindModal(record) },
```

**State pattern** (lines 64-67):
```typescript
const [writeModalOpen, setWriteModalOpen] = useState(false);
const [writeModalAction, setWriteModalAction] = useState<PortWriteAction>("shutdown");
const [writeModalRecord, setWriteModalRecord] = useState<DevicePortStatus | null>(null);
const [bulkWriteDrawerOpen, setBulkWriteDrawerOpen] = useState(false);
```

**Phase 56 state additions**:
```typescript
const [vlanModalOpen, setVlanModalOpen] = useState(false);
const [vlanModalRecord, setVlanModalRecord] = useState<DevicePortStatus | null>(null);
const [bindModalOpen, setBindModalOpen] = useState(false);
const [bindModalRecord, setBindModalRecord] = useState<DevicePortStatus | null>(null);

const openVlanModal = (record: DevicePortStatus) => {
    setVlanModalRecord(record);
    setVlanModalOpen(true);
};
const openBindModal = (record: DevicePortStatus) => {
    setBindModalRecord(record);
    setBindModalOpen(true);
};
```

**Modal mount pattern** (lines 545-551):
```typescript
<PortWriteModal
    open={writeModalOpen}
    action={writeModalAction}
    portRecord={writeModalRecord}
    onClose={() => setWriteModalOpen(false)}
    onSuccess={() => { loadPortStatus(); loadStatistics(); }}
/>
```

**Phase 56 mount additions**:
```typescript
<SetAccessVlanModal
    open={vlanModalOpen}
    portRecord={vlanModalRecord}
    onClose={() => setVlanModalOpen(false)}
    onSuccess={() => { loadPortStatus(); loadStatistics(); }}
/>
<PortBindingModal
    open={bindModalOpen}
    portRecord={bindModalRecord}
    onClose={() => setBindModalOpen(false)}
    onSuccess={() => { loadPortStatus(); loadStatistics(); }}
/>
```

**Imports** — add to existing import block (line 43-44):
```typescript
import { SetAccessVlanModal } from "@/components/network/port-write/SetAccessVlanModal";
import { PortBindingModal } from "@/components/network/port-write/PortBindingModal";
```

**canWrite gating** — wrap ALL action menus in `(canWrite ? [...] : [])` (line 337). v1.20.1 menu items inherit the same gating.

**batchInProgress** — does NOT affect single-port Modals (only batch Drawer). v1.20.1 single-port Modals don't need this prop.

---

## Shared Patterns

### Authentication / Permission

**Source:** `internal/api/v1/network/port_write_router.go:46` (group-level middleware)
```go
write.Use(middleware.RequirePermissions([]string{string(permission.NetworkPortWrite)}, core))
```

**Apply to:** All `/network/ports/write/*` routes. Group-level middleware already covers the 2 new kebab routes. No change needed in router beyond adding the `write.POST(...)` lines.

**2-arg signature** (locked, Phase 52 + `TestSetupPortWriteRouter_RequirePermissions2Arg`):
- `RequirePermissions(perms []string, core *core.Core) gin.HandlerFunc`
- Do NOT add a 3rd argument — would break the test.

### operlog Audit

**Source:** `internal/api/v1/network/port_write_handler.go:194-195` (Path C pattern)
```go
operlog.Record(c, h.core.OperLogService, h.core.GetDB(), ModulePortWrite, operType,
    operlog.WithOperParam(operParam))
```

**Apply to:** All 2 new handlers. `ModulePortWrite = "端口管理"` (line 25). OperType mapping per design.md §6:
- `set_access_vlan` → `OperTypeUpdate` (value 2)
- `port_binding op=add` → `OperTypeCreate` (value 1)
- `port_binding op=remove` → `OperTypeDelete` (value 3)

**Reuse helpers** — `buildAuditRow`, `buildSinglePortOperParam`, `buildAfterValue` extended with new actions. Phase 34 lock: 25 OperType constants + 11 sensitive keywords + 5-param `Record` signature enforced by `internal/utils/operlog/regression_test.go`.

### Error Handling (sentinel → HTTP)

**Source:** `internal/api/v1/network/port_write_handler.go:146-165` (execSinglePort sentinel translation)
```go
switch {
case errors.Is(err, portwrite.ErrPortNotFound):
    response.Error(c, http.StatusNotFound, "端口不存在")
    return
case errors.Is(err, portwrite.ErrDeviceNotFound):
    response.Error(c, http.StatusNotFound, "设备不存在")
    return
}
```

**Apply to:** 2 new handlers via `execSinglePort` DRY. New sentinel errors `ErrVlanIdOutOfRange` / `ErrBindOpInvalid` / `ErrIPAddressInvalid` / `ErrMACAddressInvalid` translate to 400. Add to the switch in `execSinglePort`:
```go
case errors.Is(err, portwrite.ErrVlanIdOutOfRange):
    response.Error(c, http.StatusBadRequest, "VLAN ID 必须在 1-4094 之间")
    return
case errors.Is(err, portwrite.ErrBindOpInvalid):
    response.Error(c, http.StatusBadRequest, "绑定操作必须是 add 或 remove")
    return
case errors.Is(err, portwrite.ErrIPAddressInvalid):
    response.Error(c, http.StatusBadRequest, "IP 地址格式不合法")
    return
case errors.Is(err, portwrite.ErrMACAddressInvalid):
    response.Error(c, http.StatusBadRequest, "MAC 地址格式不合法")
    return
```

### Audit Log Toast (Frontend)

**Source:** `PortWriteModal.tsx:73-91` (`showAuditLinkToast` helper)
```typescript
export function showAuditLinkToast(message: MessageInstance, navigate: (path: string) => void): void {
    const handleClick = (e: React.MouseEvent<HTMLAnchorElement>): void => {
        e.preventDefault();
        message.destroy();
        navigate(AUDIT_LOG_PATH);
    };
    message.open({
        type: "success",
        duration: 5,
        content: (
            <span>
                操作成功，
                <a href={AUDIT_LOG_PATH} onClick={handleClick}>
                    查看审计日志
                </a>
            </span>
        ),
    });
}
```

**Apply to:** 2 new Modals. Import `showAuditLinkToast` from `./PortWriteModal`. Call after successful wrapper invocation. Zero code change to the helper.

**LANDMINE #5** (locked): no try/catch in wrapper, no message.error in component. `post()` interceptor handles all error toasts.

### Reason Validation (Frontend)

**Source:** `constants.ts:127-145` (`validateReasonRequired`)
```typescript
export function validateReasonRequired(
    _: unknown,
    value: unknown,
    form: FormInstance
): Promise<void> {
    const reasonSelect = value;
    const reasonText = form.getFieldValue("reasonText");
    const reason = composeReason(reasonSelect, reasonText);
    if (reason === null || reason.length === 0) {
        return Promise.reject(new Error("请选择或输入操作原因"));
    }
    if (reason.length < REASON_MIN) {
        return Promise.reject(new Error(`操作原因至少 ${REASON_MIN} 个字符`));
    }
    if (reason.length > REASON_MAX) {
        return Promise.reject(new Error(`操作原因不超过 ${REASON_MAX} 个字符`));
    }
    return Promise.resolve();
}
```

**Apply to:** 2 new Modals. reason is REQUIRED (5-200 chars). Reuse the same validator + `validateReasonRequired(rule, value, form)` cross-field pattern from 55-01 WR-02 fix.

### MAC Normalization (Backend)

**Source:** `internal/services/mac_normalize.go:32-56` (`NormalizeMACAddress`)
```go
func NormalizeMACAddress(input string) string {
    // 去除常见分隔符 + 大写化 + 12 hex 校验 + 重新插入冒号
    // Returns AA:BB:CC:DD:EE:FF or "" on invalid
}
```

**Apply to:** `PortBinding` service method — normalize user input MAC before validation, then convert to vendor-specific format (Huawei/H3C: `AA-BB-CC-DD-EE-FF`; Ruijie: `AABB.CCDD.EEFF`) inside the render function.

### SSH Transport

**Source:** `internal/services/portwrite/port_write_service.go:249-279` (`deviceExecutor.ExecuteCustom` + `wrapper.SendConfigs` + `parseConfigError`)
```go
executeErr := s.deviceExecutor.ExecuteCustom(ctx, deviceID, func(execCtx context.Context, pc *device.PooledConnection) error {
    wrapper := pc.GetWrapper()
    fullCmds := cmds
    if exitCmd := portcollection.VendorExitViewCmd(dev.Vendor); exitCmd != "" {
        fullCmds = append(append([]string{}, cmds...), exitCmd)
    }
    responses, sendErr := wrapper.SendConfigs(fullCmds)
    // ... parseConfigError loop ...
}, singlePortTimeout)
```

**Apply to:** 2 new actions automatically via `executeWrite` + `writeAndRefresh`. Zero change. The new render closures feed through `RenderCommand` → `cmds` → `SendConfigs`.

**VendorExitViewCmd** (lines 95-102) — appends trailing `exit` (Ruijie) or `quit` (Huawei/H3C) to prevent cross-port view stuck. New actions inherit this automatically.

**expandInterfaceName** (lines 392-424) — converts normalized short name to vendor CLI full name. New actions inherit this automatically.

### Frontend API Wrapper Pattern (LANDMINE #5)

**Source:** `networkApi.ts:263-272` (writeShutdown)
```typescript
export const writeShutdown = async (
    portId: string,
    reason: string
): Promise<PortResult> => {
    const result = await post<PortResult>("/network/ports/write/shutdown", {
        portId,
        reason,
    });
    return result.data!;
};
```

**Apply to:** 2 new wrappers `writeSetAccessVlan` + `writePortBinding`. NO try/catch, NO message.error, kebab-aligned URL.

### form.resetFields() on Modal open

**Source:** `PortWriteModal.tsx:114-116`
```typescript
useEffect(() => {
    if (open) form.resetFields();
}, [open, action, form]);
```

**Apply to:** 2 new Modals. Reuse same pattern. `SetAccessVlanModal` deps = `[open, form]`. `PortBindingModal` deps = `[open, form]`.

### destroyOnHidden + 520px width

**Source:** `PortWriteModal.tsx:182-192`
```tsx
<Modal
    title={...}
    open={open}
    onOk={handleOk}
    onCancel={onClose}
    destroyOnHidden
    width={520}
    okText="确认执行"
    cancelText="取消"
    okButtonProps={{ loading: submitting }}
>
```

**Apply to:** 2 new Modals. Same 520px, same `okText="确认执行"`, same `okButtonProps={{ loading: submitting }}`, same `destroyOnHidden`.

---

## No Analog Found

None. Every file has a direct existing analog in v1.19. Phase 56 is purely additive — no new patterns, no new abstractions. The planner should reference v1.19 locked patterns verbatim and extend with 2 new action × 3 vendor entries.

---

## Metadata

**Analog search scope:**
- `internal/services/portcollection/` (vendor template)
- `internal/services/portwrite/` (service, pre-state, e2e, testdata)
- `internal/api/v1/network/` (handler, router)
- `pkg/permission/`, `pkg/middleware/` (perm/perm-middleware)
- `internal/utils/operlog/` (audit)
- `internal/services/mac_normalize.go` (MAC helper)
- `xingran-react-frontend/src/components/network/port-write/` (Modal/Drawer/constants)
- `xingran-react-frontend/src/types/network.ts` (type definitions)
- `xingran-react-frontend/src/lib/api/networkApi.ts` (wrappers)
- `xingran-react-frontend/src/pages/network/ports/index.tsx` (ActionButtons page)

**Files scanned:** 14 analog files (all read in full or targeted sections for the larger ones).

**Pattern extraction date:** 2026-07-09
**Pattern lock source:** v1.19 phase closure (37/37 requirements, 0 broken wires, Phase 52 + 53 + 55 fixes baked in)
