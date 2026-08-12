# Phase 56: 网络设备 VLAN + 端口绑定 (v1.20.1) - Research

**Researched:** 2026-07-09
**Domain:** Network device write operations — VLAN + IP/MAC/Port static binding (Huawei VRP / H3C Comware / Ruijie RGOS)
**Confidence:** MEDIUM-HIGH (vendor template structure locked from v1.19, but several RISK items partially resolved via web search requiring UAT confirmation)

---

## Summary

Phase 56 extends the v1.19 port-write MVP (3 vendors × 5 actions) with 2 new actions: `set_access_vlan` and `port_binding`. It is **purely additive** — no schema change, no new perm constant, no new audit table. The build extends the existing `vendorPortTemplate` map, `checkPreState` switch, `PortWriteService` interface, `port_write_handler.go` `execSinglePort` DRY, `port_write_router.go` group, and frontend `PortWriteModal`/`BulkWriteDrawer`/ActionButtons menu. ~22 requirements across VLAN / BIND / INFRA / UI / TEST.

**Critical research finding — RISK-01 RESOLVED but in unexpected direction:** Per H3C official documentation (`h3c.com/cn/d_202409/2262994_30005_0.htm`), H3C Comware V7 uses `user-bind` (without the `static` keyword), while Huawei VRP uses `user-bind static`. The design doc assumed keyword parity — this is wrong. **H3C requires a dedicated template branch**, not "类 Huawei 模板结构". Update to §3.2 of design doc required before W1 plan executes.

**Critical research finding — RISK-02 PARTIALLY RESOLVED:** Per Ruijie port-security manual, the binding syntax is **`switchport port-security binding <mac-address> [vlan <vlan-id>] <ip-address>`** (full tuple: MAC + optional VLAN + IP). The design doc claimed "不支持 MAC 参数" — incorrect. Both MAC and IP are first-class parameters. However, the *display* command `show port-security binding` shows separate columns; what the user's field-collected output lacks is unclear.

**Critical research finding — RISK-03 RESOLVED in unexpected direction:** Per S5700/S5735 community docs, `port link-type access` is a **mandatory prerequisite** for `port default vlan` on all firmware versions (not just V200R003). The design doc's "V200R008+ 可省" assumption is incorrect. **Huawei template must always emit `port link-type access` first** — already proposed in design doc §RISK-03 mitigation, so no change needed beyond confirming the rationale.

The overall architectural pattern is well-established (v1.19 closed 2026-07-08, 37/37 requirements satisfied, 6 phases shipped, 0 broken wires). The 5-wave build (vendor template → service+pre-state → router+handler → frontend → e2e+docs) is reused directly.

**Primary recommendation:** Reuse v1.19 wave structure 1:1. Update §3.2 of design doc to reflect H3C's `user-bind` (no `static`) before W1. Add H3C to HUMAN-UAT.md as required (not optional) since keyword divergence is a real risk.

