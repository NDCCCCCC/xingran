package monitor

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/services"
	monitorServices "github.com/xingran-next/xingran-go-backend/internal/services/monitor"
)

// ==================== 适配器 ====================

// CacheProviderAdapter 缓存提供者适配器
// 将 core.Core.Cache 适配为 CacheProvider、StatsProvider、MultiLevelCacheProvider 和 DirectRedisProvider
type CacheProviderAdapter struct {
	cache interface {
		Get(ctx context.Context, key string) (string, error)
		Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
		Delete(ctx context.Context, key string) error
		Exists(ctx context.Context, key string) (bool, error)
		Expire(ctx context.Context, key string, ttl time.Duration) error
		TTL(ctx context.Context, key string) (time.Duration, error)
		Keys(ctx context.Context, pattern string) ([]string, error)
		FlushDB(ctx context.Context) error
	}
	statsCache interface {
		GetStats(ctx context.Context) (map[string]interface{}, error)
	}
	multiLevelCache interface {
		KeysByLevel(ctx context.Context, pattern string, level string) ([]string, error)
	}
	directRedis interface {
		DirectRedisKeys(ctx context.Context, pattern string) ([]string, error)
		DirectRedisGet(ctx context.Context, key string) (string, error)
		DirectRedisTTL(ctx context.Context, key string) (time.Duration, error)
	}
}

// NewCacheProviderAdapter 创建缓存提供者适配器
func NewCacheProviderAdapter(core *core.Core) monitorServices.CacheProvider {
	adapter := &CacheProviderAdapter{cache: core.Cache}

	// 尝试获取统计功能支持
	type StatsCacheInterface interface {
		GetStats(ctx context.Context) (map[string]interface{}, error)
	}
	if statsCache, ok := core.Cache.(StatsCacheInterface); ok {
		adapter.statsCache = statsCache
	}

	// 尝试获取多级缓存支持
	type MultiLevelCacheInterface interface {
		KeysByLevel(ctx context.Context, pattern string, level string) ([]string, error)
	}
	if mlc, ok := core.Cache.(MultiLevelCacheInterface); ok {
		adapter.multiLevelCache = mlc
	}

	// 尝试获取直接 Redis 访问支持
	type DirectRedisInterface interface {
		DirectRedisKeys(ctx context.Context, pattern string) ([]string, error)
		DirectRedisGet(ctx context.Context, key string) (string, error)
		DirectRedisTTL(ctx context.Context, key string) (time.Duration, error)
	}
	if dr, ok := core.Cache.(DirectRedisInterface); ok {
		adapter.directRedis = dr
	}

	return adapter
}

func (a *CacheProviderAdapter) Get(ctx context.Context, key string) (string, error) {
	return a.cache.Get(ctx, key)
}

func (a *CacheProviderAdapter) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	return a.cache.Set(ctx, key, value, ttl)
}

func (a *CacheProviderAdapter) Delete(ctx context.Context, key string) error {
	return a.cache.Delete(ctx, key)
}

func (a *CacheProviderAdapter) Exists(ctx context.Context, key string) (bool, error) {
	return a.cache.Exists(ctx, key)
}

func (a *CacheProviderAdapter) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return a.cache.Expire(ctx, key, ttl)
}

func (a *CacheProviderAdapter) TTL(ctx context.Context, key string) (time.Duration, error) {
	return a.cache.TTL(ctx, key)
}

func (a *CacheProviderAdapter) Keys(ctx context.Context, pattern string) ([]string, error) {
	return a.cache.Keys(ctx, pattern)
}

func (a *CacheProviderAdapter) FlushDB(ctx context.Context) error {
	return a.cache.FlushDB(ctx)
}

// GetStats 获取缓存统计信息
// 实现 StatsProvider 接口
func (a *CacheProviderAdapter) GetStats(ctx context.Context) (map[string]interface{}, error) {
	if a.statsCache == nil {
		return nil, monitorServices.ErrCacheStatsUnsupported
	}
	return a.statsCache.GetStats(ctx)
}

// KeysByLevel 按层级获取缓存键
// 实现 MultiLevelCacheProvider 接口
func (a *CacheProviderAdapter) KeysByLevel(ctx context.Context, pattern string, level string) ([]string, error) {
	if a.multiLevelCache == nil {
		return []string{}, nil
	}
	return a.multiLevelCache.KeysByLevel(ctx, pattern, level)
}

// DirectRedisKeys 直接访问 Redis 获取键列表
// 实现 DirectRedisProvider 接口
func (a *CacheProviderAdapter) DirectRedisKeys(ctx context.Context, pattern string) ([]string, error) {
	if a.directRedis == nil {
		return []string{}, nil
	}
	return a.directRedis.DirectRedisKeys(ctx, pattern)
}

