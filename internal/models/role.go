package models

// Role 角色模型
type Role struct {
	BaseModel
	RoleName          string     `gorm:"uniqueIndex;size:50;not null" json:"roleName"`
	RoleKey           string     `gorm:"uniqueIndex;size:50;not null" json:"roleKey"`
	RoleSort          int        `gorm:"default:0" json:"roleSort"`
	DataScope         DataScope  `gorm:"default:1" json:"dataScope"`
	MenuCheckStrictly bool       `gorm:"default:true" json:"menuCheckStrictly"`
	DeptCheckStrictly bool       `gorm:"default:true" json:"deptCheckStrictly"`
	Status            RoleStatus `gorm:"default:0" json:"status"`
	Remark            string     `gorm:"size:500" json:"remark,omitempty"`

	// 关联
	Menus []Menu       `gorm:"many2many:role_menus;" json:"menus,omitempty"`
	Depts []Department `gorm:"many2many:role_depts;" json:"depts,omitempty"`
	Users []User       `gorm:"many2many:user_roles;" json:"users,omitempty"`
}

// RoleMenu 角色菜单关联
type RoleMenu struct {
	RoleID string `gorm:"type:uuid;not null" json:"roleId"`
	MenuID string `gorm:"type:uuid;not null" json:"menuId"`
}

// RoleDept 角色部门关联
type RoleDept struct {
	RoleID string `gorm:"type:uuid;not null" json:"roleId"`
	DeptID string `gorm:"type:uuid;not null" json:"deptId"`
}

// TableName 设置表名
func (Role) TableName() string {
	return "sys_role"
}

// TableName 设置表名
func (RoleMenu) TableName() string {
	return "sys_role_menu"
}

// TableName 设置表名
func (RoleDept) TableName() string {
	return "sys_role_dept"
}
