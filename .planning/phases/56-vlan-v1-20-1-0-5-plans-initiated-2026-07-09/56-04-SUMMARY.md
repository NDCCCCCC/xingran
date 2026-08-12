---
phase: 56-vlan-v1-20-1-0-5-plans-initiated-2026-07-09
plan: 04
subsystem: frontend-react
tags: [vlan, port-binding, network, frontend, antd, react, typescript]
requires:
  - 56-03 (backend routes: POST /network/ports/write/set-access-vlan + /port-binding)
  - 53-02 (v1.19 PortWriteModal + BulkWriteDrawer + constants + networkApi skeleton)
provides:
  - "SetAccessVlanModal + PortBindingModal single-port Modals driving 2 new v1.20.1 write actions"
  - "writeSetAccessVlan + writePortBinding networkApi wrappers (kebab-aligned, LANDMINE #5 compliant)"
  - "PortWriteAction union +2 literals (set_access_vlan, port_binding) + BatchWriteRequest +4 optional fields"
  - "ACTION_TITLE +2 keys + IPV4_REGEX + MAC_REGEX + BIND_OPS constants"
  - "ports/index.tsx ActionButtons menu 5 -> 7 items (canWrite gating inherited)"
affects:
  - "xingran-react-frontend/src/types/network.ts"
  - "xingran-react-frontend/src/lib/api/networkApi.ts"
  - "xingran-react-frontend/src/components/network/port-write/constants.ts"
  - "xingran-react-frontend/src/pages/network/ports/index.tsx"
tech-stack:
  added: [] # zero new npm dependencies
  patterns:
    - "v1.19 Phase 53 W4 Modal pattern (520px width, destroyOnHidden, Form vertical, reason Select+TextArea)"
    - "LANDMINE #5 wrapper pattern (no try/catch, no message.error, post() interceptor handles errors)"
    - "showAuditLinkToast helper reuse (react-router <a href> + navigate, not <Link>)"
    - "validateReasonRequired cross-field form arg (55-01 WR-02 fix)"
key-files:
  created:
    - xingran-react-frontend/src/components/network/port-write/SetAccessVlanModal.tsx
    - xingran-react-frontend/src/components/network/port-write/PortBindingModal.tsx
  modified:
    - xingran-react-frontend/src/types/network.ts
    - xingran-react-frontend/src/lib/api/networkApi.ts
    - xingran-react-frontend/src/components/network/port-write/constants.ts
    - xingran-react-frontend/src/pages/network/ports/index.tsx
decisions:
  - "2 new actions get independent Modals (not extending PortWriteModal) — field shapes too different (1 number vs 3 fields)"
  - "wrappers use positional args matching v1.19 convention (not SetAccessVlanRequest/PortBindingRequest param objects)"
  - "BatchWriteDrawer intentionally NOT extended for 2 new actions in v1.20.1 (FUTURE-BATCH-05 deferred per W2 design)"
  - "BIND_OPS cast via 'as unknown as Array' to satisfy tsc -b project-refs strictness"
  - "macAddress empty string normalized to undefined at Modal layer (IP-only binding semantics)"
metrics:
  duration: "~45 min"
  completed: 2026-07-09
  tasks_complete: 7
  files_changed: 6
  commits: 7
  bundle_delta_gzip_kB: -1.05 # 774.95 - 776 baseline (under)
---

# Phase 56 Plan 04: Frontend Types + API + 2 Modals + Ports Menu Summary

Extended the v1.19 frontend (Phase 53 W4) with 2 new single-port Modals (`SetAccessVlanModal` + `PortBindingModal`), 2 new TypeScript types, 2 new networkApi wrappers, 3 new constants, and 2 new menu items in `ports/index.tsx` — driving the v1.20.1 VLAN + port-binding write operations end-to-end with v1.19-parity form validation, audit toast, and LANDMINE #5 compliance.

## What Was Built

### Task 1 — `types/network.ts` extension (commit `03b17278`)
- `PortWriteAction` union +2 literals: `"set_access_vlan"` + `"port_binding"` (appended after 5 v1.19 literals — order preserved)
- `BatchWriteRequest` +4 optional fields: `vlanId?: number`, `op?: "add" | "remove"`, `ipAddress?: string`, `macAddress?: string`, `reason?: string` (pairs with W2 batch_orchestrator.go + W3 handler BatchWriteRequest)
- `SetAccessVlanRequest` interface: `{ portId, vlanId: number 1-4094, reason }`
- `PortBindingRequest` interface: `{ portId, op: "add"|"remove", ipAddress, macAddress?, reason }`

