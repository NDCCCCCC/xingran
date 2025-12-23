package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache Redis缓存实现
type RedisCache struct {
	client *redis.Client
	prefix string
}

// NewRedisCache 创建Redis缓存实例
func NewRedisCache(config *CacheConfig, keyPrefix string) (*RedisCache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
		DB:       config.DB,
		PoolSize: config.PoolSize,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis连接失败: %w", err)
	}

	return &RedisCache{
		client: rdb,
		prefix: keyPrefix,
	}, nil
}

// buildKey 构建缓存键
func (r *RedisCache) buildKey(key string) string {
	if r.prefix == "" {
		return key
	}
	return r.prefix + ":" + key
}

// Get 获取缓存值
func (r *RedisCache) Get(ctx context.Context, key string) (string, error) {
	result, err := r.client.Get(ctx, r.buildKey(key)).Result()
	if err == redis.Nil {
		return "", ErrNotFound
	}
	return result, err
}

// Set 设置缓存值
func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(ctx, r.buildKey(key), value, expiration).Err()
}

// Delete 删除缓存
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, r.buildKey(key)).Err()
}

// Exists 检查键是否存在
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := r.client.Exists(ctx, r.buildKey(key)).Result()
	return result > 0, err
}

// MGet 批量获取
func (r *RedisCache) MGet(ctx context.Context, keys ...string) ([]string, error) {
	if len(keys) == 0 {
		return []string{}, nil
	}

	redisKeys := make([]string, len(keys))
	for i, key := range keys {
		redisKeys[i] = r.buildKey(key)
	}

	result, err := r.client.MGet(ctx, redisKeys...).Result()
	if err != nil {
		return nil, err
	}

	values := make([]string, len(result))
	for i, value := range result {
		if value == nil {
			values[i] = ""
		} else {
			values[i] = value.(string)
		}
	}

	return values, nil
}

// MSet 批量设置
func (r *RedisCache) MSet(ctx context.Context, pairs ...interface{}) error {
	if len(pairs)%2 != 0 {
		return fmt.Errorf("参数数量必须为偶数")
	}

	if len(pairs) == 0 {
		return nil
	}

	redisPairs := make([]interface{}, len(pairs))
	for i := 0; i < len(pairs); i += 2 {
		key, ok := pairs[i].(string)
		if !ok {
			return fmt.Errorf("键必须是字符串类型")
		}
		redisPairs[i] = r.buildKey(key)
		redisPairs[i+1] = pairs[i+1]
	}

	return r.client.MSet(ctx, redisPairs...).Err()
}

// MDelete 批量删除
func (r *RedisCache) MDelete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	redisKeys := make([]string, len(keys))
	for i, key := range keys {
		redisKeys[i] = r.buildKey(key)
	}

	return r.client.Del(ctx, redisKeys...).Err()
}

// Increment 递增
func (r *RedisCache) Increment(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, r.buildKey(key)).Result()
}

// IncrementBy 按值递增
func (r *RedisCache) IncrementBy(ctx context.Context, key string, value int64) (int64, error) {
	return r.client.IncrBy(ctx, r.buildKey(key), value).Result()
}

// Decrement 递减
func (r *RedisCache) Decrement(ctx context.Context, key string) (int64, error) {
	return r.client.Decr(ctx, r.buildKey(key)).Result()
}

// DecrementBy 按值递减
func (r *RedisCache) DecrementBy(ctx context.Context, key string, value int64) (int64, error) {
	return r.client.DecrBy(ctx, r.buildKey(key), value).Result()
}

// Expire 设置过期时间
func (r *RedisCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, r.buildKey(key), expiration).Err()
}

// TTL 获取剩余时间
func (r *RedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	duration, err := r.client.TTL(ctx, r.buildKey(key)).Result()
	if err != nil {
		return 0, err
	}
	return time.Duration(duration) * time.Second, nil
}

// Keys 模式匹配
func (r *RedisCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	keys, err := r.client.Keys(ctx, r.buildKey(pattern)).Result()
	if err != nil {
		return nil, err
	}

	// 移除前缀
	if r.prefix != "" {
		prefixLen := len(r.prefix) + 1
		for i, key := range keys {
			if len(key) > prefixLen {
				keys[i] = key[prefixLen:]
			}
		}
	}

	return keys, nil
}

