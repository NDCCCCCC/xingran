package operations

import (
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// OpsRoomPhoto 机房照片关联模型
type OpsRoomPhoto struct {
	models.BaseModel
	RoomID      string  `gorm:"size:64;not null" json:"roomId"`        // 机房ID
	FileID      string  `gorm:"size:64;not null" json:"fileId"`        // 文件ID(关联sys_files)
	FileURL     *string `gorm:"size:500" json:"fileUrl,omitempty"`     // 文件URL
	SortOrder   int     `gorm:"default:0" json:"sortOrder"`            // 排序号
	IsPrimary   bool    `gorm:"default:false" json:"isPrimary"`        // 是否为主图
	Description *string `gorm:"size:500" json:"description,omitempty"` // 照片描述
}

// TableName 指定表名
func (OpsRoomPhoto) TableName() string {
	return "ops_room_photos"
}
