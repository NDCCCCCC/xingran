package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/xingran-next/rpa-worker/internal/browser"
	"github.com/xingran-next/rpa-worker/internal/communication"
	"github.com/xingran-next/rpa-worker/internal/config"
	"github.com/xingran-next/rpa-worker/internal/executor"
	"github.com/xingran-next/rpa-worker/internal/logger"
	"github.com/xingran-next/rpa-worker/internal/types"
)

// getLocalIP 获取本机 IP 地址
func getLocalIP() string {
	// 尝试获取连接到后端的本地 IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		// 如果无法连接外网，返回回环地址
		return "127.0.0.1"
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr).IP.String()
	return localAddr
}

// Worker RPA Worker
type Worker struct {
	id               string
	name             string
	config           *config.Config
	logger           logger.Logger
	redisClient      *communication.RedisClient
	apiClient        *communication.APIClient
	progressReporter *communication.ProgressReporter
	browserPool      *browser.Pool
	engine           *executor.Engine
	state            WorkerState
	stateMutex       sync.RWMutex
	currentTasks     int
	currentTasksMu   sync.Mutex

	// Dynamic scaling fields
	maxConcurrency      int
	maxConcurrencyMu    sync.RWMutex
	scaleCommandChan    chan scaleCommandWrapper

	// Hybrid mode fields
	taskSplitter        *executor.TaskSplitter

	// Lifecycle
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	shutdownSignal   chan struct{}

	// Statistics
	tasksReceived    int64
	tasksCompleted   int64
	tasksFailed      int64
}

// scaleCommandWrapper wraps scale command with metadata
type scaleCommandWrapper struct {
	command types.ScaleCommand
	ackFunc func(error)
}

// WorkerState worker state
type WorkerState int

const (
	StateInitializing WorkerState = iota
	StateOnline
	StateBusy
	StateError
	StateShuttingDown
	StateOffline
)

func (s WorkerState) String() string {
	switch s {
	case StateInitializing:
		return "initializing"
	case StateOnline:
		return "online"
	case StateBusy:
		return "busy"
	case StateError:
		return "error"
	case StateShuttingDown:
		return "shutting_down"
	case StateOffline:
		return "offline"
	default:
		return "unknown"
	}
}

// New create worker
func New(cfg *config.Config) (*Worker, error) {
	ctx, cancel := context.WithCancel(context.Background())
	log, err := logger.NewLogger(cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.Output)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("init logger failed: %w", err)
	}
	workerID := cfg.Worker.ID
	if workerID == "" {
		workerID = fmt.Sprintf("worker-%s", uuid.New().String()[:8])
	}

	// Set start time
	cfg.Worker.StartTime = time.Now()

	// Initial max concurrency from config
	initialMaxConcurrency := cfg.Worker.MaxConcurrency
	if initialMaxConcurrency <= 0 {
		initialMaxConcurrency = 5 // Default value
	}

	w := &Worker{
		id:                workerID,
		name:              cfg.Worker.Name,
		config:            cfg,
		logger:            log,
		ctx:               ctx,
		cancel:            cancel,
		shutdownSignal:    make(chan struct{}),
		state:             StateInitializing,
		maxConcurrency:    initialMaxConcurrency,
		scaleCommandChan:  make(chan scaleCommandWrapper, 10),
	}
	if err := w.initComponents(); err != nil {
		cancel()
		return nil, err
	}
	return w, nil
}

// initComponents initialize components
func (w *Worker) initComponents() error {
	var err error
	w.redisClient, err = communication.NewRedisClient(&w.config.Redis, w.id, w.logger)
	if err != nil {
		return fmt.Errorf("init redis client failed: %w", err)
	}
	w.apiClient = communication.NewAPIClient(&w.config.Backend, w.logger)
	w.progressReporter = communication.NewProgressReporter(w.apiClient, w.id, w.logger)
	w.browserPool, err = browser.NewPool(&w.config.Browser, w.logger)
	if err != nil {
		return fmt.Errorf("init browser pool failed: %w", err)
	}
	w.engine = executor.NewEngine(&w.config.Executor, &w.config.Worker, w.browserPool, w.progressReporter, w.logger, w.redisClient.GetClient())

	// 初始化任务拆分器（混合模式）
	if w.config.Worker.HybridMode.Enabled {
		w.taskSplitter = executor.NewTaskSplitter(
			w.redisClient.GetClient(),
			w.config.Redis.StreamTasks,
			w.config.Redis.StreamGroup,
			&w.config.Worker.HybridMode,
			w.logger,
		)
		w.logger.Info("task splitter initialized (hybrid mode enabled)")
	}

	w.logger.Info("worker components initialized")
	return nil
}

