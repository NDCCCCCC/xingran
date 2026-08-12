package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// TemplateType 模板类型枚举
type TemplateType string

const (
	TemplateTypeInit   TemplateType = "init"   // 初始化模板
	TemplateTypeConfig TemplateType = "config" // 配置模板
	TemplateTypeBackup TemplateType = "backup" // 备份模板
)

// TemplateVariable 模板变量定义
type TemplateVariable struct {
	Name         string   `json:"name"`              // 变量名
	Description  string   `json:"description"`       // 变量描述
	DefaultValue string   `json:"defaultValue"`      // 默认值
	Required     bool     `json:"required"`          // 是否必填
	Type         string   `json:"type"`              // 变量类型: string/int/ip/select等
	Options      []string `json:"options,omitempty"` // 选项（当type为select时）
}

// Variables 实现数据库驱动接口
func (v *TemplateVariable) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal TemplateVariable value: %v", value)
	}
	return json.Unmarshal(bytes, v)
}

// Value 实现数据库驱动接口
func (v TemplateVariable) Value() (driver.Value, error) {
	return json.Marshal(v)
}

// TemplateVariables 模板变量列表（用于数据库存储）
type TemplateVariables []TemplateVariable

// Scan 实现数据库驱动接口
func (tv *TemplateVariables) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal TemplateVariables value: %v", value)
	}
	return json.Unmarshal(bytes, tv)
}

// Value 实现数据库驱动接口
func (tv TemplateVariables) Value() (driver.Value, error) {
	return json.Marshal(tv)
}

// ConfigTemplate 配置模板模型
type ConfigTemplate struct {
	BaseModel
	TemplateName    string            `gorm:"size:100;not null" json:"templateName"`
	TemplateCode    string            `gorm:"size:50;not null;uniqueIndex" json:"templateCode"`
	TemplateType    TemplateType      `gorm:"size:50;not null" json:"templateType"`
	Vendor          DeviceVendor      `gorm:"size:50" json:"vendor,omitempty"`
	DeviceType      DeviceType        `gorm:"size:50" json:"deviceType,omitempty"`
	TemplateContent string            `gorm:"type:text;not null" json:"templateContent"`
	Variables       TemplateVariables `gorm:"type:jsonb" json:"variables,omitempty"`
	Description     string            `gorm:"type:text" json:"description,omitempty"`
	IsSystem        bool              `gorm:"default:false" json:"isSystem"`
}

// TableName 设置表名
func (ConfigTemplate) TableName() string {
	return "sys_config_template"
}
