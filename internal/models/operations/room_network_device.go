package operations

import (
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// OpsRoomNetworkDevice 机房-网络设备关联模型
type OpsRoomNetworkDevice struct {
	models.BaseModel
	RoomID       string // 机房ID
	DeviceID     string // 网络设备ID（关联sys_network_device）
	PositionU    int    // 机架位置（U）
	PositionDesc string // 位置描述（如：第1列第3个机架）
}

// TableName 指定表名
func (OpsRoomNetworkDevice) TableName() string {
	return "ops_room_network_devices"
}
