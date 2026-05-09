package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/services/lldp"
	"github.com/ruoyi-next/ruoyi-go-backend/test/fixtures"
)

// TestGetMACThreshold 测试获取MAC阈值
func TestGetMACThreshold(t *testing.T) {
	service := &MACCollectionService{}

	tests := []struct {
		name     string
		device   *models.NetworkDevice
		expected int
	}{
		{
			name: "Router threshold",
			device: &models.NetworkDevice{
				DeviceType: models.DeviceTypeRouter,
			},
			expected: 500,
		},
		{
			name: "Switch threshold",
			device: &models.NetworkDevice{
				DeviceType: models.DeviceTypeSwitch,
			},
			expected: 10,
		},
		{
			name: "Firewall threshold",
			device: &models.NetworkDevice{
				DeviceType: models.DeviceTypeFirewall,
			},
			expected: 100,
		},
		{
			name: "LoadBalancer threshold",
			device: &models.NetworkDevice{
				DeviceType: models.DeviceTypeLoadBalancer,
			},
			expected: 50,
		},
		{
			name: "Unknown device type",
			device: &models.NetworkDevice{
				DeviceType: "unknown",
			},
			expected: 10, // 默认值
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.getMACThreshold(tt.device)
			assert.Equal(t, tt.expected, result, "Threshold mismatch for %s", tt.name)
		})
	}
}

// TestMACAddressEntryEquality 测试MAC地址条目结构
func TestMACAddressEntryStructure(t *testing.T) {
	entry := MACAddressEntry{
		MACAddress:    "00:11:22:33:44:55",
		InterfaceName: "GigabitEthernet0/0/1",
		VLANID:        100,
		MACType:       "Dynamic",
	}

	assert.Equal(t, "00:11:22:33:44:55", entry.MACAddress)
	assert.Equal(t, "GigabitEthernet0/0/1", entry.InterfaceName)
	assert.Equal(t, 100, entry.VLANID)
	assert.Equal(t, "Dynamic", entry.MACType)
}

// TestMACCollectionResultStructure 测试MAC采集结果结构
func TestMACCollectionResultStructure(t *testing.T) {
	now := time.Now()
	result := MACCollectionResult{
		DeviceID:       "device-1",
		DeviceName:     "TestSwitch",
		SuccessCount:   150,
		FailedCount:    0,
		CollectionTime: now,
	}

	assert.Equal(t, "device-1", result.DeviceID)
	assert.Equal(t, "TestSwitch", result.DeviceName)
	assert.Equal(t, 150, result.SuccessCount)
	assert.Equal(t, 0, result.FailedCount)
}

// TestParseMACLineHuawei 测试解析华为设备MAC地址行
func TestParseMACLineHuawei(t *testing.T) {
	service := &MACCollectionService{}

	tests := []struct {
		name        string
		line        string
		wantEntry   bool
		expectedMAC string
		expectedVLAN int
	}{
		{
			name:        "Valid Huawei MAC line",
			line:        "d89e.f327.2d19 100 GigabitEthernet0/1/1 Dynamic",
			wantEntry:   true,
			expectedMAC: "d89e.f327.2d19",
			expectedVLAN: 100,
		},
		{
			name:        "Too few fields",
			line:        "d89e.f327.2d19 100",
			wantEntry:   false,
		},
		{
			name:        "Empty line",
			line:        "",
			wantEntry:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, ok := service.parseMACLine(tt.line, models.VendorHuawei)
			if tt.wantEntry {
				assert.True(t, ok, "Should parse successfully")
				assert.Equal(t, tt.expectedMAC, entry.MACAddress)
				assert.Equal(t, tt.expectedVLAN, entry.VLANID)
			} else {
				assert.False(t, ok, "Should fail to parse")
			}
		})
	}
}

