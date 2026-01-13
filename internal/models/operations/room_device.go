package operations

import (
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
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
	Name            string            // 设备名称
	DeviceCode      string            // 设备编码（唯一）
	DeviceType      RoomDeviceType    // 设备类型
	Model           string            // 设备型号
	Vendor          string            // 厂商
	SerialNumber    string            // 序列号
	RoomID          string            // 所在机房ID（关联ops_server_rooms）
	PositionU       int               // 机架位置（U）
	PositionDesc    string            // 位置描述（如：第1列第3个机架）
	AssetNumber     string            // 资产编号
	PurchaseDate    *time.Time        // 购买日期
	WarrantyDate    *time.Time        // 保修到期日期
	Status          RoomDeviceStatus  // 状态: 0=正常, 1=故障, 2=报废
	ResponsibleID   string            // 负责人ID（关联sys_user）
	PowerConsumption int               // 功耗（瓦）
	Remark          string            // 备注
}

// TableName 指定表名
func (OpsRoomDevice) TableName() string {
	return "ops_room_devices"
}
