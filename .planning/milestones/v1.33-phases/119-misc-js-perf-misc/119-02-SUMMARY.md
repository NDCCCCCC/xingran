# Phase 119 Plan 02 Summary

## Plan Info
- **Phase:** 119-misc-js-perf-misc
- **Plan:** 119-02
- **Requirements:** MISC-03, MISC-04

## Objective
MISC-03 + MISC-04: Two pure performance refactors, zero behavior change, zero new tests.
- useTableManager lazy sessionStorage initialization eliminates 24 list pages' first-frame blocking
- TabBar scroll short-circuit eliminates redundant setState-triggered re-renders

## Changes Made

### MISC-03: useTableManager filtersRef Lazy Initialization
**File:** `xingran-react-frontend/src/hooks/useTableManager.ts`

Changed `filtersRef` from eager to lazy initialization:
- `useRef<Record<string, unknown> | null>(null)` — initial null, not `readInitialFilters(...)` result
- Added null-guard lazy read in `persistFilters`: `if (filtersRef.current === null) { filtersRef.current = readInitialFilters(filtersStorageKey); }`
- Added null-guard lazy read in `clearPersistedFilters`: same pattern
- sessionStorage sync read deferred to first access, not mount time

**Artifacts:**
- `useRef<Record<string, unknown> | null>(null)` present (grep: line 174)
- `filtersRef.current === null` guard present (grep: lines 179, 196)

### MISC-04: TabBar updateScrollState Scroll State Short-Circuit
**File:** `xingran-react-frontend/src/components/layout/shared/TabBar.tsx`

Added value comparison before `setScrollState` to skip redundant updates:
```typescript
if (
  state.canScrollLeft === scrollState.canScrollLeft &&
  state.canScrollRight === scrollState.canScrollRight &&
  state.scrollLeft === scrollState.scrollLeft
) { return; }
```
- Added `scrollState` to `useCallback` deps (required for stable reference comparison)
- No `Object.is` needed — direct value comparison sufficient for the 3 scalar fields

**Artifacts:**
- `canScrollLeft === scrollState.canScrollLeft` pattern present
- `useCallback` deps now includes `scrollState`

## Deviation
None — plan executed exactly as written.

## Tests
- `TabBar.render.test.tsx`: 2/2 passed
- No new tests required per plan spec (pure performance refactor, zero behavior change)

## Verification
- `useRef<Record<string, unknown> | null>(null)` in useTableManager.ts: present
- `filtersRef.current === null` guard: present (2 occurrences)
- `canScrollLeft === scrollState.canScrollLeft` in TabBar.tsx: present
- `npm run type-check`: exit 0
- `npm run lint`: 0 errors (1366 warnings, all pre-existing)

## Commit
- `8aae807` — feat(119-02): misc-03/04 lazy init and scroll short-circuit
