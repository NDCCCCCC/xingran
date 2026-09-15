# v1.33 Release Notes

**Date:** 2026-09-15
**Scope:** Frontend performance audit fixes (Vercel React Best Practices)

---

## Resolved Findings

| ID | Severity | Area | Fix | Commit |
|----|----------|------|-----|---------|
| F-01 | High | Performance | `main.tsx` — eliminate white-screen render gate; render immediately with inline skeleton while encryption config loads in background | `d649767` |
| F-02 | High | Rerender | `CADFloorPlanEditor` — eliminate `React.memo` bailout across 4 element types (Wall/Door/Workstation/CADText); stable `useCallback` handlers via `data-*` attribute self-identification | `05cec17` |
| F-03 | High | JS Perf | `network/backups/utils.ts` — replace O(n²) `unshift` with `push+reverse` in computeDiff; add regression tests | `bc610b0` |
| F-04 | High | Bundle | `vite.config.ts` — extract `react-grid-layout`, `@dnd-kit/*`, `@breejs/later` into dedicated chunks; vendor-react 1980KB→1919KB (−61KB) | `d9c4660` |
| F-05 | High | Client Data | `ad-domain/logs/index.tsx` — migrate configs to `useQuery` (5min staleTime) + logs to `useTableQuery`; eliminates redundant refetch on every paginate | `e9f6564` |

## Audit Summary

Full codebase audit against 49 applicable Vercel React Best Practices rules.
**Result:** 42 findings — 5 fixed, 1 manual-only (F-06 react-query 71-page migration → next milestone), 36 P2 deferred.

| Rule Category | P0 | P1 | P2 | Zero-violation areas |
|---|---|---|---|---|
| Bundle / Async | 0 | 2 | 7 | Route lazy-split, echarts/three/xlsx按需, no waterfall on primary paths |
| Rerender | 0 | 1 | 18 | 204 useEffect deps clean, no infinite loops, Zustand selectors stable |
| Rendering / Client | 0 | 2 | 4 | 0 conditional-render leaks, passive listeners, content-visibility, localStorage versioning |
| JS / Advanced | 0 | 1 | 7 | 0 RegExp-in-loop, O(n²) fixed, init-once, toSorted immutable |

## Build Artifacts

| Chunk | Size | Change |
|-------|------|---------|
| `vendor-react` | 1,919 KB (gzip: 612 KB) | −61 KB from baseline |
| `vendor-three` | 911 KB | Unchanged (lazy) |
| `vendor-echarts` | 611 KB | Unchanged (lazy) |
| `vendor-grid` | 35.6 KB | **New** — react-grid-layout |
| `vendor-dnd` | 42.9 KB | **New** — @dnd-kit/* |
| `vendor-cron` | 25.7 KB | **New** — @breejs/later |
| `index` (entry) | 132 KB | Unchanged |

## Pending

- **F-06 (Manual-only):** react-query coverage ~14% → migrate 71 pages from hand-written `useEffect+setState` to `useTableQuery`. Recommend as next milestone.
- **36 P2 findings:** columns factory pattern (10 pages), prop→state mirrors, etc. — acceptable tech debt.

---

## Commits

```
d649767 fix(perf): resolve F-01 — eliminate white-screen render gate
d9c4660 fix(bundle): resolve F-04 — split react-grid-layout/dnd-kit/breejs into dedicated chunks
05cec17 fix(cad-editor): resolve F-02 — eliminate memo bailout via stable handlers
e9f6564 fix(ad-domain): resolve F-05 — configs useQuery + logs useTableQuery
bc610b0 fix(backups): resolve F-03 — replace unshift with push+reverse in computeDiff (O(n²)→O(n))
```