// FlushDB 清空数据库
func (r *RedisCache) FlushDB(ctx context.Context) error {
	return r.client.FlushDB(ctx).Err()
}

// Close 关闭连接
func (r *RedisCache) Close() error {
	return r.client.Close()
}

// GetJSON 获取JSON对象
func (r *RedisCache) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := r.Get(ctx, key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

// SetJSON 设置JSON对象
func (r *RedisCache) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.Set(ctx, key, data, expiration)
}

// GetInt 获取整数
func (r *RedisCache) GetInt(ctx context.Context, key string) (int, error) {
	data, err := r.Get(ctx, key)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(data)
}

// SetInt 设置整数
func (r *RedisCache) SetInt(ctx context.Context, key string, value int, expiration time.Duration) error {
	return r.Set(ctx, key, strconv.Itoa(value), expiration)
}

// GetBool 获取布尔值
func (r *RedisCache) GetBool(ctx context.Context, key string) (bool, error) {
	data, err := r.Get(ctx, key)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(data)
}

// SetBool 设置布尔值
func (r *RedisCache) SetBool(ctx context.Context, key string, value bool, expiration time.Duration) error {
	var strValue string
	if value {
		strValue = "1"
	} else {
		strValue = "0"
	}
	return r.Set(ctx, key, strValue, expiration)
}

// MGetJSON 批量获取JSON对象
func (r *RedisCache) MGetJSON(ctx context.Context, keys ...string) (map[string]interface{}, error) {
	if len(keys) == 0 {
		return make(map[string]interface{}), nil
	}

	redisKeys := make([]string, len(keys))
	for i, key := range keys {
		redisKeys[i] = r.buildKey(key)
	}

	values, err := r.client.MGet(ctx, redisKeys...).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	for i, key := range keys {
		if values[i] != nil {
			var value interface{}
			if strValue, ok := values[i].(string); ok {
				if err := json.Unmarshal([]byte(strValue), &value); err == nil {
					result[key] = value
				} else {
					result[key] = strValue
				}
			} else {
				result[key] = values[i]
			}
		}
	}

	return result, nil
}

// MSetJSON 批量设置JSON对象
func (r *RedisCache) MSetJSON(ctx context.Context, data map[string]interface{}, expiration time.Duration) error {
	if len(data) == 0 {
		return nil
	}

	// 先设置所有数据
	pairs := make([]interface{}, len(data)*2)
	i := 0
	for key, value := range data {
		pairs[i] = r.buildKey(key)
		pairs[i+1] = value
		i += 2
	}

	if err := r.client.MSet(ctx, pairs...).Err(); err != nil {
		return err
	}

	// 如果需要设置过期时间，为每个键单独设置
	if expiration > 0 {
		for key := range data {
			if err := r.client.Expire(ctx, r.buildKey(key), expiration).Err(); err != nil {
				return err
			}
		}
	}

	return nil
}

// HGet 获取Hash字段
func (r *RedisCache) HGet(ctx context.Context, key, field string) (string, error) {
	return r.client.HGet(ctx, r.buildKey(key), field).Result()
}

// HSet 设置Hash字段
func (r *RedisCache) HSet(ctx context.Context, key, field string, value interface{}) error {
	return r.client.HSet(ctx, r.buildKey(key), field, value).Err()
}

// HGetAll 获取Hash所有字段
func (r *RedisCache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	result, err := r.client.HGetAll(ctx, r.buildKey(key)).Result()
	if err != nil {
		return nil, err
	}

	// Redis的HGetAll已经返回map[string]string，直接返回
	return result, nil
}

// HDel 删除Hash字段
func (r *RedisCache) HDel(ctx context.Context, key string, fields ...string) error {
	// HDel接受字符串参数，不需要转换
	return r.client.HDel(ctx, r.buildKey(key), fields...).Err()
}

// HKeys 获取Hash所有字段名
func (r *RedisCache) HKeys(ctx context.Context, key string) ([]string, error) {
	keys, err := r.client.HKeys(ctx, r.buildKey(key)).Result()
	if err != nil {
		return nil, err
	}

	// 移除前缀（如果需要）
	if r.prefix != "" {
		prefixLen := len(r.prefix) + 1
		for i, key := range keys {
			if len(key) > prefixLen {
				keys[i] = key[prefixLen:]
			}
		}
	}

	return keys, nil
}

