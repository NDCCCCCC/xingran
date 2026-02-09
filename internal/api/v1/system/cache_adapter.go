package system

import (
	"context"
	"time"

	sysServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/cache"
)

// dataCacheAdapter DataCacheService 适配器
type dataCacheAdapter struct {
	dataCache *services.DataCacheService
	cache     cache.Cache
}

// NewDataCacheAdapter 创建 DataCacheService 适配器
func NewDataCacheAdapter(dataCache *services.DataCacheService, c cache.Cache) sysServices.CacheProvider {
	return &dataCacheAdapter{dataCache: dataCache, cache: c}
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
	if a.cache != nil {
		values, err := a.cache.MGet(ctx, keys...)
		if err != nil {
			return nil, err
		}
		result := make(map[string]string, len(keys))
		for i, key := range keys {
			if i < len(values) && values[i] != "" {
				result[key] = values[i]
			}
		}
		return result, nil
	}
	return make(map[string]string), nil
}

// MDelete 实现 CacheProvider 接口
func (a *dataCacheAdapter) MDelete(ctx context.Context, keys ...string) error {
	if a.cache != nil {
		return a.cache.MDelete(ctx, keys...)
	}
	return nil
}

// Exists 实现 CacheProvider 接口
func (a *dataCacheAdapter) Exists(ctx context.Context, key string) (bool, error) {
	if a.cache != nil {
		return a.cache.Exists(ctx, key)
	}
	return false, nil
}

// SetTTL 实现 CacheProvider 接口
func (a *dataCacheAdapter) SetTTL(ctx context.Context, key string, expiration time.Duration) error {
	if a.cache != nil {
		return a.cache.Expire(ctx, key, expiration)
	}
	return nil
}

// GetTTL 实现 CacheProvider 接口
func (a *dataCacheAdapter) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	if a.cache != nil {
		return a.cache.TTL(ctx, key)
	}
	return 0, nil
}

// GetStats 实现 CacheProvider 接口
func (a *dataCacheAdapter) GetStats(ctx context.Context) (*sysServices.CacheStats, error) {
	// 返回基本统计信息
	stats := &sysServices.CacheStats{}
	if a.cache != nil {
		if keys, err := a.cache.Keys(ctx, "*"); err == nil {
			stats.KeyCount = len(keys)
			stats.Count = int64(len(keys))
		}
	}
	return stats, nil
}
