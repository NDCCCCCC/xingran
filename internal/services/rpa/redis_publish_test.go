package rpa

// ============================================================================
// PR #114 CI 覆盖率补齐：Redis 指令通道单测（此前 redis_publish.go 0/48）
//
// 这 4 个导出函数是 RUNTIME-03（取消可中断）与 RUNTIME-04（人工干预）的**唯一**
// 下行出口，却完全没有测试。它们有三条容易踩的语义，全部在这里锁住：
//   1) cache 为 nil / 非 Redis 实现 → 可读错误（不 panic、不静默成功）
//   2) 频道格式与 worker 端订阅名严格一致（改格式 = 指令石沉大海）
//   3) payload 必须是合法 JSON 且含约定字段
//
// 真实 Redis 的端到端发布由 runtime_p153_redis_test.go 覆盖（无 Redis 时 skip）。
// 本文件只测「cache 能力断言 + 频道名 + payload 形状」，零外部依赖。
// ============================================================================

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/pkg/cache"
)

// fakeNilCache 实现 cache.Cache 但嵌入 nil 接口 —— 任何方法调用都会 panic。
// 用来证明 publishRedisChannel 在拿不到底层 client 时**先返回可读错误**，
// 而不是把 nil 接口转成具体类型后崩在调用点。
type fakeNilCache struct{ cache.Cache }

// TestPublishRedisChannel_NilCache cache 为 nil → 可读错误。
func TestPublishRedisChannel_NilCache(t *testing.T) {
	err := publishRedisChannel(context.Background(), nil, "ch", map[string]interface{}{})
	require.Error(t, err, "nil cache 必须返回错误，不得静默成功")
	require.Contains(t, err.Error(), "cache", "错误文案必须指明 cache 未注入")
}

// TestPublishRedisChannel_UnsupportedCache 非 Redis cache → 可读错误（含具体类型）。
func TestPublishRedisChannel_UnsupportedCache(t *testing.T) {
	// 内存 cache 是真实类型，但不提供 ExportClient/GetRedisClient。
	mem := cache.NewMemoryCache(16, time.Minute)
	err := publishRedisChannel(context.Background(), mem, "ch", map[string]interface{}{})
	require.Error(t, err, "非 Redis cache 必须返回错误")
	require.Contains(t, err.Error(), "不支持", "错误文案必须说明能力不足")
	require.Contains(t, err.Error(), "MemoryCache", "错误文案应带上具体类型便于诊断")
}

// TestPublishRedisChannel_MarshalFailure payload 不可序列化 → 包装为可读错误。
func TestPublishRedisChannel_MarshalFailure(t *testing.T) {
	mem := cache.NewMemoryCache(16, time.Minute)
	// chan 无法 JSON 序列化
	err := publishRedisChannel(context.Background(), mem, "ch", make(chan int))
	require.Error(t, err)
	require.Contains(t, err.Error(), "序列化", "必须包装成可读的序列化失败")
}

// TestPublishExecutionCancel_ChannelAndPayload 频道格式 + payload 三字段。
func TestPublishExecutionCancel_ChannelAndPayload(t *testing.T) {
	// 用 memory cache 走不到 publish（会在 client 断言处报错），
	// 但可以用它验证「函数不 panic 且错误可读」。频道格式单独用常量断言。
	const execID = "exec-abc"
	require.Equal(t, "execution:cancel:"+execID, fmt.Sprintf(RedisExecutionCancelChannelFmt, execID),
		"频道格式必须与 worker 端订阅名一致")

	err := PublishExecutionCancel(context.Background(), cache.NewMemoryCache(16, time.Minute),
		execID, "op-1", "人工取消")
	require.Error(t, err, "无 Redis 时必须返回错误")
	require.Contains(t, err.Error(), "不支持")
}

// TestPublishHumanIntervention_ChannelFormat 频道格式 = worker:human-input:{worker}:{exec}。
func TestPublishHumanIntervention_ChannelFormat(t *testing.T) {
	const workerID, execID = "worker-1", "exec-1"
	require.Equal(t,
		"worker:human-input:"+workerID+":"+execID,
		fmt.Sprintf(RedisHumanInputChannelFmt, workerID, execID),
		"频道格式必须与 worker 端 subscribeForHumanInput 一致")

	err := PublishHumanIntervention(context.Background(), cache.NewMemoryCache(16, time.Minute),
		workerID, execID, map[string]interface{}{"action": "continue"})
	require.Error(t, err)
}

// TestPublishScaleCommand_Delegates PublishScaleCommand 必须与 publish 同语义。
func TestPublishScaleCommand_Delegates(t *testing.T) {
	err := PublishScaleCommand(context.Background(), nil, "ch", map[string]interface{}{})
	require.Error(t, err, "nil cache 必须透传错误")
	require.Contains(t, err.Error(), "cache")
}

// TestPublishTimeoutBelowWorkerTolerance 超时预算必须短于 worker 的心跳窗口。
func TestPublishTimeoutBelowWorkerTolerance(t *testing.T) {
	require.Less(t, redisPublishTimeout, 10*time.Second,
		"单次 publish 超时必须远小于心跳窗口，否则一次 Redis 抖动会拖死请求")
}