### Task 2 — `networkApi.ts` 2 new wrappers (commit `fa1df64e`)
- `writeSetAccessVlan(portId, vlanId, reason)` → `POST /network/ports/write/set-access-vlan`
- `writePortBinding(portId, op, ipAddress, macAddress|undefined, reason)` → `POST /network/ports/write/port-binding`
- Both follow v1.19 LANDMINE #5 contract: zero `try/catch`, zero `message.error` (post() interceptor handles non-0 codes with Toast)
- Default export extended with both wrappers

### Task 3 — `constants.ts` extension (commit `9e8f7363`)
- `ACTION_TITLE` Record +2 keys: `set_access_vlan: "修改 access VLAN"`, `port_binding: "端口绑定"`
- `IPV4_REGEX` — strict IPv4 (rejects 0.x.x.x / 255.x.x.x, aligned with backend `ipv4Pattern`)
- `MAC_REGEX` — accepts colon / hyphen / no-separator (aligned with backend `NormalizeMACAddress`)
- `BIND_OPS` — readonly tuple `[{add}, {remove}]` for Radio.Group options

### Task 4 — `SetAccessVlanModal.tsx` created (commit `e7e2dcd8`)
- 195 lines, named export (not default)
- `vlanId` InputNumber (min=1 max=4094 step=1) + `extra="范围 1-4094 (VLAN 0/4095 保留)"`
- reason Select + custom TextArea (REQUIRED via `validateReasonRequired(rule, value, form)` — 55-01 WR-02 cross-field)
- `initialValues={{ vlanId: portRecord?.vlan ?? 1 }}` prefills current port VLAN
- `showAuditLinkToast(message, navigate)` on success (reused from PortWriteModal)
- `destroyOnHidden` + `useEffect [open, form] form.resetFields()` (stale-state guard)
- `okButtonProps={{ loading: submitting }}` (T-56-W4-07 anti-spam mitigation)

### Task 5 — `PortBindingModal.tsx` created (commits `a29758d6` + `78188c62`)
- 231 lines, named export
- 3 main fields: `op` Radio.Group (BIND_OPS, default "add", buttonStyle="solid") + `ipAddress` Input (required, IPV4_REGEX pattern) + `macAddress` Input (optional, MAC_REGEX pattern, `extra` hint)
- reason Select + custom TextArea (REQUIRED)
- macAddress empty string normalized to `undefined` at Modal layer → IP-only binding semantics
- `showAuditLinkToast` + `destroyOnHidden` + `form.resetFields()` on open + `submitting` guard
- **Rule 1 fix:** initial `as Array<{label,value}>` cast failed `tsc -b` (project refs stricter than `tsc --noEmit`: rejects readonly→mutable). Changed to `as unknown as Array<...>` double-cast (standard TS pattern).

### Task 6 — `ports/index.tsx` extension (commit `6f095ff9`)
- 2 new named imports: `SetAccessVlanModal` + `PortBindingModal`
- 4 new state vars: `vlanModalOpen`, `vlanModalRecord`, `bindModalOpen`, `bindModalRecord`
- 2 new openers: `openVlanModal(record)` + `openBindModal(record)`
- ActionButtons array extended 5 → 7 items: `修改 access VLAN` + `端口绑定` (appended after 5 v1.19 items)
- 2 new Modal mount points near existing `<PortWriteModal />`
- `canWrite` gating inherited (items inside existing `canWrite ? [...actions] : []` ternary — T-56-W4-01 mitigation)
- `onSuccess` callbacks call `loadPortStatus()` + `loadStatistics()` (matches v1.19 pattern)

### Task 7 — Build + bundle size gate (commits `78188c62` for the fix; gate itself no-commit)
- `npm run type-check` exit 0 (via `tsc -b`)
- `npm run build` exit 0 (Vite production build, 1m 29s)
- **vendor-react bundle gzip = 774.95 kB ≤ 776 kB baseline** (−1.05 kB — zero regression, actually slightly under)
- `git diff package.json` empty — zero new npm dependencies
- All 6 modified/new files present and compile cleanly
- Spot-check greps all meet/exceed plan thresholds

## Verification Results

