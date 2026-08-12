---
phase: 56-vlan-v1-20-1-0-5
reviewed: 2026-07-09T00:00:00Z
reviewed_followup: 2026-07-09T07:00:00Z
depth: standard
files_reviewed: 13
files_reviewed_list:
  - internal/services/portcollection/vendor_port_template.go
  - internal/services/portwrite/port_write_service.go
  - internal/services/portwrite/pre_state_check.go
  - internal/services/portwrite/batch_orchestrator.go
  - internal/api/v1/network/port_write_handler.go
  - internal/api/v1/network/port_write_router.go
  - pkg/permission/config.go
  - xingran-react-frontend/src/components/network/port-write/SetAccessVlanModal.tsx
  - xingran-react-frontend/src/components/network/port-write/PortBindingModal.tsx
  - xingran-react-frontend/src/components/network/port-write/constants.ts
  - xingran-react-frontend/src/lib/api/networkApi.ts
  - xingran-react-frontend/src/pages/network/ports/index.tsx
  - xingran-react-frontend/src/types/network.ts
findings:
  critical: 0
  warning: 6
  info: 5
  total: 11
fixes_applied:
  - commit: 415005be
    resolves: [CR-01, CR-04]
    summary: "share 4 v1.20.1 validators (validateVlanIdRange/validateBindOp/validateIPAddress/validateMACAddress) across SetAccessVlan/PortBinding + BatchWritePorts; add gin binding tags to BatchWriteRequest"
  - commit: 2460f65e
    resolves: [CR-02]
    summary: "include vlan field in buildBeforeValue for ActionSetAccessVLAN (skipped path audit completeness)"
  - commit: 58692dfb
    resolves: [CR-03, WR-05]
    summary: "SetAccessVlanModal + PortBindingModal use form.setFieldsValue on open to preserve vlanId pre-fill"
status: clean
---

# Phase 56: Code Review Report

**Reviewed:** 2026-07-09
**Depth:** standard
**Files Reviewed:** 13
**Status:** issues_found

## Summary

Phase 56 (v1.20.1 网络设备 VLAN + 端口绑定) extends the existing v1.19 port-write subsystem with 2 new actions (`set_access_vlan` + `port_binding`), spanning 3 vendor templates × 6 new renderers, 2 new service methods with sentinel validators, 2 new HTTP handlers (kebab routes), permission registry entries, and 2 new React Modals.

The single-port path (`SetAccessVlan` / `PortBinding` service methods) is well-constructed: sentinel validators run **before** any DB/SSH traffic, Extra map cleanly carries after_value to the audit layer, MAC normalization is duplicated locally to avoid the import cycle, and the IPv4 regex blocks shell metacharacters via its character class. The frontend Modals honor LANDMINE #5 (no try/catch in wrappers, no `message.error` in components) and use stable useEffect dependencies.

However, the **batch path has 4 BLOCKER-grade defects** that ship a security hole and incomplete audit coverage:

1. **CR-01 (BLOCKER)**: `BatchWritePorts` bypasses ALL 4 v1.20.1 validators — a batch request with `ipAddress=";reboot"` or `vlanId=99999` reaches the device CLI with no regex/range/op check. This is the most severe finding: the renderer does `fmt.Sprintf("ip-address %s", p.IPAddress)` with the raw input, so shell/command injection vectors that the single-port path was specifically designed to block (`ErrIPAddressInvalid`) are wide open via `/network/ports/write/batch`.
2. **CR-02 (BLOCKER)**: `buildBeforeValue` omits the `vlan` field. For `set_access_vlan` skipped (NoOp, pre-state match) operations, `buildAuditRow` overrides `after_value` with `before_value` — losing the very field being audited. Phase 56's core audit guarantee ("after_value via Extra" from commit 271444e1) is broken on the skipped path.
3. **CR-03 (BLOCKER)**: `SetAccessVlanModal` calls `form.resetFields()` in a `useEffect` on open, which **clears** the vlanId field to `undefined`. Combined with `destroyOnHidden` (Form remounts on each open), the user sees an empty VLAN ID field instead of the pre-filled current port vlan (the `initialValues={{ vlanId: portRecord?.vlan ?? 1 }}` is consulted at mount time but then immediately reset).
4. **CR-04 (BLOCKER)**: `SetAccessVlanRequest.Reason` is `omitempty` + no binding tag, but the frontend always sends `reason: ""`. `composeReason` returns `null` when the user selects nothing, and the Modal passes `reason ?? ""`. The backend `PortBindingRequest` declares `oneof=add remove` on Op but the batch `BatchWriteRequest.BindOp` has **no** binding tag, so batch requests with `op="invalid"` silently fall through the "conservative default → remove" branch in the renderer (line 73-78 of vendor_port_template.go) and delete bindings the user never asked to delete.

