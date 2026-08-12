package types

import "time"

// Task variable key constants
const (
	VariableIsSubTask          = "__isSubTask"
	VariableParentExecutionID  = "__parentExecutionID"
	VariableSubTaskIndex       = "__subTaskIndex"
	VariableSessionData        = "__sessionData"
	VariableCredentials        = "__credentials"
)

// Task 任务定义
type Task struct {
	ID          string        `json:"id"`
	TaskName    string        `json:"taskName"`
	Description string        `json:"description"`
	TargetURL   string        `json:"targetUrl"`
	Script      *Script       `json:"script"`
	Timeout     time.Duration `json:"timeout"`
	MaxRetry    int           `json:"maxRetry"`
	Priority    int           `json:"priority"`
	Status      int           `json:"status"`
	Variables   map[string]interface{} `json:"inputParams"`
}

// Script 脚本定义
type Script struct {
	Actions []Action `json:"actions"`
}

// Action 动作定义
type Action struct {
	ID          string                 `json:"id"`
	Type        ActionType            `json:"type"`
	Description string                 `json:"description"`
	Selector    string                 `json:"selector,omitempty"`
	Params      map[string]interface{} `json:"params,omitempty"`
	Timeout     int                    `json:"timeout,omitempty"`
	Retry       int                    `json:"retry,omitempty"`
	Value       string                 `json:"value,omitempty"`
	AIAssisted  bool                   `json:"aiAssisted,omitempty"`
}

// ActionType 动作类型
type ActionType string

const (
	ActionNavigate   ActionType = "navigate"
	ActionClick      ActionType = "click"
	ActionFill       ActionType = "fill"
	ActionSelect     ActionType = "select"
	ActionWait       ActionType = "wait"
	ActionScreenshot ActionType = "screenshot"
	ActionExtract    ActionType = "extract"
	ActionScroll     ActionType = "scroll"
	ActionUpload     ActionType = "upload"
	ActionDownload   ActionType = "download"
	ActionEvaluate   ActionType = "evaluate"
	ActionWaitFor    ActionType = "waitFor"
	ActionClose      ActionType = "close"
	ActionLoop       ActionType = "loop"       // Loop over data array
	ActionPause      ActionType = "pause"      // Wait for human input
	ActionCondition  ActionType = "condition"  // Conditional branching
	ActionAutoLogin  ActionType = "autologin"  // Auto login with stored credentials
)

// TaskMessage 任务消息（从 Redis）
type TaskMessage struct {
	ExecutionID string                 `json:"executionId"`
	TaskID      string                 `json:"taskId"`
	TaskName    string                 `json:"taskName"`
	TargetURL   string                 `json:"targetUrl"`
	Script      []Action               `json:"script"`
	Timeout     time.Duration         `json:"timeout"`
	MaxRetry    int                    `json:"maxRetry"`
	Variables   map[string]interface{} `json:"inputParams"`
	TriggeredBy string                 `json:"triggeredBy"`
	TriggerType string                 `json:"triggerType"`
	CreatedAt   time.Time              `json:"createdAt"`

	// 自动登录相关
	CredentialID string                 `json:"credentialId,omitempty"`
	SessionID    string                 `json:"sessionId,omitempty"`    // 已存在的会话ID
	SessionData  *SessionData           `json:"sessionData,omitempty"` // 会话数据（token/cookie）
}

// ExecutionResult 执行结果
type ExecutionResult struct {
	ExecutionID  string        `json:"executionId"`
	TaskID       string        `json:"taskId"`
	Status       ExecutionStatus `json:"status"`
	ErrorMessage string        `json:"error,omitempty"`
	StartedAt    time.Time     `json:"startedAt"`
	CompletedAt  time.Time     `json:"completedAt"`
	Duration     time.Duration `json:"duration"`
	Data         map[string]interface{} `json:"data,omitempty"`
	Screenshots  []string      `json:"screenshots,omitempty"`
	Logs         []string      `json:"logs,omitempty"`
	Step         int           `json:"step"`
	Total        int           `json:"total"`
}

// ExecutionStatus 执行状态
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusSuccess   ExecutionStatus = "success"
	StatusFailed    ExecutionStatus = "failed"
	StatusCancelled ExecutionStatus = "cancelled"
	StatusTimeout   ExecutionStatus = "timeout"
	StatusPaused    ExecutionStatus = "paused"
)

// SessionData 会话数据（用于自动登录）
type SessionData struct {
	AccessToken  string                 `json:"accessToken,omitempty"`
	RefreshToken string                 `json:"refreshToken,omitempty"`
	Cookies      []Cookie                `json:"cookies,omitempty"`
	SessionData  map[string]interface{} `json:"sessionData,omitempty"`
}

// Cookie HTTP Cookie
type Cookie struct {
	Name     string     `json:"name"`
	Value    string     `json:"value"`
	Domain   string     `json:"domain,omitempty"`
	Path     string     `json:"path,omitempty"`
	Expires  *time.Time `json:"expires,omitempty"`
	Secure   bool       `json:"secure,omitempty"`
	HTTPOnly bool       `json:"httpOnly,omitempty"`
}

