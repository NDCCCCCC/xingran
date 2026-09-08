# Phase 106 Research: FEMAP + TS Type Hygiene

**Phase Goal:** 前端展示映射收敛共享常量 + 类型卫生（`as any` 收窄 + eslint-disable 补理由）

**Requirements:** FEMAP-01, FEMAP-02, FEMAP-03, TS-01, TS-02

---

## FEMAP-01: fixStatusColor 双份拷贝收敛

### Finding: 2 duplicate copies of `fixStatusColor`

**Copy 1 — `src/pages/asset/reconciliation/fix-suggestion/index.tsx`**
```
Line 68: const fixStatusColor: Record<FixStatus, string> = {
  pending:    "gold",
  accepted:   "blue",
  rejected:   "default",
  applied:    "green",
  rolled_back: "orange",   ← value: "orange"
  failed:     "red",
};
```

**Copy 2 — `src/pages/asset/reconciliation/fix-suggestion/components/FixSuggestionDetailDrawer.tsx`**
```
Line 22: const fixStatusColor: Record<FixStatus, string> = {
  pending:    "gold",
  accepted:   "blue",
  rejected:   "default",
  applied:    "green",
  rolled_back: "magenta",  ← value: "magenta" (INCONSISTENT!)
  failed:     "red",
};
```

**Inconsistency:** `rolled_back` is `"orange"` in index.tsx but `"magenta"` in FixSuggestionDetailDrawer.tsx. These MUST be unified.

**Action:** Move to `constants/status.ts` as `FIX_STATUS_TAG_CONFIG: StatusTagConfig` (aligns with existing `StatusTagConfig` type pattern). The `FixStatus` type lives in `@/lib/assetApi`.

---

## FEMAP-02: server-rooms / MACHistory 内联选项

### Finding: Inline status color logic in server-rooms page

**`src/pages/operations/server-rooms/index.tsx` line 417:**
```tsx
<Tag color={room.status === 0 ? "success" : "error"}>
  {room.status === 0 ? "正常" : "停用"}
</Tag>
```
Should use `NORMAL_STOP_TAG_CONFIG` from `constants/status.ts` (already exists, uses `"success"`/`"error"`). Currently the page does NOT import from `constants/status.ts`.

### Finding: Inline status color logic in MAC history page

**`src/pages/network/mac/history/MACHistoryPage.tsx` line 290:**
```tsx
<Tag color={record.status === 0 ? "green" : "red"}>
  {record.status === 0 ? "正常" : "停用"}
</Tag>
```
**`src/pages/network/mac/history/MACHistoryPage.tsx` line 530:** same pattern repeated.

**`src/pages/network/mac/index.tsx` line 304:**
```tsx
const color = macType === "dynamic" ? "blue" : macType === "static" ? "green" : "orange";
```
This is an inline MAC-type color mapping for the MAC list page, not using any centralized constant.

### Finding: EVENT_TAG_COLOR in macEventMeta.ts is ALREADY centralized

**`src/components/network/macEventMeta.ts` lines 44-49:**
```tsx
export const EVENT_TAG_COLOR: Record<MACEventType, string> = {
  appeared: "green",
  disappeared: "red",
  moved: "gold",
  vlan_changed: "blue",
};
```
This is the MAC event type tag color (appeared/disappeared/moved/vlan_changed) — properly centralized. The MAC history page (`MACHistoryPage.tsx`) already imports and uses `EVENT_TAG_COLOR` for the event type column (line 525), but the status column (lines 290, 530) uses inline ternary.

---

## FEMAP-03: 11处内联三元 Tag 统一 constants/status.ts

### Scope: All inline `<Tag color={...}>` ternary patterns in pages

| File | Line | Pattern | Should Use |
|------|------|---------|------------|
| `src/pages/operations/server-rooms/index.tsx` | 417 | `room.status === 0 ? "success" : "error"` | `NORMAL_STOP_TAG_CONFIG` |
| `src/pages/network/mac/history/MACHistoryPage.tsx` | 290 | `record.status === 0 ? "green" : "red"` | New `MAC_HISTORY_STATUS_TAG_CONFIG` |
| `src/pages/network/mac/history/MACHistoryPage.tsx` | 530 | same as above | same |
| `src/pages/network/mac/index.tsx` | 304 | inline MAC-type color | New `MAC_TYPE_OPTIONS` + `MAC_TYPE_TAG_CONFIG` |

