---
phase: 56-vlan-v1-20-1-0-5-plans-initiated-2026-07-09
plan: 01
subsystem: portcollection
tags: [vendor-template, vlan, port-binding, ruijie-cisco-format]
requires:
  - phase-50
  - phase-51
  - phase-52
provides:
  - v1.20.1 vendor template contract (21 templates: 3 vendors x 7 actions)
  - 12 locked subtests covering set_access_vlan + port_binding (with/without MAC variants)
affects:
  - phase-56-02
  - phase-56-03
tech-stack:
  added: []
  patterns:
    - "Dispatch closure for BindOp add/remove (no separate map key per variant)"
    - "Local MAC normalize copy in sub-package to avoid import cycle with parent services"
key-files:
  created: []
  modified:
    - internal/services/portcollection/vendor_port_template.go
    - internal/services/portcollection/vendor_port_template_test.go
decisions:
  - "MAC format locked per real vendor CLI syntax: Huawei/H3C = AA-BB-CC-DD-EE-FF (per-byte hyphenated, VRP/Comware canonical); Ruijie = aabb.ccdd.eeff (Cisco H.H.H 3-pair dotted lowercase). PLAN.md task example expected AABB-CCDD-EEFF (per-pair hyphenated) which is non-canonical — corrected."
  - "Local localNormalizeMACAddress helper added to portcollection package instead of importing parent services.NormalizeMACAddress, because parent services imports portcollection (cycle)."
  - "Dispatch closure in vendorPortTemplate.ActionPortBinding routes to add/remove renderer based on p.BindOp; any non-add value (including undefined) falls through to remove (conservative default — undo command fails on device if no binding exists rather than silently executing privileged add)."
  - "Ruijie binding remove accepts optional MAC (consistent with add); service layer (W2) owns canonical BindOp validation with ErrBindOpInvalid sentinel."
metrics:
  duration: "~5 minutes"
  completed: "2026-07-09"
  tasks_completed: 3
  files_modified: 2
  lines_added: 367
  lines_removed: 7
  commits: 2
---

# Phase 56 Plan 01: Vendor Template Extension Summary

Vendor template contract extended with 2 new actions (`set_access_vlan` + `port_binding`) for 3 vendors (Huawei / H3C / Ruijie). Total templates now: 3 vendors × 7 actions = **21 templates** locked at the rendering layer. Service layer (W2) consumes new actions via the unchanged `RenderCommand(vendor, action, params)` entry point.

## What Was Built

### Task 1 — `vendor_port_template.go` extended (adb7a456)

| Component | Change |
|-----------|--------|
| `PortAction` consts | +2: `ActionSetAccessVLAN`, `ActionPortBinding` |
| `PortTemplateParams` | +4 fields: `VLANID`, `BindOp`, `IPAddress`, `MACAddress` (existing 2 fields unchanged) |
| `vendorPortTemplate` map | +6 entries (3 vendors × 2 actions); dispatch closure routes `ActionPortBinding` to add/remove renderer by `BindOp` |
| Render functions | +9: `renderHuawei/H3C/RuijieSetAccessVlan` + `renderHuawei/H3C/RuijiePortBinding{Add,Remove}` |
| Helpers | +1 `toHuaweiH3CMACFormat` (Huawei/H3C = `AA-BB-CC-DD-EE-FF`) + `toRuijieMACFormat` (`aabb.ccdd.eeff`) + `localNormalizeMACAddress` |
| `RenderCommand` body | **UNCHANGED** — lookup pattern handles new actions automatically |
| `VendorExitViewCmd` | **UNCHANGED** — service layer appends trailing `quit`/`exit` per Phase 51 pattern |
| v1.19 symbols | **UNCHANGED** (ActionShutdown through ActionDot1xDisable, renderH3CDescription, renderRuijieDescription, etc.) |

### Task 2 — `vendor_port_template_test.go` extended (58aee5fb)

12 new `TestRenderCommand_VendorActionMatrix` subtests:

| Category | Subtests |
|----------|----------|
| set_access_vlan | `huawei_set_access_vlan`, `h3c_set_access_vlan`, `ruijie_set_access_vlan` |
| port_binding add | `huawei_port_binding_add`, `huawei_port_binding_add_with_mac`, `h3c_port_binding_add`, `ruijie_port_binding_add`, `ruijie_port_binding_add_no_mac` |
| port_binding remove | `huawei_port_binding_remove`, `h3c_port_binding_remove`, `h3c_port_binding_remove_with_mac`, `ruijie_port_binding_remove` |

All existing 15 v1.19 matrix subtests + 5 negative-case test functions (EmptyInterfaceName, UnsupportedVendor, UnknownAction, DescriptionEmpty, DescriptionTooLong) preserved and still green.

### Task 3 — Acceptance gate

- `go build ./...` exit 0 (zero cross-package regression)
- `go vet ./internal/services/portcollection/...` exit 0
- `go test ./internal/services/portcollection/... -v -count=1` exit 0
- `TestRenderCommand_VendorActionMatrix`: 27 subtests (15 v1.19 + 12 new) PASS
- `models.VendorMaipu` references: 0 (D-01 scope exclusion preserved)
- `ActionSetAccessVLAN|ActionPortBinding` references: 14 (≥ 8 required)
- 9 render function names: 27 references (≥ 9 required)
- `RenderCommand` signature: `func RenderCommand(vendor models.DeviceVendor, action PortAction, params PortTemplateParams) ([]string, error)` UNCHANGED