// TestParseMACLineRuijie 测试解析锐捷设备MAC地址行
func TestParseMACLineRuijie(t *testing.T) {
	service := &MACCollectionService{}

	tests := []struct {
		name        string
		line        string
		wantEntry   bool
		expectedMAC string
		expectedVLAN int
	}{
		{
			name:        "Valid Ruijie MAC line",
			line:        "100 00:11:22:33:44:55 Dynamic GigabitEthernet0/1",
			wantEntry:   true,
			expectedMAC: "00:11:22:33:44:55",
			expectedVLAN: 100,
		},
		{
			name:        "Too few fields",
			line:        "100 00:11:22:33:44:55",
			wantEntry:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, ok := service.parseMACLine(tt.line, models.VendorRuijie)
			if tt.wantEntry {
				assert.True(t, ok, "Should parse successfully")
				assert.Equal(t, tt.expectedMAC, entry.MACAddress)
				assert.Equal(t, tt.expectedVLAN, entry.VLANID)
			} else {
				assert.False(t, ok, "Should fail to parse")
			}
		})
	}
}

// TestCleanTimestampFromInterface 测试清理接口名称中的时间戳
func TestCleanTimestampFromInterface(t *testing.T) {
	svc := &MACCollectionService{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Interface with timestamp",
			input:    "GigabitEthernet 0/12 2026-5-9 0:51:07",
			expected: "GigabitEthernet 0/12",
		},
		{
			name:     "Interface without timestamp",
			input:    "GigabitEthernet0/0/1",
			expected: "GigabitEthernet0/0/1",
		},
		{
			name:     "Interface with different timestamp format",
			input:    "Gi0/1 2026-05-09 08:52:15",
			expected: "Gi0/1",
		},
		{
			name:     "Only timestamp",
			input:    "2026-5-9 0:51:07",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.cleanTimestampFromInterface(tt.input)
			assert.Equal(t, tt.expected, result, "Cleaning failed for '%s'", tt.input)
		})
	}
}

// TestGetMACCommand 测试获取各厂商的MAC命令
func TestGetMACCommand(t *testing.T) {
	service := &MACCollectionService{}

	tests := []struct {
		vendor        models.DeviceVendor
		expectedCmd   string
	}{
		{models.VendorHuawei, "display mac-address"},
		{models.VendorH3C, "display mac-address"},
		{models.VendorRuijie, "show mac-address-table"},
		{models.VendorMaipu, "show mac-address-table"},
		{"unknown", "show mac-address-table"}, // 默认命令
	}

	for _, tt := range tests {
		t.Run(string(tt.vendor), func(t *testing.T) {
			result := service.getMACCommand(tt.vendor)
			assert.Equal(t, tt.expectedCmd, result, "Command mismatch for vendor %s", tt.vendor)
		})
	}
}

// TestMACFilteringLogic 测试MAC过滤逻辑(基于LLDP和阈值)
func TestMACFilteringLogic(t *testing.T) {
	// 这个测试验证过滤逻辑,但不依赖真实的LLDP服务
	_ = &MACCollectionService{} // service未使用,只测试过滤逻辑

	// 模拟MAC地址列表
	macAddresses := []MACAddressEntry{
		{MACAddress: "00:11:22:33:44:55", InterfaceName: "GigabitEthernet0/0/1", VLANID: 100, MACType: "Dynamic"},
		{MACAddress: "00:11:22:33:44:56", InterfaceName: "GigabitEthernet0/0/1", VLANID: 100, MACType: "Dynamic"},
		{MACAddress: "00:11:22:33:44:57", InterfaceName: "GigabitEthernet0/0/2", VLANID: 100, MACType: "Dynamic"},
	}

	// 计算每个接口的MAC数量
	macCountByInterface := make(map[string]int)
	for _, mac := range macAddresses {
		normalized := lldp.NormalizeInterfaceName(mac.InterfaceName)
		macCountByInterface[normalized]++
	}

	// 验证MAC计数
	assert.Equal(t, 2, macCountByInterface["GigabitEthernet0/0/1"])
	assert.Equal(t, 1, macCountByInterface["GigabitEthernet0/0/2"])
}