Additional warnings cover dead sentinel errors (`ErrMixedDevices`/`ErrPortNotFound` are never emitted), misleading IPv4 regex comment, and a frontend `useEffect` dependency drift.

## Critical Issues

### CR-01: Batch path bypasses all 4 v1.20.1 validators — command injection vector to device CLI

**File:** `internal/services/portwrite/batch_orchestrator.go:59-164` + `internal/services/portwrite/port_write_service.go:127,149`
**Issue:**
The single-port methods `SetAccessVlan` (lines 192-209) and `PortBinding` (lines 222-254) enforce 4 sentinel validators (`ErrVlanIdOutOfRange`, `ErrBindOpInvalid`, `ErrIPAddressInvalid`, `ErrMACAddressInvalid`) **before** any DB query or SSH traffic. This is the security boundary.

But `BatchWritePorts` does NOT call `SetAccessVlan` / `PortBinding` — it calls `executeWrite` directly (batch_orchestrator.go:127 and :149). The `BatchWriteRequest` struct (port_write_service.go:77-88) has only JSON tags — **no** `binding:"min=1,max=4094"` for VLANID, **no** `oneof=add remove` for BindOp, **no** regex for IPAddress.

Consequence — three concrete attack/error scenarios that the single-port path blocks but batch path allows:

1. **Command injection**: `POST /network/ports/write/batch` with `{action:"port_binding", bindOp:"add", ipAddress:"10.0.0.1;reboot", portIds:[...]}` — the renderer at `vendor_port_template.go:319` does `fmt.Sprintf("ip-address %s", p.IPAddress)` and the resulting device CLI command `user-bind static ip-address 10.0.0.1;reboot` is sent to the network device. The single-port path's IPv4 regex (`^(([1-9]?\d|1\d\d|2[0-4]\d|25[0-5])\.){3}...`) was specifically designed to block this — but batch makes it reachable.
2. **Silent wrong-op**: `{action:"port_binding", bindOp:"invalid-string", ipAddress:"10.0.0.1"}` — the renderer's conservative default at vendor_port_template.go:73-78 (`if p.BindOp == "add" {...} return renderHuaweiPortBindingRemove(p)`) silently **deletes** an existing binding instead of erroring. The user expected a 400; they get a destructive remove.
3. **VLAN range overflow**: `{action:"set_access_vlan", vlanId:99999}` — the only catch is the renderer's defensive check at vendor_port_template.go:268 (`if p.VLANID < 1 || p.VLANID > 4094`), which returns a plain `fmt.Errorf` (not a sentinel), so the handler's sentinel→HTTP-400 switch (port_write_handler.go:254-265) does not match and the error escalates to HTTP 500 instead of 400. Worse, this error is wrapped inside the batch loop and becomes part of `result.Failed[]`, so it returns 200 with a single-port failure rather than rejecting the whole request at the entry.

**Fix:**
Add an entry validator to `BatchWritePorts` that runs the same 4 checks before any DB/SSH traffic. The cleanest fix is to factor the validators into a helper and call from both single-port and batch entry:

```go
// In port_write_service.go, factor a shared validator
func validateActionParams(action portcollection.PortAction, vlanId int, bindOp, ipAddress, macAddress string) error {
    if action == portcollection.ActionSetAccessVLAN {
        if vlanId < 1 || vlanId > 4094 {
            return fmt.Errorf("%w: %d", ErrVlanIdOutOfRange, vlanId)
        }
    }
    if action == portcollection.ActionPortBinding {
        if bindOp != "add" && bindOp != "remove" {
            return fmt.Errorf("%w: %q", ErrBindOpInvalid, bindOp)
        }
        if !ipv4Pattern.MatchString(ipAddress) {
            return fmt.Errorf("%w: %q", ErrIPAddressInvalid, ipAddress)
        }
        if macAddress != "" {
            normalized := services.NormalizeMACAddress(macAddress)
            if normalized == "" || normalized == "00:00:00:00:00:00" {
                return fmt.Errorf("%w: %q", ErrMACAddressInvalid, macAddress)
            }
        }
    }
    return nil
}

// In BatchWritePorts (batch_orchestrator.go:73, after the existing 3 checks):
if vErr := validateActionParams(req.Action, req.VLANID, req.BindOp, req.IPAddress, req.MACAddress); vErr != nil {
    return nil, vErr
}
```

