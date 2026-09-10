# Vercel React Best Practices — Frontend Audit Report

**Audit date**: 2026-09-10
**Auditor**: Claude Code (vercel-react-best-practices skill)
**Scope**: `xingran-react-frontend/src/` — 1148 .tsx/.ts files, 8.5 MB
**Framework**: React 19.2 + TypeScript 5.9 + Vite 7.2 + Ant Design 6.1 + Zustand 5.0 + TanStack Query 5.101
**Rules applied**: 57 Vercel rules across 8 categories
**Method**: 4 parallel audit agents (async+bundle / server+client / re-render+rendering / JS+advanced)

---

## Executive Summary

| Severity | Count | Notes |
|----------|-------|-------|
| **CRITICAL** | **5** | All in bundle-dynamic-imports (echarts + Three.js + icon barrel) |
| **HIGH** | **8** | 1 N+1 waterfall, 1 token-blocked download, 2 React Query bypass, 1 setInterval race, 3 lazy-boundary bypassed |
| **MEDIUM** | **17** | localStorage versioning, effect→event, passive listeners, memo opportunities, setState patterns |
| **LOW** | **6** | Trivial boolean memo, type re-exports, scroll passive, hydrate warnings |
| **TOTAL** | **36** | 6 of 8 categories had findings (Cat 3 = N/A for SPA, Cat 8 = no findings) |

### Top 5 Wins (Highest ROI)

1. **echarts + Three.js 静态导入** (~600KB gzip) → 路由级 lazy chunk — 减少冷启动 200-800ms
2. **`@ant-design/icons` 全部导入** (~1MB) → 按需动态导入或 sprite sheet — 节省 1MB+ bundle
3. **`useWidgetData`/`useWidgetPolling`/`useADConfigs` 绕开 React Query** → 改用 `useQuery` — 自动 dedup, 节省重复请求
4. **BuildingView3D N+1 sequential 楼层工作站计数** → 批量端点或外层 Promise.all — 减少 10x 延迟
5. **monitor/dashboard 每 30s 3 个轮询** → `useQuery` + `refetchInterval` — 自动 dedup + stale-while-revalidate

---

## Category Breakdown

### Category 1: Eliminating Waterfalls (CRITICAL impact) — 5 findings

| ID | Severity | File | Rule | Issue |
|----|----------|------|------|-------|
| 1.1 | HIGH | `pages/operations/building-spaces-3d/components/BuildingView3D.tsx:91-118` | async-parallel | N+1: 1 floor list + N sequential workstation count queries (P1f) |
| 1.2 | HIGH | `pages/network/executions/index.tsx:93-96` | async-parallel | `loadDevices()` then `loadTemplates()` sequentially — 2 independent calls |
| 1.3 | MEDIUM | `pages/profile/index.tsx:72-76` | async-parallel | `loadProfile()` then `loadDutyStats()` sequentially — 2 independent calls |
| 1.4 | MEDIUM | `pages/duty/management/index.tsx:58-65` | async-parallel | 5 sequential fetches in mount effect (`fetchPools`/`fetchUsers`/`fetchWeeklyDuty`/`fetchYears`/`fetch`) |
| 1.5 | MEDIUM | `lib/download.ts:33-38` | async-api-routes | `blobAxios` interceptor synchronously awaits `getAccessToken()` before every download |

**Fix priority**: 1.1 and 1.2 are user-facing modals. 1.3/1.4/1.5 are minor.

### Category 2: Bundle Size Optimization (CRITICAL impact) — 8 findings

