# Phase 33 — 前端 Vercel React Best Practices 修复 — Context

**Gathered:** 2026-06-13
**Status:** Ready for planning
**Source:** 审查报告 `../reviews/20260613-frontend-react-best-practices-review.md`（25KB，26 个 defect 已分类）

## 来源说明

本 phase 直接源自 2026-06-13 前端 Vercel React Best Practices 审查报告，共 26 个可优化点
（7 CRITICAL + 5 MEDIUM-HIGH + 8 MEDIUM + 5 LOW-MEDIUM + 1 疑似 Bug）。审查范围为
`xingran-react-frontend/src/**` 全部 506 个 TS/TSX 文件，逐项匹配 Vercel 官方 57 条 React/Next.js
性能规则。

报告整体规范度 ⭐⭐⭐⭐ (4/5)，规则覆盖率 32/53 (60%) 严格遵守；性能优化潜力仍为 **HIGH**。

## 任务清单（按严重度分层，对应 4 个 Wave）

### 🔴 Wave 1 — CRITICAL 修复 + 业务 Bug（8 项，必须先做）

| ID | 文件:行 | 问题 | 规则 | 修复要点 |
|---|---|---|---|---|
| **C1** | `src/store/authStore.ts:60-98` + `src/pages/login/index.tsx:47-58` | 登录后菜单/权限"重复加载 + 瀑布流" | `async-parallel` / `client-swr-dedup` | login 内部不再自动加载菜单；login 页面单次 `Promise.all` 并行拉取 |
| **C2** | `src/hooks/useRealtimeUpdates.ts:127, 161-178` | `connect` 依赖 `widgets` 数组引用导致重连风暴 | `advanced-event-handler-refs` | 改用 `widgetsRef.current` 读取，effect 只依赖 `enabled` |
| **C3** | `src/hooks/useWidgetPolling.ts:73-85, 127-173` | 双重定时器 + 缓存读取 O(n²) | `client-swr-dedup` / `js-set-map-lookups` | 用 `widgetDataCache` Map 直接 `.get(id)`，去重 widgetIds 依赖 |
| **C4** | `src/components/layout/sidebar.tsx:101` | `useMemo` 用 `JSON.stringify(menus)` 作依赖 | `rerender-dependencies` | 改 zustand selector 取稳定引用，去除 stringify |
| **C5** | `src/pages/operations/buildings/useDepartmentData.ts:51-70` | `getOrgName` 每次调用重建 Map | `js-cache-function-results` / `js-index-maps` | 用 `useMemo` 构建 deptMap，传 `id` 直接 `.get(id)` |
| **C6** | `src/pages/operations/workstations/columns.tsx:19-105` + `index.tsx:196-201` | `getWorkstationColumns` 每次返回新数组导致 Table 全表 re-render | `rerender-dependencies` / `rerender-memo` | handleEdit/Delete 改 `useRef` 透传，columns 用 `useMemo([])` |
| **C7** | `src/components/layout/header.tsx:62-100` 等多处 | 内联 `style={{...}}` 大量创建新对象 | `rendering-hoist-jsx` / `rerender-memo-with-default-value` | 静态 style 提取到模块顶层常量（HEADER_STYLE、AVATAR_STYLE 等） |
| **R7** | `src/pages/operations/buildings/index.tsx:355` | 🐛 楼宇状态卡片视图显示字面量 `'1'`，应为 `'停用'` | 业务 Bug | 修复文案映射 |

### 🟠 Wave 2 — MEDIUM-HIGH 修复（5 项）

| ID | 文件:行 | 问题 | 规则 | 修复要点 |
|---|---|---|---|---|
| **M1** | `src/hooks/useTableManager.ts:90-107` | `loadData` deps 含 `current/pageSize` 导致连锁重建 | `rerender-dependencies` | deps 稳定化或使用 `useRef` 持有 latest 值 |
| **M2** | `src/components/shared/ExcelImport.tsx:66-81` | Excel 导入用 `fetch` 绕过 axios 拦截器（401 自动重试失效） | `client-swr-dedup` | 改用 `api.get(url, { responseType: 'blob' })` |
| **M3** | `src/pages/operations/workstations/index.tsx:600-623` | `WorkstationEditModal` 的 `onOk` 内联闭包导致 Modal 每次重渲染 | `rerender-memo` | `onOk` 提取到 `useCallback` 并稳定 deps |
| **M4** | `src/pages/operations/buildings/index.tsx:439-456, 99-124` + `workstations/index.tsx:396-413` | `onChange` 中表单 + 分页 + dept 合并逻辑重复 3 次 | `js-combine-iterations` | 提取 `buildSearchParams(searchForm, deptId, page)` 工具函数 |
| **M5** | `src/components/layout/sidebar.tsx:244-261` | `getSelectedKeys` 每次 render 重算 | `rerender-derived-state` | 用 `useMemo` 包装，依赖 `[location.pathname, menuPathMap]` |

