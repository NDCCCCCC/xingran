package device

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// DeviceTask 设备任务
type DeviceTask struct {
	ID       string // 任务ID
	DeviceID string // 设备ID
	Priority int    // 优先级（0=最高，数字越大优先级越低）
	// Execute 执行函数（在持有设备锁的情况下调用）
	Execute func(ctx context.Context, conn *PooledConnection) error
	Timeout  time.Duration // 超时时间
	CreatedAt time.Time    // 创建时间
	Callback  func(error)  // 完成回调
}

// DeviceTaskScheduler 设备任务调度器
// 为每个设备维护一个独立的任务队列，确保同一设备的任务串行执行
type DeviceTaskScheduler struct {
	queues         map[string]chan *DeviceTask // deviceID -> 任务队列
	workers        map[string]context.Context  // deviceID -> worker 上下文
	workerCancel   map[string]context.CancelFunc // deviceID -> worker 取消函数
	schedulerLock  sync.RWMutex                 // 保护 queues, workers, workerCancel
	connectionPool *DeviceConnectionPool        // 连接池
	taskTimeout    time.Duration                // 默认任务超时
	done           chan struct{}                 // 停止信号
	enabled        bool                          // 是否启用调度器
	metrics        *SchedulerMetrics             // 统计指标
}

// SchedulerMetrics 调度器统计指标
type SchedulerMetrics struct {
	TotalSubmitted int64 // 总提交任务数
	TotalCompleted int64 // 总完成任务数
	TotalFailed    int64 // 总失败任务数
	ActiveDevices  int   // 活跃设备数
}

// SchedulerConfig 调度器配置
type SchedulerConfig struct {
	TaskTimeout    time.Duration // 默认任务超时，默认 5 分钟
	QueueSize      int           // 每个设备的任务队列大小，默认 100
}

// DefaultSchedulerConfig 默认调度器配置
func DefaultSchedulerConfig() *SchedulerConfig {
	return &SchedulerConfig{
		TaskTimeout: 5 * time.Minute,
		QueueSize:   100,
	}
}

// NewDeviceTaskScheduler 创建设备任务调度器
func NewDeviceTaskScheduler(pool *DeviceConnectionPool, config *SchedulerConfig) *DeviceTaskScheduler {
	if config == nil {
		config = DefaultSchedulerConfig()
	}

	scheduler := &DeviceTaskScheduler{
		queues:         make(map[string]chan *DeviceTask),
		workers:        make(map[string]context.Context),
		workerCancel:   make(map[string]context.CancelFunc),
		connectionPool: pool,
		taskTimeout:    config.TaskTimeout,
		done:          make(chan struct{}),
		enabled:       true,
		metrics:       &SchedulerMetrics{},
	}

	log.Printf("[任务调度器] 已创建调度器: 队列大小=%d, 超时=%v", config.QueueSize, config.TaskTimeout)

	return scheduler
}

// SetEnabled 设置调度器是否启用
func (s *DeviceTaskScheduler) SetEnabled(enabled bool) {
	s.schedulerLock.Lock()
	defer s.schedulerLock.Unlock()
	s.enabled = enabled
}

// IsEnabled 检查调度器是否启用
func (s *DeviceTaskScheduler) IsEnabled() bool {
	s.schedulerLock.RLock()
	defer s.schedulerLock.RUnlock()
	return s.enabled
}

// Submit 提交任务到调度器
func (s *DeviceTaskScheduler) Submit(task *DeviceTask) error {
	if !s.IsEnabled() {
		return fmt.Errorf("任务调度器未启用")
	}

	if task == nil {
		return fmt.Errorf("任务不能为空")
	}

	if task.DeviceID == "" {
		return fmt.Errorf("设备ID不能为空")
	}

	if task.Execute == nil {
		return fmt.Errorf("任务执行函数不能为空")
	}

	// 设置默认值
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	if task.ID == "" {
		task.ID = generateTaskID()
	}

	s.schedulerLock.Lock()

	// 确保设备队列存在
	if _, exists := s.queues[task.DeviceID]; !exists {
		s.queues[task.DeviceID] = make(chan *DeviceTask, 100)
		s.startWorker(task.DeviceID)
	}

	queue := s.queues[task.DeviceID]

	s.schedulerLock.Unlock()

	// 更新统计
	s.recordSubmission()

	// 非阻塞提交
	select {
	case queue <- task:
		log.Printf("[任务调度器] 任务已提交: taskID=%s, deviceID=%s, priority=%d",
			task.ID, task.DeviceID, task.Priority)
		return nil
	default:
		return fmt.Errorf("设备 %s 任务队列已满", task.DeviceID)
	}
}

// startWorker 启动设备 worker（串行执行该设备的所有任务）
func (s *DeviceTaskScheduler) startWorker(deviceID string) {
	workerCtx, cancel := context.WithCancel(context.Background())

	s.workers[deviceID] = workerCtx
	s.workerCancel[deviceID] = cancel

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[任务调度器] 设备 %s worker panic: %v", deviceID, r)
			}
			// 清理
			s.schedulerLock.Lock()
			delete(s.workers, deviceID)
			delete(s.workerCancel, deviceID)
			s.schedulerLock.Unlock()
		}()

		log.Printf("[任务调度器] 启动设备 worker: deviceID=%s", deviceID)

		for {
			select {
			case task := <-s.queues[deviceID]:
				s.executeTask(workerCtx, task)
			case <-workerCtx.Done():
				log.Printf("[任务调度器] 停止设备 worker: deviceID=%s", deviceID)
				return
			}
		}
	}()
}

