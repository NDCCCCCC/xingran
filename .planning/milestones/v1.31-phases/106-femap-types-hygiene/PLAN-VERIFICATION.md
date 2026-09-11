# Phase 106 Plan Verification

**Phase:** FEMAP Types Hygiene (FEMAP-01/02/03, TS-01/02)
**Verification date:** 2026-09-08
**Verdict:** NEEDS_CORRECTION

---

## PLAN-01 — PASS with minor correction needed

**Files:** `status.ts`, `fix-suggestion/index.tsx`, `FixSuggestionDetailDrawer.tsx`

### Step 1 — `status.ts` append (lines 1–69)

| Check | Result |
|-------|--------|
| File has exactly 69 lines | PASS — confirmed |
| `StatusTagConfig = Record<number, { text: string; color: string }>` | PASS — confirmed at line 24 |
| `MAC_HISTORY_STATUS_TAG_CONFIG` using `number` keys (0, 1) | PASS — type matches `StatusTagConfig` |
| `FIX_STATUS_TAG_CONFIG` using `string` keys | PASS — type `Record<string, ...>` is valid |
| `MAC_TYPE_TAG_CONFIG` using `string` keys | PASS — type `Record<string, ...>` is valid |

**Note:** `FIX_STATUS_OPTIONS` and `FIX_STATUS_TAG_CONFIG` use `string` keys (FixStatus is a string union), which is intentionally different from `StatusTagConfig` (which is `Record<number, ...>`). This is correctly handled in the plan with `Record<string, ...>`.

### Step 2 — `fix-suggestion/index.tsx`

| Line | Expected | Found | Result |
|------|----------|-------|--------|
| 68 | `fixStatusColor` definition start | `const fixStatusColor: Record<FixStatus, string> = {` | PASS |
| 77 | `fixStatusLabel` definition start | `const fixStatusLabel: Record<FixStatus, string> = {` | PASS |
| 382 | Status column render | `render: (v: FixStatus) => <Tag color={fixStatusColor[v]}>{fixStatusLabel[v]}</Tag>` | PASS |
| 449 | Action column fallback | `return <Tag>{fixStatusLabel[record.fixStatus]}</Tag>` | PASS |
| 509 | Select options | `options={Object.entries(fixStatusLabel).map(([value, label]) => ({ value, label }))}` | PASS |

`rolled_back: "orange"` confirmed at line 73 (index.tsx — correct value, no bug here).

### Step 3 — `FixSuggestionDetailDrawer.tsx`

| Line | Expected | Found | Result |
|------|----------|-------|--------|
| 22 | `fixStatusColor` with `rolled_back: "magenta"` | `rolled_back: "magenta"` | PASS — bug confirmed |
| 246 | Timeline color | `color: fixStatusColor[h.fixStatus],` | PASS |
| 250 | Tag color | `color: fixStatusColor[h.fixStatus]` | PASS |
| 251 | Tag text | `{fixStatusLabel[h.fixStatus]}` | PASS |

### Corrections needed

**1. Line number for `fixStatusLabel` in index.tsx:** The plan says "lines 68–84" but the objects are at lines 68–75 (`fixStatusColor`) and 77–84 (`fixStatusLabel`). This is a minor imprecision but the intent is clear — the plan correctly identifies both objects.

**2. MAC_HISTORY_STATUS_TAG_CONFIG color values:** The plan specifies `"green"` for status 0 and `"red"` for status 1. The existing `NORMAL_STOP_TAG_CONFIG` uses `"success"` and `"error"` (Ant Design semantic colors). The MAC history page currently uses literal `"green"`/`"red"`. Using literal colors (`"green"`/`"red"`) in `MAC_HISTORY_STATUS_TAG_CONFIG` is consistent with the existing MAC history page behavior and is acceptable.

**No other corrections needed.** Plan 01 is otherwise accurate and executable.

---

## PLAN-02 — PASS with one import note

**Files:** `server-rooms/index.tsx`, `MACHistoryPage.tsx`, `network/mac/index.tsx`

### server-rooms/index.tsx

| Check | Result |
|-------|--------|
| Inline ternary at line 417 | PASS — confirmed: `room.status === 0 ? "success" : "error"` |
| No existing import from `@/constants/status` | PASS — confirmed absent |

### MACHistoryPage.tsx

