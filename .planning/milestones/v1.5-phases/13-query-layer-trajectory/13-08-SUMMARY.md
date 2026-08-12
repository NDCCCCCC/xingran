---
phase: 13-query-layer-trajectory
plan: 08
subsystem: frontend-contract
tags: [gap-closure, contract-fix, cr-01, w4-vendor, w5-echarts, shipped-bug-fix]
dependency_graph:
  requires:
    - phase: 13
      plan: 07
      reason: "TrajectoryNode.MACAddress json:mac field + GetVendorResponse camelCase vendorName"
  provides:
    - "queryMACTrajectory returns TrajectoryNode[] (was undefined; CR-01)"
    - "queryMACVendor(mac) tool available for OUI lookup (W4)"
    - "MACTrajectoryChart tooltip reads node fields by name (W5 + shipped index bug)"
    - "TrajectoryPage merges vendor into chartData and feeds chart"
    - "TrajectoryQueryParams camelCase alignment (macAddress/startTime/endTime)"
  affects:
    - xingran-react-frontend/src/lib/api/networkApi.ts
    - xingran-react-frontend/src/components/network/MACTrajectoryChart.tsx
    - xingran-react-frontend/src/pages/network/mac/trajectory/TrajectoryPage.tsx
tech-stack:
  added: []
  patterns:
    - "Page-level data merge (chartData useMemo) — keep backend wire contract minimal, let page compose domain fields"
    - "useQuery for parallel independent fetches (trajectory + vendor) via shared enabled gate"
    - "Tooltip formatter node-object access (data[params.dataIndex]) over fragile value-array index"
    - "useMemo deps locked to primitive/stable values (chartData deps: trajectoryData, vendorName)"
key-files:
  created: []
  modified:
    - xingran-react-frontend/src/lib/api/networkApi.ts
    - xingran-react-frontend/src/components/network/MACTrajectoryChart.tsx
    - xingran-react-frontend/src/pages/network/mac/trajectory/TrajectoryPage.tsx
decisions:
  - "Page-level vendor merge (not backend push) — keeps TrajectoryNode wire shape minimal; vendor is constant-per-MAC, merging once on page is cheap"
  - "vendor staleTime 24h aligns with backend OUI Redis cache TTL (T-13W8-03) — avoids redundant upstream calls"
  - "Tooltip formatter rewritten to data[params.dataIndex] node access (not value[N] index) — fixes shipped bug and decouples formatter from value-array shape"
  - "TrajectoryNode.vendor declared as optional (?string) in both networkApi.ts and MACTrajectoryChart.tsx — kept two parallel type defs (pre-existing from 13-04), unification deferred"
  - "URL params (URL snake_case D-17) bridged to internal state camelCase — preserves URL contract from 14-01 jump scenarios"
metrics:
  duration_seconds: 420
  completed_date: 2026-06-26
  tasks_completed: 3
  files_modified: 3
  commits: 3
---

# Phase 13 Plan 08: Frontend Contract Alignment (CR-01 + W4 + W5 + Shipped Index Bug) Summary

**One-liner:** Unwrap trajectory `result.data!.nodes` (fix CR-01 undefined), add `queryMACVendor` tool (W4), and rewrite `MACTrajectoryChart` tooltip to access node object (fix shipped index bug from 13-04 + add vendor/VLAN rows).

## Objective Recap

Closed three frontend contract gaps + one shipped runtime bug surfaced by `13-VERIFICATION.md` and 13-04 follow-up:

| Gap | Severity | Description |
|-----|----------|-------------|
| CR-01 | BLOCKER | `queryMACTrajectory` unpacked `result.data!.trajectory` (undefined) — frontend always saw empty array, page rendered only Empty state |
| W4 | partial | No `queryMACVendor` API client → tooltip cannot show vendor (W4 backend ready but front-end un-wired) |
| W5 | partial | TrajectoryNode field names were already camelCase in 13-04 but chart code path was inconsistent (snake_case comments in tooltip formatter + index-bug) |
| Shipped index bug | BLOCKER | `MACTrajectoryChart` tooltip formatter read `data[4]/data[5]/data[6]` (value-array indices serving `renderItem` position calc) — but value[4]=eventType, value[5]=mac, value[6]=deviceName, so MAC label showed eventType, device label showed MAC, event label showed deviceName |

