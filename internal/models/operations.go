package models

type FloorStatus int

const (
	FloorStatusNormal  FloorStatus = 0
	FloorStatusStopped FloorStatus = 1
)

type Floor struct {
	BaseModel
	Name         string      `gorm:"size:100;not null" json:"name"`
	FloorNo      string      `gorm:"size:50;not null" json:"floorNo"`
	BuildingID   string      `gorm:"size:64;not null" json:"buildingId"`
	BuildingName *string     `gorm:"size:100" json:"buildingName,omitempty"`
	Area         *float64    `gorm:"type:decimal(10,2)" json:"area,omitempty"`
	PlanImageID  *string     `gorm:"size:64" json:"planImageId,omitempty"`
	PlanImageUrl *string     `gorm:"size:500" json:"planImageUrl,omitempty"`
	Status       FloorStatus `gorm:"default:0" json:"status"`
	Remark       *string     `gorm:"size:500" json:"remark,omitempty"`
	OrderNum     int         `gorm:"default:0" json:"orderNum"`
}

func (Floor) TableName() string {
	return "ops_floors"
}

type RoomStatus int

const (
	RoomStatusNormal  RoomStatus = 0
	RoomStatusStopped RoomStatus = 1
)

type ServerRoom struct {
	BaseModel
	Name      string     `gorm:"size:100;not null" json:"name"`
	Code      string     `gorm:"uniqueIndex;size:50;not null" json:"code"`
	FloorID   string     `gorm:"size:64" json:"floorId"`
	FloorName *string    `gorm:"size:100" json:"floorName,omitempty"`
	Status    RoomStatus `gorm:"default:0" json:"status"`
	Remark    *string    `gorm:"size:500" json:"remark,omitempty"`
}

func (ServerRoom) TableName() string {
	return "ops_server_rooms"
}

type RoomDeviceStatus int

const (
	RoomDeviceStatusNormal RoomDeviceStatus = 0
	RoomDeviceStatusFault  RoomDeviceStatus = 1
	RoomDeviceStatusScrap  RoomDeviceStatus = 2
)

type RoomDeviceType string

const (
	RoomDeviceTypeServer   RoomDeviceType = "server"
	RoomDeviceTypeStorage  RoomDeviceType = "storage"
	RoomDeviceTypeUPS      RoomDeviceType = "ups"
	RoomDeviceTypePDU      RoomDeviceType = "pdu"
	RoomDeviceTypeFirewall RoomDeviceType = "firewall"
	RoomDeviceTypeKVM      RoomDeviceType = "kvm"
	RoomDeviceTypeCabinet  RoomDeviceType = "cabinet"
	RoomDeviceTypeOther    RoomDeviceType = "other"
)

type RoomDevice struct {
	BaseModel
	Name             string   `gorm:"size:100;not null" json:"name"`
	DeviceType       string   `gorm:"size:50;not null" json:"deviceType"`
	Model            *string  `gorm:"size:100" json:"model,omitempty"`
	Vendor           *string  `gorm:"size:100" json:"vendor,omitempty"`
	SerialNumber     *string  `gorm:"size:100" json:"serialNumber,omitempty"`
	RoomID           string   `gorm:"size:64" json:"roomId"`
	RoomName         *string  `gorm:"size:100" json:"roomName,omitempty"`
	PositionU        int      `gorm:"default:0" json:"positionU"`
	PositionDesc     *string  `gorm:"size:200" json:"positionDesc,omitempty"`
	AssetNumber      *string  `gorm:"size:100" json:"assetNumber,omitempty"`
	PurchaseDate     *string  `gorm:"size:50" json:"purchaseDate,omitempty"`
	WarrantyDate     *string  `gorm:"size:50" json:"warrantyDate,omitempty"`
	Status           int      `gorm:"default:0" json:"status"`
	ResponsibleID    *string  `gorm:"size:64" json:"responsibleId,omitempty"`
	ResponsibleName  *string  `gorm:"size:100" json:"responsibleName,omitempty"`
	PowerConsumption *float64 `gorm:"default:0" json:"powerConsumption,omitempty"`
	Remark           *string  `gorm:"size:500" json:"remark,omitempty"`
}

func (RoomDevice) TableName() string {
	return "ops_room_devices"
}
