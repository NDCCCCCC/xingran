# Phase 7: 用户体验优化 - Context

**Gathered:** 2026-04-21
**Status:** Ready for planning

<domain>
## Phase Boundary

优化仪表盘交互体验，处理边界情况，实现响应式布局。

**In scope:**
- Widget 拖拽布局优化（自动对齐网格、防止重叠）
- 加载状态和错误处理
- 数据为空时的友好提示
- 响应式布局适配（桌面、平板、移动端）

**Out of scope:**
- 后端数据获取（Phase 4 已完成）
- 前端交互组件（Phase 5 已完成）
- 实时数据刷新（Phase 6 已完成）

</domain>

<decisions>
## Implementation Decisions

### Widget 拖拽布局
- **D-01:** 拖拽库 — react-grid-layout（已使用）
  - 保持现有 react-grid-layout 实现
  - 配置网格对齐和碰撞检测
  - 添加拖拽时的视觉反馈

- **D-02:** 布局约束 — 网格对齐
  - 最小宽度：3 列
  - 最小高度：3 行
  - 拖拽时自动对齐到网格
  - 防止 Widget 重叠

### 加载状态
- **D-03:** 加载指示器 — Ant Design Spin
  - 首次加载：全屏 Spin
  - 刷新加载：Widget 级别 Spin
  - 骨架屏：用于首次加载的占位

- **D-04:** 错误处理 — 分级提示
  - 网络错误：全局 notification
  - API 错误：Widget 级别错误提示
  - 数据解析错误：控制台警告 + 默认值

### 空状态
- **D-05:** 空状态提示 — Ant Design Empty
  - 无数据：显示 Empty 组件
  - 无 Widget：显示引导提示
  - 可配置空状态文案和图标

### 响应式布局
- **D-06:** 响应式断点 — 三档
  - 桌面端：≥1200px，24 列
  - 平板端：768px-1199px，12 列
  - 移动端：<768px，6 列

- **D-07:** 响应式策略 — 自动调整
  - 列数自动适配
  - Widget 宽度按比例缩放
  - 移动端隐藏非关键 Widget

### Claude's Discretion
- 动画过渡：使用 CSS transition
- 拖拽手柄：右上角拖拽图标
- 响应式检测：使用 window.matchMedia

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 前端类型定义
- `ruoyi-react-frontend/src/types/dashboard.ts` — LayoutConfig 类型

### 状态管理
- `ruoyi-react-frontend/src/store/dashboardStore.ts` — Zustand store

### 布局组件
- `ruoyi-react-frontend/src/components/dashboard/layout/DashboardGrid.tsx` — 网格布局
- `ruoyi-react-frontend/src/components/dashboard/layout/GridItem.tsx` — 网格项

### Phase 5 实现
- `.planning/phases/05-frontend-interactions/05-CONTEXT.md` — 前端交互决策

### Phase 6 实现
- `.planning/phases/06-realtime-refresh/06-CONTEXT.md` — 实时刷新决策

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **DashboardGrid**: 已使用 react-grid-layout
- **dashboardStore**: 已有 showGridLines、showWidgetBorders 状态
- **Ant Design**: Empty、Spin、Skeleton 组件

### Established Patterns
- **React Grid Layout**: 响应式网格布局
- **Ant Design 组件**: Empty、Spin、Skeleton、notification
- **CSS 媒体查询**: 响应式断点

### Integration Points
- **DashboardGrid**: 添加布局约束和响应式配置
- **Widget 组件**: 添加加载状态和空状态处理
- **DashboardView**: 添加全局加载和错误处理

</code_context>

<specifics>
## Specific Ideas

- 参考 Grafana 的仪表盘布局：拖拽时有网格辅助线
- 参考 Notion 的空状态：友好的引导文案
- 参考 Zabbix 的响应式：移动端自动折叠侧边栏

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 07-ux-optimization*
*Context gathered: 2026-04-21*