| Check | Result |
|-------|--------|
| Inline ternary at line 290 | PASS — confirmed: `record.status === 0 ? "green" : "red"` |
| Inline ternary at line 530 | PASS — confirmed: same pattern |
| No existing import from `@/constants/status` | PASS — confirmed absent |

### network/mac/index.tsx

| Check | Result |
|-------|--------|
| MAC-type color at line 304 | PASS — confirmed: `macType === "dynamic" ? "blue" : macType === "static" ? "green" : "orange"` |
| `macTypeOptions` local array at line 251 | PASS — this will be replaced by `MAC_TYPE_OPTIONS` import |

### Dependency note

Plan 02 depends on Plan 01. The plan correctly notes that `MAC_HISTORY_STATUS_TAG_CONFIG` and `MAC_TYPE_TAG_CONFIG` must be added to `status.ts` by Plan 01 first.

### Minor type concern

The plan uses `NORMAL_STOP_TAG_CONFIG[room.status]?.color ?? "default"` for server rooms. Since `NORMAL_STOP_TAG_CONFIG` is `StatusTagConfig = Record<number, ...>` and `room.status` is `number`, this is type-safe. However, the plan's fallback text `{room.status === 0 ? "正常" : "停用"}` is a bit odd since `NORMAL_STOP_TAG_CONFIG` already has `text` for both 0 and 1. A simpler fallback would be `NORMAL_STOP_TAG_CONFIG[room.status]?.text ?? "未知"` but the plan's approach is functionally equivalent and safe.

**No corrections needed.** Plan 02 is accurate and executable.

---

## PLAN-03 — PASS

**Files:** `login/index.tsx`, `assets/index.tsx`, `floors/utils.ts`

### login/index.tsx

| Check | Result |
|-------|--------|
| Line 28: `const anyError = error as any;` | PASS — confirmed |

The plan's proposed replacement correctly avoids `as any` by using narrow typed casts at each use site. The approach is sound.

### assets/index.tsx

| Check | Result |
|-------|--------|
| Line 255: `await assetApi.excel.export(params as any);` | PASS — confirmed |

The plan's fix (typing `params` as `Record<string, unknown>` explicitly) will work. The `assetApi.excel.export` accepts `Record<string, unknown>`.

### floors/utils.ts

| Check | Result |
|-------|--------|
| Line 75: `return value as any;` with `eslint-disable-next-line` | PASS — confirmed |

The proposed additional justification comment is appropriate for this JSON parse fallback.

**No corrections needed.** Plan 03 is accurate and executable.

---

## PLAN-04 — PASS

**Files:** `App.tsx`, `TargetSelector.tsx`, `fix-suggestion/index.tsx`, `exceptions/index.tsx`

| File | Line | Content | Result |
|------|------|---------|--------|
| App.tsx | 35 | `// eslint-disable-next-line react-hooks/exhaustive-deps` | PASS — confirmed, no reason |
| TargetSelector.tsx | 91 | `// eslint-disable-next-line react-hooks/exhaustive-deps` | PASS — confirmed, no reason |
| fix-suggestion/index.tsx | 146 | `// eslint-disable-next-line react-hooks/exhaustive-deps` | PASS — confirmed, no reason |
| exceptions/index.tsx | 166 | `// eslint-disable-next-line react-hooks/exhaustive-deps` | PASS — confirmed, no reason |

**No corrections needed.** Plan 04 is accurate and executable.

---

## Wave dependency check

- Wave 1 (Plans 01, 02): No dependencies
- Wave 2 (Plans 03, 04): Depend on Plan 01 ✓ — correctly declared

---

## Summary

| Plan | Status | Corrections |
|------|--------|-------------|
| PLAN-01 | NEEDS_CORRECTION | 1. Minor: object range "lines 68–84" should note two separate objects (68-75 and 77-84); 2. MAC_HISTORY_STATUS_TAG_CONFIG colors must use literal `"green"`/`"red"` (not `"success"`/`"error"`) to match MAC history page's existing literal colors |
| PLAN-02 | APPROVED | None |
| PLAN-03 | APPROVED | None |
| PLAN-04 | APPROVED | None |

**Overall verdict:** NEEDS_CORRECTION — apply the two corrections to PLAN-01, then all four plans are ready for execution.
