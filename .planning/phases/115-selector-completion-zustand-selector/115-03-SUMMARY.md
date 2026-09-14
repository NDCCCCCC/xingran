---
phase: 115-selector-completion-zustand-selector
plan: 03
subsystem: ui
tags: [zustand, react-19, selector, performance, frontend, test-mock]

# Dependency graph
requires:
  - phase: 115-selector-completion-zustand-selector (plan 01/02)
    provides: SELECTOR-01~04 已落位的统一 selector 改造范式（D-05 实操基线）与 useRouteTabs dual-form mock 范本
  - phase: 115-selector-completion-zustand-selector (RESEARCH/PATTERNS)
    provides: 4 个必改 mock 文件逐文件改造对照与 Pitfall 3 冷缓存假失败判定纪律
provides:
  - SELECTOR-05 全部落位：6 处页面级整店订阅（4 action + 2 state 字段 user）单字段 selector 化
  - NotificationBell.tsx 7 个单字段 selector（OQ-1 定案推荐 a 纳入，3 state + 4 action）
  - 4 个测试 mock dual-form 化（let 变量调用时求值保持，isAdmin/dataScope 断言语义零弱化）
  - criterion 5 收口：全库无参 useXxxStore() 调用恰好 12 处且全部归属 Phase 116/120
affects: [116-dashboard-selector, 118-data-menu, 120-dead-code]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "页面级 action/state 单字段订阅：每字段一次 useXxxStore((s) => s.field)，action 引用不包 useMemo（zustand 5.0.15 身份稳定）"
    - "测试 dual-form mock：工厂内 useXxxStoreFn 函数体 selector ? selector(state) : state；对 let 变量保持函数体调用时求值（beforeEach/用例内重赋值继续生效）"

key-files:
  created: []
  modified:
    - xingran-react-frontend/src/pages/login/index.tsx
    - xingran-react-frontend/src/pages/login/index.test.tsx
    - xingran-react-frontend/src/pages/login/__tests__/index.test.tsx
    - xingran-react-frontend/src/pages/profile/index.tsx
    - xingran-react-frontend/src/pages/my-notices/detail.tsx
    - xingran-react-frontend/src/pages/my-notices/__tests__/detail.test.tsx
    - xingran-react-frontend/src/pages/duty/my-duty/index.tsx
    - xingran-react-frontend/src/components/dashboard/settings/DashboardScopeSelector.tsx
    - xingran-react-frontend/src/components/dashboard/settings/__tests__/DashboardScopeSelector.test.tsx
    - xingran-react-frontend/src/components/NotificationBell.tsx

key-decisions:
  - "NotificationBell.tsx:67-75 按 OQ-1 定案推荐 a 纳入本 plan：拆 7 个独立 selector（unreadCount/notifications/loading + 4 action），其两个测试均用真实 store 零 mock 改动"
  - "DashboardScopeSelector.test.tsx 用 dual-form 而非简化返回（T-115-03-A mitigate）：:46-53/:55-63/:75-85 的 isAdmin/dataScope 分支断言语义原样保住并跑绿"
  - "detail.test.tsx 与 DashboardScopeSelector.test.tsx 的 mock 函数体读 let 变量（noticeStoreState/mockUser）保持调用时求值，不在工厂执行时快照"

patterns-established:
  - "Pattern: 组件订阅 selector 化与测试 mock dual-form 化同任务落位——组件改 selector 前先同步 mock，避免中间红态"

requirements-completed: [SELECTOR-05]

# Metrics
duration: 33min
completed: 2026-09-14
---

# Phase 115 Plan 03: SELECTOR-05 收尾 + NotificationBell + mock dual-form 化 Summary

**页面级 6 处整店订阅（4 action-only + 2 state 字段）与 NotificationBell 7 项全部单字段 selector 化，4 个非 selector 兼容测试 mock 同任务 dual-form 化，criterion 5 收口为全库恰好 12 处且全部归属 Phase 116/120**

## Performance

