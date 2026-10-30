package rpa

import (
	"encoding/json"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"gorm.io/gorm"
)

// Template RPA脚本模板
type Template struct {
	models.BaseModel
	TemplateName   string          `gorm:"size:100;not null" json:"templateName"`
	TemplateCode   string          `gorm:"size:100;uniqueIndex;not null" json:"templateCode"`
	Category       string          `gorm:"size:50" json:"category"`
	Description    string          `gorm:"type:text" json:"description"`
	ScriptTemplate json.RawMessage `gorm:"type:jsonb;not null" json:"scriptTemplate"`
	InputSchema    json.RawMessage `gorm:"type:jsonb" json:"inputSchema"`
	IsPublic       bool            `gorm:"default:true" json:"isPublic"`
	Tags           string          `gorm:"type:text" json:"tags"`
	UsageCount     int             `gorm:"default:0" json:"usageCount"`
	Rating         float64         `gorm:"type:decimal(3,2);default:0.00" json:"rating"`
}

// TableName 指定表名
func (Template) TableName() string {
	return "sys_rpa_templates"
}

// BeforeCreate GORM钩子
func (t *Template) BeforeCreate(tx *gorm.DB) error {
	_ = t.BaseModel.BeforeCreate(tx)
	return nil
}

// GetScriptTemplate 获取脚本模板
func (t *Template) GetScriptTemplate() ([]ScriptAction, error) {
	if t.ScriptTemplate == nil {
		return []ScriptAction{}, nil
	}
	var actions []ScriptAction
	if err := json.Unmarshal(t.ScriptTemplate, &actions); err != nil {
		return nil, err
	}
	return actions, nil
}

// SetScriptTemplate 设置脚本模板
func (t *Template) SetScriptTemplate(actions []ScriptAction) error {
	data, err := json.Marshal(actions)
	if err != nil {
		return err
	}
	t.ScriptTemplate = data
	return nil
}

// GetInputSchema 获取输入参数Schema
func (t *Template) GetInputSchema() (map[string]interface{}, error) {
	if t.InputSchema == nil {
		return make(map[string]interface{}), nil
	}
	var schema map[string]interface{}
	if err := json.Unmarshal(t.InputSchema, &schema); err != nil {
		return nil, err
	}
	return schema, nil
}

// SetInputSchema 设置输入参数Schema
func (t *Template) SetInputSchema(schema map[string]interface{}) error {
	data, err := json.Marshal(schema)
	if err != nil {
		return err
	}
	t.InputSchema = data
	return nil
}

// GetTags 获取标签列表
func (t *Template) GetTags() []string {
	if t.Tags == "" {
		return []string{}
	}
	// 简单的逗号分隔处理
	return []string{t.Tags}
}

// IncrementUsage 增加使用次数
func (t *Template) IncrementUsage() {
	t.UsageCount++
}

// IsPublicTemplate 是否公开模板
func (t *Template) IsPublicTemplate() bool {
	return t.IsPublic
}
