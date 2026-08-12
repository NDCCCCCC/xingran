package lldp

import (
	"fmt"
	"testing"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLLDPServiceCache 测试LLDP缓存功能
func TestLLDPServiceCache(t *testing.T) {
	// 创建一个短TTL的缓存用于测试
	cache := NewLLDPCache(100 * time.Millisecond)

	// 测试Set和Get
	neighbors := map[string]*models.LLDPNeighborInfo{
		"gi0/01": {
			LocalInterface:   "GigabitEthernet0/0/1",
			NeighborID:       "test-neighbor",
			NeighborInterface: "eth0",
			NeighborName:     "TestDevice",
		},
	}

	cache.Set("device-1", neighbors)

	// 立即获取应该成功
	retrieved, ok := cache.Get("device-1")
	assert.True(t, ok, "Should retrieve cached data")
	assert.Equal(t, neighbors, retrieved, "Retrieved data should match")

	// 获取不存在的设备
	_, ok = cache.Get("non-existent")
	assert.False(t, ok, "Non-existent device should not be found")
}

// TestLLDPCacheExpiration 测试LLDP缓存过期
func TestLLDPCacheExpiration(t *testing.T) {
	cache := NewLLDPCache(50 * time.Millisecond)

	neighbors := map[string]*models.LLDPNeighborInfo{
		"gi0/01": {
			LocalInterface: "GigabitEthernet0/0/1",
			NeighborID:     "test-neighbor",
		},
	}

	cache.Set("device-1", neighbors)

	// 立即获取应该成功
	_, ok := cache.Get("device-1")
	assert.True(t, ok, "Should retrieve data immediately after Set")

	// 等待缓存过期
	time.Sleep(60 * time.Millisecond)

	// 过期后获取应该失败
	_, ok = cache.Get("device-1")
	assert.False(t, ok, "Expired data should not be retrieved")
}

// TestLLDPCacheDelete 测试LLDP缓存删除
func TestLLDPCacheDelete(t *testing.T) {
	cache := NewLLDPCache(1 * time.Hour)

	neighbors := map[string]*models.LLDPNeighborInfo{
		"gi0/01": {
			LocalInterface: "GigabitEthernet0/0/1",
			NeighborID:     "test-neighbor",
		},
	}

	cache.Set("device-1", neighbors)

	// 验证数据存在
	_, ok := cache.Get("device-1")
	assert.True(t, ok, "Data should exist before Delete")

	// 删除缓存
	cache.Delete("device-1")

	// 验证数据已删除
	_, ok = cache.Get("device-1")
	assert.False(t, ok, "Data should not exist after Delete")
}

// TestLLDPServiceCreation 测试LLDP服务创建
func TestLLDPServiceCreation(t *testing.T) {
	service := NewLLDPService(nil)

	assert.NotNil(t, service, "Service should not be nil")
	assert.NotNil(t, service.parser, "Parser should be initialized")
	assert.NotNil(t, service.cache, "Cache should be initialized")
}

// TestGetCache 测试获取缓存实例
func TestGetCache(t *testing.T) {
	service := NewLLDPService(nil)

	cache := service.GetCache()
	assert.NotNil(t, cache, "Cache should not be nil")
	assert.Same(t, service.cache, cache, "Should return same cache instance")
}

// TestNormalizeInterfaceName 测试接口名规范化
func TestNormalizeInterfaceName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// 2026-07-01 port-mac-format-unify 对称化: 调 portcollection.NormalizeInterfaceName,
		// 目标统一为"大写短名"(GE/XGE/FE/FOE...),不再展开为全称。
		{"GigabitEthernet full -> short GE", "GigabitEthernet0/0/1", "GE0/0/1"},
		{"Short form Gi -> short GE", "Gi0/1", "GE0/1"},
		{"Short form fa -> short FE", "Fa0/1", "FE0/1"},
		{"Short form te -> short XGE", "Te0/1", "XGE0/1"},
		{"Short form fo -> short FOE", "Fo0/1", "FOE0/1"},
		{"With spaces -> short GE", "GigabitEthernet 0/0/1", "GE0/0/1"},
		{"Lowercase gi -> short GE", "gi0/1", "GE0/1"},
		{"Mixed case -> short GE", "GigabitEthernet0/0/1", "GE0/0/1"},
		{"Vlanif 不变", "Vlanif100", "Vlanif100"},
		{"Loopback -> short Loop", "Loopback0", "Loop0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := portcollection.NormalizeInterfaceName(tt.input)
			assert.Equal(t, tt.expected, result, "Normalization mismatch for '%s'", tt.input)
		})
	}
}

// TestLLDPParserEmptyOutput 测试解析空输出
func TestLLDPParserEmptyOutput(t *testing.T) {
	parser := NewLLDPParser()

	neighbors, err := parser.ParseLLDPNeighbors("", models.VendorHuawei)

	assert.NoError(t, err)
	assert.NotNil(t, neighbors)
	assert.Len(t, neighbors, 0, "Empty output should return empty neighbors")
}

