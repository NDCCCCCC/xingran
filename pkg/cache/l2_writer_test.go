package cache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockCache 模拟缓存实现，用于测试
type mockCache struct {
	setCount atomic.Int64
	setError error
	delay    time.Duration
}

func (m *mockCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	m.setCount.Add(1)
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	return m.setError
}

func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
	return "", nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *mockCache) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}

func (m *mockCache) MGet(ctx context.Context, keys ...string) ([]string, error) {
	return nil, nil
}

func (m *mockCache) MSet(ctx context.Context, pairs ...interface{}) error {
	return nil
}

func (m *mockCache) MDelete(ctx context.Context, keys ...string) error {
	return nil
}

func (m *mockCache) Increment(ctx context.Context, key string) (int64, error) {
	return 0, nil
}

func (m *mockCache) IncrementBy(ctx context.Context, key string, value int64) (int64, error) {
	return 0, nil
}

func (m *mockCache) Decrement(ctx context.Context, key string) (int64, error) {
	return 0, nil
}

func (m *mockCache) DecrementBy(ctx context.Context, key string, value int64) (int64, error) {
	return 0, nil
}

func (m *mockCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return nil
}

func (m *mockCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, nil
}

func (m *mockCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	return nil, nil
}

func (m *mockCache) FlushDB(ctx context.Context) error {
	return nil
}

func (m *mockCache) Close() error {
	return nil
}

func (m *mockCache) MGetJSON(ctx context.Context, keys ...string) (map[string]interface{}, error) {
	return nil, nil
}

func (m *mockCache) MSetJSON(ctx context.Context, data map[string]interface{}, expiration time.Duration) error {
	return nil
}

func (m *mockCache) HGet(ctx context.Context, key, field string) (string, error) {
	return "", nil
}

func (m *mockCache) HSet(ctx context.Context, key, field string, value interface{}) error {
	return nil
}

func (m *mockCache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return nil, nil
}

func (m *mockCache) HDel(ctx context.Context, key string, fields ...string) error {
	return nil
}

func (m *mockCache) HKeys(ctx context.Context, key string) ([]string, error) {
	return nil, nil
}

func (m *mockCache) HExists(ctx context.Context, key, field string) (bool, error) {
	return false, nil
}

func (m *mockCache) HLen(ctx context.Context, key string) (int64, error) {
	return 0, nil
}

func (m *mockCache) HIncrBy(ctx context.Context, key, field string, value int64) (int64, error) {
	return 0, nil
}

func (m *mockCache) SetInt(ctx context.Context, key string, value int, expiration time.Duration) error {
	return nil
}

func (m *mockCache) GetInt(ctx context.Context, key string) (int, error) {
	return 0, nil
}

func (m *mockCache) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return nil
}

func (m *mockCache) GetJSON(ctx context.Context, key string, dest interface{}) error {
	return nil
}

func (m *mockCache) GetCount() int64 {
	return m.setCount.Load()
}

// TestL2WriteWorker_BasicOperations 测试基本操作
func TestL2WriteWorker_BasicOperations(t *testing.T) {
	cache := &mockCache{}
	config := &L2WriterConfig{
		WorkerCount:          2,
		QueueSize:            100,
		EnqueueTimeout:       100 * time.Millisecond,
		WriteTimeout:         1 * time.Second,
		FallbackWriteTimeout: 500 * time.Millisecond,
	}

	worker := NewL2WriteWorker(config)
	worker.Start()
	defer worker.Stop()

	ctx := context.Background()

	// 测试入队
	err := worker.Enqueue(ctx, cache, "key1", "value1", time.Minute)
	if err != nil {
		t.Fatalf("入队失败: %v", err)
	}

	// 等待处理完成
	time.Sleep(200 * time.Millisecond)

	// 验证统计数据
	stats := worker.GetStats()
	if stats["enqueued"].(int64) != 1 {
		t.Errorf("期望 enqueued=1, 实际=%d", stats["enqueued"])
	}
	if stats["completed"].(int64) != 1 {
		t.Errorf("期望 completed=1, 实际=%d", stats["completed"])
	}

	// 验证缓存被调用
	if cache.GetCount() != 1 {
		t.Errorf("期望缓存Set被调用1次, 实际=%d", cache.GetCount())
	}
}