- **Duration:** 33 min（含两轮全量测试 ~21min）
- **Started:** 2026-09-14T02:46:23Z
- **Completed:** 2026-09-14T03:19:30Z
- **Tasks:** 3（Task 3 为验收型，无代码改动无独立提交）
- **Files modified:** 10

## Accomplishments

- SELECTOR-05 前簇 3 文件：login/index.tsx 拆 login/fetchMenus/fetchPermissions 3 个 action selector（跨 authStore/menuStore）、profile/index.tsx updateUser、my-notices/detail.tsx markAsRead（:49 useCallback 依赖数组不动，action 引用语义不变）
- SELECTOR-05 后簇 2 文件：my-duty/index.tsx:64 与 DashboardScopeSelector.tsx:25 的 `user` 改 state 字段单字段订阅（RESEARCH 已纠正非 action），isAdmin 派生与 dataScope 派生计算原样保留
- NotificationBell.tsx:67-75（OQ-1 定案推荐 a）解构块拆 7 个独立 selector：unreadCount/notifications/loading（3 state）+ markAsRead/markAllAsRead/removeNotification/setNotifications（4 action）；:89 `useNoticeStore.getState()` 静态方法不受影响
- 4 个 mock 文件 dual-form 化：login/index.test.tsx（authStore + menuStore 两工厂）、login/__tests__/index.test.tsx（authStore，state 提到工厂作用域 login vi.fn() 身份保持）、detail.test.tsx（let noticeStoreState 调用时求值）、DashboardScopeSelector.test.tsx（let mockUser 调用时求值）
- criterion 5 收口：全库无参 `use[A-Z][A-Za-z0-9]*Store()` 调用恰好 12 处，全部命中三组白名单前缀 `hooks/(useTabSync|useWidgetData).ts`（3 处）/ `pages/dashboard-system/`（7 处）/ `components/dashboard/`（2 处），零清单外命中——全部归属 Phase 116（DASH 11 处）与 Phase 120（useTabSync 死代码 1 处）

## Task Commits

Each task was committed atomically:

1. **Task 1: 前簇 3 处整店订阅改单字段 selector + 3 mock dual-form 化（SELECTOR-05）** - `3b3af05` (feat)
2. **Task 2: 后簇 selector 化 + NotificationBell 7 selector + mock dual-form（SELECTOR-05, OQ-1）** - `8a10865` (feat)
3. **Task 3: Phase gate（全库 grep 断言 + 全量测试 + lint/type-check）** - 验收型 task，零代码改动，无独立提交（验收结果见 Verification 段）

## Files Created/Modified

- `xingran-react-frontend/src/pages/login/index.tsx` - 2 行解构 → 3 个 action selector（登录流程/SM2 链路一行未动）
- `xingran-react-frontend/src/pages/login/index.test.tsx` - authStore + menuStore 两个 vi.mock 工厂改 dual-form
- `xingran-react-frontend/src/pages/login/__tests__/index.test.tsx` - authStore 工厂改 dual-form（state 提到工厂作用域）
- `xingran-react-frontend/src/pages/profile/index.tsx` - updateUser 单字段 selector
- `xingran-react-frontend/src/pages/my-notices/detail.tsx` - markAsRead 单字段 selector，useCallback 依赖不动
- `xingran-react-frontend/src/pages/my-notices/__tests__/detail.test.tsx` - noticeStore 工厂改 dual-form（let 惰性求值保持）
- `xingran-react-frontend/src/pages/duty/my-duty/index.tsx` - user 单字段 selector
- `xingran-react-frontend/src/components/dashboard/settings/DashboardScopeSelector.tsx` - user 单字段 selector，两处派生原样
- `xingran-react-frontend/src/components/dashboard/settings/__tests__/DashboardScopeSelector.test.tsx` - authStore 工厂改 dual-form（let mockUser 惰性求值保持）
- `xingran-react-frontend/src/components/NotificationBell.tsx` - 7 行解构 → 7 个独立 selector

## Decisions Made

