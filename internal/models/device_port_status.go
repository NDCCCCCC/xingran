package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Dot1xPortStatus 802.1X端口状态枚举
type Dot1xPortStatus string

const (
	Dot1xStatusAuthorized   Dot1xPortStatus = "authorized"   // 已授权
	Dot1xStatusUnauthorized Dot1xPortStatus = "unauthorized" // 未授权
	Dot1xStatusUnknown      Dot1xPortStatus = "unknown"      // 未知
)

// PortSecurityMode 端口安全模式枚举
type PortSecurityMode string

const (
	PortSecurityModeNone     PortSecurityMode = ""         // 无
	PortSecurityModeProtect  PortSecurityMode = "protect"  // 保护模式
	PortSecurityModeRestrict PortSecurityMode = "restrict" // 限制模式
	PortSecurityModeShutdown PortSecurityMode = "shutdown" // 关闭模式
)

// DevicePortStatus 设备端口状态模型
type DevicePortStatus struct {
	ID            string `gorm:"type:uuid;primary_key" json:"id"`
	DeviceID      string `gorm:"type:uuid;not null;uniqueIndex:uniq_device_interface,priority:1" json:"deviceId"`
	InterfaceName string `gorm:"size:100;not null;uniqueIndex:uniq_device_interface,priority:2" json:"interfaceName"`
	AdminStatus   string `gorm:"size:20" json:"adminStatus,omitempty"`  // 管理状态
	OperStatus    string `gorm:"size:20" json:"operStatus,omitempty"`   // 操作状态
	Description   string `gorm:"size:500" json:"description,omitempty"` // 端口描述

	// 物理属性
	VLAN     *int   `json:"vlan,omitempty"`                    // VLAN ID
	Duplex   string `gorm:"size:20" json:"duplex,omitempty"`   // 双工模式
	Speed    string `gorm:"size:20" json:"speed,omitempty"`    // 速率
	PortType string `gorm:"size:50" json:"portType,omitempty"` // 端口类型

	// 802.1X 状态
	Dot1xEnabled    bool            `gorm:"default:false" json:"dot1xEnabled"`
	Dot1xPortStatus Dot1xPortStatus `gorm:"size:20" json:"dot1xPortStatus,omitempty"`

	// 端口安全配置
	PortSecurityEnabled bool             `gorm:"default:false" json:"portSecurityEnabled"`
	PortSecurityMode    PortSecurityMode `gorm:"size:50" json:"portSecurityMode,omitempty"`
	MaxMACCount         *int             `json:"maxMacCount,omitempty"`
	CurrentMACCount     *int             `json:"currentMacCount,omitempty"`

	CollectedAt time.Time `gorm:"not null" json:"collectedAt"`
	CreatedAt   time.Time `json:"createdAt"`
}

// TableName 设置表名
func (DevicePortStatus) TableName() string {
	return "sys_device_port_status"
}

// BeforeCreate GORM钩子 - 创建前自动生成UUID
func (d *DevicePortStatus) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	return nil
}
