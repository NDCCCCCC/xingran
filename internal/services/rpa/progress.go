package rpa

// ProgressUpdate 进度更新结构体
type ProgressUpdate struct {
	ExecutionID   string `json:"executionId"`   // 执行记录ID
	TaskID        string `json:"taskId"`        // 任务ID
	TaskName      string `json:"taskName"`      // 任务名称
	Step          int    `json:"step"`          // 当前步骤
	Total         int    `json:"total"`         // 总步骤数
	Message       string `json:"message"`       // 进度消息
	Status        string `json:"status"`        // 状态
	TriggeredBy   string `json:"triggeredBy"`   // 触发人
	WorkerID      string `json:"workerId"`      // Worker ID
	WorkerName    string `json:"workerName"`    // Worker 名称
	ScreenshotURL string `json:"screenshotUrl"` // 截图URL（可选）
}

// ProgressStep 进度步骤结构体
type ProgressStep struct {
	StepNumber int    `json:"stepNumber"` // 步骤编号
	Name       string `json:"name"`       // 步骤名称
	Status     string `json:"status"`     // 状态: pending, running, success, failed
	Message    string `json:"message"`    // 步骤消息
	StartTime  int64  `json:"startTime"`  // 开始时间
	EndTime    int64  `json:"endTime"`    // 结束时间
}

// ProgressDetail 进度详情结构体
type ProgressDetail struct {
	ExecutionID      string         `json:"executionId"`      // 执行记录ID
	TaskID           string         `json:"taskId"`           // 任务ID
	TaskName         string         `json:"taskName"`         // 任务名称
	CurrentStep      int            `json:"currentStep"`      // 当前步骤
	TotalSteps       int            `json:"totalSteps"`       // 总步骤数
	Progress         float64        `json:"progress"`         // 进度百分比 (0-100)
	Status           string         `json:"status"`           // 状态
	Steps            []ProgressStep `json:"steps"`            // 步骤列表
	StartTime        int64          `json:"startTime"`        // 开始时间
	EstimatedEndTime int64          `json:"estimatedEndTime"` // 预计结束时间
	TriggeredBy      string         `json:"triggeredBy"`      // 触发人
	WorkerName       string         `json:"workerName"`       // Worker 名称
}
