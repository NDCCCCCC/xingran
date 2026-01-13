package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MACType MAC地址类型枚举
type MACType string

const (
	MACTypeDynamic MACType = "dynamic" // 动态学习
	MACTypeStatic  MACType = "static"  // 静态配置
	MACTypeSecure  MACType = "secure"  // 安全MAC
)

// DeviceMACAddress 设备MAC地址模型
type DeviceMACAddress struct {
	ID            string    `gorm:"type:uuid;primary_key" json:"id"`
	DeviceID      string    `gorm:"type:uuid;not null" json:"deviceId"`
	MACAddress    string    `gorm:"size:30;not null" json:"macAddress"`
	InterfaceName string    `gorm:"size:100;not null" json:"interfaceName"`
	VLANID        *int      `json:"vlanId,omitempty"`
	MACType       MACType   `gorm:"size:20" json:"macType,omitempty"`
	CollectedAt   time.Time `gorm:"not null" json:"collectedAt"`
	CreatedAt     time.Time `json:"createdAt"`
}

// TableName 设置表名
func (DeviceMACAddress) TableName() string {
	return "sys_device_mac_address"
}

// BeforeCreate GORM 钩子：在创建记录前自动生成 UUID
func (d *DeviceMACAddress) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" || d.ID == "00000000-0000-0000-0000-000000000000" {
		d.ID = uuid.New().String()
	}
	return nil
}
