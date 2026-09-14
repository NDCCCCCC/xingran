# Phase 115: selector-completion（Zustand selector 收尾） - Context

**Gathered:** 2026-09-14
**Status:** Ready for planning
**Mode:** Auto-generated (infrastructure phase — pure selector refactor, all-technical success criteria, no user-facing design decisions)

<domain>
## Phase Boundary

全库剩余整店订阅清零（PROJECT 口径 26 处）——任意 store 切片变化只重渲真正消费该字段的组件，键击/弹窗等高频路径不再被无关 store 变化牵连重渲。范围 = SELECTOR-01~05（useTabs/useLayout 内部 selector 化 / 路由层 RouteGuard+DynamicRoutes / 3D 页 5 处 visualizationStore / 5 处 action-only 清理 / 无参整店订阅 grep 归零）。纯性能重构，D-04 零回归口径（现有测试 + lint/type-check 零回归为准，无行为变更）。

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion
All implementation choices are at Claude's discretion — pure infrastructure/performance phase. Use REQUIREMENTS.md SELECTOR-01~05 条目、success criteria 行号锚点与 D-05 复用范本（useRouteTabs:45 selector 风格）指导实现。

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `useRouteTabs:45` 逐字段 selector 范本（D-05 锁定）
- Phase 114 已改造同模块 3D 页两组件（HubeiMap/HubeiMapGL）——本 phase SELECTOR-04 触及同文件 73/86 行，文件已热
- Zustand v5（`create<T>()(...)` curried 形态）selector 语义成熟

### Established Patterns
- 整店订阅形态：`const store = useXxxStore()` 无参调用 + 解构（tabsStore:310-326 14 字段 / layoutStore:287-299 10 字段）
- action-only 订阅：仅取 action 引用（action 引用在 Zustand 中稳定，可用 `useXxxStore(s => s.action)` 单字段订阅）
- grep 断言验收：`useXxxStore()` 无参调用归零为机械可验标准

### Integration Points
- useTabs/useLayout 是内部 hook（store 文件导出），改造收敛在 store 文件内部，消费组件零改动
- 路由层 RouteGuard/DynamicRoutes 影响 menuStore 订阅面（DATA-02 在 Phase 118 也触及 DynamicRoutes——本 phase 先落 selector 避免同文件冲突，ROADMAP 已注明依赖顺序）

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches. REQUIREMENTS.md 已给出行号锚点与清零口径（无参 `useXxxStore()` grep 归零）。

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope。

</deferred>
