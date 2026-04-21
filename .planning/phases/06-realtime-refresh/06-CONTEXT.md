# Phase 6: 实时数据刷新 - Context

**Gathered:** 2026-04-21
**Status:** Ready for planning

<domain>
## Phase Boundary

实现仪表盘数据的实时更新机制，支持 WebSocket 推送和轮询两种模式。

**In scope:**
- WebSocket 连接管理（建立、断开、重连）
- 基于刷新间隔的轮询机制
- 缓存策略（避免重复请求）
- 前端实时数据订阅

**Out of scope:**
- 后端 WebSocket 服务器实现（Phase 4 已完成基础支持）
- Widget 数据获取逻辑（Phase 4 已完成）
- 前端交互组件（Phase 5 已完成）
- 响应式布局优化（Phase 7）

</domain>

<decisions>
## Implementation Decisions

### WebSocket 连接管理
- **D-01:** 连接方式 — 原生 WebSocket API
  - 使用浏览器原生 WebSocket API
  - 封装为 React Hook (useWebSocket)
  - 支持自动重连（指数退避策略）

- **D-02:** 连接状态管理 — Zustand Store
  - 扩展现有 dashboardStore 添加 WebSocket 状态
  - 连接状态：connecting | connected | disconnected | error
  - 提供连接/断开/重连方法

- **D-03:** 重连策略 — 指数退避
  - 初始重连延迟：1秒
  - 最大重连延迟：30秒
  - 重连次数上限：10次
  - 网络恢复后自动重连

### 轮询机制
- **D-04:** 轮询实现 — setInterval + 可配置间隔
  - 每个仪表盘可配置刷新间隔
  - 默认刷新间隔：300秒（5分钟）
  - 最小刷新间隔：30秒
  - 轮询在仪表盘不可见时暂停（Page Visibility API）

- **D-05:** 轮询触发 — 仪表盘级别
  - 刷新间隔在仪表盘级别配置
  - 所有 Widget 共享同一刷新间隔
  - 支持手动刷新按钮

### 缓存策略
- **D-06:** 缓存位置 — dashboardStore
  - 使用现有 widgetDataCache Map
  - 缓存键：widgetId
  - 缓存值：{ data, timestamp }

- **D-07:** 缓存过期 — 基于刷新间隔
  - 缓存有效期 = 刷新间隔 / 2
  - 最小缓存时间：30秒
  - 缓存在数据请求前检查

- **D-08:** 缓存失效 — 手动 + 自动
  - 手动刷新时清除缓存
  - Widget 配置变更时清除对应缓存
  - 仪表盘切换时清除所有缓存

### 前端数据订阅
- **D-09:** 订阅模式 — 发布-订阅
  - 使用 Zustand store 的订阅功能
  - Widget 组件订阅数据变化
  - 数据更新时自动触发 UI 重渲染

- **D-10:** 数据流 — 单向数据流
  - 数据源 → Store → Widget 组件
  - Widget 不直接调用 API
  - 所有数据更新通过 Store

### Claude's Discretion
- 错误处理策略：连接失败、请求超时、数据解析错误
- 加载状态显示：首次加载、刷新中、重连中
- 网络状态检测：navigator.onLine、online/offline 事件
- 性能优化：批量请求、请求去重

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 前端类型定义
- `ruoyi-react-frontend/src/types/dashboard.ts` — Dashboard、WidgetConfig 类型

### 状态管理
- `ruoyi-react-frontend/src/store/dashboardStore.ts` — Zustand store，已有 widgetDataCache

### 服务层
- `ruoyi-react-frontend/src/services/dashboardService.ts` — getWidgetData、getBatchWidgetData API

### Phase 4 实现
- `.planning/phases/04-widget-data-fetching/04-CONTEXT.md` — 后端数据获取决策
- `.planning/phases/04-widget-data-fetching/04-01-SUMMARY.md` — GetWidgetData 实现
- `.planning/phases/04-widget-data-fetching/04-03-SUMMARY.md` — 批量获取和缓存

### Phase 5 实现
- `.planning/phases/05-frontend-interactions/05-CONTEXT.md` — 前端交互决策
- `.planning/phases/05-frontend-interactions/05-02-SUMMARY.md` — DashboardSettings 组件

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **dashboardStore**: 已有 widgetDataCache、cacheWidgetData、getCachedWidgetData、clearWidgetCache
- **dashboardService**: 已有 getWidgetData、getBatchWidgetData API
- **RefreshIntervalSelector**: 已在 Phase 5 实现，支持预设和自定义刷新间隔

### Established Patterns
- **Zustand 状态管理**: 使用 create + persist 中间件
- **React Hooks**: 自定义 Hook 封装复杂逻辑
- **Ant Design 组件**: message、notification 用于提示

### Integration Points
- **WebSocket 连接**: 需要后端 WebSocket 端点（Phase 4 已支持）
- **轮询触发**: 使用 dashboardStore 的 refreshInterval 配置
- **数据更新**: 调用 dashboardService.getBatchWidgetData

### 待实现组件
```
src/hooks/
├── useWebSocket.ts         # WebSocket 连接管理 Hook
├── useWidgetPolling.ts     # Widget 轮询 Hook
└── useNetworkStatus.ts     # 网络状态检测 Hook

src/store/
└── dashboardStore.ts       # 扩展 WebSocket 状态和轮询逻辑
```

</code_context>

<specifics>
## Specific Ideas

- 参考 Zabbix 的实时数据更新机制：WebSocket 推送 + 轮询兜底
- 参考 Grafana 的仪表盘刷新：可配置刷新间隔、暂停刷新
- 网络断开时显示离线提示，恢复后自动重连

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 06-realtime-refresh*
*Context gathered: 2026-04-21*