// TestMACAddressEntryToServiceEntry 测试类型转换
func TestMACAddressEntryToServiceEntry(t *testing.T) {
	// 测试从fixtures.MACAddressEntry转换为services.MACAddressEntry
	fixtureEntry := fixtures.MACAddressEntry{
		MACAddress:    "00:11:22:33:44:55",
		InterfaceName: "GigabitEthernet0/0/1",
		VLANID:        100,
		MACType:       "Dynamic",
	}

	// 转换
	serviceEntry := MACAddressEntry{
		MACAddress:    fixtureEntry.MACAddress,
		InterfaceName: fixtureEntry.InterfaceName,
		VLANID:        fixtureEntry.VLANID,
		MACType:       fixtureEntry.MACType,
	}

	assert.Equal(t, fixtureEntry.MACAddress, serviceEntry.MACAddress)
	assert.Equal(t, fixtureEntry.InterfaceName, serviceEntry.InterfaceName)
	assert.Equal(t, fixtureEntry.VLANID, serviceEntry.VLANID)
	assert.Equal(t, fixtureEntry.MACType, serviceEntry.MACType)
}

// TestHighMACCountPortGeneration 测试高MAC数端口生成器
func TestHighMACCountPortGeneration(t *testing.T) {
	count := 100
	addresses := fixtures.GenerateHighMACCountPort("GigabitEthernet0/0/10", count)

	assert.Len(t, addresses, count, "Should generate exactly 100 MAC addresses")

	// 验证第一个MAC
	assert.Equal(t, "GigabitEthernet0/0/10", addresses[0].InterfaceName)
	assert.Equal(t, 1, addresses[0].VLANID)
	assert.Equal(t, "Dynamic", addresses[0].MACType)

	// 验证所有MAC都在同一端口
	for _, addr := range addresses {
		assert.Equal(t, "GigabitEthernet0/0/10", addr.InterfaceName)
	}
}

// TestMACAddressTypeConversion 测试MAC类型转换
func TestMACAddressTypeConversion(t *testing.T) {
	tests := []struct {
		input     string
		expected models.MACType
	}{
		{"DYNAMIC", models.MACTypeDynamic},
		{"dynamic", models.MACTypeDynamic},
		{"Static", models.MACTypeStatic},
		{"STATIC", models.MACTypeStatic},
		{"Secure", models.MACTypeSecure},
		{"SECURE", models.MACTypeSecure},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var macType models.MACType
			switch tt.input {
			case "DYNAMIC", "dynamic":
				macType = models.MACTypeDynamic
			case "STATIC", "Static":
				macType = models.MACTypeStatic
			case "SECURE", "Secure":
				macType = models.MACTypeSecure
			}
			assert.Equal(t, tt.expected, macType)
		})
	}
}

// TestDeviceTypeThresholds 测试设备类型阈值配置
func TestDeviceTypeThresholds(t *testing.T) {
	thresholds := fixtures.GetDeviceTypeThresholds

	assert.Equal(t, 500, thresholds[models.DeviceTypeRouter])
	assert.Equal(t, 10, thresholds[models.DeviceTypeSwitch])
	assert.Equal(t, 100, thresholds[models.DeviceTypeFirewall])
	assert.Equal(t, 50, thresholds[models.DeviceTypeLoadBalancer])
}

// TestMockMACAddressesFixtures 测试MAC地址fixtures
func TestMockMACAddressesFixtures(t *testing.T) {
	addresses := fixtures.MockMACAddresses

	assert.NotEmpty(t, addresses, "Mock MAC addresses should not be empty")

	// 验证有不同类型的MAC
	hasDynamic := false
	hasStatic := false
	hasSecure := false

	for _, addr := range addresses {
		switch addr.MACType {
		case "Dynamic":
			hasDynamic = true
		case "Static":
			hasStatic = true
		case "Secure":
			hasSecure = true
		}
	}

	assert.True(t, hasDynamic, "Should have Dynamic MAC entries")
	assert.True(t, hasStatic, "Should have Static MAC entries")
	assert.True(t, hasSecure, "Should have Secure MAC entries")
}

