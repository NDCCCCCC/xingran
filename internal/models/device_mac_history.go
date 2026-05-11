package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MACEventType MAC地址历史事件类型枚举
type MACEventType string

const (
	EventAppeared    MACEventType = "appeared"    // MAC地址出现
	EventDisappeared MACEventType = "disappeared" // MAC地址消失
	EventMoved       MACEventType = "moved"       // MAC地址移动到其他接口
	EventVLANChanged MACEventType = "vlan_changed" // VLAN变更
)

// DeviceMACHistory 设备MAC地址历史记录模型
type DeviceMACHistory struct {
	ID                  string       `gorm:"type:uuid;primary_key" json:"id"`
	DeviceID            string       `gorm:"type:uuid;not null" json:"deviceId"`
	DeviceNameSnapshot  string       `gorm:"size:100" json:"deviceNameSnapshot"` // 设备名称快照，非外键
	MACAddress          string       `gorm:"size:30;not null" json:"macAddress"`
	InterfaceName       string       `gorm:"size:100;not null" json:"interfaceName"`
	VLANID              *int         `json:"vlanId,omitempty"`
	EventType           MACEventType `gorm:"size:20;not null" json:"eventType"`
	FirstSeen           time.Time    `gorm:"not null" json:"firstSeen"` // 记录首次出现时间
	LastSeen            time.Time    `gorm:"not null" json:"lastSeen"`  // 记录最后看到时间
	CollectedAt         time.Time    `gorm:"not null" json:"collectedAt"` // 采集时间戳
	CreatedAt           time.Time    `json:"createdAt"`
}

// TableName 设置表名
func (DeviceMACHistory) TableName() string {
	return "sys_device_mac_history"
}

// BeforeCreate GORM 钩子：在创建记录前自动生成 UUID
func (d *DeviceMACHistory) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" || d.ID == "00000000-0000-0000-0000-000000000000" {
		d.ID = uuid.New().String()
	}
	return nil
}
