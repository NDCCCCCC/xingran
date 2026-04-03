package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ruoyi-next/rpa-worker/internal/logger"
)

// HealthServer 健康检查HTTP服务器
type HealthServer struct {
	worker   *Worker
	server   *http.Server
	logger   logger.Logger
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status    string    `json:"status"`
	WorkerID  string    `json:"worker_id"`
	WorkerName string   `json:"worker_name"`
	State     string    `json:"state"`
	Timestamp time.Time `json:"timestamp"`
	Uptime    string    `json:"uptime"`
	Stats     WorkerStats `json:"stats"`
}

// WorkerStats Worker统计信息
type WorkerStats struct {
	CurrentTasks  int `json:"current_tasks"`
	MaxConcurrency int `json:"max_concurrency"`
	TasksReceived int64 `json:"tasks_received"`
	TasksCompleted int64 `json:"tasks_completed"`
	TasksFailed int64 `json:"tasks_failed"`
}

// NewHealthServer 创建健康检查服务器
func NewHealthServer(worker *Worker, logger logger.Logger) *HealthServer {
	return &HealthServer{
		worker: worker,
		logger: logger,
	}
}

// Start 启动健康检查服务器
func (s *HealthServer) Start(addr string) error {
	mux := http.NewServeMux()

	// 健康检查端点
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/ready", s.handleReady)
	mux.HandleFunc("/metrics", s.handleMetrics)

	s.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	s.logger.Info("健康检查服务器启动", logger.String("addr", addr))

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("健康检查服务器错误", logger.Err(err))
		}
	}()

	return nil
}

// Shutdown 关闭服务器
func (s *HealthServer) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// handleHealth 处理健康检查请求
func (s *HealthServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	s.worker.currentTasksMu.Lock()
	currentTasks := s.worker.currentTasks
	s.worker.currentTasksMu.Unlock()

	maxConcurrency := s.worker.getMaxConcurrency()
	state := s.worker.State()

	response := HealthResponse{
		Status:     "healthy",
		WorkerID:   s.worker.id,
		WorkerName: s.worker.name,
		State:      state.String(),
		Timestamp:  time.Now(),
		Uptime:     time.Since(s.worker.config.Worker.StartTime).String(),
		Stats: WorkerStats{
			CurrentTasks:   currentTasks,
			MaxConcurrency: maxConcurrency,
			TasksReceived:  s.worker.tasksReceived,
			TasksCompleted: s.worker.tasksCompleted,
			TasksFailed:    s.worker.tasksFailed,
		},
	}

	// 检查状态是否健康
	if state == StateError || state == StateOffline {
		response.Status = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(response)
}

// handleReady 处理就绪检查请求
func (s *HealthServer) handleReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	state := s.worker.State()

	if state == StateOnline || state == StateBusy {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ready",
			"state":  state.String(),
		})
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "not_ready",
			"state":  state.String(),
		})
	}
}

// handleMetrics 处理指标请求
func (s *HealthServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	s.worker.currentTasksMu.Lock()
	currentTasks := s.worker.currentTasks
	s.worker.currentTasksMu.Unlock()

	maxConcurrency := s.worker.getMaxConcurrency()

	metrics := fmt.Sprintf(`# HELP rpa_worker_state Worker state
# TYPE rpa_worker_state gauge
rpa_worker_state{worker_id="%s",state="%s"} 1

# HELP rpa_worker_current_tasks Current number of tasks
# TYPE rpa_worker_current_tasks gauge
rpa_worker_current_tasks{worker_id="%s"} %d

# HELP rpa_worker_max_concurrency Maximum concurrency
# TYPE rpa_worker_max_concurrency gauge
rpa_worker_max_concurrency{worker_id="%s"} %d

# HELP rpa_worker_tasks_received Total tasks received
# TYPE rpa_worker_tasks_received counter
rpa_worker_tasks_received{worker_id="%s"} %d

# HELP rpa_worker_tasks_completed Total tasks completed
# TYPE rpa_worker_tasks_completed counter
rpa_worker_tasks_completed{worker_id="%s"} %d

# HELP rpa_worker_tasks_failed Total tasks failed
# TYPE rpa_worker_tasks_failed counter
rpa_worker_tasks_failed{worker_id="%s"} %d
`,
		s.worker.id, s.worker.State().String(),
		s.worker.id, currentTasks,
		s.worker.id, maxConcurrency,
		s.worker.id, s.worker.tasksReceived,
		s.worker.id, s.worker.tasksCompleted,
		s.worker.id, s.worker.tasksFailed,
	)

	w.Write([]byte(metrics))
}
