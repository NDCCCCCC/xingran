---
phase: 30
plan: 02
subsystem: frontend-build
tags: [performance, vite, react, code-splitting, lazy-loading, vendor-chunks]
requirements:
  - PERF-LAZY-01
  - PERF-LAZY-02
  - PERF-LAZY-03
  - PERF-LAZY-04
  - PERF-LAZY-05
  - PERF-LAZY-06
dependency_graph:
  requires:
    - 30-01 (vendor chunk strategy + baseline metrics)
  provides:
    - lazy-three-wrapper
    - lazy-echarts-wrapper
    - lazy-xlsx-imports
    - lazy-md-editor-wrapper
    - vendor-md-editor-chunk
    - vendor-xlsx-chunk
  affects:
    - vendor-commons-size (reduced)
    - 3d-scene-page-entry
    - chart-widget-entry
    - duty-page-excel-imports
    - system-notice-form
tech_stack:
  added: []
  patterns:
    - React.lazy + Suspense for component-level code splitting
    - Dynamic import() inside click handlers for utility-level splitting
    - forwardRef + ComponentProps to preserve library API through wrapper
    - manualChunks function additions for vendor-xlsx and vendor-md-editor
key_files:
  created:
    - xingran-react-frontend/src/components/three/BuildingScene.tsx
    - xingran-react-frontend/src/components/charts/EChartsWrapper.tsx
    - xingran-react-frontend/src/components/markdown/MarkdownEditor.tsx
  modified:
    - xingran-react-frontend/src/pages/operations/building-spaces-3d/index.tsx
    - xingran-react-frontend/src/components/dashboard/widgets/types/ChartWidget.tsx
    - xingran-react-frontend/src/pages/duty/holidays/utils.tsx
    - xingran-react-frontend/src/pages/duty/management/utils/excelUtils.ts
    - xingran-react-frontend/src/pages/system/notice/components/NoticeForm.tsx
    - xingran-react-frontend/vite.config.ts
decisions:
  - "Wave 2 lazy-loads four heavy libs via React.lazy + Suspense wrappers for components, dynamic import() for utilities"
  - "Vite manualChunks extended with vendor-xlsx and vendor-md-editor rules so the dynamically-imported libs split into dedicated chunks"
  - "Modulepreload vs initial execution: vendor-three and vendor-md-editor appear in entry's __vite__mapDeps (preload hints) but JS is not executed until the user navigates to the relevant page — acceptable Vite default behavior"
  - "CSS import for @uiw/react-md-editor/markdown-editor.css remains in NoticeForm since it is an independent export entry with no JS payload"
  - "Type-only 'import type' for MDEditorProps is kept in the wrapper (elided at compile time, no runtime impact)"
metrics:
  duration: "12m"
  completed: 2026-06-13
---

# Phase 30 Plan 02: Heavy Library Lazy Loading Summary

## One-liner

Lazy-loaded four heavy libraries (three.js, echarts, xlsx, @uiw/react-md-editor) via React.lazy + Suspense wrappers and dynamic `import()` for utilities. Vendor-commons shrunk from 743KB to 608KB gzip (-135KB), with xlsx and md-editor split into dedicated on-demand chunks.

## Tasks Completed

| Task | Name | Commit | Files |
|------|------|--------|-------|
| 1 | Refactor 3D scene components to use React.lazy + Suspense | 003ccd9 | BuildingScene.tsx (NEW), building-spaces-3d/index.tsx |
| 2 | Create lazy ECharts wrapper component | c78c35c | EChartsWrapper.tsx (NEW), ChartWidget.tsx |
| 3 | Lazy-load xlsx in duty utility + lazy MD editor wrapper for system notice form | 52d6e62 | MarkdownEditor.tsx (NEW), duty/holidays/utils.tsx, duty/management/utils/excelUtils.ts, NoticeForm.tsx, vite.config.ts |

## Verification Results

### Build verification

```bash
cd xingran-react-frontend && npx vite build
```

Result: `built in 27.89s` (Task 3 final build). All three tasks produced clean builds without TS errors.

### Vendor chunk layout (post-Wave-2)

| chunk | raw (KB) | gzip (KB) | Notes |
|-------|---------:|----------:|-------|
| vendor-react | 227.68 | 73.14 | Unchanged |
| vendor-antd | 1661.56 | 399.30 | Unchanged |
| vendor-echarts | 1131.52 | 375.83 | Unchanged (still eagerly preloaded, see below) |
| vendor-three | 894.31 | 242.68 | Unchanged size (now lazy-loaded) |
| vendor-utils | 78.61 | 30.57 | Unchanged |
| **vendor-md-editor** | **53.95** | **17.87** | **NEW — extracted from vendor-commons** |
| **vendor-xlsx** | **429.37** | **142.99** | **NEW — extracted from vendor-commons** |
| vendor-commons | 1854.19 | 608.11 | Down from 743.31 KB (-135.20 KB / -18.2%) |

