---
phase: 15-performance-optimization
plan: 04a
type: execute
wave: 3
depends_on:
  - 15-02
  - 15-03
files_modified:
  - internal/services/mac_history_heatmap_service.go
  - internal/api/v1/network/mac_history_heatmap_handler.go
  - internal/api/v1/network/mac_history_router.go
  - internal/services/mac_history_heatmap_service_test.go
autonomous: true
requirements:
  - PERF-04
must_haves:
  truths:
    - POST /network/history/heatmap exists in mac_history_router.go and returns data from mv_mac_port_daily_count
    - Heatmap service implements cache-aside with key mac:query:heatmap:<sha256(params)>, TTL 5 min
    - topN defaults to sys_config.network.mac.perf.heatmap_top_n (default 100)
    - go build ./... exits 0
  artifacts:
    - path: internal/services/mac_history_heatmap_service.go
      provides: Heatmap service reads MV-04 with cache-aside
      contains: mv_mac_port_daily_count
    - path: internal/api/v1/network/mac_history_heatmap_handler.go
      provides: POST /network/history/heatmap handler
      contains: QueryHeatmap
  key_links:
    - from: internal/api/v1/network/mac_history_heatmap_handler.go
      to: internal/services/mac_history_heatmap_service.go
      via: service call
      pattern: macHistoryHeatmapService\.QueryHeatmap
    - from: internal/services/mac_history_heatmap_service.go
      to: mv_mac_port_daily_count
      via: GORM raw SQL
      pattern: FROM mv_mac_port_daily_count
---

<objective>
实现 PERF-04 后端部分: heatmap service + handler + 路由注册 + 单元测试。后端数据源严格走 MV-04 (mv_mac_port_daily_count),不走原表 sys_device_mac_history;使用 15-03 已交付的 WithCacheAside 包裹查询。

Purpose: 把 5 分钟预聚合的物化视图 MV-04 暴露为后端 API,供前端 heatmap 页面拉取。

Output: 4 个后端文件 (1 service + 1 handler + 1 router 修改 + 1 test)。
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@.planning/ROADMAP.md
@.planning/REQUIREMENTS.md
@.planning/phases/15-performance-optimization/15-CONTEXT.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: 实现 heatmap 后端服务</name>
  <files>internal/services/mac_history_heatmap_service.go</files>
  <read_first>
    - internal/services/mac_history_query_service.go (BuildMACQueryCacheKey 签名)
    - internal/services/mac_history_cache_decorator.go (WithCacheAside 签名)
    - internal/services/cache_config_service.go (sys_config 读 topN 模式)
  </read_first>
  <action>
    新建 `internal/services/mac_history_heatmap_service.go`:

    1. 包名 `services`。
    2. 定义结构体:
       - `HeatmapQuery{ StartTime string \`json:"startTime"\`; EndTime string \`json:"endTime"\`; TopN int \`json:"topN,omitempty"\` }`
       - `HeatmapPoint{ DeviceID, DeviceName, InterfaceName, Date string; ChangeCount int }`
       - `HeatmapResult{ Points []HeatmapPoint; TotalDevices, TotalPorts int; TopN int; StartTime, EndTime string }`
    3. 接口 `MACHistoryHeatmapService { QueryHeatmap(ctx, *HeatmapQuery) (*HeatmapResult, error) }`。
    4. 实现 `macHistoryHeatmapServiceImpl{ db *gorm.DB; dataCache DataCacheService; cacheConfig CacheConfigService }`。
    5. 构造函数 `NewMACHistoryHeatmapService(db, dataCache, cacheConfig) MACHistoryHeatmapService`。
    6. 方法 `QueryHeatmap` 实现:
       - 解析 StartTime / EndTime (RFC3339 → time.Time),失败返回 error。
       - topN 为 0 时调 `cacheConfig.GetInt(ctx, "network.mac.perf.heatmap_top_n")`,失败 fallback 100。
       - 用 `BuildMACQueryCacheKey("heatmap", req)` 构造键。
       - 用 `WithCacheAside(ctx, dataCache, key, 5*time.Minute, queryFn)` 包裹。
       - queryFn 内: 直接 SELECT * FROM mv_mac_port_daily_count (严禁 sys_device_mac_history 全表聚合); device_name 通过 sys_device 表 IN 查询批量填充; 扫到 []HeatmapPoint。
       - 组装 HeatmapResult 返回 TotalDevices / TotalPorts 统计。
    7. 复用 15-03 已交付的 WithCacheAside,不在此文件重复实现。
  </action>
  <verify>
    <automated>cd D:/CODE/ClaudeCode/xingran-go-backend && go build ./internal/services/... 2>&1 | head -20</automated>
  </verify>
  <done>
    - mac_history_heatmap_service.go 存在,导出 MACHistoryHeatmapService 接口
    - QueryHeatmap 体内包含 FROM mv_mac_port_daily_count 字符串
    - go build ./internal/services/... 退出码 0
  </done>
