# Phase 118: Code Review Report

**Reviewed:** 2026-09-14T00:00:00Z
**Depth:** standard
**Files Reviewed:** 9
**Status:** issues_found

## Summary

Phase 118 implements three data-fetch improvements: VDI server query deduplication via React Query (DATA-01), sessionStorage menu hydrate-then-revalidate (DATA-02), and useColumnConfig cache-hit early-return (DATA-03), plus Promise.all parallelization of holiday year/list fetches (DATA-04). The implementation is generally sound with good regression test coverage (44/44 passing). Two blocking issues were found: the VDI server query cache is never invalidated after mutations (DATA-01), and VDIServerConfig bypasses the React Query cache entirely creating an inconsistency. One info-level issue: `clearAllTableState` does not clear `MENU_CACHE` or `LAST_PATH` on logout.

---

## Critical Issues

### CR-01: VDI server query cache not invalidated after server mutations

**File:** `xingran-react-frontend/src/pages/vdi/VirtualMachineList/index.tsx:123-128`
**Issue:** `queryKeys.vdi.servers()` is set up with `staleTime: 5 * 60 * 1000` (5 minutes). After `handleCreate` (line 515), `handleDelete` (line 559), `handleSync` (line 577), or `handleBindUser` (line 623), the VM list is refreshed via `loadVMs()` but the VDI server dropdown cache is **never invalidated**. The server dropdown (lines 993-999) renders `vdiServers` from the shared cache. A server created/deleted/disabled on VDIServerConfig would remain visible (or still-filtered-as-available) in the VM creation dropdown for up to 5 minutes.

**Failure Scenario:** Admin disables a VDI server on VDIServerConfig. User opens VM creation modal within 5 minutes — the disabled server still appears in the dropdown with `status === 0` (if filtering by `s.status === 0`), allowing the user to attempt VM creation against a known-bad server.

**Fix:** Reduce `staleTime` to 30 seconds so the server list naturally re-fetches more frequently:
```typescript
const { data: serverData } = useQuery({
  queryKey: queryKeys.vdi.servers(),
  queryFn: () => vdiServerApi.list({ current: 1, pageSize: 100 }),
  enabled: canCreateVM,
  staleTime: 30 * 1000, // 30s — server list changes infrequently, 5min is too long
});
```

### CR-02: VDIServerConfig bypasses React Query deduplication cache

**File:** `xingran-react-frontend/src/pages/vdi/VDIServerConfig/index.tsx:40-51`
**Issue:** VDIServerConfig uses raw `vdiServerApi.list()` calls (lines 43, 131) wrapped in manual `useState` + `useEffect` loading. It does not use `useQuery` with `queryKeys.vdi.servers()`. This means:

1. VDIServerConfig create/delete operations do not benefit from shared React Query caching/deduplication
2. VDIServerConfig's `loadServers()` is not connected to the VirtualMachineList's `serverData` cache — they are two independent data sources that can diverge
3. The DATA-01 deduplication goal (single shared cache for all VDI server reads on the page) is only partially achieved — VirtualMachineList uses the cache, VDIServerConfig does not

When the admin creates a new server on VDIServerConfig, VirtualMachineList's cache is not invalidated (CR-01), AND VDIServerConfig does not read from that cache either. The two components independently maintain separate server lists with separate refresh timing.

**Fix:** VDIServerConfig should use `useQuery` with `queryKeys.vdi.servers()` instead of raw API calls, and `handleCreate`/`handleDelete` should call `queryClient.invalidateQueries({ queryKey: queryKeys.vdi.servers() })` to keep VirtualMachineList's cache fresh. Alternatively, use a shared `useVDIServers()` hook.

---

## Warnings

### WR-01: `clearAllTableState` does not clear `MENU_CACHE` or `LAST_PATH`

