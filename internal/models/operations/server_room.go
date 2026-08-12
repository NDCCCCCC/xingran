package operations

import (
	"github.com/xingran-next/xingran-go-backend/internal/models"
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
	Name         string     `gorm:"size:100;not null" json:"name"`          // 机房名称
	BuildingID   string     `gorm:"size:64;not null" json:"buildingId"`     // 所属楼宇ID
	BuildingName *string    `gorm:"size:100" json:"buildingName,omitempty"` // 所属楼宇名称
	FloorID      string     `gorm:"size:64;not null" json:"floorId"`        // 所在楼层ID
	FloorName    *string    `gorm:"size:100" json:"floorName,omitempty"`    // 所在楼层名称
	Status       RoomStatus `gorm:"default:0" json:"status"`                // 状态: 0=正常, 1=停用
	Remark       *string    `gorm:"size:500" json:"remark,omitempty"`       // 备注
}

// TableName 指定表名
func (OpsServerRoom) TableName() string {
	return "ops_server_rooms"
}
