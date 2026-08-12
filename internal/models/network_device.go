package models

// DeviceType 设备类型枚举
type DeviceType string

const (
	DeviceTypeRouter       DeviceType = "router"       // 路由器
	DeviceTypeSwitch       DeviceType = "switch"       // 交换机
	DeviceTypeFirewall     DeviceType = "firewall"     // 防火墙
	DeviceTypeAP           DeviceType = "ap"           // 无线接入点
	DeviceTypeLoadBalancer DeviceType = "loadbalancer" // 负载均衡器
)

// DeviceVendor 设备厂商枚举
type DeviceVendor string

const (
	VendorHuawei DeviceVendor = "huawei" // 华为
	VendorH3C    DeviceVendor = "h3c"    // 华三
	VendorRuijie DeviceVendor = "ruijie" // 锐捷
	VendorMaipu  DeviceVendor = "maipu"  // 迈普
)

// DeviceStatus 设备状态枚举
type DeviceStatus int

const (
	DeviceStatusOnline  DeviceStatus = 0 // 在线
	DeviceStatusOffline DeviceStatus = 1 // 离线
	DeviceStatusUnknown DeviceStatus = 2 // 未知
)

// NetworkDevice 网络设备模型
type NetworkDevice struct {
	BaseModel
	DeviceName     string       `gorm:"size:100;not null" json:"deviceName"`
	DeviceType     DeviceType   `gorm:"size:50;not null" json:"deviceType"`
	Vendor         DeviceVendor `gorm:"size:50;not null" json:"vendor"`
	Model          string       `gorm:"size:100" json:"model,omitempty"`
	IPAddress      string       `gorm:"size:45;not null;uniqueIndex" json:"ipAddress"`
	Port           int          `gorm:"default:22" json:"port"`
	SNMPPort       int          `gorm:"default:161" json:"snmpPort"`
	CredentialID   *string      `gorm:"type:uuid" json:"credentialId,omitempty"`
	CredentialName *string      `gorm:"-" json:"credentialName,omitempty"` // 临时字段，不存储到数据库
	DeptID         *string      `gorm:"type:uuid" json:"deptId,omitempty"`
	DeptName       *string      `gorm:"size:100" json:"deptName,omitempty"`
	Location       string       `gorm:"size:200" json:"location,omitempty"`
	Status         DeviceStatus `gorm:"default:2" json:"status"`
	LastSeenAt     *Time        `json:"lastSeenAt,omitempty"`
	LastConfigAt   *Time        `json:"lastConfigAt,omitempty"`
	Description    string       `gorm:"type:text" json:"description,omitempty"`
	// 设备详细信息（通过自动采集更新）
	SerialNumber    string `gorm:"size:100" json:"serialNumber,omitempty"`
	SoftwareVersion string `gorm:"size:100" json:"softwareVersion,omitempty"`
	Uptime          string `gorm:"size:100" json:"uptime,omitempty"`
}

// TableName 设置表名
func (NetworkDevice) TableName() string {
	return "sys_network_device"
}
