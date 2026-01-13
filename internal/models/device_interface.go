package models

import "time"

// DeviceInterface 设备接口信息模型
type DeviceInterface struct {
	ID            string    `gorm:"type:uuid;primary_key" json:"id"`
	DeviceID      string    `gorm:"type:uuid;not null" json:"deviceId"`
	InterfaceName string    `gorm:"size:100;not null" json:"interfaceName"`
	AdminStatus   string    `gorm:"size:20" json:"adminStatus,omitempty"` // up/down
	OperStatus    string    `gorm:"size:20" json:"operStatus,omitempty"`  // up/down
	Description   string    `gorm:"size:200" json:"description,omitempty"`
	MAC           string    `gorm:"size:30" json:"mac,omitempty"`
	Mtu           int64     `json:"mtu,omitempty"`
	Bandwidth     int64     `json:"bandwidth,omitempty"` // bps
	InputBytes    int64     `json:"inputBytes,omitempty"`
	OutputBytes   int64     `json:"outputBytes,omitempty"`
	InputErrors   int64     `json:"inputErrors,omitempty"`
	OutputErrors  int64     `json:"outputErrors,omitempty"`
	CollectedAt   time.Time `gorm:"not null" json:"collectedAt"`
	CreatedAt     time.Time `json:"createdAt"`
}

// TableName 设置表名
func (DeviceInterface) TableName() string {
	return "sys_device_interface"
}