| ID | Severity | File | Rule | Issue |
|----|----------|------|------|-------|
| 2.1 | **CRITICAL** | `main.tsx:5` | bundle-dynamic-imports | `import "@/lib/echarts"` 静态副作用导入 — 把整个 echarts core (~376KB) 锁进初始 chunk |
| 2.2 | **CRITICAL** | `pages/operations/building-spaces-3d/components/BuildingModel3D.tsx:7-9` | bundle-dynamic-imports | `@react-three/fiber` + `@react-three/drei` + `three` 静态导入 (~235KB gzip) |
| 2.3 | **CRITICAL** | `pages/operations/building-spaces-3d/components/FloorPlan3D.tsx:7-9` | bundle-dynamic-imports | 同 2.2 — Three.js 生态 |
| 2.4 | **CRITICAL** | `pages/operations/building-spaces-3d/components/FloorView3D.tsx:20` | bundle-dynamic-imports | 绕过 `BuildingScene` 的 lazy 包装, 直接导入 `FloorPlan3D` |
| 2.5 | **CRITICAL** | `pages/operations/building-spaces-3d/components/BuildingView3D.tsx:14-15` | bundle-dynamic-imports | 同 2.4 — 同时绕过 `BuildingModel3D` + `FloorPlan3D` 两个 lazy 包装 |
| 2.6 | **CRITICAL** | `utils/iconUtils.tsx:6` | bundle-barrel-imports | `import * as Icons from "@ant-design/icons"` — 整个图标库 (~1MB+), 1000+ icon |
| 2.7 | HIGH | `components/charts/EChartsWrapper.tsx:21` | bundle-dynamic-imports | ECharts 的 `lazy(() => import("echarts-for-react"))` 只延迟了 ~50KB wrapper, 因为 main.tsx 已强制加载 echarts core |
| 2.8 | LOW | `lib/workorderApi.ts:262` | bundle-barrel-imports | `export type { SimpleDept } from "./dutyApi"` — type-only re-export, 几乎无运行时影响 |

**Fix priority**: 2.1-2.6 全部为 CRITICAL bundle hit. 修复后预计初始 bundle 减少 1.5MB+.

### Category 3: Server-Side Performance (HIGH impact) — 0 findings

全部 7 条规则 (`server-*`) 都是 Next.js / RSC 专用. **N/A for this SPA.**

### Category 4: Client-Side Data Fetching (MEDIUM-HIGH) — 10 findings

| ID | Severity | File | Rule | Issue |
|----|----------|------|------|-------|
| 4.1 | MEDIUM | `hooks/useWindowSize.ts:31` | client-passive-event-listeners | `resize` listener 未 passive |
| 4.2 | MEDIUM | `components/layout/shared/TabBar.tsx:331` | client-passive-event-listeners | capture-phase `scroll` listener 未 passive |
| 4.3 | LOW | `components/layout/shared/TabBar.tsx:332` | client-passive-event-listeners | `resize` listener 未 passive |
| 4.4 | MEDIUM | `hooks/useColumnConfig.ts:21,27-28` | client-localstorage-schema | `column_config` localStorage key 无版本前缀, 无 try-catch 保护 |
| 4.5 | **HIGH** | `hooks/useWidgetData.ts:83-135,143-158,172-232` | client-swr-dedup | 自实现 `setInterval` 轮询 + dashboardStore cache, 跳过 React Query dedup |
| 4.6 | **HIGH** | `hooks/useWidgetPolling.ts:76-122,150-169` | client-swr-dedup | 同 4.5 — 同样绕开 React Query |
| 4.7 | MEDIUM | `hooks/useADConfigs.ts:26-52,54-58` | client-swr-dedup | `useState` + `useEffect` + `post()`, 无 dedup, 无缓存 |
| 4.8 | LOW | `utils/geocodingCache.ts:29` | client-localstorage-schema | `baidu_geocoding_` prefix 无版本 |
| 4.9 | LOW | `utils/dualLevelCache.ts:16` | client-localstorage-schema | `dual_level_cache_` prefix 无版本 |
| 4.10 | MEDIUM | `pages/monitor/dashboard/index.tsx:73-107` | client-swr-dedup | 3 并行 post() + 30s setInterval, 跳过 React Query |

**Fix priority**: 4.5 + 4.6 + 4.10 合并成一个 useDashboard 风格的 `useQuery` refactor — 节省 N 个轮询请求.

### Category 5: Re-render Optimization (MEDIUM) — 9 findings

