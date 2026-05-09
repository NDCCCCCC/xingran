package models

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MACFilterRule MAC地址过滤规则模型
type MACFilterRule struct {
	ID               string      `gorm:"type:uuid;primary_key" json:"id"`
	RuleName         string      `gorm:"size:100;not null" json:"ruleName"`
	DeviceType       DeviceType  `gorm:"size:50;not null" json:"deviceType"`
	Vendor           DeviceVendor `gorm:"size:50" json:"vendor,omitempty"` // 空字符串表示任意厂商
	MACThreshold     int         `gorm:"not null;default:10" json:"macThreshold"`
	EnableLLDPFilter bool        `gorm:"not null;default:true" json:"enableLLDPFilter"`
	Priority         int         `gorm:"not null;default:0" json:"priority"`
	IsSystem         bool        `gorm:"not null;default:false" json:"isSystem"`
	Remark           string      `gorm:"type:text" json:"remark,omitempty"`
	CreatedBy        string      `gorm:"size:100" json:"createdBy,omitempty"`
	UpdatedBy        string      `gorm:"size:100" json:"updatedBy,omitempty"`
	BaseModel
}

// TableName 设置表名
func (MACFilterRule) TableName() string {
	return "sys_mac_filter_rules"
}

// BeforeCreate GORM 钩子：在创建记录前自动生成 UUID
func (m *MACFilterRule) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" || m.ID == "00000000-0000-0000-0000-000000000000" {
		m.ID = uuid.New().String()
	}
	return nil
}

// Validate 验证规则数据
func (m *MACFilterRule) Validate() error {
	if m.MACThreshold < 0 {
		return fmt.Errorf("MAC阈值必须大于等于0")
	}
	if m.Priority < 0 {
		return fmt.Errorf("优先级必须大于等于0")
	}
	return nil
}
