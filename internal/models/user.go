package models

import (
	"time"
)

// User 用户模型
type User struct {
	BaseModel
	Username      string     `gorm:"uniqueIndex;type:varchar;not null" json:"username"`
	Password      string     `gorm:"size:128;not null" json:"-"`
	Salt          string     `gorm:"size:64;not null" json:"-"`
	Nickname      *string    `gorm:"type:varchar;column:nickname" json:"nickname,omitempty"`
	EmployeeNo    *string    `gorm:"size:64" json:"employeeNo,omitempty"` // 工号（企业微信代码）
	Email         *string    `gorm:"size:128" json:"email,omitempty"`
	Phone         *string    `gorm:"size:32" json:"phone,omitempty"`
	Avatar        *string    `gorm:"size:255" json:"avatar,omitempty"`
	Gender        Gender     `gorm:"default:2" json:"gender"`
	Status        UserStatus `gorm:"default:0" json:"status"`
	DeptID        *string    `gorm:"size:64" json:"deptId,omitempty"`
	DeptName      *string    `gorm:"size:64" json:"deptName,omitempty"`
	DeptFullName  *string    `gorm:"-" json:"deptFullName,omitempty"` // 完整部门路径（所有层级）
	DeptAncestors *string    `gorm:"column:ancestors;->" json:"-"`     // 只读字段，接收 JOIN 的 ancestors
	LoginIP       *string    `gorm:"size:128" json:"loginIp,omitempty"`
	LoginTime     *time.Time `json:"loginTime,omitempty"`
	PwdUpdateTime *time.Time `json:"pwdUpdateTime,omitempty"`
	PwdExpireDays int        `gorm:"default:90" json:"pwdExpireDays"`
	InitFlag      bool       `gorm:"default:false" json:"initFlag"`
	Remark        string     `gorm:"size:500" json:"remark,omitempty"`

	// AD认证相关字段
	AuthSource string  `gorm:"size:10;default:'local';not null" json:"authSource"`
	ADUsername *string `gorm:"size:100" json:"adUsername,omitempty"`
	AdDn       *string `gorm:"type:text;column:ad_dn" json:"adDn,omitempty"`
	AdOuDn     *string `gorm:"type:text;column:ad_ou_dn" json:"adOuDn,omitempty"`
	AdSyncedAt *time.Time `gorm:"column:ad_synced_at" json:"adSyncedAt,omitempty"`

	// 关联
	Roles   []string    `gorm:"-" json:"roles"`           // 简化为字符串数组，避免复杂关联
	RoleIds []string    `gorm:"-" json:"roleIds,omitempty"` // 角色ID数组，用于前端表单回显
	Posts   []Post      `gorm:"many2many:user_posts;" json:"posts,omitempty"`
	Dept    *Department `gorm:"foreignKey:DeptID" json:"department,omitempty"`
}

// UserRole 用户角色关联
type UserRole struct {
	UserID    string    `gorm:"type:uuid;not null;index:idx_sys_user_role_user_id_role_id" json:"userId"`
	RoleID    string    `gorm:"type:uuid;not null;index:idx_sys_user_role_user_id_role_id" json:"roleId"`
	CreatedAt time.Time `json:"createdAt"`
}

// UserPost 用户岗位关联
type UserPost struct {
	UserID    string    `gorm:"type:uuid;not null" json:"userId"`
	PostID    string    `gorm:"type:uuid;not null" json:"postId"`
	CreatedAt time.Time `json:"createdAt"`
}

// TableName 设置表名
func (User) TableName() string {
	return "sys_user"
}

// TableName 设置表名
func (UserRole) TableName() string {
	return "sys_user_role"
}

// TableName 设置表名
func (UserPost) TableName() string {
	return "sys_user_post"
}
