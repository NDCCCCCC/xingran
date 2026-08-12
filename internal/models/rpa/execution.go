package rpa

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RPAExecutionStatus RPA执行状态
type RPAExecutionStatus string

const (
	RPAExecutionStatusPending   RPAExecutionStatus = "pending"   // 待执行
	RPAExecutionStatusRunning   RPAExecutionStatus = "running"   // 执行中
	RPAExecutionStatusSuccess   RPAExecutionStatus = "success"   // 成功
	RPAExecutionStatusFailed    RPAExecutionStatus = "failed"    // 失败
	RPAExecutionStatusCancelled RPAExecutionStatus = "cancelled" // 已取消
	RPAExecutionStatusTimeout   RPAExecutionStatus = "timeout"   // 超时
)

// StringArray 用于存储字符串数组为 JSON 字符串
// 实现 sql.Scanner 和 driver.Valuer 接口，自动与数据库转换
type StringArray []string

// Scan 实现 sql.Scanner 接口，从数据库读取 JSON 字符串并解析为数组
func (sa *StringArray) Scan(value interface{}) error {
	if value == nil {
		*sa = make(StringArray, 0)
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return errors.New("无法扫描 StringArray")
	}

	if len(bytes) == 0 {
		*sa = make(StringArray, 0)
		return nil
	}

	// 只支持 JSON 格式，如果不是 JSON 则返回空数组
	if !json.Valid(bytes) {
		*sa = make(StringArray, 0)
		return nil
	}

	return json.Unmarshal(bytes, sa)
}

// Value 实现 driver.Valuer 接口，将数组转换为 JSON 字符串存入数据库
func (sa StringArray) Value() (driver.Value, error) {
	if len(sa) == 0 {
		return "[]", nil
	}
	return json.Marshal(sa)
}

// Execution RPA 任务执行记录
type Execution struct {
	ID           string      `gorm:"type:uuid;primary_key" json:"id"`
	CreatedAt    time.Time   `json:"createdAt"`
	UpdatedAt    time.Time   `json:"updatedAt"`
	DeletedAt    *time.Time  `gorm:"index" json:"deletedAt,omitempty"`
	TaskID       string      `gorm:"type:uuid;not null" json:"taskId"`
	TaskName     string      `gorm:"size:255" json:"taskName"`
	WorkerID     *string     `gorm:"type:uuid" json:"workerId"`
	WorkerName   string      `gorm:"size:255" json:"workerName"`
	Status       string      `gorm:"size:20;not null;default:'pending'" json:"status"`  // pending, running, success, failed, cancelled, timeout
	StartTime    *time.Time  `gorm:"column:start_time" json:"startedAt"`                // 前端期望 startedAt
	EndTime      *time.Time  `gorm:"column:end_time" json:"completedAt"`                // 前端期望 completedAt
	Duration     *int        `json:"duration"`                                          // 毫秒
	Step         int         `gorm:"column:progress_current;default:0" json:"step"`     // 前端期望 step
	TotalSteps   int         `gorm:"column:progress_total;default:0" json:"totalSteps"` // 前端期望 totalSteps
	Progress     float64     `gorm:"-" json:"progress"`                                 // 计算属性：进度百分比
	Screenshots  StringArray `gorm:"type:text;default:'[]'" json:"screenshots"`         // JSON 字符串数组
	Logs         string      `gorm:"type:text" json:"logs"`
	ErrorMessage string      `gorm:"column:error_message" json:"error"` // 前端期望 error
	RetryCount   int         `gorm:"default:0" json:"retryCount"`
	TriggeredBy  string      `gorm:"size:64" json:"triggeredBy"`
	TriggerType  string      `gorm:"size:20" json:"triggerType"` // manual, schedule, event, webhook
}

// TableName 指定表名
func (Execution) TableName() string {
	return "sys_rpa_executions"
}

// BeforeCreate GORM钩子 - 创建前生成UUID
func (e *Execution) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	return nil
}

// AfterFind GORM钩子 - 查询后计算进度
func (e *Execution) AfterFind(tx *gorm.DB) error {
	// 计算进度百分比
	if e.TotalSteps > 0 {
		e.Progress = float64(e.Step) / float64(e.TotalSteps) * 100
	} else {
		e.Progress = 0
	}
	return nil
}
