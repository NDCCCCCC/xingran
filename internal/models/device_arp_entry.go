package models

import "time"

// DeviceARPEntry 设备ARP表条目模型
type DeviceARPEntry struct {
	ID          string    `gorm:"type:uuid;primary_key" json:"id"`
	DeviceID    string    `gorm:"type:uuid;not null" json:"deviceId"`
	IPAddress   string    `gorm:"size:45;not null" json:"ipAddress"`
	MACAddress  string    `gorm:"size:30;not null" json:"macAddress"`
	Interface   string    `gorm:"size:100" json:"interface,omitempty"`
	Type        string    `gorm:"size:20" json:"type,omitempty"` // dynamic/static
	VLAN        int       `json:"vlan,omitempty"`
	CollectedAt time.Time `gorm:"not null" json:"collectedAt"`
	CreatedAt   time.Time `json:"createdAt"`
}

// TableName 设置表名
func (DeviceARPEntry) TableName() string {
	return "sys_device_arp_entry"
}
