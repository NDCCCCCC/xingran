package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestRetryConfig_Default 测试默认重试配置
func TestRetryConfig_Default(t *testing.T) {
	config := DefaultRetryConfig()

	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 50*time.Millisecond, config.InitialDelay)
	assert.Equal(t, 2*time.Second, config.MaxDelay)
	assert.Equal(t, 2.0, config.BackoffFactor)
	assert.NotNil(t, config.RetryableCheck)
}

// TestRetryStats 测试重试统计
func TestRetryStats(t *testing.T) {
	stats := NewRetryStats()

	// 测试初始状态
	statsMap := stats.GetStats()
	assert.Equal(t, int64(0), statsMap["total_attempts"])
	assert.Equal(t, int64(0), statsMap["total_success"])
	assert.Equal(t, int64(0), statsMap["total_failure"])

	// 记录成功
	stats.RecordSuccess()
	statsMap = stats.GetStats()
	assert.Equal(t, int64(1), statsMap["total_success"])

	// 记录失败
	testErr := errors.New("test error")
	stats.RecordFailure(testErr)
	statsMap = stats.GetStats()
	assert.Equal(t, int64(1), statsMap["total_failure"])
	assert.Equal(t, int64(1), statsMap["consecutive_fails"])
	assert.NotNil(t, statsMap["last_error"])

	// 记录重试
	stats.RecordRetry()
	statsMap = stats.GetStats()
	assert.Equal(t, int64(1), statsMap["total_retries"])
}

// TestIsRetryableError 测试可重试错误判断
func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"连接被拒绝", errors.New("connection refused"), true},
		{"连接重置(完整短语)", errors.New("connection reset by peer"), true},
		{"连接重置(简写)", errors.New("connection reset"), false}, // 不再匹配简写形式，避免误判
		{"broken pipe", errors.New("broken pipe"), true},
		{"i/o timeout", errors.New("i/o timeout"), true},
		{"超时(通用)", errors.New("timeout"), false}, // "timeout"单独不再匹配，避免误判
		{"read超时", errors.New("read: connection timed out"), true},
		{"dial超时", errors.New("dial tcp: timeout"), true},
		{"context deadline exceeded", errors.New("context deadline exceeded"), true},
		{"网络不可达", errors.New("network is unreachable"), true},
		{"unexpected EOF", errors.New("unexpected EOF"), true},
		{"EOF(简写)", errors.New("EOF"), false}, // 单独EOF不再匹配，避免误判业务EOF
		{"Redis加载中", errors.New("LOADING Redis is loading"), true},
		{"Redis只读", errors.New("READONLY You can't write"), true},
		{"Redis集群下线", errors.New("CLUSTERDOWN"), true},
		{"语法错误", errors.New("ERR syntax error"), false}, // 明确不重试
		{"权限错误", errors.New("NOPERM"), false},           // 明确不重试
		{"其他错误", errors.New("some other error"), false},
		{"空错误", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			assert.Equal(t, tt.expected, result)
			if tt.err != nil {
				// 只在非nil错误时打印详细信息
				assert.Equal(t, tt.expected, result, "错误: %s", tt.err.Error())
			}
		})
	}
}

// TestRetryWithContext_Success 测试重试成功场景
func TestRetryWithContext_Success(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:     3,
		InitialDelay:   10 * time.Millisecond,
		MaxDelay:       100 * time.Millisecond,
		BackoffFactor:  2.0,
		RetryableCheck: IsRetryableError,
	}

	attempts := 0
	fn := func() error {
		attempts++
		if attempts < 3 {
			return errors.New("connection refused")
		}
		return nil
	}

	err := RetryWithContext(context.Background(), config, fn)
	assert.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

// TestRetryWithContext_AllRetriesFailed 测试所有重试都失败
func TestRetryWithContext_AllRetriesFailed(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:     3,
		InitialDelay:   10 * time.Millisecond,
		MaxDelay:       100 * time.Millisecond,
		BackoffFactor:  2.0,
		RetryableCheck: IsRetryableError,
	}

	attempts := 0
	fn := func() error {
		attempts++
		return errors.New("connection refused")
	}

	err := RetryWithContext(context.Background(), config, fn)
	assert.Error(t, err)
	assert.Equal(t, 4, attempts) // 初始调用 + 3次重试
	assert.Contains(t, err.Error(), "重试3次后仍失败")
}

// TestRetryWithContext_NonRetryableError 测试不可重试错误
func TestRetryWithContext_NonRetryableError(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:     3,
		InitialDelay:   10 * time.Millisecond,
		MaxDelay:       100 * time.Millisecond,
		BackoffFactor:  2.0,
		RetryableCheck: IsRetryableError,
	}

	attempts := 0
	fn := func() error {
		attempts++
		return errors.New("permission denied")
	}

	err := RetryWithContext(context.Background(), config, fn)
	assert.Error(t, err)
	assert.Equal(t, 1, attempts) // 只调用一次，不重试
}

