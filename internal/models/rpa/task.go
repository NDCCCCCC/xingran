package rpa

import (
	"encoding/json"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// TaskStatus 任务状态
type TaskStatus int

const (
	TaskStatusEnabled  TaskStatus = 0 // 启用
	TaskStatusDisabled TaskStatus = 1 // 停用
)

// TaskPriority 任务优先级
type TaskPriority int

const (
	TaskPriorityLow    TaskPriority = 0 // 低
	TaskPriorityMedium TaskPriority = 1 // 中
	TaskPriorityHigh   TaskPriority = 2 // 高
	TaskPriorityUrgent TaskPriority = 3 // 紧急
)

// ScriptAction 脚本动作（供所有模型使用）
type ScriptAction struct {
	Type       string                 `json:"type"`       // click, fill, wait, select, upload, download, navigate, etc.
	Selector   string                 `json:"selector"`   // CSS selector
	Value      string                 `json:"value"`      // 填充值
	Attributes map[string]interface{} `json:"attributes"` // 其他属性
	Timeout    int                    `json:"timeout"`    // 超时时间（毫秒）
	Retry      int                    `json:"retry"`      // 重试次数
}

// Task RPA任务定义
type Task struct {
	models.BaseModel
	TaskName    string          `gorm:"column:name;size:255;not null" json:"taskName"`
	Description string          `gorm:"type:text" json:"description"`
	Script      json.RawMessage `gorm:"type:jsonb;not null" json:"script"`
	Timeout     int             `gorm:"column:timeout_seconds;default:300" json:"timeout"`
	RetryCount  int             `gorm:"column:retry_count;default:0" json:"retryCount"`
	Priority    TaskPriority    `gorm:"default:5" json:"priority"`
	Status      TaskStatus      `gorm:"default:0" json:"status"`
	Tags        string          `gorm:"size:500" json:"tags"`
}

// TableName 指定表名
func (Task) TableName() string {
	return "sys_rpa_tasks"
}

// BeforeCreate GORM钩子
func (t *Task) BeforeCreate(tx *gorm.DB) error {
	_ = t.BaseModel.BeforeCreate(tx)
	return nil
}

// GetActions 获取脚本动作列表
func (t *Task) GetActions() ([]ScriptAction, error) {
	var actions []ScriptAction
	if err := json.Unmarshal(t.Script, &actions); err != nil {
		return nil, err
	}
	return actions, nil
}

// SetActions 设置脚本动作列表
func (t *Task) SetActions(actions []ScriptAction) error {
	data, err := json.Marshal(actions)
	if err != nil {
		return err
	}
	t.Script = data
	return nil
}

// IsEnabled 是否启用
func (t *Task) IsEnabled() bool {
	return t.Status == TaskStatusEnabled
}
