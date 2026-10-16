---
phase: 15-performance-optimization
plan: 04a
wave: 3
status: complete
requirements:
  - PERF-04
commit: feat(15-04a): add MAC port usage heatmap backend service + handler
---

# 15-04a 热力图后端 API

## 完成内容

实现 PERF-04 后端部分: heatmap service + handler + 路由注册。后端数据源严格走 MV-04 (mv_mac_port_daily_count),不走原表 sys_device_mac_history;复用 15-03 缓存装饰器。

## 实施细节

### 新建文件

1. **`internal/services/mac_history_heatmap_service.go`** — 热力图 service
   - 接口 `MACHistoryHeatmapService` 暴露 `QueryHeatmap(ctx, req) (*HeatmapResult, error)`
   - 数据结构: `HeatmapCell` (device / interface / date / changeCount) + `HeatmapResult` (cells + topN + start + end)
   - 默认时间范围 7 天 (与 Phase 14 D-14-D-07 一致)
   - 默认 topN 100, 从 sys_config `network.mac.perf.heatmap_top_n` 读取, fallback 100
   - `QueryHeatmap` 走 cache-aside: 复用 `BuildMACQueryCacheKey("heatmap", req)` + `dataCache.GetOrSet`
   - `queryHeatmapFromMV` 直查物化视图 MV-04 (GORM Raw SQL)
   - 非 PostgreSQL 返回空结果 (沿用降级模式)
   - **不走** sys_device_mac_history 原表 (D-17 锁定)

2. **`internal/api/v1/network/mac_history_heatmap_handler.go`** — 热力图 handler
   - `QueryHeatmap(c *gin.Context)`: Bind JSON → 调 service → response.Success/Error
   - 错误处理沿用 `response.Error` 包装风格

### 修改文件

3. **`internal/api/v1/network/mac_history_router.go`** — 注册 heatmap 路由
   - 添加 `r.POST("/history/heatmap", heatmapHandler.QueryHeatmap)`
   - 创建 `heatmapService` + `heatmapHandler` 实例 (dataCache/perfConfig 传 nil 走降级)
   - applogger 日志追加 `/history/heatmap`

## 验证结果

- ✅ `go build ./...` 退出码 0
- ✅ `POST /network/history/heatmap` 路由已注册
- ✅ Service 数据源 `mv_mac_port_daily_count` 严格符合 D-17
- ✅ Cache key `mac:query:heatmap:<sha256(params)>`, TTL 5 分钟

## 决策遵循

- D-16: 设备 × 端口 二维热力图 (X/Y 轴定义在 15-04b 前端) ✅
- D-17: 数据源 MV-04 + cache-aside ✅
- D-18: 路由路径 `/network/history/heatmap` (与 trajectory 对齐) ✅
- Claude's Discretion: 颜色梯度 (前端 ECharts 默认蓝-绿-黄-红) 留给 15-04b

## 后续 wave 依赖

- 15-04b: 前端 heatmap 页 + ECharts 组件 + 路由 + 菜单 (本会话 context 受限未完成)
- 15-05: 验证 E2E (前端 → 后端 → MV-04)
