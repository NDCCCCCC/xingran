package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/xingran-next/xingran-go-backend/pkg/normalize"
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

// BeforeCreate GORM 钩子：在创建记录前自动生成 UUID + 强制归一化 MAC/接口名
//
// 2026-07-01 根治(port-mac-format-unify): 写入前兜底归一化 MAC(大写+冒号)与
// 接口名(大写短名)。即使上游调用方漏调 normalize,hook 也保证入库格式一致,
// 消灭 verify-format-unify 反复报告的脏数据。归一化实现见 pkg/normalize。
func (d *DeviceMACAddress) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" || d.ID == "00000000-0000-0000-0000-000000000000" {
		d.ID = uuid.New().String()
	}
	d.MACAddress = normalize.MACAddress(d.MACAddress)
	d.InterfaceName = normalize.InterfaceName(d.InterfaceName)
	return nil
}
