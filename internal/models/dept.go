package models

// Department 部门模型
type Department struct {
	BaseModel
	DeptName       string  `gorm:"size:100;not null" json:"deptName"`
	DeptCode       string  `gorm:"size:100;uniqueIndex;not null" json:"deptCode"` // 部门编码，用于Excel导入
	ParentID       *string `gorm:"type:uuid" json:"parentId,omitempty"`
	Ancestors      string  `gorm:"size:500;default:''" json:"ancestors"`
	OrderNum       int     `gorm:"default:0" json:"orderNum"`
	Leader         *string `gorm:"size:100" json:"leader,omitempty"`
	LeaderName     *string `gorm:"-" json:"leaderName,omitempty"`     // 负责人姓名（非持久化，仅用于返回）
	LeaderUsername *string `gorm:"-" json:"leaderUsername,omitempty"` // 负责人用户名（非持久化，仅用于返回）
	Phone          *string `gorm:"size:50" json:"phone,omitempty"`
	Email          *string `gorm:"size:100" json:"email,omitempty"`
	IsExternalOrg  int     `gorm:"default:0" json:"isExternalOrg"` // 是否为外部机构：0=否，1=是
	// OrgLevel 机构层级：0=非机构（普通部门/科室）1=分公司 2=中支 3=支公司 4=网点。
	//
	// 只服务数据权限锚点定位（anchor_mode=5）；不参与其它业务分支。
	// 中支含分公司本部等特例必须允许手工覆盖，禁止纯靠祖先深度推导。
	OrgLevel OrgLevel   `gorm:"default:0;index:idx_sys_dept_org_level" json:"orgLevel"`
	Status   DeptStatus `gorm:"default:0" json:"status"`
	Remark   string     `gorm:"size:500" json:"remark,omitempty"`

	// 关联
	Children []*Department `gorm:"foreignKey:ParentID" json:"children,omitempty"`
	Users    []User        `gorm:"foreignKey:DeptID" json:"users,omitempty"`
}

// TableName 设置表名
func (Department) TableName() string {
	return "sys_dept"
}
