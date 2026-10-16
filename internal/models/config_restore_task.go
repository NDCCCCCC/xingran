package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RestoreTaskStatus 恢复任务状态（string 枚举，跟随 ConfigBackup 的
// BackupType/StorageType 风格；四态状态机，Phase 93 D-34——不支持取消）。
const (
	RestoreTaskStatusPending RestoreTaskStatus = "pending"
	RestoreTaskStatusRunning RestoreTaskStatus = "running"
	RestoreTaskStatusSuccess RestoreTaskStatus = "success"
	RestoreTaskStatusFailed  RestoreTaskStatus = "failed"
)

// RestoreTaskStatus 恢复任务状态类型
type RestoreTaskStatus string

// ConfigRestoreTask 配置恢复任务（Phase 93 D-15：异步恢复的生命周期载体，
// 与备份版本链 sys_config_backup 职责分离——任务运行时状态在此，恢复成功
// 事件以版本链记录落 sys_config_backup）。
type ConfigRestoreTask struct {
	ID           string     `gorm:"type:uuid;primary_key" json:"id"`
	DeviceID     string     `gorm:"type:uuid;not null;index" json:"deviceId"`
	BackupID     string     `gorm:"type:uuid;not null" json:"backupId"`
	Status       string     `gorm:"size:20;not null;index" json:"status"`
	TotalLines   int        `json:"totalLines,omitempty"`
	SentLines    int        `json:"sentLines,omitempty"`
	FailedLine   string     `gorm:"type:text" json:"failedLine,omitempty"`
	ResultJSON   string     `gorm:"type:text" json:"resultJson,omitempty"`
	ErrorMessage string     `gorm:"type:text" json:"errorMessage,omitempty"`
	StartedAt    *time.Time `json:"startedAt,omitempty"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	CreatedBy    string     `gorm:"size:64" json:"createdBy,omitempty"`
}

// BeforeCreate GORM 钩子：创建前生成 UUID 与时间戳（BaseModel 无 DB default，
// PG 下 uuid 列无 default 会 23502——23502 防御惯例）。
func (c *ConfigRestoreTask) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	if c.UpdatedAt.IsZero() {
		c.UpdatedAt = now
	}
	return nil
}

// BeforeUpdate GORM 钩子：更新前刷新 UpdatedAt
func (c *ConfigRestoreTask) BeforeUpdate(tx *gorm.DB) error {
	c.UpdatedAt = time.Now()
	return nil
}

// TableName 设置表名
func (ConfigRestoreTask) TableName() string {
	return "sys_config_restore_task"
}
