package cache

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/xingran-next/xingran-go-backend/pkg/logger"
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
	tlsCfg := (*tls.Config)(nil)
	if config.TLS {
		// 托管 Redis (Upstash 等) 强制 TLS;InsecureSkipVerify 与现有 LDAPS 路径一致,
		// 待生产化时统一替换为受信 CA 池 — 单独跟踪,不在本 quick task scope。
		tlsCfg = &tls.Config{InsecureSkipVerify: true}
	}

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
		TLSConfig:       tlsCfg,
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

// IncrementWithExpire 递增并在首次创建时设置过期时间（原子操作）
// 使用 Lua 脚本确保 INCR 和 EXPIRE 要么全成功要么全失败，避免 key 永不过期的问题
func (r *RedisCache) IncrementWithExpire(ctx context.Context, key string, expire time.Duration) (int64, error) {
	// Lua 脚本：原子操作 INCR + EXPIRE
	// 当 key 不存在时，INCR 返回 1，此时设置过期时间
	// 当 key 已存在时，仅递增，不重置过期时间
	script := redis.NewScript(`
		local current = redis.call('INCR', KEYS[1])
		if current == 1 then
			redis.call('EXPIRE', KEYS[1], ARGV[1])
		end
		return current
	`)

	// expire 参数为秒数
	result, err := script.Run(ctx, r.client, []string{r.buildKey(key)}, int64(expire.Seconds())).Result()
	if err != nil {
		return 0, fmt.Errorf("IncrementWithExpire failed: %w", err)
	}
	return result.(int64), nil
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
	l1Cache         Cache             // 本地缓存
	l2Cache         Cache             // Redis缓存
	l2Writer        *L2WriteWorker    // L2写入Worker Pool
	retryWorker     *AsyncRetryWorker // 异步重试工作器
	retryEnabled    bool              // 是否启用重试
	l2WriterEnabled bool              // 是否启用L2 Writer
}

// NewMultiLevelCache 创建多级缓存（不带重试，带默认L2Writer）
func NewMultiLevelCache(l1, l2 Cache) *MultiLevelCache {
	// 创建默认配置的L2Writer
	l2Writer := NewL2WriteWorker(DefaultL2WriterConfig())
	l2Writer.Start()

	return &MultiLevelCache{
		l1Cache:         l1,
		l2Cache:         l2,
		l2Writer:        l2Writer,
		l2WriterEnabled: true,
		retryEnabled:    false,
	}
}

// NewMultiLevelCacheWithWriter 创建带自定义L2Writer的多级缓存
func NewMultiLevelCacheWithWriter(l1, l2 Cache, l2WriterConfig *L2WriterConfig) *MultiLevelCache {
	l2Writer := NewL2WriteWorker(l2WriterConfig)
	l2Writer.Start()

	return &MultiLevelCache{
		l1Cache:         l1,
		l2Cache:         l2,
		l2Writer:        l2Writer,
		l2WriterEnabled: true,
		retryEnabled:    false,
	}
}

// NewMultiLevelCacheSimple 创建多级缓存（不使用L2Writer，保持原有行为）
func NewMultiLevelCacheSimple(l1, l2 Cache) *MultiLevelCache {
	return &MultiLevelCache{
		l1Cache:         l1,
		l2Cache:         l2,
		l2WriterEnabled: false,
		retryEnabled:    false,
	}
}

// NewMultiLevelCacheWithRetry 创建带重试的多级缓存（带默认L2Writer）
func NewMultiLevelCacheWithRetry(l1, l2 Cache, retryConfig *RetryConfig, workerCount int) *MultiLevelCache {
	// 创建默认配置的L2Writer
	l2Writer := NewL2WriteWorker(DefaultL2WriterConfig())
	l2Writer.Start()

	// 创建重试Worker
	worker := NewAsyncRetryWorker(retryConfig, workerCount)
	worker.Start()

	return &MultiLevelCache{
		l1Cache:         l1,
		l2Cache:         l2,
		l2Writer:        l2Writer,
		l2WriterEnabled: true,
		retryWorker:     worker,
		retryEnabled:    true,
	}
}

