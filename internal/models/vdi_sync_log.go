package models

import (
	"time"
)

// VDISyncLog VDI同步日志模型
type VDISyncLog struct {
	ID          string     `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ServerID    string     `gorm:"type:uuid;not null;index" json:"serverId"`
	ServerName  string     `gorm:"size:100;not null" json:"serverName"`
	StartTime   time.Time  `gorm:"not null;index" json:"startTime"`
	EndTime     *time.Time `json:"endTime,omitempty"`
	Duration    *int64     `gorm:"comment:同步时长(毫秒)" json:"duration,omitempty"`
	Status      string     `gorm:"size:20;not null;index" json:"status"` // success, failed, partial
	TotalCount  int        `gorm:"default:0" json:"totalCount"`
	SuccessCount int       `gorm:"default:0" json:"successCount"`
	FailCount   int        `gorm:"default:0" json:"failCount"`
	ErrorMsg    *string    `gorm:"type:text" json:"errorMsg,omitempty"`
	BaseModel
}

// TableName 指定表名
func (VDISyncLog) TableName() string {
	return "sys_vdi_sync_log"
}