// Start start worker
func (w *Worker) Start() error {
	w.setState(StateOnline)
	if err := w.register(); err != nil {
		return fmt.Errorf("register failed: %w", err)
	}

	// Start heartbeat loop
	w.wg.Add(1)
	go w.heartbeatLoop()

	// Start task consumption loop
	w.wg.Add(1)
	go w.consumeLoop()

	// Start scale command listener
	w.wg.Add(1)
	go w.scaleCommandListener()

	// Start scale command processor
	w.wg.Add(1)
	go w.processScaleCommands()

	if w.config.Monitor.HealthEnabled {
		w.wg.Add(1)
		go w.startHealthCheck()
	}

	w.logger.Info("worker started successfully",
		logger.Int("max_concurrency", w.maxConcurrency))

	go w.waitForShutdown()
	return nil
}

// Shutdown graceful shutdown
func (w *Worker) Shutdown(ctx context.Context) error {
	w.setState(StateShuttingDown)
	w.logger.Info("worker shutting down...")

	// Critical: Stop all active progress monitors before canceling context
	// This prevents goroutine leaks
	if w.taskSplitter != nil {
		w.taskSplitter.StopAllMonitors()
		w.logger.Info("all subtask progress monitors stopped")
	}

	w.cancel()
	close(w.shutdownSignal)

	// Wait a bit for scale commands to finish processing
	time.Sleep(w.config.Worker.ShutdownTimeout)
	close(w.scaleCommandChan)

	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		w.logger.Info("all tasks completed")
	case <-ctx.Done():
		w.logger.Warn("shutdown timeout, force exit")
		return ctx.Err()
	}
	if err := w.cleanup(); err != nil {
		w.logger.Error("cleanup failed", logger.Err(err))
	}
	w.setState(StateOffline)
	w.logger.Info("worker shutdown complete")
	return nil
}

// register register worker
func (w *Worker) register() error {
	capabilities := []string{"chromium", "auto-scaling"}

	// 声明混合模式能力
	if w.config.Worker.HybridMode.Enabled {
		capabilities = append(capabilities, "hybrid-mode")
	}

	req := &types.WorkerRegisterRequest{
		WorkerID:      w.id,
		Name:          w.name,
		Host:          getLocalIP(),
		Port:          8080,
		MaxConcurrency: w.getMaxConcurrency(),
		Version:       w.config.Worker.Version,
		Capabilities:  capabilities,
	}
	_, err := w.apiClient.Register(w.ctx, req)
	if err != nil {
		return err
	}
	w.logger.Info("worker registered successfully")
	return nil
}

// heartbeatLoop heartbeat loop
func (w *Worker) heartbeatLoop() {
	defer w.wg.Done()
	ticker := time.NewTicker(w.config.Backend.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.sendHeartbeat()
		case <-w.shutdownSignal:
			return
		}
	}
}

// sendHeartbeat send heartbeat with current capacity
func (w *Worker) sendHeartbeat() {
	w.currentTasksMu.Lock()
	currentTasks := w.currentTasks
	w.currentTasksMu.Unlock()

	// 根据当前任务数确定状态
	status := "online"
	if currentTasks > 0 {
		status = "busy"
		w.setState(StateBusy)
	} else {
		w.setState(StateOnline)
	}

	req := &types.WorkerHeartbeatRequest{
		WorkerID:      w.id,
		CurrentTasks:  currentTasks,
		TotalExecuted: int(w.tasksCompleted),
		TotalFailed:   int(w.tasksFailed),
		Status:        status,
	}
	if err := w.apiClient.Heartbeat(w.ctx, req); err != nil {
		w.logger.Error("heartbeat failed", logger.Err(err))
		w.setState(StateError)
	}
}

