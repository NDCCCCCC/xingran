# Phase 106 Plan 01 — FEMAP Types Hygiene: Summary

## Status: COMPLETE

## Changes

### Task 1 — Read status.ts
Confirmed: `StatusTagConfig = Record<number, { text: string; color: string }>`, 69 lines, existing constants: `ENABLE_DISABLE_OPTIONS/TAG_CONFIG`, `NORMAL_STOP_OPTIONS/TAG_CONFIG`, `WORKSTATION_STATUS_OPTIONS/TAG_CONFIG`.

### Task 2 — Append 3 constant blocks to status.ts

**Block A — FixStatus:**
- `FIX_STATUS_OPTIONS`: 6 options (pending/accepted/rejected/applied/rolled_back/failed)
- `FIX_STATUS_TAG_CONFIG`: color map, key bugfix: `rolled_back: "orange"` (Drawer previously had `"magenta"`)

**Block B — MAC History Status:**
- `MAC_HISTORY_STATUS_OPTIONS`: 0=正常, 1=停用
- `MAC_HISTORY_STATUS_TAG_CONFIG`: 0=green, 1=red

**Block C — MAC Type:**
- `MAC_TYPE_OPTIONS`: dynamic/static/secure
- `MAC_TYPE_TAG_CONFIG`: dynamic=blue, static=green, secure=orange

**Also widened `StatusOption.value` from `number` to `number | string`** to support string-keyed status types (FixStatus, MAC_TYPE).

### Task 3 — Migrate fix-suggestion/index.tsx
- Added import: `FIX_STATUS_OPTIONS`, `FIX_STATUS_TAG_CONFIG` from `@/constants/status`
- Deleted local `fixStatusColor` and `fixStatusLabel` objects
- Updated status column render: `FIX_STATUS_TAG_CONFIG[v]?.color ?? "default"` / `FIX_STATUS_TAG_CONFIG[v]?.text ?? v`
- Updated action column fallback tag: `FIX_STATUS_TAG_CONFIG[record.fixStatus]?.text ?? record.fixStatus`
- Updated Select options: `FIX_STATUS_OPTIONS`

### Task 4 — Migrate FixSuggestionDetailDrawer.tsx
- Added import: `FIX_STATUS_TAG_CONFIG` from `@/constants/status`
- Deleted local `fixStatusColor` (`rolled_back: "magenta"`) and `fixStatusLabel`
- Updated history timeline `color` and `Tag` render to use `FIX_STATUS_TAG_CONFIG[h.fixStatus]?.color ?? "default"` / `FIX_STATUS_TAG_CONFIG[h.fixStatus]?.text ?? h.fixStatus`
- **Key bugfix**: Drawer previously used `rolled_back: "magenta"`; now correctly uses `"orange"` via centralized constant

### Task 5 — Verify
- `npm run type-check`: PASS (no errors)
- `npm run lint`: PASS (0 errors, 1379 pre-existing warnings)
- `npm run test -- --run src/lib/apiFactory.invariants.test.ts`: PASS (10/10 tests)

### Task 6 — Type fix in user/constants.ts
`SelectOption.value` widened from `number` to `number | string` to match the updated `StatusOption` interface, resolving the type incompatibility introduced by the widening change.

## Key Bugfix
`rolled_back` status color unified: `FixSuggestionDetailDrawer.tsx` had `rolled_back: "magenta"` (wrong), `index.tsx` had `rolled_back: "orange"` (correct). Both now use the centralized `FIX_STATUS_TAG_CONFIG["rolled_back"].color = "orange"`.
