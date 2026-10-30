package fixtures

import (
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// MACAddressEntry MAC地址条目（测试用，避免循环导入services包）
type MACAddressEntry struct {
	MACAddress    string
	InterfaceName string
	VLANID        int
	MACType       string
}

// MockMACAddresses 模拟MAC地址条目
// 用于测试MAC采集和过滤逻辑
var MockMACAddresses = []MACAddressEntry{
	{
		MACAddress:    "00:11:22:33:44:55",
		InterfaceName: "GigabitEthernet0/0/1",
		VLANID:        100,
		MACType:       "Dynamic",
	},
	{
		MACAddress:    "00:11:22:33:44:56",
		InterfaceName: "GigabitEthernet0/0/1",
		VLANID:        100,
		MACType:       "Dynamic",
	},
	{
		MACAddress:    "00:11:22:33:44:57",
		InterfaceName: "GigabitEthernet0/0/2",
		VLANID:        100,
		MACType:       "Dynamic",
	},
	{
		MACAddress:    "00:11:22:33:44:58",
		InterfaceName: "GigabitEthernet0/0/2",
		VLANID:        100,
		MACType:       "Dynamic",
	},
	{
		MACAddress:    "00:11:22:33:44:59",
		InterfaceName: "GigabitEthernet0/0/2",
		VLANID:        100,
		MACType:       "Dynamic",
	},
	// Uplink port with many MACs (模拟上行链路端口)
	{
		MACAddress:    "00:11:22:33:44:60",
		InterfaceName: "GigabitEthernet0/0/10",
		VLANID:        1,
		MACType:       "Dynamic",
	},
	// Static MAC entry
	{
		MACAddress:    "00:11:22:33:44:70",
		InterfaceName: "GigabitEthernet0/0/3",
		VLANID:        200,
		MACType:       "Static",
	},
	// Secure MAC entry
	{
		MACAddress:    "00:11:22:33:44:80",
		InterfaceName: "GigabitEthernet0/0/4",
		VLANID:        300,
		MACType:       "Secure",
	},
}

// GenerateHighMACCountPort 生成包含大量MAC地址的端口
// 用于测试MAC数量阈值过滤
// 参数:
//   - interfaceName: 接口名称
//   - count: MAC地址数量
// 返回:
//   - MAC地址条目列表
func GenerateHighMACCountPort(interfaceName string, count int) []MACAddressEntry {
	addresses := make([]MACAddressEntry, count)
	for i := 0; i < count; i++ {
		addresses[i] = MACAddressEntry{
			MACAddress:    fmt.Sprintf("00:11:22:33:44:%02x", i%256),
			InterfaceName: interfaceName,
			VLANID:        1,
			MACType:       "Dynamic",
		}
	}
	return addresses
}

// MockLLDPNeighbors 模拟LLDP邻居信息
// map的key使用规范化接口名
var MockLLDPNeighbors = map[string]*models.LLDPNeighborInfo{
	"gi0/01": {
		LocalInterface:   "GigabitEthernet0/0/1",
		NeighborID:       "3815.sep12.eth0",
		NeighborInterface: "eth0",
		NeighborName:     "60P-28",
		Capabilities:     "Router",
	},
	"gi0/02": {
		LocalInterface:   "GigabitEthernet0/0/2",
		NeighborID:       "3815.sep13.eth0",
		NeighborInterface: "eth0",
		NeighborName:     "60P-29",
		Capabilities:     "Bridge",
	},
}

// MockLLDPNeighborsEmpty 空LLDP邻居信息
var MockLLDPNeighborsEmpty = map[string]*models.LLDPNeighborInfo{}

// MockNetworkDevices 模拟网络设备列表
// 注意: 使用BaseModel的ID字段,IPAddress字段
var MockNetworkDevices = []models.NetworkDevice{
	{
		BaseModel:   models.BaseModel{ID: "device-1"},
		DeviceName:  "TestSwitch-Core",
		DeviceType:  models.DeviceTypeSwitch,
		Vendor:      models.VendorHuawei,
		IPAddress:   "192.168.1.1",
		Status:      models.DeviceStatusOnline,
	},
	{
		BaseModel:   models.BaseModel{ID: "device-2"},
		DeviceName:  "TestRouter-Edge",
		DeviceType:  models.DeviceTypeRouter,
		Vendor:      models.VendorRuijie,
		IPAddress:   "192.168.1.2",
		Status:      models.DeviceStatusOnline,
	},
	{
		BaseModel:   models.BaseModel{ID: "device-3"},
		DeviceName:  "TestFirewall",
		DeviceType:  models.DeviceTypeFirewall,
		Vendor:      models.VendorH3C,
		IPAddress:   "192.168.1.3",
		Status:      models.DeviceStatusOnline,
	},
}

// MockLLDPNeighborsByDevice 按设备组织的LLDP邻居信息
var MockLLDPNeighborsByDevice = map[string]map[string]*models.LLDPNeighborInfo{
	"device-1": {
		"gi0/01": {
			LocalInterface:   "GigabitEthernet0/0/1",
			NeighborID:       "3815.sep12.eth0",
			NeighborInterface: "eth0",
			NeighborName:     "60P-28",
			Capabilities:     "Router",
		},
		"gi0/02": {
			LocalInterface:   "GigabitEthernet0/0/2",
			NeighborID:       "3815.sep13.eth0",
			NeighborInterface: "eth0",
			NeighborName:     "60P-29",
			Capabilities:     "Bridge",
		},
	},
	"device-2": {
		"gi0/01": {
			LocalInterface:   "GigabitEthernet0/0/1",
			NeighborID:       "router-2.core",
			NeighborInterface: "Port-Channel1",
			NeighborName:     "CoreRouter",
			Capabilities:     "Router",
		},
	},
	"device-3": {}, // 防火墙设备没有LLDP邻居
}

// MockMACAddressesByInterface 按接口组织的MAC地址列表
var MockMACAddressesByInterface = map[string][]MACAddressEntry{
	"GigabitEthernet0/0/1": {
		{
			MACAddress:    "00:11:22:33:44:55",
			InterfaceName: "GigabitEthernet0/0/1",
			VLANID:        100,
			MACType:       "Dynamic",
		},
		{
			MACAddress:    "00:11:22:33:44:56",
			InterfaceName: "GigabitEthernet0/0/1",
			VLANID:        100,
			MACType:       "Dynamic",
		},
	},
	"GigabitEthernet0/0/2": {
		{
			MACAddress:    "00:11:22:33:44:57",
			InterfaceName: "GigabitEthernet0/0/2",
			VLANID:        100,
			MACType:       "Dynamic",
		},
		{
			MACAddress:    "00:11:22:33:44:58",
			InterfaceName: "GigabitEthernet0/0/2",
			VLANID:        100,
			MACType:       "Dynamic",
		},
		{
			MACAddress:    "00:11:22:33:44:59",
			InterfaceName: "GigabitEthernet0/0/2",
			VLANID:        100,
			MACType:       "Dynamic",
		},
	},
}

// GetDeviceTypeThresholds 获取各设备类型的MAC阈值
// 用于测试阈值配置
var GetDeviceTypeThresholds = map[models.DeviceType]int{
	models.DeviceTypeRouter:       500,
	models.DeviceTypeSwitch:       10,
	models.DeviceTypeFirewall:     100,
	models.DeviceTypeLoadBalancer: 50,
}

// MockFilterRules 模拟过滤规则
var MockFilterRules = []models.MACFilterRule{
	{
		ID:               "rule-1",
		RuleName:         "华为交换机规则",
		DeviceType:       models.DeviceTypeSwitch,
		Vendor:           models.VendorHuawei,
		MACThreshold:     20,
		EnableLLDPFilter: true,
		Priority:         10,
		IsSystem:         false,
	},
	{
		ID:               "rule-2",
		RuleName:         "通用交换机规则",
		DeviceType:       models.DeviceTypeSwitch,
		Vendor:           "",
		MACThreshold:     10,
		EnableLLDPFilter: true,
		Priority:         0,
		IsSystem:         true,
	},
	{
		ID:               "rule-3",
		RuleName:         "路由器规则",
		DeviceType:       models.DeviceTypeRouter,
		Vendor:           "",
		MACThreshold:     500,
		EnableLLDPFilter: true,
		Priority:         0,
		IsSystem:         true,
	},
}