**File:** `xingran-react-frontend/src/constants/storage.ts:107-119`
**Issue:** `clearAllTableState()` iterates `sessionStorage` for keys prefixed with `TABLE_STATE_PREFIX` only. The `MENU_CACHE` (STORAGE_KEYS.MENU_CACHE) and `LAST_PATH` (STORAGE_KEYS.LAST_PATH) keys are NOT cleared. However, `menuStore.clearMenus()` (called on logout + 401) DOES clear `MENU_CACHE` via `sessionStorage.removeItem(STORAGE_KEYS.MENU_CACHE)` (menuStore.ts line 194). The `LAST_PATH` is never cleared on logout — `clearLastPath()` is defined in DynamicRoutes.tsx but not called by authStore.logout().

**Failure Scenario:** User A logs in, visits `/operations/buildings`, then logs out. User B logs in on the same browser tab. User B will be redirected to `/operations/buildings` (User A's last path) instead of `/dashboard` on first render.

**Fix:** Add `clearLastPath()` call to authStore's logout handler, or integrate it into `clearAllTableState` by also handling the LAST_PATH key.

### WR-02: `readMenuCache` / `writeMenuCache` / `getLastPath` lack `typeof window` guards

**File:** `xingran-react-frontend/src/router/DynamicRoutes.tsx:50-89`
**Issue:** `readMenuCache`, `writeMenuCache`, and `getLastPath` directly access `sessionStorage` without checking `typeof window !== 'undefined'`. While DynamicRoutes.tsx is client-only, other callers of `clearTableStateByPath` / `clearAllTableState` (storage.ts lines 86-119) DO have `if (typeof window === 'undefined') return` guards. Inconsistent SSR safety.

**Fix:** Add `if (typeof window === 'undefined') return null` guards to `readMenuCache` and `getLastPath`, and `if (typeof window === 'undefined') return` to `writeMenuCache` and `clearLastPath`.

---

## Info

### IN-01: `clearAllTableState` calls `sessionStorage.removeItem` inside try-catch with empty body

**File:** `xingran-react-frontend/src/constants/storage.ts:107-119`
**Issue:** The `catch` block has no comment explaining why storage errors are silently swallowed. Other try-catches in the same file at least have trailing comments. This is minor but inconsistent.

### IN-02: VDI server `staleTime` inconsistency between App.tsx default and VirtualMachineList

**File:** `xingran-react-frontend/src/App.tsx:18` vs `VirtualMachineList/index.tsx:127`
**Issue:** App.tsx sets `defaultOptions.queries.staleTime = 5 * 60 * 1000` (5 min). VirtualMachineList explicitly sets `staleTime: 5 * 60 * 1000` for the vdi query (line 127), matching the default but being redundant. Consider relying on the default instead (or documenting why explicit override is needed).

### IN-03: `useColumnConfig` early-return test validates API is NOT called on cache hit

**File:** `xingran-react-frontend/src/hooks/useColumnConfig.test.tsx:88-103`
**Note:** This is a positive finding — the test at line 101 explicitly asserts `expect(columnConfigApiMock.getByPageKey).not.toHaveBeenCalled()`, correctly validating the DATA-03 short-circuit behavior. No issue here.

---

## Structural Findings (fallow)

None — no structural_findings JSON payload was provided.

---

## Verdict

| Finding | Severity | Category | File:Line | Status |
|---------|----------|----------|-----------|--------|
| VDI server query cache never invalidated after mutations | BLOCKER | Correctness | VirtualMachineList/index.tsx:123-128 | Must fix |
| VDIServerConfig bypasses React Query cache, inconsistent with VirtualMachineList | BLOCKER | Correctness | VDIServerConfig/index.tsx:40-51 | Must fix |
| LAST_PATH not cleared on logout | WARNING | Security/UX | storage.ts:107-119, DynamicRoutes.tsx:104-109 | Should fix |
| sessionStorage access lacks SSR guards | WARNING | Code Quality | DynamicRoutes.tsx:50-89 | Should fix |
| clearAllTableState catch block silent | INFO | Code Quality | storage.ts:117-118 | Minor |
| VDI staleTime explicitly set to same value as App default | INFO | Code Quality | VirtualMachineList/index.tsx:127 | Minor |

**BLOCKER count: 2 | WARNING count: 2 | Info count: 2**

---

_Reviewed: 2026-09-14T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
