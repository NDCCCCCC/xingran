package system

import (
	"context"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/services"
)

// NewCacheProvider 创建缓存提供者
// 根据是否提供 DataCacheService 返回相应的实现
func NewCacheProvider(dataCache *services.DataCacheService) CacheProvider {
	if dataCache != nil {
		return &cacheProviderAdapter{dataCache: dataCache}
	}
	return &NoOpCacheProvider{}
}

// cacheProviderAdapter 缓存提供者适配器
type cacheProviderAdapter struct {
	dataCache *services.DataCacheService
}

// GetOrSet 实现 CacheProvider 接口
func (a *cacheProviderAdapter) GetOrSet(
	ctx context.Context,
	key string,
	dest interface{},
	expiration time.Duration,
	query func() (interface{}, error),
) error {
	return a.dataCache.GetOrSet(ctx, key, dest, expiration, query)
}

// Delete 实现 CacheProvider 接口
func (a *cacheProviderAdapter) Delete(ctx context.Context, key string) error {
	return a.dataCache.Delete(ctx, key)
}

// DeleteByPattern 实现 CacheProvider 接口
func (a *cacheProviderAdapter) DeleteByPattern(ctx context.Context, pattern string) error {
	return a.dataCache.DeleteByPattern(ctx, pattern)
}

// MGet 实现 CacheProvider 接口
func (a *cacheProviderAdapter) MGet(ctx context.Context, keys ...string) (map[string]string, error) {
	return a.dataCache.MGet(ctx, keys...)
}

// MDelete 实现 CacheProvider 接口
func (a *cacheProviderAdapter) MDelete(ctx context.Context, keys ...string) error {
	return a.dataCache.MDelete(ctx, keys...)
}

// Exists 实现 CacheProvider 接口
func (a *cacheProviderAdapter) Exists(ctx context.Context, key string) (bool, error) {
	return a.dataCache.Exists(ctx, key)
}

// SetTTL 实现 CacheProvider 接口
func (a *cacheProviderAdapter) SetTTL(ctx context.Context, key string, expiration time.Duration) error {
	return a.dataCache.SetTTL(ctx, key, expiration)
}

// GetTTL 实现 CacheProvider 接口
func (a *cacheProviderAdapter) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return a.dataCache.GetTTL(ctx, key)
}

// GetStats 实现 CacheProvider 接口
func (a *cacheProviderAdapter) GetStats(ctx context.Context) (*CacheStats, error) {
	stats, err := a.dataCache.GetStats(ctx)
	if err != nil {
		return nil, err
	}
	return &CacheStats{
		Hits:          stats.Hits,
		Misses:        stats.Misses,
		Count:         stats.Count,
		MemorySize:    stats.MemorySize,
		HitRate:       stats.HitRate,
		KeyCount:      stats.KeyCount,
		ExtendedStats: stats.ExtendedStats,
	}, nil
}