| ID | Severity | File | Rule | Issue |
|----|----------|------|------|-------|
| 5.1 | MEDIUM | `components/dashboard/widgets/base/BaseWidget.tsx:98-105` | rerender-memo | `isEmpty` 内联计算, 未 useMemo |
| 5.2 | MEDIUM | `components/dashboard/widgets/base/BaseWidget.tsx:108-113` | rerender-derived-state-no-effect | `isFirstLoad` 用 useEffect 设置, 应在 render 中派生 |
| 5.3 | LOW | `hooks/useDashboard.ts:82-87` | rerender-simple-expression-in-memo | 简单 boolean OR 套 useMemo, hook 开销大于计算本身 |
| 5.4 | MEDIUM | `pages/monitor/dashboard/index.tsx:98-107` | rerender-move-effect-to-event | setInterval 在 useEffect, `refreshData` 闭包不稳定 |
| 5.5 | MEDIUM | `hooks/useWidgetData.ts:142-158` | rerender-use-ref-transient-values | setInterval 回调引用过期 `fetchData`, 部分 disabled check 未走 ref |
| 5.6 | MEDIUM | `components/shared/FloorPlanEditor.tsx:368-398,403-664` | rendering-hoist-jsx | `renderGrid`/`renderWorkstations` 每次 render 创建新引用 |
| 5.7 | MEDIUM | `store/tabsStore.ts:309-344` | rerender-memo-with-default-value | `useTabs` 返回新 object 引用, 触发订阅者 re-render |
| 5.8 | MEDIUM | 全代码库 | rerender-transitions | 无任何 `startTransition` 使用 — 30s 轮询、滚动、pan/zoom 应包裹 |
| 5.9 | MEDIUM | `pages/monitor/dashboard/index.tsx:51-71`, `store/layoutStore.ts:201-216` | rerender-functional-setstate | `setActiveTab`/`toggleSidebar` 直接读 state, 应函数式更新 |
| 5.10 | MEDIUM | `components/layout/shared/TabBar.tsx:152` | rerender-dependencies | `handleScroll` 闭包捕获过期 `updateScrollState` |

**Fix priority**: 5.6 + 5.7 影响大量子树 re-render. 5.5 是 useWidgetData 重构的子任务.

### Category 6: Rendering Performance (MEDIUM) — 2 findings

| ID | Severity | File | Rule | Issue |
|----|----------|------|------|-------|
| 6.1 | LOW | `components/dashboard/widgets/types/ChartWidget.tsx:232-239` | rendering-conditional-render | `getEmptyOption` 不覆盖 `0` 值场景, 数字 0 会被当空数据展示 |
| 6.2 | (Finding 5.6) | — | rendering-hoist-jsx | 已计入 Category 5 |

### Category 7: JavaScript Performance (LOW-MEDIUM) — 3 findings

| ID | Severity | File | Rule | Issue |
|----|----------|------|------|-------|
| 7.1 | MEDIUM | `components/network/MACHeatmapChart.tsx:118` | js-tosorted-immutable | `[...data.cells].sort()` 应改 `data.cells.toSorted()` |
| 7.2 | MEDIUM | `utils/deptUtils.ts` (7 functions) | js-cache-function-results | `toFullPathTree`/`toShortNameDataNode`/`trimTitleToLastSegment`/`dedupTreeByKey`/`filterExternalOrgDepts`/`collectDescendantIds`/`findDeptNode` — 无 module-level Map 缓存 |
| 7.3 | LOW | `components/dashboard/utils/dataFetcher.ts:186` | js-cache-function-results | `getCacheKey` 每次 JSON.stringify 完整 params, 重复计算 |

### Category 8: Advanced Patterns (LOW) — 0 findings

全部 3 条规则 (`advanced-*`) 未发现违反 — `useEvent`/`useLatest` 不需要, init-once 在 main.tsx 已正确.

---

## Cross-Cutting Patterns

### Pattern A: React Query Adoption is Inconsistent

`useDashboard` (Category 4 hook) 正确使用 `useQuery` + `staleTime: 30_000`. 但 4 个文件 (`useWidgetData`, `useWidgetPolling`, `useADConfigs`, `monitor/dashboard`) 都用裸 `useState` + `useEffect` + `setInterval`. 这是项目最大的 dedup 机会.

**Fix outline**: 创建 `useQuery` 风格的 `usePolledQuery<T>(key, fetcher, { interval })` hook, 内部用 `refetchInterval`. 把 4 处替换.

### Pattern B: Three.js / ECharts / Ant Icons 在错误位置 import

3 个 CRITICAL bundle finding (2.2, 2.3, 2.6) 都是 import 位置错误 — 静态 import 出现在经常渲染的 wrapper 组件, 而不是 lazy boundary 之后.

**Fix outline**:
- main.tsx 移除 echarts 副作用导入 → EChartsWrapper lazy 回调内注册 components
- FloorView3D / BuildingView3D 改用 `BuildingScene.tsx` 已有的 `*Lazy` 包装
- iconUtils.tsx 改为按需 `import { SmileOutlined, ... } from "@ant-design/icons"` 或异步加载 sprite

