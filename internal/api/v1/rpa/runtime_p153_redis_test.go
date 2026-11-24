package rpa

// Phase 153 / RUNTIME-03 / RUNTIME-04 真 Redis 集成用例。
//
// 依赖真实 Redis（REDIS_ADDR，默认 127.0.0.1:6379）。不可达时 skip ——
// 不引入 miniredis（Phase 153 硬约束：零新增依赖），publish 降级语义已由
// runtime_p153_test.go 的 fakeNoRedisCache 用例覆盖。
//
// 验证内容：
//   RUNTIME-03  Cancel → execution:cancel:{id} 频道收到指令（payload 含 operator/reason）
//   RUNTIME-04  SubmitHumanIntervention → worker:human-input:{worker}:{exec}
//               频道收到 {action,input,reason}（worker 端 subscribeForHumanInput 同格式）

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	svcrpa "github.com/xingran-next/xingran-go-backend/internal/services/rpa"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// redisCacheAdapter 测试适配器：真 redis.Client 包装为 publishRedisChannel
// 可断言的 cache.Cache（嵌入 nil 接口 + GetRedisClient 出口）。
type redisCacheAdapter struct {
	cache.Cache
	client *redis.Client
}

func (a redisCacheAdapter) GetRedisClient() *redis.Client { return a.client }

func envLookup(key string) string { return os.Getenv(key) }

func runtimeRedisClient(t *testing.T) *redis.Client {
	t.Helper()
	addr := envOrDefault("REDIS_ADDR", "127.0.0.1:6379")
	client := redis.NewClient(&redis.Options{Addr: addr, DialTimeout: 2 * time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis 不可达（%s），跳过真 Redis 集成用例: %v", addr, err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func envOrDefault(key, def string) string {
	if v := envLookup(key); v != "" {
		return v
	}
	return def
}

func TestRuntime03_Cancel_RealRedis_PublishesCommand(t *testing.T) {
	client := runtimeRedisClient(t)
	ctx := context.Background()
	executionID := "rt03-redis-" + time.Now().Format("150405.000000000")

	sub := client.Subscribe(ctx, "execution:cancel:"+executionID)
	defer sub.Close()
	if _, err := sub.Receive(ctx); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	if err := svcrpa.PublishExecutionCancel(ctx, redisCacheAdapter{Cache: nil, client: client}, executionID, "op-user", "参数有误"); err != nil {
		t.Fatalf("PublishExecutionCancel: %v", err)
	}

	msg, err := sub.ReceiveTimeout(ctx, 5*time.Second)
	if err != nil {
		t.Fatalf("未收到取消指令: %v", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(msg.(*redis.Message).Payload), &payload); err != nil {
		t.Fatalf("payload 解析失败: %v", err)
	}
	if payload["operator"] != "op-user" || payload["reason"] != "参数有误" {
		t.Fatalf("payload 应携带谁/原因（审计），实际: %v", payload)
	}
	if payload["executionId"] != executionID {
		t.Fatalf("executionId 不匹配: %v", payload)
	}
}

func TestRuntime04_HumanInput_RealRedis_PublishesAction(t *testing.T) {
	client := runtimeRedisClient(t)
	ctx := context.Background()
	workerID := "wk-redis"
	executionID := "rt04-redis-" + time.Now().Format("150405.000000000")
	channel := "worker:human-input:" + workerID + ":" + executionID

	sub := client.Subscribe(ctx, channel)
	defer sub.Close()
	if _, err := sub.Receive(ctx); err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	// 与 handler SubmitHumanIntervention 同款 payload 形状
	payload := map[string]interface{}{
		"action":      "resume",
		"input":       map[string]interface{}{"captcha": "1234"},
		"reason":      "人工确认",
		"executionId": executionID,
		"operator":    "op-user",
	}
	if err := svcrpa.PublishHumanIntervention(ctx, redisCacheAdapter{Cache: nil, client: client}, workerID, executionID, payload); err != nil {
		t.Fatalf("PublishHumanIntervention: %v", err)
	}

	msg, err := sub.ReceiveTimeout(ctx, 5*time.Second)
	if err != nil {
		t.Fatalf("未收到干预输入: %v", err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal([]byte(msg.(*redis.Message).Payload), &got); err != nil {
		t.Fatalf("payload 解析失败: %v", err)
	}
	if got["action"] != "resume" {
		t.Fatalf("action 应为 resume，实际: %v", got["action"])
	}
	input, _ := got["input"].(map[string]interface{})
	if input["captcha"] != "1234" {
		t.Fatalf("input 应原样透传（worker 写入任务变量），实际: %v", got["input"])
	}
}