// consumeLoop task consumption loop with dynamic concurrency
func (w *Worker) consumeLoop() {
	defer w.wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-w.shutdownSignal:
			return
		default:
			// Get current max concurrency (dynamic)
			maxConcurrency := w.getMaxConcurrency()

			// Check concurrency limit
			w.currentTasksMu.Lock()
			if w.currentTasks >= maxConcurrency {
				w.currentTasksMu.Unlock()
				time.Sleep(w.config.Worker.RetryCheckInterval)
				continue
			}
			w.currentTasksMu.Unlock()

			// Consume task from Redis
			task, messageID, err := w.redisClient.ConsumeTasks(w.ctx)
			if err != nil {
				w.logger.Error("consume task failed", logger.Err(err))
				time.Sleep(w.config.Worker.RetryCheckInterval)
				continue
			}

			if task == nil {
				time.Sleep(w.config.Worker.RetryCheckInterval)
				continue
			}

			w.tasksReceived++
			w.currentTasksMu.Lock()
			w.currentTasks++
			currentTaskCount := w.currentTasks
			w.currentTasksMu.Unlock()

			w.logger.Debug("task received",
				logger.String("execution_id", task.ExecutionID),
				logger.Int("current_tasks", currentTaskCount),
				logger.Int("max_concurrency", maxConcurrency))

			go func(task *types.TaskMessage, msgID string) {
				defer func() {
					if r := recover(); r != nil {
						w.logger.Error("task execution panic",
							logger.String("panic", fmt.Sprintf("%v", r)),
							logger.String("execution_id", task.ExecutionID))
						w.tasksFailed++
					}
					w.currentTasksMu.Lock()
					w.currentTasks--
					w.currentTasksMu.Unlock()
				}()
				w.executeTask(task, msgID)
			}(task, messageID)
		}
	}
}

// executeTask execute task
func (w *Worker) executeTask(task *types.TaskMessage, messageID string) {
	w.logger.Info("start executing task",
		logger.String("execution_id", task.ExecutionID),
		logger.String("task_id", task.TaskID))

	ctx, cancel := context.WithTimeout(w.ctx, w.config.Worker.TaskTimeout)
	defer cancel()

	// 检查是否正在关闭
	select {
	case <-ctx.Done():
		w.logger.Warn("task cancelled due to shutdown",
			logger.String("execution_id", task.ExecutionID))
		w.tasksFailed++
		return
	default:
	}

	// 检查是否需要拆分任务
	if w.taskSplitter != nil && w.taskSplitter.ShouldSplit(task) {
		w.logger.Info("task requires splitting", logger.String("execution_id", task.ExecutionID))

		splitResult, err := w.taskSplitter.SplitTask(ctx, task)
		if err != nil {
			w.tasksFailed++
			w.logger.Error("task splitting failed", logger.Err(err))

			// 报告失败
			result := &types.ExecutionResult{
				ExecutionID:  task.ExecutionID,
				TaskID:       task.TaskID,
				Status:       types.StatusFailed,
				ErrorMessage: fmt.Sprintf("Task splitting failed: %v", err),
				StartedAt:    time.Now(),
				CompletedAt:  time.Now(),
			}
			w.progressReporter.ReportCompletion(ctx, result)
		} else {
			w.logger.Info("task split successfully",
				logger.String("execution_id", task.ExecutionID),
				logger.Int("subtask_count", splitResult.SubTaskCount))

			// 父任务拆分成功，等待子任务完成
			// 拆分器会自动处理进度聚合和最终结果上报

			// 确认原始任务消息
			if messageID != "" {
				if err := w.redisClient.AckTask(ctx, messageID); err != nil {
					w.logger.Error("ack parent task failed", logger.Err(err))
				}
			}
		}
		return
	}

	// 原有执行逻辑（不拆分的任务或子任务）
	result, err := w.engine.Execute(ctx, task, w.id)
	if err != nil {
		w.tasksFailed++
		w.logger.Error("task execution failed", logger.Err(err))
	} else {
		w.tasksCompleted++
		w.logger.Info("task executed successfully", logger.Duration("duration", result.Duration))
	}
	w.progressReporter.ReportCompletion(w.ctx, result)
	if messageID != "" {
		if err := w.redisClient.AckTask(w.ctx, messageID); err != nil {
			w.logger.Error("ack task failed", logger.Err(err))
		}
	}
}

