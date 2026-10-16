package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationConfigStatus 通知配置启停状态枚举（0=正常 1=停用）
type NotificationConfigStatus int

const (
	NotificationConfigStatusNormal  NotificationConfigStatus = 0 // 正常
	NotificationConfigStatusStopped NotificationConfigStatus = 1 // 停用
)

// EmailConfig 邮箱服务器配置
type EmailConfig struct {
	ID          string    `gorm:"primaryKey;size:64" json:"id"`
	ConfigName  string    `gorm:"size:100;not null" json:"configName"`
	Host        string    `gorm:"size:255;not null" json:"host"`
	Port        int       `gorm:"default:587;not null" json:"port"`
	Username    string    `gorm:"size:255;not null" json:"username"`
	Password    string    `gorm:"size:500;not null" json:"-"` // 不在JSON中返回密码
	FromName    string    `gorm:"size:100" json:"fromName"`
	FromEmail   string    `gorm:"size:255" json:"fromEmail"`
	UseSSL      bool      `gorm:"default:true" json:"useSsl"`
	UseSTARTTLS bool      `gorm:"column:use_start_tls;default:true" json:"useStartTls"` // 是否使用STARTTLS（false=纯SMTP）
	IsDefault   bool      `gorm:"default:false" json:"isDefault"`
	Status      int       `gorm:"default:0" json:"status"` // 0=正常 1=停用
	Remark      string    `gorm:"size:500" json:"remark"`
	CreatedBy   string    `gorm:"size:64" json:"createdBy"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedBy   string    `gorm:"size:64" json:"updatedBy"`
	UpdatedAt   time.Time `json:"updatedAt"`
	DelFlag     int       `gorm:"default:0" json:"delFlag"`
}

// TableName 指定表名
func (EmailConfig) TableName() string {
	return "sys_email_config"
}

// APIConfigType API配置类型
type APIConfigType string

const (
	APIConfigTypeSMS     APIConfigType = "sms"     // 短信
	APIConfigTypeWebhook APIConfigType = "webhook" // Webhook
	APIConfigTypePush    APIConfigType = "push"    // 推送
)

// AuthType 认证类型
type AuthType string

const (
	AuthTypeNone   AuthType = "none"   // 无认证
	AuthTypeBasic  AuthType = "basic"  // BasicAuth
	AuthTypeBearer AuthType = "bearer" // BearerToken
	AuthTypeAPIKey AuthType = "apikey" // APIKey
)

// MapFields 自定义JSON字段类型
type MapFields map[string]interface{}

// Scan 实现 sql.Scanner 接口
func (m *MapFields) Scan(value interface{}) error {
	if value == nil {
		*m = make(MapFields)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, m)
}

// Value 实现 driver.Valuer 接口
func (m MapFields) Value() (driver.Value, error) {
	if len(m) == 0 {
		return nil, nil
	}
	return json.Marshal(m)
}

// APINotificationConfig API通知配置
type APINotificationConfig struct {
	ID           string        `gorm:"primaryKey;size:64" json:"id"`
	ConfigName   string        `gorm:"size:100;not null" json:"configName"`
	ConfigType   APIConfigType `gorm:"size:50;not null" json:"configType"`
	APIURL       string        `gorm:"size:500;not null" json:"apiUrl"`
	APIMethod    string        `gorm:"size:10;default:POST" json:"apiMethod"`
	Headers      MapFields     `gorm:"type:text" json:"headers"`
	TemplateBody string        `gorm:"type:text" json:"templateBody"`
	AuthType     AuthType      `gorm:"size:50" json:"authType"`
	AuthConfig   MapFields     `gorm:"type:text" json:"authConfig"`
	RetryCount   int           `gorm:"default:3" json:"retryCount"`
	Timeout      int           `gorm:"default:30" json:"timeout"`
	IsDefault    bool          `gorm:"default:false" json:"isDefault"`
	Status       int           `gorm:"default:0" json:"status"` // 0=正常 1=停用
	Remark       string        `gorm:"size:500" json:"remark"`
	CreatedBy    string        `gorm:"size:64" json:"createdBy"`
	CreatedAt    time.Time     `json:"createdAt"`
	UpdatedBy    string        `gorm:"size:64" json:"updatedBy"`
	UpdatedAt    time.Time     `json:"updatedAt"`
	DelFlag      int           `gorm:"default:0" json:"delFlag"`
}

// TableName 指定表名
func (APINotificationConfig) TableName() string {
	return "sys_api_notification_config"
}

// NotificationChannelType 通知渠道类型
type NotificationChannelType string

const (
	ChannelTypeWeb   NotificationChannelType = "web"   // 站内信
	ChannelTypeEmail NotificationChannelType = "email" // 邮件
	ChannelTypeSMS   NotificationChannelType = "sms"   // 短信
	ChannelTypeAPI   NotificationChannelType = "api"   // API通知
)

// NotificationChannel 通知发送渠道配置（关联表）
type NotificationChannel struct {
	ID               string                  `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	NoticeID         string                  `gorm:"type:uuid;not null" json:"noticeId"`
	ChannelType      NotificationChannelType `gorm:"size:20;not null" json:"channelType"`
	EmailConfigID    *string                 `gorm:"type:uuid" json:"emailConfigId"`                     // 邮件配置ID
	APIConfigID      *string                 `gorm:"type:uuid" json:"apiConfigId"`                       // API配置ID
	CustomRecipients *[]string               `gorm:"type:jsonb;serializer:json" json:"customRecipients"` // 自定义收件人列表（邮件地址或企微用户代码）
	CreatedAt        time.Time               `json:"createdAt"`
}

// BeforeCreate GORM钩子 - 创建前自动生成UUID
func (nc *NotificationChannel) BeforeCreate(tx *gorm.DB) error {
	if nc.ID == "" || nc.ID == "00000000-0000-0000-0000-000000000000" {
		nc.ID = uuid.New().String()
	}
	return nil
}

// TableName 指定表名
func (NotificationChannel) TableName() string {
	return "sys_notification_channel"
}
