package services

import (
	"context"
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

