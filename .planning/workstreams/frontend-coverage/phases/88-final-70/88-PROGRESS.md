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

**maxWorkers=4** 修复 159 测试文件的 Windows 资源竞争 flake。

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
| 88-R5 components+widgets | 34.02% | — |
| 88-R6 dashboard+reconciliation | 35.27% | — |
| 88-R7-9 ops子/vdi/wo/ad-dom | 36.19% | — |
| **88-R10-12 子页零散** | **36.86%** | **1264** |

## 目录现状(TOP)

| 目录 | stmts | 覆盖 |
|------|-------|------|
| utils/store/lib/hooks/constants | ~3700 | 86-95% ✅ |
| monitor | 627 | 38.92% |
| login/settings | 160 | 62-83% |
| knowledge | 262 | 46.95% |
| network | 1962 | 33%+ |
| duty | 1190 | 35.55% |
| design-system | 194 | 53.09% |
| system | 2203 | 29.82% |
| operations | 3611 | 17.53% ⚠ 最大洼地 |
| components | 3958 | 25.95% ⚠ |
| ad-domain | 1082 | 20.52% |
| vdi/workorder | 1118 | ~22% |

## 剩余到 70%(差 ~7100 stmts)

未触达关键子页:
1. **operations**: workstations 主页(jsdom 死锁,需专项修复)、rpa executions/workers 详情、server-rooms/dedicated-lines/info-points 子组件
2. **components 深测**: design-system 组件(已 53%,待 push 至 70%+)、reconciliation 网络 Modal、reconciliation 健康详情
3. **system/network**: Modal/Drawer 交互路径、knowledge 子组件
4. **零散组件**: NoticeDetail/Modal/Form/RecurrenceConfig、Mac Detail、Profile page

## 已知阻塞
- workstations/index.tsx: jsdom 渲染死锁(useEffect 链或轮询),需专项修复
- asset/reconciliation: 首屏空分支只 +1 stmt(子组件化)
- Layout/HybridLayout: menuStore 深依赖 + 菜单轮询导致死锁
- design-system 组件: 已 53%,需 ConfigProvider 链路测试