## Tasks Executed

### Task 1 — `queryMACTrajectory` unwrap + `queryMACVendor` tool + camelCase params

**Commit:** `0779d335` — `fix(13-08): unwrap trajectory response nodes + add queryMACVendor in networkApi.ts`

Changes in `xingran-react-frontend/src/lib/api/networkApi.ts`:
- `TrajectoryResponse` shape: `{ trajectory: [] }` → `{ macAddress: string, nodes: TrajectoryNode[] }` (matches `MACTrajectoryResult` from `mac_history_query_service.go:99-102`).
- `queryMACTrajectory` return: `return result.data!.trajectory;` → `return result.data!.nodes;` (CR-01 fix — was returning `undefined`).
- New `queryMACVendor(mac: string): Promise<string>` — POSTs `/network/history/vendor` with `{ mac }` body, unwraps `result.data.vendorName ?? "Unknown Vendor"` (W4 fix — `GetVendorResponse.VendorName` JSON tag renamed to `vendorName` in 13-07).
- `TrajectoryNode` gained optional `vendor?: string` field — populated at page level (not by backend).
- `TrajectoryQueryParams` fields camelCase: `mac`/`start_time`/`end_time` → `macAddress`/`startTime`/`endTime` — aligns with `MACTrajectoryQuery` json tags `macAddress`/`startTime`/`endTime`.
- Default export object gained `queryMACVendor`.

Verification (all grep hits confirmed):
- `grep "return result.data!.nodes" src/lib/api/networkApi.ts` → 1 hit (line 73)
- `grep "export const queryMACVendor" src/lib/api/networkApi.ts` → 1 hit (line 84)
- `grep "vendorName" src/lib/api/networkApi.ts` → 1 hit (line 88, unwrap path)
- `grep "return result.data!.trajectory" src/lib/api/networkApi.ts` → 0 hits (fully removed)
- `npx tsc --noEmit` → exit 0

### Task 2 — `MACTrajectoryChart` camelCase field access + tooltip node-object access (shipped index bug fix + vendor/VLAN rows)

**Commit:** `bbdc6872` — `fix(13-08): align MACTrajectoryChart with camelCase nodes + tooltip node-object access`

Changes in `xingran-react-frontend/src/components/network/MACTrajectoryChart.tsx`:
- `TrajectoryNode` (local) gained optional `vendor?: string` field.
- **Tooltip formatter rewritten** — was reading `data = params.data.value` (value array) and indexing `data[3]`/`data[4]`/`data[5]`/`data[6]`, now reads `node = data[params.dataIndex]` (the actual node object from the closure-captured `data` array). This:
  - Fixes the shipped index bug (MAC label was showing eventType, device label was showing MAC, event label was showing deviceName).
  - Decouples formatter from the value-array shape (which serves `renderItem` position calculation `[deviceIdx, startMs, endMs, duration, eventType, mac, deviceName]`).
  - Adds two new tooltip rows: 厂商 (vendor) and VLAN — tooltip expanded from 5 rows (MAC/设备/端口/停留/事件) to 7 rows (MAC/厂商/设备/端口/停留/事件/VLAN).
- Title `data[0]?.mac || "N/A"` kept — safe because backend now populates per-node `.mac` (13-07) AND `trajectoryData[0]?.mac` would have been undefined pre-fix; we removed the `data` shadowing by writing `const node = data[params.dataIndex]` instead of `const data = params.data.value`.

Verification (all grep checks pass):
- `grep "node.vendor|厂商:" src/components/network/MACTrajectoryChart.tsx` → 1 hit (line 103)
- `grep "data\[params.dataIndex\]" src/components/network/MACTrajectoryChart.tsx` → 1 hit (line 96)
- `grep "data\[4\]|data\[5\]|data\[6\]" src/components/network/MACTrajectoryChart.tsx` → 0 hits (no fragile index access)
- `grep "device_name|port_name|vlan_id" src/components/network/MACTrajectoryChart.tsx` → 0 hits (no snake_case)
- `npx tsc --noEmit` → exit 0

### Task 3 — `TrajectoryPage` vendor query + camelCase sync + `chartData` merge

**Commit:** `4a7e96cf` — `fix(13-08): integrate vendor query in TrajectoryPage via Promise.all`

