package models

// Post 岗位模型
type Post struct {
	BaseModel
	PostCode string     `gorm:"uniqueIndex;size:64;not null" json:"postCode"`
	PostName string     `gorm:"size:50;not null" json:"postName"`
	PostSort int        `gorm:"default:0" json:"postSort"`
	Status   PostStatus `gorm:"default:0" json:"status"`
	Remark   string     `gorm:"size:500" json:"remark,omitempty"`

	// 关联
	// Users 岗位任职人。关联表为真实关联表 sys_user_post（见 UserPost 模型），
	// 早期误写为 user_posts 导致 GORM 额外创建一张永远为空的影子关联表。
	Users []User `gorm:"many2many:sys_user_post;" json:"users,omitempty"`
}

// TableName 设置表名
func (Post) TableName() string {
	return "sys_post"
}