Then both the single-port methods and batch share one validator. Add binding tags as defense-in-depth:

```go
type BatchWriteRequest struct {
    DeviceID    string   `json:"deviceId" binding:"required"`
    Action      Action   `json:"action" binding:"required"`
    PortIDs     []string `json:"portIds" binding:"required,min=1,max=50"`
    VLANID int    `json:"vlanId,omitempty" binding:"omitempty,min=1,max=4094"`
    BindOp string `json:"bindOp,omitempty" binding:"omitempty,oneof=add remove"`
    // IPAddress / MACAddress cannot use gin binding tags for regex; rely on service validator.
}
```

Add tests: `TestBatchWritePorts_BindingCmdInjectionRejected`, `TestBatchWritePorts_VlanOutOfRangeRejected`, `TestBatchWritePorts_BadBindOpRejected`.

---

### CR-02: buildBeforeValue omits `vlan` field — set_access_vlan skipped-path audit loses the audited field

**File:** `internal/api/v1/network/port_write_handler.go:405-413` + `:491-495`
**Issue:**
`buildBeforeValue` marshals only `{admin_status, dot1x_enabled, description, interface_name}` — it does NOT include `port.VLAN`. For `set_access_vlan` operations this is the central field being changed.

`buildAuditRow` (line 491-495) has this override:
```go
afterValue := buildAfterValue(pr.Action, pr)
if pr.Status == "skipped" {
    afterValue = beforeValue
}
```

For a **skipped** `set_access_vlan` operation (port already at target VLAN, NoOp short-circuit via `pre_state_check.go:96-104`):
- `before_value` = `{"admin_status":"up","dot1x_enabled":false,"description":"...","interface_name":"GE1/0/1"}` — no vlan
- `after_value` = before_value (override) — no vlan

