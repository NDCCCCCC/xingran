---
phase: 88-final-70
status: in-progress
updated: 2026-08-28
---

# Phase 88 收口进度(阶段性)

## 突破: 页面渲染测试模式

**renderPage helper**(`src/test/utils/renderPage.tsx`)落地:
- 真实 hooks 执行(useTableManager/usePagination/React Query 全链计入覆盖率)
- 只 mock @/lib/api 端点(fixture 数据)
- QueryClient 自动注入 + 未注册端点安全 fallback

单页收益 ~70-300 stmts,单文件可批量测 4-12 页面。

## 里程碑推进(本会话)

| 阶段 | GLOBAL | tests |
|------|--------|-------|
| Phase 84 完成 | 20.06% | 1005 |
| Phase 85 (operations) | 20.75% | 1067 |
| Phase 86 (system+network) | 21.60% | 1128 |
| Phase 87 (剩余页面) | 22.11% | 1162 |
| 88-R1 renderPage 原型 | 23.83% | — |
| 88-R2 批量页面渲染 | 29.03% | — |
| 88-R3 monitor+ad-domain+rpa | 30.77% | 1203 |
| 88-R4 子页面深挖 | 33.42% | — |
| **88-R5 components+widgets** | **34.02%** | **1219** |

## 目录现状(TOP)

| 目录 | stmts | 覆盖 |
|------|-------|------|
| utils/store/lib/hooks/constants | ~3700 | 86-95% ✅ |
| monitor | 627 | 38.92% |
| login/settings | 160 | 62-83% |
| network | 1962 | 33.03% |
| system | 2203 | 29.82% |
| duty | 1190 | 29.41% |
| knowledge | 262 | 30.15% |
| design-system | 194 | 53.09% |
| operations | 3611 | 12.66% ⚠ 最大洼地 |
| components | 3958 | 19.18% ⚠ |
| ad-domain | 1082 | 18.39% |
| vdi/workorder | 1118 | ~17% |

## 剩余到 70%(差 ~7700 stmts)

优先级(按 stmts×提升空间):
1. **operations 3611**(12.66%→65% 需 +1900): workstations 主页 jsdom 死锁需专项修复;子页面(modals/views)可加
2. **components 3958**(19.18%→60% 需 +1600): dashboard settings/layout 子组件渲染 + reconciliation/network 组件深测
3. **ad-domain/vdi/workorder/duty**(3951,~18%→55% 需 +1500): 剩余子页 + 详情页
4. **system/network 深测**(4165,~31%→55% 需 +1000): Modal/Drawer 交互路径

## 已知阻塞

- workstations/index.tsx: jsdom 渲染死锁(useEffect 链或轮询),单独进程跑挂起——需 vitest单独 pool 或页面重构才可测
- dashboard widgets: lazy Suspense 早退,display config 需完整 fixture 才走主路径
- asset/reconciliation: 首屏空分支只 +1 stmt
- Layout 组件: menuStore 深依赖,vi.mock store 后可测(未做)
