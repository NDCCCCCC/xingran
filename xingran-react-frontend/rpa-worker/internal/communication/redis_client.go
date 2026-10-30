package communication

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/logger"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// RedisClient Redis 客户端
type RedisClient struct {
	client     *redis.Client
	streamName string
	groupName  string
	consumerID string
	logger     logger.Logger
}

// NewRedisClient 创建 Redis 客户端
func NewRedisClient(cfg *config.RedisConfig, workerID string, log logger.Logger) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: 10,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis 连接失败: %w", err)
	}

	log.Info("Redis 连接成功", logger.String("addr", cfg.Addr))

	return &RedisClient{
		client:     client,
		streamName: cfg.StreamTasks,
		groupName:  cfg.StreamGroup,
		consumerID:  workerID,
		logger:     log,
	}, nil
}

// ConsumeTasks 消费任务
func (r *RedisClient) ConsumeTasks(ctx context.Context) (*types.TaskMessage, string, error) {
	// 创建消费者组（如果不存在）
	r.client.XGroupCreate(ctx, r.streamName, r.groupName, "$", 0).Err()

	// 阻塞读取任务
	results, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    r.groupName,
		Consumer: r.consumerID,
		Streams:  []string{r.streamName, ">"},
		Count:    1,
		Block:    r.blockTime(),
	}).Result()

	if err != nil && err != redis.Nil {
		return nil, "", fmt.Errorf("读取任务失败: %w", err)
	}

	if len(results) == 0 {
		return nil, "", nil
	}

	// 解析任务
	for _, stream := range results {
		for _, message := range stream.Messages {
			var task types.TaskMessage
			
			// 将 Values 转换为 JSON
			valuesJSON, err := json.Marshal(message.Values)
			if err != nil {
				r.logger.Error("序列化消息失败", logger.Err(err))
				continue
			}
			
			if err := json.Unmarshal(valuesJSON, &task); err != nil {
				r.logger.Error("解析任务失败", logger.Err(err))
				continue
			}

			r.logger.Info("获取到任务", 
				logger.String("execution_id", task.ExecutionID),
				logger.String("task_id", task.TaskID),
				logger.String("message_id", message.ID))
			
			return &task, message.ID, nil
		}
	}

	return nil, "", nil
}

// AckTask 确认任务完成
func (r *RedisClient) AckTask(ctx context.Context, messageID string) error {
	return r.client.XAck(ctx, r.streamName, r.groupName, messageID).Err()
}

// PendingTask 处理挂起的任务
func (r *RedisClient) PendingTask(ctx context.Context) ([]*redis.XPending, error) {
	result, err := r.client.XPending(ctx, r.streamName, r.groupName).Result()
	if err != nil {
		return nil, err
	}
	return result, nil
}

// blockTime 获取阻塞时间
func (r *RedisClient) blockTime() time.Duration {
	return 5 * time.Second
}

// Close 关闭连接
func (r *RedisClient) Close() error {
	return r.client.Close()
}