// TestL2WriteWorker_TryEnqueue 测试非阻塞入队
func TestL2WriteWorker_TryEnqueue(t *testing.T) {
	cache := &mockCache{}
	config := &L2WriterConfig{
		WorkerCount:    1,
		QueueSize:      2, // 小队列
		EnqueueTimeout: 100 * time.Millisecond,
		WriteTimeout:   100 * time.Millisecond,
	}

	worker := NewL2WriteWorker(config)
	worker.Start()
	defer worker.Stop()

	ctx := context.Background()

	// 填满队列
	success := worker.TryEnqueue(ctx, cache, "key1", "value1", time.Minute)
	if !success {
		t.Fatal("第一次入队应该成功")
	}
	success = worker.TryEnqueue(ctx, cache, "key2", "value2", time.Minute)
	if !success {
		t.Fatal("第二次入队应该成功")
	}

	// 队列已满，第三次应该失败（非阻塞）
	// 但由于worker在处理，可能需要快速连续调用
	for i := 0; i < 10; i++ {
		success = worker.TryEnqueue(ctx, cache, "key3", "value3", time.Minute)
		if !success {
			t.Log("队列满时TryEnqueue正确返回false")
			return
		}
	}

	t.Error("队列满时TryEnqueue应该返回false")
}

// TestL2WriteWorker_QueueFull 测试队列满时的行为
func TestL2WriteWorker_QueueFull(t *testing.T) {
	// 创建一个非常慢的缓存，让worker处理变慢，确保队列会满
	slowCache := &mockCache{delay: 500 * time.Millisecond}

	config := &L2WriterConfig{
		WorkerCount:          1,
		QueueSize:            1, // 队列大小为1
		EnqueueTimeout:       100 * time.Millisecond,
		WriteTimeout:         1000 * time.Millisecond,
		FallbackWriteTimeout: 100 * time.Millisecond,
	}

	worker := NewL2WriteWorker(config)
	worker.Start()
	defer worker.Stop()

	ctx := context.Background()

	// 入队一个任务，队列将满（1个在队列中，1个在worker处理）
	_ = worker.Enqueue(ctx, slowCache, "key1", "value1", time.Minute)

	// 稍等一下，让worker开始处理第一个任务
	time.Sleep(50 * time.Millisecond)

	// 快速入队第二个任务，填满队列
	_ = worker.Enqueue(ctx, slowCache, "key2", "value2", time.Minute)

	// 队列已满，第三次入队应该超时
	start := time.Now()
	err := worker.Enqueue(ctx, slowCache, "key3", "value3", time.Minute)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("队列满时入队应该返回错误")
	}
	// 验证等待了合理的超时时间
	if elapsed < 50*time.Millisecond {
		t.Logf("警告: 入队超时时间较短，可能队列未满，耗时=%v", elapsed)
	}
	t.Logf("队列满测试: 耗时=%v, 错误=%v", elapsed, err)
}

