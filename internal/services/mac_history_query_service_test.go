package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// TestValidateMACAddress 测试MAC地址验证
func TestValidateMACAddress(t *testing.T) {
	tests := []struct {
		name    string
		mac     string
		wantErr bool
	}{
		{"合法MAC（大写无分隔符）", "AABBCCDDEEFF", false},
		{"合法MAC（小写无分隔符）", "aabbccddeeff", false},
		{"合法MAC（冒号分隔）", "AA:BB:CC:DD:EE:FF", false},
		{"合法MAC（点分）", "aabb.ccddeeff", false},
		{"合法MAC（横线分隔）", "AA-BB-CC-DD-EE-FF", false},
		{"非法MAC（字符不足）", "AABBCC", true},
		{"非法MAC（字符过多）", "AABBCCDDEEFFGG", true},
		{"非法MAC（非法字符）", "AA:BB:CC:DD:EE:GG", true},
		{"空MAC", "", true},
		{"非法MAC（格式错误）", "AA-BB-CC-DD-EE", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMACAddress(tt.mac)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestExtractOUIPrefix 测试OUI前缀提取
func TestExtractOUIPrefix(t *testing.T) {
	tests := []struct {
		name     string
		mac      string
		expected string
	}{
		{"标准格式", "AA:BB:CC:DD:EE:FF", "AABBCC"},
		{"点分格式", "aabb.ccddeeff", "AABBCC"},
		{"横线分隔", "AA-BB-CC-DD-EE-FF", "AABBCC"},
		{"无分隔符", "aabbccddeeff", "AABBCC"},
		{"空字符串", "", ""},
		{"短MAC", "AA:BB", "AABB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractOUIPrefix(tt.mac)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAggregateTrajectory 测试轨迹聚合逻辑
func TestAggregateTrajectory(t *testing.T) {
	// 创建测试用的service实例
	service := &macHistoryQueryServiceImpl{}

	// 准备测试数据：模拟从LAG查询返回的原始事件
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	vlan100 := 100
	vlan200 := 200

	rawEvents := []rawEvent{
		// 同一位置连续状态（应该合并）
		{
			ID:                 "id1",
			DeviceID:           "device1",
			DeviceNameSnapshot: "Switch-01",
			MACAddress:         "AABBCCDDEEFF",
			InterfaceName:      "Gi0/1",
			VLANID:             &vlan100,
			EventType:          "appeared",
			FirstSeen:          baseTime,
			LastSeen:           baseTime.Add(1 * time.Hour),
		},
		{
			ID:                 "id2",
			DeviceID:           "device1",
			DeviceNameSnapshot: "Switch-01",
			MACAddress:         "AABBCCDDEEFF",
			InterfaceName:      "Gi0/1",
			VLANID:             &vlan100,
			EventType:          "moved",
			FirstSeen:          baseTime.Add(1 * time.Hour),
			LastSeen:           baseTime.Add(2 * time.Hour),
		},
		// 不同位置（应该分开）
		{
			ID:                 "id3",
			DeviceID:           "device1",
			DeviceNameSnapshot: "Switch-01",
			MACAddress:         "AABBCCDDEEFF",
			InterfaceName:      "Gi0/2",
			VLANID:             &vlan200,
			EventType:          "moved",
			FirstSeen:          baseTime.Add(3 * time.Hour),
			LastSeen:           baseTime.Add(4 * time.Hour),
		},
	}

	// 执行聚合
	nodes := service.aggregateTrajectory(rawEvents)

	// 验证结果
	assert.Equal(t, 2, len(nodes), "应该聚合为2个节点")

	// 第一个节点：合并了前两个连续状态
	firstNode := nodes[0]
	assert.Equal(t, "device1", firstNode.DeviceID)
	assert.Equal(t, "Switch-01", firstNode.DeviceName)
	assert.Equal(t, "Gi0/1", firstNode.Interface)
	assert.Equal(t, 100, *firstNode.VLANID)
	assert.Equal(t, "AABBCCDDEEFF", firstNode.MACAddress, "第一个节点的MACAddress应与rawEvent一致")
	assert.Equal(t, baseTime, firstNode.StartTime)
	assert.Equal(t, baseTime.Add(2*time.Hour), firstNode.EndTime)
	assert.Equal(t, int64(7200), firstNode.Duration, "停留时长应为2小时")

	// 第二个节点：不同位置
	secondNode := nodes[1]
	assert.Equal(t, "Gi0/2", secondNode.Interface)
	assert.Equal(t, 200, *secondNode.VLANID)
	assert.Equal(t, "AABBCCDDEEFF", secondNode.MACAddress, "第二个节点的MACAddress也应正确填充")
	assert.Equal(t, baseTime.Add(3*time.Hour), secondNode.StartTime)
}

// TestAggregateTrajectory_MACAddressPropagation 验证 TrajectoryNode.MACAddress
// 字段在所有节点(包括合并区间后)上均被正确填充(Phase 13 CR-03 修复覆盖)。
func TestAggregateTrajectory_MACAddressPropagation(t *testing.T) {
	service := &macHistoryQueryServiceImpl{}

	baseTime := time.Date(2025, 6, 1, 8, 0, 0, 0, time.UTC)
	vlan10 := 10
	vlan20 := 20
	const expectedMAC = "AABBCCDDEEFF"

	rawEvents := []rawEvent{
		// 同一位置连续状态(被合并为一个 TrajectoryNode)
		{
			ID:                 "m1",
			DeviceID:           "dev-A",
			DeviceNameSnapshot: "Core-Switch-A",
			MACAddress:         expectedMAC,
			InterfaceName:      "GigabitEthernet0/1",
			VLANID:             &vlan10,
			EventType:          "appeared",
			FirstSeen:          baseTime,
			LastSeen:           baseTime.Add(30 * time.Minute),
		},
		{
			ID:                 "m2",
			DeviceID:           "dev-A",
			DeviceNameSnapshot: "Core-Switch-A",
			MACAddress:         expectedMAC,
			InterfaceName:      "GigabitEthernet0/1",
			VLANID:             &vlan10,
			EventType:          "moved",
			FirstSeen:          baseTime.Add(30 * time.Minute),
			LastSeen:           baseTime.Add(1 * time.Hour),
		},
		// 不同位置(产生第二个 TrajectoryNode)
		{
			ID:                 "m3",
			DeviceID:           "dev-B",
			DeviceNameSnapshot: "Access-Switch-B",
			MACAddress:         expectedMAC,
			InterfaceName:      "FastEthernet0/2",
			VLANID:             &vlan20,
			EventType:          "moved",
			FirstSeen:          baseTime.Add(1 * time.Hour),
			LastSeen:           baseTime.Add(2 * time.Hour),
		},
	}

	nodes := service.aggregateTrajectory(rawEvents)

	// 断言:3 个 rawEvent 应聚合为 2 个节点(前 2 个合并)
	require.Equal(t, 2, len(nodes), "前2个rawEvent应合并为1个节点")

	// 所有节点的 MACAddress 必须非空且等于 expectedMAC
	for i, node := range nodes {
		assert.Equal(t, expectedMAC, node.MACAddress,
			"节点 #%d (device=%s, interface=%s) 的MACAddress应等于rawEvent中的MAC",
			i, node.DeviceID, node.Interface)
		assert.NotEmpty(t, node.MACAddress,
			"节点 #%d 的MACAddress字段不能为空(tooltip需要展示)", i)
	}
}

// TestAggregateTrajectory_MACAddressJSONSerialization 验证 TrajectoryNode
// 序列化到 JSON 时 MACAddress 字段名为 "mac"(小写单字段名),且与其他 camelCase
// 字段(deviceId/deviceName/interface/vlanId/eventType/startTime/endTime/duration)
// 一致。这是前端 MACTrajectoryChart tooltip 读取 node.mac 的合约保证
// (Phase 13 CR-03 + 13-08 W5)。
func TestAggregateTrajectory_MACAddressJSONSerialization(t *testing.T) {
	vlan10 := 10
	node := TrajectoryNode{
		DeviceID:   "device-x",
		DeviceName: "Switch-X",
		Interface:  "Gi0/1",
		VLANID:     &vlan10,
		MACAddress: "AABBCCDDEEFF",
		EventType:  "appeared",
		StartTime:  time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndTime:    time.Date(2025, 6, 1, 1, 0, 0, 0, time.UTC),
		Duration:   3600,
	}

	data, err := json.Marshal(node)
	require.NoError(t, err, "TrajectoryNode 应能成功 JSON 序列化")

	// 断言关键字段 wire-format 名称
	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))

	assert.Equal(t, "device-x", raw["deviceId"], "DeviceID JSON tag 应为 deviceId")
	assert.Equal(t, "Switch-X", raw["deviceName"], "DeviceName JSON tag 应为 deviceName")
	assert.Equal(t, "Gi0/1", raw["interface"], "Interface JSON tag 应为 interface")
	assert.Equal(t, "AABBCCDDEEFF", raw["mac"], "MACAddress JSON tag 应为 mac(供前端 tooltip 读取)")
	assert.Equal(t, "appeared", raw["eventType"], "EventType JSON tag 应为 eventType")
	assert.Equal(t, float64(3600), raw["duration"], "Duration JSON tag 应为 duration")

	// 关键合约断言:不存在 snake_case 残留(回归保护,防止有人误改回 snake_case)
	for _, snake := range []string{"device_id", "device_name", "mac_address", "event_type", "start_time", "end_time"} {
		_, exists := raw[snake]
		assert.False(t, exists, "JSON 输出不应包含 snake_case 字段 %q", snake)
	}
}

// TestAggregateTrajectory_MACAddressEdgeCases 验证 MACAddress 字段在边界场景
// 下(空输入 / 单事件 / 全不同位置)的行为,确保 aggregateTrajectory 的 MAC 传播
// invariant 在所有形状的输入下都成立。
func TestAggregateTrajectory_MACAddressEdgeCases(t *testing.T) {
	service := &macHistoryQueryServiceImpl{}
	baseTime := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	const expectedMAC = "AABBCCDDEEFF"

	t.Run("empty input returns empty nodes (no panic, no nil deref)", func(t *testing.T) {
		nodes := service.aggregateTrajectory([]rawEvent{})
		assert.Equal(t, 0, len(nodes), "空输入应返回空切片")
		assert.NotNil(t, nodes, "空输入应返回非 nil 切片(避免前端空检查陷阱)")
	})

	t.Run("single event creates one node with MACAddress set", func(t *testing.T) {
		vlan := 100
		nodes := service.aggregateTrajectory([]rawEvent{{
			ID:                 "s1",
			DeviceID:           "dev-1",
			DeviceNameSnapshot: "S1",
			MACAddress:         expectedMAC,
			InterfaceName:      "Gi0/1",
			VLANID:             &vlan,
			EventType:          "appeared",
			FirstSeen:          baseTime,
			LastSeen:           baseTime.Add(1 * time.Hour),
		}})
		require.Equal(t, 1, len(nodes))
		assert.Equal(t, expectedMAC, nodes[0].MACAddress, "单事件节点 MACAddress 必须填充")
	})

	t.Run("all distinct locations keep per-node MAC independent", func(t *testing.T) {
		// 3 个不同位置,产生 3 个节点(不被合并)
		vlan1, vlan2, vlan3 := 1, 2, 3
		nodes := service.aggregateTrajectory([]rawEvent{
			{DeviceID: "d1", DeviceNameSnapshot: "S1", MACAddress: expectedMAC, InterfaceName: "Gi0/1", VLANID: &vlan1, EventType: "appeared", FirstSeen: baseTime, LastSeen: baseTime.Add(1 * time.Hour)},
			{DeviceID: "d2", DeviceNameSnapshot: "S2", MACAddress: expectedMAC, InterfaceName: "Gi0/2", VLANID: &vlan2, EventType: "moved", FirstSeen: baseTime.Add(1 * time.Hour), LastSeen: baseTime.Add(2 * time.Hour)},
			{DeviceID: "d3", DeviceNameSnapshot: "S3", MACAddress: expectedMAC, InterfaceName: "Gi0/3", VLANID: &vlan3, EventType: "disappeared", FirstSeen: baseTime.Add(2 * time.Hour), LastSeen: baseTime.Add(3 * time.Hour)},
		})
		require.Equal(t, 3, len(nodes), "3 个不同位置应产生 3 个节点")
		// 3 个节点必须有 MACAddress 且等于 expectedMAC
		for i, n := range nodes {
			assert.Equal(t, expectedMAC, n.MACAddress,
				"全不同位置场景下,节点 #%d 的 MACAddress 仍应等于 rawEvent MAC", i)
		}
	})
}

// TestSameLocation 测试位置比较逻辑
func TestSameLocation(t *testing.T) {
	service := &macHistoryQueryServiceImpl{}

	baseTime := time.Now()
	vlan100 := 100
	vlan200 := 200

	node := TrajectoryNode{
		DeviceID:  "device1",
		Interface: "Gi0/1",
		VLANID:    &vlan100,
		StartTime: baseTime,
		EndTime:   baseTime.Add(1 * time.Hour),
	}

	tests := []struct {
		name     string
		node     TrajectoryNode
		event    rawEvent
		expected bool
	}{
		{
			name: "完全相同",
			event: rawEvent{
				DeviceID:      "device1",
				InterfaceName: "Gi0/1",
				VLANID:        &vlan100,
			},
			expected: true,
		},
		{
			name: "设备不同",
			event: rawEvent{
				DeviceID:      "device2",
				InterfaceName: "Gi0/1",
				VLANID:        &vlan100,
			},
			expected: false,
		},
		{
			name: "接口不同",
			event: rawEvent{
				DeviceID:      "device1",
				InterfaceName: "Gi0/2",
				VLANID:        &vlan100,
			},
			expected: false,
		},
		{
			name: "VLAN不同",
			event: rawEvent{
				DeviceID:      "device1",
				InterfaceName: "Gi0/1",
				VLANID:        &vlan200,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.sameLocation(node, tt.event)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// setupTestService 创建测试用的service实例和内存数据库
func setupTestService(t *testing.T) *macHistoryQueryServiceImpl {
	// 创建内存SQLite数据库
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	// 创建测试表
	err = db.AutoMigrate(&models.MACOUIVendor{})
	require.NoError(t, err)

	// 创建service实例（cache为nil，不测试缓存逻辑）
	service := &macHistoryQueryServiceImpl{db: db, cache: nil}
	return service
}

// TestGetVendor 测试GetVendor方法
func TestGetVendor(t *testing.T) {
	service := setupTestService(t)

	// 准备测试数据
	oui := &models.MACOUIVendor{
		OUIPrefix:  "AABBCC",
		VendorName: "Test Vendor Inc",
	}
	require.NoError(t, service.db.Create(oui).Error)

	tests := []struct {
		name       string
		mac        string
		wantVendor string
		wantErr    bool
	}{
		{
			name:       "已知OUI - 大写带分隔符",
			mac:        "AA:BB:CC:DD:EE:FF",
			wantVendor: "Test Vendor Inc",
			wantErr:    false,
		},
		{
			name:       "已知OUI - 小写无分隔符",
			mac:        "aabbccddeeff",
			wantVendor: "Test Vendor Inc",
			wantErr:    false,
		},
		{
			name:       "已知OUI - 点分格式",
			mac:        "aabb.ccddeeff",
			wantVendor: "Test Vendor Inc",
			wantErr:    false,
		},
		{
			name:       "未知OUI",
			mac:        "DD:EE:FF:11:22:33",
			wantVendor: "Unknown Vendor",
			wantErr:    false,
		},
		{
			name:       "无效MAC格式 - 太短",
			mac:        "AA:BB",
			wantVendor: "Unknown Vendor",
			wantErr:    false,
		},
		{
			name:       "无效MAC格式 - 非十六进制",
			mac:        "GG:HH:II:JJ:KK:LL",
			wantVendor: "Unknown Vendor",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vendor, err := service.GetVendor(context.Background(), tt.mac)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantVendor, vendor)
			}
		})
	}
}

