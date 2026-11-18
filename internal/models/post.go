package models

// Post 岗位模型
type Post struct {
	BaseModel
	PostCode string `gorm:"uniqueIndex;size:64;not null" json:"postCode"`
	PostName string `gorm:"size:50;not null" json:"postName"`
	PostSort int    `gorm:"default:0" json:"postSort"`
	// AnchorMode / AnchorParam 岗位级机构锚点：被授予该岗位即获得该机构维度授权。
	//
	// 岗位是全局扁平字典（无 dept_id / org_id），锚点靠**任职人所属机构**动态解析，
	// 因此同一个「中支管理员」岗位可分给所有中支的人，各自得到各自的中支。
	// 只允许 0/2/3/5，不允许 All/Custom 语义（防止"岗位授予全库"）。
	AnchorMode  AnchorMode `gorm:"default:0" json:"anchorMode"`
	AnchorParam string     `gorm:"size:64;default:''" json:"anchorParam"`
	Status      PostStatus `gorm:"default:0" json:"status"`
	Remark      string     `gorm:"size:500" json:"remark,omitempty"`

	// 关联
	// Users 岗位任职人。关联表为真实关联表 sys_user_post（见 UserPost 模型），
	// 早期误写为 user_posts 导致 GORM 额外创建一张永远为空的影子关联表。
	Users []User `gorm:"many2many:sys_user_post;" json:"users,omitempty"`
}

// TableName 设置表名
func (Post) TableName() string {
	return "sys_post"
}