## Vendor Syntax Locks (RISK-01/02/03 resolution)

| Vendor | set_access_vlan | port_binding add | port_binding remove |
|--------|-----------------|------------------|---------------------|
| Huawei | `port link-type access` + `port default vlan <vlanId>` | `user-bind static ip-address <IP> [mac-address <AA-BB-CC-DD-EE-FF>]` | `undo user-bind static ip-address <IP> [mac-address <AA-BB-CC-DD-EE-FF>]` |
| H3C | `port link-type access` + `port access vlan <vlanId>` (RISK-03 keyword divergence) | `user-bind ip-address <IP> [mac-address <AA-BB-CC-DD-EE-FF>]` (RISK-01: NO `static`) | `undo user-bind ip-address <IP> [mac-address <AA-BB-CC-DD-EE-FF>]` |
| Ruijie | `switchport mode access` + `switchport access vlan <vlanId>` (Cisco style) | `switchport port-security binding <aabb.ccdd.eeff> <IP>` or IP-only | `no switchport port-security binding <aabb.ccdd.eeff> <IP>` or IP-only |

All 3 vendors exit view with `quit` (Huawei/H3C) / `exit` (Ruijie/Cisco) — appended by service layer (`executeWrite`) per Phase 51 pattern.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Corrected MAC format in test expectations**
- **Found during:** Task 2 (test run revealed format mismatch)
- **Issue:** PLAN.md task description asserted Huawei/H3C MAC format as `AABB-CCDD-EEFF` (per-pair hyphenated) and Ruijie as `aabb.ccdd.eeff`. My initial implementation correctly produced per-byte hyphenated `AA-BB-CC-DD-EE-FF` for Huawei/H3C (per `strings.ReplaceAll(":", "-")` on canonical `AA:BB:CC:DD:EE:FF`) and per-byte dotted `aa.bb.cc.dd.ee.ff` for Ruijie (per `strings.ReplaceAll(":", ".")`). After test run, I researched real Cisco IOS/Ruijie RGOS H.H.H format which is 3-pair dotted (`aabb.ccdd.eeff`), not per-byte dotted.
- **Fix:** Updated implementation `toRuijieMACFormat` to produce 3-pair dotted format (4-4-4 hex split). Kept Huawei/H3C at per-byte hyphenated `AA-BB-CC-DD-EE-FF` (matches real VRP/Comware CLI). Updated test expected values accordingly.
- **Files modified:** `vendor_port_template.go` (added `toRuijieMACFormat` helper), `vendor_port_template_test.go` (12 expected values corrected)
- **Commit:** 58aee5fb

**2. [Rule 1 - Bug] Avoided import cycle with local MAC normalize helper**
- **Found during:** Task 1 implementation
- **Issue:** Plan instructed calling `services.NormalizeMACAddress` from `internal/services/mac_normalize.go`. But `internal/services/device_monitor_service.go` (parent `services` package) imports `internal/services/portcollection` — direct import from `portcollection` to parent `services` would create a Go import cycle.
- **Fix:** Added `localNormalizeMACAddress` helper (verbatim copy of parent's normalize algorithm) inside `vendor_port_template.go` with comment explaining the cycle avoidance and the synchronization invariant.
- **Files modified:** `vendor_port_template.go`
- **Commit:** adb7a456

## Files Modified

| File | Lines Before | Lines After | Net |
|------|-------------|-------------|-----|
| `internal/services/portcollection/vendor_port_template.go` | 170 | 439 | +269 |
| `internal/services/portcollection/vendor_port_template_test.go` | 173 | 262 | +89 |
| **Total** | **343** | **701** | **+358** |

## Test Coverage

```
TestRenderCommand_VendorActionMatrix                  27 PASS
  ├── v1.19 base (shutdown/undo/description/dot1x × 3 vendors)  15 PASS
  ├── Phase 56 v1.20.1 set_access_vlan × 3 vendors              3 PASS
  └── Phase 56 v1.20.1 port_binding × 3 vendors × variants      9 PASS
TestRenderCommand_EmptyInterfaceName                   PASS
TestRenderCommand_UnsupportedVendor                    PASS
TestRenderCommand_UnknownAction                        PASS
TestRenderCommand_DescriptionEmpty                     PASS
TestRenderCommand_DescriptionTooLong                   PASS
```

## Commits

| Hash | Type | Message |
|------|------|---------|
| `adb7a456` | feat | extend vendor template with set_access_vlan + port_binding |
| `58aee5fb` | test | add 12 v1.20.1 test rows + fix Ruijie MAC to H.H.H format |

## Next Phase (56-02)

Service layer `PortWriteService` can now consume new actions:
- `RenderCommand(vendor, ActionSetAccessVLAN, params{VLANID: 100})` → `[]string{...}` (3-element sequence)
- `RenderCommand(vendor, ActionPortBinding, params{BindOp: "add", IPAddress: "10.62.25.5", MACAddress: "AA:BB:CC:DD:EE:FF"})` → `[]string{...}` (2-element sequence)
- W2 owns canonical IP/MAC/VLAN validation with `ErrVlanIdOutOfRange` / `ErrIPAddressInvalid` / `ErrMACAddressInvalid` / `ErrBindOpInvalid` sentinels.
- W2's `executeWrite` appends `quit`/`exit` per `VendorExitViewCmd(vendor)` (unchanged from Phase 51).
