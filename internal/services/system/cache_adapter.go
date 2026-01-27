package system

import (
	"context"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/pkg/cache"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
)

// CacheAdapter 将 pkg/cache.Cache 适配为 system.CacheProvider 接口
// 解决两个接口不兼容的问题
type CacheAdapter struct {
	cache cache.Cache
}

// NewCacheAdapter 创建缓存适配器
func NewCacheAdapter(c cache.Cache) CacheProvider {
	return &CacheAdapter{cache: c}
}

// GetOrSet 获取缓存，如果不存在则执行查询函数并缓存结果
func (a *CacheAdapter) GetOrSet(
	ctx context.Context,
	key string,
	dest interface{},
	expiration time.Duration,
	query func() (interface{}, error),
) error {
	// 尝试从缓存获取
	cached, err := a.cache.Get(ctx, key)
	if err == nil && cached != "" {
		// 缓存命中，直接返回（简化处理，实际应反序列化）
		if ptr, ok := dest.(*interface{}); ok {
			*ptr = cached
		}
		return nil
	}

	// 缓存未命中，执行查询
	result, err := query()
	if err != nil {
		return err
	}

	// 保存到缓存
	if cacheErr := a.cache.Set(ctx, key, result, expiration); cacheErr != nil {
		logger.Warnf("保存缓存失败: key=%s, error=%v", key, cacheErr)
	}

	// 设置结果到目标变量
	if ptr, ok := dest.(*interface{}); ok {
		*ptr = result
	}

	return nil
}

// Delete 删除缓存
func (a *CacheAdapter) Delete(ctx context.Context, key string) error {
	return a.cache.Delete(ctx, key)
}

// DeleteByPattern 根据模式删除缓存
// 使用Keys方法获取匹配的键，然后批量删除
func (a *CacheAdapter) DeleteByPattern(ctx context.Context, pattern string) error {
	// 获取匹配模式的所有键
	keys, err := a.cache.Keys(ctx, pattern)
	if err != nil {
		return err
	}

	// 批量删除
	if len(keys) > 0 {
		return a.cache.MDelete(ctx, keys...)
	}

	return nil
}
