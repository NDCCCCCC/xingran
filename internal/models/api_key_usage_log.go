package models

import (
	"time"
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

// TableName 设置表名
func (APIKeyUsageLog) TableName() string {
	return "sys_api_key_usage_logs"
}
