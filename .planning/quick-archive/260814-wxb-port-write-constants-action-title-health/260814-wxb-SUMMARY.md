---
quick_id: 260814-wxb
slug: port-write-constants-action-title-health
subsystem: frontend-tests
tags: [vitest, test-expectations, port-write, reconciliation]
key-files:
  modified:
    - xingran-react-frontend/src/components/network/port-write/__tests__/constants.test.ts
    - xingran-react-frontend/src/components/reconciliation/__tests__/HealthCard.test.tsx
commits:
  - "19ac4f6: test(260814-wxb): action_title 断言对齐 v1.20.1 的 7 个 action key"
  - "bbc5248: test(260814-wxb): healthcard loaded 断言对齐单行紧凑版 DOM"
completed: 2026-08-14
---

# Quick Task 260814-wxb Summary: 修复两个存量前端测试失败（仅改测试预期，零产品代码改动）

## What Was Done

两个测试文件均为**有意的实现变更后未同步的测试预期**，按计划只改测试、不动产品代码：

1. **constants.test.ts (Task 1)** — `ACTION_TITLE` 在 v1.20.1 (Phase 56) 新增 `set_access_vlan` 与 `port_binding`（constants.ts 注释明确 "5 + 2 = 7"）。期望数组从 5 key 更新为 7 key，头部注释与 describe/测试名同步，并落地计划中的可选项：两个 v1.20.1 标题值冒烟断言（`set_access_vlan` 含 "VLAN"、`port_binding` 含 "绑定"）。
2. **HealthCard.test.tsx (Task 2)** — 组件在 gsd-fast 紧凑重构（2026-06-30）后为单行布局（无 Card 标题、无 Statistic、无 ECharts）。loaded 状态测试重写：保留 `getByText("75")`（score span 唯一直接文本子节点）、新增嵌套 `/100` 断言、5 个 KPI 标题断言替换为单 span 拼接文本断言（`正常 5 · 漂移 2 · 冲突 1 · 无数据 1 · 例外 1`）、删除 echarts-mock 断言、更新头部注释。其余 4 个测试未动。

## Verification

- `npx vitest run src/components/network/port-write/__tests__/constants.test.ts` — **12/12 通过**（11 存量 + 1 新增冒烟）
- `npx vitest run src/components/reconciliation/__tests__/HealthCard.test.tsx` — **5/5 通过**
- 全量回归 `npx vitest run` — **13 files / 80 tests 全部通过**（基线：任务前恰好只有这 2 个失败 → 现 0 失败；jsdom getComputedStyle 告警为存量噪音，非本次引入）
- 两次 commit 均通过 husky pre-commit（eslint --fix + type-check + prettier）
- 零产品代码改动确认：两次 commit 合计仅触及 2 个 `__tests__` 下文件

## Deviations from Plan

1. **[可选落地] constants 测试 11 → 12**：计划 verify 标注 "全绿（11/11）"，同时提供可选项"对两个 v1.20.1 标题值做冒烟断言"。落地该可选项后测试数为 12/12，全绿标准满足。
2. **[提交信息修正] commitlint subject-case**：首个 Task 1 commit 因 subject 以大写 `ACTION_TITLE` 开头被 commitlint `subject-case` 规则拒绝，改为小写 `action_title` 开头后通过。无代码影响。

## Self-Check: PASSED

- 文件存在：constants.test.ts / HealthCard.test.tsx 均在 commit 中修改（`git show --stat` 确认）
- Commit 存在：19ac4f6 / bbc5248 均在 `git log --oneline` 中