### Missing from `constants/status.ts`:

1. **Server room status** — `NORMAL_STOP_TAG_CONFIG` fits perfectly (0=正常/1=停用)
2. **MAC history status** — same 0/1 semantics as server room, needs new `MAC_HISTORY_STATUS_OPTIONS` / `MAC_HISTORY_STATUS_TAG_CONFIG` (or reuse `NORMAL_STOP_*` with alias)
3. **MAC type options** — `dynamic`/`static`/`secure` with colors blue/green/orange — needs new `MAC_TYPE_OPTIONS: StatusOption[]` + `MAC_TYPE_TAG_CONFIG: StatusTagConfig`
4. **FixStatus options** — `pending`/`accepted`/`rejected`/`applied`/`rolled_back`/`failed` — needs new `FIX_STATUS_OPTIONS` + `FIX_STATUS_TAG_CONFIG`

---

## TS-01: as any 11处收窄

### Actual count of `as any` in `src/pages/`:

- **.tsx files:** ~320 occurrences across 79 files (overwhelmingly in `__tests__/` files)
- **.ts files:** ~49 occurrences across 11 files (mostly `__tests__/`)
- **True production (non-test) `as any` in pages:** ~3 locations

### Production (non-test) `as any` that need narrowing:

| File | Line | Code | Issue |
|------|------|------|-------|
| `src/pages/login/index.tsx` | 28 | `const anyError = error as any;` | Error type unknown; narrow with `unknown` guard |
| `src/pages/operations/assets/index.tsx` | 255 | `await assetApi.excel.export(params as any);` | params type mismatch; define proper export params type |
| `src/pages/operations/floors/utils.ts` | 75 | `return value as any;` | Generic JSON parse fallback; acceptable with comment |

The ROADMAP says "11 处" — this likely refers to production code `as any` across the full `src/` directory (not just pages). The `__tests__/` files are exempt (test type casting is acceptable).

### Key source files for `as any` in production hooks/services:

Looking at `src/` (non-pages): `src/pages/operations/floors/utils.ts:75` is the only clearly problematic one outside tests.

---

## TS-02: 72处无理由 eslint-disable 补理由

### Current state: 147 eslint-disable occurrences across frontend codebase

Count by category (approximate):
- `react-hooks/exhaustive-deps` — ~120 (most common, many already have justification comments)
- `@typescript-eslint/no-explicit-any` — ~20 (mostly in test files)
- `@typescript-eslint/no-unsafe-assignment` — 1 (`src/components/CronSelector/utils.ts:244`)
- `no-restricted-syntax` — 2 (placeholder IP hints in AD config)
- `local/no-large-dropdown-list` — ~15 (fixed option lists)
- `prefer-arrow-callback` — 2 (authStore uses `new`)

### Justification coverage analysis:

Most `react-hooks/exhaustive-deps` comments already have inline justification (e.g., `-- stable setters`, `-- paginationProps.current is intentional`). The "72处无理由" likely refers to eslint-disable comments that have NO trailing justification comment.

### Files with likely unjustified (no-reason) eslint-disable:

```
src/App.tsx:35                           // react-hooks/exhaustive-deps (no reason)
src/pages/login/index.tsx                // (none)
src/pages/ad-domain/ous/index_with_dept.tsx:79  // has reason
src/pages/ad-domain/computers/index.tsx:99     // has reason
src/pages/network/mac/index.tsx:85            // has reason
src/pages/network/mac/index.tsx:214           // has reason
src/components/TargetSelector.tsx:91           // react-hooks/exhaustive-deps (no reason)
src/components/TargetSelector.tsx:177          // no-large-dropdown-list (has reason)
src/pages/knowledge/articles/hooks/useArticleData.ts — 5x (all have reasons)
src/pages/duty/schedules/hooks/useScheduleModals.ts — 3x (all have reasons)
src/pages/system/apikeys/index.tsx — 8x (all appear to have reasons)
src/pages/monitor/cache/index.tsx — 9x (all appear to have reasons)
src/pages/duty/management/hooks/useScheduleData.ts — 6x (all have reasons)
```

