package types

import "time"

// TaskMessage 从 Redis 消费的任务消息
type TaskMessage struct {
	CreatedAt    time.Time         `json:"created_at"`
	ExecutionID  string            `json:"execution_id"`
	TaskID       string            `json:"task_id"`
	TargetURL    string            `json:"target_url"`
	Script       []Action          `json:"script"`
	Variables    map[string]interface{} `json:"variables"`
	Priority     int               `json:"priority"`
}

// Action 单个脚本动作
type Action struct {
	ID           string                 `json:"id"`
	Type         ActionType             `json:"type"`
	Description  string                 `json:"description"`
	Selector     string                 `json:"selector"`
	Params       map[string]interface{} `json:"params"`
	Timeout      int                    `json:"timeout"`
	Retry        int                    `json:"retry"`
	AIAssisted   bool                   `json:"ai_assisted"`
}

// ActionType 动作类型
type ActionType string

const (
	ActionNavigate  ActionType = "navigate"
	ActionClick     ActionType = "click"
	ActionFill      ActionType = "fill"
	ActionSelect    ActionType = "select"
	ActionWait      ActionType = "wait"
	ActionScreenshot ActionType = "screenshot"
	ActionWaitFor   ActionType = "wait_for"
	ActionExtract   ActionType = "extract"
)

// ExecutionStatus 执行状态
type ExecutionStatus string

const (
	StatusPending  ExecutionStatus = "pending"
	StatusRunning  ExecutionStatus = "running"
	StatusSuccess  ExecutionStatus = "success"
	StatusFailed   ExecutionStatus = "failed"
	StatusCanceled ExecutionStatus = "canceled"
)

// ExecutionResult 执行结果
type ExecutionResult struct {
	ExecutionID  string          `json:"execution_id"`
	TaskID       string          `json:"task_id"`
	Status       ExecutionStatus `json:"status"`
	ErrorMessage string          `json:"error_message,omitempty"`
	Step         int             `json:"step"`
	Total        int             `json:"total"`
	StartedAt    time.Time       `json:"started_at"`
	CompletedAt  time.Time       `json:"completed_at,omitempty"`
	Duration     time.Duration   `json:"duration,omitempty"`
	Screenshot   string          `json:"screenshot,omitempty"`
	Output       map[string]interface{} `json:"output,omitempty"`
}

// ProgressReport 进度报告
type ProgressReport struct {
	ExecutionID     string          `json:"execution_id"`
	WorkerID        string          `json:"worker_id"`
	ProgressCurrent int             `json:"progress_current"`
	ProgressTotal   int             `json:"progress_total"`
	Message         string          `json:"message"`
	Status          ExecutionStatus `json:"status"`
	Step            int             `json:"step,omitempty"`
	Total           int             `json:"total,omitempty"`
	Screenshot      string          `json:"screenshot,omitempty"`
	Timestamp       time.Time       `json:"timestamp"`
}

// WorkerRegisterRequest Worker 注册请求
type WorkerRegisterRequest struct {
	WorkerID      string   `json:"worker_id"`
	Name          string   `json:"name"`
	Port          int      `json:"port"`
	MaxConcurrency int     `json:"max_concurrency"`
	Version       string   `json:"version"`
	Capabilities  []string `json:"capabilities"`
}

// WorkerRegisterResponse Worker 注册响应
type WorkerRegisterResponse struct {
	WorkerID  string `json:"worker_id"`
	Token     string `json:"token"`
	Config    string `json:"config"`
}

// WorkerHeartbeatRequest Worker 心跳请求
type WorkerHeartbeatRequest struct {
	WorkerID      string `json:"worker_id"`
	CurrentTasks  int    `json:"current_tasks"`
	TotalExecuted int    `json:"total_executed"`
	TotalFailed   int    `json:"total_failed"`
	Status        string `json:"status"`
}

// APIResponse 通用 API 响应
type APIResponse struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	Timestamp int64  `json:"timestamp"`
	RequestID string `json:"request_id,omitempty"`
}
