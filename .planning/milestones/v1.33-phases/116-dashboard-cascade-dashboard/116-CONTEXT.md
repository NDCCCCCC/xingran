# Phase 116: dashboard-cascade（Dashboard 重渲染级联消除） - Context

**Gathered:** 2026-09-14
**Status:** Ready for planning
**Mode:** Auto-generated (autonomous mode + yolo — discuss skipped, ROADMAP used as spec)

<domain>
## Phase Boundary

可配置仪表盘（`/dashboard-system`）打开与数据刷新不再出现 N² 重渲染级联——单个 widget 数据到达只重渲该 widget；dashboard 模块 9 处整店订阅收敛为字段 selector；widget 数据 L1 缓存从响应式 state 移出（模块级 Map + getState 读写，参考 `noticeStore` P1-M4 先例）；拖拽布局期间 widgets 不被级联重渲。范围仅前端 `xingran-react-frontend/src/pages/dashboard-system/`、`hooks/useWidgetData.ts`、`store/dashboardStore.ts`、`DashboardGrid.tsx`，不动后端，不引入新业务功能。

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion

All implementation choices are at Claude's discretion per autonomous mode + D-05 复用范本约束. Key constraints (from REQUIREMENTS.md D-01/D-02/D-04/D-05 + Phase 115 决策保持):

- **DASH-01 / DASH-03 selector 风格**：复用 Phase 115 `useRouteTabs:45` 范本——逐字段 `useDashboardStore((s) => s.fieldX)`，action 引用稳定即可，不引入 `useShallow`（维持全库 0 使用）
- **DASH-02 L1 缓存方案**：复用 `noticeStore` P1-M4 先例——模块级 `Map<string, widgetData>` + `getState().x` 读写，不订阅响应式；或在原 state 写入前 deep-equal 跳过无变化（phase 内 planner 二选一）
- **DASH-04 DashboardGrid**：移除 `useWindowSize` 订阅（`useState(() => clientWidth)` 初始化一次）；`gridProps` 三件套（layouts 包装/containerPadding/handleLayoutChange）useMemo 化，deps 由 planner 按调用图收敛
- **回归纪律**：纯性能重构以现有测试零回归为准（selector 化 / Map 出 state / useMemo 化均不改行为）；若引入 cache 命中行为变化则附回归测试
- **不动文件交集**：与 Phase 115 SELECTOR-03 同触 DynamicRoutes.tsx 无关（DASH 在 `dashboard-system/` 与 `dashboardStore`，无文件交集）
- **前置依赖**：Phase 115 selector 范式已落位（useRouteTabs/useLayout/路由层已逐字段化），DASH-03 复用同风格无歧义

### Grey Areas（auto-accepted, yolo 模式）

- **DASH-02 缓存出 state vs deep-equal 比较**：采用"模块级 Map + getState 读写"（noticeStore P1-M4 先例，更彻底；DASH 命中数据高频轮询，Map 模式完全避开 selector 重渲）
- **DashboardGrid clientWidth 初始化时机**：mount 后一次 useEffect 设置 state（与 useLayoutEffect 等价但避免 SSR warning）；mount 后窗口 resize 期间不感知（接受 trade-off）
- **9 处整店订阅改动粒度**：每处仅改订阅形态（state 字段逐字段化、action 引用稳定），不改 effect 依赖数组、不改业务逻辑、不改数据流（D-04 最小 diff）
- **widget memo 依赖**：`useWidgetData` action 引用稳定后 widget memo 自然恢复，无需额外 React.memo 包装

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `src/store/noticeStore.ts` P1-M4 模块级 Map + getState 缓存模式（DASH-02 复用先例）
- `src/hooks/useRouteTabs.ts:45-48` 逐字段 selector 范本（DASH-01/03 复用风格）
- `src/hooks/useTabs.ts` / `useLayout.ts` Phase 115 已落位的 14/10 字段 selector 化（参照同模块改法）
- `src/lib/apiFactory.ts` createResourceApi（DASH 数据获取复用既有工厂）

### Established Patterns
- Zustand 单字段 selector：`const x = useStore((s) => s.x)`（全库 0 useShallow，115 决策保持）
- 操作日志：handler 写操作走 `operlog.Record(...)`（DASH 范围不涉及后端写）
- 缓存出 state 模式：`noticeStore` P1-M4（模块级 Map + `getState().x` 读写，selector 仅订阅元数据）

### Integration Points
- `src/pages/dashboard-system/` 9 个组件文件（index/DashboardHome/DashboardList/DashboardView/DashboardEdit/edit/view/WidgetEditor/DashboardSettings）
- `src/store/dashboardStore.ts`（widget 元数据 state + cache state）
- `src/hooks/useWidgetData.ts`（widget 数据获取 hook）
- `src/components/dashboard/DashboardGrid.tsx`（响应式布局组件）

</code_context>

<specifics>
## Specific Ideas

无新需求——纯性能重构（D-04 零回归口径）。关键 trade-off 已在 decisions 中标注。复用 Phase 115 已落位的 selector 风格与 noticeStore P1-M4 缓存出 state 模式，确保最小 diff、不引入新依赖、不破坏既有测试。

</specifics>

<deferred>
## Deferred Ideas

无——讨论保持在 phase 范围内。Phase 115 已挂账的 WR-01（my-notices/detail 空 data 时 setLoading 不可达）与本 phase 无关；WR-04（DashboardScopeSelector 系统仪表盘约束矛盾态）已挂账后续 phase 处理。

</deferred>