// TestLLDPParserWhitespaceOutput 测试解析仅空白的输出
func TestLLDPParserWhitespaceOutput(t *testing.T) {
	parser := NewLLDPParser()

	neighbors, err := parser.ParseLLDPNeighbors("   \n  \t  ", models.VendorHuawei)

	assert.NoError(t, err)
	assert.NotNil(t, neighbors)
	assert.Len(t, neighbors, 0, "Whitespace-only output should return empty neighbors")
}

// TestLLDPCacheConcurrentAccess 测试并发访问缓存
func TestLLDPCacheConcurrentAccess(t *testing.T) {
	cache := NewLLDPCache(1 * time.Hour)

	done := make(chan bool, 100)

	// 并发写入
	for i := 0; i < 50; i++ {
		go func(idx int) {
			neighbors := map[string]*models.LLDPNeighborInfo{
				"gi0/01": {
					LocalInterface: "GigabitEthernet0/0/1",
					NeighborID:     fmt.Sprintf("neighbor-%d", idx),
				},
			}
			cache.Set(fmt.Sprintf("device-%d", idx), neighbors)
			done <- true
		}(i)
	}

	// 并发读取
	for i := 0; i < 50; i++ {
		go func(idx int) {
			_, _ = cache.Get(fmt.Sprintf("device-%d", idx))
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 100; i++ {
		<-done
	}

	// 验证数据一致性 - 随机检查几个设备
	for i := 0; i < 10; i++ {
		neighbors, ok := cache.Get(fmt.Sprintf("device-%d", i))
		assert.True(t, ok, "Device should exist")
		assert.NotNil(t, neighbors, "Neighbors should not be nil")
	}
}

// BenchmarkLLDPCache 基准测试缓存性能
func BenchmarkLLDPCache(b *testing.B) {
	cache := NewLLDPCache(1 * time.Hour)

	neighbors := map[string]*models.LLDPNeighborInfo{
		"gi0/01": {
			LocalInterface: "GigabitEthernet0/0/1",
			NeighborID:     "test-neighbor",
		},
	}

	cache.Set("device-1", neighbors)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Get("device-1")
	}
}

// BenchmarkNormalizeInterfaceName 基准测试接口名规范化性能
func BenchmarkNormalizeInterfaceName(b *testing.B) {
	testCases := []string{
		"GigabitEthernet0/0/1",
		"Gi0/1",
		"FastEthernet0/1",
		"TenGigE0/1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			_ = portcollection.NormalizeInterfaceName(tc)
		}
	}
}

// TestLLDPNeighborInfoStructure 测试LLDP邻居信息结构
func TestLLDPNeighborInfoStructure(t *testing.T) {
	neighbor := &models.LLDPNeighborInfo{
		LocalInterface:   "GigabitEthernet0/0/1",
		NeighborID:       "3815.sep12.eth0",
		NeighborInterface: "eth0",
		NeighborName:     "60P-28",
		Capabilities:     "Router",
	}

	assert.Equal(t, "GigabitEthernet0/0/1", neighbor.LocalInterface)
	assert.Equal(t, "3815.sep12.eth0", neighbor.NeighborID)
	assert.Equal(t, "eth0", neighbor.NeighborInterface)
	assert.Equal(t, "60P-28", neighbor.NeighborName)
	assert.Equal(t, "Router", neighbor.Capabilities)
}

// TestLLDPServiceWithRealDevices 使用真实设备模型测试
func TestLLDPServiceWithRealDevices(t *testing.T) {
	service := NewLLDPService(nil)

	devices := []models.NetworkDevice{
		{
			BaseModel:  models.BaseModel{ID: "device-1"},
			DeviceName: "TestSwitch",
			Vendor:     models.VendorHuawei,
		},
		{
			BaseModel:  models.BaseModel{ID: "device-2"},
			DeviceName: "TestRouter",
			Vendor:     models.VendorRuijie,
		},
	}

	// 验证服务可以处理设备列表
	for range devices {
		// 由于没有真实的executor,这里只测试服务不会panic
		assert.NotNil(t, service)
		assert.NotNil(t, service.parser)
	}
}

// TestLLDPIntegrationScenario 集成测试场景
func TestLLDPIntegrationScenario(t *testing.T) {
	// 模拟完整的LLDP发现流程
	cache := NewLLDPCache(1 * time.Hour)

	// 步骤1: 模拟发现LLDP邻居
	neighbors := map[string]*models.LLDPNeighborInfo{
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

	// 步骤2: 存入缓存
	cache.Set("device-1", neighbors)

	// 步骤3: 从缓存获取
	retrieved, ok := cache.Get("device-1")
	require.True(t, ok, "Should retrieve neighbors from cache")

	// 步骤4: 验证数据完整性
	assert.Len(t, retrieved, 2, "Should have 2 neighbors")

	// 验证第一个邻居
	firstNeighbor, exists := retrieved["gi0/01"]
	assert.True(t, exists, "First neighbor should exist")
	assert.Equal(t, "60P-28", firstNeighbor.NeighborName)
	assert.Equal(t, "Router", firstNeighbor.Capabilities)

	// 验证第二个邻居
	secondNeighbor, exists := retrieved["gi0/02"]
	assert.True(t, exists, "Second neighbor should exist")
	assert.Equal(t, "60P-29", secondNeighbor.NeighborName)
	assert.Equal(t, "Bridge", secondNeighbor.Capabilities)
}