// DirectRedisGet 直接访问 Redis 获取值
// 实现 DirectRedisProvider 接口
func (a *CacheProviderAdapter) DirectRedisGet(ctx context.Context, key string) (string, error) {
	if a.directRedis == nil {
		return "", nil
	}
	return a.directRedis.DirectRedisGet(ctx, key)
}

// DirectRedisTTL 直接访问 Redis 获取 TTL
// 实现 DirectRedisProvider 接口
func (a *CacheProviderAdapter) DirectRedisTTL(ctx context.Context, key string) (time.Duration, error) {
	if a.directRedis == nil {
		return 0, nil
	}
	return a.directRedis.DirectRedisTTL(ctx, key)
}

// MultiLevelCacheProviderAdapter 多级缓存适配器（保留用于兼容性）
type MultiLevelCacheProviderAdapter struct {
	cache interface {
		KeysByLevel(ctx context.Context, pattern string, level string) ([]string, error)
	}
}

// NewMultiLevelCacheProviderAdapter 创建多级缓存适配器
func NewMultiLevelCacheProviderAdapter(core *core.Core) monitorServices.MultiLevelCacheProvider {
	type MultiLevelCacheInterface interface {
		KeysByLevel(ctx context.Context, pattern string, level string) ([]string, error)
	}
	if cache, ok := core.Cache.(MultiLevelCacheInterface); ok {
		return &MultiLevelCacheProviderAdapter{cache: cache}
	}
	return nil
}

func (a *MultiLevelCacheProviderAdapter) KeysByLevel(ctx context.Context, pattern string, level string) ([]string, error) {
	if a == nil || a.cache == nil {
		return []string{}, nil
	}
	return a.cache.KeysByLevel(ctx, pattern, level)
}

// DirectRedisProviderAdapter 直接 Redis 访问适配器
type DirectRedisProviderAdapter struct {
	cache interface {
		DirectRedisKeys(ctx context.Context, pattern string) ([]string, error)
		DirectRedisGet(ctx context.Context, key string) (string, error)
		DirectRedisTTL(ctx context.Context, key string) (time.Duration, error)
	}
}

// NewDirectRedisProviderAdapter 创建直接 Redis 访问适配器
func NewDirectRedisProviderAdapter(core *core.Core) monitorServices.DirectRedisProvider {
	type DirectRedisInterface interface {
		DirectRedisKeys(ctx context.Context, pattern string) ([]string, error)
		DirectRedisGet(ctx context.Context, key string) (string, error)
		DirectRedisTTL(ctx context.Context, key string) (time.Duration, error)
	}
	if cache, ok := core.Cache.(DirectRedisInterface); ok {
		return &DirectRedisProviderAdapter{cache: cache}
	}
	return nil
}

func (a *DirectRedisProviderAdapter) DirectRedisKeys(ctx context.Context, pattern string) ([]string, error) {
	if a == nil || a.cache == nil {
		return []string{}, nil
	}
	return a.cache.DirectRedisKeys(ctx, pattern)
}

func (a *DirectRedisProviderAdapter) DirectRedisGet(ctx context.Context, key string) (string, error) {
	if a == nil || a.cache == nil {
		return "", nil
	}
	return a.cache.DirectRedisGet(ctx, key)
}

func (a *DirectRedisProviderAdapter) DirectRedisTTL(ctx context.Context, key string) (time.Duration, error) {
	if a == nil || a.cache == nil {
		return 0, nil
	}
	return a.cache.DirectRedisTTL(ctx, key)
}

// StatsProviderAdapter 统计信息适配器
type StatsProviderAdapter struct {
	cache interface {
		GetStats(ctx context.Context) (map[string]interface{}, error)
	}
}

// NewStatsProviderAdapter 创建统计信息适配器
func NewStatsProviderAdapter(core *core.Core) monitorServices.StatsProvider {
	type StatsInterface interface {
		GetStats(ctx context.Context) (map[string]interface{}, error)
	}
	if cache, ok := core.Cache.(StatsInterface); ok {
		return &StatsProviderAdapter{cache: cache}
	}
	return nil
}

func (a *StatsProviderAdapter) GetStats(ctx context.Context) (map[string]interface{}, error) {
	if a == nil || a.cache == nil {
		return map[string]interface{}{}, nil
	}
	return a.cache.GetStats(ctx)
}

// CacheConfigProviderAdapter 缓存配置适配器
type CacheConfigProviderAdapter struct {
	service *services.CacheConfigService
}

