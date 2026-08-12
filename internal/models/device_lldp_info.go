package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LLDPNeighborInfo LLDP邻居信息模型
type LLDPNeighborInfo struct {
	ID               string    `gorm:"type:uuid;primary_key" json:"id"`
	DeviceID         string    `gorm:"type:uuid;not null" json:"deviceId"`
	LocalInterface   string    `gorm:"size:100;not null" json:"localInterface"`
	NeighborID       string    `gorm:"size:100;not null" json:"neighborId"`
	NeighborInterface string   `gorm:"size:100;not null" json:"neighborInterface"`
	NeighborName     string    `gorm:"size:200" json:"neighborName,omitempty"`
	Capabilities     string    `gorm:"size:200" json:"capabilities,omitempty"`
	DiscoveredAt     time.Time `gorm:"not null" json:"discoveredAt"`
	CreatedAt        time.Time `json:"createdAt"`
}

// TableName 设置表名
func (LLDPNeighborInfo) TableName() string {
	return "sys_device_lldp_info"
}

// BeforeCreate GORM 钩子：在创建记录前自动生成 UUID
func (l *LLDPNeighborInfo) BeforeCreate(tx *gorm.DB) error {
	if l.ID == "" || l.ID == "00000000-0000-0000-0000-000000000000" {
		l.ID = uuid.New().String()
	}
	return nil
}
