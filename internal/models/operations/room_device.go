package operations

import (
	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// RoomDeviceType 机房设备类型枚举
type RoomDeviceType string

const (
	DeviceTypeServer   RoomDeviceType = "server"   // 服务器
	DeviceTypeStorage  RoomDeviceType = "storage"  // 存储设备
	DeviceTypeUPS      RoomDeviceType = "ups"      // UPS不间断电源
	DeviceTypePDU      RoomDeviceType = "pdu"      // PDU电源分配
	DeviceTypeFirewall RoomDeviceType = "firewall" // 硬件防火墙
	DeviceTypeKVM      RoomDeviceType = "kvm"      // KVM切换器
	DeviceTypeCabinet  RoomDeviceType = "cabinet"  // 机柜
	DeviceTypeOther    RoomDeviceType = "other"    // 其他设备
)

// RoomDeviceStatus 机房设备状态枚举
type RoomDeviceStatus int

const (
	RoomDeviceStatusNormal   RoomDeviceStatus = 0 // 正常
	RoomDeviceStatusFault    RoomDeviceStatus = 1 // 故障
	RoomDeviceStatusScrapped RoomDeviceStatus = 2 // 报废
)

// OpsRoomDevice 机房设备模型
type OpsRoomDevice struct {
	models.BaseModel
	Name             string   `gorm:"size:100;not null" json:"name"`               // 设备名称
	DeviceCode       string   `gorm:"size:200;not null;unique" json:"deviceCode"`   // 设备编码
	DeviceType       string   `gorm:"size:50;not null" json:"deviceType"`          // 设备类型
	Model            *string  `gorm:"size:100" json:"model,omitempty"`             // 设备型号
	Vendor           *string  `gorm:"size:100" json:"vendor,omitempty"`            // 厂商
	SerialNumber     *string  `gorm:"size:100" json:"serialNumber,omitempty"`      // 序列号
	RoomID           string   `gorm:"size:64;not null" json:"roomId"`              // 所在机房ID（关联ops_server_rooms）
	RoomName         *string  `gorm:"size:100" json:"roomName,omitempty"`          // 所在机房名称
	PositionU        int      `gorm:"default:0" json:"positionU"`                  // 机架位置（U）
	PositionDesc     *string  `gorm:"size:200" json:"positionDesc,omitempty"`      // 位置描述（如：第1列第3个机架）
	AssetNumber      *string  `gorm:"size:100" json:"assetNumber,omitempty"`       // 资产编号
	PurchaseDate     *string  `gorm:"size:50" json:"purchaseDate,omitempty"`       // 购买日期
	WarrantyDate     *string  `gorm:"size:50" json:"warrantyDate,omitempty"`       // 保修到期日期
	Status           int      `gorm:"default:0" json:"status"`                     // 状态: 0=正常, 1=故障, 2=报废
	ResponsibleID    *string  `gorm:"size:64" json:"responsibleId,omitempty"`      // 负责人ID
	ResponsibleName  *string  `gorm:"size:100" json:"responsiblename,omitempty"`   // 负责人名称
	PowerConsumption *float64 `gorm:"default:0" json:"powerConsumption,omitempty"` // 功耗（瓦）
	Remark           *string  `gorm:"size:500" json:"remark,omitempty"`            // 备注
}

// TableName 指定表名
func (OpsRoomDevice) TableName() string {
	return "ops_room_devices"
}
