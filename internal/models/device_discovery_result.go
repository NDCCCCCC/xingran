package models

import "time"

// DeviceDiscoveryResult 设备发现结果明细（Phase 137 / FIX-07）。
//
// 背景：DeviceDiscovery 只保存 DiscoveredCount（发现数量），ExecuteDiscovery
// 产出的设备明细此前仅存在于返回内存中，GetDiscoveryResults 因此恒返回空列表。
// 本模型把每台存活设备持久化成一行，供结果查询端点返回真实数据。
//
// 字段使用 camelCase JSON tag，与前端「发现结果」表格列
// （ipAddress / deviceName / vendor / model / deviceType）对齐。
type DeviceDiscoveryResult struct {
	ID          string    `gorm:"type:uuid;primary_key" json:"id"`
	DiscoveryID string    `gorm:"type:uuid;not null;index:idx_ddr_discovery_id" json:"discoveryId"`
	IPAddress   string    `gorm:"size:64" json:"ipAddress"`
	DeviceType  string    `gorm:"size:64" json:"deviceType"`
	Vendor      string    `gorm:"size:128" json:"vendor"`
	Model       string    `gorm:"size:128" json:"model"`
	SysName     string    `gorm:"size:255" json:"deviceName"`
	SysDescr    string    `gorm:"type:text" json:"sysDescr"`
	IsAlive     bool      `gorm:"default:true" json:"isAlive"`
	// Seq 保留设备在本次发现结果中的原始顺序，避免批量插入后因随机 UUID 主键
	// 导致查询顺序不确定（Phase 137 / FIX-07）。
	Seq       int       `gorm:"default:0" json:"seq"`
	CreatedAt time.Time `json:"createdAt"`
}

// TableName 设置表名
func (DeviceDiscoveryResult) TableName() string {
	return "sys_device_discovery_result"
}