// TestRetryWithContext_ContextCancelled 测试上下文取消
func TestRetryWithContext_ContextCancelled(t *testing.T) {
	config := &RetryConfig{
		MaxRetries:     3,
		InitialDelay:   100 * time.Millisecond,
		MaxDelay:       1 * time.Second,
		BackoffFactor:  2.0,
		RetryableCheck: IsRetryableError,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	fn := func() error {
		return errors.New("connection refused")
	}

	err := RetryWithContext(ctx, config, fn)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

// TestAsyncRetryWorker_Enqueue 测试异步重试工作器队列
func TestAsyncRetryWorker_Enqueue(t *testing.T) {
	config := DefaultRetryConfig()
	worker := NewAsyncRetryWorker(config, 2)
	worker.Start()
	defer worker.Stop()

	// 使用真实的内存缓存进行测试
	memoryCache := NewMemoryCache(100, 5*time.Minute)

	ctx := context.Background()

	// 测试入队
	success := worker.Enqueue(ctx, memoryCache, "test_key", "test_value", time.Minute)
	assert.True(t, success)

	// 验证队列大小（任务可能已被处理，所以只需验证>=0）
	queueSize := worker.QueueSize()
	assert.True(t, queueSize >= 0)

	// 等待处理
	time.Sleep(200 * time.Millisecond)

	// 验证统计信息
	stats := worker.GetStats()
	// 验证worker_count存在
	assert.Equal(t, 2, stats["worker_count"])
	// 验证队列大小（worker 并发消费下随时变化，只能验证非负，
	// 不能断言与 sleep 前相等——那是时序竞态，曾导致 ~10% 概率 flake）
	statsQueueSize, ok := stats["queue_size"].(int)
	assert.True(t, ok, "queue_size 应为 int 类型")
	assert.GreaterOrEqual(t, statsQueueSize, 0)
}

// TestAsyncRetryWorker_QueueSemantics 不启动 worker 消费者，
// 确定性地验证 Enqueue 的队列语义：入队成功 → 队列大小精确 +1。
// 这是 TestAsyncRetryWorker_Enqueue 中竞态断言的确定性替代。
func TestAsyncRetryWorker_QueueSemantics(t *testing.T) {
	config := DefaultRetryConfig()
	// 不调用 Start()——无消费者，队列大小完全由入队驱动，无竞态
	worker := NewAsyncRetryWorker(config, 2)
	defer worker.Stop()

	memoryCache := NewMemoryCache(100, 5*time.Minute)
	ctx := context.Background()

	before := worker.QueueSize()
	assert.Equal(t, 0, before, "初始队列应为空")

	ok := worker.Enqueue(ctx, memoryCache, "k1", "v1", time.Minute)
	assert.True(t, ok)
	assert.Equal(t, before+1, worker.QueueSize(), "入队1项后队列大小应精确 +1")

	ok = worker.Enqueue(ctx, memoryCache, "k2", "v2", time.Minute)
	assert.True(t, ok)
	assert.Equal(t, before+2, worker.QueueSize(), "入队2项后队列大小应精确 +2")

	// GetStats 的 queue_size 应与 QueueSize 一致（同一时刻读取）
	stats := worker.GetStats()
	statsQueueSize, isInt := stats["queue_size"].(int)
	assert.True(t, isInt)
	assert.Equal(t, worker.QueueSize(), statsQueueSize)
	assert.Equal(t, 2, stats["worker_count"])
}

// TestRetryDelayCalculation 测试重试延迟计算
func TestRetryDelayCalculation(t *testing.T) {
	config := DefaultRetryConfig()

	// 测试退避计算
	currentDelay := config.InitialDelay
	for i := 0; i < config.MaxRetries; i++ {
		expectedMax := config.MaxDelay
		if currentDelay > expectedMax {
			t.Errorf("第%d次重试延迟%.2fms超过最大延迟%.2fms", i+1, float64(currentDelay), float64(expectedMax))
		}
		// 计算下次延迟
		currentDelay = time.Duration(float64(currentDelay) * config.BackoffFactor)
	}
}

// TestRetryStats_SuccessRate 测试成功率计算
func TestRetryStats_SuccessRate(t *testing.T) {
	stats := NewRetryStats()

	// 模拟10次尝试，8次成功，2次失败
	for i := 0; i < 10; i++ {
		stats.RecordAttempt() // 记录尝试
		if i < 8 {
			stats.RecordSuccess() // 前8次成功
		} else {
			stats.RecordFailure(errors.New("test error")) // 后2次失败
		}
	}

	statsMap := stats.GetStats()
	assert.Equal(t, int64(10), statsMap["total_attempts"])
	assert.Equal(t, int64(8), statsMap["total_success"])
	assert.Equal(t, int64(2), statsMap["total_failure"])

	// 验证成功率计算
	successRate, ok := statsMap["success_rate"].(string)
	assert.True(t, ok)
	assert.Contains(t, successRate, "80.00%")
}

// BenchmarkRetryWithoutFailure 基准测试：无失败场景
func BenchmarkRetryWithoutFailure(b *testing.B) {
	config := &RetryConfig{
		MaxRetries:     3,
		InitialDelay:   10 * time.Millisecond,
		MaxDelay:       50 * time.Millisecond,
		BackoffFactor:  2.0,
		RetryableCheck: IsRetryableError,
	}

	fn := func() error {
		return nil // 总是成功
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RetryWithContext(context.Background(), config, fn)
	}
}

// BenchmarkRetryWithOneRetry 基准测试：一次重试后成功
func BenchmarkRetryWithOneRetry(b *testing.B) {
	config := &RetryConfig{
		MaxRetries:     3,
		InitialDelay:   10 * time.Millisecond,
		MaxDelay:       50 * time.Millisecond,
		BackoffFactor:  2.0,
		RetryableCheck: IsRetryableError,
	}

	attempts := 0
	fn := func() error {
		attempts++
		if attempts == 1 {
			return errors.New("connection refused")
		}
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		attempts = 0
		_ = RetryWithContext(context.Background(), config, fn)
	}
}
