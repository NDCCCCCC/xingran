package rpa

import (
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// Notification RPA 通知配置
type Notification struct {
	models.BaseModel
	Name       string                 `gorm:"size:255;not null" json:"name"`
	TaskID     *string                `gorm:"type:uuid" json:"taskId"`         // NULL 表示全局通知
	Events     string                 `gorm:"size:100;not null" json:"events"` // on_success, on_failure, on_timeout
	Channels   []NotificationChannel  `gorm:"type:jsonb;not null" json:"channels"`
	Recipients map[string]interface{} `gorm:"type:jsonb" json:"recipients"`
	Template   string                 `gorm:"type:text" json:"template"`
	Status     int                    `gorm:"not null;default:0" json:"status"` // 0=enabled 1=disabled
}

// NotificationChannel 通知渠道
type NotificationChannel struct {
	Type   string                 `json:"type"` // email, webhook, dingtalk, wechat
	Config map[string]interface{} `json:"config"`
}

// TableName 指定表名
func (Notification) TableName() string {
	return "sys_rpa_notifications"
}