// scaleCommandListener subscribes to Redis Pub/Sub for scale commands
func (w *Worker) scaleCommandListener() {
	defer w.wg.Done()

	// Redis channel for scale commands
	scaleChannel := fmt.Sprintf("worker:scale:%s", w.id)
	globalScaleChannel := "worker:scale:all"

	// Get Redis client and subscribe
	redisClient := w.redisClient.GetClient()
	pubsub := redisClient.Subscribe(w.ctx, scaleChannel, globalScaleChannel)
	defer pubsub.Close()

	w.logger.Info("subscribed to scale channels",
		logger.String("channel", scaleChannel),
		logger.String("global_channel", globalScaleChannel))

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-w.shutdownSignal:
			return
		case msg := <-pubsub.Channel():
			if msg.Payload == "" {
				continue
			}

			var cmd types.ScaleCommand
			if err := json.Unmarshal([]byte(msg.Payload), &cmd); err != nil {
				w.logger.Error("failed to parse scale command", logger.Err(err))
				continue
			}

			// Filter commands not for this worker (if specified)
			if cmd.WorkerID != "" && cmd.WorkerID != w.id {
				continue
			}

			w.logger.Info("received scale command",
				logger.String("direction", string(cmd.Direction)),
				logger.Int("concurrency", cmd.Concurrency),
				logger.String("reason", cmd.Reason))

			// Send to processor with ack function
			w.scaleCommandChan <- scaleCommandWrapper{
				command: cmd,
				ackFunc: func(err error) {
					// Send acknowledgment
					ackChannel := fmt.Sprintf("worker:scale:ack:%s", cmd.CommandID)
					ackMsg := map[string]interface{}{
						"worker_id":  w.id,
						"command_id": cmd.CommandID,
						"status":     "success",
						"error":       "",
					}
					if err != nil {
						ackMsg["status"] = "failed"
						ackMsg["error"] = err.Error()
					}
					if data, err := json.Marshal(ackMsg); err == nil {
						w.redisClient.GetClient().Publish(w.ctx, ackChannel, data)
					}
				},
			}
		}
	}
}

// processScaleCommands processes scale commands
func (w *Worker) processScaleCommands() {
	defer w.wg.Done()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-w.shutdownSignal:
			return
		case wrapper, ok := <-w.scaleCommandChan:
			if !ok {
				return
			}

			cmd := wrapper.command
			oldConcurrency := w.getMaxConcurrency()

			var err error
			switch cmd.Direction {
			case types.ScaleUp:
				err = w.scaleUp(cmd.Concurrency, cmd.Reason)
			case types.ScaleDown:
				err = w.scaleDown(cmd.Concurrency, cmd.Reason)
			default:
				err = fmt.Errorf("unknown scale direction: %s", cmd.Direction)
			}

			newConcurrency := w.getMaxConcurrency()

			// Log scale event
			w.logger.Info("scale event",
				logger.String("direction", string(cmd.Direction)),
				logger.Int("old_value", oldConcurrency),
				logger.Int("new_value", newConcurrency),
				logger.String("reason", cmd.Reason),
				logger.Err(err))

			// Send acknowledgment
			if wrapper.ackFunc != nil {
				wrapper.ackFunc(err)
			}

			// Re-register with new capacity if scale succeeded
			if err == nil && oldConcurrency != newConcurrency {
				go w.registerWithCapacity(newConcurrency)
			}
		}
	}
}

