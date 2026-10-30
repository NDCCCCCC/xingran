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
	// OrgType 机构层级（对应系统三层架构：分公司 → 中支（含分公司本部）→ 支公司）。
	// 0=分公司 1=中支 2=支公司。
	//
	// 说明：仅用于展示与自动生成时的默认值推导，规则引擎不按该字段分支。
	OrgType int `gorm:"default:1" json:"orgType"`
	Enabled     bool    `gorm:"default:true" json:"enabled"`
}

func (ManagerOrg) TableName() string { return "sys_manager_org" }
