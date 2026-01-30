package system

import (
	"context"
	"time"

	sysServices "github.com/ruoyi-next/ruoyi-go-backend/internal/services/system"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services"
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
