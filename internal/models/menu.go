package models

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

	// 关联
	Children []Menu `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Roles    []Role `gorm:"many2many:role_menus;" json:"roles,omitempty"`
}

// TableName 设置表名
func (Menu) TableName() string {
	return "sys_menu"
}
