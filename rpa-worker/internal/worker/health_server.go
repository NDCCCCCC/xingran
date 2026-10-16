package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/xingran-next/rpa-worker/internal/logger"
)

// HealthServer HTTP health check server
type HealthServer struct {
	worker   *Worker
	server   *http.Server
	logger   logger.Logger
}

// HealthResponse Health check response
type HealthResponse struct {
	Status    string    `json:"status"`
	WorkerID  string    `json:"worker_id"`
	WorkerName string   `json:"worker_name"`
	State     string    `json:"state"`
	Timestamp time.Time `json:"timestamp"`
	Uptime    string    `json:"uptime"`
	Stats     WorkerStats `json:"stats"`
}

// WorkerStats Worker statistics
type WorkerStats struct {
	CurrentTasks  int `json:"current_tasks"`
	MaxConcurrency int `json:"max_concurrency"`
	TasksReceived int64 `json:"tasks_received"`
	TasksCompleted int64 `json:"tasks_completed"`
	TasksFailed int64 `json:"tasks_failed"`
}

// NewHealthServer Create health check server
func NewHealthServer(worker *Worker, logger logger.Logger) *HealthServer {
	return &HealthServer{
		worker: worker,
		logger: logger,
	}
}

// Start Start health check server
func (s *HealthServer) Start(addr string) error {
	mux := http.NewServeMux()

	// Health check endpoints
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/ready", s.handleReady)
	mux.HandleFunc("/metrics", s.handleMetrics)

	s.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	s.logger.Info("health check server started", logger.String("addr", addr))

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("health check server error", logger.Err(err))
		}
	}()

	return nil
}

// Shutdown Shutdown server
func (s *HealthServer) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// handleHealth Handle health check request
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

	// Check if state is healthy
	if state == StateError || state == StateOffline {
		response.Status = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(response)
}

// handleReady Handle readiness check request
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

// handleMetrics Handle metrics request
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
