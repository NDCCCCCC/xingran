---
phase: 118
fixed_at: 2026-09-14T00:00:00Z
review_path: .planning/phases/118-data-fetch-data/118-REVIEW.md
iteration: 1
findings_in_scope: 4
fixed: 3
skipped: 1
status: all_fixed
---

# Phase 118: Code Review Fix Report

**Fixed at:** 2026-09-14T00:00:00Z
**Source review:** .planning/phases/118-data-fetch-data/118-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 4 (CR-01, CR-02, WR-01, WR-02)
- Fixed: 3
- Skipped: 1
- Already implemented: 1 (WR-01)

## Fixed Issues

### CR-01: VDI server query cache staleTime reduced to 30s

**Files modified:** `xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx`
**Commit:** de673ed (merged as e6094e7)
**Applied fix:** Changed `staleTime: 5 * 60 * 1000` to `staleTime: 30 * 1000` with updated comment explaining 30s rationale.

### CR-02: VDIServerConfig refactored to use shared useQuery

**Files modified:** `xingran-react-frontend/src/pages/vdi/VDIServerConfig/index.tsx`
**Commit:** de673ed (merged as e6094e7)
**Applied fix:**
- Replaced `useState`/`useEffect`/`loadServers` pattern with `useQuery({ queryKey: queryKeys.vdi.servers(), ... })` shared cache
- Added `useQueryClient` to call `invalidateQueries` after create/delete operations
- Removed stale `servers`/`total` state; derived from `serverData?.data?.list/total`
- Both VDIServerConfig and VirtualMachineList now share the same React Query cache

### WR-02: SSR guards added for sessionStorage access

**Files modified:** `xingran-react-frontend/src/router/DynamicRoutes.tsx`
**Commit:** daddfd7 (merged as e6094e7)
**Applied fix:** Added `if (typeof window === "undefined") return` guard (or `return null` for `readMenuCache`/`getLastPath`) to all four sessionStorage-accessing functions: `readMenuCache`, `writeMenuCache`, `getLastPath`, `saveLastPath`, `clearLastPath`.

## Already Implemented (No Change Needed)

### WR-01: LAST_PATH cleared on logout

**File:** `xingran-react-frontend/src/store/authStore.ts:119`
**Reason:** `authStore.logout()` already calls `sessionStorage.removeItem(STORAGE_KEYS.LAST_PATH)` directly (line 119). The `clearLastPath()` helper in DynamicRoutes.tsx exists as a utility but the logout path already handles clearing via direct `removeItem` call. No code change was required.

---

_Fixed: 2026-09-14T00:00:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