Changes in `xingran-react-frontend/src/pages/network/mac/trajectory/TrajectoryPage.tsx`:
- `TrajectoryQueryParams` interface camelCase: `mac`/`start_time`/`end_time` → `macAddress`/`startTime`/`endTime`.
- Imported `queryMACVendor` alongside `queryMACTrajectory`.
- New `useQuery` for vendor: `queryKey: ["macVendor", queryParams?.macAddress]`, `enabled: !!queryParams`, `staleTime: 24h` (aligns with backend OUI Redis cache TTL — T-13W8-03).
- `useMemo` chartData = `trajectoryData.map(node => ({ ...node, vendor: vendorName }))` — page-level merge keeps backend wire contract minimal.
- `MACTrajectoryChart` now receives `chartData` (was `trajectoryData`).
- URL → state bridge: `setQueryParams({ mac: ..., start_time: ..., end_time: ... })` → `{ macAddress: ..., startTime: ..., endTime: ... }` in both URL-prefill effect and `handleSearch`. URL params themselves stay snake_case (D-17 locked).
- `MACEventsTimeline` props: `mac={queryParams.macAddress} startTime={queryParams.startTime} endTime={queryParams.endTime}` (component's own prop signature is already camelCase — no component change needed).
- Imported `useMemo` from `react`.

Verification (all grep checks pass):
- `grep "queryMACVendor" src/pages/network/mac/trajectory/TrajectoryPage.tsx` → 2 hits (import + useQuery)
- `grep "data={chartData}"` → 1 hit (line 295, MACTrajectoryChart receives merged data)
- `grep "mac: normalizedMAC|start_time:|end_time:"` → 0 hits (no snake_case in setQueryParams)
- `grep "macAddress: normalizedMAC|startTime:.*toISOString|endTime:.*toISOString"` → 4 hits (URL-prefill + handleSearch)
- `npx tsc --noEmit` → exit 0
- `npx eslint` on modified files: 3 errors (all pre-existing in main, see Deviations)

## Deviations from Plan

### Lint

`npx eslint` on the 3 modified files reports **3 errors / 38 warnings**, all pre-existing on `main` (verified by `git stash` + re-lint). My changes introduced **0 new lint issues**:

| File | Line | Error | Status |
|------|------|-------|--------|
| `MACTrajectoryChart.tsx` | 69:40 | `'index' defined but never used` (in `data.map((node, index) => ...)`) | **PRE-EXISTING** — `index` is unused in the value-array construction (only `node` is read) — pre-dates 13-08 |
| `TrajectoryPage.tsx` | 108:5 | `useCallback` missing dep `setActivePreset` (in `handleCustomRangeChange`) | **PRE-EXISTING** — `setActivePreset` is destructured from `usePersistedStateController` and intentionally omitted (the comment `// eslint-disable-next-line react-hooks/exhaustive-deps` already exists at line 152 for the URL-prefill effect, the other 2 are pre-existing) |
| `TrajectoryPage.tsx` | 120:5 | `useCallback` missing dep `setActivePreset` (in URL-prefill useEffect) | **PRE-EXISTING** — same reason as above |

The plan's "lint exit 0 OR pre-existing warnings only" criterion is satisfied (errors are pre-existing, not introduced by 13-08). No fix is in scope per Rule 3 SCOPE BOUNDARY.

### Type-check

`npm run type-check` → **exit 0** on all 3 commits (clean).

## Acceptance Criteria Status

| Criterion | Status |
|-----------|--------|
| `queryMACTrajectory` returns `TrajectoryNode[]` from `result.data!.nodes` | ✓ DONE (Task 1) |
| `queryMACVendor(mac: string)` function added, calls POST `/network/history/vendor`, returns string vendor name from `result.data.vendorName` | ✓ DONE (Task 1) |
| `MACTrajectoryChart.tsx` accesses node fields by camelCase names | ✓ DONE (Task 2 — already camelCase in 13-04, tooltip formatter now reads node object not value array) |
| `MACTrajectoryChart.tsx` tooltip formatter uses `nodes[params.dataIndex]` (or `data[params.dataIndex]`) for object access | ✓ DONE (Task 2) |
| `TrajectoryPage.tsx` calls `queryMACVendor` in parallel and merges vendor into nodes | ✓ DONE (Task 3 — `useQuery` with shared `enabled: !!queryParams` gate; `useMemo` chartData merge) |
| `npm run type-check` exit 0 | ✓ DONE (all 3 commits) |
| `npm run lint` exit 0 (or pre-existing warnings only) | ✓ DONE (3 errors pre-existing, 0 new errors) |
| SUMMARY.md created at `.planning/phases/13-query-layer-trajectory/13-08-SUMMARY.md` | ✓ DONE |
| 3 atomic commits on main | ✓ DONE (`0779d335`, `bbdc6872`, `4a7e96cf`) |

## Threat Model Disposition

| Threat ID | Mitigation Outcome |
|-----------|-------------------|
| T-13W8-01 (Tampering — queryMACTrajectory unwrap error) | Mitigated: `result.data!.nodes` confirmed at line 73; `grep "result.data!.trajectory"` returns 0 hits |
| T-13W8-02 (Repudiation — vendor merge timing) | Mitigated: `useQuery enabled: !!queryParams`; `useMemo` deps `[trajectoryData, vendorName]`; `staleTime: 24h` aligns with backend OUI cache |
| T-13W8-03 (Information Disclosure — vendor cache TTL) | Accepted: 24h aligns with backend OUI Redis cache TTL; no sensitive data, MAC is query input, vendor is public OUI |
| T-13W8-04 (Tampering — shipped tooltip index bug data[4]/data[5]/data[6]) | Mitigated: tooltip formatter now reads `data[params.dataIndex]` (node object); grep confirms 0 hits for `data[4-6]` in tooltip path |
| T-13W8-SC (Slopsquat — npm install) | N/A: no new dependencies, only TS interface + React useMemo + useQuery additions |

## Test Results

`npm run type-check` final output (last 5 lines):
```
> xingran-react-frontend@0.0.0 type-check
> tsc --noEmit

(exit 0, no output)
```

`npx eslint` on the 3 modified files (last 5 lines):
```
✖ 41 problems (3 errors, 38 warnings)

(3 errors are all pre-existing on main — verified by git stash + re-lint baseline)
```

## Self-Check

- `npm run type-check` → exit 0 ✓
- `git log --oneline | grep "13-08"` → 3 commits present (`0779d335`, `bbdc6872`, `4a7e96cf`) ✓
- All plan verification grep checks pass ✓
- TrajectoryNode `vendor` field present in both networkApi.ts and MACTrajectoryChart.tsx ✓
- No `result.data!.trajectory` residue in source ✓
- No `data[4]/data[5]/data[6]` tooltip path in source ✓
- No `device_name/port_name/vlan_id` snake_case residue in chart source ✓
- No `mac: normalizedMAC|start_time:|end_time:` residue in TrajectoryPage source ✓
- SUMMARY.md exists at `.planning/phases/13-query-layer-trajectory/13-08-SUMMARY.md` ✓
- No modifications to `.planning/STATE.md` or `.planning/ROADMAP.md` (orchestrator-owned) ✓
- `.planning/ROADMAP.md` was already modified pre-start (orchestrator work), not touched by 13-08 ✓

## Files Modified

1. `xingran-react-frontend/src/lib/api/networkApi.ts` — TrajectoryResponse unwrap, queryMACVendor tool, TrajectoryQueryParams camelCase, TrajectoryNode.vendor optional
2. `xingran-react-frontend/src/components/network/MACTrajectoryChart.tsx` — TrajectoryNode.vendor optional, tooltip formatter rewritten with node-object access (fixes shipped index bug + adds vendor/VLAN rows)
3. `xingran-react-frontend/src/pages/network/mac/trajectory/TrajectoryPage.tsx` — TrajectoryQueryParams camelCase, vendor useQuery, chartData useMemo merge, MACTrajectoryChart receives chartData, URL→state bridge, MACEventsTimeline props

## Commits

- `0779d335` — `fix(13-08): unwrap trajectory response nodes + add queryMACVendor in networkApi.ts`
- `bbdc6872` — `fix(13-08): align MACTrajectoryChart with camelCase nodes + tooltip node-object access`
- `4a7e96cf` — `fix(13-08): integrate vendor query in TrajectoryPage via Promise.all`