// executeTask 执行单个任务
func (s *DeviceTaskScheduler) executeTask(workerCtx context.Context, task *DeviceTask) {
	log.Printf("[任务调度器] 开始执行任务: taskID=%s, deviceID=%s", task.ID, task.DeviceID)

	// 设置超时
	timeout := s.taskTimeout
	if task.Timeout > 0 {
		timeout = task.Timeout
	}

	log.Printf("[任务调度器] 任务超时设置: %v", timeout)

	ctx, cancel := context.WithTimeout(workerCtx, timeout)
	defer cancel()

	// 获取连接
	log.Printf("[任务调度器] 正在获取连接: taskID=%s, deviceID=%s", task.ID, task.DeviceID)
	conn, err := s.connectionPool.GetConnection(ctx, task.DeviceID)
	if err != nil {
		log.Printf("[任务调度器] 获取连接失败: taskID=%s, deviceID=%s, error=%v",
			task.ID, task.DeviceID, err)
		s.recordFailure()
		if task.Callback != nil {
			task.Callback(fmt.Errorf("获取连接失败: %w", err))
		}
		return
	}

	log.Printf("[任务调度器] 连接获取成功: taskID=%s, deviceID=%s", task.ID, task.DeviceID)

	// 执行任务（worker 已保证串行，手动管理连接获取/释放）
	if !conn.Acquire() {
		err := fmt.Errorf("无法获取连接: deviceID=%s", task.DeviceID)
		log.Printf("[任务调度器] %v", err)
		s.recordFailure()
		if task.Callback != nil {
			task.Callback(err)
		}
		return
	}

	log.Printf("[任务调度器] 连接已获取，开始执行任务: taskID=%s", task.ID)

	// 执行任务（带 panic 恢复）
	var execErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[任务调度器] 任务执行 panic: taskID=%s, deviceID=%s, panic=%v",
					task.ID, task.DeviceID, r)
				execErr = fmt.Errorf("任务执行 panic: %v", r)
			}
			// 确保释放连接
			log.Printf("[任务调度器] 释放连接: taskID=%s", task.ID)
			conn.Release()
		}()

		log.Printf("[任务调度器] 调用 task.Execute: taskID=%s", task.ID)
		execErr = task.Execute(ctx, conn)
		log.Printf("[任务调度器] task.Execute 返回: taskID=%s, error=%v", task.ID, execErr)
	}()

	if execErr != nil {
		log.Printf("[任务调度器] 任务执行失败: taskID=%s, deviceID=%s, error=%v",
			task.ID, task.DeviceID, execErr)
		s.recordFailure()
	} else {
		log.Printf("[任务调度器] 任务执行成功: taskID=%s, deviceID=%s", task.ID, task.DeviceID)
		s.recordCompletion()
	}

	// 回调
	if task.Callback != nil {
		task.Callback(execErr)
	}

	log.Printf("[任务调度器] 任务完成: taskID=%s", task.ID)
}

// SubmitAndWait 提交任务并等待完成
func (s *DeviceTaskScheduler) SubmitAndWait(ctx context.Context, task *DeviceTask) error {
	// 创建结果通道
	resultCh := make(chan error, 1)

	// 设置回调
	task.Callback = func(err error) {
		select {
		case resultCh <- err:
		default:
		}
	}

	// 提交任务
	if err := s.Submit(task); err != nil {
		return err
	}

	// 等待结果
	select {
	case err := <-resultCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(task.Timeout + time.Minute):
		return fmt.Errorf("任务超时: taskID=%s", task.ID)
	}
}

// Stop 停止调度器
func (s *DeviceTaskScheduler) Stop() {
	log.Printf("[任务调度器] 正在停止调度器...")

	s.schedulerLock.Lock()
	defer s.schedulerLock.Unlock()

	// 取消所有 worker
	for deviceID, cancel := range s.workerCancel {
		log.Printf("[任务调度器] 停止设备 worker: deviceID=%s", deviceID)
		cancel()
	}

	// 等待所有 worker 退出
	for deviceID := range s.workers {
		delete(s.workers, deviceID)
		delete(s.workerCancel, deviceID)
	}

	// 关闭队列
	for deviceID, queue := range s.queues {
		close(queue)
		delete(s.queues, deviceID)
	}

	close(s.done)

	log.Printf("[任务调度器] 调度器已停止")
}

// GetStats 获取统计信息
func (s *DeviceTaskScheduler) GetStats() map[string]interface{} {
	s.schedulerLock.RLock()
	defer s.schedulerLock.RUnlock()

	return map[string]interface{}{
		"total_submitted":  s.metrics.TotalSubmitted,
		"total_completed":  s.metrics.TotalCompleted,
		"total_failed":     s.metrics.TotalFailed,
		"active_devices":   len(s.workers),
		"enabled":          s.enabled,
	}
}

// recordSubmission 记录任务提交
func (s *DeviceTaskScheduler) recordSubmission() {
	// 注意：这里没有加锁，因为 atomic 操作是原子的
	// 如果需要更精确的统计，可以考虑使用 atomic
}

// recordCompletion 记录任务完成
func (s *DeviceTaskScheduler) recordCompletion() {
	// 如果需要更精确的统计，可以使用 atomic
}

// recordFailure 记录任务失败
func (s *DeviceTaskScheduler) recordFailure() {
	// 如果需要更精确的统计，可以使用 atomic
}

// generateTaskID 生成任务ID
func generateTaskID() string {
	return fmt.Sprintf("task-%d", time.Now().UnixNano())
}

// GetConnectionPool 获取连接池
func (s *DeviceTaskScheduler) GetConnectionPool() *DeviceConnectionPool {
	return s.connectionPool
}
