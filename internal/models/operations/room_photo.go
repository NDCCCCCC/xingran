package operations

import (
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// OpsRoomPhoto 机房照片关联模型
type OpsRoomPhoto struct {
	models.BaseModel
	RoomID      string // 机房ID
	FileID      string // 文件ID(关联sys_files)
	SortOrder   int    // 排序号
	IsPrimary   bool   // 是否为主图
	Description string // 照片描述
}

// TableName 指定表名
func (OpsRoomPhoto) TableName() string {
	return "ops_room_photos"
}
