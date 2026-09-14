---
phase: 115
slug: selector-completion-zustand-selector
status: approved
nyquist_compliant: true
wave_0_complete: true
created: 2026-09-14
---

# Phase 115 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Vitest 4（^4.0.18）+ @testing-library/react，jsdom |
| **Config file** | `xingran-react-frontend/vitest.config.ts` |
| **Quick run command** | `cd xingran-react-frontend && npx vitest run <受影响测试文件...>` |
| **Full suite command** | `cd xingran-react-frontend && npx vitest run`（+ lint / type-check / test:coverage） |
| **Estimated runtime** | ~11s（受影响 5 文件热缓存实测）/ 全量 ~3min |

---

## Sampling Rate

- **After every task commit:** 该 task 触及的测试 quick run + `npm run type-check`
- **After every plan wave:** 受影响 4 大簇（store / router / 3D / 页面）+ `npm run lint`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** ~30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| （随 plan 划分） | 01 | 1 | SELECTOR-01 | — | N/A | 既有回归 | `npx vitest run src/store/tabsStore.test.ts src/components/layout/shared/__tests__/TabBar.render.test.tsx src/components/layout/shared/__tests__/useRouteTabs.test.tsx src/hooks/useUtilityHooks.test.tsx` | ✅ | ⬜ pending |
| （随 plan 划分） | 01 | 1 | SELECTOR-02 | — | N/A | 既有回归（派生值 renderHook 断言） | `npx vitest run src/store/layoutStore.test.ts` | ✅ | ⬜ pending |
| （随 plan 划分） | 02 | 1 | SELECTOR-03 | V4 守护项 | RouteGuard 权限判断逐字保留（UX-only 注释语义不变） | 既有回归 + type-check | `npx vitest run src/router/__tests__/routeGuard-lastpath.test.tsx && npm run type-check` | ✅ | ⬜ pending |
| （随 plan 划分） | 02 | 1 | SELECTOR-04 | — | N/A | 既有回归（真实 store） | `npx vitest run src/pages/operations/building-spaces-3d/ src/store/visualizationStore.test.ts` | ✅ | ⬜ pending |
| （随 plan 划分） | 03 | 2 | SELECTOR-05 | — | DashboardScopeSelector mock 保持 isAdmin/dataScope 断言语义（dual-form mock） | 既有回归 + 4 个 mock 必改 | `npx vitest run src/pages/login/ src/pages/my-notices/__tests__/detail.test.tsx src/components/dashboard/settings/__tests__/DashboardScopeSelector.test.tsx src/pages/profile/ src/pages/duty/my-duty/` | ✅（mock 更新后） | ⬜ pending |
| （随 plan 划分） | 03 | 2 | Criterion 5 | — | N/A | grep 断言（机械） | `grep -rnE "use[A-Z][A-Za-z0-9]*Store\(\)" xingran-react-frontend/src --include="*.ts" --include="*.tsx" \| grep -vE "__tests__\|\.test\."` → **恰好 12 处**（dashboardStore 家族 11 + useTabSync 1，Phase 116/120 清单文件），且全部命中三组白名单前缀：`hooks/(useTabSync\|useWidgetData).ts` / `pages/dashboard-system/` / `components/dashboard/`——清单外命中即回修对应簇，禁改断言口径（03-T3 定案，OQ-1 全库口径） | — | ⬜ pending |
| — | — | — | 七 gate 相关项 | — | N/A | lint / type-check / coverage floor / size-limit | `npm run lint && npm run type-check && npm run test:coverage && npm run size` | ✅ | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

**假失败警告（RESEARCH Pitfall 3）:** 冷缓存批量首跑可能出现 1 例超时红（login "导出为函数组件"）——隔离跑/热缓存重跑绿即非回归，判红前必须重跑。

---

## Wave 0 Requirements

None — 既有测试基础设施完整覆盖全部 phase 需求（唯一新工作是 4 个既有 mock 文件就地修改，随组件同任务落位）。

---

## Manual-Only Verifications

All phase behaviors have automated verification.（D-04 零回归口径 = 测试 + lint/type-check + grep 断言；无用户可见行为变更，无人工验证项。）

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved — 2026-09-14（checker 维度 8 复核 8a-8d 全部通过；Criterion 5 行已按 03-T3 定案口径修订为恰好 12 处 + 三组白名单前缀；plan 02 行 Wave 修正为 1）
