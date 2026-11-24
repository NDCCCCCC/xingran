package rpa

import (
	"fmt"

	rpamodels "github.com/xingran-next/xingran-go-backend/internal/models/rpa"
	"github.com/xingran-next/xingran-go-backend/internal/services/base"
)

// ListParams 通用列表查询参数
// 嵌入 base.BaseListRequest 使 json 顶层自动获得 orderByColumn/isAsc 字段,
// 所有嵌入 ListParams 的 XxxListParams(Task/Worker/Execution)自动获得服务端排序能力。
type ListParams struct {
	base.BaseListRequest
}

// rpa 三个表的可排序字段白名单(同包共享)。
var (
	taskAllowedSortFields = map[string]string{
		"name":      "name",
		"status":    "status",
		"priority":  "priority",
		"createdAt": "created_at",
	}
	workerAllowedSortFields = map[string]string{
		"workerName":    "worker_name",
		"workerId":      "worker_id",
		"ipAddress":     "ip_address",
		"status":        "status",
		"lastHeartbeat": "last_heartbeat",
		"createdAt":     "created_at",
	}
	executionAllowedSortFields = map[string]string{
		"taskId":    "task_id",
		"workerId":  "worker_id",
		"status":    "status",
		"startTime": "start_time",
		"endTime":   "end_time",
		"createdAt": "created_at",
	}
)

// PageResult 分页结果
type PageResult struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Current  int         `json:"current"`
	PageSize int         `json:"pageSize"`
}

// TaskListParams 任务列表参数
//
// CONTRACT-02（D2）：Status 从 *int 改为 *rpamodels.TaskStatus，语义与模型列
// 同构（0=启用 / 1=停用，见 models/rpa/task.go TaskStatusEnabled/Disabled）。
// 前端必须传数字（旧前端误传执行态字符串 "pending"/"running" 会在这里 400，
// 而不是被 SQL 静默当作非法值过滤出空列表）。
// Priority 保持 *int，与 CONTRACT-03 裁定「优先级域放宽为 0-100」一致。
type TaskListParams struct {
	ListParams
	Name     string                `json:"name,omitempty"`
	Status   *rpamodels.TaskStatus `json:"status,omitempty"`
	Priority *int                  `json:"priority,omitempty"`
	Tags     string                `json:"tags,omitempty"`
}

// Validate 校验任务列表筛选参数的取值域。
//
// CONTRACT-02：非法枚举必须 400 且错误可读（handler 用 bindAndValidate 之后的
// 显式校验，而不是让 SQL 层静默返回空列表）。Status 只接受 0/1。
func (p *TaskListParams) Validate() error {
	if p == nil || p.Status == nil {
		return nil
	}
	if *p.Status != rpamodels.TaskStatusEnabled && *p.Status != rpamodels.TaskStatusDisabled {
		return fmt.Errorf("status 非法取值 %d：任务状态只接受 0（启用）或 1（停用）", int(*p.Status))
	}
	return nil
}

// TaskPriorityMax 任务优先级取值域上界。
//
// CONTRACT-03 裁定（推荐项，见 phases/152-SUMMARY.md）：**放宽后端到 0-100**，
// 与前端 EditModal 的 InputNumber(min=0,max=100) 一致，不做 DB 迁移 ——
// 列是 SMALLINT，0-100 完全在 SMALLINT 域内。原 0-3 枚举
// （TaskPriorityLow/Medium/High/Urgent）保留为语义常量但不再是硬约束：
// 任务优先级是**排序权重**而非四档分类，前端滑块语义（50=中等）才符合直觉。
const TaskPriorityMax = 100

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

// Validate 校验创建请求的枚举/取值域（CONTRACT-02/03）。
func (r *CreateTaskRequest) Validate() error {
	if r == nil {
		return nil
	}
	if r.Status != int(rpamodels.TaskStatusEnabled) && r.Status != int(rpamodels.TaskStatusDisabled) {
		return fmt.Errorf("status 非法取值 %d：任务状态只接受 0（启用）或 1（停用）", r.Status)
	}
	if r.Priority < 0 || r.Priority > TaskPriorityMax {
		return fmt.Errorf("priority 非法取值 %d：优先级取值域 0-%d", r.Priority, TaskPriorityMax)
	}
	return nil
}

