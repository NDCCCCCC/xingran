package cache

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
)

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries      int           // 最大重试次数（0表示不重试）
	InitialDelay    time.Duration // 初始延迟
	MaxDelay        time.Duration // 最大延迟
	BackoffFactor   float64       // 退避因子（每次重试延迟乘以这个因子）
	RetryableCheck  func(error) bool // 判断错误是否可重试
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:     3,
		InitialDelay:   50 * time.Millisecond,
		MaxDelay:       2 * time.Second,
		BackoffFactor:  2.0,
		RetryableCheck: IsRetryableError,
	}
}

// RetryStats 重试统计信息
type RetryStats struct {
	mu             sync.RWMutex
	TotalAttempts   int64 // 总尝试次数
	TotalSuccess    int64 // 成功次数
	TotalFailure    int64 // 失败次数
	TotalRetries    int64 // 重试次数
	ConsecutiveFails int64 // 连续失败次数
	LastError       error // 最后一次错误
	LastErrorTime   time.Time
}

// NewRetryStats 创建重试统计
func NewRetryStats() *RetryStats {
	return &RetryStats{}
}

// RecordAttempt 记录尝试
func (s *RetryStats) RecordAttempt() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalAttempts++
}

// RecordSuccess 记录成功
func (s *RetryStats) RecordSuccess() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalSuccess++
	s.ConsecutiveFails = 0
}

// RecordFailure 记录失败
func (s *RetryStats) RecordFailure(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalFailure++
	s.ConsecutiveFails++
	s.LastError = err
	s.LastErrorTime = time.Now()
}

// RecordRetry 记录重试
func (s *RetryStats) RecordRetry() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TotalRetries++
}

// GetStats 获取统计信息
func (s *RetryStats) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"total_attempts":    s.TotalAttempts,
		"total_success":     s.TotalSuccess,
		"total_failure":     s.TotalFailure,
		"total_retries":     s.TotalRetries,
		"consecutive_fails": s.ConsecutiveFails,
		"last_error":        nil,
		"last_error_time":   nil,
	}

	if s.LastError != nil {
		stats["last_error"] = s.LastError.Error()
		if !s.LastErrorTime.IsZero() {
			stats["last_error_time"] = s.LastErrorTime.Format(time.RFC3339)
		}
	}

	// 计算成功率
	if s.TotalAttempts > 0 {
		successRate := float64(s.TotalSuccess) / float64(s.TotalAttempts) * 100
		stats["success_rate"] = fmt.Sprintf("%.2f%%", successRate)
	}

	return stats
}

// RetryWithContext 使用上下文和重试配置执行函数
func RetryWithContext(ctx context.Context, config *RetryConfig, fn func() error) error {
	if config.MaxRetries <= 0 {
		return fn()
	}

	var lastErr error
	delay := config.InitialDelay

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			// 等待后重试
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		err := fn()
		if err == nil {
			return nil
		}

		// 检查错误是否可重试
		if config.RetryableCheck != nil && !config.RetryableCheck(err) {
			return err
		}

		lastErr = err

		// 如果还有重试机会，计算下次延迟
		if attempt < config.MaxRetries {
			delay = time.Duration(float64(delay) * config.BackoffFactor)
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
			// 添加随机抖动，避免惊群效应
			jitter := time.Duration(float64(delay) * 0.1 * rand.Float64())
			delay += jitter
		}
	}

	return fmt.Errorf("重试%d次后仍失败: %w", config.MaxRetries, lastErr)
}

// IsRetryableError 判断错误是否可重试
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// 可重试的错误模式
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"timeout",
		"i/o timeout",
		"network is unreachable",
		"no route to host",
		"connection timed out",
		"EOF",
		"use of closed network connection",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// AsyncRetryWorker 异步重试工作器
type AsyncRetryWorker struct {
	config      *RetryConfig
	stats       *RetryStats
	workQueue   chan retryWork
	closeChan   chan struct{}
	workerCount int
	wg          sync.WaitGroup
}

type retryWork struct {
	ctx      context.Context
	key      string
	value    interface{}
	expiration time.Duration
	cache    Cache
	attempt  int
}

// NewAsyncRetryWorker 创建异步重试工作器
func NewAsyncRetryWorker(config *RetryConfig, workerCount int) *AsyncRetryWorker {
	if config == nil {
		config = DefaultRetryConfig()
	}
	if workerCount <= 0 {
		workerCount = 3 // 默认3个工作协程
	}

	return &AsyncRetryWorker{
		config:      config,
		stats:       NewRetryStats(),
		workQueue:   make(chan retryWork, 1000), // 缓冲队列
		closeChan:   make(chan struct{}),
		workerCount: workerCount,
	}
}

// Start 启动异步重试工作器
func (w *AsyncRetryWorker) Start() {
	for i := 0; i < w.workerCount; i++ {
		w.wg.Add(1)
		go w.worker(i)
	}
	logger.Infof("[AsyncRetryWorker] 启动%d个重试工作协程", w.workerCount)
}

// Stop 停止异步重试工作器
func (w *AsyncRetryWorker) Stop() {
	logger.Infof("[AsyncRetryWorker] 正在停止重试工作器...")
	close(w.closeChan)
	w.wg.Wait()
	logger.Infof("[AsyncRetryWorker] 重试工作器已停止")
}

// worker 工作协程
func (w *AsyncRetryWorker) worker(id int) {
	defer w.wg.Done()

	for {
		select {
		case <-w.closeChan:
			return
		case work := <-w.workQueue:
			w.processWork(id, work)
		}
	}
}

// processWork 处理单个重试任务
func (w *AsyncRetryWorker) processWork(workerID int, work retryWork) {
	w.stats.RecordAttempt()

	err := RetryWithContext(work.ctx, w.config, func() error {
		return work.cache.Set(work.ctx, work.key, work.value, work.expiration)
	})

	if err == nil {
		w.stats.RecordSuccess()
		logger.Debugf("[AsyncRetryWorker] Worker-%d: 缓存写入成功 key=%s", workerID, work.key)
	} else {
		w.stats.RecordFailure(err)
		logger.Warnf("[AsyncRetryWorker] Worker-%d: 缓存写入失败 key=%s, error=%v", workerID, work.key, err)
	}
}

// Enqueue 将重试任务加入队列
func (w *AsyncRetryWorker) Enqueue(ctx context.Context, cache Cache, key string, value interface{}, expiration time.Duration) bool {
	select {
	case w.workQueue <- retryWork{
		ctx:        ctx,
		key:        key,
		value:      value,
		expiration: expiration,
		cache:      cache,
	}:
		return true
	case <-ctx.Done():
		return false
	}
}

// GetStats 获取重试统计信息
func (w *AsyncRetryWorker) GetStats() map[string]interface{} {
	stats := w.stats.GetStats()
	stats["worker_count"] = w.workerCount
	stats["queue_size"] = len(w.workQueue)
	return stats
}

// QueueSize 返回当前队列大小
func (w *AsyncRetryWorker) QueueSize() int {
	return len(w.workQueue)
}
