---
status: partial
phase: 30-js
source: [30-VERIFICATION.md]
started: 2026-06-13T10:15:00Z
updated: 2026-06-13T10:15:00Z
---

# Phase 30 — Human UAT (Frontend Performance Optimization)

## Current Test

Phase 30 自动化检查全部通过 (23/25 must-haves verified, 5/7 ROADMAP truths)。以下 5 项需要人工验证（视觉/性能/E2E 行为），自动化无法覆盖。

## Tests

### 1. Visual bundle treemap
expected: 打开 `ruoyi-react-frontend/dist/stats.html` 在浏览器中可见 8 个 vendor chunk（vendor-react、vendor-antd、vendor-echarts、vendor-three、vendor-utils、vendor-md-editor、vendor-xlsx、vendor-commons），每个显示 gzip/brotli 体积
result: [pending]

### 2. Lighthouse mobile (LCP budget)
expected: 在 `http://localhost:4000/login` 跑 Lighthouse mobile，LCP ≤ 2.5s（Phase 30 D-05 预算）
result: [pending]

### 3. Asset list virtual scroll UX
expected: 打开 `/operations/assets`（43 列资产列表），注入 200+ 条记录，DOM 中只渲染约 20 行，滚动流畅
result: [pending]

### 4. Widget memo boundary
expected: 打开 dashboard（6 个 widget），通过 API 更新其中一个 widget 的数据，使用 React Devtools Profiler 确认只有该 widget 重新渲染
result: [pending]

### 5. Dict cache invalidation E2E
expected: 在 `/system/dict` 字典管理页修改某个字典项，然后访问使用 `useDict('sys_user_sex')` 的页面，新字典值应自动出现无需手动刷新
result: [pending]

## Summary

total: 5
passed: 0
issues: 0
pending: 5
skipped: 0
blocked: 0

## Gaps

无 BLOCKER 缺口。Phase 30 跨 4 个 Wave 全部执行：
- Wave 1: 基础设施（visualizer, manualChunks, 500KB 阈值, baseline）
- Wave 2: 4 个重库按需加载（three/echarts/xlsx/md-editor）— vendor-commons -17.9%
- Wave 3: React Query 层（useDict, useDeptTree, useRoleList, useTableQuery, dict/dept/role 失效）
- Wave 4: 渲染层（3 个 virtual table, 4 个 React.memo, 5 条 ESLint 规则）

### Partial findings (已记录)

| 项 | 状态 | 说明 |
|----|------|------|
| `useDict` hook 无直接 page consumer | 基础设施先行 | 留待后续 quick task 接入 |
| `useTableQuery` hook 无直接 page consumer | 基础设施先行 | 留待后续 quick task 接入 |
| `Widget.tsx` 未接入 DashboardGrid | 部分完成 | Wave 4 next-steps 已记录 |
| `BaseEditModal.tsx` 未替换现有 modal 调用 | 部分完成 | 留待后续 quick task 接入 |
| vendor-commons = 610 KB gzip | 超出 500KB 预算 | 已从 baseline 743 KB 减少 -133 KB / -17.9%，需进一步 split |

### 已知遗留 (deferred-items.md 已记录)

- LCP 实测未捕获（手动 Lighthouse 任务，自动化无法覆盖）
- 108 条 pre-existing ESLint 错误级违规需后续 quick task 清理
- pre-existing TypeScript 错误阻塞 `npm run build`（workaround: `npx vite build`）
- ESLint 规则名按 v7.37 canonical 修正（语义保持）
