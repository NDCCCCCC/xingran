package operations

import (
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// InfoPointType 信息点类型枚举
type InfoPointType string

const (
	InfoPointTypeNetwork InfoPointType = "network" // 网络信息点
	InfoPointTypePower   InfoPointType = "power"   // 电源信息点
	InfoPointTypeOther   InfoPointType = "other"   // 其他
)

// InfoPointStatus 信息点状态枚举
type InfoPointStatus int

const (
	InfoPointStatusNormal   InfoPointStatus = 0 // 正常
	InfoPointStatusFault    InfoPointStatus = 1 // 故障
	InfoPointStatusDisabled InfoPointStatus = 2 // 停用
)

// OpsInfoPoint 信息点模型
type OpsInfoPoint struct {
	models.BaseModel
	Name            string          `gorm:"size:100;not null" json:"name"`             // 信息点名称
	InfoPointType   InfoPointType   `gorm:"size:50;not null" json:"infoPointType"`     // 信息点类型
	WorkstationID   string          `gorm:"size:64;not null" json:"workstationId"`       // 关联工位ID
	WorkstationName *string `gorm:"->;-:migration" json:"workstationName,omitempty"` // 工位名称（JOIN sys_workstation 动态填充，非物理列）
	BuildingID      *string `gorm:"->;-:migration" json:"buildingId,omitempty"`     // 楼宇ID（JOIN ops_buildings 动态填充，非物理列）
	BuildingName    *string `gorm:"->;-:migration" json:"buildingName,omitempty"`   // 楼宇名称（JOIN ops_buildings 动态填充，非物理列）
	FloorName       *string `gorm:"->;-:migration" json:"floorName,omitempty"`     // 楼层名称（JOIN ops_floors 动态填充，非物理列）
	DeviceID        *string         `gorm:"size:64" json:"deviceId,omitempty"`           // 关联设备ID（sys_network_device）
	DeviceName      *string         `gorm:"size:100" json:"deviceName,omitempty"`      // 设备名称（冗余）
	PortID          *string         `gorm:"size:64" json:"portId,omitempty"`           // 关联设备端口ID（sys_device_port_status）
	PortName        *string         `gorm:"size:100" json:"portName,omitempty"`        // 端口名称（冗余）
	Status          InfoPointStatus `gorm:"default:0" json:"status"`                   // 状态: 0=正常, 1=故障, 2=停用
	Remark          *string         `gorm:"size:500" json:"remark,omitempty"`          // 备注
}

// TableName 指定表名
func (OpsInfoPoint) TableName() string {
	return "ops_info_points"
}
