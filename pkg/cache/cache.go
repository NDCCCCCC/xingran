package cache

import (
	"context"
	"time"
)

// Cache 缓存接口
type Cache interface {
	// 基础操作
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)

	// 批量操作
	MGet(ctx context.Context, keys ...string) ([]string, error)
	MSet(ctx context.Context, pairs ...interface{}) error
	MDelete(ctx context.Context, keys ...string) error

	// 高级操作
	Increment(ctx context.Context, key string) (int64, error)
	IncrementBy(ctx context.Context, key string, value int64) (int64, error)
	Decrement(ctx context.Context, key string) (int64, error)
	DecrementBy(ctx context.Context, key string, value int64) (int64, error)

	// 过期操作
	Expire(ctx context.Context, key string, expiration time.Duration) error
	TTL(ctx context.Context, key string) (time.Duration, error)

	// 模式匹配
	Keys(ctx context.Context, pattern string) ([]string, error)
	FlushDB(ctx context.Context) error

	// 关闭连接
	Close() error

	// JSON操作扩展
	MGetJSON(ctx context.Context, keys ...string) (map[string]interface{}, error)
	MSetJSON(ctx context.Context, data map[string]interface{}, expiration time.Duration) error

	// Hash操作扩展（用于缓存元数据管理）
	HGet(ctx context.Context, key, field string) (string, error)
	HSet(ctx context.Context, key, field string, value interface{}) error
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	HDel(ctx context.Context, key string, fields ...string) error
	HKeys(ctx context.Context, key string) ([]string, error)

	// 类型扩展方法（用于验证码服务）
	SetInt(ctx context.Context, key string, value int, expiration time.Duration) error
	GetInt(ctx context.Context, key string) (int, error)
	SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	GetJSON(ctx context.Context, key string, dest interface{}) error
}

// CacheStrategy 缓存策略
type CacheStrategy interface {
	GetKey(params ...interface{}) string
	GetExpiration() time.Duration
	ShouldRefresh(data interface{}) bool
}

// RateLimitCache 提供原子自增并设置 TTL 的限流能力（由 RedisCache 实现）。
// 限流场景（如 captcha）通过类型断言到该具名接口，避免隐式匿名接口在实现改名后静默走 fallback。
type RateLimitCache interface {
	IncrementWithExpire(ctx context.Context, key string, ttl time.Duration) (int64, error)
}

// 编译期保证 *RedisCache 实现 RateLimitCache：方法改名/签名变更将立即编译失败而非静默降级。
var _ RateLimitCache = (*RedisCache)(nil)

// L2ExposingCache 暴露底层 L2（Redis）缓存，用于绕过异步 L2Writer 直接读写（由 MultiLevelCache 实现）。
type L2ExposingCache interface {
	GetL2Cache() Cache
}

// 编译期保证 *MultiLevelCache 实现 L2ExposingCache。
var _ L2ExposingCache = (*MultiLevelCache)(nil)

// CacheConfig 缓存配置
type CacheConfig struct {
	Type     string `yaml:"type"` // redis, memory
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`

	// 内存缓存配置
	MaxSize     int           `yaml:"max_size"`
	CleanupTime time.Duration `yaml:"cleanup_time"`

	// L2写入Worker配置
	L2Writer *L2WriterConfig `yaml:"l2_writer"`
}

// CacheItem 缓存项
type CacheItem struct {
	Value      interface{}
	Expiration int64
	Created    time.Time
}

// IsExpired 检查是否过期
func (item *CacheItem) IsExpired() bool {
	if item.Expiration == 0 {
		return false // 永不过期
	}
	return time.Now().UnixNano() > item.Expiration
}
