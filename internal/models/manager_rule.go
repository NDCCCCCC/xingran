package models

// 经理推导规则的常量
const (
	// 规则适用主体（subject_type）
	ManagerSubjectEmployee   = "employee"    // 普通员工（不是任何部门负责人/一把手）
	ManagerSubjectDeptLeader = "dept_leader" // 部门负责人（某部门 sys_dept.leader 指向的人）
	ManagerSubjectOrgHead    = "org_head"    // 机构一把手（"总经理"类岗位任职人）
	ManagerSubjectPostHolder = "post_holder" // 指定岗位任职人

	// 经理取值来源（manager_type）
	ManagerSourceDeptLeader = "dept_leader" // 部门负责人（sys_dept.leader）
	ManagerSourcePostHolder = "post_holder" // 指定岗位任职人
	ManagerSourceOrgLeader  = "org_leader"  // 机构节点的 leader
	ManagerSourceOrgHead    = "org_head"    // 机构一把手（岗位任职人）

	// 规则作用范围（scope_type）
	ManagerScopeAll      = 0 // 全部
	ManagerScopeDeptTree = 1 // 部门子树
	ManagerScopeOrgTree  = 2 // 机构子树
)

// ManagerOrgSpecialParent 目标机构取值：上级机构
const ManagerOrgSpecialParent = "parent"

// ManagerRule 经理推导规则
//
// 语义：对每个用户，先按 priority 升序找第一条「主体匹配 + 范围匹配」的规则，
// 再按 manager_type 解析出其经理（经理 = 目标用户的 ad_dn）。
type ManagerRule struct {
	BaseModel
	RuleName      string  `gorm:"size:100;not null" json:"ruleName"`
	Priority      int     `gorm:"default:0" json:"priority"`
	Enabled       bool    `gorm:"default:true" json:"enabled"`
	SubjectType   string  `gorm:"size:32;not null" json:"subjectType"`
	SubjectPostID *string `gorm:"type:uuid" json:"subjectPostId,omitempty"`
	SubjectOrgID  *string `gorm:"type:uuid" json:"subjectOrgId,omitempty"`
	ManagerType   string  `gorm:"size:32;not null" json:"managerType"`
	ManagerPostID *string `gorm:"type:uuid" json:"managerPostId,omitempty"`
	ManagerOrgID  *string `gorm:"size:64" json:"managerOrgId,omitempty"`
	ScopeType     int     `gorm:"default:0" json:"scopeType"`
	ScopeID       *string `gorm:"type:uuid" json:"scopeId,omitempty"`
}

func (ManagerRule) TableName() string { return "sys_manager_rule" }