// NewMultiLevelCacheWithRetryAndWriter 创建带重试和自定义L2Writer的多级缓存
func NewMultiLevelCacheWithRetryAndWriter(l1, l2 Cache, retryConfig *RetryConfig, workerCount int, l2WriterConfig *L2WriterConfig) *MultiLevelCache {
	l2Writer := NewL2WriteWorker(l2WriterConfig)
	l2Writer.Start()

	worker := NewAsyncRetryWorker(retryConfig, workerCount)
	worker.Start()

	return &MultiLevelCache{
		l1Cache:         l1,
		l2Cache:         l2,
		l2Writer:        l2Writer,
		l2WriterEnabled: true,
		retryWorker:     worker,
		retryEnabled:    true,
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

	_ = m.l1Cache.Set(ctx, key, value, 5*time.Minute)
	return value, nil
}

func (m *MultiLevelCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	// L1 缓存同步写入
	if err := m.l1Cache.Set(ctx, key, value, 5*time.Minute); err != nil {
		return err
	}

	// L2 缓存异步写入
	if m.l2WriterEnabled && m.l2Writer != nil {
		// 使用 Worker Pool 模式
		// P1 fix: 不能用请求 ctx 去做 L2 异步入队 —— HTTP 请求 ctx 通常只有 5-30s 截止时间,
		// 但 L2 写入是后台任务,应当独立于请求生命周期。改用独立 ctx 隔离,
		// 真正的客户端取消不应阻塞 L2 异步刷盘。
		enqueueCtx, cancelEnqueue := context.WithTimeout(context.Background(), m.l2Writer.GetFallbackTimeout())
		defer cancelEnqueue()
		if err := m.l2Writer.Enqueue(enqueueCtx, m.l2Cache, key, value, expiration); err != nil {
			// 入队失败（队列满或超时），降级为同步写入以保证数据一致性
			logger.Warnf("[MultiLevelCache] L2写入入队失败，降级为同步写入: key=%s, error=%v", key, err)

			// 获取配置的降级超时时间
			timeout := m.l2Writer.GetFallbackTimeout()

			// 同步写入L2，使用带超时的context防止阻塞
			syncCtx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			if syncErr := m.l2Cache.Set(syncCtx, key, value, expiration); syncErr != nil {
				// 同步写入也失败，记录错误但不影响主流程
				// （L1已成功，客户端至少能从L1获取数据）
				logger.Errorf("[MultiLevelCache] L2同步写入失败: key=%s, error=%v", key, syncErr)
			} else {
				logger.Infof("[MultiLevelCache] L2同步写入成功: key=%s", key)
			}
		}
	} else {
		// 降级到原有的 goroutine 模式（向后兼容）
		go m.asyncSetL2(ctx, key, value, expiration)
	}

	return nil
}

// asyncSetL2 异步设置L2缓存（带重试机制）
// 注意：此方法保留用于向后兼容，当L2Writer未启用时使用
func (m *MultiLevelCache) asyncSetL2(ctx context.Context, key string, value interface{}, expiration time.Duration) {
	l2Ctx := context.Background()
	err := m.l2Cache.Set(l2Ctx, key, value, expiration)

	if err != nil {
		// 写入失败，根据配置决定是否重试
		if m.retryEnabled && m.retryWorker != nil {
			// 将重试任务加入队列
			enqueued := m.retryWorker.Enqueue(ctx, m.l2Cache, key, value, expiration)
			if !enqueued {
				logger.Warnf("[MultiLevelCache] L2缓存写入失败且重试队列已满: key=%s, error=%v", key, err)
			} else {
				logger.Debugf("[MultiLevelCache] L2缓存写入失败，已加入重试队列: key=%s, error=%v", key, err)
			}
		} else {
			logger.Warnf("[MultiLevelCache] L2缓存写入失败（重试未启用）: key=%s, error=%v", key, err)
		}
	}
}

func (m *MultiLevelCache) Delete(ctx context.Context, key string) error {
	_ = m.l1Cache.Delete(ctx, key)
	return m.l2Cache.Delete(ctx, key)
}

func (m *MultiLevelCache) Close() error {
	// 先停止L2写入Worker
	if m.l2Writer != nil {
		m.l2Writer.Stop()
	}

	// 停止重试工作器
	if m.retryWorker != nil {
		m.retryWorker.Stop()
	}

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
	_ = m.l1Cache.MDelete(ctx, keys...)
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

// GetL2WriterStats 获取L2写入Worker的统计信息
func (m *MultiLevelCache) GetL2WriterStats() map[string]interface{} {
	if m.l2Writer != nil {
		return m.l2Writer.GetStats()
	}
	return nil
}

// GetL2WriterQueueSize 获取L2写入Worker的当前队列大小
func (m *MultiLevelCache) GetL2WriterQueueSize() int {
	if m.l2Writer != nil {
		return m.l2Writer.QueueSize()
	}
	return 0
}

// IsL2WriterEnabled 检查L2写入Worker是否启用
func (m *MultiLevelCache) IsL2WriterEnabled() bool {
	return m.l2WriterEnabled && m.l2Writer != nil && m.l2Writer.IsRunning()
}

// GetL2Cache 获取底层L2缓存（Redis）用于需要同步写入的场景
// 避免通过L2Writer异步写入，确保数据立即可用
func (m *MultiLevelCache) GetL2Cache() Cache {
	return m.l2Cache
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
	_ = m.l1Cache.FlushDB(ctx)
	return m.l2Cache.FlushDB(ctx)
}

func (m *MultiLevelCache) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// L1缓存统计
	if l1Stats, ok := m.getCacheStats(m.l1Cache, ctx); ok {
		stats["l1"] = l1Stats
	}

	// L2缓存统计
	if l2Stats, ok := m.getCacheStats(m.l2Cache, ctx); ok {
		stats["l2"] = l2Stats
	}

	// L2写入Worker统计
	if m.l2WriterEnabled && m.l2Writer != nil {
		stats["l2_writer"] = m.GetL2WriterStats()
	}

	// 重试Worker统计
	if m.retryEnabled && m.retryWorker != nil {
		stats["retry_worker"] = m.GetRetryStats()
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

// ==================== 重试相关方法 ====================

// GetRetryStats 获取重试统计信息
func (m *MultiLevelCache) GetRetryStats() map[string]interface{} {
	if !m.retryEnabled || m.retryWorker == nil {
		return map[string]interface{}{
			"retry_enabled": false,
			"message":       "重试功能未启用",
		}
	}

	stats := m.retryWorker.GetStats()
	stats["retry_enabled"] = true
	return stats
}

// IsRetryEnabled 返回重试是否启用
func (m *MultiLevelCache) IsRetryEnabled() bool {
	return m.retryEnabled
}

// EnableRetry 启用重试功能（运行时启用，需要重新创建缓存实例）
func (m *MultiLevelCache) EnableRetry(retryConfig *RetryConfig, workerCount int) {
	if m.retryEnabled {
		logger.Warnf("[MultiLevelCache] 重试功能已启用，跳过")
		return
	}

	m.retryWorker = NewAsyncRetryWorker(retryConfig, workerCount)
	m.retryWorker.Start()
	m.retryEnabled = true
	logger.Infof("[MultiLevelCache] 重试功能已启用")
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
						_ = m.l1Cache.Set(ctx, key, jsonValue, 5*time.Minute)
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
		_ = m.l1Cache.Set(ctx, key, value, 5*time.Minute)
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
		_ = m.l1Cache.Set(ctx, key, jsonValue, 5*time.Minute)
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

// DirectRedisXAdd 直接向 Redis Stream 添加消息
func (m *MultiLevelCache) DirectRedisXAdd(ctx context.Context, stream string, values map[string]interface{}) error {
	if redisCache, ok := m.l2Cache.(redisCacheInterface); ok {
		client := redisCache.getClient()
		if client == nil {
			return fmt.Errorf("Redis客户端未初始化")
		}
		result := client.XAdd(ctx, &redis.XAddArgs{
			Stream: stream,
			Values: values,
		})
		return result.Err()
	}
	return fmt.Errorf("L2缓存不支持Redis操作")
}

// GetRedisClient 获取 Redis 客户端（用于特殊操作如 Stream）
func (m *MultiLevelCache) GetRedisClient() *redis.Client {
	if redisCache, ok := m.l2Cache.(redisCacheInterface); ok {
		return redisCache.getClient()
	}
	return nil
}