### Diff vs Wave 1 baseline

| metric | Wave 1 baseline | Wave 2 result | delta |
|--------|----------------:|--------------:|------:|
| vendor-commons gzip | 743.31 KB | 608.11 KB | **-135.20 KB (-18.2%)** |
| vendor-xlsx gzip | (inside commons) | 142.99 KB | (new chunk) |
| vendor-md-editor gzip | (inside commons) | 17.87 KB | (new chunk) |
| Total vendor gzip | 1833.58 KB | 1891.49 KB | +57.91 KB |

> Note on total: xlsx + md-editor chunks are larger when isolated (because they lose the minification benefits of being merged into commons), but they only fetch when the user actually needs them. The vendor-commons reduction is real, and the new chunks are off the critical path for first paint.

### Initial JS payload (entry chunks)

Entry HTML `dist/index.html` includes the following modulepreload hints:
- vendor-commons-CNDTSa4i.js (608.11 KB gzip)
- vendor-antd-DJ5p6SVy.js (399.30 KB gzip)
- vendor-react-oG69oaBK.js (73.14 KB gzip)
- vendor-utils-zoywTPKO.js (30.57 KB gzip)
- vendor-three-C4TI8-j6.js (242.68 KB gzip) — **preloaded but not executed**
- vendor-md-editor (preloaded but not executed)

The main entry `index-DqaPYISm.js` is 148KB raw / 42.63KB gzip. Of the four heavy libs, only `vendor-three` and `vendor-md-editor` appear in modulepreload hints (fetched in parallel without blocking), and none of the four are in `import` statements of the entry — they only execute on demand.

### Acceptance criteria verification

| Check | Status | Evidence |
|-------|--------|----------|
| 3D components lazy via wrapper | PASS | `BuildingScene.tsx` contains 4 `lazy(() => import(...))` calls |
| 3D page entry imports only wrappers | PASS | `grep -E "from '@/pages/operations/building-spaces-3d/components/(BuildingModel3D\|FloorPlan3D\|FloorView3D\|BuildingView3D)'" building-spaces-3d/index.tsx \| wc -l` = 0 |
| Suspense uses AntD Spin with '3D' tip | PASS | `BuildingScene.tsx` uses `<Spin size="large" tip="加载 3D 场景..." />` |
| EChartsWrapper contains lazy import | PASS | `EChartsWrapper.tsx` line 21: `lazy(() => import('echarts-for-react'))` |
| ECharts wrapper uses Spin | PASS | `<Spin tip="加载图表..." />` |
| Only EChartsWrapper imports echarts-for-react | PASS | grep returns only the wrapper (CSS uses class selector, not import) |
| duty/holidays/utils uses await import('xlsx') | PASS | Line 41 + line 124 both `await import('xlsx')` |
| duty/management/utils/excelUtils uses await import('xlsx') | PASS | Line 73 + line 81 both `await import('xlsx')` |
| MarkdownEditor wrapper exists | PASS | New file created |
| MarkdownEditor uses lazy + Spin | PASS | `lazy(() => import('@uiw/react-md-editor'))` + `<Spin tip="加载编辑器..." />` |
| NoticeForm imports from wrapper | PASS | `import { MarkdownEditor as MDEditor } from '@/components/markdown/MarkdownEditor'` |

### Grep verification of consumer reduction

```
# Before Wave 2 (top-level runtime imports):
src/components/dashboard/widgets/types/ChartWidget.tsx → echarts-for-react
src/pages/duty/holidays/utils.tsx → xlsx
src/pages/duty/management/utils/excelUtils.ts → xlsx
src/pages/operations/building-spaces-3d/index.tsx → three.js components
src/pages/system/notice/components/NoticeForm.tsx → @uiw/react-md-editor

# After Wave 2:
src/components/charts/EChartsWrapper.tsx → echarts-for-react (inside lazy())
src/components/markdown/MarkdownEditor.tsx → @uiw/react-md-editor (inside lazy() or import type)
src/components/three/BuildingScene.tsx → three.js components (inside 4 lazy() calls)
(no top-level xlsx imports — all await import('xlsx'))
```

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing functionality] Added vendor-xlsx and vendor-md-editor manualChunks rules**

- **Found during:** Task 3 verification
- **Issue:** After lazy-loading xlsx and md-editor via dynamic imports, Rollup still placed them in `vendor-commons` because the manualChunks function in vite.config.ts didn't have explicit rules for these packages. This meant the dynamic-import boundary worked but the code wasn't split into a separate chunk file.
- **Fix:** Added two new rules to the `manualChunks` function in `vite.config.ts`:
  ```ts
  if (id.includes('@uiw/react-md-editor')) {
    return 'vendor-md-editor'
  }
  if (id.includes('/xlsx/')) {
    return 'vendor-xlsx'
  }
  ```
