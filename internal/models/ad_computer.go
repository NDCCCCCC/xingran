package models

import (
	"time"
)

// ComputerStatus 电脑状态
type ComputerStatus int

const (
	ComputerStatusOnline  ComputerStatus = 0 // 在线
	ComputerStatusOffline ComputerStatus = 1 // 离线
)

// ADComputer AD域电脑
type ADComputer struct {
	BaseModel
	ADConfigID        string         `gorm:"type:uuid;not null;index:idx_ad_computer_config,priority:1;uniqueIndex:uni_sys_ad_computer_config_name,priority:1;index:idx_ad_computer_dn,priority:1" json:"adConfigId"`
	ComputerName      string         `gorm:"size:255;not null;index:idx_ad_computer_name;uniqueIndex:uni_sys_ad_computer_config_name,priority:2" json:"computerName"`
	DistinguishedName string         `gorm:"size:500;not null;index:idx_ad_computer_dn,priority:2" json:"distinguishedName"`
	LastLogon         *time.Time     `json:"lastLogon,omitempty"`
	PasswordLastSet   *time.Time     `json:"passwordLastSet,omitempty"`
	LogonCount        int            `gorm:"default:0" json:"logonCount"`
	OUDN              string         `gorm:"size:500;index:idx_ad_computer_ou;column:oudn" json:"ouDn,omitempty"`
	Status            ComputerStatus `gorm:"default:0" json:"status"`

	// 解析后的字段（从description中解析）
	OriginalDescription string     `gorm:"type:text" json:"originalDescription,omitempty"` // 原始描述
	IPAddress           string     `gorm:"size:50" json:"ipAddress,omitempty"`             // IP地址
	MacAddress          string     `gorm:"size:50" json:"macAddress,omitempty"`            // MAC地址
	ManagedBy           string     `gorm:"size:255" json:"managedBy,omitempty"`            // 管理者
	OperatingSystem     string     `gorm:"size:255" json:"operatingSystem,omitempty"`      // 操作系统
	OSVersion           string     `gorm:"size:255" json:"osVersion,omitempty"`            // 操作系统版本
	CPUModel            string     `gorm:"size:255" json:"cpuModel,omitempty"`             // CPU型号
	Architecture        string     `gorm:"size:50" json:"architecture,omitempty"`          // 架构(32/64位)
	MemoryCapacity      string     `gorm:"size:50" json:"memoryCapacity,omitempty"`        // 内存容量
	HardDiskCapacity    string     `gorm:"size:50" json:"hardDiskCapacity,omitempty"`      // 硬盘容量
	LastOnlineTime      *time.Time `json:"lastOnlineTime,omitempty"`                       // 最后上线时间
	SerialNumber        string     `gorm:"size:255" json:"serialNumber,omitempty"`         // 序列号
	SystemInfo          string     `gorm:"type:text" json:"systemInfo,omitempty"`          // 系统信息完整描述
}

func (ADComputer) TableName() string {
	return "sys_ad_computer"
}
