---
phase: 67-build-regression-and-visual-confirmation
plan: 01
task: T1 (QA-03 four-gate + bundle three-dimension comparison)
date: 2026-08-18
status: COMPLETE
requirements: [QA-03]
---

# Phase 67 QA-03 Bundle Comparison

## Scope

QA-03 四门终验 + bundle 三口径前后对比（dist 总量 / vendor-react gzip / app chunk 字节）。引用 Phase 64-66 基线数据，验证 v1.22 全程未引入 bundle 回归。

## QA-03 四门结果（2026-08-18 实测）

| 门 | 命令 | 退出码 | 关键输出 |
|---|------|--------|----------|
| `npm run type-check` | `tsc --noEmit` | **0** | 空检查（solution-style），按 plan 记 |
| `npm run build` | `tsc -b && vite build` | **0** | `✓ built in 49.78s` |
| `npm run lint` | `eslint . && check-hardcoded-colors.mjs` | **0** | 0 errors / 1,032 warnings（与基线 1,032 持平，零新增）；scanner `[ok] no hardcoded colors found in 627 scanned files` |
| `npx vitest run` | 全量 | **0** | **14 files / 120 tests passed**（含 colors.test.ts 40/40） |

**结论**：四门全绿，无回归。

## Bundle 三口径对比（前后）

### 口径 A — Vendor gzip（Phase 66 → Phase 67）

| Vendor chunk | Phase 64 后 | Phase 65 后 | Phase 66 后 | **Phase 67 后** | 趋势 |
|---|---|---|---|---|---|
| vendor-react | 774.94 kB | 774.94 kB | 774.94 kB | **774.94 kB** | 持平 ✓ |
| vendor-echarts | 374.55 kB | 374.55 kB | 374.55 kB | **374.55 kB** | 持平 ✓ |
| vendor-three | 242.65 kB | 242.65 kB | 242.65 kB | **242.65 kB** | 持平 ✓ |
| vendor-xlsx | 142.99 kB | 142.99 kB | 142.99 kB | **142.99 kB** | 持平 ✓ |
| vendor-markdown | 116.13 kB | 116.13 kB | 116.13 kB | **116.13 kB** | 持平 ✓ |

vendor-md-editor（`@uiw/react-md-editor` 拆分 chunk）17.28 kB gzip — Phase 67 引入（注：Phase 64-66 SUMMARY 未单独记录此项；按 Phase 67 build 实测存在）。

### 口径 B — dist/assets 总量

| 度量 | Phase 65 后 | **Phase 67 后** | 趋势 |
|---|---|---|---|
| `du -sk dist/` | — | **7,289 kB** | — |
| `du -sk dist/assets/` | 7,281 kB | **7,281 kB** | **持平 ✓**（精确到 kB） |
| Raw JS bytes (sum) | — | **6,934,915 bytes** (~6.6 MB raw, ungzipped) | — |

### 口径 C — Chunk 数（app chunks 与 vendor chunks）

| 度量 | Phase 65 后 | **Phase 67 后** | 趋势 |
|---|---|---|---|
| 总 JS chunks (`dist/assets/*.js`) | 134 | **134** | 持平 ✓ |
| App chunks (excluding `vendor-*`) | — | **128** | — |
| Vendor chunks | — | 6 (`vendor-react` / `vendor-echarts` / `vendor-three` / `vendor-xlsx` / `vendor-markdown` / `vendor-md-editor`) | — |

### Top 5 最大单 chunk（按 raw bytes）

| # | File | Size (raw bytes) | Gzip |
|---|---|---|---|
| 1 | `dist/assets/vendor-react-B8R72xYa.js` | 2,830,143 | 774.94 kB |
| 2 | `dist/assets/vendor-echarts-D4lsRrLc.js` | 1,127,803 | 374.55 kB |
| 3 | `dist/assets/vendor-three-D7YmM2rC.js` | 894,264 | 242.65 kB |
| 4 | `dist/assets/vendor-xlsx-BvJTHLik.js` | 429,371 | 142.99 kB |
| 5 | `dist/assets/vendor-markdown-CTNRp5o7.js` | 372,250 | 116.13 kB |

5 个 vendor chunk 占据约 **5,653,831 bytes raw（~5.4 MB）**，约占 JS 总量的 81.5%（5,653,831 / 6,934,915）。

## 对比结论（QA-03 SC#1 解读）

> **ROADMAP SC#1 verbatim**：vendor-react 打包后 gzip 体积较 v1.21 baseline（774.96 kB）不增（预期下降 —— 移除 6 套主题节约 ~数 kB），前后对比数值记录到 ROADMAP Progress 段。

### 1. vendor-react gzip 持平 = PASS

Phase 67 实测 vendor-react gzip = **774.94 kB**，与 Phase 66 后、Phase 65 后完全一致；与 v1.21 baseline（774.96 kB）相比 **-0.02 kB**（实际微降 20 字节）。

**持平即过**（按 plan Constraints #5 + Phase 65 Deviation #4 锁定决策）：
- 主题系统代码（themes/、ColorSwitcher）从未进入 vendor-react chunk（该 chunk 仅含 react/antd 等框架依赖）
- 主题代码的收益体现在**源码层**（Phase 65 净减 **-4,357 行**）与**应用路由层 chunk**
- vendor-react hash 多次变化（CofIW6P5 → B8R72xYa），但字节体积不变

### 2. dist/assets 总量持平 = PASS

`dist/assets/` = **7,281 kB**（与 Phase 65 后 kB 级精确一致）。`du -sk dist/` = 7,289 kB（多 8 kB 来自 `dist/index.html` 等根级文件）。

### 3. Chunk 数持平 = PASS

JS chunks 总计 **134 个**（与 Phase 65 后完全一致）；其中 **app chunks 128 个** + **vendor chunks 6 个**。

## lint Warning 对比（无新增证明）

- Phase 67 终值：**1,032 warnings / 0 errors**
- Phase 66 终值：1,032 warnings / 0 errors（基线持平）
- Phase 65 中途测量（T7 短暂 +6 已修复归零，T8 终值 1,032）
- **Phase 67 净增 0 条 warning**

## Self-Check

- [x] 四门全绿（type-check 0 / build 0 / lint 0 / vitest 14-120）
- [x] vendor-react gzip = 774.94 kB（与 Phase 66 持平，无回归）
- [x] dist/assets = 7,281 kB（与 Phase 65 持平，无回归）
- [x] 总 chunks = 134（与 Phase 65 持平，无回归）
- [x] scanner 0 命中（check-hardcoded-colors.mjs）
- [x] lint warning 持平 1,032（零新增）
- [x] 未触碰业务代码（D-05 范围纪律）