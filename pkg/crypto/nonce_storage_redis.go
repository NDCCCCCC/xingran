// Package crypto 提供基于 Redis 的分布式 Nonce 存储
package crypto

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisNonceStorage 基于 Redis 的分布式 nonce 存储
// 支持多实例部署，天然防止重放攻击
type redisNonceStorage struct {
	client         *redis.Client
	ctx            context.Context
	replayWindowSec int // 时间戳容差(秒),用于 nonce TTL。<=0 时使用 DefaultReplayWindowSec
}

// RedisNonceStorageConfig Redis 配置
type RedisNonceStorageConfig struct {
	Addr     string // Redis 地址，如 "localhost:6379"
	Password string // Redis 密码
	DB       int    // Redis 数据库编号
	ReplayWindowSec int // nonce TTL = 2 * ReplayWindowSec (秒),<=0 时使用 DefaultReplayWindowSec
}

// NewRedisNonceStorage 创建基于 Redis 的 nonce 存储
func NewRedisNonceStorage(config RedisNonceStorageConfig) (NonceStorage, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis 连接失败: %w", err)
	}

	window := config.ReplayWindowSec
	if window <= 0 {
		window = DefaultReplayWindowSec
	}
	return &redisNonceStorage{
		client:         client,
		ctx:            context.Background(),
		replayWindowSec: window,
	}, nil
}

// getNonceKey 生成 Redis 中存储 nonce 的 key
func (r *redisNonceStorage) getNonceKey(nonce string) string {
	return fmt.Sprintf("nonce:%s", nonce)
}

// CheckAndStore 检查并存储 nonce
// 使用 Redis 的 SETNX（SET if Not eXists）命令，原子性操作
func (r *redisNonceStorage) CheckAndStore(nonce string, timestamp int64) bool {
	key := r.getNonceKey(nonce)

	// SETNX: 如果 key 不存在则设置，返回 true；如果已存在则不设置，返回 false
	// 过期时间 = 2 * replayWindowSec,与 anti-replay 窗口对齐
	ok, err := r.client.SetNX(r.ctx, key, timestamp, time.Duration(r.replayWindowSec)*2*time.Second).Result()
	if err != nil {
		// Redis 出错时，为安全起见拒绝请求
		return false
	}

	return ok
}

// Close 关闭 Redis 连接
func (r *redisNonceStorage) Close() error {
	return r.client.Close()
}

// GetNonceCount 获取当前存储的 nonce 数量（用于监控）
func (r *redisNonceStorage) GetNonceCount() int {
	keys, err := r.client.Keys(r.ctx, "nonce:*").Result()
	if err != nil {
		return 0
	}
	return len(keys)
}
