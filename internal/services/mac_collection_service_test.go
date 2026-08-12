package services

import (
	"testing"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	"github.com/xingran-next/xingran-go-backend/tests/fixtures"
	"github.com/stretchr/testify/assert"
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
			expectedMAC: "D8:9E:F3:27:2D:19", // 2026-07-01: parseMACLine 现在写入前归一化为大写+冒号
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
			entry, ok := service.parseMACLine(tt.line, models.VendorHuawei, MACCommandTypeMacTable)
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

// TestParseMACLineRuijie 测试解析锐捷设备MAC地址行（show mac-address-table 格式）
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
			entry, ok := service.parseMACLine(tt.line, models.VendorRuijie, MACCommandTypeMacTable)
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

// TestParseMACLineHuawei 解析华为设备的 parseMACLine 调用现在需要 cmdType 参数,
// 此测试同时验证旧 case 仍按 mac-table 格式工作。
func TestParseMACLineHuaweiSignature(t *testing.T) {
	service := &MACCollectionService{}
	entry, ok := service.parseMACLine("d89e.f327.2d19 100 GigabitEthernet0/1/1 Dynamic", models.VendorHuawei, MACCommandTypeMacTable)
	assert.True(t, ok)
	assert.Equal(t, "D8:9E:F3:27:2D:19", entry.MACAddress) // 2026-07-01: 归一化为大写+冒号
	assert.Equal(t, 100, entry.VLANID)
}

// TestParseRuijiePortSecurityLine 验证锐捷 show port-security address 格式解析
// 格式: Index VLAN MAC Interface(<多 token>) Type LearnAge Action
// 真实设备输出样本（来源: CX-WH-WH-04F-FL-RS8607E-03, 2026-06-29）:
//   56   308   b022.7a2e.4a4f  GigabitEthernet 2/25      Dynamic            --          active
func TestParseRuijiePortSecurityLine(t *testing.T) {
	service := &MACCollectionService{}

	tests := []struct {
		name              string
		line              string
		wantEntry         bool
		expectedMAC       string
		expectedVLAN      int
		expectedInterface string
		expectedType      string
	}{
		{
			name:              "User-reported port-security line (8 fields)",
			line:              "56   308   b022.7a2e.4a4f  GigabitEthernet 2/25      Dynamic            --          active",
			wantEntry:         true,
			expectedMAC:       "B0:22:7A:2E:4A:4F", // 2026-07-03 Phase 47 R5: NormalizeMACAddress 归一 cisco 点分 → canonical 冒号
			expectedVLAN:      308,
			expectedInterface: "GE2/25",            // 2026-07-01: 全称+空格 → 短名
			expectedType:      "Dynamic",
		},
		{
			name:              "Standard port-security with hyphenated MAC (8 fields)",
			line:              "12   100   0011.2233.4455  GigabitEthernet 0/1  SecureDynamic  00:01:23  active",
			wantEntry:         true,
			expectedMAC:       "00:11:22:33:44:55", // 2026-07-03 Phase 47 R5: 归一
			expectedVLAN:      100,
			expectedInterface: "GE0/1",             // 2026-07-01: 全称+空格 → 短名
			expectedType:      "SecureDynamic",
		},
		{
			name:              "Single-token interface (7 fields)",
			line:              "1   200   aabb.ccdd.eeff  Gi0/5  Static  --  active",
			wantEntry:         true,
			expectedMAC:       "AA:BB:CC:DD:EE:FF", // 2026-07-03 Phase 47 R5: 归一
			expectedVLAN:      200,
			expectedInterface: "GE0/5", // Gi 短名经 NormalizeInterfaceName 对称化映射到大写短名
			expectedType:      "Static",
		},
		{
			name:      "Too few fields for port-security format",
			line:      "1 200 aabb.ccdd.eeff Gi0/5",
			wantEntry: false,
		},
		// === Phase 47 R5 (D-04) 负向用例: isCanonicalMAC 拦截解析层垃圾行 ===
		{
			// fields[2] = "Flags:" → NormalizeMACAddress 剥离分隔符后是 "FLAGS" → 非 12 hex → 返回 ""
			// entry.MACAddress == "" 短路,守卫返回 (_, false)
			name:      "R5-D04 Header row 'Flags:' rejected",
			line:      "1   100   Flags:    GigabitEthernet 0/1      Dynamic    --    active",
			wantEntry: false,
		},
		{
			// fields[2] = "Total"(strings.Fields 按空格 split,误认"Total entries: 42" 中 9 个 token)
			//   → NormalizeMACAddress 剥离分隔符后是 "TOTAL" → 非 12 hex → ""
			// entry.MACAddress == "" 守卫短路
			name:      "R5-D04 Summary row 'Total' rejected",
			line:      "1   100   Total entries: 42    --    --    --    active",
			wantEntry: false,
		},
		{
			// fields[2] = "#" → NormalizeMACAddress 剥离后是 "#" → 非 hex → ""
			name:      "R5-D04 Comment line '#' rejected",
			line:      "1   100   #    GigabitEthernet 0/1    Dynamic    --    active",
			wantEntry: false,
		},
		{
			// fields[2] = "GigabitEthernet"(因 MAC 槽被接口名占据 — 真实锐捷 show port-security 在某些版本会这样)
			//   → NormalizeMACAddress 剥离后是 "GIGABITETHERNET" → 非 12 hex → ""
			// entry.MACAddress == "" 守卫短路
			name:      "R5-D04 MAC slot occupied by interface name rejected",
			line:      "1   100   GigabitEthernet 0/1    Dynamic    --    active",
			wantEntry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, ok := service.parseMACLine(tt.line, models.VendorRuijie, MACCommandTypePortSecurity)
			if tt.wantEntry {
				assert.True(t, ok, "Should parse port-security line: %q", tt.line)
				assert.Equal(t, tt.expectedMAC, entry.MACAddress, "MAC mismatch")
				assert.Equal(t, tt.expectedVLAN, entry.VLANID, "VLAN mismatch")
				assert.Equal(t, tt.expectedInterface, entry.InterfaceName, "Interface mismatch")
				assert.Equal(t, tt.expectedType, entry.MACType, "Type mismatch")
			} else {
				assert.False(t, ok, "Should fail to parse: %q", tt.line)
			}
		})
	}
}

