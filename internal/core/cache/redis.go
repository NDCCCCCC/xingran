package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
)

// Redis Redis缓存客户端
type Redis struct {
	Client *redis.Client
}

// NewRedis 创建Redis连接
func NewRedis(cfg *config.CacheConfig) (*Redis, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.GetRedisAddr(),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("Redis连接失败: %w", err)
	}

	log.Println("Redis连接成功")

	return &Redis{Client: rdb}, nil
}

// Close 关闭Redis连接
func (r *Redis) Close() error {
	return r.Client.Close()
}

// Set 设置缓存
func (r *Redis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}

	return r.Client.Set(ctx, key, jsonValue, expiration).Err()
}

// Get 获取缓存
func (r *Redis) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := r.Client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("缓存不存在")
		}
		return fmt.Errorf("获取缓存失败: %w", err)
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return fmt.Errorf("反序列化失败: %w", err)
	}

	return nil
}

// GetString 获取字符串缓存值（用于监控）
func (r *Redis) GetString(ctx context.Context, key string) (string, error) {
	return r.GetRaw(ctx, key)
}

// Del 删除缓存
func (r *Redis) Del(ctx context.Context, keys ...string) error {
	return r.Client.Del(ctx, keys...).Err()
}

// Exists 检查缓存是否存在
func (r *Redis) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.Client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Expire 设置缓存过期时间
func (r *Redis) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.Client.Expire(ctx, key, expiration).Err()
}

// Keys 获取匹配的键
func (r *Redis) Keys(ctx context.Context, pattern string) ([]string, error) {
	return r.Client.Keys(ctx, pattern).Result()
}

// FlushDB 清空当前数据库
func (r *Redis) FlushDB(ctx context.Context) error {
	return r.Client.FlushDB(ctx).Err()
}

// GetRaw 获取原始字符串值（用于监控）
func (r *Redis) GetRaw(ctx context.Context, key string) (string, error) {
	return r.Client.Get(ctx, key).Result()
}

// TTL 获取键的过期时间
func (r *Redis) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.Client.TTL(ctx, key).Result()
}

// GetType 获取键的类型
func (r *Redis) GetType(ctx context.Context, key string) (string, error) {
	return r.Client.Type(ctx, key).Result()
}

// GetSize 获取键的大小（字节）
func (r *Redis) GetSize(ctx context.Context, key string) (int64, error) {
	// 使用MEMORY命令获取内存使用
	result, err := r.Client.Do(ctx, "MEMORY", "USAGE", key).Result()
	if err != nil {
		// 如果MEMORY命令不支持，使用近似估算
		value, err := r.GetRaw(ctx, key)
		if err != nil {
			return 0, err
		}
		return int64(len(value)), nil
	}

	// 解析MEMORY命令的结果
	if str, ok := result.(string); ok {
		// MEMORY USAGE 返回格式如： "keyname (type string) #size bytes"
		parts := strings.Fields(str)
		for i, part := range parts {
			if part == "bytes" && i > 0 {
				if size, err := strconv.ParseInt(parts[i-1], 10, 64); err == nil {
					return size, nil
				}
			}
		}
	}

	// 如果解析失败，返回估算值
	value, err := r.GetRaw(ctx, key)
	if err != nil {
		return 0, err
	}
	return int64(len(value)), nil
}

// GetStats 获取Redis统计信息
func (r *Redis) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 获取服务器信息
	serverInfo, err := r.Client.Info(ctx, "server").Result()
	if err == nil {
		lines := strings.Split(serverInfo, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "#") || line == "" {
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				valueStr := strings.TrimSpace(parts[1])
				if key == "redis_version" || key == "uptime_in_seconds" || key == "uptime_in_days" {
					if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
						stats[key] = value
					} else {
						stats[key] = valueStr
					}
				} else {
					stats[key] = valueStr
				}
			}
		}
	}

	// 获取内存信息
	memInfo, err := r.Client.Info(ctx, "memory").Result()
	if err == nil {
		lines := strings.Split(memInfo, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "#") || line == "" {
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				valueStr := strings.TrimSpace(parts[1])

				// 转换数值类型的值
				if key == "used_memory" || key == "used_memory_rss" || key == "used_memory_peak" ||
				   key == "maxmemory" || key == "used_memory_dataset" || key == "used_memory_overhead" {
					if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
						stats[key] = value
					}
				} else {
					stats[key] = valueStr
				}
			}
		}
	}

	// 获取键的数量
	keyCount, err := r.Client.DBSize(ctx).Result()
	if err == nil {
		stats["key_count"] = keyCount
	}

	// 获取统计信息
	statsInfo, err := r.Client.Info(ctx, "stats").Result()
	if err == nil {
		statsLines := strings.Split(statsInfo, "\n")
		for _, line := range statsLines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "#") || line == "" {
				continue
			}

			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				valueStr := strings.TrimSpace(parts[1])

				if key == "keyspace_hits" || key == "keyspace_misses" ||
				   key == "total_connections_received" || key == "total_commands_processed" ||
				   key == "instantaneous_ops_per_sec" || key == "keyspace_misses" ||
				   key == "expired_keys" || key == "evicted_keys" {
					if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
						stats[key] = value
					}
				}
			}
		}

		// 计算命中率
		hits, hitOk := stats["keyspace_hits"].(int64)
		misses, missOk := stats["keyspace_misses"].(int64)
		if hitOk && missOk && (hits+misses) > 0 {
			hitRate := float64(hits) / float64(hits+misses) * 100
			stats["hit_rate"] = hitRate
		} else {
			stats["hit_rate"] = 0.0
		}
	}

	// 获取数据库信息
	dbInfo, err := r.Client.Info(ctx, "keyspace").Result()
	if err == nil {
		stats["keyspace_info"] = dbInfo
	}

	return stats, nil
}

// Scan 扫描key（支持大量key时的内存友好操作）
func (r *Redis) Scan(ctx context.Context, cursor uint64, match string, count int64) (uint64, []string, error) {
	keys, nextCursor, err := r.Client.Scan(ctx, cursor, match, count).Result()
	return nextCursor, keys, err
}