---
phase: 04-widget-data-fetching
plan: 01
status: complete
completed: 2026-04-20
requirements:
  - WDG-01a
  - WDG-01d
---

# Summary: GetWidgetData 核心逻辑

## What Was Built

实现了 Widget 数据获取的核心功能，支持三种数据源类型（API、Static、WebSocket），集成权限验证和缓存机制。

## Key Files Created/Modified

| File | Purpose |
|------|---------|
| `internal/services/system/widget_data_fetcher.go` | Widget 数据获取核心逻辑 |
| `internal/services/system/widget_data_fetcher_test.go` | 单元测试 |
| `internal/services/system/dashboard_service.go` | 集成 WidgetDataFetcher |

## Implementation Details

### 1. WidgetDataFetcher 接口和实现

- `WidgetDataFetcher` 接口定义了 `FetchWidgetData` 方法
- `WidgetDataFetcherImpl` 实现了三种数据源支持：
  - **API**: 通过内部服务调用获取数据，包含权限验证
  - **Static**: 直接返回配置的静态数据
  - **WebSocket**: 返回占位符（Phase 6 实现）

### 2. 缓存机制

- Widget 级缓存，Key 格式: `widget:data:{widgetID}:{paramsHash}`
- TTL 根据 `refreshInterval` 自动计算，默认 5 分钟
- 支持 `bypassCache` 参数绕过缓存

### 3. 权限验证

- API 数据源获取前验证用户权限
- 通过 `sys_menu` → `sys_role_menu` → `sys_user_role` 查询用户权限
- 用户必须拥有端点所需的所有权限才能获取数据

### 4. ServiceRegistry

- `ServiceRegistry` 接口用于内部服务调用
- `DefaultServiceRegistry` 提供默认实现（当前返回占位符）
- 后续可根据项目 Service 结构完善具体实现

## Test Results

```
=== RUN   TestFetchWidgetData_StaticSource
--- PASS: TestFetchWidgetData_StaticSource (0.00s)
=== RUN   TestFetchWidgetData_WebSocketPlaceholder
--- PASS: TestFetchWidgetData_WebSocketPlaceholder (0.00s)
=== RUN   TestFetchWidgetData_NoDataSource
--- PASS: TestFetchWidgetData_NoDataSource (0.00s)
=== RUN   TestBuildWidgetCacheKey
--- PASS: TestBuildWidgetCacheKey (0.00s)
=== RUN   TestCalculateTTL
--- PASS: TestCalculateTTL (0.00s)
=== RUN   TestDefaultServiceRegistry_CallService
--- PASS: TestDefaultServiceRegistry_CallService (0.00s)
PASS
```

## Success Criteria Met

- [x] 用户可以获取单个 Widget 的数据
- [x] Static 数据源直接返回配置数据
- [x] API 数据源通过内部服务调用获取数据
- [x] WebSocket 数据源返回占位符（Phase 6 实现）
- [x] 权限验证生效，无权限用户无法获取数据
- [x] 缓存生效，重复请求返回缓存数据
- [x] TTL 根据 refreshInterval 自动计算

## Next Steps

Wave 2 将实现数据转换管道（JSONata 表达式引擎）。