// TestGetMACCommands 验证各厂商的 MAC 采集命令列表
// Ruijie/Maipu 返回双命令（show mac-address-table + show port-security address）,
// 其他厂商保持单命令。
func TestGetMACCommands(t *testing.T) {
	service := &MACCollectionService{}

	tests := []struct {
		vendor        models.DeviceVendor
		expectedCmds  []MACCommand
	}{
		{
			models.VendorHuawei,
			[]MACCommand{{Cmd: "display mac-address", Type: MACCommandTypeMacTable}},
		},
		{
			models.VendorH3C,
			[]MACCommand{{Cmd: "display mac-address", Type: MACCommandTypeMacTable}},
		},
		{
			models.VendorRuijie,
			[]MACCommand{
				{Cmd: "show mac-address-table", Type: MACCommandTypeMacTable},
				{Cmd: "show port-security address", Type: MACCommandTypePortSecurity},
			},
		},
		{
			models.VendorMaipu,
			[]MACCommand{
				{Cmd: "show mac-address-table", Type: MACCommandTypeMacTable},
				{Cmd: "show port-security address", Type: MACCommandTypePortSecurity},
			},
		},
		{
			"unknown",
			[]MACCommand{{Cmd: "show mac-address-table", Type: MACCommandTypeMacTable}},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.vendor), func(t *testing.T) {
			cmds := service.getMACCommands(tt.vendor)
			assert.Equal(t, tt.expectedCmds, cmds, "Commands mismatch for vendor %s", tt.vendor)
		})
	}
}