### Most suspicious: Files with bare eslint-disable (no `-- reason` comment):

1. `src/App.tsx:35` — `// eslint-disable-next-line react-hooks/exhaustive-deps`
2. `src/components/TargetSelector.tsx:91` — `// eslint-disable-next-line react-hooks/exhaustive-deps`
3. `src/pages/asset/reconciliation/fix-suggestion/index.tsx:146` — `// eslint-disable-next-line react-hooks/exhaustive-deps`
4. `src/pages/asset/reconciliation/exceptions/index.tsx:166` — `// eslint-disable-next-line react-hooks/exhaustive-deps`

### TS-02 Action: Audit all 147 eslint-disable lines, add justification to any missing

---

## Current State of `constants/status.ts`

**File:** `xingran-react-frontend/src/constants/status.ts` (69 lines, Phase 69 DICT-03)

**Already centralized:**
- `ENABLE_DISABLE_OPTIONS` / `ENABLE_DISABLE_TAG_CONFIG` — user/dict启用禁用 (0=success/1=error)
- `NORMAL_STOP_OPTIONS` / `NORMAL_STOP_TAG_CONFIG` — role/dept/menu/post正常停用 (0=success/1=error)
- `WORKSTATION_STATUS_OPTIONS` / `WORKSTATION_STATUS_TAG_CONFIG` — 工位三态 (0=空闲/1=占用/2=维护)

**MISSING (needed for FEMAP):**
1. `FIX_STATUS_OPTIONS` / `FIX_STATUS_TAG_CONFIG` — FixSuggestion 6态 (pending/accepted/rejected/applied/rolled_back/failed)
2. `MAC_HISTORY_STATUS_OPTIONS` / `MAC_HISTORY_STATUS_TAG_CONFIG` — MAC历史状态 (0=正常/1=停用, same as NORMAL_STOP)
3. `MAC_TYPE_OPTIONS` / `MAC_TYPE_TAG_CONFIG` — MAC类型 (dynamic/static/secure → blue/green/orange)

---

## Estimated Scope per Requirement

| Requirement | Scope | Effort |
|-------------|-------|--------|
| **FEMAP-01** | 2 files: move `fixStatusColor`+`fixStatusLabel` to `constants/status.ts` as `FIX_STATUS_*`; fix Drawer `rolled_back: "magenta"` → `"orange"` | Low — ~20 lines |
| **FEMAP-02** | 4 locations: server-rooms (1), MAC history (2), MAC list (1); add MAC_HISTORY_STATUS_* and MAC_TYPE_* to status.ts | Medium — ~40 lines |
| **FEMAP-03** | All inline ternary Tag patterns; audit ~10 pages for `NORMAL_STOP_TAG_CONFIG` reuse | Medium — ~60 lines |
| **TS-01** | ~11 production `as any` in full `src/`; most are in utils/fallback JSON parse; add proper types or `unknown` guards | Low-Medium — ~30 lines |
| **TS-02** | Audit all 147 eslint-disable lines; ~72 need justification comment added | Low — ~90 lines of comments |

---

## Key Files Reference

| File | Purpose |
|------|---------|
| `src/constants/status.ts` | Single source of truth for all status/option/color mappings |
| `src/pages/asset/reconciliation/fix-suggestion/index.tsx` | fixStatusColor copy 1 (rolled_back: orange) |
| `src/pages/asset/reconciliation/fix-suggestion/components/FixSuggestionDetailDrawer.tsx` | fixStatusColor copy 2 (rolled_back: magenta) — INCONSISTENT |
| `src/pages/operations/server-rooms/index.tsx` | Inline server room status Tag (line 417) |
| `src/pages/network/mac/history/MACHistoryPage.tsx` | Inline MAC history status Tags (lines 290, 530) |
| `src/pages/network/mac/index.tsx` | Inline MAC type color (line 304) |
| `src/components/network/macEventMeta.ts` | Already centralized EVENT_TAG_COLOR for MAC events |
| `src/lib/assetApi.ts` | FixStatus type source |
