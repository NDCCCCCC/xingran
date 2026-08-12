---
phase: 56
slug: vlan-v1-20-1-0-5-plans-initiated-2026-07-09
status: approved
shadcn_initialized: false
preset: none
created: 2026-07-09
reviewed_at: 2026-07-09
---

# Phase 56 — UI Design Contract

> Visual and interaction contract for v1.20.1 网络设备 VLAN + 端口绑定 (Phase 56).
> Purely additive: 2 new single-port Modals + 2 menu items. **Zero new component library, zero new dependency, zero new design tokens.**
> All visual contracts inherit verbatim from v1.19 PortWriteModal / BulkWriteDrawer / ActionButtons (Phase 53, shipped 2026-07-07).

---

## Design System

| Property | Value | Source |
|----------|-------|--------|
| Tool | none | No shadcn — project uses Ant Design 6.1 + Tailwind 4.1 + design-system tokens |
| Preset | not applicable | No shadcn initialization; antd 6.1 + design-system tokens are the established UI layer |
| Component library | Ant Design 6.1 | Phase 53 W4 established Modal / Drawer / Form / Select / InputNumber / Radio.Group / Input / Table |
| Icon library | @ant-design/icons 5.x | Phase 53 + CLAUDE.md §Frontend (SettingOutlined, ReloadOutlined, SearchOutlined, etc. used in ports/index.tsx) |
| Font | `-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif` | design-system/tokens/typography.ts `fontFamily.sans` (project standard, antd default inherits) |
| Theme | antd ConfigProvider + flat2.0 / glassmorphism / luxury-quiet / minimal / neumorphism 5 themes | design-system/themes/index.ts — Modal/Drawer must remain theme-agnostic (no hardcoded hex) |