| Check | Expected | Actual | Status |
|-------|----------|--------|--------|
| `npm run type-check` exit | 0 | 0 | PASS |
| `npm run build` exit | 0 | 0 | PASS |
| vendor-react gzip | ≤ 776 kB | 774.95 kB | PASS |
| `package.json` diff | empty | empty | PASS |
| PortWriteAction literals | 7 (5+2) | 7 | PASS |
| ACTION_TITLE keys | 7 | 7 | PASS |
| ActionButtons items | 7 (5→7) | 7 | PASS |
| networkApi wrappers | 2 new | 2 new | PASS |
| networkApi grep count (writeSetAccessVlan\|writePortBinding) | ≥4 | 4 | PASS |
| ports/index grep (SetAccessVlanModal\|PortBindingModal) | ≥4 | 5 | PASS |
| constants grep (set_access_vlan\|port_binding) | ≥4 | 5 | PASS |
| LANDMINE #5 (no try/catch in wrappers) | 0 | 0 | PASS |
| LANDMINE #5 (no message.error in Modals) | 0 | 0 | PASS |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed PortBindingModal BIND_OPS readonly cast for `tsc -b`**
- **Found during:** Task 7 build gate
- **Issue:** The plan's Task 5 code example used `options={BIND_OPS as Array<{ label: string; value: string }>}`. This passed `tsc --noEmit -p .` (used in Tasks 4-6 verification) but failed `tsc -b` (the build script's first step, which uses project references with stricter settings). The error: `readonly [...]` tuple cannot be cast to mutable `Array<...>` — TS2352.
- **Fix:** Changed to `as unknown as Array<{ label: string; value: string }>` (standard TS double-cast pattern for readonly tuples). Root cause is that `BIND_OPS` is declared `as const` (readonly) but antd's `Radio.Group` `options` prop expects a mutable array type.
- **Files modified:** `xingran-react-frontend/src/components/network/port-write/PortBindingModal.tsx` (1 line)
- **Commit:** `78188c62`

**2. [Process] Generated artifact `asset_columns_schema.json` excluded from commits**
- **Found during:** Task 7 build gate
- **Issue:** The frontend's `prebuild` script (`npm run sync-columns-schema`) auto-regenerates `internal/services/system/asset_columns_schema.json` with a fresh `__generated__` timestamp on every build. This produced an uncommitted diff (timestamp bump only, no schema change) unrelated to Phase 56 W4.
- **Fix:** Reverted the file via `git checkout --` (out of scope per scope boundary rules — pre-existing generated artifact, no Phase 56 schema change). Did NOT commit.
- **Files affected:** `internal/services/system/asset_columns_schema.json` (reverted, untracked change discarded)

## Known Limitations

**BulkWriteDrawer NOT extended for 2 new actions (FUTURE-BATCH-05 deferred).** Per the plan's `<success_criteria>` #6 and W2 design decision, the batch path for `set_access_vlan` / `port_binding` is deferred to a future batch-05 workstream. The `BatchWriteRequest` type was extended with 4 optional fields (Task 1) to enable this future work, and `constants.ts` `ACTION_TITLE` includes the 2 new keys so any future `ACTION_OPTIONS` extension can reference them — but `BulkWriteDrawer.tsx` itself is UNCHANGED in v1.20.1. The 2 new actions are single-port only via the 2 new Modals.

## STRIDE Threat Mitigation Summary

All 7 threats from the plan's `<threat_model>` are mitigated:
- **T-56-W4-01** (EoP): `canWrite` gating from menu store wraps entire `actions` array — 2 new items inherit.
- **T-56-W4-02** (XSS): All form values rendered via antd components (React-escaped); no `dangerouslySetInnerHTML`.
- **T-56-W4-03/04** (Tampering): antd `rules` (pattern + required + min/max) + backend service re-validation (defense in depth).
- **T-56-W4-05/06** (Info Disclosure): `op`/`macAddress` accepted — no sensitive data, not in operlog 11-keyword mask list.
- **T-56-W4-07** (DoS spam): `submitting` state + `okButtonProps={{ loading }}` + `finally { setSubmitting(false) }`.
- **T-56-W4-SC** (Supply chain): Zero new npm packages — `package.json` diff empty.

## Self-Check: PASSED

**Files verified present:**
- FOUND: `xingran-react-frontend/src/types/network.ts`
- FOUND: `xingran-react-frontend/src/lib/api/networkApi.ts`
- FOUND: `xingran-react-frontend/src/components/network/port-write/constants.ts`
- FOUND: `xingran-react-frontend/src/components/network/port-write/SetAccessVlanModal.tsx`
- FOUND: `xingran-react-frontend/src/components/network/port-write/PortBindingModal.tsx`
- FOUND: `xingran-react-frontend/src/pages/network/ports/index.tsx`

**Commits verified in git log:**
- FOUND: `03b17278` (Task 1)
- FOUND: `fa1df64e` (Task 2)
- FOUND: `9e8f7363` (Task 3)
- FOUND: `e7e2dcd8` (Task 4)
- FOUND: `a29758d6` (Task 5)
- FOUND: `6f095ff9` (Task 6)
- FOUND: `78188c62` (Task 7 Rule 1 fix)
