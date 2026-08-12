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
	Users []User `gorm:"many2many:user_posts;" json:"users,omitempty"`
}

// TableName 设置表名
func (Post) TableName() string {
	return "sys_post"
}