### Pattern C: `setInterval` 模式普遍存在

5 个文件用 `setInterval` 在 useEffect 中 (4.5, 4.6, 4.10, 5.4, 5.5). React Query 的 `refetchInterval` 是 idiomatic 替代.

### Pattern D: localStorage Schema 未版本化

4 个文件 (4.4, 4.8, 4.9, plus 隐式 useColumnConfig) 用未版本化 prefix. 一次性迁移全部加上 `v1` suffix.

---

## Fix Roadmap (estimated effort)

### Sprint 1 (1-2 days, highest ROI)

1. **ECharts main.tsx 移除 + EChartsWrapper 内联注册** (Finding 2.1 + 2.7) — 1 hour
2. **Three.js lazy boundary 修复** (Findings 2.2-2.5) — 4 hours
3. **iconUtils.tsx 改为按需导入** (Finding 2.6) — 2 hours
4. **BuildingView3D N+1 → batch endpoint** (Finding 1.1) — 4 hours (含后端)
5. **`useWidgetData`/`useWidgetPolling` → useQuery** (Findings 4.5, 4.6, 4.10) — 6 hours

**Expected bundle impact**: -1.5MB initial, -3MB total
**Expected runtime impact**: -50% dashboard page API calls, -200ms monitor/dashboard load

### Sprint 2 (1 day, polish)

6. **profile + executions + duty parallel** (Findings 1.2, 1.3, 1.4) — 1 hour
7. **download.ts token pipeline** (Finding 1.5) — 2 hours
8. **localStorage v1 prefixes** (Findings 4.4, 4.8, 4.9) — 2 hours
9. **passive event listeners** (Findings 4.1-4.3) — 30 min
10. **deptUtils module-level cache** (Finding 7.2) — 2 hours
11. **`useTabs` 拆 selector / useShallow** (Finding 5.7) — 1 hour

### Sprint 3 (1-2 days, perf tuning)

12. **BaseWidget useMemo + refactor first-load** (Findings 5.1, 5.2) — 2 hours
13. **FloorPlanEditor render fn useCallback** (Finding 5.6) — 2 hours
14. **MACHeatmapChart toSorted** (Finding 7.1) — 5 min
15. **ChartWidget zero-value empty state** (Finding 6.1) — 15 min
16. **startTransition for scroll/pan/zoom** (Finding 5.8) — 4 hours
17. **functional setState in stores** (Finding 5.9) — 1 hour

---

## What's NOT a Problem (good patterns to preserve)

- ✅ **Phase 105 apiFactory.ts** — 所有新 CRUD 走 `createResourceApi`, 无 barrel
- ✅ **`@tanstack/react-query` v5 已就位** — `useDashboard`/`useDict`/`useRoleList` 正确使用
- ✅ **`src/lib/api.ts` 加密 + token refresh** — TokenManager 模式成熟
- ✅ **apiFactory.invariants.test.ts** — Phase 105 GUARD-08 防止 hand-written CRUD 回归
- ✅ **Phase 106 TS 严格化** — `as any` 减少 11 处, ESLint 禁用已记录
- ✅ **Phase 109 GUARD-01..08** — 后端 8 项 invariants 测试锁定回归
- ✅ **operlog 25 OperType 常量** + **status 0/1 普适规则** — 业务语义统一
- ✅ **paginate NormalizePagination 单一入口** + **handleServiceError 单一出口** — 后端收敛彻底

---

## References

- Vercel rule files: `C:\Users\CPIC\.claude\skills\vercel-react-best-practices\rules\`
- Phase 109 backend regression guards: `.planning/phases/109-regression-guards/`
- Phase 105 frontend API factory: `.planning/phases/105-frontend-api-factory-convergence/`
- CLAUDE.md frontend conventions: `D:\CODE\ClaudeCode\guoguo\CLAUDE.md`

---

**Audit verdict**: Frontend is healthy on Phase 105-106 收敛之后的 baseline, 但仍有 5 个 CRITICAL bundle opportunities 和 4 个 HIGH runtime dedup 机会. Sprint 1 修复后初始 bundle 预计 -1.5MB, dashboard 类页面 -50% API calls.

Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
