package rpa

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// ============================================================================
// Phase 153 / RUNTIME-03 / RUNTIME-04：Redis Pub/Sub 指令通道（后端侧）
//
// 频道约定（与 rpa-worker 端订阅端一致）：
//   - execution:cancel:{executionID}          取消指令（RUNTIME-03）
//   - worker:human-input:{workerID}:{execID}  人工干预输入（RUNTIME-04）
//
// publish 保持「尽力而为」语义：cache 不支持 Redis 操作（如纯内存 cache 的
// 单元测试环境）时返回可读错误，由调用方决定是否降级 —— DB 终态更新不受影响。
// ============================================================================

const (
	// RedisExecutionCancelChannelFmt 取消指令频道（RUNTIME-03）
	RedisExecutionCancelChannelFmt = "execution:cancel:%s"
	// RedisHumanInputChannelFmt 人工干预输入频道（RUNTIME-04，worker 端
	// subscribeForHumanInput 订阅同一格式）
	RedisHumanInputChannelFmt = "worker:human-input:%s:%s"
	// redisPublishTimeout 单次 publish 超时
	redisPublishTimeout = 3 * time.Second
)

// publishRedisChannel 向 Redis 频道发布 JSON 消息。
// 从 cache.Cache 接口断言底层 Redis 客户端（复用 publishTaskToRedis 同款
// 双实现断言：RedisCache 直连 / MultiLevelCache.GetRedisClient）。
func publishRedisChannel(ctx context.Context, c cache.Cache, channel string, payload interface{}) error {
	if c == nil {
		return fmt.Errorf("cache 未注入，无法发布 Redis 指令")
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化指令失败: %w", err)
	}

	pubCtx, cancel := context.WithTimeout(ctx, redisPublishTimeout)
	defer cancel()

	var client *redis.Client
	if exporter, ok := c.(interface{ ExportClient() *redis.Client }); ok {
		client = exporter.ExportClient()
	}
	if client == nil {
		if ml, ok := c.(interface{ GetRedisClient() *redis.Client }); ok {
			client = ml.GetRedisClient()
		}
	}
	if client == nil {
		return fmt.Errorf("cache 类型不支持 Redis Pub/Sub: %T", c)
	}

	if err := client.Publish(pubCtx, channel, data).Err(); err != nil {
		return fmt.Errorf("PUBLISH %s 失败: %w", channel, err)
	}
	return nil
}

// PublishScaleCommand 发布扩缩容指令到 Redis 频道（Phase 158 / CLEANUP-01）。
//
// 复用 core.Cache 的底层 Redis 客户端（与取消/人工干预同款断言路径），
// 彻底消除 WorkerHandler「每实例 redis.NewClient」的连接泄漏。
//
// 调用方（api/v1/rpa/worker_handler.go）不再持有独立 *redis.Client，
// 统一走 h.core.Cache —— cache 未注入或不支持 Redis 操作时返回可读错误，
// 由调用方决定是否降级。
func PublishScaleCommand(ctx context.Context, c cache.Cache, channel string, payload interface{}) error {
	return publishRedisChannel(ctx, c, channel, payload)
}

// PublishExecutionCancel 发布取消指令（RUNTIME-03）。
// payload 携带谁、何时、原因 —— worker 侧仅按 executionID 取消，字段用于日志留痕。
func PublishExecutionCancel(ctx context.Context, c cache.Cache, executionID, operator, reason string) error {
	return publishRedisChannel(ctx, c, fmt.Sprintf(RedisExecutionCancelChannelFmt, executionID), map[string]interface{}{
		"executionId": executionID,
		"operator":    operator,
		"reason":      reason,
		"timestamp":   time.Now().Unix(),
	})
}

// PublishHumanIntervention 发布人工干预输入（RUNTIME-04）。
// 频道 worker:human-input:{workerID}:{executionID}，payload 为 map（worker 端
// subscribeForHumanInput 按 map 解析后写入变量 / 解析 action）。
func PublishHumanIntervention(ctx context.Context, c cache.Cache, workerID, executionID string, payload map[string]interface{}) error {
	channel := fmt.Sprintf(RedisHumanInputChannelFmt, workerID, executionID)
	return publishRedisChannel(ctx, c, channel, payload)
}