// MultiLevelCache 多级缓存
type MultiLevelCache struct {
	l1Cache Cache // 本地缓存
	l2Cache Cache // Redis缓存
}

// NewMultiLevelCache 创建多级缓存
func NewMultiLevelCache(l1, l2 Cache) *MultiLevelCache {
	return &MultiLevelCache{
		l1Cache: l1,
		l2Cache: l2,
	}
}

// Get 获取缓存值（先查本地缓存，再查Redis）
func (m *MultiLevelCache) Get(ctx context.Context, key string) (string, error) {
	// 先查本地缓存
	if value, err := m.l1Cache.Get(ctx, key); err == nil {
		return value, nil
	}

	// 查Redis缓存
	value, err := m.l2Cache.Get(ctx, key)
	if err != nil {
		return "", err
	}

	// 回写本地缓存（短时间）
	m.l1Cache.Set(ctx, key, value, 5*time.Minute)

	return value, nil
}

// Set 设置缓存值（同时设置两级缓存）
func (m *MultiLevelCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	// 设置本地缓存（短时间）
	if err := m.l1Cache.Set(ctx, key, value, 5*time.Minute); err != nil {
		return err
	}

	// 设置Redis缓存
	return m.l2Cache.Set(ctx, key, value, expiration)
}

// Delete 删除缓存（同时删除两级缓存）
func (m *MultiLevelCache) Delete(ctx context.Context, key string) error {
	// 删除本地缓存
	m.l1Cache.Delete(ctx, key)

	// 删除Redis缓存
	return m.l2Cache.Delete(ctx, key)
}

// Close 关闭缓存
func (m *MultiLevelCache) Close() error {
	if err := m.l1Cache.Close(); err != nil {
		return err
	}
	return m.l2Cache.Close()
}

// 其他方法代理到L2缓存
func (m *MultiLevelCache) Exists(ctx context.Context, key string) (bool, error) {
	return m.l2Cache.Exists(ctx, key)
}

func (m *MultiLevelCache) MGet(ctx context.Context, keys ...string) ([]string, error) {
	return m.l2Cache.MGet(ctx, keys...)
}

func (m *MultiLevelCache) MSet(ctx context.Context, pairs ...interface{}) error {
	return m.l2Cache.MSet(ctx, pairs...)
}

func (m *MultiLevelCache) MDelete(ctx context.Context, keys ...string) error {
	return m.l2Cache.MDelete(ctx, keys...)
}

func (m *MultiLevelCache) Increment(ctx context.Context, key string) (int64, error) {
	return m.l2Cache.Increment(ctx, key)
}

func (m *MultiLevelCache) IncrementBy(ctx context.Context, key string, value int64) (int64, error) {
	return m.l2Cache.IncrementBy(ctx, key, value)
}

func (m *MultiLevelCache) Decrement(ctx context.Context, key string) (int64, error) {
	return m.l2Cache.Decrement(ctx, key)
}

func (m *MultiLevelCache) DecrementBy(ctx context.Context, key string, value int64) (int64, error) {
	return m.l2Cache.DecrementBy(ctx, key, value)
}

func (m *MultiLevelCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return m.l2Cache.Expire(ctx, key, expiration)
}

func (m *MultiLevelCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return m.l2Cache.TTL(ctx, key)
}

func (m *MultiLevelCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	return m.l2Cache.Keys(ctx, pattern)
}

func (m *MultiLevelCache) FlushDB(ctx context.Context) error {
	m.l1Cache.FlushDB(ctx)
	return m.l2Cache.FlushDB(ctx)
}

// GetStats 获取多级缓存统计信息（主要返回L2缓存的统计信息）
func (m *MultiLevelCache) GetStats(ctx context.Context) (map[string]interface{}, error) {
	// 尝试从L2缓存（Redis）获取统计信息
	if cacheWithStats, ok := m.l2Cache.(interface {
		GetStats(ctx context.Context) (map[string]interface{}, error)
	}); ok {
		return cacheWithStats.GetStats(ctx)
	}

	// 如果L2缓存不支持统计信息，返回基本信息
	stats := map[string]interface{}{
		"cache_type": "multi_level",
		"l1_cache":   "memory",
		"l2_cache":   "redis",
	}

	// 获取键的总数
	keys, err := m.Keys(ctx, "*")
	if err == nil {
		stats["key_count"] = int64(len(keys))
	}

	return stats, nil
}

