package operations

import (
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// OpsRoomNetworkDevice 机房-网络设备关联模型
type OpsRoomNetworkDevice struct {
	models.BaseModel
	RoomID       string  `gorm:"size:64;not null" json:"roomId"`         // 机房ID
	DeviceID     string  `gorm:"size:64;not null" json:"deviceId"`       // 网络设备ID（关联sys_network_device）
	PositionU    int     `gorm:"default:0" json:"positionU"`             // 机架位置（U）
	PositionDesc *string `gorm:"size:200" json:"positionDesc,omitempty"` // 位置描述（如：第1列第3个机架）
}

// TableName 指定表名
func (OpsRoomNetworkDevice) TableName() string {
	return "ops_room_network_devices"
}