- NotificationBell 纳入方式照 OQ-1 定案执行（RESEARCH 推荐 a）：1 处行级改动使 criterion 5 全库口径在本 phase 可验，其两个测试（notificationBell.interact / components.render）用真实 store，零 mock 风险
- mock dual-form 写法统一为 `const useXxxStoreFn: any = (selector?: ...) => selector ? selector(state) : state;`，state 用常量或 let 变量按文件现状选择；不添加 `.getState` 附加项（PATTERNS 已说明 4 个被测组件均只用 hook）
- 不引入 useShallow（维持全库 0 使用）、不用对象 selector、不包 useMemo/useCallback 包裹 action 引用（zustand 5.0.15 action 创建后身份稳定）
- 全量首跑 1 例失败按 RESEARCH Pitfall 3 纪律重跑定性（热缓存重跑 3800/3800 全绿），未修改任何断言口径

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- 全量套件首跑（冷缓存）出现 1 failed / 3799 passed，完整失败输出被管道截断无法定位；按 plan Task 3 action 2 与 RESEARCH Pitfall 3 重跑一次，553 文件 / 3800 tests 全绿——定性为冷缓存超时假失败，非本 phase 回归
- 工作树存在非本 plan 范围的 pre-existing 改动 `internal/services/system/asset_columns_schema.json`（后端文件，session 开始前已 dirty），已按 scope boundary 规则排除在全部提交之外，原样保留
- 未触发 lint-staged 重排版问题：mock 三元写法预先对齐 prettier printWidth=100（menuStore mock 用 state 常量收敛单行三元，保留 `selector ? selector` 字面量供 grep 断言命中）

## Verification

- grep 断言（criterion 5，OQ-1 全库口径）：`grep -rnE "use[A-Z][A-Za-z0-9]*Store\(\)" src --include="*.ts" --include="*.tsx" | grep -vE "__tests__|\.test\."` = 12 处，`grep -vE` 三组白名单前缀后 0 命中
- Task 1：3 源文件无参调用 0 + 3 mock 含 `selector ? selector`；受影响测试 6 文件 17 tests 全绿；type-check exit 0
- Task 2：3 源文件无参调用 0 + NotificationBell `useNoticeStore((s) =>` 计数 = 7 + DashboardScopeSelector.test.tsx dual-form 命中；受影响测试 4 文件 23 tests 全绿（含 admin 分支 isAdmin/dataScope 断言）；type-check exit 0
- Task 3：`npx vitest run` 重跑 553 files / 3800 tests 全绿；`npm run lint` exit 0（0 errors，1366 warnings 均为存量）；`npm run type-check` exit 0
- threat_model 落实：T-115-03-A dual-form mock 非简化返回且 admin 分支断言跑绿；T-115-03-B detail.tsx useCallback 依赖数组逐字未动；T-115-03-C grep 断言口径未修改（12 处 + 白名单前缀写死）

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 115 全部 3 plan 完成：SELECTOR-01~05 + NotificationBell 落位，本 phase 范围 16+1 处归零，剩余 12 处全部有明确归属（Phase 116 dashboard 11 处 / Phase 120 useTabSync 1 处）
- Phase 116 (DASH) 可直接复用三波次确立的 selector 范式与 dual-form mock 范本（useRouteTabs.test.tsx + 本次 4 文件）
- `test:coverage` / `size` 留给 /gsd:verify-work phase 级验证（plan Task 3 action 3 明示不在本 task 重复）

## Threat Flags

无新增攻击面：本 plan 零新增依赖（T-115-SC 不适用确认）、零网络端点/auth 路径/schema 变更；唯一安全相关面为 DashboardScopeSelector mock 改造，已按 T-115-03-A 用 dual-form 落实且 admin 分支断言原样跑绿。

## Self-Check: PASSED

- 115-03-SUMMARY.md 存在 ✓
- Task 1 commit `3b3af05` 存在 ✓
- Task 2 commit `8a10865` 存在 ✓
- 提交无意外文件删除 ✓

---
*Phase: 115-selector-completion-zustand-selector*
*Completed: 2026-09-14*
