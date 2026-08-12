package models

import "time"

// EnrichmentStatus 补充任务状态
type EnrichmentStatus string

const (
	EnrichmentStatusPending   EnrichmentStatus = "pending"   // 待执行
	EnrichmentStatusRunning   EnrichmentStatus = "running"   // 执行中
	EnrichmentStatusSuccess   EnrichmentStatus = "success"   // 成功
	EnrichmentStatusFailed    EnrichmentStatus = "failed"    // 失败
	EnrichmentStatusCancelled EnrichmentStatus = "cancelled" // 已取消
)

// DeviceEnrichmentTask 设备信息采集任务
// 用于跟踪通过 SSH 后台异步获取设备详细信息的任务
type DeviceEnrichmentTask struct {
	BaseModel
	DeviceID     string           `gorm:"size:36;not null;index:idx_device_enrichment_device" json:"deviceId"` // 设备ID
	Status       EnrichmentStatus `gorm:"size:20;not null;default:'pending'" json:"status"`                    // 任务状态
	StartedAt    *time.Time       `gorm:"type:timestamp" json:"startedAt,omitempty"`                           // 开始时间
	CompletedAt  *time.Time       `gorm:"type:timestamp" json:"completedAt,omitempty"`                         // 完成时间
	ErrorMessage string           `gorm:"type:text" json:"errorMessage,omitempty"`                             // 错误信息
	// 采集获取的信息
	EnrichedModel        *string `gorm:"size:100" json:"enrichedModel,omitempty"`           // 采集获取的设备型号
	EnrichedSerialNumber *string `gorm:"size:100" json:"enrichedSerialNumber,omitempty"`    // 采集获取的序列号
	EnrichedSoftwareVer  *string `gorm:"size:100" json:"enrichedSoftwareVersion,omitempty"` // 采集获取的软件版本
	EnrichedUptime       *string `gorm:"size:100" json:"enrichedUptime,omitempty"`          // 采集获取的运行时间
}

// TableName 设置表名
func (DeviceEnrichmentTask) TableName() string {
	return "net_device_enrichment_task"
}
