package models

import "time"

// DeviceAsset 设备资产信息模型
type DeviceAsset struct {
	ID                string    `gorm:"type:uuid;primary_key" json:"id"`
	DeviceID          string    `gorm:"type:uuid;not null" json:"deviceId"`
	SerialNumber      string    `gorm:"size:100" json:"serialNumber,omitempty"`
	HardwareVersion   string    `gorm:"size:50" json:"hardwareVersion,omitempty"`
	FirmwareVersion   string    `gorm:"size:50" json:"firmwareVersion,omitempty"`
	SoftwareVersion   string    `gorm:"size:50" json:"softwareVersion,omitempty"`
	BootROMVersion    string    `gorm:"size:50" json:"bootromVersion,omitempty"`
	SystemDescription string    `gorm:"type:text" json:"systemDescription,omitempty"`
	ProductName       string    `gorm:"size:100" json:"productName,omitempty"`
	DeviceModel       string    `gorm:"size:100" json:"deviceModel,omitempty"`
	Uptime            int64     `json:"uptime,omitempty"`      // seconds
	TotalMemory       int64     `json:"totalMemory,omitempty"` // KB
	FreeMemory        int64     `json:"freeMemory,omitempty"`  // KB
	CPUUsage          float64   `json:"cpuUsage,omitempty"`
	CollectedAt       time.Time `gorm:"not null" json:"collectedAt"`
	CreatedAt         time.Time `json:"createdAt"`
}

// TableName 设置表名
func (DeviceAsset) TableName() string {
	return "sys_device_asset"
}