// UpdateTaskRequest 更新任务请求
//
// CONTRACT-03（D5）：Status / Priority 改为 **指针** 语义。
// 旧实现用 `if req.Status > 0` / `if req.Priority > 0` 判定「是否传了」，
// 导致 status=0（启用）与 priority=0 永远写不回 —— 任务一旦停用就无法再启用。
// 指针区分「未传（nil，保留原值）」与「显式传 0（写回 0）」。
type UpdateTaskRequest struct {
	// ID 不带 binding:"required"：handler 在校验后总是用路径参数覆盖它
	// （task_handler.go Update: `req.ID = id`）。旧 tag 会让前端不传 id 的
	// 正常更新请求在 ShouldBindJSON 阶段就 400（错误信息还只写"请求参数错误"）。
	ID          string        `json:"id,omitempty"`
	Name        string        `json:"name,omitempty"`
	Description string        `json:"description,omitempty"`
	Status      *int          `json:"status,omitempty"`
	Priority    *int          `json:"priority,omitempty"`
	Script      []interface{} `json:"script,omitempty"`
	TargetURL   string        `json:"targetUrl,omitempty"`
	Timeout     int           `json:"timeout,omitempty"`
	MaxRetry    int           `json:"maxRetry,omitempty"`
	Tags        string        `json:"tags,omitempty"`
}

// Validate 校验更新请求的枚举/取值域；nil 表示未传（合法）。
func (r *UpdateTaskRequest) Validate() error {
	if r == nil {
		return nil
	}
	if r.Status != nil {
		if *r.Status != int(rpamodels.TaskStatusEnabled) && *r.Status != int(rpamodels.TaskStatusDisabled) {
			return fmt.Errorf("status 非法取值 %d：任务状态只接受 0（启用）或 1（停用）", *r.Status)
		}
	}
	if r.Priority != nil && (*r.Priority < 0 || *r.Priority > TaskPriorityMax) {
		return fmt.Errorf("priority 非法取值 %d：优先级取值域 0-%d", *r.Priority, TaskPriorityMax)
	}
	return nil
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
//
// CONTRACT-02（D2）：Worker.Status 在模型里是 string（online/offline/busy），
// 此处同步改 *string，worker_service.List 的 *params.Status 解引用天然生效。
type WorkerListParams struct {
	ListParams
	Name   string  `json:"name,omitempty"`
	Status *string `json:"status,omitempty"`
}

// Validate 校验 Worker 列表筛选参数的取值域（online/offline/busy）。
func (p *WorkerListParams) Validate() error {
	if p == nil || p.Status == nil || *p.Status == "" {
		return nil
	}
	switch rpamodels.WorkerStatus(*p.Status) {
	case rpamodels.WorkerStatusOnline, rpamodels.WorkerStatusOffline, rpamodels.WorkerStatusBusy:
		return nil
	}
	return fmt.Errorf("status 非法取值 %q：Worker 状态只接受 online / offline / busy", *p.Status)
}

// ExecutionListParams 执行记录列表查询参数
//
// CONTRACT-02（D2）：Execution.Status 在模型里是 string
// （pending/running/success/failed/cancelled/timeout），此处同步改 *string。
type ExecutionListParams struct {
	ListParams
	TaskID   string  `json:"taskId,omitempty"`
	WorkerID string  `json:"workerId,omitempty"`
	Status   *string `json:"status,omitempty"`
}

// Validate 校验执行记录列表筛选参数的取值域。
func (p *ExecutionListParams) Validate() error {
	if p == nil || p.Status == nil || *p.Status == "" {
		return nil
	}
	switch rpamodels.RPAExecutionStatus(*p.Status) {
	case rpamodels.RPAExecutionStatusPending, rpamodels.RPAExecutionStatusRunning,
		rpamodels.RPAExecutionStatusSuccess, rpamodels.RPAExecutionStatusFailed,
		rpamodels.RPAExecutionStatusCancelled, rpamodels.RPAExecutionStatusTimeout,
		// Phase 155 / LIVEVIEW-03：Phase 153 引入了 paused 状态（RUNTIME-05 状态机
		// 的 running→paused 合法迁移），但列表筛选的值域当时漏补 —— 前端按
		// status=paused 精确筛选会被这里 400 拒绝。实时画面页要列出「执行中 +
		// 人工干预暂停中」两类可观看对象，故一并补齐（值域与模型常量保持同源）。
		rpamodels.RPAExecutionStatusPaused:
		return nil
	}
	return fmt.Errorf("status 非法取值 %q：执行状态只接受 pending/running/paused/success/failed/cancelled/timeout", *p.Status)
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
	// RunningExecutionIDs Phase 153 / RUNTIME-01：worker 当前运行中的执行 ID 列表。
	// 心跳通道顺带补归属：这些 execution 若 worker_id 为空则回写（无独立进度
	// 上报的长任务在首个心跳前也有归属）。
	RunningExecutionIDs []string `json:"runningExecutionIds,omitempty"`
}

// WorkerProgressRequest Worker 进度上报请求
type WorkerProgressRequest struct {
	ExecutionID     string `json:"executionId" binding:"required"`
	WorkerID        string `json:"workerId"` // Phase 153 / RUNTIME-01：上报方 Worker 业务 ID（归属回写依据）
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
