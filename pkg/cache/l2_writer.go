package cache

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xingran-next/xingran-go-backend/pkg/logger"
)

const (
	// DefaultWorkerCount 默认工作goroutine数量
	DefaultWorkerCount = 5
	// DefaultQueueSize 默认队列大小
	DefaultQueueSize = 5000
	// DefaultEnqueueTimeout 默认入队超时时间
	DefaultEnqueueTimeout = 1 * time.Second
	// DefaultL2WriteTimeout 默认L2写入超时时间
	DefaultL2WriteTimeout = 5 * time.Second
	// DefaultFallbackWriteTimeout 默认同步降级超时时间
	DefaultFallbackWriteTimeout = 3 * time.Second
)

// L2WriterConfig L2写入Worker配置
type L2WriterConfig struct {
	WorkerCount          int           // 工作goroutine数量
	QueueSize            int           // 队列大小
	EnqueueTimeout       time.Duration // 入队超时
	WriteTimeout         time.Duration // L2写入超时
	FallbackWriteTimeout time.Duration // 入队失败时同步降级超时（保证数据一致性）
}

// DefaultL2WriterConfig 返回默认的L2写入Worker配置
func DefaultL2WriterConfig() *L2WriterConfig {
	return &L2WriterConfig{
		WorkerCount:          DefaultWorkerCount,
		QueueSize:            DefaultQueueSize,
		EnqueueTimeout:       DefaultEnqueueTimeout,
		WriteTimeout:         DefaultL2WriteTimeout,
		FallbackWriteTimeout: DefaultFallbackWriteTimeout,
	}
}

// normalize 规范化配置，确保参数在合理范围内
func (c *L2WriterConfig) normalize() *L2WriterConfig {
	if c.WorkerCount <= 0 {
		c.WorkerCount = DefaultWorkerCount
	}
	if c.QueueSize <= 0 {
		c.QueueSize = DefaultQueueSize
	}
	if c.EnqueueTimeout <= 0 {
		c.EnqueueTimeout = DefaultEnqueueTimeout
	}
	if c.WriteTimeout <= 0 {
		c.WriteTimeout = DefaultL2WriteTimeout
	}
	if c.FallbackWriteTimeout <= 0 {
		c.FallbackWriteTimeout = DefaultFallbackWriteTimeout
	}
	return c
}

// L2WriterStats L2写入Worker统计信息
type L2WriterStats struct {
	enqueued     atomic.Int64 // 已入队任务数
	completed    atomic.Int64 // 已完成任务数
	dropped      atomic.Int64 // 已丢弃任务数
	failed       atomic.Int64 // 失败任务数
	totalLatency atomic.Int64 // 总延迟(纳秒)
}

// Enqueued 获取已入队任务数
func (s *L2WriterStats) Enqueued() int64 {
	return s.enqueued.Load()
}

// Completed 获取已完成任务数
func (s *L2WriterStats) Completed() int64 {
	return s.completed.Load()
}

// Dropped 获取已丢弃任务数
func (s *L2WriterStats) Dropped() int64 {
	return s.dropped.Load()
}

// Failed 获取失败任务数
func (s *L2WriterStats) Failed() int64 {
	return s.failed.Load()
}

// AvgLatency 获取平均延迟(毫秒)
func (s *L2WriterStats) AvgLatency() float64 {
	total := s.totalLatency.Load()
	count := s.completed.Load()
	if count == 0 {
		return 0
	}
	return float64(total) / float64(count) / 1_000_000 // 转换为毫秒
}

// Snapshot 获取统计快照，返回map格式以兼容现有监控接口
func (s *L2WriterStats) Snapshot() map[string]interface{} {
	return map[string]interface{}{
		"enqueued":       s.Enqueued(),
		"completed":      s.Completed(),
		"dropped":        s.Dropped(),
		"failed":         s.Failed(),
		"avg_latency_ms": s.AvgLatency(),
	}
}

type l2WriteTask struct {
	ctx         context.Context
	cancel      context.CancelFunc // 释放 detached ctx 的 timer;入队失败或 processTask 完成时必须调用(防泄漏)
	key         string
	value       interface{}
	expiration  time.Duration
	cache       Cache
	enqueueTime time.Time // 入队时间，用于计算延迟
}

// L2WriteWorker L2缓存写入Worker Pool
// 使用固定数量的goroutine处理L2缓存写入，避免每次Set操作创建新goroutine
type L2WriteWorker struct {
	config    *L2WriterConfig
	stats     *L2WriterStats
	workQueue chan l2WriteTask
	closeChan chan struct{}
	wg        sync.WaitGroup
	running   atomic.Bool
}