// MGetJSON 批量获取JSON对象
func (m *MultiLevelCache) MGetJSON(ctx context.Context, keys ...string) (map[string]interface{}, error) {
	if len(keys) == 0 {
		return make(map[string]interface{}), nil
	}

	// 优先从L1缓存获取
	result := make(map[string]interface{})
	missingKeys := make([]string, 0, len(keys))

	for _, key := range keys {
		if value, err := m.l1Cache.Get(ctx, key); err == nil && value != "" {
			var jsonValue interface{}
			if err := json.Unmarshal([]byte(value), &jsonValue); err == nil {
				result[key] = jsonValue
			} else {
				result[key] = value
			}
		} else {
			missingKeys = append(missingKeys, key)
		}
	}

	// 从L2缓存获取缺失的数据
	if len(missingKeys) > 0 {
		if cacheWithMGet, ok := m.l2Cache.(interface {
			MGetJSON(ctx context.Context, keys ...string) (map[string]interface{}, error)
		}); ok {
			l2Result, err := cacheWithMGet.MGetJSON(ctx, missingKeys...)
			if err == nil {
				for key, value := range l2Result {
					result[key] = value
					// 回写L1缓存
					if jsonValue, err := json.Marshal(value); err == nil {
						m.l1Cache.Set(ctx, key, jsonValue, 5*time.Minute)
					}
				}
			}
		} else {
			// 降级到单个获取
			for _, key := range missingKeys {
				if value, err := m.l2Cache.Get(ctx, key); err == nil && value != "" {
					var jsonValue interface{}
					if err := json.Unmarshal([]byte(value), &jsonValue); err == nil {
						result[key] = jsonValue
						// 回写L1缓存
						m.l1Cache.Set(ctx, key, value, 5*time.Minute)
					}
				}
			}
		}
	}

	return result, nil
}

// MSetJSON 批量设置JSON对象
func (m *MultiLevelCache) MSetJSON(ctx context.Context, data map[string]interface{}, expiration time.Duration) error {
	if len(data) == 0 {
		return nil
	}

	// 设置到L1缓存（短时间）
	l1Data := make(map[string]interface{})
	for key, value := range data {
		if jsonValue, err := json.Marshal(value); err == nil {
			l1Data[key] = jsonValue
		}
	}

	if len(l1Data) > 0 {
		if cacheWithMSet, ok := m.l1Cache.(interface {
			MSetJSON(ctx context.Context, data map[string]interface{}, expiration time.Duration) error
		}); ok {
			cacheWithMSet.MSetJSON(ctx, l1Data, 5*time.Minute)
		} else {
			// 降级到单个设置
			for key, value := range l1Data {
				m.l1Cache.Set(ctx, key, value, 5*time.Minute)
			}
		}
	}

	// 设置到L2缓存（长过期时间）
	if cacheWithMSet, ok := m.l2Cache.(interface {
		MSetJSON(ctx context.Context, data map[string]interface{}, expiration time.Duration) error
	}); ok {
		return cacheWithMSet.MSetJSON(ctx, data, expiration)
	} else {
		// 降级到单个设置
		for key, value := range data {
			if err := m.l2Cache.Set(ctx, key, value, expiration); err != nil {
				return err
			}
		}
	}

	return nil
}

// HGet 获取Hash字段
func (m *MultiLevelCache) HGet(ctx context.Context, key, field string) (string, error) {
	return m.l2Cache.HGet(ctx, key, field)
}

// HSet 设置Hash字段
func (m *MultiLevelCache) HSet(ctx context.Context, key, field string, value interface{}) error {
	return m.l2Cache.HSet(ctx, key, field, value)
}

// HGetAll 获取Hash所有字段
func (m *MultiLevelCache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return m.l2Cache.HGetAll(ctx, key)
}

// HDel 删除Hash字段
func (m *MultiLevelCache) HDel(ctx context.Context, key string, fields ...string) error {
	return m.l2Cache.HDel(ctx, key, fields...)
}

// HKeys 获取Hash所有字段名
func (m *MultiLevelCache) HKeys(ctx context.Context, key string) ([]string, error) {
	return m.l2Cache.HKeys(ctx, key)
}