---

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Vendor command template (Huawei/H3C/Ruijie syntax) | Backend service (Go: `internal/services/portcollection/vendor_port_template.go`) | — | Pure-Go string templating, table-driven; already locked from v1.19 |
| Pre-state read command + parser | Backend service (Go: `internal/services/portwrite/pre_state_check.go`) | Frontend (read-only DB cache) | SSH read-side device-state collection for NoOp short-circuit |
| Service entry + validators (VLAN range, IP regex, MAC regex) | Backend service (Go: `port_write_service.go`) | — | All validations happen in `SetAccessVlan` / `PortBinding` methods; rejects bad params before SSH |
| HTTP handlers + operlog audit + permission | Backend API (Go: `port_write_handler.go` + `port_write_router.go`) | — | execSinglePort DRY pattern + OperType Update/Create/Delete |
| Single-port Modal form (VLAN ID / IP / MAC) | Frontend component (`SetAccessVlanModal.tsx`, `PortBindingModal.tsx`) | Frontend types (`network.ts`) | New TSX components; reuse v1.19 `port-write/constants.ts` REASON validator |
| Network API wrappers | Frontend lib (`networkApi.ts`) | — | kebab-aligned wrappers, 0 try/catch + 0 Toast duplicate (LANDMINE #5) |
| Bulk operation UI reuse | Frontend (`BulkWriteDrawer.tsx`) | — | Reuse v1.19 three-state machine + audit toast — **zero code change** for v1.20.1 |
| Real-device UAT verification | On-site operator | Backend e2e (scrapligo FileTransport) | Site-visit SSH validation deferred per v1.18/v1.19 precedent; mock SSH e2e in W5 |

---

## User Constraints

### Locked Decisions (from `docs/plans/2026-07-09-v1.20.1-design.md`)

| # | Decision | Source |
|---|----------|--------|
| DECISION-01 | Don't rename existing 5 actions (v1.19 audit-log compat). New actions: `ActionSetAccessVLAN` / `ActionPortBinding`. | design.md §13 |
| DECISION-02 | `port_binding` 1 action covers add/remove via `op` field. operlog OperType branches: add=Create, remove=Delete. | design.md §13 |
| DECISION-03 | No "binding list view" UI feature. Reuse `display user-bind static all` in pre-state to read full list; UI viewing deferred to FUTURE-OP-03. | design.md §13 |
| Vendors | Huawei / H3C / Ruijie only (Cisco deferred per v1.19 FUTURE-OP-08). | design.md §2.3 |
| Audit | Reuse `sys_port_write_audit` table (json.RawMessage JSONB columns). No schema change. | design.md §7 |
| Permission | Reuse `network:port:write` constant (no new perm). | design.md §5 |
| Menu | Reuse "端口配置" F-button menu (no new menu entry). | design.md §5 |
| Architecture | Reuse v1.19 5-wave structure (vendor template → service → router → frontend → e2e+docs). | design.md §11 |
| OperType mapping | set_access_vlan=Update(2), port_binding add=Create(1), port_binding remove=Delete(3). | design.md §6 |

### Claude's Discretion (from design doc §13 RISK mitigations)

- H3C `user-bind` keyword — **MUST be revised**: use H3C-specific template branch with `user-bind` (no `static`) per H3C Comware V7 manual.
- Ruijie MAC parameter support — **use full `<mac-address> [vlan <vlan-id>] <ip-address>` syntax** per Ruijie manual; if MAC column is missing from display, render MAC in oper_param only (audit visibility).
- Huawei legacy firmware `port link-type access` prefix — **always emit** (universally compatible per S5700/S5735 community validation).

### Deferred Ideas (out of scope)

- trunk VLAN / hybrid VLAN / port-security enable/max-mac/sticky / dry-run / operation-history UI / Cisco / auto-rollback / multi-user mutex — all explicitly deferred per design.md §2.2 + §14.

---

## Standard Stack

### Core (no change — reuse v1.19)

| Library | Version | Purpose | Already in v1.19 |
|---------|---------|---------|------------------|
| `github.com/scrapli/scrapligo` | v1.3.3 | SSH transport + SendConfigs | YES |
| `github.com/gin-gonic/gin` | v1.10.0 | HTTP routing + middleware | YES |
| `gorm.io/gorm` | v1.30.5 | ORM (auto-migrate sys_port_write_audit, sys_device_port_status) | YES |
| `github.com/google/uuid` | v1.6.0 | UUID primary keys | YES |
| `github.com/xingran-next/xingran-go-backend/internal/utils/operlog` | (project internal) | `Record` / `OperType` constants | YES |
| `github.com/xingran-next/xingran-go-backend/internal/services/mac_normalize.go` | (project internal) | `NormalizeMACAddress()` returns `AA:BB:CC:DD:EE:FF` | YES |

### Frontend (no change — reuse v1.19)

| Library | Version | Purpose | Already in v1.19 |
|---------|---------|---------|------------------|
| `react` + `react-dom` | 19.2 | UI runtime | YES |
| `antd` | 6.1 | Modal / Form / Radio / InputNumber / Drawer | YES |
| `zustand` | 5.0 | authStore (TokenManager) | YES |
| `@/lib/api` | (project) | `post<T>(url, body)` + 401 interceptor | YES |
| `dayjs` | 1.11 | Timestamp formatting | YES |

### New code only — no new packages, no new dependencies

Per `pkg/permission/config.go:189 NetworkPortWrite` constant and `internal/services/portcollection/` existing scaffold, Phase 56 adds **zero new go.mod entries** and **zero new package.json entries**.

---

## Package Legitimacy Audit

**N/A** — Phase 56 adds no external packages. All work uses existing v1.19 dependencies. Verified via `git diff v1.19..HEAD -- go.mod package.json` expectation: no diff.

---

## Architecture Patterns

### System Architecture Diagram

```
User Action (Web)
        ↓
ports/index.tsx ActionButtons menu (新增 2 项)
        ↓ click
SetAccessVlanModal / PortBindingModal
        ↓ validateFields + IP/MAC/VLAN regex
writeSetAccessVlan / writePortBinding wrapper
        ↓ POST /network/ports/write/{set-access-vlan|port-binding}
Gin router (2 kebab routes, group-level perm)
        ↓ execSinglePort DRY
port_write_handler.go → operlog.Record + audit row
        ↓
port_write_service.go → SetAccessVlan / PortBinding
        ↓
pre_state_check.go (checkPreState) — read current PVID / bindings, NoOp if match
        ↓ NoOp=false
executeWrite → RenderCommand → vendor_port_template map
        ↓ (Huawei: port link-type access | port default vlan X | quit)
        ↓ (H3C:     port link-type access | port access vlan X   | quit)
        ↓ (Ruijie:  switchport mode access | switchport access vlan X | exit)
deviceExecutor.ExecuteCustom → scrapligo SendConfigs
        ↓ SSH
Device (Huawei/H3C/Ruijie) ← execute
        ↓ response
parseConfigError → success/failed
        ↓ success
fire-and-forget CollectDevice(deviceID) (refresh sys_device_port_status)
        ↓
response.Success → frontend showAuditLinkToast
```

### Recommended File Structure (extending v1.19)

**Backend (8 files modified, 0 created):**
```
internal/services/portcollection/vendor_port_template.go         [MODIFY: +2 actions × 3 vendors]
internal/services/portcollection/vendor_port_template_test.go    [MODIFY: +12 test cases]
internal/services/portwrite/pre_state_check.go                  [MODIFY: +2 action handlers]
internal/services/portwrite/port_write_service.go               [MODIFY: +2 methods]
internal/services/portwrite/port_write_service_test.go          [MODIFY: +5 unit tests]
internal/api/v1/network/port_write_handler.go                   [MODIFY: +2 handlers]
internal/api/v1/network/port_write_router.go                    [MODIFY: +2 routes]
internal/services/portwrite/port_write_e2e_test.go              [MODIFY: +10 e2e tests]
internal/services/portwrite/testdata/                           [CREATE: 6+ fixtures]
```

**Frontend (6 files: 4 modified, 2 created):**
```
xingran-react-frontend/src/types/network.ts                        [MODIFY: +2 type aliases]
xingran-react-frontend/src/lib/api/networkApi.ts                  [MODIFY: +2 wrappers]
xingran-react-frontend/src/components/network/port-write/constants.ts  [MODIFY: +2 entry + 2 regex]
xingran-react-frontend/src/pages/network/ports/index.tsx          [MODIFY: +2 menu items]
xingran-react-frontend/src/components/network/port-write/SetAccessVlanModal.tsx  [CREATE]
xingran-react-frontend/src/components/network/port-write/PortBindingModal.tsx    [CREATE]
xingran-react-frontend/src/components/network/port-write/BulkWriteDrawer.tsx    [UNCHANGED — zero modification]
```

**Documentation (3 files modified, 1 created):**
```
docs/API响应规范.md                                             [MODIFY: network-device-write section]
docs/开发规范.md                                               [MODIFY: vendor command matrix]
CHANGELOG.md                                                  [MODIFY: v1.20.1 entries]
.planning/phases/56-.../56-HUMAN-UAT.md                        [CREATE: site-visit deferral]
```

### Pattern 1: Vendor Template Extension (Pattern locked from v1.19)

**What:** Add 2 new actions to the `vendorPortTemplate` map; each vendor gets its own closure.

**When to use:** Every new port-write action that differs in vendor syntax.

**Example:**
```go
// Source: v1.19 vendor_port_template.go:47-72 pattern
const (
    ActionSetAccessVLAN PortAction = "set_access_vlan"
    ActionPortBinding   PortAction = "port_binding"
)

type PortTemplateParams struct {
    InterfaceName string
    Description   string
    // v1.20.1 additions:
    VLANID    int      // ActionSetAccessVLAN only
    BindOp    string   // "add"|"remove" — ActionPortBinding only
    IPAddress string   // ActionPortBinding only
    MACAddress string  // ActionPortBinding only
}

var vendorPortTemplate = map[models.DeviceVendor]map[PortAction]func(PortTemplateParams) ([]string, error){
    models.VendorHuawei: {
        // ... v1.19 5 actions ...
        ActionSetAccessVLAN: renderHuaweiSetAccessVlan,
        ActionPortBinding:   renderHuaweiPortBinding,
    },
    // ...
}
```

**v1.20.1 render functions** (revised from design doc per H3C RISK-01 resolution):

```go
// Huawei: always emit `port link-type access` first (RISK-03 confirmed universal requirement)
func renderHuaweiSetAccessVlan(p PortTemplateParams) ([]string, error) {
    return []string{
        fmt.Sprintf("interface %s", p.InterfaceName),
        "port link-type access",
        fmt.Sprintf("port default vlan %d", p.VLANID),
    }, nil
}

// H3C: `user-bind` keyword WITHOUT `static` (per H3C Comware V7 manual)
func renderH3CPortBindingAdd(p PortTemplateParams) ([]string, error) {
    cmd := fmt.Sprintf("user-bind ip-address %s", p.IPAddress)
    if p.MACAddress != "" {
        // H3C MAC format: AABB-CCDD-EEFF (hyphenated, uppercase) — see §Normalization
        macH3C := toH3CMACFormat(p.MACAddress)
        cmd += fmt.Sprintf(" mac-address %s", macH3C)
    }
    return []string{
        fmt.Sprintf("interface %s", p.InterfaceName),
        cmd,
    }, nil
}

// Ruijie: full <mac-address> [vlan <vlan-id>] <ip-address> syntax (per Ruijie port-security manual)
func renderRuijiePortBindingAdd(p PortTemplateParams) ([]string, error) {
    cmd := "switchport port-security binding"
    if p.MACAddress != "" {
        macRuijie := toRuijieMACFormat(p.MACAddress) // AABB.CCDD.EEFF (Cisco dot-style)
        cmd += fmt.Sprintf(" %s", macRuijie)
    }
    cmd += fmt.Sprintf(" %s", p.IPAddress)
    return []string{
        fmt.Sprintf("interface %s", p.InterfaceName),
        cmd,
    }, nil
}
```

### Pattern 2: Pre-State Check Extension (Pattern locked from v1.19)

**What:** Extend `checkPreState` switch with 2 new action handlers; reuse port collection's existing read commands.

**When to use:** Every action that has a deterministic target state queryable via SSH.

**Example:**
```go
// Source: v1.19 pre_state_check.go:22-75 pattern
func (s *portWriteServiceImpl) checkPreState(
    port *models.DevicePortStatus,
    action Action,
    desc string,                  // v1.19
    vlanId int,                   // v1.20.1 NEW
    bindOp, ipAddr, macAddr string, // v1.20.1 NEW
) *PortResult {
    switch action {
    // ... v1.19 5 actions ...
    case portcollection.ActionSetAccessVLAN:
        // port.VLAN 字段（已在 DevicePortStatus 模型中）若 == 目标 vlanId → NoOp
        if port.VLAN != nil && *port.VLAN == vlanId {
            return &PortResult{...CurrentState: "vlan_match"...}
        }
    case portcollection.ActionPortBinding:
        // 调用 portcollection 现有的 displayBindingTable 方法或新增 SSH read
        // 查找 (ipAddr, [macAddr], port.InterfaceName) tuple
        // add 命中 → NoOp
        // remove 未命中 → NoOp
    }
}
```

**⚠️ Pre-state check pre-condition:** Current `pre_state_check.go` operates on DB-resident `DevicePortStatus`. For `set_access_vlan`, the model already has a `VLAN` field (see `types/network.ts:125 vlan?: number`) — reuse directly. For `port_binding`, **NO DB model** stores binding tuples; pre-state check requires a fresh SSH read (`display user-bind` / `display user-bind static all` / `show port-security binding`). This adds a NEW SSH round-trip per write — design.md §4.2 acknowledges this but does not flag the cost. Mitigation: keep the SSH read simple (parse line-by-line, exit on first match); vendors typically return <50 lines per interface.

### Pattern 3: Reuse execSinglePort DRY (Pattern locked from v1.19)

**What:** Add 2 new handlers that call `h.execSinglePort(c, action, operType, serviceCall)` — zero change to DRY logic.

**Example:**
```go
// Source: v1.19 port_write_handler.go:65-105 pattern

func (h *PortWriteHandler) SetAccessVlan(c *gin.Context) {
    var req SetAccessVlanRequest  // NEW struct: { PortID, VLANID, Reason }
    // ... c.ShouldBindJSON, validation ...
    h.execSinglePort(c, portcollection.ActionSetAccessVLAN, operlog.OperTypeUpdate,
        func(ctx context.Context, portID, operator, desc string) (*portwrite.PortResult, error) {
            return h.service.SetAccessVlan(ctx, portID, req.VLANID, operator)
        })
}

func (h *PortWriteHandler) PortBinding(c *gin.Context) {
    var req PortBindingRequest  // NEW struct: { PortID, Op, IPAddress, MACAddress, Reason }
    // ... c.ShouldBindJSON, validation ...
    operType := operlog.OperTypeCreate
    if req.Op == "remove" {
        operType = operlog.OperTypeDelete
    }
    h.execSinglePort(c, portcollection.ActionPortBinding, operType,
        func(ctx context.Context, portID, operator, desc string) (*portwrite.PortResult, error) {
            return h.service.PortBinding(ctx, portID, req.Op, req.IPAddress, req.MACAddress, operator)
        })
}
```

### Pattern 4: BulkWriteDrawer Zero-Modification Reuse

**What:** `BulkWriteDrawer` already supports any `PortWriteAction` value via `ACTION_OPTIONS` map. Adding `set_access_vlan` + `port_binding` to `ACTION_TITLE` is sufficient.

**Example:**
```typescript
// Source: v1.19 constants.ts:41-54 — extend Record with 2 keys
export const ACTION_TITLE: Record<
  | "shutdown" | "undo_shutdown" | "description" | "dot1x_enable" | "dot1x_disable"
  | "set_access_vlan" | "port_binding",  // v1.20.1 NEW
  string
> = {
  // ... existing 5 ...
  set_access_vlan: "修改 access VLAN",
  port_binding: "端口绑定",
};
```

### Anti-Patterns to Avoid

- **Don't** add a new permission constant. `network:port:write` already covers write operations; UI gating (`canWrite`) reuses v1.19.
- **Don't** add new audit table. `sys_port_write_audit.before_value` / `after_value` JSONB already supports arbitrary JSON.
- **Don't** add new menu seed. "端口配置" F-button menu already mounts the ActionButtons group.
- **Don't** bypass `execSinglePort` DRY. The handler is locked for D-04 / D-13 / D-14 invariant guards.
- **Don't** introduce new SSH transport pattern. `deviceExecutor.ExecuteCustom` is the only SSH entry; `BatchWritePorts` remains the only batch entry.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| MAC format conversion | Custom string-format loop | `NormalizeMACAddress()` from `internal/services/mac_normalize.go` | Already handles all 4 input formats → `AA:BB:CC:DD:EE:FF`. Project memory `mac-address-normalize-returns-colon-format` documents expected behavior. |
| Interface name full-name expansion | Custom prefix map | `expandInterfaceName()` from `port_write_service.go:392-424` | Already handles Huawei (no expand) / H3C (no space) / Ruijie (with space) per project memory `normalize-iface-reverse-expand-trap`. |
| Sensitive-key masking | Custom regex over oper_param | `operlog.RecordWithBody` or `operlog.Record` with `RecordOption` | Project-wide Phase 34 lock — 11 mandatory keywords + 5-param Record signature enforced by `regression_test.go`. |
| Batch SSH write orchestration | Custom fail-fast loop | `BatchWritePorts(ctx, req, operator)` from `port_write_service.go` | D-17 serial fail-fast + detached 30min context + per-port refresh invariants are encoded and tested. |
| HTTP error translation | Custom `if err != nil` chains | `response.Error(c, status, msg)` from `pkg/response` | Project-wide standardized; 401/403/400/500 codes locked by `pkg/middleware`. |
| Permission gating | Custom role lookup | `middleware.RequirePermissions([]string{perm}, core)` | 2-arg signature locked by Phase 52 + verified by `TestSetupPortWriteRouter_RequirePermissions2Arg`. |

**Key insight:** v1.19 already shipped a complete infrastructure (vendor template, pre-state, service, batch, handler, router, audit, permission, frontend modal/drawer/api-wrappers/types). v1.20.1 must reuse this infrastructure end-to-end — every "new" feature is an extension of an existing file, never a parallel implementation.

---

## Common Pitfalls

### Pitfall 1: H3C `user-bind` keyword divergence (RESEARCH FINDING — REQUIRES DESIGN DOC UPDATE)

**What goes wrong:** Assuming H3C uses `user-bind static` (same as Huawei). Device returns `% Unrecognized command` and binding silently fails.

**Why it happens:** H3C Comware V7's `user-bind` command is structurally similar to Huawei's `user-bind static` but lacks the `static` keyword. The two vendors diverged in IPSG (IP Source Guard) implementation despite sharing VRP heritage.

**How to avoid:** Add H3C-specific template branch in `vendor_port_template.go`. Use `user-bind ip-address <IP> [mac-address <MAC>]` for H3C; use `user-bind static ip-address <IP> [mac-address <MAC>]` for Huawei. **Update design.md §3.2 row for H3C before W1 plan executes.**

**Warning signs:** H3C UAT returns `% Error: Unrecognized command found at '^'.` after `user-bind static` — pre-state check is bypassed because command never executes against device.

### Pitfall 2: Ruijie MAC parameter assumed unsupported (RESEARCH FINDING — PARTIAL CORRECTION)

**What goes wrong:** Design doc claims Ruijie `switchport port-security binding <IP>` (no MAC). Per Ruijie port-security manual, the full syntax is `switchport port-security binding <mac-address> [vlan <vlan-id>] <ip-address>` — MAC IS supported.

**Why it happens:** Design doc relied on user's field-collected `show port-security binding` output (which shows columns but may not show MAC in some config scenarios). Web search of Ruijie manual confirms MAC is first-class parameter.

**How to avoid:** Update design.md §3.2 row for Ruijie port_binding to: `switchport port-security binding <mac-address> [vlan <vlan-id>] <ip-address>` (full syntax). For pre-state read, the display command `show port-security binding` shows columns differently per firmware — match by IP+Interface tuple when MAC column is absent.

**Warning signs:** Pre-state NoOp detection rate <50% on Ruijie (because binding tuples include MAC that parser ignores) — debug by logging raw `show port-security binding` output for a known bound port.

### Pitfall 3: Huawei `port link-type access` universality (RESEARCH FINDING — CONFIRMED MANDATORY)

**What goes wrong:** Skipping `port link-type access` prefix assuming "V200R008+ 可省" (per design doc). Device returns `% Error: Wrong parameter` or silently ignores `port default vlan`.

**Why it happens:** Community-validated across S5700 / S5735 / S8700 — the prerequisite is universal on all S-series firmware. The design doc's "V200R008+ 可省" claim is incorrect (no Huawei official documentation supports this).

**How to avoid:** Always emit `port link-type access` in Huawei `set_access_vlan` template (matches design doc §RISK-03 mitigation). The 3-step sequence `interface X | port link-type access | port default vlan Y` is idempotent: subsequent re-execution after the port is already access-mode succeeds without side effects.

**Warning signs:** Device rejects `port default vlan` with `% Error: Please renew the default configurations` — port is in trunk/hybrid mode and requires explicit `port link-type access` first.

### Pitfall 4: VLAN ID out-of-range silent acceptance

**What goes wrong:** Service layer accepts VLAN ID 0 / 4095 (reserved per IEEE 802.1Q) without validation. Device rejects with `% Error: Invalid VLAN ID` or in worst case accepts and breaks network.

**Why it happens:** Frontend `InputNumber` with `min={1} max={4094}` only enforces client-side; backend service must re-validate to prevent bypass.

**How to avoid:** Add validator in `SetAccessVlan` service method (VLAN-05): `if vlanId < 1 || vlanId > 4094 { return nil, ErrVlanIdOutOfRange }`. Sentinel error → 400 HTTP per `execSinglePort` pattern. Also consider rejecting `vlanId == 1` (default VLAN, may have side effects) and `vlanId in [1002-1005]` (Cisco reserved for FDDI/Token Ring legacy) — but this varies per vendor; document as WARNING in HUMAN-UAT, not strict rejection.

**Warning signs:** Audit table shows `vlanId=0` or `vlanId=4095` rows from successful operations — device accepted the value but it's network-invalid.

### Pitfall 5: IP/MAC regex false-positive acceptance

**What goes wrong:** Regex matches `0.0.0.0` / `255.255.255.255` (broadcast/reserved IPs) or `00:00:00:00:00:00` (null MAC). Device rejects or, worse, accepts and binds to invalid endpoint.

**Why it happens:** Common IP regex `^(\d{1,3}\.){3}\d{1,3}$` validates octet format but not value range. Common MAC regex `^[0-9A-Fa-f]{12}$` validates length but not "non-zero".

**How to avoid:** Use strict regexes in service layer:

- **IPv4**: `^(([1-9]?\d|1\d\d|2[0-4]\d|25[0-5])\.){3}([1-9]?\d|1\d\d|2[0-4]\d|25[0-5])$` (rejects 0.x.x.x and 255.x.x.x ranges; allows 0.0.0.0 to be rejected as "host min" — design choice to disallow)
- **MAC**: `^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$` (canonical colon format) OR `^[0-9A-Fa-f]{12}$` (no separator). After normalization, validate `mac != "00:00:00:00:00:00"` to reject null MAC.

**Warning signs:** Audit table shows `ipAddress=0.0.0.0` rows from successful operations — operator input validation skipped.

### Pitfall 6: Pre-state SSH read adds latency

**What goes wrong:** For `port_binding` action, pre-state check requires fresh SSH read of binding table (`display user-bind static all` / `show port-security binding`). On devices with 1000+ bindings, this read takes 3-5 seconds — adding latency to every write.

**Why it happens:** Unlike `set_access_vlan` (which reuses cached `DevicePortStatus.VLAN` field), `port_binding` has no DB model.

**How to avoid:** Three options ranked by complexity:
1. **Skip pre-state for `port_binding`** (MVP) — always execute the write; rely on device-side duplicate detection. Trade-off: extra SSH round-trip waste if binding already exists.
2. **Implement lightweight pre-state** — parse only the target IP/interface line, exit early on match. Sub-second parse.
3. **Cache binding table per device** — TTL 30s in Redis. Overkill for MVP.

**Recommended for Phase 56:** Option 1 (skip pre-state for port_binding). Document as known limitation in 56-HUMAN-UAT.md. Re-evaluate in v1.20.x if real-device UAT shows high duplicate-binding frequency.

### Pitfall 7: MAC format divergence across vendors

**What goes wrong:** Renderer passes MAC in `AA:BB:CC:DD:EE:FF` format to all 3 vendors. Huawei/H3C accept it but expect `AABB-CCDD-EEFF` (hyphenated). Ruijie rejects it, expects `AABB.CCDD.EEFF` (Cisco dot-style).

**Why it happens:** Each vendor's command parser expects its native format. Canonical colon-format MAC is for internal storage / display only.

**How to avoid:** Convert MAC format per-vendor in render function before sprintf:
- Huawei: `AA-BB-CC-DD-EE-FF` (replace `:` with `-`)
- H3C: `AA-BB-CC-DD-EE-FF` (same as Huawei — shared VRP heritage)
- Ruijie: `AABB.CCDD.EEFF` (replace `:` with `.`, strip dashes)

Use `NormalizeMACAddress` from `mac_normalize.go` to get canonical form first, then transform.

**Warning signs:** Huawei `% Error: Invalid MAC address format`; Ruijie `% Unknown command` after `switchport port-security binding AA:BB:CC:DD:EE:FF`.

### Pitfall 8: Audit JSONB size bloat for port_binding tuples

**What goes wrong:** `port_binding` after_value contains full MAC + IP + interface tuple on every write. Audit table grows faster than v1.19 baseline.

**Why it happens:** Each binding write adds 1 JSON row with ~150 bytes payload (vs ~50 bytes for shutdown/description).

**How to avoid:** Document expected audit row size in 56-HUMAN-UAT.md. Not a blocker for MVP — operator dashboard can filter by `Action=port_binding` to scope queries.

---

## Code Examples

### 1. Service Method (Pattern: SetAccessVlan)

```go
// Source: v1.19 port_write_service.go:135 pattern + VLAN validator

func (s *portWriteServiceImpl) SetAccessVlan(
    ctx context.Context, portID string, vlanId int, operator string,
) (*PortResult, error) {
    // VLAN-05: service-layer validator (in addition to frontend InputNumber min/max)
    if vlanId < 1 || vlanId > 4094 {
        return nil, fmt.Errorf("%w: %d (must be 1-4094)", ErrVlanIdOutOfRange, vlanId)
    }
    return s.writeAndRefresh(ctx, portID, portcollection.ActionSetAccessVlan,
        vlanId, "", operator)  // vlanId passed via desc-string slot OR new param
}
```

**Design question** — `writeAndRefresh(ctx, portID, action, desc, operator)` currently uses `desc` as the 4th param (string). To accommodate VLAN ID (int) and binding tuple (multi-field), the signature must change. **Two options:**
- (a) Pass `vlanId` as int via new param: `writeAndRefresh(ctx, portID, action, payload any, operator)` — generic `payload` interface{}. Less type-safe but reusable.
- (b) Add new methods `writeAndRefreshVlan(ctx, portID, action, vlanId int, ...)` and `writeAndRefreshBinding(ctx, portID, action, op, ip, mac string, ...)`. More type-safe, more code duplication.

**Recommendation:** Option (a) with `payload any` — wrapper extracts in `executeWrite` via action-specific switch. Maintains DRY principle.

### 2. Service Method (Pattern: PortBinding)

```go
// Source: v1.19 service.go + IP/MAC validators

func (s *portWriteServiceImpl) PortBinding(
    ctx context.Context, portID string, op string, ipAddress string, macAddress string, operator string,
) (*PortResult, error) {
    // BIND-07: service-layer validators
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
    return s.writeAndRefresh(ctx, portID, portcollection.ActionPortBinding,
        bindPayload{Op: op, IP: ipAddress, MAC: macAddress}, operator)
}

type bindPayload struct {
    Op  string
    IP  string
    MAC string
}
```

### 3. Frontend Type Definition

```typescript
// Source: v1.19 types/network.ts:282 PortWriteAction + BatchWriteRequest pattern

/**
 * 端口写操作 action 字面量联合类型 (Phase 53 + 56)
 * v1.20.1: 新增 set_access_vlan + port_binding
 */
export type PortWriteAction =
  | "shutdown" | "undo_shutdown" | "description" | "dot1x_enable" | "dot1x_disable"
  | "set_access_vlan" | "port_binding";

/**
 * 修改 access VLAN 请求 (v1.20.1 VLAN-01)
 */
export interface SetAccessVlanRequest {
  portId: string;
  vlanId: number;        // 1-4094 (前端 InputNumber min/max + 后端 service 校验)
  reason: string;
}

/**
 * 端口绑定请求 (v1.20.1 BIND-01/02)
 */
export interface PortBindingRequest {
  portId: string;
  op: "add" | "remove";
  ipAddress: string;     // IPv4 regex
  macAddress?: string;   // MAC regex (optional, Huawei/H3C 接受; Ruijie 显示但不使用)
  reason: string;
}
```

### 4. Frontend API Wrapper

```typescript
// Source: v1.19 networkApi.ts:263 writeShutdown pattern (kebab-aligned)

/**
 * 修改端口 access VLAN (set_access_vlan)
 * - 端点: POST /network/ports/write/set-access-vlan
 */
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

/**
 * 端口绑定 (port_binding)
 * - 端点: POST /network/ports/write/port-binding
 * - op=add 创建静态绑定; op=remove 删除已有绑定
 */
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

### 5. Frontend SetAccessVlanModal (skeleton)

```tsx
// Source: v1.19 PortWriteModal.tsx:99-238 pattern + new VLAN ID field

import { Modal, Form, InputNumber, Select } from "antd";
import { writeSetAccessVlan } from "@/lib/api/networkApi";
import {
  PRESET_REASONS, REASON_MIN, REASON_MAX,
  REASON_CUSTOM_SENTINEL, composeReason, validateReasonRequired,
} from "./constants";
import { showAuditLinkToast } from "./PortWriteModal";

export function SetAccessVlanModal({ open, portRecord, onClose, onSuccess }) {
  const [form] = Form.useForm();
  const { message } = App.useApp();
  const navigate = useNavigate();
  const [submitting, setSubmitting] = useState(false);

  const handleOk = async () => {
    setSubmitting(true);
    try {
      const values = await form.validateFields();
      const reason = composeReason(values.reasonSelect, values.reasonText);
      await writeSetAccessVlan(portRecord.id, values.vlanId, reason ?? "");
      showAuditLinkToast(message, navigate);
      form.resetFields();
      onSuccess();
      onClose();
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal title={`修改 access VLAN - ${portRecord?.interfaceName}`} open={open}
      onOk={handleOk} onCancel={onClose} destroyOnHidden width={520}
      okText="确认执行" cancelText="取消" okButtonProps={{ loading: submitting }}>
      <Form form={form} layout="vertical" initialValues={{ vlanId: portRecord?.vlan ?? 1 }}>
        <Form.Item name="vlanId" label="VLAN ID" rules={[
          { required: true, message: "请输入 VLAN ID" },
          { type: "number", min: 1, max: 4094, message: "VLAN ID 必须在 1-4094 之间" },
        ]}>
          <InputNumber min={1} max={4094} step={1} style={{ width: "100%" }} />
        </Form.Item>
        {/* reasonSelect + reasonText + __custom__ sentinel — 复用 v1.19 constants */}
        <Form.Item name="reasonSelect" label="操作原因" rules={[{
          required: true,
          validator: (rule, value) => validateReasonRequired(rule, value, form),
        }]}>
          <Select placeholder="请选择操作原因"
            options={PRESET_REASONS.map(opt => ({ label: opt.label, value: opt.value }))} />
        </Form.Item>
        {/* ... reasonText TextArea (mirror PortWriteModal.tsx:217-234) ... */}
      </Form>
    </Modal>
  );
}
```

---

## State of the Art

| Old Approach (v1.19) | Current Approach (v1.20.1) | Source | Impact |
|----------------------|----------------------------|--------|--------|
| 5 actions (shutdown/undo/description/dot1x_enable/dot1x_disable) | 7 actions (+set_access_vlan, +port_binding) | design.md §2.1 | +2 vendor commands; same DRY pattern |
| Single `desc` payload in writeAndRefresh | Generic `payload any` for type safety OR action-specific methods | trade-off | Decision required (see Pitfall 1 above) |
| Pre-state from DB only | Pre-state from DB (VLAN) + SSH read (binding table) | new SSH round-trip for port_binding | Acceptable trade-off; skip pre-state if SSH latency is concern (Pitfall 6) |
| Huawei `port default vlan X` (no prefix) | Huawei `port link-type access | port default vlan X` | RISK-03 confirmed universal | Idempotent — already-access ports unaffected |
| Single permission `network:port:write` | Same permission covers 7 actions | design.md §5 | Zero perm change |
| operlog OperTypeStatus/Update/Batch for 5 actions | Same Status/Update/Create/Delete/Batch for 7 actions | design.md §6 | +Create (port_binding add) +Delete (port_binding remove) — both constants exist (Phase 34 lock) |

**Deprecated/outdated:**
- (none — v1.20.1 is purely additive)

---

## Assumptions Log

> Claims tagged `[ASSUMED]` need user confirmation before locking decisions.

| # | Claim | Section | Risk if Wrong | Verification Status |
|---|-------|---------|---------------|---------------------|
| A1 | H3C uses `user-bind` (no `static`) per H3C Comware V7 manual | Pitfall 1 / Code Examples | H3C template fails; binding silently rejected by device | `[VERIFIED: h3c.com manual]` — H3C's `user-bind ip-address X mac-address Y` (no `static`) confirmed; Huawei uses `user-bind static` (separate) |
| A2 | Ruijie `switchport port-security binding <mac-address> [vlan <vlan-id>] <ip-address>` supports MAC | Pitfall 2 | If actually `binding <ip>` only, MAC is silently dropped on Ruijie (still works for IP-only binding) | `[VERIFIED: Ruijie port-security manual]` — full syntax with MAC is documented; user field-collected output may have been MAC-less due to specific binding configuration |
| A3 | Huawei `port link-type access` is mandatory before `port default vlan` on all firmware | Pitfall 3 | If newer firmware accepts `port default vlan` without prefix, our template has redundant line (no functional impact) | `[VERIFIED: S5700/S5735 community docs]` — universal prerequisite across S-series firmware versions |
| A4 | VLAN ID range 1-4094 inclusive (excludes 0 and 4095) | Pitfall 4 | If 0/4095 are valid in some vendor, our validator rejects them | `[VERIFIED: IEEE 802.1Q]` — 0 and 4095 reserved; 1-4094 is user-assignable |
| A5 | VLAN 1 is default VLAN but assignable; VLANs 1002-1005 reserved on Cisco-style but may be assignable on Huawei/H3C/Ruijie | Pitfall 4 | Rejecting 1002-1005 blocks valid configs on some vendors | `[ASSUMED]` — varies per vendor firmware; recommend client-side WARNING not server-side REJECT for 1002-1005 |
| A6 | `deviceExecutor.ExecuteCustom` is the only SSH entry; reuse for both new actions | Don't Hand-Roll | If a different SSH pattern is needed (e.g., different transport options), our extension breaks DRY | `[VERIFIED: v1.19 batch_orchestrator.go + port_write_service.go:71-73]` — `portWriteExecutor` interface locked |
| A7 | Pre-state check for `port_binding` requires fresh SSH read (no DB model) | Pitfall 6 | If we add a binding DB model later, pre-state can move to DB layer | `[ASSUMED]` — current `DevicePortStatus` model has no `bindings` field; per design.md §4.2, SSH read is the pre-state approach |
| A8 | `set_access_vlan` can reuse cached `DevicePortStatus.VLAN` field | Pitfall 6 / Pattern 2 | If `VLAN` field is not consistently populated by collectors, pre-state returns stale data | `[VERIFIED: types/network.ts:125 vlan?: number]` — field exists in TS type; needs confirmation that backend `sys_device_port_status` has equivalent column populated by port-collection cron |
| A9 | `BindOp` is "add" or "remove" string literal | Code Examples | If frontend uses different vocabulary (e.g., "create"/"delete"), UI/handler mismatch | `[VERIFIED: design.md §2.1]` — `op: "add"|"remove"` |
| A10 | `BatchWriteRequest.Action` reuses `ActionSetAccessVLAN` and `ActionPortBinding` for batch path | Pattern 4 / BATCH path | If batch path needs different action constants, separate code paths required | `[ASSUMED]` — design.md §4.2 says reuse batch endpoint with same action values; verify with batch_orchestrator.go switch coverage |

**Total assumptions:** 10 (4 vendor-syntax, 2 vendor-firmware, 2 architecture-reuse, 2 data-model)

---

## Open Questions

1. **H3C user-bind sub-mode (VLAN required?)**
   - What we know: H3C `user-bind` command allows `ip-address X mac-address Y vlan Z interface W` — vlan-id parameter is documented as optional in some H3C manuals.
   - What's unclear: Whether H3C **requires** vlan-id for port-binding in 1:1 mode (vs IP-source-guard mode). The design doc doesn't include vlan-id field.
   - Recommendation: For MVP, omit vlan-id field. Document in 56-HUMAN-UAT.md as "verify H3C accepts binding without vlan-id; if rejected, add vlan-id field in v1.20.x".

2. **Huawei V200R022+ command behavior change**
   - What we know: Design doc's "V200R008+ 可省" claim is unverified.
   - What's unclear: Whether Huawei V200R022+ (S8700 firmware in production) accepts `port default vlan` without `port link-type access` prefix.
   - Recommendation: Always emit prefix (Pitfall 3 mitigation). No-op on newer firmware.

3. **Pre-state check for port_binding: skip or include?**
   - What we know: SSH read of binding table adds 3-5s latency per write.
   - What's unclear: User tolerance for latency vs. duplicate-write prevention.
   - Recommendation: Per Pitfall 6, skip pre-state for `port_binding` in MVP. Add `display-binding` SSH call only if `pre_state_check.go` detects cached binding table (e.g., `deviceBindings` Redis key with 30s TTL). Defer full implementation to v1.20.x.

4. **Frontend InputNumber for VLAN ID: show keyboard hint?**
   - What we know: Ant Design InputNumber supports keyboard up/down arrows.
   - What's unclear: Whether to add explicit hint text "(范围 1-4094)" near field label.
   - Recommendation: Yes, add hint via `Form.Item extra="范围 1-4094 (VLAN 0/4095 保留)"` — surfaces validation rules at input time.

---

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Backend build + tests | ✓ | 1.24 | — |
| scrapligo | SSH transport (reuse v1.19) | ✓ | v1.3.3 | — |
| PostgreSQL/SQLite | e2e test DB | ✓ | both via GORM | — |
| scrapligo FileTransport | e2e tests (replay fixtures) | ✓ | (in scrapligo) | Real SSH (deferred to HUMAN-UAT) |
| React + antd | frontend build | ✓ | 19.2 / 6.1 | — |
| Huawei S8700 (production) | Real-device UAT | ✗ (production only, no local) | — | Site-visit by 现场运维同事 |
| H3C device (production) | Real-device UAT | ✗ (no production H3C access from this env) | — | Site-visit OR deferred (per design.md §10.4) |
| Ruijie RS8607E (production) | Real-device UAT | ✗ (production only) | — | Site-visit by 现场运维同事 |

**Missing dependencies with no fallback (require human action):**
- Real-device SSH verification for 3 vendors — owner = 现场运维同事; deferral doc `56-HUMAN-UAT.md` to be created in W5.

**Missing dependencies with fallback:**
- H3C physical device — per design.md §10.4 "if production has H3C, supplement; else deferred". Acceptable to defer to v1.20.x if no H3C found.

---

## Validation Architecture

> Per `config.json`, `workflow.nyquist_validation` is enabled (absent = enabled).

### Test Framework
| Property | Value |
|----------|-------|
| Framework (Go) | `testing` + `testify/assert` + `testify/mock` |
| Framework (Frontend) | `vitest` (existing) — Phase 53/56 don't add frontend unit tests beyond type-check |
| Config file | None new — reuses v1.19 layout |
| Quick run command (Go) | `go test ./internal/services/portcollection/ ./internal/services/portwrite/ -count=1` |
| Full suite command (Go) | `go test ./...` |
| Quick run command (Frontend) | `npm run type-check` (frontend e2e is out of scope per FUTURE-OP-12) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| VLAN-02 | Huawei `port link-type access` + `port default vlan X` renders correctly | unit | `go test ./internal/services/portcollection/ -run TestRenderCommand_VendorActionMatrix/huawei_set_access_vlan` | ❌ Wave 0 (W1 creates) |
| VLAN-02 | H3C `port access vlan X` renders correctly | unit | `go test ./internal/services/portcollection/ -run TestRenderCommand_VendorActionMatrix/h3c_set_access_vlan` | ❌ Wave 0 |
| VLAN-02 | Ruijie `switchport mode access` + `switchport access vlan X` | unit | `go test ./internal/services/portcollection/ -run TestRenderCommand_VendorActionMatrix/ruijie_set_access_vlan` | ❌ Wave 0 |
| BIND-03 | Huawei `user-bind static ip-address X [mac-address Y] interface Z` | unit | `go test ./internal/services/portcollection/ -run TestRenderCommand_VendorActionMatrix/huawei_port_binding_add` | ❌ Wave 0 |
| BIND-03 | H3C `user-bind ip-address X [mac-address Y]` (no `static`) | unit | `go test ./internal/services/portcollection/ -run TestRenderCommand_VendorActionMatrix/h3c_port_binding_add` | ❌ Wave 0 |
| BIND-03 | Ruijie `switchport port-security binding <mac> <ip>` | unit | `go test ./internal/services/portcollection/ -run TestRenderCommand_VendorActionMatrix/ruijie_port_binding_add` | ❌ Wave 0 |
| VLAN-03 | Pre-state check PVID == target → NoOp | unit | `go test ./internal/services/portwrite/ -run TestCheckPreState_SetAccessVlan` | ❌ Wave 0 (W2 creates) |
| BIND-05 | Pre-state check binding exists → NoOp | unit | `go test ./internal/services/portwrite/ -run TestCheckPreState_PortBinding` | ❌ Wave 0 |
| VLAN-05 | Service rejects VLAN ID < 1 or > 4094 | unit | `go test ./internal/services/portwrite/ -run TestSetAccessVlan_Validation` | ❌ Wave 0 |
| BIND-07 | Service rejects invalid IP / MAC | unit | `go test ./internal/services/portwrite/ -run TestPortBinding_Validation` | ❌ Wave 0 |
| TEST-03 | E2E scrapligo FileTransport for set_access_vlan | e2e | `go test ./internal/services/portwrite/ -run TestE2E_SetAccessVlan` | ❌ Wave 0 (W5 creates) |
| TEST-03 | E2E for port_binding add/remove | e2e | `go test ./internal/services/portwrite/ -run TestE2E_PortBinding` | ❌ Wave 0 |
| INFRA-03 | operlog OperType mapping Update/Create/Delete | regression | `go test ./internal/utils/operlog/ -run TestOperTypeConstantStability` | ✓ (regression_test.go exists) |
| INFRA-04 | Permission `network:port:write` covers 7 actions | smoke | `go test ./internal/api/v1/network/ -run TestSetupPortWriteRouter_RequirePermissions2Arg` | ✓ (v1.19 test exists) |

### Sampling Rate
- **Per task commit:** `go test ./internal/services/portcollection/ ./internal/services/portwrite/ -count=1` (full suite for the changed packages)
- **Per wave merge:** `go build ./... && go test ./... && npm run type-check && npm run build`
- **Phase gate:** Full suite green + operlog regression + vendor-react bundle ≤ 776 kB baseline (zero regression) + 56-HUMAN-UAT.md created

### Wave 0 Gaps
- [ ] `internal/services/portcollection/vendor_port_template_test.go` — extend with 12 test cases (3 vendors × 2 actions × 2 variants)
- [ ] `internal/services/portwrite/pre_state_check_test.go` — create (new file) with 6 test cases (3 vendors × 2 actions × match/mismatch)
- [ ] `internal/services/portwrite/port_write_service_test.go` — extend with 5 test cases (VLAN validator, IP validator, MAC validator, SetAccessVlan success, PortBinding success)
- [ ] `internal/services/portwrite/port_write_e2e_test.go` — extend with 10 e2e tests
- [ ] `internal/services/portwrite/testdata/` — 6 new fixtures: huawei_set_access_vlan_success, ruijie_set_access_vlan_success, huawei_port_binding_add_success, huawei_port_binding_add_with_mac_success, huawei_port_binding_remove_success, ruijie_port_binding_add_success
- [ ] `xingran-react-frontend/src/types/network.ts` — extend with 2 type aliases
- [ ] `xingran-react-frontend/src/components/network/port-write/SetAccessVlanModal.tsx` — CREATE
- [ ] `xingran-react-frontend/src/components/network/port-write/PortBindingModal.tsx` — CREATE

*(No conftest.py / shared fixture changes needed — v1.19 infrastructure is reused as-is.)*

---

## Security Domain

> `security_enforcement` is enabled (absent = enabled) per project CLAUDE.md.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | No (reuses v1.19 JWT + perm) | (inherited) |
| V3 Session Management | No (reuses v1.19 dual-token) | (inherited) |
| V4 Access Control | Yes | `middleware.RequirePermissions(["network:port:write"], core)` — group-level, 2-arg signature (v1.19 locked) |
| V5 Input Validation | Yes | Service-layer validators: VLAN ID range (1-4094), IP regex, MAC regex. Frontend InputNumber min/max as UX hint, not security boundary. |
| V6 Cryptography | No (no new crypto) | (inherited SM2+SM4 request encryption for write endpoints) |
| V7 Error Handling | Yes | Sentinel errors → HTTP 4xx per `execSinglePort` pattern (no internal stack trace leakage) |
| V8 Data Protection | Yes (audit JSONB may contain IP/MAC) | IP address and MAC address are **NOT sensitive** under project classification — they are public network identifiers. Operlog RecordWithoutBody used (no RecordWithBody) since no passwords/keys/tokens in payload. |
| V9 Communication | No (reuses v1.19) | (inherited) |
| V10 Malicious Code | N/A | (inherited) |
| V11 Business Logic | Yes | OperType mapping (Create/Update/Delete) matches business semantics; pre-state check prevents redundant writes; fail-fast batch prevents resource exhaustion |
| V12 Files and Resources | N/A | No new file uploads |

### Known Threat Patterns for v1.20.1

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Invalid VLAN ID injection (e.g., `0`, `4095`, negative) | Tampering | Service-layer validator rejects with sentinel error → 400 |
| IP injection in binding command (e.g., `10.0.0.1; reboot`) | Tampering | Strict IPv4 regex blocks command separator characters |
| MAC injection (e.g., `AA:BB:CC:DD:EE:FF;quit`) | Tampering | Strict MAC regex (canonical colon-format only); normalize to canonical form before command construction |
| Privilege escalation via binding (binding all ports to admin IP) | Elevation of Privilege | `network:port:write` perm required; ACL on roles unchanged |
| Audit log tampering (forging binding add/remove records) | Repudiation | OperType Create/Delete audit; oper_param carries device_name + interface_name for human-readable audit; CRC-free (rely on DB permissions) |
| Pre-state check DoS (forcing repeated binding-table reads) | Denial of Service | Pitfall 6 mitigation: skip pre-state for port_binding in MVP |

---

## Sources

### Primary (HIGH confidence)
- v1.19 internal codebase: `internal/services/portcollection/vendor_port_template.go`, `internal/services/portwrite/{port_write_service,batch_orchestrator,pre_state_check,parse_error}.go`, `internal/api/v1/network/port_write_handler.go`, `internal/api/v1/network/port_write_router.go` — **direct file reads verified v1.19 patterns**
- `internal/services/mac_normalize.go` — verified `NormalizeMACAddress()` returns colon format (project memory `mac-address-normalize-returns-colon-format` aligns)
- `internal/utils/operlog/regression_test.go` — verified 25 OperType constants + 11 sensitive keywords + 5-param Record signature
- `.planning/milestones/v1.19-MILESTONE-AUDIT.md` — verified 37/37 v1.19 requirements PASSED
- `.planning/milestones/v1.19-REQUIREMENTS.md` — verified locked decision patterns (CONV-01..04, PERM-01..03)
- `docs/plans/2026-07-09-v1.20.1-design.md` — verified v1.20.1 design decisions + RISK items

### Secondary (MEDIUM confidence)
- H3C official documentation (`h3c.com/cn/d_202409/2262994_30005_0.htm`) — verified `user-bind` keyword (no `static`) on Comware V7
- H3C IP Source Guard commands reference (`h3c.com/cn/d_201108/723303_30005_0.htm`) — verified earlier Comware versions
- Huawei CX320 Switch Module V100R001 Command Reference (`support.huawei.com/enterprise/en/doc/EDOC1000128405/fe1387b2/display-user-bind-static`) — confirmed Huawei uses `user-bind static` (WebFetch failed; URL cited from search snippet)
- Huawei S5700/S5735 community docs — verified `port link-type access` mandatory prerequisite
- Ruijie port-security configuration manuals — verified `switchport port-security binding <mac-address> [vlan <vlan-id>] <ip-address>` syntax

### Tertiary (LOW confidence — requires real-device UAT confirmation)
- H3C `user-bind` accepts binding without `vlan-id` (varies per Comware version)
- Ruijie `show port-security binding` output columns (varies per RGOS version; user-collected output showed limited columns)
- Huawei V200R022+ behavior on `port default vlan` without `port link-type access` prefix (likely still requires prefix per universal S-series pattern)
- Reserved VLAN behavior: VLAN 1 (assignable, but default), VLAN 1002-1005 (varies per vendor; Cisco reserves for legacy FDDI/Token Ring)

---

## Metadata

**Confidence breakdown:**
- **Standard Stack:** HIGH — all libraries are v1.19 inherited, no new dependencies
- **Architecture:** HIGH — 5-wave pattern reused verbatim from v1.19, file inventory confirmed
- **Vendor Command Syntax:** MEDIUM-HIGH — H3C `user-bind` (no static) verified via H3C official; Huawei prefix verified via community; Ruijie MAC syntax verified via Ruijie manual; real-device UAT deferred for cross-firmware variations
- **Pitfalls:** MEDIUM — design doc RISK items partially resolved via web search; remaining items (H3C vlan-id optional, Ruijie MAC display columns, Huawei newer firmware) require site-visit UAT
- **Frontend:** HIGH — pattern locked from v1.19 PortWriteModal + BulkWriteDrawer + ACTION_TITLE; new Modals are direct extensions
- **Validation:** HIGH — test framework + sample commands verified; Wave 0 gaps explicitly listed

**Research date:** 2026-07-09
**Valid until:** 2026-08-09 (30 days; vendor firmware evolution may invalidate some claims if vendors release major updates)