// NewCacheConfigProviderAdapter 创建缓存配置适配器
func NewCacheConfigProviderAdapter(service *services.CacheConfigService) monitorServices.CacheConfigProvider {
	return &CacheConfigProviderAdapter{service: service}
}

func (a *CacheConfigProviderAdapter) GetConfigInfo() map[string]monitorServices.CacheConfigInfo {
	configInfoMap := a.service.GetConfigInfo()
	result := make(map[string]monitorServices.CacheConfigInfo)
	for k, v := range configInfoMap {
		result[k] = monitorServices.CacheConfigInfo{
			Key:         v.Key,
			Name:        v.Name,
			Description: v.Description,
			Category:    v.Category,
			Min:         v.Min,
			Max:         v.Max,
			Default:     v.Default,
		}
	}
	return result
}

func (a *CacheConfigProviderAdapter) GetAllConfigs(ctx context.Context) map[string]int {
	return a.service.GetAllConfigs(ctx)
}

func (a *CacheConfigProviderAdapter) ReloadConfig(ctx context.Context) error {
	return a.service.ReloadConfig(ctx)
}

// ==================== 路由设置 ====================

// SetupCacheRouter 设置缓存监控路由
func SetupCacheRouter(r *gin.RouterGroup, core *core.Core) {
	// 复用 core 已有的 CacheConfigService(在 core 初始化阶段已 LoadConfigs)。
	// 之前这里 new 一个新实例会重复打印 37 项 cache.* 配置 + 触发一次 DB 扫描,
	// 是启动日志的"通知中心已启动"后第二段 [CACHE_CONFIG] 日志的根因。
	cacheConfigService := core.CacheConfigService
	if cacheConfigService == nil {
		// 兜底:core.CacheConfigService 仅在 c.Cache != nil 时初始化(参见 core.go:224-226)。
		// 若禁用缓存启动(CacheConfigService 为 nil),按需 new 一个本地实例供本路由使用。
		cacheConfigService = services.NewCacheConfigService(core.DB.GetDB())
	}

	// 创建适配器
	cacheProvider := NewCacheProviderAdapter(core)
	configProvider := NewCacheConfigProviderAdapter(cacheConfigService)

	// 创建 CacheService
	cacheService := monitorServices.NewCacheService(
		core.DB.GetDB(),
		cacheProvider,
		configProvider,
	)

	// 创建Handler
	cacheHandler := NewCacheHandler(cacheService, core)

	// ==================== 缓存信息路由 ====================
	r.POST("/cache/list", cacheHandler.GetCacheList)
	r.GET("/cache/:key", cacheHandler.GetCacheInfo)

	// ==================== 缓存操作路由 ====================
	r.POST("/cache/operate", cacheHandler.OperateCache)
	r.POST("/cache/batch", cacheHandler.BatchOperateCache)
	r.POST("/cache/clear", cacheHandler.ClearCache)

	// ==================== 缓存统计路由 ====================
	r.POST("/cache/stats/list", cacheHandler.GetCacheStats)
	r.POST("/cache/monitor", cacheHandler.GetCacheMonitor)

	// ==================== 缓存导出路由 ====================
	r.POST("/cache/export", cacheHandler.ExportCache)

	// ==================== 缓存配置管理路由 ====================
	r.GET("/cache/config", cacheHandler.GetCacheConfigs)
	r.PUT("/cache/config", cacheHandler.UpdateCacheConfig)
	r.POST("/cache/config/reload", cacheHandler.ReloadCacheConfigs)

	// ==================== 测试端点路由 ====================
	r.GET("/cache/test", cacheHandler.TestCacheEndpoint)
	r.GET("/cache/debug/raw-keys", cacheHandler.DebugRawKeys)
	r.GET("/cache/debug/l1", cacheHandler.DebugL1Cache)

	// ==================== 增强缓存管理路由 ====================
	enhancedCacheHandler := NewCacheEnhancedHandler(core)
	r.POST("/cache/stats", enhancedCacheHandler.GetCacheStats)                    // 获取缓存统计信息
	r.POST("/cache/invalidate", enhancedCacheHandler.InvalidateByModule)          // 按模块清除缓存
	r.POST("/cache/invalidate-pattern", enhancedCacheHandler.InvalidateByPattern) // 按模式清除缓存
	r.POST("/cache/warmup", enhancedCacheHandler.WarmUpCache)                     // 执行缓存预热
	r.POST("/cache/key-info", enhancedCacheHandler.GetKeyInfo)                    // 获取缓存键信息
}
