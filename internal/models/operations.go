package models

// 运维管理相关模型

// FloorStatus 楼层状态
type FloorStatus int

const (
	FloorStatusNormal  FloorStatus = 0 // 正常
	FloorStatusStopped FloorStatus = 1 // 停用
)

// Floor 楼层模型
type Floor struct {
	BaseModel
	Name         string      `gorm:"size:100;not null" json:"name"`
	FloorNo      string      `gorm:"size:50;not null" json:"floorNo"`
	BuildingID   string      `gorm:"size:64;not null" json:"buildingId"`
	BuildingName *string     `gorm:"size:100" json:"buildingName,omitempty"`
	Status       FloorStatus `gorm:"default:0" json:"status"`
	Remark       *string     `gorm:"size:500" json:"remark,omitempty"`
	OrderNum     int         `gorm:"default:0" json:"orderNum"`
}

// TableName 设置表名
func (Floor) TableName() string {
	return "ops_floors"
}

// RoomStatus 机房状态
type RoomStatus int

const (
	RoomStatusNormal  RoomStatus = 0 // 正常
	RoomStatusStopped RoomStatus = 1 // 停用
)

// ServerRoom 机房模型
type ServerRoom struct {
	BaseModel
	Name      string     `gorm:"size:100;not null" json:"name"`
	Code      string     `gorm:"uniqueIndex;size:50;not null" json:"code"`
	FloorID   string     `gorm:"size:64" json:"floorId"`
	FloorName *string    `gorm:"size:100" json:"floorName,omitempty"`
	Status    RoomStatus `gorm:"default:0" json:"status"`
	Remark    *string    `gorm:"size:500" json:"remark,omitempty"`
}

// TableName 设置表名
func (ServerRoom) TableName() string {
	return "ops_server_rooms"
}

// RoomDeviceStatus 机房设备状态
type RoomDeviceStatus int

const (
	RoomDeviceStatusNormal RoomDeviceStatus = 0 // 正常
	RoomDeviceStatusFault  RoomDeviceStatus = 1 // 故障
	RoomDeviceStatusScrap  RoomDeviceStatus = 2 // 报废
)

// RoomDeviceType 机房设备类型
type RoomDeviceType string

const (
	RoomDeviceTypeServer   RoomDeviceType = "server"   // 服务器
	RoomDeviceTypeStorage  RoomDeviceType = "storage"  // 存储设备
	RoomDeviceTypeUPS      RoomDeviceType = "ups"      // UPS
	RoomDeviceTypePDU      RoomDeviceType = "pdu"      // PDU
	RoomDeviceTypeFirewall RoomDeviceType = "firewall" // 防火墙
	RoomDeviceTypeKVM      RoomDeviceType = "kvm"      // KVM
	RoomDeviceTypeCabinet  RoomDeviceType = "cabinet"  // 机柜
	RoomDeviceTypeOther    RoomDeviceType = "other"    // 其他
)

// RoomDevice 机房设备模型
type RoomDevice struct {
	BaseModel
	Name             string           `gorm:"size:100;not null" json:"name"`
	DeviceType       RoomDeviceType   `gorm:"size:50;not null" json:"deviceType"`
	Model            *string          `gorm:"size:100" json:"model,omitempty"`
	Vendor           *string          `gorm:"size:100" json:"vendor,omitempty"`
	SerialNumber     *string          `gorm:"size:100" json:"serialNumber,omitempty"`
	RoomID           string           `gorm:"size:64" json:"roomId"`
	RoomName         *string          `gorm:"size:100" json:"roomName,omitempty"`
	PositionU        int              `gorm:"default:0" json:"positionU"`
	PositionDesc     *string          `gorm:"size:200" json:"positionDesc,omitempty"`
	AssetNumber      *string          `gorm:"size:100" json:"assetNumber,omitempty"`
	PurchaseDate     *string          `gorm:"size:50" json:"purchaseDate,omitempty"`
	WarrantyDate     *string          `gorm:"size:50" json:"warrantyDate,omitempty"`
	Status           RoomDeviceStatus `gorm:"default:0" json:"status"`
	ResponsibleID    *string          `gorm:"size:64" json:"responsibleId,omitempty"`
	ResponsibleName  *string          `gorm:"size:100" json:"responsibleName,omitempty"`
	PowerConsumption *float64         `gorm:"default:0" json:"powerConsumption,omitempty"`
	Remark           *string          `gorm:"size:500" json:"remark,omitempty"`
}

// TableName 设置表名
func (RoomDevice) TableName() string {
	return "ops_room_devices"
}
