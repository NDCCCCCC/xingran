---
phase: 15-performance-optimization
plan: 03
wave: 2
status: complete
requirements:
  - PERF-03
commit: feat(15-03): add cache-aside wrapper for 3 MAC history query methods
---

# 15-03 Redis 缓存中间层集成

## 完成内容

在 `macHistoryQueryService` 实现层注入 cache-aside 装饰:`QueryPortHistory` / `QueryDeviceHistory` / `QueryConnectionStats` 三个高频点查方法加 5 分钟 Redis 缓存,`QueryMACTrajectory` 不缓存。Redis 不可用降级 SQL 直查。

## 实施细节

### 新建文件

1. **`internal/services/mac_history_cache_decorator.go`** — 缓存键构造函数
   - 4 个键前缀常量 (port-history / device-history / stats / heatmap)
   - `BuildMACQueryCacheKey(method, params)` 返回 `<prefix>:<sha256(json(params))>`
   - 序列化用 `encoding/json` (Go struct 字段顺序稳定保证哈希稳定)
   - 哈希用 `crypto/sha256` 标准库 (项目内已有 crypto 库)
   - **不带** `xingran:` 前缀 (Core 缓存层自动加,沿用 D-13-3.2 风格)

### 修改文件

2. **`internal/services/mac_history_query_service.go`** — 注入 cache 字段 + 包装 3 个方法
   - `macHistoryQueryServiceImpl` 追加 `dataCache *DataCacheService` + `perfConfig *CacheConfigService`
   - 新增构造函数 `NewMACHistoryQueryServiceWithCache(db, dataCache, perfConfig)`
   - 新增 helper `perfCacheTTL()`: 读 `network.mac.perf.cache_ttl_seconds` 配置,5 分钟兜底
   - `QueryPortHistory` / `QueryDeviceHistory` / `QueryConnectionStats` 各加 cache-aside 前置
   - 原 SQL 体提取为 `queryPortHistoryFromDB` / `queryDeviceHistoryFromDB` / `queryConnectionStatsFromDB`
   - `QueryMACTrajectory` **不加装饰**,保持原样 (D-12 锁定:参数组合爆炸,命中率低)
   - 缓存命中失败时 `applogger.Warnf` + 走 DB 直查 (D-15 降级锁定)
   - 利用 `DataCacheService.GetOrSet` 已就位能力,不引入新缓存层

## 验证结果

- ✅ `go build ./...` 退出码 0
- ✅ 3 个方法体内含 `dataCache.GetOrSet` 调用
- ✅ `QueryMACTrajectory` 方法体内不含 `dataCache` 调用
- ✅ 缓存键格式 `mac:query:<method>:<sha256>`
- ✅ dataCache 为 nil 时直查 DB (向后兼容现有调用方)

## 决策遵循

- D-12: 缓存范围 port-history + device-history + stats,不含 trajectory ✅
- D-13: 键命名空间 + SHA-256 ✅
- D-14: 失效策略依赖 TTL,5 分钟 (300 秒) ✅
- D-15: Redis 不可用降级 SQL 直查 ✅

## 后续 wave 依赖

- 15-04: heatmap service 复用 `BuildMACQueryCacheKey("heatmap", ...)` + `dataCache.GetOrSet`
- 15-05: 验证 hit / miss / degrade 行为
