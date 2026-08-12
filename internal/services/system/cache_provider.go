package system

import (
	"context"
	"reflect"
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

	// MGet 批量获取缓存
	MGet(ctx context.Context, keys ...string) (map[string]string, error)

	// MDelete 批量删除缓存
	MDelete(ctx context.Context, keys ...string) error

	// Exists 检查缓存是否存在
	Exists(ctx context.Context, key string) (bool, error)

	// SetTTL 设置缓存过期时间
	SetTTL(ctx context.Context, key string, expiration time.Duration) error

	// GetTTL 获取缓存过期时间
	GetTTL(ctx context.Context, key string) (time.Duration, error)

	// GetStats 获取缓存统计信息
	GetStats(ctx context.Context) (*CacheStats, error)
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
	// 将结果设置到目标变量（使用反射）
	setValue(dest, result)
	return nil
}

// setValue 使用反射设置目标变量的值
func setValue(dest interface{}, value interface{}) {
	if dest == nil {
		return
	}

	// 使用反射来设置值
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return
	}

	// 获取指向的值
	elem := destValue.Elem()

	// 设置新值
	val := reflect.ValueOf(value)
	if elem.IsValid() && val.IsValid() && val.Type().AssignableTo(elem.Type()) {
		elem.Set(val)
	}
}

// CacheStats 缓存统计信息
type CacheStats struct {
	Hits          int64                  // 命中次数
	Misses        int64                  // 未命中次数
	Count         int64                  // 缓存项数量
	MemorySize    int64                  // 内存占用（字节）
	HitRate       float64                // 命中率
	KeyCount      int                    // 键数量
	ExtendedStats map[string]interface{} // 扩展统计（L2Writer、RetryWorker等）
}

// CacheEntry 缓存条目详情
type CacheEntry struct {
	Key       string        // 键
	Value     string        // 值
	TTL       time.Duration // 剩余存活时间
	Size      int64         // 大小（字节）
	CreatedAt time.Time     // 创建时间
}

func (n *NoOpCacheProvider) MGet(ctx context.Context, keys ...string) (map[string]string, error) {
	return make(map[string]string), nil
}

func (n *NoOpCacheProvider) MDelete(ctx context.Context, keys ...string) error {
	return nil
}

func (n *NoOpCacheProvider) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}

func (n *NoOpCacheProvider) SetTTL(ctx context.Context, key string, expiration time.Duration) error {
	return nil
}

func (n *NoOpCacheProvider) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, nil
}

func (n *NoOpCacheProvider) GetStats(ctx context.Context) (*CacheStats, error) {
	return &CacheStats{}, nil
}

func (n *NoOpCacheProvider) Delete(ctx context.Context, key string) error              { return nil }
func (n *NoOpCacheProvider) DeleteByPattern(ctx context.Context, pattern string) error { return nil }
