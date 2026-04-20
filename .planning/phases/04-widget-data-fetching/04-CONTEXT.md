# Phase 4: Widget 数据获取机制 - Context

**Gathered:** 2026-04-20
**Status:** Ready for planning

<domain>
## Phase Boundary

实现后端 Widget 数据获取核心功能，支持从 API 端点获取数据并进行转换处理。

**In scope:**
- `GetWidgetData` 方法实现
- `GetBatchWidgetData` 批量获取实现
- 数据转换管道（JSONata 表达式求值）
- 三种数据源类型支持（API、WebSocket、Static）
- 权限过滤
- Widget 级缓存

**Out of scope:**
- 前端交互（Phase 5）
- WebSocket 连接管理（Phase 6）
- 实时数据订阅（Phase 6）
- UI 优化（Phase 7）

</domain>

<decisions>
## Implementation Decisions

### API 调用方式
- **D-01:** 使用内部服务调用方式获取数据
  - 直接调用 Service 方法，性能最优
  - 通过路由注册表找到对应的 Handler/Service
  - 权限验证复用现有机制（`CheckAccess`, `getUserPermissions`）
  - 不使用 HTTP 客户端调用，避免性能开销和认证传递问题

### 数据转换实现
- **D-02:** 使用 JSONata 表达式引擎实现数据转换
  - 引入 `github.com/jsonata-go/jsonata` 库
  - 支持复杂表达式（如 `data.items[0].value`）
  - 支持聚合函数（sum、avg、count、min、max）
  - 支持排序（asc、desc）和限制（limit）
  - 配置驱动，无需手写代码

### 批量获取策略
- **D-03:** 使用并行获取策略
  - 使用 `golang.org/x/sync/errgroup` 并行获取
  - 控制并发数（最大 10 个并发）
  - 单个失败不影响其他 Widget

- **D-04:** 部分失败继续处理
  - 返回每个 Widget 的数据或错误
  - 前端单独处理失败项（显示错误状态）
  - 不采用快速失败模式

### 缓存策略
- **D-05:** Widget 级缓存
  - 每个 Widget 数据独立缓存
  - TTL 基于 Widget 的 `refreshInterval` 配置
  - 缓存 Key 格式：`widget:data:{widgetID}:{paramsHash}`

- **D-06:** TTL 自动过期
  - 根据 `Widget.refreshInterval` 自动过期
  - 默认 TTL 5 分钟（未配置 refreshInterval 时）
  - 手动刷新接口绕过缓存

### Claude's Discretion
- 超时处理：单个 Widget 获取超时 5 秒
- 并发控制：使用 semaphore 或 errgroup.SetLimit(10)
- 错误响应格式：`{"widgetId": "xxx", "error": "message", "code": 500}`

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### 数据模型
- `internal/models/dashboard.go` — Widget、DataSource、Transform 数据结构定义
- `internal/core/db/migrations/045_create_dashboard_tables.sql` — 数据库表结构

### 现有实现
- `internal/services/system/dashboard_service.go` — 已实现的方法和未实现的方法签名
- `internal/api/v1/system/dashboard_handler.go` — Handler 层实现模式
- `internal/api/v1/system/dashboard_router.go` — 路由配置

### 权限机制
- `internal/services/api_endpoint_service.go` — API 端点权限过滤

### 前端类型
- `ruoyi-react-frontend/src/types/dashboard.ts` — TypeScript 类型定义

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **DashboardServiceImpl**: 已有完整的 CRUD 实现，只需添加 GetWidgetData/GetBatchWidgetData 方法
- **缓存模式**: `dashboardCachePrefix` 已定义，复用 Redis 缓存
- **权限检查**: `CheckAccess` 方法可直接复用
- **用户上下文**: `UserContext` 结构和提取方法已实现

### Established Patterns
- **Handler-Service 模式**: Handler 依赖 Service 接口，不直接访问 DB
- **响应包装器**: 使用 `response.Success()` / `response.Error()`
- **错误处理**: 使用 `apperrors` 包定义业务错误
- **缓存 Key**: 使用前缀 + ID 格式，如 `dashboard:{id}`

### Integration Points
- **路由**: `/dashboard-system/widgets/:id/data` 和 `/dashboard-system/widgets/batch-data`
- **Handler**: `DashboardHandler.GetWidgetData` 和 `DashboardHandler.GetBatchWidgetData`
- **Service**: `DashboardService` 接口添加方法签名

### 待实现方法签名
```go
// GetWidgetData 获取单个 Widget 数据
func (s *DashboardServiceImpl) GetWidgetData(ctx context.Context, widgetID string, bypassCache bool) (interface{}, error)

// GetBatchWidgetData 批量获取 Widget 数据
func (s *DashboardServiceImpl) GetBatchWidgetData(ctx context.Context, widgetIDs []string, bypassCache bool) (map[string]WidgetDataResult, error)

type WidgetDataResult struct {
    Data   interface{} `json:"data,omitempty"`
    Error  string      `json:"error,omitempty"`
    Cached bool        `json:"cached"`
}
```

</code_context>

<specifics>
## Specific Ideas

- 参考 Zabbix 的 Widget 数据获取模式：配置驱动、支持多种数据源
- 数据转换表达式语法参考 JSONata（https://jsonata.org）
- 批量获取时使用 errgroup 实现并行+错误收集

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 04-widget-data-fetching*
*Context gathered: 2026-04-20*