// NewL2WriteWorker 创建L2写入Worker
func NewL2WriteWorker(config *L2WriterConfig) *L2WriteWorker {
	if config == nil {
		config = DefaultL2WriterConfig()
	}

	// 创建配置副本，避免修改原始配置
	normalized := config.normalize()

	return &L2WriteWorker{
		config:    normalized,
		stats:     &L2WriterStats{},
		workQueue: make(chan l2WriteTask, normalized.QueueSize),
		closeChan: make(chan struct{}),
	}
}

// Start 启动L2写入Worker
func (w *L2WriteWorker) Start() {
	if !w.running.CompareAndSwap(false, true) {
		logger.Warnf("[L2WriteWorker] 已经在运行中")
		return
	}

	for i := 0; i < w.config.WorkerCount; i++ {
		w.wg.Add(1)
		go w.worker(i)
	}

	logger.Infof("[L2WriteWorker] 已启动，worker数量=%d，队列大小=%d",
		w.config.WorkerCount, w.config.QueueSize)
}

// Stop 优雅停止L2写入Worker，处理完队列中剩余任务
func (w *L2WriteWorker) Stop() {
	if !w.running.CompareAndSwap(true, false) {
		return
	}

	logger.Infof("[L2WriteWorker] 正在停止...")

	// 关闭入队通道，不再接受新任务
	close(w.workQueue)

	// 等待所有worker完成
	w.wg.Wait()

	logger.Infof("[L2WriteWorker] 已停止，统计: %v", w.stats.Snapshot())
}

// Enqueue 入队L2写入任务（阻塞模式，带超时）
func (w *L2WriteWorker) Enqueue(ctx context.Context, cache Cache, key string, value interface{}, expiration time.Duration) error {
	if !w.running.Load() {
		return fmt.Errorf("L2WriteWorker未运行")
	}

	task, err := w.buildTask(ctx, cache, key, value, expiration)
	if err != nil {
		return fmt.Errorf("构建任务失败: %w", err)
	}

	w.stats.enqueued.Add(1)

	// 创建timer并确保清理，避免goroutine泄漏
	timer := time.NewTimer(w.config.EnqueueTimeout)
	defer timer.Stop()

	select {
	case w.workQueue <- task:
		return nil
	case <-timer.C:
		w.stats.dropped.Add(1)
		task.cancel() // 任务未入队,释放 detached ctx
		return fmt.Errorf("L2写入队列已满，入队超时")
	case <-ctx.Done():
		w.stats.dropped.Add(1)
		task.cancel() // 任务未入队,释放 detached ctx
		return fmt.Errorf("L2写入入队被取消: %w", ctx.Err())
	}
}

// TryEnqueue 尝试入队L2写入任务（非阻塞模式）
func (w *L2WriteWorker) TryEnqueue(ctx context.Context, cache Cache, key string, value interface{}, expiration time.Duration) bool {
	if !w.running.Load() {
		return false
	}

	task, err := w.buildTask(ctx, cache, key, value, expiration)
	if err != nil {
		return false
	}

	// 统一在入队前增加计数（与Enqueue保持一致）
	w.stats.enqueued.Add(1)

	select {
	case w.workQueue <- task:
		return true
	default:
		w.stats.dropped.Add(1)
		task.cancel() // 任务未入队,释放 detached ctx
		return false
	}
}

// buildTask 构建写入任务（提取公共逻辑，消除DRY违规）
//
// ctx 解耦 (login-menu-timeout-20260817 H8 修复):
//   L2 写入是后台任务,必须独立于调用方(通常是 HTTP 请求) ctx 的生命周期。
//   历史上 MultiLevelCache.Set 曾传入 defer-cancel 的临时 ctx,Set 返回即取消,
//   processTask 前置 task.ctx.Done() 检查 100% 命中 → 几乎所有 L2 异步写被丢弃,
//   menu:user:* 缓存永远写不进 Redis,形成"缓存永久 miss → 重复慢查询 → 超时 →
//   客户端中止 → 写入再被取消"的恶性循环。
//   这里统一 detach: context.WithoutCancel 保留 values(tracing) 但剥离取消信号,
//   并加上有界存活期(队列积压过久视为过期写入,由 processTask 前置检查丢弃)。
//   cancel 存于任务上,由 processTask 在处理完成(成功/失败/丢弃)时调用,
//   或在 Enqueue/TryEnqueue 入队失败路径调用 — 不会泄漏。
func (w *L2WriteWorker) buildTask(ctx context.Context, cache Cache, key string, value interface{}, expiration time.Duration) (l2WriteTask, error) {
	taskCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), w.taskQueueTTL())
	return l2WriteTask{
		ctx:         taskCtx,
		cancel:      cancel,
		key:         key,
		value:       value,
		expiration:  expiration,
		cache:       cache,
		enqueueTime: time.Now(),
	}, nil
}