So the audit row for a set_access_vlan NoOp shows zero VLAN information on either side. The Phase 56 commit 271444e1 ("audit after_value via Extra") explicitly framed the Extra map as the after_value carrier for the success path, but the skipped path was not addressed — and for skipped ops `pr.Extra` is never populated (pre_state_check.go returns a bare PortResult without Extra, and `buildAfterValue`'s Extra-reading branch is only reached for non-skipped). Result: an auditor reviewing "why didn't this port's vlan change?" cannot see either the target vlan or the current vlan from the audit row.

The same gap affects `port_binding` skipped paths, but `port_binding` intentionally never returns skipped (pre_state_check.go:105-111 explicitly returns nil → always goes through executeWrite), so in practice only `set_access_vlan` is affected.

**Fix:**
Extend `buildBeforeValue` to include the fields relevant to v1.20.1 actions:

```go
func buildBeforeValue(port *models.DevicePortStatus) json.RawMessage {
    snapshot := map[string]interface{}{
        "admin_status":   port.AdminStatus,
        "dot1x_enabled":  port.Dot1xEnabled,
        "description":    port.Description,
        "interface_name": port.InterfaceName,
    }
    if port.VLAN != nil {
        snapshot["vlan"] = *port.VLAN
    }
    b, _ := json.Marshal(snapshot)
    return b
}
```

Additionally, for the `set_access_vlan` skipped path, `buildAuditRow` should override with a merge of before + Extra-target rather than just before (so the audit shows "before=N, after=N, reason=already at target"). Minimal fix is just adding `vlan` to `buildBeforeValue`; deeper fix is to special-case skipped v1.20.1 actions to emit `{"vlan": before}` style after_value. Add regression test `TestSetAccessVlanAudit_NoOp_ContainsVlan`.

---

### CR-03: SetAccessVlanModal `form.resetFields()` on open wipes the pre-filled vlanId

**File:** `xingran-react-frontend/src/components/network/port-write/SetAccessVlanModal.tsx:62-64, 119`
**Issue:**
The Modal flow:

```tsx
useEffect(() => {
  if (open) form.resetFields();  // line 63
}, [open, form]);

<Form
  form={form}
  initialValues={{ vlanId: portRecord?.vlan ?? 1 }}  // line 119
>
```

`initialValues` in antd Form only applies once at Form **mount**. Because the Modal uses `destroyOnHidden`, the Form remounts on each open — so `initialValues` IS read on mount. But then the `useEffect([open, form])` fires **after** mount and calls `form.resetFields()`, which resets all fields to their **defaults** (undefined for vlanId, since resetFields doesn't re-consult initialValues — it clears to the field's declared default which is empty for InputNumber).

Result: when the user opens "修改 access VLAN" for a port currently on VLAN 100, the InputNumber shows empty (placeholder "请输入 1-4094 之间的 VLAN ID") instead of `100`. The user must re-type the vlan every time, defeating the "D-02 pre-fill current port vlan" design goal stated in the file header comment (line 13).

Note: `PortBindingModal` has the same pattern (line 68-70 + initialValues at line 134), but its `initialValues` are constants (`op:"add"`, empty strings) that happen to coincide with what `resetFields()` resets to — so the bug is invisible there. `SetAccessVlanModal` is the only one where the pre-fill value is dynamic per-open.

**Fix:**
Replace `resetFields()` with explicit field initialization on open, OR use the `preserve={false}` prop + remove the manual reset:

```tsx
useEffect(() => {
  if (open && portRecord) {
    form.setFieldsValue({
      vlanId: portRecord.vlan ?? 1,
      reasonSelect: undefined,
      reasonText: undefined,
    });
  }
}, [open, portRecord, form]);
```

`setFieldsValue` explicitly writes the field values after mount, so the pre-fill works regardless of `destroyOnHidden` remounting. Verify in a quick render test that opening for VLAN 100 shows "100" in the InputNumber.

---

### CR-04: BatchWriteRequest has no binding validators — destructive op silently runs on bad input

**File:** `internal/api/v1/network/port_write_handler.go:325-352` + `internal/services/portwrite/batch_orchestrator.go:64-73`
**Issue:**
The handler's batch endpoint binds `portwrite.BatchWriteRequest` with only `c.ShouldBindJSON(&req)` — no gin validation tags on the struct (CR-01 covered the security side). Combined with the renderer's "conservative default to remove" branch (vendor_port_template.go:73-78, 87-92, 101-106), the following are reachable:

| Input | Single-port result | Batch result |
|---|---|---|
| `bindOp: "delete"` (typo for remove) | `ErrBindOpInvalid` → HTTP 400 | silently runs `renderXxxPortBindingRemove` → deletes binding |
| `vlanId: 0` | `ErrVlanIdOutOfRange` → HTTP 400 | renderer returns `vlanId 0 out of range` error → goes into `result.Failed[]` → HTTP 200 with single-port failure (not rejected) |
| `vlanId: -1` | `ErrVlanIdOutOfRange` → HTTP 400 | same as above — HTTP 200 with one failure |
| `ipAddress: ""` (empty) | `ErrIPAddressInvalid` → HTTP 400 | renderer's `p.IPAddress == ""` check returns `fmt.Errorf` → per-port failure in result, not entry-rejection |

The batch handler's sentinel switch (port_write_handler.go:337-350) only matches `ErrBatchTooLarge` / `ErrEmptyBatch` / `ErrMixedDevices` / `ErrPortNotFound` / `ErrDeviceNotFound` — the 4 v1.20.1 sentinels are **not** in the batch switch, so even if CR-01's fix makes the service emit them, the handler would fall through to `default: 500`. This is a double gap.

Additionally `ErrMixedDevices` (port_write_service.go:25) is defined and matched by the handler (line 342) but the service never emits it — dead code that misleads maintainers into thinking cross-device batches are rejected. (`ErrPortNotFound` has the same problem: defined + matched but never returned by any service code path.)

**Fix:**
Two-part fix:

1. After fixing CR-01 (service emits the 4 sentinels from batch entry), add the 4 sentinel cases to the batch handler switch:
```go
case errors.Is(err, portwrite.ErrVlanIdOutOfRange):
    response.Error(c, http.StatusBadRequest, "VLAN ID 必须在 1-4094 之间")
case errors.Is(err, portwrite.ErrBindOpInvalid):
    response.Error(c, http.StatusBadRequest, "绑定操作必须是 add 或 remove")
case errors.Is(err, portwrite.ErrIPAddressInvalid):
    response.Error(c, http.StatusBadRequest, "IP 地址格式不合法")
case errors.Is(err, portwrite.ErrMACAddressInvalid):
    response.Error(c, http.StatusBadRequest, "MAC 地址格式不合法")
```

2. Either wire up `ErrMixedDevices` (the pre-state query already filters by `device_id = ? AND id IN ?`, so cross-device ports just won't appear — making ErrMixedDevices truly dead) or delete the constant + its handler case to avoid confusion. Same for `ErrPortNotFound`.

## Warnings

### WR-01: IPv4 regex comment lies — `0.0.0.0` and `255.255.255.255` actually match

**File:** `internal/services/portwrite/port_write_service.go:35-41`
**Issue:**
The comment block at lines 36-40 claims:
> 各段值域 [0-255]，且首段不允许 0（避免 0.0.0.0/255.255.255.255 等边界值穿透到设备）

But the regex `^(([1-9]?\d|1\d\d|2[0-4]\d|25[0-5])\.){3}...` actually matches `0.0.0.0` and `255.255.255.255`. Verified empirically:
```
0.0.0.0                       match=true
255.255.255.255               match=true
00.0.0.0                      match=false  (leading zero blocked)
01.1.1.1                      match=false  (leading zero blocked)
```

The `[1-9]?\d` alternation matches `0` (since `\d` covers 0-9 and `[1-9]?` is optional). The intent ("first segment cannot be 0") is not implemented — only "first segment cannot have leading zero" is enforced.

This is a comment/code mismatch. Functionally, allowing `0.0.0.0` and `255.255.255.255` to reach the device is a minor concern (these are RFC-valid addresses; user may legitimately want to bind them), but the comment is misleading and any future reviewer will trust the comment over the code.

**Fix:**
Either fix the regex to actually reject 0.0.0.0 (add a negative-lookahead-style structure, hard in Go's RE2), or update the comment to accurately describe what's enforced:
```go
// 校验收紧：
//   - 各段值域 [0-255]（RFC-legal，含 0.0.0.0 / 255.255.255.255 边界值）
//   - 拒绝前导零（01.1.1.1 等）
//   - 不含 ; | & 等 shell 命令分隔符（input 自正则字符类已排除）
// 注：原注释"首段不允许 0"是误描述 — 现实只拒前导零。如需拒 0.0.0.0 业务地址，
// 在 validator 里加 `if ipAddress == "0.0.0.0" { return ErrIPAddressInvalid }`。
```

---

### WR-02: `all := append(append(result.Succeeded, result.Failed...), result.Skipped...)` may corrupt `result.Succeeded` backing array

**File:** `internal/api/v1/network/port_write_handler.go:355`
**Issue:**
```go
all := append(append(result.Succeeded, result.Failed...), result.Skipped...)
```

If `cap(result.Succeeded) >= len(Succeeded)+len(Failed)+len(Skipped)`, the inner `append(result.Succeeded, result.Failed...)` writes into `result.Succeeded`'s backing array beyond its length. The outer `append(...)` then extends further with Skipped. The returned `all` slice shares `result.Succeeded`'s backing array.

While the contents of `all` are logically correct, `result.Succeeded`'s backing array now contains data beyond its length — which is normally invisible. But the same `result` is later serialized into the HTTP response (line 396 `response.Success(c, result)`). Go's JSON marshaling only reads up to `len(result.Succeeded)`, so the visible `succeeded` array in the response is correct.

This is **not** a correctness bug today (verified logically), but it's an aliased-mutation landmine: any future code that mutates `all` will silently corrupt `result.Succeeded`, and any code that appends to `result.Succeeded` after this line will overwrite `all`'s contents. The conventional fix is to allocate fresh.

**Fix:**
```go
all := make([]portwrite.PortResult, 0, len(result.Succeeded)+len(result.Failed)+len(result.Skipped))
all = append(all, result.Succeeded...)
all = append(all, result.Failed...)
all = append(all, result.Skipped...)
```

---

### WR-03: `ErrMixedDevices` and `ErrPortNotFound` are dead sentinels — handler maps them but service never emits

**File:** `internal/services/portwrite/port_write_service.go:25-26` + `internal/api/v1/network/port_write_handler.go:248,342-347`
**Issue:**
- `ErrMixedDevices` is matched by the batch handler (line 342) but `grep -r "ErrMixedDevices" internal/services/portwrite/` shows the constant is only **defined** at line 25 — never returned. The batch pre-state query already filters by `device_id = ? AND id IN ?` (batch_orchestrator.go:80-83), so cross-device ports just don't appear in the map → they hit the "DB 查不到该 port" fallback at line 110-116 → `result.Failed` with "port not found". The dead sentinel misleads maintainers.
- `ErrPortNotFound` is matched by both single (line 248) and batch (line 344) handlers, but `writeSinglePort` on missing port falls through to `executeWrite` with empty deviceID → `ErrDeviceNotFound` (port_write_service.go:292-297). So `ErrPortNotFound` is unreachable.

**Fix:**
Either delete the constants and their handler cases, or wire them up:
- For `ErrPortNotFound`: have `writeSinglePort` return `ErrPortNotFound` directly on `gorm.ErrRecordNotFound` instead of falling through to the SSH path (current behavior is intentional per the D-13 comment, so deletion is cleaner).
- For `ErrMixedDevices`: delete the constant and the handler case.

---

### WR-04: `SetAccessVlanRequest` / `PortBindingRequest` use `binding:"required"` on integer/Op — 0/false empty values rejected inconsistently

**File:** `internal/api/v1/network/port_write_handler.go:74,87`
**Issue:**
```go
type SetAccessVlanRequest struct {
    VLANID int    `json:"vlanId" binding:"required,min=1,max=4094"`
    // ...
}
```

`binding:"required"` on an `int` in gin (which uses validator/v10) rejects the zero value. Combined with `min=1`, this is redundant: `min=1` already implies rejection of 0. But `required` also rejects `vlanId` being **absent** from JSON — which is the intent. The redundancy is harmless.

The real issue: `Op` uses `binding:"required,oneof=add remove"`. If the client sends `"op":""` (empty string), `required` rejects it → 400. If the client omits `op` entirely, `required` also rejects → 400. Both are correct. But the corresponding **batch** struct has no tags (CR-01/CR-04), so the same field has different validation behavior depending on endpoint.

**Fix:**
Addressed by CR-04's fix (add binding tags to `BatchWriteRequest`). No standalone change needed.

---

### WR-05: PortBindingModal `useEffect([open, form])` resets `op` field to undefined on re-open despite `initialValues`

**File:** `xingran-react-frontend/src/components/network/port-write/PortBindingModal.tsx:68-70, 134`
**Issue:**
Same pattern as CR-03 but with smaller user impact. `initialValues={{ op: "add", ipAddress: "", macAddress: "" }}` is read on mount, then `form.resetFields()` clears `op` to undefined (not "add"). The Radio.Group shows no selection on re-open.

In practice the user will click a Radio button anyway, so the UX impact is lower than CR-03's vlanId pre-fill, but it's still a deviation from the stated default `op: "add"`.

**Fix:**
Mirror CR-03's fix:
```tsx
useEffect(() => {
  if (open) {
    form.setFieldsValue({ op: "add", ipAddress: "", macAddress: undefined });
  }
}, [open, form]);
```

---

### WR-06: Frontend `BatchWriteRequest` type declares optional fields but `BulkWriteDrawer` likely doesn't collect them

**File:** `xingran-react-frontend/src/types/network.ts:316-328` + `xingran-react-frontend/src/pages/network/ports/index.tsx:586-592`
**Issue:**
The frontend `BatchWriteRequest` type now has `vlanId?`, `op?`, `ipAddress?`, `macAddress?` fields. The `ports/index.tsx` page mounts `<BulkWriteDrawer>` (line 586-592) but the drawer's source is not in the review scope. Given that Phase 56 explicitly added these batch fields, the drawer must have UI for collecting vlanId (for set_access_vlan batch) and op/ip/mac (for port_binding batch).

Since the drawer source is out of scope, this is flagged as a WARNING: if the drawer doesn't collect these fields, batch operations for set_access_vlan / port_binding will send empty payloads — which combined with CR-01/CR-04 means a user who batch-selects ports and chooses "set_access_vlan" will send `vlanId: 0` → renderer returns "vlanId 0 out of range" error → each port fails individually → user sees 50 failures with no clear cause.

**Fix:**
Verify `BulkWriteDrawer.tsx` renders the v1.20.1-specific input fields when `action === "set_access_vlan"` or `action === "port_binding"` is selected. If not, add them. (This file is out of review scope; flag for follow-up review.)

## Info

### IN-01: Dead `MessageInstance` type import in both Modals

**File:** `xingran-react-frontend/src/components/network/port-write/SetAccessVlanModal.tsx:20` + `PortBindingModal.tsx:23`
**Issue:**
```ts
import type { MessageInstance } from "antd/es/message/interface";
```
This type is imported but never referenced. The `message` variable from `App.useApp()` is used directly (passed to `showAuditLinkToast`), and `MessageInstance` is only the type alias for that — no annotation uses it.

**Fix:**
Delete the import line.

---

### IN-02: `localNormalizeMACAddress` duplicate of `services.NormalizeMACAddress` — drift risk

**File:** `internal/services/portcollection/vendor_port_template.go:139-156`
**Issue:**
The file contains a local copy of `NormalizeMACAddress` with a comment explaining it's to avoid an import cycle (`portcollection` cannot import `services`). The comment acknowledges "任何字段含义变更须同步两边" — but there's no compile-time or test-time enforcement of this. The two implementations could drift silently.

`NormalizeMACAddress` is in `internal/services/mac_normalize.go`. The clean fix would be to move `mac_normalize.go` to a leaf package (e.g. `pkg/normalize/mac.go`) that both `services` and `portcollection` can import.

**Fix:**
Low priority — move `NormalizeMACAddress` to `pkg/normalize/mac.go` and have both packages import it. Adds a single source of truth.

---

### IN-03: `Reason` field declared in `BatchWriteRequest` on frontend but absent on backend struct

**File:** `xingran-react-frontend/src/types/network.ts:327` vs `internal/services/portwrite/port_write_service.go:77-88`
**Issue:**
Frontend type:
```ts
interface BatchWriteRequest {
  // ...
  reason?: string;  // line 327
}
```

Backend struct (port_write_service.go:77-88) has no `Reason` field. The frontend type is wider than the backend contract; sending `reason` in a batch request will be silently ignored by Go's JSON unmarshaling (unknown field). UI-02 reason capture is documented as single-port-only.

**Fix:**
Either remove `reason?` from the frontend `BatchWriteRequest` type, or add `Reason string` to the backend struct and propagate it to the audit/operlog layer for batch ops.

---

### IN-04: `SetAccessVlanModal` / `PortBindingModal` use `composeReason(...) ?? ""` — backend Reason field is `omitempty`

**File:** `xingran-react-frontend/src/components/network/port-write/SetAccessVlanModal.tsx:91` + `PortBindingModal.tsx:105`
**Issue:**
Both Modals call:
```ts
await writeSetAccessVlan(portRecord.id, values.vlanId as number, reason ?? "");
```

The `reason ?? ""` coerces null to empty string. The wrapper then sends `{reason: ""}` in the JSON body. Backend `SetAccessVlanRequest.Reason` is `json:"reason,omitempty"` so the field is serialized, and `binding:"required"` is NOT applied — empty string passes. This is consistent with UI-02 ("reason 仅记录不校验").

However, since the frontend has `validateReasonRequired` enforcing non-empty reason (constants.ts:162-180), `reason` should never be null at this point. The `?? ""` is defensive dead code.

**Fix:**
Cosmetic — remove `?? ""` and rely on the validator, or keep as defensive. No functional impact.

---

### IN-05: `batch_orchestrator.go` line 33 comment references "Pitfall #5" — no link to source

**File:** `internal/services/portwrite/batch_orchestrator.go:33`
**Issue:**
Comment says "这是 PROJECT.md Pitfall #5 的核心兜底" but `PROJECT.md` is not in the repo root. Multiple Phase 56 files reference Pitfall #5 / #6 / LANDMINE #5 without a discoverable source document. Future maintainers will struggle to verify these design constraints.

**Fix:**
Either commit `PROJECT.md` / `PITFALLS.md` to the repo, or inline the pitfall description in each comment block.

---

_Reviewed: 2026-07-09_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
