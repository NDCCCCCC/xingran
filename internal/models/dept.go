package models

// Department 部门模型
type Department struct {
	BaseModel
	DeptName   string     `gorm:"size:50;not null" json:"deptName"`
	ParentID   *string    `gorm:"type:uuid" json:"parentId,omitempty"`
	Ancestors  string     `gorm:"size:500;default:''" json:"ancestors"`
	OrderNum   int        `gorm:"default:0" json:"orderNum"`
	Leader     *string    `gorm:"size:32" json:"leader,omitempty"`
	Phone      *string    `gorm:"size:20" json:"phone,omitempty"`
	Email      *string    `gorm:"size:50" json:"email,omitempty"`
	Status     DeptStatus `gorm:"default:0" json:"status"`
	Remark     string     `gorm:"size:500" json:"remark,omitempty"`

	// 关联
	Children   []*Department `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Users      []User        `gorm:"foreignKey:DeptID" json:"users,omitempty"`
}

// TableName 设置表名
func (Department) TableName() string {
	return "sys_dept"
}