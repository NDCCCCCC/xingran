package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// APIKeyUsageLog API密钥使用日志模型
type APIKeyUsageLog struct {
	ID         string    `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"` // 主键
	APIKeyID   string    `gorm:"type:uuid;not null" json:"apiKeyId"`                         // API密钥ID（外键→sys_api_keys）
	UserID     string    `gorm:"type:uuid;not null" json:"userId"`                            // 用户ID（外键→sys_user）
	Method     string    `gorm:"size:10" json:"method"`                                       // HTTP方法
	Path       string    `gorm:"size:500" json:"path"`                                        // 请求路径
	StatusCode int       `json:"statusCode"`                                                 // 响应状态码
	ClientIP   string    `gorm:"size:50" json:"clientIp"`                                     // 客户端IP
	UserAgent  *string   `gorm:"type:text" json:"userAgent,omitempty"`                        // User-Agent字符串
	Duration   int       `json:"duration"`                                                    // 请求耗时（毫秒）
	Success    bool      `json:"success"`                                                     // 是否成功
	CreatedAt  time.Time `gorm:"index" json:"createdAt"`                                      // 创建时间

	// 关联
	APIKey *APIKey `gorm:"foreignKey:APIKeyID" json:"apikey,omitempty"`
	User   *User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// BeforeCreate GORM钩子 - 创建前填充 UUID 主键(ID 为空时)。
// PG 下列自带 default:gen_random_uuid() 兜底;该钩子让 GORM Create 应用层生成 ID,
// 行为等价,同时兼容无函数式默认值的 SQLite (quick-260817-hfl)。
func (l *APIKeyUsageLog) BeforeCreate(tx *gorm.DB) error {
	if l.ID == "" {
		l.ID = uuid.New().String()
	}
	return nil
}

// TableName 设置表名
func (APIKeyUsageLog) TableName() string {
	return "sys_api_key_usage_logs"
}
