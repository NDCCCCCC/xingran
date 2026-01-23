package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
)

const (
	dialTimeout  = 10 * time.Second
	readTimeout  = 5 * time.Second
	writeTimeout = 5 * time.Second
	poolTimeout  = 10 * time.Second
)

// RedisCache Redis缓存实现
type RedisCache struct {
	client *redis.Client
	prefix string
}

// NewRedisCache 创建Redis缓存实例
func NewRedisCache(config *CacheConfig, keyPrefix string) (*RedisCache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:            fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password:        config.Password,
		DB:              config.DB,
		PoolSize:        config.PoolSize,
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
		DialTimeout:     dialTimeout,
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		PoolTimeout:     poolTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis连接失败: %w", err)
	}

	logger.Infof("[Redis] 连接成功 %s:%d (DB:%d)", config.Host, config.Port, config.DB)

	return &RedisCache{
		client: rdb,
		prefix: keyPrefix,
	}, nil
}

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
	return r.client.TTL(ctx, r.buildKey(key)).Result()
}

// Keys 模式匹配
func (r *RedisCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	builtPattern := r.buildKey(pattern)
	keys, err := r.client.Keys(ctx, builtPattern).Result()
	if err != nil {
		return nil, err
	}

	if r.prefix != "" {
		prefixWithColon := r.prefix + ":"
		prefixLen := len(prefixWithColon)
		for i, key := range keys {
			if strings.HasPrefix(key, prefixWithColon) && len(key) > prefixLen {
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

func (r *RedisCache) getClient() *redis.Client {
	return r.client
}

func (r *RedisCache) getPrefix() string {
	return r.prefix
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
	strValue := "0"
	if value {
		strValue = "1"
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
			if strValue, ok := values[i].(string); ok {
				var value interface{}
				if err := json.Unmarshal([]byte(strValue), &value); err != nil {
					result[key] = strValue
				} else {
					result[key] = value
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

	if expiration > 0 {
		for key := range data {
			r.client.Expire(ctx, r.buildKey(key), expiration)
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
	return r.client.HGetAll(ctx, r.buildKey(key)).Result()
}

// HDel 删除Hash字段
func (r *RedisCache) HDel(ctx context.Context, key string, fields ...string) error {
	return r.client.HDel(ctx, r.buildKey(key), fields...).Err()
}

// HKeys 获取Hash所有字段名
func (r *RedisCache) HKeys(ctx context.Context, key string) ([]string, error) {
	keys, err := r.client.HKeys(ctx, r.buildKey(key)).Result()
	if err != nil {
		return nil, err
	}

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

// GetStats 获取Redis统计信息
func (r *RedisCache) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	serverInfo, err := r.client.Info(ctx, "server").Result()
	if err == nil {
		stats = r.parseInfoSection(serverInfo, stats, []string{
			"redis_version", "uptime_in_seconds", "uptime_in_days",
		}, func(v string) interface{} {
			if v, err := strconv.ParseInt(v, 10, 64); err == nil {
				return v
			}
			return v
		})
	}

	memInfo, err := r.client.Info(ctx, "memory").Result()
	if err == nil {
		stats = r.parseInfoSection(memInfo, stats, []string{
			"used_memory", "used_memory_rss", "used_memory_peak",
			"maxmemory", "total_system_memory",
		}, func(v string) interface{} {
			if v, err := strconv.ParseInt(v, 10, 64); err == nil {
				return v
			}
			return v
		})
	}

	keyCount, err := r.client.DBSize(ctx).Result()
	if err == nil {
		stats["key_count"] = keyCount
	}

	statsInfo, err := r.client.Info(ctx, "stats").Result()
	if err == nil {
		stats = r.parseInfoSection(statsInfo, stats, []string{
			"keyspace_hits", "keyspace_misses",
			"total_connections_received", "total_commands_processed",
			"instantaneous_ops_per_sec", "expired_keys", "evicted_keys",
		}, func(v string) interface{} {
			if v, err := strconv.ParseInt(v, 10, 64); err == nil {
				return v
			}
			return v
		})

		hits, _ := stats["keyspace_hits"].(int64)
		misses, _ := stats["keyspace_misses"].(int64)
		if hits+misses > 0 {
			stats["hit_rate"] = float64(hits) / float64(hits+misses) * 100
		} else {
			stats["hit_rate"] = 0.0
		}
	}

	dbInfo, _ := r.client.Info(ctx, "keyspace").Result()
	stats["keyspace_info"] = dbInfo

	return stats, nil
}

func (r *RedisCache) parseInfoSection(info string, stats map[string]interface{}, intKeys []string, parser func(string) interface{}) map[string]interface{} {
	lines := strings.Split(info, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			valueStr := strings.TrimSpace(parts[1])
			if contains(intKeys, key) {
				stats[key] = parser(valueStr)
			} else {
				stats[key] = valueStr
			}
		}
	}
	return stats
}

// contains 字符串切片包含检查（简单版本，用于内部辅助）
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ==================== MultiLevelCache 多级缓存 ====================

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

func (m *MultiLevelCache) Get(ctx context.Context, key string) (string, error) {
	if value, err := m.l1Cache.Get(ctx, key); err == nil {
		return value, nil
	}

	value, err := m.l2Cache.Get(ctx, key)
	if err != nil {
		return "", err
	}

	m.l1Cache.Set(ctx, key, value, 5*time.Minute)
	return value, nil
}

func (m *MultiLevelCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if err := m.l1Cache.Set(ctx, key, value, 5*time.Minute); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := m.l2Cache.Set(ctx, key, value, expiration); err != nil {
		logger.Warnf("[MultiLevelCache] L2缓存写入失败: %v", err)
	}

	return nil
}

func (m *MultiLevelCache) Delete(ctx context.Context, key string) error {
	m.l1Cache.Delete(ctx, key)
	return m.l2Cache.Delete(ctx, key)
}

func (m *MultiLevelCache) Close() error {
	if err := m.l1Cache.Close(); err != nil {
		return err
	}
	return m.l2Cache.Close()
}

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
	m.l1Cache.MDelete(ctx, keys...)
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
	l1Keys, _ := m.l1Cache.Keys(ctx, pattern)
	l2Keys, _ := m.l2Cache.Keys(ctx, pattern)
	return m.mergeKeys(l1Keys, l2Keys), nil
}

func (m *MultiLevelCache) KeysByLevel(ctx context.Context, pattern string, level string) ([]string, error) {
	switch strings.ToLower(level) {
	case "l1":
		return m.l1Cache.Keys(ctx, pattern)
	case "l2":
		return m.l2Cache.Keys(ctx, pattern)
	default:
		l1Keys, _ := m.l1Cache.Keys(ctx, pattern)
		l2Keys, _ := m.l2Cache.Keys(ctx, pattern)
		return m.mergeKeys(l1Keys, l2Keys), nil
	}
}

func (m *MultiLevelCache) FlushDB(ctx context.Context) error {
	m.l1Cache.FlushDB(ctx)
	return m.l2Cache.FlushDB(ctx)
}

func (m *MultiLevelCache) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	if l1Stats, ok := m.getCacheStats(m.l1Cache, ctx); ok {
		stats["l1"] = l1Stats
	}

	if l2Stats, ok := m.getCacheStats(m.l2Cache, ctx); ok {
		stats["l2"] = l2Stats
	}

	return stats, nil
}

func (m *MultiLevelCache) getCacheStats(cache Cache, ctx context.Context) (map[string]interface{}, bool) {
	if cacheWithStats, ok := cache.(interface {
		GetStats(ctx context.Context) (map[string]interface{}, error)
	}); ok {
		stats, err := cacheWithStats.GetStats(ctx)
		return stats, err == nil
	}
	return nil, false
}

func (m *MultiLevelCache) mergeKeys(l1, l2 []string) []string {
	keySet := make(map[string]bool, len(l1)+len(l2))
	for _, k := range l1 {
		keySet[k] = true
	}
	for _, k := range l2 {
		keySet[k] = true
	}
	result := make([]string, 0, len(keySet))
	for k := range keySet {
		result = append(result, k)
	}
	return result
}

// MGetJSON 批量获取JSON对象
func (m *MultiLevelCache) MGetJSON(ctx context.Context, keys ...string) (map[string]interface{}, error) {
	if len(keys) == 0 {
		return make(map[string]interface{}), nil
	}

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

	if len(missingKeys) > 0 {
		if cacheWithMGet, ok := m.l2Cache.(interface {
			MGetJSON(ctx context.Context, keys ...string) (map[string]interface{}, error)
		}); ok {
			l2Result, err := cacheWithMGet.MGetJSON(ctx, missingKeys...)
			if err == nil {
				for key, value := range l2Result {
					result[key] = value
					if jsonValue, err := json.Marshal(value); err == nil {
						m.l1Cache.Set(ctx, key, jsonValue, 5*time.Minute)
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

	l1Data := make(map[string]interface{})
	for key, value := range data {
		if jsonValue, err := json.Marshal(value); err == nil {
			l1Data[key] = jsonValue
		}
	}

	for key, value := range l1Data {
		m.l1Cache.Set(ctx, key, value, 5*time.Minute)
	}

	for key, value := range data {
		if err := m.l2Cache.Set(ctx, key, value, expiration); err != nil {
			return err
		}
	}

	return nil
}

func (m *MultiLevelCache) HGet(ctx context.Context, key, field string) (string, error) {
	return m.l2Cache.HGet(ctx, key, field)
}

func (m *MultiLevelCache) HSet(ctx context.Context, key, field string, value interface{}) error {
	return m.l2Cache.HSet(ctx, key, field, value)
}

func (m *MultiLevelCache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return m.l2Cache.HGetAll(ctx, key)
}

func (m *MultiLevelCache) HDel(ctx context.Context, key string, fields ...string) error {
	return m.l2Cache.HDel(ctx, key, fields...)
}

func (m *MultiLevelCache) HKeys(ctx context.Context, key string) ([]string, error) {
	return m.l2Cache.HKeys(ctx, key)
}

func (m *MultiLevelCache) SetInt(ctx context.Context, key string, value int, expiration time.Duration) error {
	return m.l2Cache.SetInt(ctx, key, value, expiration)
}

func (m *MultiLevelCache) GetInt(ctx context.Context, key string) (int, error) {
	return m.l2Cache.GetInt(ctx, key)
}

func (m *MultiLevelCache) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if jsonValue, err := json.Marshal(value); err == nil {
		m.l1Cache.Set(ctx, key, jsonValue, 5*time.Minute)
	}
	return m.l2Cache.SetJSON(ctx, key, value, expiration)
}

func (m *MultiLevelCache) GetJSON(ctx context.Context, key string, dest interface{}) error {
	if value, err := m.l1Cache.Get(ctx, key); err == nil && value != "" {
		if err := json.Unmarshal([]byte(value), dest); err == nil {
			return nil
		}
	}
	return m.l2Cache.GetJSON(ctx, key, dest)
}

// ==================== 直接访问 Redis 的方法（用于缓存监控）====================

type redisCacheInterface interface {
	getClient() *redis.Client
	getPrefix() string
}

// DirectRedisKeys 直接查询 Redis 中的所有键（不使用前缀）
func (m *MultiLevelCache) DirectRedisKeys(ctx context.Context, pattern string) ([]string, error) {
	if redisCache, ok := m.l2Cache.(redisCacheInterface); ok {
		client := redisCache.getClient()
		if client == nil {
			return []string{}, nil
		}
		return client.Keys(ctx, pattern).Result()
	}
	return []string{}, nil
}

// DirectRedisGet 直接从 Redis 获取缓存值（不使用前缀）
func (m *MultiLevelCache) DirectRedisGet(ctx context.Context, key string) (string, error) {
	if redisCache, ok := m.l2Cache.(redisCacheInterface); ok {
		client := redisCache.getClient()
		if client == nil {
			return "", nil
		}
		result, err := client.Get(ctx, key).Result()
		if err == redis.Nil {
			return "", nil
		}
		return result, err
	}
	return "", nil
}

// DirectRedisTTL 直接从 Redis 获取 TTL（不使用前缀）
func (m *MultiLevelCache) DirectRedisTTL(ctx context.Context, key string) (time.Duration, error) {
	if redisCache, ok := m.l2Cache.(redisCacheInterface); ok {
		client := redisCache.getClient()
		if client == nil {
			return 0, nil
		}
		return client.TTL(ctx, key).Result()
	}
	return 0, nil
}
