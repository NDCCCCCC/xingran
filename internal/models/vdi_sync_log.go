package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
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

// BeforeCreate GORM钩子 - 创建前填充 UUID 主键(ID 为空时)。
// 注意:本结构体嵌入 BaseModel 但声明了独立 ID 字段(遮蔽 BaseModel.ID),
// BaseModel.BeforeCreate 无法触达外层 ID,故需显式钩子。
// PG 下列自带 default:gen_random_uuid() 兜底;该钩子行为等价且兼容 SQLite
// (quick-260817-hfl)。
func (l *VDISyncLog) BeforeCreate(tx *gorm.DB) error {
	if l.ID == "" {
		l.ID = uuid.New().String()
	}
	return nil
}

// TableName 指定表名
func (VDISyncLog) TableName() string {
	return "sys_vdi_sync_log"
}
