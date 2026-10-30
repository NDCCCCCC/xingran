package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ADLoginEventType AD 设备登录历史事件类型
type ADLoginEventType string

const (
	// ADLoginEventInitial 设备首次发现且 lastLogon 非 nil(基线事件)
	ADLoginEventInitial ADLoginEventType = "initial"
	// ADLoginEventLogin lastLogon 值变大(检测到新登录)
	ADLoginEventLogin ADLoginEventType = "login"
)

// ADLoginHistory AD 域控设备登录历史模型(月分区表 sys_ad_login_history)
//
// 设计要点(quick 261016-ad-login-history):
//   - 表为 PG 分区表(PARTITION BY RANGE (login_time)),由 migration_214 手写 SQL 创建,
//     禁止加入 database.go 的 AutoMigrate 模型列表(GORM 不支持 PARTITION BY,
//     AutoMigrate 对分区父表做 ALTER 会触发 42P16 类问题)。
//   - 时间列全部 TIMESTAMPTZ,从 schema 层根治时区问题(区别于 sys_device_mac_history
//     的 TIMESTAMP 历史包袱),因此本模型刻意不写 AfterFind 时区修补。
type ADLoginHistory struct {
	ID                   string           `gorm:"type:uuid;primary_key" json:"id"`
	ComputerID           string           `gorm:"type:uuid;not null" json:"computerId"`                    // 设备ID快照(非外键)
	ComputerNameSnapshot string           `gorm:"size:255;not null" json:"computerNameSnapshot"`           // 计算机名快照(设备改名后历史可读)
	ADConfigID           string           `gorm:"type:uuid;not null" json:"adConfigId"`                    // 所属AD配置
	LoginTime            time.Time        `gorm:"not null" json:"loginTime"`                               // 登录时间(AD lastLogon 值,分区键)
	DetectedAt           time.Time        `gorm:"not null" json:"detectedAt"`                              // 检测时间(同步时刻)
	LoginUser            string           `gorm:"size:255" json:"loginUser,omitempty"`                     // 登录用户(description parts[1])
	IPAddress            string           `gorm:"size:50" json:"ipAddress,omitempty"`                      // IP 地址(description parts[2])
	EventType            ADLoginEventType `gorm:"size:20;not null" json:"eventType"`                       // initial / login
	CreatedAt            time.Time        `json:"createdAt"`
}

// TableName 设置表名
func (ADLoginHistory) TableName() string {
	return "sys_ad_login_history"
}

// BeforeCreate GORM 钩子:创建记录前自动生成 UUID(仿 DeviceMACHistory)
func (h *ADLoginHistory) BeforeCreate(tx *gorm.DB) error {
	if h.ID == "" || h.ID == "00000000-0000-0000-0000-000000000000" {
		h.ID = uuid.New().String()
	}
	return nil
}
