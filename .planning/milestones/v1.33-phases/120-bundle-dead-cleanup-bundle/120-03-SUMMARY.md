---
phase: 120
plan: 120-03
title: DEAD-01 + DEAD-03 — 5 处死代码删除 + vite.config 注释修正
status: complete
wave: 3
requirements_completed: [DEAD-01, DEAD-03]
commits: [a9fc516, 250acec]
files_modified: 8
---

## What Shipped

### DEAD-01 — 5 处死代码删除
- **`hooks/useTabSync.ts` + 测试**（含 useUtilityHooks.test.tsx 中的 import 清理）— 全库无引用验证通过
- **`components/dashboard/DashboardView.tsx` + CSS + test** — 与 `pages/dashboard-system/components/DashboardView.tsx`（活消费者）不同，DEAD-01 删的是死组件
- **`building-spaces-3d/utils.ts` `getWorkstationStats`** — 已重命名为 `calculateWorkstationStats`，原名无引用
- **`executions/index.tsx` `_detailColumns`** — 删除孤立列定义
- **`components/operations/index.ts` barrel** — 删除（DeptSidebar/StatisticsCards 已改用直接路径导入）

### DEAD-03 — vite.config 注释修正
- `vite.config.ts:23` react-markdown 过时注释修正（它确实是 React 组件）
- `vite.config.ts:169-171` react-markdown 注释完整化

## Recovery

- **意外事故**：commit a9fc516 中 `git rm` 误删整个 `executions/index.tsx` (349 行)，本意只删 `_detailColumns`
- **检测**：executor 死后人工 git state check 发现 `?? src/pages/network/executions/index.tsx` untracked
- **恢复**：commit `250acec` 验证磁盘版本（345 行已删 `_detailColumns`）正确后重新 add + commit
- **教训**：DEAD-01 中执行 `git rm` 前必须 grep 验证 only-`_detailColumns` 被删；后续 milestone 类似操作应改用 Edit 而不是文件级删除

## Verification

- 全库无引用验证：
  - `grep "useTabSync"` → 0 (1 stale comment in DynamicRoutes.tsx 保留，注释文字提及)
  - `grep "components/dashboard/DashboardView"` → 0 (pages/dashboard-system/components/DashboardView 活消费者保留)
  - `grep "getWorkstationStats"` → 0 (`calculateWorkstationStats` 活消费者保留)
  - `grep "_detailColumns"` in executions/index.tsx → 0
  - `ls components/operations/index.ts` → No such file
- npm run type-check / lint 绿
- 3803/3807 全量测试通过（4 个 Phase 116 selector mock 回归后续修复 commit cb65750/b707b72）

## Deviations

1. **`getWorkstationStats` 实际已重命名为 `calculateWorkstationStats`** — auditor 标记的 dead code 名与现状不符；保留活函数 `calculateWorkstationStats`（FloorView3D.tsx 是消费者），符合用户全局偏好 "废弃函数直接删除不留兼容壳"——但此函数非废弃，故保留

2. **executions/index.tsx 误删事故**：如上 Recovery 节，需