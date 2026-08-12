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

// RedisClient Redis client
type RedisClient struct {
	client     *redis.Client
	streamName string
	groupName  string
	consumerID string
	logger     logger.Logger
	config     *config.RedisConfig
}

// NewRedisClient create Redis client
func NewRedisClient(cfg *config.RedisConfig, workerID string, log logger.Logger) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: 10,
	})

	// Test connection with configured timeout
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectionTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("Redis connection failed: %w", err)
	}

	log.Info("Redis connected", logger.String("addr", cfg.Addr))

	return &RedisClient{
		client:     client,
		streamName: cfg.StreamTasks,
		groupName:  cfg.StreamGroup,
		consumerID:  workerID,
		logger:     log,
		config:     cfg,
	}, nil
}

// ConsumeTasks consume tasks
func (r *RedisClient) ConsumeTasks(ctx context.Context) (*types.TaskMessage, string, error) {
	// Create consumer group and stream if not exists
	// MKSTREAM creates the stream if it doesn't exist
	err := r.client.XGroupCreateMkStream(ctx, r.streamName, r.groupName, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		r.logger.Warn("create consumer group", logger.Err(err))
		// Continue anyway, group might already exist
	}

	// Blocking read tasks
	results, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    r.groupName,
		Consumer: r.consumerID,
		Streams:  []string{r.streamName, ">"},
		Count:    1,
		Block:    r.blockTime(),
	}).Result()

	if err != nil && err != redis.Nil {
		return nil, "", fmt.Errorf("read tasks failed: %w", err)
	}

	if len(results) == 0 {
		return nil, "", nil
	}

	// Parse tasks
	for _, stream := range results {
		for _, message := range stream.Messages {
			var task types.TaskMessage

			// Backend wraps task message in "data" field
			dataValue, ok := message.Values["data"]
			if !ok {
				r.logger.Error("missing data field in message", logger.String("message_id", message.ID))
				continue
			}

			// Convert data value to string and unmarshal
			dataJSON, ok := dataValue.(string)
			if !ok {
				r.logger.Error("data field is not string", logger.String("message_id", message.ID))
				continue
			}

			if err := json.Unmarshal([]byte(dataJSON), &task); err != nil {
				r.logger.Error("parse task failed", logger.Err(err), logger.String("message_id", message.ID))
				continue
			}

			r.logger.Info("got task",
				logger.String("execution_id", task.ExecutionID),
				logger.String("task_id", task.TaskID),
				logger.String("message_id", message.ID))
			return &task, message.ID, nil
		}
	}

	return nil, "", nil
}

// AckTask acknowledge task completion
func (r *RedisClient) AckTask(ctx context.Context, messageID string) error {
	return r.client.XAck(ctx, r.streamName, r.groupName, messageID).Err()
}

// blockTime get block time from config
func (r *RedisClient) blockTime() time.Duration {
	return r.config.BlockTime
}

// Close close connection
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// GetClient returns the underlying Redis client
// This is used for advanced operations like Pub/Sub
func (r *RedisClient) GetClient() *redis.Client {
	return r.client
}