// scaleUp increases max concurrency
func (w *Worker) scaleUp(targetConcurrency int, reason string) error {
	w.maxConcurrencyMu.Lock()
	defer w.maxConcurrencyMu.Unlock()

	if targetConcurrency <= w.maxConcurrency {
		return fmt.Errorf("target concurrency (%d) must be greater than current (%d)",
			targetConcurrency, w.maxConcurrency)
	}

	// Apply maximum limit
	maxAllowed := w.config.Worker.MaxScaleUpConcurrency
	if maxAllowed == 0 {
		maxAllowed = 50 // Fallback default
	}
	if targetConcurrency > maxAllowed {
		targetConcurrency = maxAllowed
	}

	oldValue := w.maxConcurrency
	w.maxConcurrency = targetConcurrency

	w.logger.Info("scaled up",
		logger.Int("from", oldValue),
		logger.Int("to", w.maxConcurrency),
		logger.String("reason", reason))

	return nil
}

// scaleDown decreases max concurrency
func (w *Worker) scaleDown(targetConcurrency int, reason string) error {
	w.maxConcurrencyMu.Lock()
	defer w.maxConcurrencyMu.Unlock()

	if targetConcurrency >= w.maxConcurrency {
		return fmt.Errorf("target concurrency (%d) must be less than current (%d)",
			targetConcurrency, w.maxConcurrency)
	}

	// Apply minimum limit
	minAllowed := w.config.Worker.MinScaleDownConcurrency
	if minAllowed == 0 {
		minAllowed = 1 // Fallback default
	}
	if targetConcurrency < minAllowed {
		targetConcurrency = minAllowed
	}

	// Wait for current tasks to drop below new limit
	w.currentTasksMu.Lock()
	currentTasks := w.currentTasks
	w.currentTasksMu.Unlock()

	if currentTasks > targetConcurrency {
		return fmt.Errorf("cannot scale down to %d: %d tasks still running",
			targetConcurrency, currentTasks)
	}

	oldValue := w.maxConcurrency
	w.maxConcurrency = targetConcurrency

	w.logger.Info("scaled down",
		logger.Int("from", oldValue),
		logger.Int("to", w.maxConcurrency),
		logger.String("reason", reason))

	return nil
}

// getMaxConcurrency returns current max concurrency (thread-safe)
func (w *Worker) getMaxConcurrency() int {
	w.maxConcurrencyMu.RLock()
	defer w.maxConcurrencyMu.RUnlock()
	return w.maxConcurrency
}

// registerWithCapacity registers worker with specific capacity
func (w *Worker) registerWithCapacity(capacity int) error {
	req := &types.WorkerRegisterRequest{
		WorkerID:      w.id,
		Name:          w.name,
		Port:          8080,
		MaxConcurrency: capacity,
		Version:       w.config.Worker.Version,
		Capabilities:  []string{"chromium", "auto-scaling"},
	}
	_, err := w.apiClient.Register(w.ctx, req)
	return err
}

// waitForShutdown wait for shutdown signal
func (w *Worker) waitForShutdown() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	select {
	case <-w.ctx.Done():
	case <-sigCh:
		w.logger.Info("received shutdown signal")
		w.Shutdown(context.Background())
	case <-w.shutdownSignal:
	}
}

// cleanup cleanup resources
func (w *Worker) cleanup() error {
	var errs []error
	if err := w.browserPool.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := w.redisClient.Close(); err != nil {
		errs = append(errs, err)
	}
	if syncer, ok := w.logger.(interface{ Sync() error }); ok {
		if err := syncer.Sync(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %v", errs)
	}
	return nil
}

// startHealthCheck start health check
func (w *Worker) startHealthCheck() {
	defer w.wg.Done()

	healthServer := NewHealthServer(w, w.logger)
	addr := fmt.Sprintf(":%d", w.config.Monitor.HealthPort)

	if err := healthServer.Start(addr); err != nil {
		w.logger.Error("health check server start failed", logger.Err(err))
		return
	}

	// Wait for shutdown signal
	<-w.ctx.Done()

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := healthServer.Shutdown(ctx); err != nil {
		w.logger.Error("health check server shutdown failed", logger.Err(err))
	}
}

// setState set state
func (w *Worker) setState(state WorkerState) {
	w.stateMutex.Lock()
	defer w.stateMutex.Unlock()
	w.state = state
}

// State get state
func (w *Worker) State() WorkerState {
	w.stateMutex.RLock()
	defer w.stateMutex.RUnlock()
	return w.state
}
