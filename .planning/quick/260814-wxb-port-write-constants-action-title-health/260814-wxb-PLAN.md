---
quick_id: 260814-wxb
slug: port-write-constants-action-title-health
description: 修复两个存量前端测试失败（port-write constants ACTION_TITLE 断言 + HealthCard Statistic 断言）
created: 2026-08-14
status: planned
---

# Quick Plan 260814-wxb: 修复两个存量前端测试失败

**原则：仅改测试预期，零产品代码改动**（实现本身是 v1.20.1 / gsd-fast 紧凑重构的有意变更，不是 bug）。

## 诊断（已通过实际测试运行确认）

1. **constants.test.ts** — `ACTION_TITLE` 在 v1.20.1 (Phase 56) 中新增 `set_access_vlan` 与 `port_binding`，
   constants.ts 注释明确记录 "5 + 2 = 7"。测试仍断言 5 个 key。
   失败输出：`expected [ Array(7) ] to deeply equal [ Array(5) ]`。
2. **HealthCard.test.tsx** — 根因不是 antd Statistic/jsdom 问题。组件在 gsd-fast 紧凑重构
   （2026-06-30）中改为单行布局：无 Card 标题、无 Statistic、无 ECharts。
   首个失败点是 `getByText("对账健康度")`（102 行），而非 "75"。
   DOM 验证：score span 唯一直接文本子节点是 "75"（嵌套 "/100" span 是元素子节点，不参与
   getNodeText），所以 `getByText("75")` 本身可用。

## Task 1: 更新 ACTION_TITLE key 断言

- **files**: `xingran-react-frontend/src/components/network/port-write/__tests__/constants.test.ts`
- **action**: 头部注释 + describe/测试名 + 期望数组改为 7 个 key（新增 `port_binding`、`set_access_vlan`）；
  可选：对两个 v1.20.1 标题值做冒烟断言。
- **verify**: `npx vitest run src/components/network/port-write/__tests__/constants.test.ts` 全绿（11/11）。
- **done**: 测试文件预期与 constants.ts 实际 7 key 一致，无产品代码改动。

## Task 2: 重写 HealthCard loaded 状态测试

- **files**: `xingran-react-frontend/src/components/reconciliation/__tests__/HealthCard.test.tsx`
- **action**: 重写 "renders 5 KPIs + score + trend when data is loaded" 测试以匹配紧凑实现：
  - 保留 `getByText("75")`，新增 `/100` 断言；
  - 5 个 KPI 标题断言替换为对拼接单 span 文本 `正常 5 · 漂移 2 · 冲突 1 · 无数据 1 · 例外 1` 的单条断言；
  - 删除 `echarts-mock` 相关断言；更新头部注释。其余 4 个测试不动（当前已通过）。
- **verify**: `npx vitest run src/components/reconciliation/__tests__/HealthCard.test.tsx` 全绿（5/5）。
- **done**: loaded 状态测试与紧凑版组件 DOM 一致，无产品代码改动。

## 收尾验证

运行整个前端测试套件确认无附带回归（基线：本任务前恰好只有这 2 个失败）。