// TestMergeMACEntries 验证多命令结果合并去重
// 同一 MAC 在 mac-address-table 和 port-security address 中都出现时,
// 应只保留一条（不影响后续过滤逻辑的计数）。
func TestMergeMACEntries(t *testing.T) {
	service := &MACCollectionService{}

	// 模拟 mac-address-table 输出
	macTableOutput := `Vlan    Mac Address       Type    Ports
----    -----------       ----    -----
100     0001.0001.0001    DYNAMIC Gi0/1
308     b022.7a2e.4a4f    DYNAMIC GigabitEthernet 2/25`

	// 模拟 port-security address 输出（同一 MAC 也出现在 port-security 表）
	portSecurityOutput := `Vlan    Mac Address       Type           Interface
----    -----------       ----           ---------
308     b022.7a2e.4a4f    Dynamic        GigabitEthernet 2/25`

	// 解析两条命令输出
	cmds := service.getMACCommands(models.VendorRuijie)
	assert.Equal(t, 2, len(cmds), "Ruijie should have 2 commands")

	entriesA, _ := service.parseMACAddressTable(macTableOutput, models.VendorRuijie, MACCommandTypeMacTable)
	entriesB, _ := service.parseMACAddressTable(portSecurityOutput, models.VendorRuijie, MACCommandTypePortSecurity)

	// 合并去重
	merged := service.mergeMACEntries([][]MACAddressEntry{entriesA, entriesB})

	// 应当有 2 条唯一记录 (0001.0001.0001 和 b022.7a2e.4a4f)
	assert.Equal(t, 2, len(merged), "Should have 2 unique MAC entries after dedup")

	// 验证 b022.7a2e.4a4f (归一化后 B0:22:7A:2E:4A:4F) 在合并结果中
	found := false
	for _, e := range merged {
		if e.MACAddress == "B0:22:7A:2E:4A:4F" { // 2026-07-01: parseMACLine 归一化为大写+冒号
			found = true
			assert.Equal(t, 308, e.VLANID)
			assert.Equal(t, "GE2/25", e.InterfaceName) // 2026-07-01: 全称+空格 → 短名
		}
	}
	assert.True(t, found, "B0:22:7A:2E:4A:4F should be in merged result")
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
		normalized := portcollection.NormalizeInterfaceName(mac.InterfaceName)
		macCountByInterface[normalized]++
	}

	// 验证MAC计数(归一化后 key 是短名 GE0/0/1 / GE0/0/2)
	// portcollection.NormalizeInterfaceName 把全称 GigabitEthernet0/0/1 折叠为短名 GE0/0/1
	assert.Equal(t, 2, macCountByInterface["GE0/0/1"])
	assert.Equal(t, 1, macCountByInterface["GE0/0/2"])
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
		portcollection.NormalizeInterfaceName("GigabitEthernet0/0/1"): true, // LLDP邻居端口
	}

	// 模拟过滤逻辑
	threshold := 10
	filteredMACs := []MACAddressEntry{}
	macCountByInterface := make(map[string]int)

	// 计算每个端口的MAC数
	for _, mac := range macAddresses {
		normalized := portcollection.NormalizeInterfaceName(mac.InterfaceName)
		macCountByInterface[normalized]++
	}

	// 应用过滤规则
	for _, mac := range macAddresses {
		normalized := portcollection.NormalizeInterfaceName(mac.InterfaceName)

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
		ne := portcollection.NormalizeInterfaceName(mac.InterfaceName)
		assert.NotEqual(t, "GigabitEthernet0/0/1", ne, "LLDP port should be filtered")
	}
}

// TestParseMACAddressTable_FiltersGarbageLines 锁定 2026-07-01 垃圾行过滤契约:
// 华为 display mac-address 输出的表头/汇总行/注释行(Flags:/Total/#/forwarding logical)
// 必须被丢弃,不能当成 MAC 条目入库污染 mac_address 与 mac_history 轨迹表。
//
// 双层防护: parseMACAddressTable 关键词过滤 + parseMACLine 的 isCanonicalMAC 校验。
func TestParseMACAddressTable_FiltersGarbageLines(t *testing.T) {
	svc := &MACCollectionService{}
	output := `MAC address table of slot 1:
MAC Address    VLAN/  Learned-from      Type
---
00e0-fc12-3456 100/-  GE0/0/1           dynamic
Flags:  -       -      forwarding
# forwarding logical interface, operations cannot be performed
Total items displayed = 1`

	entries, err := svc.parseMACAddressTable(output, models.VendorHuawei, MACCommandTypeMacTable)
	assert.NoError(t, err)
	assert.Len(t, entries, 1, "只应解析出 1 条有效 MAC,表头/汇总/注释行全过滤")
	assert.Equal(t, "00:E0:FC:12:34:56", entries[0].MACAddress)
	assert.Equal(t, "GE0/0/1", entries[0].InterfaceName)
}

// BenchmarkParseMACLine 基准测试MAC地址解析性能
func BenchmarkParseMACLine(b *testing.B) {
	service := &MACCollectionService{}
	line := "d89e.f327.2d19 100 GigabitEthernet0/1/1 Dynamic"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.parseMACLine(line, models.VendorHuawei, MACCommandTypeMacTable)
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
			_ = portcollection.NormalizeInterfaceName(tc)
		}
	}
}

