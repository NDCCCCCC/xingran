package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserColumnConfig 用户列配置
type UserColumnConfig struct {
	BaseModel
	UserID       string `gorm:"type:uuid;not null;index:idx_user_page,priority:1" json:"userId"`
	PageKey      string `gorm:"size:100;not null;index:idx_user_page,priority:2" json:"pageKey"`
	ColumnKey    string `gorm:"size:100;not null" json:"columnKey"`
	Visible      bool   `gorm:"type:bool;default:true" json:"visible"`
	DisplayOrder int    `gorm:"type:int;default:0" json:"displayOrder"`
	Width        int    `gorm:"type:int;default:0" json:"width"`
}

// BeforeCreate GORM钩子 - 创建前
func (u *UserColumnConfig) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}

// TableName 指定表名
func (UserColumnConfig) TableName() string {
	return "sys_user_column_config"
}