// TestL2WriteWorker_TaskSurvivesCallerCancel 回归测试 (login-menu-timeout-20260817 H8):
// 调用方 ctx 在入队成功后取消(模拟 HTTP 请求结束/客户端中止),任务仍必须被执行。
// 历史 bug: MultiLevelCache.Set 用 defer-cancel 的临时 ctx 入队,Set 返回即取消,
// processTask 前置 task.ctx.Done() 检查 100% 丢弃任务 → menu 缓存永远写不进 L2。
// 修复后 buildTask 内部 detach(WithoutCancel + 有界 TTL),任务随调用方 ctx 取消而存活。
func TestL2WriteWorker_TaskSurvivesCallerCancel(t *testing.T) {
	// 慢缓存: 让第一个任务占住唯一 worker,确保第二个任务在队列中等待时 ctx 已取消
	slowCache := &mockCache{delay: 150 * time.Millisecond}
	config := &L2WriterConfig{
		WorkerCount:          1,
		QueueSize:            10,
		EnqueueTimeout:       100 * time.Millisecond,
		WriteTimeout:         500 * time.Millisecond,
		FallbackWriteTimeout: 100 * time.Millisecond,
	}

	worker := NewL2WriteWorker(config)
	worker.Start()
	defer worker.Stop()

	// 任务1: 占住 worker(150ms 处理时间)
	if err := worker.Enqueue(context.Background(), slowCache, "blocker", "v", time.Minute); err != nil {
		t.Fatalf("blocker 入队失败: %v", err)
	}
	time.Sleep(20 * time.Millisecond) // 确保 blocker 已被 worker 取走

	// 任务2: 模拟 MultiLevelCache.Set 的历史行为 — 入队成功后调用方 ctx 立即取消
	ctx, cancel := context.WithCancel(context.Background())
	if err := worker.Enqueue(ctx, slowCache, "menu:user:menus:u1", "v", time.Minute); err != nil {
		t.Fatalf("victim 入队失败: %v", err)
	}
	cancel() // Set 返回 / 客户端中止

	// 等待两个任务都处理完
	time.Sleep(500 * time.Millisecond)

	if got := slowCache.GetCount(); got != 2 {
		t.Errorf("H8 回归: 调用方 ctx 取消后 L2 写入被丢弃, setCount=%d (期望 2)", got)
	}
	stats := worker.GetStats()
	if stats["dropped"].(int64) != 0 {
		t.Errorf("H8 回归: 任务被前置检查丢弃, dropped=%v (期望 0)", stats["dropped"])
	}
	if stats["completed"].(int64) != 2 {
		t.Errorf("期望 completed=2, 实际=%v", stats["completed"])
	}
}

// TestL2WriteWorker_ContextCancellation 测试上下文取消
func TestL2WriteWorker_ContextCancellation(t *testing.T) {
	slowCache := &mockCache{delay: 500 * time.Millisecond}
	config := &L2WriterConfig{
		WorkerCount:          1,
		QueueSize:            1,               // 小队列
		EnqueueTimeout:       5 * time.Second, // 长超时
		WriteTimeout:         1000 * time.Millisecond,
		FallbackWriteTimeout: 100 * time.Millisecond,
	}

	worker := NewL2WriteWorker(config)
	worker.Start()
	defer worker.Stop()

	// 填满队列
	ctx := context.Background()
	_ = worker.Enqueue(ctx, slowCache, "key1", "value1", time.Minute)
	_ = worker.Enqueue(ctx, slowCache, "key2", "value2", time.Minute)

	// 创建会被取消的上下文
	cancelCtx, cancel := context.WithCancel(context.Background())
	// 在另一个 goroutine 中短暂延迟后取消
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// 队列满，应该阻塞直到上下文被取消
	start := time.Now()
	err := worker.Enqueue(cancelCtx, slowCache, "key3", "value3", time.Minute)
	elapsed := time.Since(start)

	// 应该因为上下文取消而失败
	if err == nil {
		t.Error("上下文取消后入队应该返回错误")
	}
	// 验证等待时间合理（约50ms）
	if elapsed < 40*time.Millisecond || elapsed > 200*time.Millisecond {
		t.Logf("警告: 上下文取消时间可能异常，耗时=%v", elapsed)
	}
	t.Logf("上下文取消测试通过: 耗时=%v, 错误=%v", elapsed, err)
}

// TestL2WriteWorker_GracefulShutdown 测试优雅关闭
func TestL2WriteWorker_GracefulShutdown(t *testing.T) {
	cache := &mockCache{}
	config := &L2WriterConfig{
		WorkerCount:          1,
		QueueSize:            10,
		EnqueueTimeout:       100 * time.Millisecond,
		WriteTimeout:         100 * time.Millisecond,
		FallbackWriteTimeout: 100 * time.Millisecond,
	}

	worker := NewL2WriteWorker(config)
	worker.Start()

	ctx := context.Background()

	// 入队多个任务
	for i := 0; i < 5; i++ {
		_ = worker.Enqueue(ctx, cache, "key", "value", time.Minute)
	}

	// 停止worker，应该处理完队列中剩余任务
	worker.Stop()

	stats := worker.GetStats()
	t.Logf("关闭后统计: enqueued=%d, completed=%d, dropped=%d",
		stats["enqueued"], stats["completed"], stats["dropped"])

	// 验证所有任务都被处理（或至少大部分）
	completed := stats["completed"].(int64)
	enqueued := stats["enqueued"].(int64)
	if completed < enqueued-1 { // 允许1个任务在关闭时未处理
		t.Errorf("关闭时应该处理大部分任务, enqueued=%d, completed=%d", enqueued, completed)
	}

	// 验证不能在已停止的worker上入队
	err := worker.Enqueue(ctx, cache, "key", "value", time.Minute)
	if err == nil {
		t.Error("已停止的worker不应接受新任务")
	}
}

