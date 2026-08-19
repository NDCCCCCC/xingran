package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// ==================== RPA 枚举定义 ====================

// TaskStatus 任务状态枚举
type TaskStatus int

const (
	TaskStatusDraft    TaskStatus = 0 // 草稿
	TaskStatusEnabled  TaskStatus = 1 // 启用
	TaskStatusDisabled TaskStatus = 2 // 停用
)

// TaskPriority 任务优先级枚举
type TaskPriority int

const (
	TaskPriorityLow    TaskPriority = 0 // 低
	TaskPriorityMedium TaskPriority = 1 // 中
	TaskPriorityHigh   TaskPriority = 2 // 高
)

// RPAExecutionStatus RPA 执行状态枚举
type RPAExecutionStatus int

const (
	RPAExecutionStatusPending   RPAExecutionStatus = 0 // 待执行
	RPAExecutionStatusRunning   RPAExecutionStatus = 1 // 执行中
	RPAExecutionStatusCompleted RPAExecutionStatus = 2 // 已完成
	RPAExecutionStatusFailed    RPAExecutionStatus = 3 // 执行失败
	RPAExecutionStatusCancelled RPAExecutionStatus = 4 // 已取消
)

// WorkerStatus Worker 状态枚举
type WorkerStatus int

const (
	WorkerStatusOffline WorkerStatus = 0 // 离线
	WorkerStatusOnline  WorkerStatus = 1 // 在线
	WorkerStatusBusy    WorkerStatus = 2 // 忙碌
	WorkerStatusError   WorkerStatus = 3 // 错误
)

// ScheduleStatus 调度状态枚举
type ScheduleStatus int

const (
	ScheduleStatusActive   ScheduleStatus = 0 // 激活
	ScheduleStatusInactive ScheduleStatus = 1 // 未激活
	ScheduleStatusPaused   ScheduleStatus = 2 // 暂停
)

// RPACredentialStatus RPA 凭证启停状态（簇 A：0=正常, 1=停用）。
// 实体 RPACredential 定义于 internal/models/rpa/credentials.go（Status int,
// check:status IN (0,1)），常量按 Phase 69 DICT-01 约定收敛在本文件——
// status_constants_test.go 的 AST 扫描范围（models/rpa/ 子目录不在扫描路径）。
const (
	RPACredentialStatusNormal  = 0 // 正常
	RPACredentialStatusStopped = 1 // 停用
)

// ScheduleType 调度类型枚举
type ScheduleType string

const (
	ScheduleTypeCron ScheduleType = "cron" // Cron 表达式
	ScheduleTypeOnce ScheduleType = "once" // 单次执行
	ScheduleTypeRate ScheduleType = "rate" // 固定频率
)

// ScriptActionType 脚本动作类型
type ScriptActionType string

const (
	ScriptActionGoto       ScriptActionType = "goto"       // 导航到URL
	ScriptActionClick      ScriptActionType = "click"      // 点击元素
	ScriptActionFill       ScriptActionType = "fill"       // 填写表单
	ScriptActionSelect     ScriptActionType = "select"     // 选择选项
	ScriptActionWait       ScriptActionType = "wait"       // 等待
	ScriptActionScreenshot ScriptActionType = "screenshot" // 截图
	ScriptActionExtract    ScriptActionType = "extract"    // 提取数据
	ScriptActionLoop       ScriptActionType = "loop"       // 循环
	ScriptActionCondition  ScriptActionType = "condition"  // 条件判断
	ScriptActionScroll     ScriptActionType = "scroll"     // 滚动
	ScriptActionHover      ScriptActionType = "hover"      // 悬停
	ScriptActionUpload     ScriptActionType = "upload"     // 上传文件
)

// ==================== RPA 模型定义 ====================

// ScriptAction 脚本动作
type ScriptAction struct {
	ID       string                 `json:"id"`
	Type     ScriptActionType       `json:"type"`
	Selector string                 `json:"selector,omitempty"` // CSS选择器
	Params   map[string]interface{} `json:"params,omitempty"`   // 动作参数
	Timeout  int                    `json:"timeout,omitempty"`  // 超时时间(毫秒)
	Retry    int                    `json:"retry,omitempty"`    // 重试次数
}

// Scan 实现 sql.Scanner 接口
func (a *ScriptAction) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, a)
}

// Value 实现 driver.Valuer 接口
func (a ScriptAction) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// RPATask RPA 任务定义
type RPATask struct {
	BaseModel
	Name            string          `gorm:"size:200;not null" json:"name"`
	Description     string          `gorm:"type:text" json:"description,omitempty"`
	Status          TaskStatus      `gorm:"default:0" json:"status"`
	Priority        TaskPriority    `gorm:"default:1" json:"priority"`
	Script          json.RawMessage `gorm:"type:jsonb" json:"script"` // ScriptAction 数组
	TargetURL       string          `gorm:"size:500" json:"targetUrl,omitempty"`
	Timeout         int             `gorm:"default:300" json:"timeout"` // 超时时间(秒)
	MaxRetry        int             `gorm:"default:3" json:"maxRetry"`  // 最大重试次数
	Tags            string          `gorm:"size:500" json:"tags,omitempty"`
	LastExecutionID *string         `gorm:"type:uuid" json:"lastExecutionId,omitempty"`
	CreatedBy       string          `gorm:"size:64" json:"createdBy"`
	UpdatedBy       string          `gorm:"size:64" json:"updatedBy"`
	Version         int             `json:"version"`
}

// TableName 指定表名
func (RPATask) TableName() string {
	return "sys_rpa_tasks"
}