// taskQueueTTL 返回任务在队列中的最大存活时间。过期任务被 processTask 视为陈旧写入丢弃。
// 取 WriteTimeout 的 6 倍且不少于 30s: 容忍正常队列积压,同时保证过期写最终被淘汰。
// 注意这只管"值不值得写";写入本身始终由 processTask 的 WriteTimeout 单独约束。
func (w *L2WriteWorker) taskQueueTTL() time.Duration {
	ttl := w.config.WriteTimeout * 6
	if ttl < 30*time.Second {
		ttl = 30 * time.Second
	}
	return ttl
}

// worker 工作goroutine
func (w *L2WriteWorker) worker(id int) {
	defer w.wg.Done()

	for {
		select {
		case <-w.closeChan:
			// 处理队列中剩余任务
			w.drainQueue(id)
			logger.Debugf("[L2WriteWorker] Worker-%d 退出", id)
			return
		case task, ok := <-w.workQueue:
			if !ok {
				// 队列已关闭，退出
				logger.Debugf("[L2WriteWorker] Worker-%d 退出", id)
				return
			}
			w.processTask(id, task)
		}
	}
}

// drainQueue 处理队列中剩余任务
func (w *L2WriteWorker) drainQueue(workerID int) {
	// 使用非阻塞方式处理剩余任务
	for {
		select {
		case task, ok := <-w.workQueue:
			if !ok {
				return
			}
			w.processTask(workerID, task)
		default:
			// 队列已空
			return
		}
	}
}

// processTask 处理单个L2写入任务
func (w *L2WriteWorker) processTask(workerID int, task l2WriteTask) {
	// 释放 detached ctx 的 timer(buildTask 创建);成功/失败/丢弃所有出口统一在此释放
	if task.cancel != nil {
		defer task.cancel()
	}

	// 检查context是否已取消（任务在队列中滞留超过 taskQueueTTL,视为陈旧写入）
	select {
	case <-task.ctx.Done():
		w.stats.dropped.Add(1)
		logger.Debugf("[L2WriteWorker] Worker-%d: 任务已取消，跳过 key=%s", workerID, task.key)
		return
	default:
	}

	startTime := time.Now()

	// 使用带超时的context执行L2写入
	// 防止Redis连接问题导致worker永久阻塞
	writeCtx, cancel := context.WithTimeout(context.Background(), w.config.WriteTimeout)
	defer cancel()

	err := task.cache.Set(writeCtx, task.key, task.value, task.expiration)

	// 记录延迟
	latency := time.Since(startTime)
	w.stats.totalLatency.Add(latency.Nanoseconds())

	if err != nil {
		w.stats.failed.Add(1)
		logger.Warnf("[L2WriteWorker] Worker-%d: L2写入失败 key=%s, error=%v",
			workerID, task.key, err)
	} else {
		w.stats.completed.Add(1)
		logger.Debugf("[L2WriteWorker] Worker-%d: L2写入成功 key=%s, latency=%v",
			workerID, task.key, latency)
	}
}

// GetStats 获取统计信息（返回map格式以兼容现有监控接口）
func (w *L2WriteWorker) GetStats() map[string]interface{} {
	stats := w.stats.Snapshot()
	stats["queue_depth"] = w.QueueSize()
	stats["is_running"] = w.IsRunning()
	return stats
}

// QueueSize 获取当前队列大小（与AsyncRetryWorker保持命名一致）
func (w *L2WriteWorker) QueueSize() int {
	return len(w.workQueue)
}

// IsRunning 检查是否正在运行
func (w *L2WriteWorker) IsRunning() bool {
	return w.running.Load()
}

// GetFallbackTimeout 获取同步降级超时时间
func (w *L2WriteWorker) GetFallbackTimeout() time.Duration {
	if w.config != nil && w.config.FallbackWriteTimeout > 0 {
		return w.config.FallbackWriteTimeout
	}
	return DefaultFallbackWriteTimeout
}