// TestL2WriteWorker_ConcurrentAccess 测试并发访问
func TestL2WriteWorker_ConcurrentAccess(t *testing.T) {
	cache := &mockCache{}
	config := &L2WriterConfig{
		WorkerCount:          5,
		QueueSize:            1000,
		EnqueueTimeout:       1 * time.Second,
		WriteTimeout:         100 * time.Millisecond,
		FallbackWriteTimeout: 100 * time.Millisecond,
	}

	worker := NewL2WriteWorker(config)
	worker.Start()
	defer worker.Stop()

	ctx := context.Background()
	numGoroutines := 10
	tasksPerGoroutine := 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < tasksPerGoroutine; j++ {
				key := fmt.Sprintf("goroutine-%d-key-%d", id, j)
				err := worker.Enqueue(ctx, cache, key, "value", time.Minute)
				if err != nil {
					t.Logf("Goroutine %d: 入队失败: %v", id, err)
				}
			}
		}(i)
	}

	wg.Wait()

	// 等待所有任务处理完成
	time.Sleep(2 * time.Second)

	stats := worker.GetStats()
	totalTasks := int64(numGoroutines * tasksPerGoroutine)
	enqueued := stats["enqueued"].(int64)
	completed := stats["completed"].(int64)

	t.Logf("并发测试结果: 总任务=%d, 入队=%d, 完成=%d, 丢弃=%d",
		totalTasks, enqueued, completed, stats["dropped"])

	if enqueued < totalTasks-10 { // 允许少量任务入队失败
		t.Errorf("入队任务数过少, 期望约%d, 实际=%d", totalTasks, enqueued)
	}
}

// TestL2WriteWorker_Statistics 测试统计信息
func TestL2WriteWorker_Statistics(t *testing.T) {
	cache := &mockCache{}
	config := DefaultL2WriterConfig()

	worker := NewL2WriteWorker(config)
	worker.Start()
	defer worker.Stop()

	ctx := context.Background()

	// 执行一些操作
	for i := 0; i < 5; i++ {
		_ = worker.Enqueue(ctx, cache, "key", "value", time.Minute)
	}

	// 等待处理
	time.Sleep(500 * time.Millisecond)

	stats := worker.GetStats()

	// 验证统计字段存在
	requiredFields := []string{"enqueued", "completed", "dropped", "failed", "avg_latency_ms", "queue_depth", "is_running"}
	for _, field := range requiredFields {
		if _, ok := stats[field]; !ok {
			t.Errorf("统计信息缺少字段: %s", field)
		}
	}

	// 验证运行状态
	if !stats["is_running"].(bool) {
		t.Error("worker应该处于运行状态")
	}

	t.Logf("统计信息: %+v", stats)
}

// TestL2WriteWorker_DefaultConfig 测试默认配置
func TestL2WriteWorker_DefaultConfig(t *testing.T) {
	config := DefaultL2WriterConfig()

	if config.WorkerCount != DefaultWorkerCount {
		t.Errorf("默认WorkerCount错误: 期望=%d, 实际=%d", DefaultWorkerCount, config.WorkerCount)
	}
	if config.QueueSize != DefaultQueueSize {
		t.Errorf("默认QueueSize错误: 期望=%d, 实际=%d", DefaultQueueSize, config.QueueSize)
	}
	if config.EnqueueTimeout != DefaultEnqueueTimeout {
		t.Errorf("默认EnqueueTimeout错误: 期望=%v, 实际=%v", DefaultEnqueueTimeout, config.EnqueueTimeout)
	}
	if config.WriteTimeout != DefaultL2WriteTimeout {
		t.Errorf("默认WriteTimeout错误: 期望=%v, 实际=%v", DefaultL2WriteTimeout, config.WriteTimeout)
	}
	if config.FallbackWriteTimeout != DefaultFallbackWriteTimeout {
		t.Errorf("默认FallbackWriteTimeout错误: 期望=%v, 实际=%v", DefaultFallbackWriteTimeout, config.FallbackWriteTimeout)
	}
}

