package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// MenuMeta 菜单元数据（JSONB 类型）
// 用于统一管理路由相关元数据：标题、图标、权限、缓存等
type MenuMeta struct {
	Title       string   `json:"title"`                 // 页面标题（必填）
	Icon        string   `json:"icon,omitempty"`        // 图标
	Hidden      bool     `json:"hidden,omitempty"`      // 隐藏菜单（但路由可访问）
	Affix       bool     `json:"affix,omitempty"`       // 固定标签页（不可关闭）
	KeepAlive   bool     `json:"keepAlive,omitempty"`   // 缓存组件
	Permissions []string `json:"permissions,omitempty"` // 需要的权限标识
	Roles       []string `json:"roles,omitempty"`       // 允许的角色
	I18nKey     string   `json:"i18nKey,omitempty"`     // 国际化 key（预留）
	NoCache     bool     `json:"noCache,omitempty"`     // 禁用缓存
	Link        string   `json:"link,omitempty"`        // 外链跳转
}

// Scan 实现 sql.Scanner 接口，用于从数据库读取 JSONB 数据
func (m *MenuMeta) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, m)
}

// Value 实现 driver.Valuer 接口，用于将数据写入数据库
func (m MenuMeta) Value() (driver.Value, error) {
	if len(m.Title) == 0 {
		return nil, nil
	}
	return json.Marshal(m)
}

// Menu 菜单模型
type Menu struct {
	BaseModel
	MenuName  string      `gorm:"size:50;not null" json:"menuName"`
	ParentID  *string     `gorm:"type:uuid" json:"parentId,omitempty"`
	OrderNum  int         `gorm:"default:0" json:"orderNum"`
	Path      *string     `gorm:"size:200" json:"path,omitempty"`
	Component *string     `gorm:"size:255" json:"component,omitempty"`
	MenuType  MenuType    `gorm:"default:'M'" json:"menuType"`
	Visible   VisibleType `gorm:"default:1" json:"visible"`
	Status    MenuStatus  `gorm:"default:0" json:"status"`
	Perms     *string     `gorm:"size:100" json:"perms,omitempty"`
	Icon      *string     `gorm:"size:100" json:"icon,omitempty"`
	Remark    string      `gorm:"size:500" json:"remark,omitempty"`

	// Meta �单元数据（JSONB 类型）
	// 统一管理路由相关元数据，替代原有的分散字段
	Meta *MenuMeta `gorm:"type:jsonb" json:"meta,omitempty"`

	// 关联
	Children []Menu `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Roles    []Role `gorm:"many2many:role_menus;" json:"roles,omitempty"`
}

// TableName 设置表名
func (Menu) TableName() string {
	return "sys_menu"
}
