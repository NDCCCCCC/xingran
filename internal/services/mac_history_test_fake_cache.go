package services

import (
	"context"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/services/base"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// -------------------------------------------------------------------------
// Phase 103 CONV-01 (D-103-19) 测试 fixture：fakeMACHistoryCacheProvider
//
// B8 import-cycle 约束：package services 测试文件不能 import
// internal/services/system（成环 services→system→services），因此无法在
// 测试内调用 system.NewCacheProvider。本 fixture 在 services 包内实现
// base.CacheProvider 接口，内部持有 *DataCacheService 提供真实缓存行为
// （JSON 序列化 + L1/L2 写路径），供 mac_history 域测试走真实 cache
// hit/miss 断言。生产路径不受影响（router 层走 system.NewCacheProvider）。
//
// 支持注入 Get/Set 错误，用于错误透传（D-103-6）与 fallback 直查
// （D-103-9）语义的回归测试。
// -------------------------------------------------------------------------

// fakeMACHistoryCacheProvider 实现 base.CacheProvider，内部委托 DataCacheService。
type fakeMACHistoryCacheProvider struct {
	dataCache *DataCacheService
	getOrSetErr error // GetOrSet 整体注错；nil=正常
	deleteErr   error // Delete 注错；nil=正常
}

var _ base.CacheProvider = (*fakeMACHistoryCacheProvider)(nil) // 编译期断言

// newFakeMACHistoryCacheProvider 基于内存缓存构造真实行为的 CacheProvider fixture。
func newFakeMACHistoryCacheProvider(mem cache.Cache) *fakeMACHistoryCacheProvider {
	return &fakeMACHistoryCacheProvider{dataCache: NewDataCacheService(mem)}
}

func (f *fakeMACHistoryCacheProvider) GetOrSet(ctx context.Context, key string, dest interface{}, expiration time.Duration, query func() (interface{}, error)) error {
	if f.getOrSetErr != nil {
		return f.getOrSetErr
	}
	return f.dataCache.GetOrSet(ctx, key, dest, expiration, query)
}

func (f *fakeMACHistoryCacheProvider) Delete(ctx context.Context, key string) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	return f.dataCache.Delete(ctx, key)
}

func (f *fakeMACHistoryCacheProvider) DeleteByPattern(ctx context.Context, pattern string) error {
	return f.dataCache.DeleteByPattern(ctx, pattern)
}

func (f *fakeMACHistoryCacheProvider) MGet(ctx context.Context, keys ...string) (map[string]string, error) {
	return f.dataCache.MGet(ctx, keys...)
}

func (f *fakeMACHistoryCacheProvider) MDelete(ctx context.Context, keys ...string) error {
	return f.dataCache.MDelete(ctx, keys...)
}

func (f *fakeMACHistoryCacheProvider) Exists(ctx context.Context, key string) (bool, error) {
	return f.dataCache.Exists(ctx, key)
}

func (f *fakeMACHistoryCacheProvider) SetTTL(ctx context.Context, key string, expiration time.Duration) error {
	return f.dataCache.SetTTL(ctx, key, expiration)
}

func (f *fakeMACHistoryCacheProvider) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return f.dataCache.GetTTL(ctx, key)
}

func (f *fakeMACHistoryCacheProvider) GetStats(ctx context.Context) (*base.CacheStats, error) {
	stats, err := f.dataCache.GetStats(ctx) // *services.CacheStats
	if err != nil {
		return nil, err
	}
	return &base.CacheStats{
		Hits:          stats.Hits,
		Misses:        stats.Misses,
		Count:         stats.Count,
		MemorySize:    stats.MemorySize,
		HitRate:       stats.HitRate,
		KeyCount:      stats.KeyCount,
		ExtendedStats: stats.ExtendedStats,
	}, nil
}
