package rpa

// ListParams 通用列表查询参数
type ListParams struct {
	Current  int `json:"current"`
	PageSize int `json:"pageSize"`
}

// PageResult 分页结果
type PageResult struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Current  int         `json:"current"`
	PageSize int         `json:"pageSize"`
}

// TaskListParams 任务列表参数
type TaskListParams struct {
	ListParams
	Name     string `json:"name,omitempty"`
	Status   *int   `json:"status,omitempty"`
	Priority *int   `json:"priority,omitempty"`
	Tags     string `json:"tags,omitempty"`
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Name        string        `json:"name" binding:"required"`
	Description string        `json:"description,omitempty"`
	Status      int           `json:"status"`
	Priority    int           `json:"priority"`
	Script      []interface{} `json:"script" binding:"required"`
	TargetURL   string        `json:"targetUrl,omitempty"`
	Timeout     int           `json:"timeout"`
	MaxRetry    int           `json:"maxRetry"`
	Tags        string        `json:"tags,omitempty"`
}

// UpdateTaskRequest 更新任务请求
type UpdateTaskRequest struct {
	ID          string        `json:"id" binding:"required"`
	Name        string        `json:"name,omitempty"`
	Description string        `json:"description,omitempty"`
	Status      int           `json:"status,omitempty"`
	Priority    int           `json:"priority,omitempty"`
	Script      []interface{} `json:"script,omitempty"`
	TargetURL   string        `json:"targetUrl,omitempty"`
	Timeout     int           `json:"timeout,omitempty"`
	MaxRetry    int           `json:"maxRetry,omitempty"`
	Tags        string        `json:"tags,omitempty"`
}

// ExecuteTaskRequest 执行任务请求
type ExecuteTaskRequest struct {
	TaskID       string                 `json:"taskId" binding:"required"`
	InputParams  map[string]interface{} `json:"inputParams,omitempty"`
	WorkerID     string                 `json:"workerId,omitempty"`
	Priority     int                    `json:"priority"`
	CredentialID string                 `json:"credentialId,omitempty"` // 凭证ID（用于自动登录）
}

// WorkerListParams Worker列表查询参数
type WorkerListParams struct {
	ListParams
	Name   string `json:"name,omitempty"`
	Status *int   `json:"status,omitempty"`
}

// ExecutionListParams 执行记录列表查询参数
type ExecutionListParams struct {
	ListParams
	TaskID   string `json:"taskId,omitempty"`
	WorkerID string `json:"workerId,omitempty"`
	Status   *int   `json:"status,omitempty"`
}

// WorkerRegisterRequest Worker 注册请求
type WorkerRegisterRequest struct {
	WorkerID       string                 `json:"workerId" binding:"required"`
	Name           string                 `json:"name" binding:"required"`
	Host           string                 `json:"host,omitempty"`
	Port           int                    `json:"port"`
	MaxConcurrency int                    `json:"maxConcurrency"`
	Version        string                 `json:"version,omitempty"`
	Capabilities   []string               `json:"capabilities,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// WorkerHeartbeatRequest Worker 心跳请求
type WorkerHeartbeatRequest struct {
	WorkerID      string `json:"workerId" binding:"required"`
	CurrentTasks  int    `json:"currentTasks"`
	TotalExecuted int    `json:"totalExecuted"`
	TotalFailed   int    `json:"totalFailed"`
	Status        string `json:"status"`
}

// WorkerProgressRequest Worker 进度上报请求
type WorkerProgressRequest struct {
	ExecutionID     string `json:"executionId" binding:"required"`
	ProgressCurrent int    `json:"progressCurrent"`
	ProgressTotal   int    `json:"progressTotal"`
	Message         string `json:"message"`
	Status          string `json:"status"`
	Screenshot      string `json:"screenshot"`
	Log             string `json:"log"`
}

// AIScriptGenerateRequest AI 脚本生成请求
type AIScriptGenerateRequest struct {
	Description string `json:"description" binding:"required"`
	URL         string `json:"url"`
}

// AIScriptOptimizeRequest AI 脚本优化请求
type AIScriptOptimizeRequest struct {
	Script      []interface{} `json:"script" binding:"required"`
	Description string        `json:"description,omitempty"`
	Goals       []string      `json:"goals"`
}

// AIAgentDecisionRequest AI Agent 决策请求
type AIAgentDecisionRequest struct {
	TaskDescription    string   `json:"taskDescription" binding:"required"`
	CurrentStep        int      `json:"currentStep"`
	FailedSelector     string   `json:"failedSelector,omitempty"`
	ScreenshotBase64   string   `json:"screenshotBase64,omitempty"`
	HTMLSnippet        string   `json:"htmlSnippet,omitempty"`
	AvailableSelectors []string `json:"availableSelectors,omitempty"`
	LastError          string   `json:"lastError"`
}

// AIAgentAction AI Agent 返回的动作
type AIAgentAction struct {
	Type              string         `json:"type"`
	Selector          string         `json:"selector"`
	Coordinates       []int          `json:"coordinates"`
	Value             string         `json:"value"`
	Reasoning         string         `json:"reasoning"`
	Confidence        float64        `json:"confidence"`
	SuggestedFix      string         `json:"suggestedFix"`
	AlternativeAction *AIAgentAction `json:"alternativeAction"`
}

// OptimizeScriptRequest 优化脚本请求
type OptimizeScriptRequest = AIScriptOptimizeRequest

// OptimizeScriptResponse 优化脚本响应
type OptimizeScriptResponse struct {
	Script       []interface{} `json:"script"`
	Changes      []string      `json:"changes"`
	Improvements []string      `json:"improvements"`
}

// AnalyzeFailureRequest 分析失败原因请求
type AnalyzeFailureRequest struct {
	TaskDescription  string      `json:"taskDescription" binding:"required"`
	CurrentStep      int         `json:"currentStep"`
	FailedAction     interface{} `json:"failedAction"`
	ErrorMessage     string      `json:"errorMessage" binding:"required"`
	ScreenshotBase64 string      `json:"screenshotBase64"`
	HTMLSnippet      string      `json:"htmlSnippet"`
}

// FixAction 修复动作
type FixAction struct {
	OriginalAction interface{} `json:"originalAction"`
	FixedAction    interface{} `json:"fixedAction"`
	Reason         string      `json:"reason"`
	Confidence     float64     `json:"confidence"`
}