### 🟡 Wave 3 — MEDIUM 修复（7 项，不含已放入 Wave 1 的 R7）

| ID | 文件:行 | 问题 | 规则 | 修复要点 |
|---|---|---|---|---|
| **R1** | `src/router/DynamicRoutes.tsx:91-106, 161-178` | `useMenuStore.fetchAll()` 在 DynamicRoutes 串行等待阻塞首屏 | (UX 改进) | 已缓存的 menu 直接从 store 读，跳过 `InitializingFallback` |
| **R2** | `src/hooks/useWidgetPolling.ts:148-173` | 多实例重复注册 visibilitychange | `client-event-listeners` | 模块顶层单例 + `Set<callback>` 模式 |
| **R3** | `src/hooks/useWidgetPolling.ts:155-165` | visibilitychange 重设 interval 导致定时器漂移 | `client-event-listeners` | effect cleanup 复用同一个 interval 引用 |
| **R4** | `src/components/layout/HybridLayout.tsx:65-166` | 3 个 useEffect 共用 pathname，重复 `store.getState()` | `rerender-derived-state` | 合并为 `useTabSync(pathname)` |
| **R5** | `src/components/layout/sidebar.utils.ts:49-96` | `findMenuByFullPath` 三重递归 | `js-set-map-lookups` | store 中维护 `path → id` Map |
| **R6** | `src/router/routeConfigManager.ts:224-247` | `fallbackBreadcrumb` 反复 split | `js-combine-iterations` | 模块级正则 `PATH_SPLIT = /\/+/` |
| **R8** | `src/components/layout/sidebar.tsx:339` | `<Menu selectedKeys>` 每次新数组 | `rerender-memo` | 同 M5 修复一起 |

### 🟢 Wave 4 — LOW-MEDIUM JavaScript 性能（5 项）

| ID | 文件:行 | 问题 | 规则 | 修复要点 |
|---|---|---|---|---|
| **J1** | `src/lib/api.ts:42-47, 159-165` | `rawAxios` 和 `api` 重复 baseURL/timeout/headers | `js-cache-property-access` | 提取 `BASE_CONFIG` 常量 |
| **J2** | `src/lib/api.ts:35` | (无需修改) 加密 key 用 `Map` 已足够 | — | 跳过 |
| **J3** | `src/utils/dateFormat.ts` 等 | `formatDateTime` 未 memoize | `js-cache-function-results` | LRU 缓存（≤1000 行时影响小，可放最后） |
| **J4** | `src/router/routeConfigManager.ts:118, 225-247` 等 | 字符串 split/join 反复执行 | `js-combine-iterations` | 影响小，可选优化 |
| **J5** | `src/components/layout/sidebar.utils.ts:32-43` | `findMenuById` 递归查找 | `js-set-map-lookups` | store 维护 `Map<id, Menu>`（与 R5 一并处理） |

## 范围之外（Out of Scope）

- 审查报告"表现良好的部分"列举的已落地优化（不需修改）
- 已有 refactor/已通过 quick task 修复的同源问题
- 任何非性能/质量范畴的新功能开发（按"只修报告中的 26 项"原则）

## 依赖

- **强依赖**: Phase 32 完成（v1.14 P1 安全加固与 P2 架构优化）
- **建议顺序**: Wave 1 → Wave 2 → Wave 3 → Wave 4（严重度优先）

## 验收

- 报告 26 个 ID 在 commit message 中可追溯（如 `perf(auth): C1 移除 login 内部菜单加载`）
- TypeScript 类型检查零错误（`npm run type-check`）
- ESLint 零警告（`npm run lint`）
- 工位管理、楼宇管理页面首屏、表格滚动流畅度肉眼可感知提升
- Bug R7 修复后楼宇卡片"停用"状态显示正确文案

## 后续

运行 `/gsd-plan-phase 33` 生成 4 个 Wave 的 detailed PLAN 文件。

---

*Phase: 33-vercel-react-best-practices-20260613-26*
*Context gathered: 2026-06-13 via 审查报告 express path*