// TestL2WriteWorker_ConfigNormalization 测试配置规范化
func TestL2WriteWorker_ConfigNormalization(t *testing.T) {
	tests := []struct {
		name  string
		input *L2WriterConfig
	}{
		{
			name: "零值配置",
			input: &L2WriterConfig{
				WorkerCount:          0,
				QueueSize:            0,
				EnqueueTimeout:       0,
				WriteTimeout:         0,
				FallbackWriteTimeout: 0,
			},
		},
		{
			name: "负值配置",
			input: &L2WriterConfig{
				WorkerCount:          -1,
				QueueSize:            -1,
				EnqueueTimeout:       -1,
				WriteTimeout:         -1,
				FallbackWriteTimeout: -1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// normalize应该修改配置并返回
			normalized := tt.input.normalize()

			if normalized.WorkerCount <= 0 {
				t.Error("WorkerCount应该被规范化为正数")
			}
			if normalized.QueueSize <= 0 {
				t.Error("QueueSize应该被规范化为正数")
			}
			if normalized.EnqueueTimeout <= 0 {
				t.Error("EnqueueTimeout应该被规范化为正数")
			}
			if normalized.WriteTimeout <= 0 {
				t.Error("WriteTimeout应该被规范化为正数")
			}
			if normalized.FallbackWriteTimeout <= 0 {
				t.Error("FallbackWriteTimeout应该被规范化为正数")
			}
		})
	}
}

// TestL2WriteWorker_GetFallbackTimeout 测试获取降级超时
func TestL2WriteWorker_GetFallbackTimeout(t *testing.T) {
	tests := []struct {
		name            string
		config          *L2WriterConfig
		expectedTimeout time.Duration
	}{
		{
			name: "自定义配置",
			config: &L2WriterConfig{
				FallbackWriteTimeout: 2 * time.Second,
			},
			expectedTimeout: 2 * time.Second,
		},
		{
			name:            "nil配置",
			config:          nil,
			expectedTimeout: DefaultFallbackWriteTimeout,
		},
		{
			name: "零值配置",
			config: &L2WriterConfig{
				FallbackWriteTimeout: 0,
			},
			expectedTimeout: DefaultFallbackWriteTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worker := NewL2WriteWorker(tt.config)
			timeout := worker.GetFallbackTimeout()

			if timeout != tt.expectedTimeout {
				t.Errorf("期望FallbackWriteTimeout=%v, 实际=%v", tt.expectedTimeout, timeout)
			}
		})
	}
}

// TestMultiLevelCache_L2WriterIntegration 测试MultiLevelCache与L2Writer集成
func TestMultiLevelCache_L2WriterIntegration(t *testing.T) {
	// 使用内存缓存作为L1，避免依赖Redis
	l1Cache := NewMemoryCache(100, 1*time.Minute)
	l2Cache := &mockCache{}

	config := &L2WriterConfig{
		WorkerCount:          2,
		QueueSize:            100,
		EnqueueTimeout:       100 * time.Millisecond,
		WriteTimeout:         500 * time.Millisecond,
		FallbackWriteTimeout: 200 * time.Millisecond,
	}

	mlCache := NewMultiLevelCacheWithWriter(l1Cache, l2Cache, config)
	defer mlCache.Close()

	ctx := context.Background()

	// 测试Set操作
	err := mlCache.Set(ctx, "key1", "value1", time.Minute)
	if err != nil {
		t.Fatalf("Set失败: %v", err)
	}

	// 等待L2写入完成
	time.Sleep(300 * time.Millisecond)

	// 验证L2Writer统计
	stats := mlCache.GetL2WriterStats()
	if stats == nil {
		t.Fatal("L2Writer统计不应为nil")
	}

	t.Logf("L2Writer统计: %+v", stats)

	// 验证队列大小
	queueSize := mlCache.GetL2WriterQueueSize()
	t.Logf("当前队列大小: %d", queueSize)

	// 验证L2Writer启用状态
	if !mlCache.IsL2WriterEnabled() {
		t.Error("L2Writer应该被启用")
	}
}
