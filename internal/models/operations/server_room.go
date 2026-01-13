package operations

import (
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// RoomStatus 机房状态枚举
type RoomStatus int

const (
	RoomStatusNormal  RoomStatus = 0 // 正常
	RoomStatusStopped RoomStatus = 1 // 停用
)

// OpsServerRoom 机房模型
type OpsServerRoom struct {
	models.BaseModel
	Name    string     // 机房名称
	Code    string     // 机房编号（唯一）
	FloorID string     // 所在楼层ID
	Status  RoomStatus // 状态: 0=正常, 1=停用
	Remark  string     // 备注
}

// TableName 指定表名
func (OpsServerRoom) TableName() string {
	return "ops_server_rooms"
}
