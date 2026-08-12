package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// ExecutionType 执行类型枚举
type ExecutionType string

const (
	ExecutionTypeTemplate ExecutionType = "template" // 模板执行
	ExecutionTypeCommand  ExecutionType = "command"  // 命令执行
)

// ExecutionStatus 执行状态枚举
type ExecutionStatus int

const (
	ExecutionStatusPending   ExecutionStatus = 0 // 待执行
	ExecutionStatusRunning   ExecutionStatus = 1 // 执行中
	ExecutionStatusSuccess   ExecutionStatus = 2 // 成功
	ExecutionStatusFailed    ExecutionStatus = 3 // 失败
	ExecutionStatusCancelled ExecutionStatus = 4 // 已取消
)

// ExecutionStrategy 执行策略枚举
type ExecutionStrategy string

const (
	ExecutionStrategyParallel ExecutionStrategy = "parallel" // 并行执行
	ExecutionStrategySerial   ExecutionStrategy = "serial"   // 串行执行
)

// DeviceIDList 设备ID列表（用于数据库存储）
type DeviceIDList []string

// Scan 实现数据库驱动接口
func (d *DeviceIDList) Scan(value interface{}) error {
	if value == nil {
		*d = make(DeviceIDList, 0)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal DeviceIDList value: %v", value)
	}
	return json.Unmarshal(bytes, d)
}

// Value 实现数据库驱动接口
func (d DeviceIDList) Value() (driver.Value, error) {
	if len(d) == 0 {
		return "[]", nil
	}
	return json.Marshal(d)
}

// ConfigExecution 配置执行记录模型
type ConfigExecution struct {
	ID                string            `gorm:"type:uuid;primary_key" json:"id"`
	ExecutionName     string            `gorm:"size:200;not null" json:"executionName"`
	ExecutionType     ExecutionType     `gorm:"size:50;not null" json:"executionType"`
	TemplateID        *string           `gorm:"type:uuid" json:"templateId,omitempty"`
	DeviceIDs         DeviceIDList      `gorm:"type:jsonb;not null" json:"deviceIds"`
	Status            ExecutionStatus   `gorm:"default:0" json:"status"`
	TotalDevices      int               `gorm:"default:0" json:"totalDevices"`
	SuccessCount      int               `gorm:"default:0" json:"successCount"`
	FailureCount      int               `gorm:"default:0" json:"failureCount"`
	CommandContent    string            `gorm:"type:text" json:"commandContent,omitempty"`
	ExecutionStrategy ExecutionStrategy `gorm:"size:50;default:parallel" json:"executionStrategy"`
	Concurrency       int               `gorm:"default:10" json:"concurrency"`
	Timeout           int               `gorm:"default:300" json:"timeout"`
	StartedAt         *Time             `json:"startedAt,omitempty"`
	CompletedAt       *Time             `json:"completedAt,omitempty"`
	ErrorMessage      string            `gorm:"type:text" json:"errorMessage,omitempty"`
	CreatedBy         string            `gorm:"size:64" json:"createdBy"`
	CreatedAt         Time              `json:"createdAt"`
	UpdatedAt         Time              `json:"updatedAt"`
	DeletedAt         *Time             `gorm:"index" json:"deletedAt,omitempty"`
}

// TableName 设置表名
func (ConfigExecution) TableName() string {
	return "sys_config_execution"
}

// ConfigExecutionDetail 配置执行明细模型
type ConfigExecutionDetail struct {
	ID             string          `gorm:"type:uuid;primary_key" json:"id"`
	ExecutionID    string          `gorm:"type:uuid;not null" json:"executionId"`
	DeviceID       string          `gorm:"type:uuid;not null" json:"deviceId"`
	DeviceName     string          `gorm:"size:100" json:"deviceName,omitempty"`
	IPAddress      string          `gorm:"size:45" json:"ipAddress,omitempty"`
	Status         ExecutionStatus `gorm:"default:0" json:"status"`
	CommandSent    string          `gorm:"type:text" json:"commandSent,omitempty"`
	OutputReceived string          `gorm:"type:text" json:"outputReceived,omitempty"`
	ErrorMessage   string          `gorm:"type:text" json:"errorMessage,omitempty"`
	StartedAt      *Time           `json:"startedAt,omitempty"`
	CompletedAt    *Time           `json:"completedAt,omitempty"`
	Duration       int             `json:"duration,omitempty"` // 执行时长（毫秒）
	CreatedAt      Time            `json:"createdAt"`
	UpdatedAt      Time            `json:"updatedAt"`
}

// TableName 设置表名
func (ConfigExecutionDetail) TableName() string {
	return "sys_config_execution_detail"
}

// Time 时间类型（用于兼容）
type Time = time.Time
