package system

import (
	"context"
	"time"
)

// CacheProvider 缓存提供者接口
// 由 system 模块外部实现，用于解耦缓存逻辑
type CacheProvider interface {
	// GetOrSet 获取缓存，如果不存在则执行查询函数并缓存结果
	GetOrSet(
		ctx context.Context,
		key string,
		dest interface{},
		expiration time.Duration,
		query func() (interface{}, error),
	) error

	// Delete 删除缓存
	Delete(ctx context.Context, key string) error

	// DeleteByPattern 根据模式删除缓存
	DeleteByPattern(ctx context.Context, pattern string) error
}

// NoOpCacheProvider 空缓存提供者（用于无缓存场景）
type NoOpCacheProvider struct{}

func (n *NoOpCacheProvider) GetOrSet(ctx context.Context, key string, dest interface{},
	expiration time.Duration, query func() (interface{}, error)) error {
	// 直接执行查询，不使用缓存
	result, err := query()
	if err != nil {
		return err
	}
	// 将结果设置到目标变量（使用辅助函数）
	if ptr, ok := dest.(*interface{}); ok {
		*ptr = result
	}
	return nil
}

func (n *NoOpCacheProvider) Delete(ctx context.Context, key string) error                { return nil }
func (n *NoOpCacheProvider) DeleteByPattern(ctx context.Context, pattern string) error { return nil }
