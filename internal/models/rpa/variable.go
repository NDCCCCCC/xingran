package rpa

import (
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// Variable RPA 全局变量
type Variable struct {
	models.BaseModel
	Name        string `gorm:"size:100;not null;unique" json:"name"`
	Value       string `gorm:"type:text;not null" json:"value"`
	Type        string `gorm:"size:20;not null;default:'string'" json:"type"` // string, number, boolean, json
	Description string `gorm:"type:text" json:"description"`
	IsEncrypted bool   `gorm:"default:false" json:"isEncrypted"`
	Status      int    `gorm:"not null;default:0" json:"status"` // 0=enabled 1=disabled
}

// TableName 指定表名
func (Variable) TableName() string {
	return "sys_rpa_variables"
}
