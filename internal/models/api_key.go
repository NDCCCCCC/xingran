package models

import (
	"time"
)

// APIKey API密钥模型
type APIKey struct {
	BaseModel
	Name         string     `gorm:"size:100;not null" json:"name"`                           // 密钥名称
	Key          string     `gorm:"size:100;uniqueIndex;not null" json:"key"`                 // 密钥值（rec_ + 64位hex）
	UserID       *string    `gorm:"type:uuid" json:"userId,omitempty"`                        // 所属用户ID（外键→sys_user）
	ExpiresAt    *time.Time `json:"expiresAt,omitempty"`                                      // 过期时间
	LastUsedAt   *time.Time `json:"lastUsedAt,omitempty"`                                     // 最后使用时间
	IsActive     bool       `gorm:"default:true" json:"isActive"`                             // 是否启用
	Scopes       []string   `gorm:"type:jsonb;serializer:json" json:"scopes"`                 // 作用域 JSON 数组
	IPWhitelist  []string   `gorm:"type:jsonb;serializer:json" json:"ipWhitelist"`            // IP 白名单 JSON 数组
	Description  *string    `gorm:"size:500" json:"description,omitempty"`                    // 描述信息
	InheritPerms bool       `gorm:"default:false" json:"inheritPerms"`                        // 是否继承用户角色权限

	// 关联
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 设置表名
func (APIKey) TableName() string {
	return "sys_api_keys"
}
