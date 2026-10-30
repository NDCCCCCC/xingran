package system

import (
	"context"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/services"
	sysServices "github.com/xingran-next/xingran-go-backend/internal/services/system"
)

// dataCacheAdapter DataCacheService 适配器
type dataCacheAdapter struct {
	dataCache *services.DataCacheService
}

// NewDataCacheAdapter 创建 DataCacheService 适配器
func NewDataCacheAdapter(dataCache *services.DataCacheService) sysServices.CacheProvider {
	return &dataCacheAdapter{dataCache: dataCache}
}

// GetOrSet 实现 CacheProvider 接口
func (a *dataCacheAdapter) GetOrSet(
	ctx context.Context,
	key string,
	dest interface{},
	expiration time.Duration,
	query func() (interface{}, error),
) error {
	return a.dataCache.GetOrSet(ctx, key, dest, expiration, query)
}

// Delete 实现 CacheProvider 接口
func (a *dataCacheAdapter) Delete(ctx context.Context, key string) error {
	return a.dataCache.Delete(ctx, key)
}

// DeleteByPattern 实现 CacheProvider 接口
func (a *dataCacheAdapter) DeleteByPattern(ctx context.Context, pattern string) error {
	return a.dataCache.DeleteByPattern(ctx, pattern)
}

// MGet 实现 CacheProvider 接口
func (a *dataCacheAdapter) MGet(ctx context.Context, keys ...string) (map[string]string, error) {
	return a.dataCache.MGet(ctx, keys...)
}

// MDelete 实现 CacheProvider 接口
func (a *dataCacheAdapter) MDelete(ctx context.Context, keys ...string) error {
	return a.dataCache.MDelete(ctx, keys...)
}

// Exists 实现 CacheProvider 接口
func (a *dataCacheAdapter) Exists(ctx context.Context, key string) (bool, error) {
	return a.dataCache.Exists(ctx, key)
}

// SetTTL 实现 CacheProvider 接口
func (a *dataCacheAdapter) SetTTL(ctx context.Context, key string, expiration time.Duration) error {
	return a.dataCache.SetTTL(ctx, key, expiration)
}

// GetTTL 实现 CacheProvider 接口
func (a *dataCacheAdapter) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return a.dataCache.GetTTL(ctx, key)
}

// GetStats 实现 CacheProvider 接口
func (a *dataCacheAdapter) GetStats(ctx context.Context) (*sysServices.CacheStats, error) {
	stats, err := a.dataCache.GetStats(ctx)
	if err != nil {
		return nil, err
	}
	return &sysServices.CacheStats{
		Hits:          stats.Hits,
		Misses:        stats.Misses,
		Count:         stats.Count,
		MemorySize:    stats.MemorySize,
		HitRate:       stats.HitRate,
		KeyCount:      stats.KeyCount,
		ExtendedStats: stats.ExtendedStats,
	}, nil
}