// RPAWorker RPA Worker 节点
type RPAWorker struct {
	BaseModel
	WorkerID       string       `gorm:"size:100;uniqueIndex;not null" json:"workerId"`
	Name           string       `gorm:"size:200;not null" json:"name"`
	Host           string       `gorm:"size:100" json:"host,omitempty"`
	Port           int          `gorm:"default:3000" json:"port"`
	Status         WorkerStatus `gorm:"default:0" json:"status"`
	MaxConcurrency int          `gorm:"default:3" json:"maxConcurrency"`
	CurrentTasks   int          `gorm:"default:0" json:"currentTasks"`
	TotalExecuted  int          `gorm:"default:0" json:"totalExecuted"`
	TotalFailed    int          `gorm:"default:0" json:"totalFailed"`
	LastHeartbeat  *time.Time   `gorm:"type:timestamp" json:"lastHeartbeat,omitempty"`
	Version        string       `gorm:"size:50" json:"version,omitempty"`
	Capabilities   string       `gorm:"type:jsonb" json:"capabilities,omitempty"` // JSON: 支持的功能列表
	Metadata       string       `gorm:"type:jsonb" json:"metadata,omitempty"`     // JSON: 其他元数据
}

// TableName 指定表名
func (RPAWorker) TableName() string {
	return "sys_rpa_workers"
}

// RPAExecution RPA 执行记录
type RPAExecution struct {
	BaseModel
	TaskID      string             `gorm:"type:uuid;not null;index:idx_rpa_exec_task,priority:1" json:"taskId"`
	WorkerID    string             `gorm:"size:100;index:idx_rpa_exec_worker,priority:1" json:"workerId"`
	Status      RPAExecutionStatus `gorm:"default:0;index:idx_rpa_exec_status,priority:1" json:"status"`
	StartedAt   *time.Time         `gorm:"type:timestamp" json:"startedAt,omitempty"`
	CompletedAt *time.Time         `gorm:"type:timestamp" json:"completedAt,omitempty"`
	Duration    int                `gorm:"default:0" json:"duration"` // 执行时长(秒)
	Progress    int                `gorm:"default:0" json:"progress"` // 进度百分比 0-100
	CurrentStep int                `gorm:"default:0" json:"currentStep"`
	TotalSteps  int                `gorm:"default:0" json:"totalSteps"`
	Logs        string             `gorm:"type:text" json:"logs,omitempty"`        // 日志内容
	Screenshots string             `gorm:"type:text" json:"screenshots,omitempty"` // 截图URL列表(JSON数组)
	Error       string             `gorm:"type:text" json:"error,omitempty"`
	Result      string             `gorm:"type:jsonb" json:"result,omitempty"`      // 执行结果(JSON)
	InputParams string             `gorm:"type:jsonb" json:"inputParams,omitempty"` // 输入参数(JSON)
	CreatedBy   string             `gorm:"size:64" json:"createdBy"`
}

// TableName 指定表名
func (RPAExecution) TableName() string {
	return "sys_rpa_executions"
}

// RPASchedule RPA 定时调度
type RPASchedule struct {
	BaseModel
	TaskID      string         `gorm:"type:uuid;not null;index:idx_rpa_sched_task,priority:1" json:"taskId"`
	Name        string         `gorm:"size:200;not null" json:"name"`
	Type        ScheduleType   `gorm:"size:20;not null" json:"type"`
	CronExpr    string         `gorm:"size:100" json:"cronExpr,omitempty"`        // Cron表达式
	Interval    int            `json:"interval,omitempty"`                        // 间隔(秒)
	RunOnceAt   *time.Time     `gorm:"type:timestamp" json:"runOnceAt,omitempty"` // 单次执行时间
	Status      ScheduleStatus `gorm:"default:1" json:"status"`
	NextRunAt   *time.Time     `gorm:"type:timestamp;index" json:"nextRunAt,omitempty"`
	LastRunAt   *time.Time     `gorm:"type:timestamp" json:"lastRunAt,omitempty"`
	Description string         `gorm:"type:text" json:"description,omitempty"`
	CreatedBy   string         `gorm:"size:64" json:"createdBy"`
}

// TableName 指定表名
func (RPASchedule) TableName() string {
	return "sys_rpa_schedules"
}

// RPAVariable RPA 变量管理
type RPAVariable struct {
	BaseModel
	Key         string `gorm:"size:100;not null;uniqueIndex" json:"key"`
	Value       string `gorm:"type:text" json:"value"`
	Description string `gorm:"size:500" json:"description,omitempty"`
	DataType    string `gorm:"size:20;default:string" json:"dataType"` // string, number, boolean, json
	IsSecret    bool   `gorm:"default:false" json:"isSecret"`          // 是否敏感
	Category    string `gorm:"size:50" json:"category,omitempty"`
	CreatedBy   string `gorm:"size:64" json:"createdBy"`
}

// TableName 指定表名
func (RPAVariable) TableName() string {
	return "sys_rpa_variables"
}

// RPATemplate RPA 脚本模板
type RPATemplate struct {
	BaseModel
	Name        string          `gorm:"size:200;not null" json:"name"`
	Description string          `gorm:"type:text" json:"description,omitempty"`
	Category    string          `gorm:"size:50" json:"category,omitempty"`
	Script      json.RawMessage `gorm:"type:jsonb;not null" json:"script"`      // ScriptAction 数组
	Parameters  string          `gorm:"type:jsonb" json:"parameters,omitempty"` // 参数定义(JSON)
	Icon        string          `gorm:"size:50" json:"icon,omitempty"`
	IsPublic    bool            `gorm:"default:true" json:"isPublic"`
	CreatedBy   string          `gorm:"size:64" json:"createdBy"`
}

// TableName 指定表名
func (RPATemplate) TableName() string {
	return "sys_rpa_templates"
}