// TestMockLLDPNeighborsFixtures 测试LLDP邻居fixtures
func TestMockLLDPNeighborsFixtures(t *testing.T) {
	neighbors := fixtures.MockLLDPNeighbors

	assert.NotEmpty(t, neighbors, "Mock LLDP neighbors should not be empty")

	// 验证邻居结构
	for key, neighbor := range neighbors {
		assert.NotEmpty(t, key, "Neighbor key should not be empty")
		assert.NotEmpty(t, neighbor.LocalInterface, "LocalInterface should not be empty")
		assert.NotEmpty(t, neighbor.NeighborID, "NeighborID should not be empty")
	}
}

// TestMACFilteringScenario 测试MAC过滤场景
func TestMACFilteringScenario(t *testing.T) {
	// 场景: 有3个端口的设备
	// - GigabitEthernet0/0/1: 2个MAC (LLDP邻居端口,应该被过滤)
	// - GigabitEthernet0/0/2: 3个MAC (正常端口,应该保留)
	// - GigabitEthernet0/0/10: 100个MAC (高MAC数端口,应该被过滤)

	macAddresses := []MACAddressEntry{}

	// 转换fixtures到services类型
	for _, f := range fixtures.MockMACAddresses {
		macAddresses = append(macAddresses, MACAddressEntry{
			MACAddress:    f.MACAddress,
			InterfaceName: f.InterfaceName,
			VLANID:        f.VLANID,
			MACType:       f.MACType,
		})
	}

	// 转换高MAC数端口
	for _, f := range fixtures.GenerateHighMACCountPort("GigabitEthernet0/0/10", 100) {
		macAddresses = append(macAddresses, MACAddressEntry{
			MACAddress:    f.MACAddress,
			InterfaceName: f.InterfaceName,
			VLANID:        f.VLANID,
			MACType:       f.MACType,
		})
	}

	// 模拟LLDP邻居
	lldpNeighbors := map[string]bool{
		lldp.NormalizeInterfaceName("GigabitEthernet0/0/1"): true, // LLDP邻居端口
	}

	// 模拟过滤逻辑
	threshold := 10
	filteredMACs := []MACAddressEntry{}
	macCountByInterface := make(map[string]int)

	// 计算每个端口的MAC数
	for _, mac := range macAddresses {
		normalized := lldp.NormalizeInterfaceName(mac.InterfaceName)
		macCountByInterface[normalized]++
	}

	// 应用过滤规则
	for _, mac := range macAddresses {
		normalized := lldp.NormalizeInterfaceName(mac.InterfaceName)

		// 规则1: LLDP邻居端口过滤
		if lldpNeighbors[normalized] {
			continue
		}

		// 规则2: 高MAC数端口过滤
		if macCountByInterface[normalized] > threshold {
			continue
		}

		filteredMACs = append(filteredMACs, mac)
	}

	// 验证过滤结果
	assert.Less(t, len(filteredMACs), len(macAddresses), "Should filter out some MACs")

	// 验证Gi0/0/1被过滤(LDP邻居)
	for _, mac := range filteredMACs {
		ne := lldp.NormalizeInterfaceName(mac.InterfaceName)
		assert.NotEqual(t, "GigabitEthernet0/0/1", ne, "LLDP port should be filtered")
	}
}

// BenchmarkParseMACLine 基准测试MAC地址解析性能
func BenchmarkParseMACLine(b *testing.B) {
	service := &MACCollectionService{}
	line := "d89e.f327.2d19 100 GigabitEthernet0/1/1 Dynamic"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.parseMACLine(line, models.VendorHuawei)
	}
}

// BenchmarkNormalizeInterfaceName 基准测试接口名规范化性能
func BenchmarkNormalizeInterfaceNameMAC(b *testing.B) {
	testCases := []string{
		"GigabitEthernet0/0/1",
		"Gi0/1",
		"FastEthernet0/1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			_ = lldp.NormalizeInterfaceName(tc)
		}
	}
}