// ProgressReport progress report
type ProgressReport struct {
	ExecutionID     string          `json:"executionId"`
	WorkerID        string          `json:"workerId,omitempty"`
	ProgressCurrent int             `json:"progressCurrent"`
	ProgressTotal   int             `json:"progressTotal"`
	Message         string          `json:"message"`
	Status          ExecutionStatus `json:"status"`
	Screenshot      string          `json:"screenshot,omitempty"`
	Log             string          `json:"log,omitempty"`
	Step            int             `json:"step,omitempty"`
	Total           int             `json:"total,omitempty"`
	Timestamp       time.Time       `json:"timestamp,omitempty"`
}

// WorkerInfo Worker 信息
type WorkerInfo struct {
	WorkerID       string   `json:"workerId"`
	WorkerName     string   `json:"workerName"`
	Host           string   `json:"host,omitempty"`
	Port           int      `json:"port,omitempty"`
	Version        string   `json:"version,omitempty"`
	Capabilities   []string `json:"capabilities,omitempty"`
	MaxConcurrency int      `json:"maxConcurrency"`
}

// WorkerRegisterRequest Worker 注册请求
type WorkerRegisterRequest struct {
	WorkerID       string   `json:"workerId" binding:"required"`
	Name           string   `json:"name" binding:"required"`
	Host           string   `json:"host,omitempty"`
	Port           int      `json:"port"`
	MaxConcurrency int      `json:"maxConcurrency"`
	Version        string   `json:"version,omitempty"`
	Capabilities   []string `json:"capabilities,omitempty"`
}

// WorkerRegisterResponse Worker 注册响应
type WorkerRegisterResponse struct {
	WorkerID string `json:"workerId"`
	Token     string `json:"token,omitempty"`
}

// WorkerHeartbeatRequest 心跳请求
type WorkerHeartbeatRequest struct {
	WorkerID      string `json:"workerId"`
	CurrentTasks  int    `json:"currentTasks"`
	TotalExecuted int    `json:"totalExecuted"`
	TotalFailed   int    `json:"totalFailed"`
	Status        string `json:"status"`
}

// APIResponse API 响应
type APIResponse struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
	RequestID string      `json:"request_id"`
}

// ScaleDirection 扩缩容方向
type ScaleDirection string

const (
	ScaleUp   ScaleDirection = "up"
	ScaleDown ScaleDirection = "down"
)

// ScaleCommand 扩缩容指令
type ScaleCommand struct {
	CommandID   string         `json:"commandId"`
	WorkerID    string         `json:"workerId,omitempty"`
	Direction   ScaleDirection `json:"direction"`
	Concurrency int            `json:"concurrency"`
	Reason      string         `json:"reason"`
	Timestamp   int64          `json:"timestamp"`
}

// ScaleEvent 扩缩容事件
type ScaleEvent struct {
	EventID     string         `json:"eventId"`
	WorkerID    string         `json:"workerId"`
	Direction   ScaleDirection `json:"direction"`
	OldValue    int            `json:"oldValue"`
	NewValue    int            `json:"newValue"`
	Reason      string         `json:"reason"`
	Timestamp   time.Time      `json:"timestamp"`
}

// ========== 混合模式相关类型 ==========

// SubTaskMessage 子任务消息（继承 TaskMessage）
type SubTaskMessage struct {
	*TaskMessage                      // 嵌入原始任务消息
	ParentExecutionID string                 `json:"parentExecutionId"` // 父任务执行ID
	SubTaskIndex      int                    `json:"subTaskIndex"`      // 子任务索引
	SubTaskTotal      int                    `json:"subTaskTotal"`      // 子任务总数
	LoopItemVar       string                 `json:"loopItemVar"`       // 循环项变量名
	LoopItemData      map[string]interface{} `json:"loopItemData"`      // 循环项数据
}

// SubTaskResult 子任务执行结果（用于进度聚合）
type SubTaskResult struct {
	SubTaskIndex int             `json:"subTaskIndex"`
	Status       ExecutionStatus `json:"status"`
	ErrorMessage string          `json:"errorMessage,omitempty"`
	StartedAt    time.Time       `json:"startedAt"`
	CompletedAt  time.Time       `json:"completedAt"`
	Data         map[string]interface{} `json:"data,omitempty"`
}

// ProgressAggregateMessage 进度聚合消息
type ProgressAggregateMessage struct {
	ParentExecutionID string          `json:"parentExecutionId"`
	CompletedCount    int             `json:"completedCount"`
	TotalCount        int             `json:"totalCount"`
	SuccessCount      int             `json:"successCount"`
	FailureCount      int             `json:"failureCount"`
	Status            ExecutionStatus `json:"status"`
	SubTaskResults    []SubTaskResult `json:"subTaskResults,omitempty"`
	Timestamp         time.Time       `json:"timestamp"`
}
