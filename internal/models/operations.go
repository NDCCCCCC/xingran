package models

// 运维管理相关模型

// BuildingStatus 楼宇状态
type BuildingStatus int

const (
	BuildingStatusNormal  BuildingStatus = 0 // 正常
	BuildingStatusStopped BuildingStatus = 1 // 停用
)

// Building 楼宇模型
type Building struct {
	BaseModel
	Name     string         `gorm:"size:100;not null" json:"name"`
	Code     string         `gorm:"uniqueIndex;size:50;not null" json:"code"`
	Address  string         `gorm:"size:200" json:"address"`
	OrgID    string         `gorm:"size:64" json:"orgId"`
	OrgName  *string        `gorm:"size:100" json:"orgName,omitempty"`
	Status   BuildingStatus `gorm:"default:0" json:"status"`
	Remark   *string        `gorm:"size:500" json:"remark,omitempty"`
	OrderNum int            `gorm:"default:0" json:"orderNum"`
}

// TableName 设置表名
func (Building) TableName() string {
	return "ops_buildings"
}

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

// LineStatus 专线状态
type LineStatus int

const (
	LineStatusNormal   LineStatus = 0 // 正常
	LineStatusFault    LineStatus = 1 // 故障
	LineStatusDisabled LineStatus = 2 // 停用
)

// LineType 专线类型
type LineType string

const (
	LineTypeInternet   LineType = "internet"    // 互联网专线
	LineTypeMPLS       LineType = "mpls"        // MPLS VPN
	LineTypeLeasedLine LineType = "leased_line" // 租用专线
	LineTypeFiber      LineType = "fiber"       // 光纤专线
)

// DedicatedLine 专线模型
type DedicatedLine struct {
	BaseModel
	Name             string     `gorm:"size:100;not null" json:"name"`
	LineCode         string     `gorm:"uniqueIndex;size:50;not null" json:"lineCode"`
	LineType         LineType   `gorm:"size:50;not null" json:"lineType"`
	Bandwidth        string     `gorm:"size:50" json:"bandwidth"`
	ISP              string     `gorm:"size:100" json:"isp"`
	SourceDeviceID   *string    `gorm:"size:64" json:"sourceDeviceId,omitempty"`
	SourceDeviceName *string    `gorm:"size:100" json:"sourceDeviceName,omitempty"`
	SourcePort       *string    `gorm:"size:50" json:"sourcePort,omitempty"`
	DestDeviceID     *string    `gorm:"size:64" json:"destDeviceId,omitempty"`
	DestDeviceName   *string    `gorm:"size:100" json:"destDeviceName,omitempty"`
	DestPort         *string    `gorm:"size:50" json:"destPort,omitempty"`
	IPAddress        *string    `gorm:"size:50" json:"ipAddress,omitempty"`
	SubnetMask       *string    `gorm:"size:50" json:"subnetMask,omitempty"`
	Gateway          *string    `gorm:"size:50" json:"gateway,omitempty"`
	VLAN             *string    `gorm:"size:50" json:"vlan,omitempty"`
	Status           LineStatus `gorm:"default:0" json:"status"`
	ActivationDate   *string    `gorm:"size:50" json:"activationDate,omitempty"`
	ExpiryDate       *string    `gorm:"size:50" json:"expiryDate,omitempty"`
	ResponsibleID    *string    `gorm:"size:64" json:"responsibleId,omitempty"`
	ResponsibleName  *string    `gorm:"size:100" json:"responsibleName,omitempty"`
	ContractNo       *string    `gorm:"size:100" json:"contractNo,omitempty"`
	MonthlyFee       float64    `gorm:"default:0" json:"monthlyFee"`
	Remark           *string    `gorm:"size:500" json:"remark,omitempty"`
}

// TableName 设置表名
func (DedicatedLine) TableName() string {
	return "ops_dedicated_lines"
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
	DeviceCode       string           `gorm:"uniqueIndex;size:50;not null" json:"deviceCode"`
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
