package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// DiscoveryType 发现类型枚举
type DiscoveryType string

const (
	DiscoveryTypeSNMP DiscoveryType = "snmp" // SNMP发现
	DiscoveryTypeScan DiscoveryType = "scan" // 扫描发现
)

// DiscoveryStatus 发现状态枚举
type DiscoveryStatus int

const (
	DiscoveryStatusPending   DiscoveryStatus = 0 // 待执行
	DiscoveryStatusRunning   DiscoveryStatus = 1 // 执行中
	DiscoveryStatusSuccess   DiscoveryStatus = 2 // 成功
	DiscoveryStatusFailed    DiscoveryStatus = 3 // 失败
	DiscoveryStatusCancelled DiscoveryStatus = 4 // 已取消
)

// IPRange IP范围定义
type IPRange struct {
	StartIP string `json:"startIp"` // 起始IP
	EndIP   string `json:"endIp"`   // 结束IP
}

// IPRangeList IP范围列表（用于数据库存储）
type IPRangeList []IPRange

// Scan 实现数据库驱动接口
func (i *IPRangeList) Scan(value interface{}) error {
	if value == nil {
		*i = make(IPRangeList, 0)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal IPRangeList value: %v", value)
	}
	return json.Unmarshal(bytes, i)
}

// Value 实现数据库驱动接口
func (i IPRangeList) Value() (driver.Value, error) {
	if len(i) == 0 {
		return "[]", nil
	}
	return json.Marshal(i)
}

// DeviceDiscovery 设备发现任务模型
type DeviceDiscovery struct {
	ID              string          `gorm:"type:uuid;primary_key" json:"id"`
	TaskName        string          `gorm:"size:200;not null" json:"taskName"`
	DiscoveryType   DiscoveryType   `gorm:"size:50;not null" json:"discoveryType"`
	IPRanges        IPRangeList     `gorm:"type:jsonb;not null" json:"ipRanges"`
	SNMPCommunity   string          `gorm:"size:100" json:"snmpCommunity,omitempty"`
	SNMPPort        int             `gorm:"default:161" json:"snmpPort"`
	Status          DiscoveryStatus `gorm:"default:0" json:"status"`
	TotalIPs        int             `gorm:"default:0" json:"totalIps"`
	DiscoveredCount int             `gorm:"default:0" json:"discoveredCount"`
	AutoImport      bool            `gorm:"default:false" json:"autoImport"`
	GroupID         *string         `gorm:"type:uuid" json:"groupId,omitempty"` // 关联部门ID
	StartedAt       *time.Time      `json:"startedAt,omitempty"`
	CompletedAt     *time.Time      `json:"completedAt,omitempty"`
	ErrorMessage    string          `gorm:"type:text" json:"errorMessage,omitempty"`
	CreatedBy       string          `gorm:"size:64" json:"createdBy"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
	DeletedAt       *time.Time      `gorm:"index" json:"deletedAt,omitempty"`
}

// TableName 设置表名
func (DeviceDiscovery) TableName() string {
	return "sys_device_discovery"
}