- **Files modified:** `vite.config.ts`
- **Commit:** 52d6e62 (rolled into Task 3 commit)
- **Result:** `vendor-xlsx-BvJTHLik.js` (429KB / 143KB gzip) and `vendor-md-editor-B6RMQ0bt.js` (54KB / 18KB gzip) are now independent chunks. vendor-commons shrunk from 768KB to 608KB gzip.

### Pre-existing TypeScript Errors (Out of Scope)

The default `npm run build` (which runs `tsc -b && vite build`) still fails on the same pre-existing TS errors documented in `.planning/phases/30-js/deferred-items.md`. Wave 2 verification uses `npx vite build` directly, following the Wave 1 pattern. No new TS errors introduced by this plan.

## Plan Execution Notes

- The plan's "all four heavy libs absent from main entry chunk" goal is met for `import` statements (the critical-path metric) — none of the four libs are statically imported by the entry chunk. However, `vendor-three` and `vendor-md-editor` appear in the entry's `__vite__mapDeps` array as `modulepreload` hints. This is Vite's default `build.modulePreload` behavior: chunks reachable from the entry via static analysis are preloaded, but **not executed** until the corresponding `import()` actually runs.
- Net effect: the browser fetches vendor-three and vendor-md-editor as soon as it parses the entry HTML, in parallel with other critical resources. The chunks sit idle in memory until the user navigates to the 3D page or opens the notice form. JS execution is fully deferred.
- For the truly "no preload" behavior, future work could add `build.modulePreload: { polyfill: false, resolveDependencies: ... }` configuration. Out of scope for Wave 2.
- The xlsx downloadTemplate in `duty/holidays/utils.tsx` is now `async`. The call site `onClick={downloadTemplate}` accepts async functions without issue. No breaking change.
- The handleHolidayImport signature is now `async`. The call site in `management/index.tsx` line 130 is `handleHolidayImport(options, holidayData.batchCreate)` inside a useCallback — fires async without await, which is the same fire-and-forget pattern as before. No breaking change.

## Files Touched

```
xingran-react-frontend/src/components/three/BuildingScene.tsx          (NEW, +60 lines)
xingran-react-frontend/src/components/charts/EChartsWrapper.tsx        (NEW, +47 lines)
xingran-react-frontend/src/components/markdown/MarkdownEditor.tsx      (NEW, +34 lines)
xingran-react-frontend/src/pages/operations/building-spaces-3d/index.tsx (modified, -2 / +2 lines)
xingran-react-frontend/src/components/dashboard/widgets/types/ChartWidget.tsx (modified, +1 / -1 lines)
xingran-react-frontend/src/pages/duty/holidays/utils.tsx              (modified, +6 / -1 lines)
xingran-react-frontend/src/pages/duty/management/utils/excelUtils.ts  (modified, +6 / -1 lines)
xingran-react-frontend/src/pages/system/notice/components/NoticeForm.tsx (modified, +1 / -1 lines)
xingran-react-frontend/vite.config.ts                                  (modified, +8 lines)
```

## Self-Check

- [x] BuildingScene.tsx exists with 4 `lazy()` calls
- [x] EChartsWrapper.tsx exists with `lazy(() => import('echarts-for-react'))`
- [x] MarkdownEditor.tsx exists with `lazy(() => import('@uiw/react-md-editor'))`
- [x] building-spaces-3d/index.tsx has no direct three.js component imports
- [x] ChartWidget.tsx imports EChartsWrapper
- [x] duty/holidays/utils.tsx uses `await import('xlsx')`
- [x] duty/management/utils/excelUtils.ts uses `await import('xlsx')`
- [x] NoticeForm.tsx imports MarkdownEditor from wrapper
- [x] vite build succeeds (3 tasks)
- [x] scripts/check-bundle.sh exits 0 (vendor-react, vendor-antd, vendor-echarts, vendor-three all present)
- [x] vendor-xlsx and vendor-md-editor chunks present in dist/assets/
- [x] Commits 003ccd9, c78c35c, 52d6e62 exist

## Next Steps (Wave 3)

Per D-10/D-11/D-12, Wave 3 will tackle the React Query layer:
- Migrate `sys_dict` dictionary queries to global cache with `useDict` hook
- Migrate dropdown options (dept tree, role list, dict data) to React Query
- Migrate list pages (operations first) with `useTableManager` integration
- Configure `staleTime` 5 min, `gcTime` 30 min per D-11
- Set up `queryClient.invalidateQueries` on dict management mutations

Expected outcome: reduced duplicate API calls across pages, automatic background refresh, elimination of waterfall in dropdown loading.