</task>

<task type="auto">
  <name>Task 2: heatmap handler + 路由注册</name>
  <files>internal/api/v1/network/mac_history_heatmap_handler.go, internal/api/v1/network/mac_history_router.go</files>
  <read_first>
    - internal/api/v1/network/mac_history_handler.go (handler 模板 + response.Page 模式)
    - internal/api/v1/network/mac_history_router.go (现有路由注册风格)
    - internal/core/core.go (确认 DataCacheService / CacheConfigService 是直接公开字段,行 173/176)
  </read_first>
  <action>
    新建 `internal/api/v1/network/mac_history_heatmap_handler.go`:

    1. 包名 `network`。
    2. struct `MACHistoryHeatmapHandler { heatmapService services.MACHistoryHeatmapService }`。
    3. 构造函数 `NewMACHistoryHeatmapHandler(svc services.MACHistoryHeatmapService) *MACHistoryHeatmapHandler`。
    4. 方法 `QueryHeatmap(c *gin.Context)`:
       - `var req services.HeatmapQuery` + `c.ShouldBindJSON(&req)`,失败 `response.Error(c, 400, "参数错误")`。
       - 调 `h.heatmapService.QueryHeatmap(c.Request.Context(), &req)`,失败 `response.Error(c, 500, err.Error())`。
       - 成功 `response.Success(c, result)` (不用 response.Page,heatmap 不分页)。

    修改 `internal/api/v1/network/mac_history_router.go`:

    1. 在 `SetupMACHistoryRouter` 函数体内创建 heatmapService:
       `heatmapService := mac_history_query_service.NewMACHistoryHeatmapService(core.GetDB(), core.DataCacheService, core.CacheConfigService)`
       (直接访问 Core 公开字段,不使用 getter;若类型不匹配,加 setter 转换)。
    2. 创建 handler: `heatmapHandler := NewMACHistoryHeatmapHandler(heatmapService)`。
    3. 注册路由: `r.POST("/history/heatmap", heatmapHandler.QueryHeatmap)`。
    4. `applogger.Infof` 末尾追加 `/history/heatmap` 启动日志。

    注意: handler 调用约定与 `MACHistoryHandler` 保持一致。
  </action>
  <verify>
    <automated>cd D:/CODE/ClaudeCode/xingran-go-backend && go build ./... 2>&1 | head -20</automated>
  </verify>
  <done>
    - handler 文件存在,导出 MACHistoryHeatmapHandler struct
    - router 文件包含 r.POST("/history/heatmap"...) 字符串
    - go build ./... 退出码 0
  </done>
</task>

<task type="auto">
  <name>Task 3: heatmap 单元测试</name>
  <files>internal/services/mac_history_heatmap_service_test.go</files>
  <read_first>
    - internal/services/mac_history_query_service_test.go (测试模式)
  </read_first>
  <action>
    新建 `internal/services/mac_history_heatmap_service_test.go`:

    1. 包名 `services`。
    2. 至少 2 个测试:
       - `TestQueryHeatmap_CacheMiss`: mock DataCacheService.Get 返回 miss,断言 SQL 走 MV-04,响应正确。
       - `TestQueryHeatmap_TopNFromConfig`: mock CacheConfigService.GetInt 返回 50,断言 topN=50 被使用。
    3. 风格沿用 15-03 测试模式,使用 gomock 或 sqlmock。
    4. 不需要测试前端(前端在 15-04b)。
  </action>
  <verify>
    <automated>cd D:/CODE/ClaudeCode/xingran-go-backend && go test ./internal/services/ -run TestQueryHeatmap -v 2>&1 | head -20</automated>
  </verify>
  <done>
    - heatmap_service_test.go 含 ≥2 个测试
    - go test ./internal/services/ -run TestQueryHeatmap 退出码 0
  </done>
</task>

</tasks>

<verification>
- go build ./... 退出码 0
- POST /network/history/heatmap 注册到 mac_history_router.go
- 后端数据源来自 MV-04 (mv_mac_port_daily_count)
- go test ./internal/services/ -run TestQueryHeatmap 退出码 0
</verification>

<success_criteria>
- 后端 heatmap API 编译通过 + 测试通过
- 路由注册到 /network/history/heatmap
- 缓存键格式 mac:query:heatmap:<sha256>
- go build ./... 退出码 0
</success_criteria>

<output>
Create `.planning/phases/15-performance-optimization/15-04a-SUMMARY.md` when done
</output>
