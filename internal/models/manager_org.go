package models

// ManagerOrg 机构定义（经理同步用）
//
// 一个「机构」= sys_dept 树上的某个节点（通常是一级部门/分公司/中支）。
// 用户所属机构 = 其部门 ancestors 链上最近的、命中本表的部门节点。
type ManagerOrg struct {
	BaseModel
	OrgName     string  `gorm:"size:100;not null" json:"orgName"`
	DeptID      string  `gorm:"type:uuid;not null;uniqueIndex" json:"deptId"`
	ParentOrgID *string `gorm:"type:uuid;index" json:"parentOrgId,omitempty"`
	OrgType     int     `gorm:"default:1" json:"orgType"` // 0=分公司本部 1=分支机构
	Enabled     bool    `gorm:"default:true" json:"enabled"`
}

func (ManagerOrg) TableName() string { return "sys_manager_org" }
