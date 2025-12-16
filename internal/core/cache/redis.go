package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/config"
)

// Redis Redis缓存客户端
type Redis struct {
	Client *redis.Client
}

// NewRedis 创建Redis连接
func NewRedis(cfg *config.RedisConfig) (*Redis, error) {
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