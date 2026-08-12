package models

import "time"

// MACOUIVendor IEEE OUI厂商信息表
type MACOUIVendor struct {
	OUIPrefix  string    `gorm:"primaryKey;column:oui_prefix;type:varchar(6);notNull" json:"oui_prefix"` // AABBCC格式
	VendorName string    `gorm:"column:vendor_name;type:varchar(255);notNull" json:"vendor_name"`
	UpdatedAt  time.Time `gorm:"column:updated_at;type:timestamp;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

// TableName 设置表名
func (MACOUIVendor) TableName() string {
	return "sys_mac_oui_vendor"
}