**Rationale for "Tool: none":** Phase 53 already shipped the full Modal/Drawer/ActionButtons skeleton. Phase 56 is purely a form-field extension — no new design system, no new component primitives, no new theme. Reusing the existing tokens ensures visual continuity and zero bundle regression (vendor-react gzip 776 kB baseline preserved per ROADMAP §Success Criteria #5).

---

## Spacing Scale

**Inherited verbatim from v1.19 Phase 53 (no changes).** All spacing uses the project `design-system/tokens/spacing.ts` 8px grid.

| Token | Value | Usage in this phase |
|-------|-------|---------------------|
| xs | 4px | Icon gaps inside Modal footer buttons; Form.Item extra hint gap |
| sm | 8px | Radio.Group option gap; InputNumber/Input internal padding |
| md | 16px | Default Form.Item vertical marginBottom; Modal body padding; Tag inline gap |
| lg | 24px | Card padding (Alert "Mixed devices" warning); Drawer section spacing |
| xl | 32px | (reserved for layout-level — not used in Modals) |
| 2xl | 48px | (reserved — not used) |
| 3xl | 64px | (reserved — not used) |

Exceptions: **None**. Phase 56 introduces no new spacing patterns. Both new Modals reuse the 520px Modal width from PortWriteModal (no new width variant needed).

---

## Typography

**Inherited verbatim from v1.19 Phase 53 (no changes).** All typography uses antd defaults + design-system `typography.h*` / `body` presets. Modal/Drawer renders all text via antd `Typography.Title` / `Typography.Text` / default labels — no custom font-size overrides.

| Role | Size | Weight | Line Height | Source |
|------|------|--------|-------------|--------|
| Body | 14px (antd default) | 400 (normal) | 1.5715 (antd default) | antd 6.1 Form.Item label + Input value |
| Label | 14px | 500 (medium) | 1.5715 | antd Form.Item label default |
| Heading (Modal title) | 16px | 500 (medium) | 1.5 | antd Modal default header — `${ACTION_TITLE[action]} - ${interfaceName}` |
| Drawer title | 16px | 500 | 1.5 | antd Drawer default (BulkWriteDrawer unchanged) |
| Statistic value (BulkWriteDrawer result) | 24px | 500 | 1.5 | antd Statistic default |
| Table cell | 14px | 400 | 1.5 | antd Table default |
| Tag | 12px | 400 | 1.5 | antd Tag default (status badges in failure detail) |

**Constraints (locked from Phase 53):**
- **Body line-height: 1.5** (per design system `lineHeight.normal`)
- **Heading line-height: 1.2~1.5** (antd default, no override)
- **Max 2 font weights in any single Modal/Drawer view:** 400 (body/Input value) + 500 (label/button). Bold (700) reserved for the `✓ 成功 / ⚠ 跳过 / ✗ 失败` Statistic titles only.
- No custom font-size declarations; all text rendered through antd components that inherit theme tokens.

---

## Color

**Inherited verbatim from v1.19 Phase 53 (no changes).** The v1.19 modal already uses CSS custom properties (`var(--theme-*)`) with antd fallback colors. Phase 56 adds no new color tokens.

| Role | Value | Usage in this phase |
|------|-------|---------------------|
| Dominant (60%) | `#ffffff` (theme.canvas) | Modal/Drawer body background |
| Secondary (30%) | `#fafafa` (gray-50, table expanded row bg) | `expandedRowRender` panel background — **not used by new Modals** |
| Accent (10%) | `#3b82f6` (primary-500, `var(--theme-info, #1890ff)` / antd colorPrimary) | Primary CTA button ("确认执行" / "开始批量配置"); Audit link in Toast |
| Destructive | `#cf1322` (`var(--theme-error)` / antd colorError) | `✗ 失败` Statistic valueStyle; BatchWriteDrawer disabled-error-text button (batch delete — pre-existing, not modified by Phase 56) |
| Theme success | `#3f8600` (`var(--theme-success)` / antd colorSuccess) | `✓ 成功` Statistic valueStyle (BulkWriteDrawer, reused for batch path of new actions) |
| Theme warning | `#faad14` (antd colorWarning) | `⚠ 跳过` secondary text when skipped>0 (BulkWriteDrawer, reused) |

**Accent reserved for (explicit list — never "all interactive elements"):**
1. Primary CTA "确认执行" button on SetAccessVlanModal + PortBindingModal
2. Primary CTA "开始批量配置" button on BulkWriteDrawer (already exists, supports new actions via ACTION_OPTIONS extension)
3. Audit link "查看审计日志" inside success Toast (port-write/PortWriteModal `showAuditLinkToast` helper — reused as-is)
4. Active Form.Item focus border (`antd colorPrimary` token, not explicit hex)

**Forbidden (Phase 56 must NOT add new color usage):**
- No new hex color literals in TSX
- No new CSS variables in `theme-styles.css`
- No hardcoded `style={{ color: '#xxx' }}` in the 2 new Modal components
- All semantic colors must resolve via antd theme tokens OR existing `var(--theme-*)` fallbacks from Phase 53

**Reasoning:** Phase 56 has no destructive operations (set_access_vlan and port_binding are reversible operations — the device accepts the inverse command). The `Destructive` color is preserved for the pre-existing "批量删除" button only (out of scope, not modified).

---

## Copywriting Contract

**All copy in Chinese (project-wide convention per CLAUDE.md §Frontend — 中文模块名 + 中文 UI labels).** Locked strings derive from v1.19 PortWriteModal / BulkWriteDrawer + Phase 56 design doc §8.

| Element | Copy | Source / Notes |
|---------|------|----------------|
| **ACTION_TITLE entries (constants.ts)** | `set_access_vlan: "修改 access VLAN"` / `port_binding: "端口绑定"` | Phase 56 design doc §8.2 |
| Modal title pattern | `${ACTION_TITLE[action]} - ${interfaceName}` | e.g. "修改 access VLAN - GE1/0/0" — reuses PortWriteModal pattern verbatim |
| Primary CTA (single-port Modal) | `确认执行` | Matches v1.19 PortWriteModal `okText="确认执行"` |
| Primary CTA (batch Drawer) | `开始批量配置` | BulkWriteDrawer existing `开始批量配置` button text |
| Cancel button | `取消` | v1.19 standard |
| Empty state heading | (n/a — no empty list state) | Port list always has data; the new Modals never render in empty-list context |
| Empty state body | (n/a) | — |
| Error state (validation toast) | `请输入 VLAN ID` / `VLAN ID 必须在 1-4094 之间` | antd Form.Item `rules` messages — see SetAccessVlanModal form rules below |
| Error state (regex) | `请输入合法 IPv4 地址` / `请输入合法 MAC 地址（如 AA:BB:CC:DD:EE:FF）` | antd Form.Item rules on PortBindingModal |
| Error state (network failure) | (handled by `post()` interceptor per LANDMINE #5) | New Modals inherit this — no local `message.error` |
| Success Toast | `操作成功，查看审计日志` (5s duration, with link) | Reuse `showAuditLinkToast(message, navigate)` from PortWriteModal.tsx — **zero modification** |
| OperType semantic labels (Toast variant) | (n/a — Toast text is unified "操作成功" for Create/Update/Delete) | design doc §8.3 says OperType 分流 but the **Toast text stays identical** to v1.19; OperType 仅影响 oper_param 字段，不改变 UI 文案 |
| Destructive confirmation | (n/a — no destructive action in Phase 56) | VLAN change and binding add/remove are all reversible via the inverse command |

### SetAccessVlanModal Form Rules

```typescript
// Reason: 复用 v1.19 validateReasonRequired (5-200 字符, 跨字段跨 form 取值)
// Source: port-write/constants.ts:127 REASON_VALIDATOR pattern

Form.Item name="vlanId" label="VLAN ID" rules={[
  { required: true, message: "请输入 VLAN ID" },
  { type: "number", min: 1, max: 4094, message: "VLAN ID 必须在 1-4094 之间" },
]}>
  <InputNumber min={1} max={4094} step={1} style={{ width: "100%" }} placeholder="请输入 1-4094 之间的 VLAN ID" />
</Form.Item>
// Form.Item extra hint (per research A5 + open question #4):
//   extra="范围 1-4094 (VLAN 0/4095 保留)"

// reasonSelect: 复用 PRESET_REASONS + validateReasonRequired
// reasonText: 复用 Input.TextArea + REASON_CUSTOM_SENTINEL 分支
```

### PortBindingModal Form Rules

```typescript
// Op: 复用 antd Radio.Group (横向 Button.Group 样式)
// IP/MAC: 复用 antd Input (非 InputNumber)

Form.Item name="op" label="操作" rules={[{ required: true, message: "请选择绑定操作" }]}>
  <Radio.Group buttonStyle="solid" options={[
    { label: "新增绑定 (add)", value: "add" },
    { label: "删除绑定 (remove)", value: "remove" },
  ]} />
</Form.Item>

Form.Item name="ipAddress" label="IP 地址" rules={[
  { required: true, message: "请输入 IP 地址" },
  { pattern: IPv4_REGEX, message: "请输入合法 IPv4 地址（如 10.62.25.5）" },
]}>
  <Input placeholder="例如 10.62.25.5" allowClear />
</Form.Item>

Form.Item name="macAddress" label="MAC 地址（可选）" rules={[
  { pattern: MAC_REGEX, message: "请输入合法 MAC 地址（如 AA:BB:CC:DD:EE:FF）" },
]}>
  <Input placeholder="例如 AA:BB:CC:DD:EE:FF（不填则仅 IP 绑定）" allowClear />
</Form.Item>
// 注: 客户端 regex 仅 UX hint, 后端 service 层 (BIND-07) 校验为真相源
//     后端会归一化 MAC 格式: Huawei/H3C 用 AABB-CCDD-EEFF, Ruijie 用 aabb.ccdd.eeff

// reasonSelect + reasonText: 同 SetAccessVlanModal
```

### IPv4 / MAC Regex 常量 (constants.ts 扩展)

```typescript
// 与后端 service 层 regex 对齐 (BIND-07, RISK-05)

// IPv4: 严格 4 段十进制 0-255, 拒 0.x.x.x / 255.x.x.x 范围
export const IPV4_REGEX = /^(([1-9]?\d|1\d\d|2[0-4]\d|25[0-5])\.){3}([1-9]?\d|1\d\d|2[0-4]\d|25[0-5])$/;

// MAC: 接受 3 种常见格式 (canonical colon / hyphen / no-separator)
export const MAC_REGEX = /^([0-9A-Fa-f]{2}[:\-]?){5}[0-9A-Fa-f]{2}$/;
```

---

## Component Inventory (locked, no new dependencies)

| Component | Path | Action |
|-----------|------|--------|
| `SetAccessVlanModal` | `src/components/network/port-write/SetAccessVlanModal.tsx` | **CREATE** — single-port Modal for VLAN ID 1-4094 + reason |
| `PortBindingModal` | `src/components/network/port-write/PortBindingModal.tsx` | **CREATE** — single-port Modal for add/remove binding + IP + MAC + reason |
| `PortWriteModal` | `src/components/network/port-write/PortWriteModal.tsx` | UNCHANGED — reused as-is for the 5 v1.19 actions |
| `BulkWriteDrawer` | `src/components/network/port-write/BulkWriteDrawer.tsx` | **MINIMAL MODIFY** — extend `ACTION_OPTIONS` array by 2 entries, extend `ACTION_TITLE` Record by 2 keys; no UI structural change |
| `port-write/constants.ts` | `src/components/network/port-write/constants.ts` | **MODIFY** — extend `ACTION_TITLE` Record by 2 keys (`set_access_vlan` / `port_binding`); add `IPV4_REGEX` + `MAC_REGEX` constants; add `BIND_OPS` const tuple for Radio.Group options |
| `ports/index.tsx` | `src/pages/network/ports/index.tsx` | **MODIFY** — add 2 menu items to ActionButtons array (line ~344); add 2 Modal mount points (line ~545); add 2 state vars (`vlanModalOpen`, `bindModalOpen`, `vlanModalRecord`, `bindModalRecord`); reuse `canWrite` gating pattern |
| `types/network.ts` | `src/types/network.ts` | **MODIFY** — extend `PortWriteAction` union by 2 literals; add `SetAccessVlanRequest` + `PortBindingRequest` interfaces |
| `lib/api/networkApi.ts` | `src/lib/api/networkApi.ts` | **MODIFY** — add `writeSetAccessVlan` + `writePortBinding` 2 wrappers (kebab-aligned) |

**No third-party UI blocks, no shadcn, no new npm packages.**

---

## Interaction Contract (locked patterns from Phase 53)

### Single-port Modal flow (SetAccessVlanModal / PortBindingModal)

```
ActionButtons menu item onClick
        ↓ openWriteModal("set_access_vlan" | "port_binding", record)
        ↓ set state: { open=true, record=record }
        ↓ form.resetFields() on open (useEffect [open, action, form])
        ↓
User opens Modal (520px width, destroyOnHidden)
        ↓ antd Form vertical layout
        ↓ vlanId OR op/ipAddress/macAddress fields render
        ↓ reasonSelect pre-populated with PRESET_REASONS
        ↓
User clicks "确认执行"
        ↓ submitting=true (button loading spinner)
        ↓ form.validateFields() — antd marks red on fail, no Toast
        ↓ composeReason(reasonSelect, reasonText) helper
        ↓ wrapper call (writeSetAccessVlan / writePortBinding)
        ↓ 0 try/catch in wrapper, 0 message.error in component (LANDMINE #5)
        ↓ post() interceptor handles non-0 code → Toast
        ↓ on success: showAuditLinkToast(message, navigate)
        ↓ form.resetFields(); onSuccess(); onClose()
        ↓ submitting=false in finally
```

### Bulk path (BulkWriteDrawer — minimal extension)

```
User selects N ports (selectedRowKeys) + clicks "批量配置"
        ↓ BulkWriteDrawer opens (720px width, three-state machine)
        ↓ ACTION_OPTIONS now has 7 entries (was 5)
        ↓ User selects "set_access_vlan" OR "port_binding"
        ↓ New fields appear via shouldUpdate:
            - set_access_vlan → vlanId InputNumber (1-4094)
            - port_binding   → op Radio.Group + ipAddress Input + macAddress Input
        ↓ User selects reason (PRESET_REASONS / __custom__ TextArea)
        ↓ "开始批量配置" click
        ↓ buildRequest(deviceId, action, portIds, optional fields)
        ↓ batchWritePorts(req) — same endpoint as v1.19, action value new
        ↓ backend dispatches based on action string literal
        ↓ result view: ✓ 成功 / ⚠ 跳过 / ✗ 失败 Statistic cards
        ↓ failure detail Table + collapsible skipped list + "重试失败端口" button
        ↓ showAuditLinkToast on full success (failed.length === 0)
```

### ActionButtons menu — ports/index.tsx extension

```
Operation column menu items (5 → 7, canWrite gated):
  关闭端口 (shutdown)              → openWriteModal("shutdown", record)
  启用端口 (undo_shutdown)         → openWriteModal("undo_shutdown", record)
  修改描述 (description)           → openWriteModal("description", record)
  启用 802.1X (dot1x_enable)      → openWriteModal("dot1x_enable", record)
  停用 802.1X (dot1x_disable)     → openWriteModal("dot1x_disable", record)
  修改 access VLAN (set_access_vlan)  → openVlanModal(record)              [NEW]
  端口绑定 (port_binding)             → openBindModal(record)              [NEW]
```

### Audit Toast (reused verbatim)

```typescript
// From src/components/network/port-write/PortWriteModal.tsx:73-91
// All 3 single-port Modals + BulkWriteDrawer call this helper.
// Phase 56 introduces no new Toast variant — operlog OperType 仅分流后端 audit,
//   前端 Toast 文案统一为 "操作成功, 查看审计日志"。
//   OperType Create (port_binding add) / Update (set_access_vlan) / Delete (port_binding remove)
//   全部映射到同一条 Toast 文本。

import { showAuditLinkToast } from "@/components/network/port-write/PortWriteModal";
// 调用: showAuditLinkToast(message, navigate)
```

### Pre-existing constraints (do NOT violate)

| Constraint | Source | Impact on Phase 56 |
|------------|--------|---------------------|
| Wrapper no try/catch, no message.error | LANDMINE #5 | 2 new wrappers (`writeSetAccessVlan` / `writePortBinding`) MUST follow the same pattern |
| Toast handled by `post()` interceptor only | LANDMINE #5 | 2 new Modals MUST NOT add local `message.error` calls |
| `canWrite` gating from menu store | D-09 / ROADMAP #4 笔误纠正 | 2 new menu items MUST be wrapped in `canWrite ? [..., vlanItem, bindItem] : []` |
| `batchInProgress` disables refresh+collect | D-07 / LANDMINE #4 | BulkWriteDrawer unchanged — already handles this |
| `form.resetFields()` on Modal open | v1.19 useEffect pattern | 2 new Modals MUST include `useEffect(() => { if (open) form.resetFields(); }, [open, form])` |
| `destroyOnHidden` on Modal | v1.19 default | 2 new Modals MUST set `destroyOnHidden` to prevent stale form state |
| `okButtonProps={{ loading: submitting }}` | v1.19 default | 2 new Modals MUST wire the submitting state |
| reason validation: `validateReasonRequired(rule, value, form)` cross-field | 55-01 WR-02 fix | 2 new Modals MUST use the same helper, NOT `validateReasonRequired` without `form` arg |
| Reason is required (5-200 chars, no empty) | D-02 | Both new Modals reason field is REQUIRED (not Optional like description) |
| Audit link via `<a href>` not `<Link>` | 53-02 Bug #1 fix | Phase 56 does NOT modify the helper — `showAuditLinkToast` is reused as-is |

---

## Registry Safety

| Registry | Blocks Used | Safety Gate |
|----------|-------------|-------------|
| shadcn official | none | N/A — no shadcn in project |
| Third-party (any) | none | N/A — Phase 56 introduces zero new blocks, zero new dependencies, zero new npm packages. All UI primitives (Modal / Drawer / Form / Select / InputNumber / Radio.Group / Input / Input.TextArea / Table / Tag / Card / Statistic / Alert) are antd 6.1 components already in the project since v1.19 |

**Phase 56 confirmation:**
- No `npx shadcn view` required
- No `package.json` diff
- No `go.mod` diff (backend) — see RESEARCH.md §Standard Stack for confirmation
- All visual contracts inherit from v1.19 shipped codebase

---

## Pre-Populated Decisions (from upstream artifacts)

| Source | Decisions Used |
|--------|----------------|
| `docs/plans/2026-07-09-v1.20.1-design.md` §8.2 | Modal field spec: vlanId InputNumber + reason; op Radio.Group + ip + mac + reason |
| `docs/plans/2026-07-09-v1.20.1-design.md` §8.3 | Audit Toast reuse — OperType semantics for Create/Update/Delete分流 |
| `docs/plans/2026-07-09-v1.20.1-design.md` §6 | OperType mapping (Create=1 / Update=2 / Delete=3) — affects backend operlog only, not UI copy |
| `.planning/phases/56-vlan-v1-20-1-0-5-plans-initiated-2026-07-09/56-RESEARCH.md` §Standard Stack | Zero new dependencies, full v1.19 inheritance |
| `.planning/phases/56-vlan-v1-20-1-0-5-plans-initiated-2026-07-09/56-RESEARCH.md` §Architecture Patterns | Pattern 4: BulkWriteDrawer zero-modification reuse (only ACTION_TITLE + ACTION_OPTIONS extension) |
| `.planning/phases/56-vlan-v1-20-1-0-5-plans-initiated-2026-07-09/56-RESEARCH.md` §Code Examples #5 | SetAccessVlanModal skeleton — confirms Form.Item + InputNumber + reason pattern |
| `xingran-react-frontend/src/components/network/port-write/PortWriteModal.tsx` | showAuditLinkToast helper, validateReasonRequired, validateReasonOptional, composeReason, PRESET_REASONS, REASON_MIN/MAX/CUSTOM_SENTINEL — all reused |
| `xingran-react-frontend/src/components/network/port-write/BulkWriteDrawer.tsx` | ACTION_OPTIONS array pattern, three-state machine, ResultView Statistic cards — pattern locked |
| `xingran-react-frontend/src/components/network/port-write/constants.ts` | REASON_MIN=5, REASON_MAX=200, REASON_CUSTOM_SENTINEL, composeReason, validateReasonRequired — locked helpers |
| `xingran-react-frontend/src/pages/network/ports/index.tsx` | ActionButtons integration pattern, canWrite gating, batchInProgress disable |
| `xingran-react-frontend/src/design-system/tokens/{spacing,typography,colors}.ts` | 8px spacing grid, antd typography defaults, semantic colors — all locked |
| `CLAUDE.md` (project) §Frontend | 中文 UI labels + antd wrapper functions + useEffect dependencies stable |

---

## Checker Sign-Off

- [ ] Dimension 1 Copywriting: PASS (all copy Chinese, locked from design doc §8.2 + v1.19 patterns)
- [ ] Dimension 2 Visuals: PASS (Modal/Drawer pattern inherited from Phase 53, zero new visual primitive)
- [ ] Dimension 3 Color: PASS (60/30/10 split preserved, accent reserved for primary CTA + audit link only, zero new hex)
- [ ] Dimension 4 Typography: PASS (antd defaults + design-system tokens, max 2 weights per view)
- [ ] Dimension 5 Spacing: PASS (8px grid, Form.Item vertical marginBottom=16px, zero exceptions)
- [ ] Dimension 6 Registry Safety: PASS (no shadcn, no third-party blocks, zero new dependencies)

**Approval:** pending (awaiting gsd-ui-checker verification)