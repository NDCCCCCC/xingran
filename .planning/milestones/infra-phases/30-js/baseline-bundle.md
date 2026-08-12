# Phase 30 Wave 1 — Bundle Baseline

Captured: 2026-06-13
Build command: `cd xingran-react-frontend && npm run analyze`
Node: see package.json (vite ^7.2.4)
Rollup: bundled with Vite 7.3.0

## Vendor Chunks (D-07 layout)

| chunk | raw (KB) | gzip (KB) |
|-------|---------:|----------:|
| vendor-react | 222.34 | 71.07 |
| vendor-antd | 1622.62 | 389.35 |
| vendor-echarts | 1104.93 | 365.25 |
| vendor-three | 873.35 | 235.01 |
| vendor-utils | 76.77 | 29.59 |
| vendor-commons | 2279.85 | 743.31 |
| **vendor subtotal** | **6179.86** | **1833.58** |

## Entry / First-Frame Chunks

The largest route-level chunks (top 5 by gzip):

| chunk | raw (KB) | gzip (KB) |
|-------|---------:|----------:|
| index-QkYhMALQ.js | 144.86 | 41.53 |
| index-B-vdY91G.js | 60.96 | 20.55 |
| index-CVpAy61w.js | 59.11 | 15.24 |
| index-Do1su_6W.js | 22.27 | 7.03 |
| index-Dp9c8B6n.js | 27.03 | 8.76 |

## Total Bundle Metrics

- Total vendor (gzip): 1833.58 KB
- Largest single chunk (gzip): vendor-commons = 743.31 KB
- Second largest (gzip): vendor-antd = 389.35 KB
- Third largest (gzip): vendor-echarts = 365.25 KB
- Total route-level chunks (gzip sum): TBD (70+ route files)

## Observations

1. **vendor-commons is the dominant chunk (743 KB gzip)** — likely contains
   `@uiw/react-baidu-map`, `xlsx`, `@uiw/react-md-editor`, `react-markdown`,
   `react-grid-layout`, `jsonata`, `@breejs/later`, `cron-parser`,
   `cron-validate`, `react-baidu-map`, `@dnd-kit/*`, etc. Wave 2 (重库按需加载)
   should split these.

2. **vendor-antd (389 KB gzip) is within 500KB budget but close to limit**.
   Wave 2 might trim antd locale (zh_CN) into a separate chunk.

3. **vendor-echarts (365 KB gzip) is loaded eagerly**. Wave 2 should defer
   echarts to dynamic import.

4. **vendor-three (235 KB gzip) is eagerly loaded**. Wave 2 should defer
   three to dynamic import (3D pages only).

5. **vendor-react (71 KB gzip)** is reasonable. No action needed in Wave 1.

6. **vendor-utils (30 KB gzip)** contains sm-crypto + dayjs + axios. Per D-06,
   sm-crypto must stay sync (used by all encrypted endpoints). No action.

## First-Frame Budget Status

- Initial JS gzip budget per D-05: **500 KB**
- Largest single chunk (vendor-commons 743 KB gzip) **exceeds** 500 KB budget —
  this chunk is on the warning list.
- Wave 2 will reduce the initial load by lazy-loading echarts/three/xlsx and
  moving locale/icons to dynamic imports.

## Reproduction

```bash
cd xingran-react-frontend
npm run analyze
# Open dist/stats.html in browser for treemap visualization
```

## Verification

```bash
bash scripts/check-bundle.sh
# Expect: OK: All required vendor chunks present
```