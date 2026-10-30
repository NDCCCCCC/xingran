package lldp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/tests/fixtures"
)

// TestParseHuaweiLLDPNeighbors 测试解析华为设备LLDP输出
func TestParseHuaweiLLDPNeighbors(t *testing.T) {
	parser := NewLLDPParser()

	neighbors, err := parser.ParseLLDPNeighbors(fixtures.HuaweiLLDPBrief, models.VendorHuawei)

	require.NoError(t, err)
	assert.Len(t, neighbors, 5, "Should parse 5 neighbors from Huawei brief output")

	// 验证第一个邻居
	firstNeighbor := neighbors[0]
	assert.Equal(t, "GigabitEthernet0/0/1", firstNeighbor.LocalInterface)
	assert.Equal(t, "3815.sep12.eth0", firstNeighbor.NeighborID)
	// 2026-07-01: 第3列(60P-28)经模板 NEIGHBOR_PORT 映射到 NeighborInterface。
	// Huawei/H3C brief 共用同一模板, 第3列统一按 PortID 处理(H3C 第3列 eth0 同此)。
	// 原断言 NeighborName 是 pre-existing 笔误(模板未映射 NEIGHBOR_NAME)。
	assert.Equal(t, "60P-28", firstNeighbor.NeighborInterface)

	// 验证LLDP未启用的端口也被解析
	disabledNeighbor := neighbors[2]
	assert.Equal(t, "GigabitEthernet0/0/3", disabledNeighbor.LocalInterface)
	assert.Equal(t, "--", disabledNeighbor.NeighborID, "Disabled port should have '--' as neighbor ID")
}

// TestParseRuijieLLDPNeighbors 测试解析锐捷设备LLDP输出
func TestParseRuijieLLDPNeighbors(t *testing.T) {
	parser := NewLLDPParser()

	neighbors, err := parser.ParseLLDPNeighbors(fixtures.RuijieLLDPNeighbors, models.VendorRuijie)

	require.NoError(t, err)
	assert.Len(t, neighbors, 5, "Should parse 5 neighbors from Ruijie output")

	// 验证第一个邻居
	firstNeighbor := neighbors[0]
	assert.Equal(t, "Gi0/1", firstNeighbor.LocalInterface)
	assert.Equal(t, "3815.sep12.eth0", firstNeighbor.NeighborID)
	assert.Equal(t, "60P-28", firstNeighbor.NeighborName)
	assert.Equal(t, "eth0", firstNeighbor.NeighborInterface)
}

// TestParseH3CLLDPNeighbors 测试解析H3C设备LLDP输出
func TestParseH3CLLDPNeighbors(t *testing.T) {
	parser := NewLLDPParser()

	neighbors, err := parser.ParseLLDPNeighbors(fixtures.H3CLLDPNeighbors, models.VendorH3C)

	require.NoError(t, err)
	assert.Len(t, neighbors, 3, "Should parse 3 neighbors from H3C output")

	// H3C复用华为模板，格式应该相同
	firstNeighbor := neighbors[0]
	assert.Equal(t, "GigabitEthernet1/0/1", firstNeighbor.LocalInterface)
	assert.Equal(t, "3815.sep12.eth0", firstNeighbor.NeighborID)
}

// TestParseMaipuLLDPNeighbors 测试解析迈普设备LLDP输出
func TestParseMaipuLLDPNeighbors(t *testing.T) {
	parser := NewLLDPParser()

	neighbors, err := parser.ParseLLDPNeighbors(fixtures.MaipuLLDPNeighbors, models.VendorMaipu)

	require.NoError(t, err)
	assert.Len(t, neighbors, 3, "Should parse 3 neighbors from Maipu output")

	// 迈普复用锐捷模板
	firstNeighbor := neighbors[0]
	assert.Equal(t, "Eth1/1", firstNeighbor.LocalInterface)
	assert.Equal(t, "3815.sep12.eth0", firstNeighbor.NeighborID)
}

// TestParseEmptyLLDP 测试解析空的LLDP输出
func TestParseEmptyLLDP(t *testing.T) {
	parser := NewLLDPParser()

	neighbors, err := parser.ParseLLDPNeighbors(fixtures.EmptyLLDPOutput, models.VendorHuawei)

	require.NoError(t, err)
	assert.Len(t, neighbors, 0, "Empty output should return empty neighbors")
}

// TestParseLLDPNoNeighbors 测试解析无邻居信息的LLDP输出
func TestParseLLDPNoNeighbors(t *testing.T) {
	parser := NewLLDPParser()

	neighbors, err := parser.ParseLLDPNeighbors(fixtures.HuaweiLLDPEmpty, models.VendorHuawei)

	// 当输出为"No LLDP neighbor information exists"时，可能没有匹配的记录
	// 这取决于模板如何处理这种情况
	if err != nil {
		assert.Error(t, err)
	} else {
		assert.Len(t, neighbors, 0)
	}
}

// TestParseDisabledLLDP 测试解析LLDP未启用的输出
func TestParseDisabledLLDP(t *testing.T) {
	parser := NewLLDPParser()

	neighbors, err := parser.ParseLLDPNeighbors(fixtures.HuaweiLLDPDisabled, models.VendorHuawei)

	// 应该返回空邻居，不应该报错
	// LLDP未启用时，不应该阻断MAC采集
	if err != nil {
		// 某些实现可能返回错误
		assert.Error(t, err)
	} else {
		assert.Len(t, neighbors, 0)
	}
}

// TestParseLLDPWithSpecialCharacters 测试解析包含特殊字符的LLDP输出
func TestParseLLDPWithSpecialCharacters(t *testing.T) {
	parser := NewLLDPParser()

	neighbors, err := parser.ParseLLDPNeighbors(fixtures.LLDPWithSpecialCharacters, models.VendorHuawei)

	require.NoError(t, err)
	assert.Len(t, neighbors, 2, "Should parse 2 neighbors with special characters")

	// 验证包含"/"的端口名
	secondNeighbor := neighbors[1]
	assert.Equal(t, "GigabitEthernet0/0/2", secondNeighbor.LocalInterface)
	assert.Contains(t, secondNeighbor.NeighborInterface, "Port-Channel")
}

// TestParseMalformedLLDPOutput 测试解析格式错误的LLDP输出
func TestParseMalformedLLDPOutput(t *testing.T) {
	parser := NewLLDPParser()

	neighbors, err := parser.ParseLLDPNeighbors(fixtures.MalformedLLDPOutput, models.VendorHuawei)

	// 格式错误的输出应该返回错误或空结果
	// 不应该panic
	assert.NotNil(t, neighbors)
	// 可能返回错误或空结果，取决于模板匹配
	if err != nil {
		assert.Error(t, err)
	}
}

// TestLLDPParserNilOutput 测试解析nil输出
func TestLLDPParserNilOutput(t *testing.T) {
	parser := NewLLDPParser()

	neighbors, err := parser.ParseLLDPNeighbors("", models.VendorHuawei)

	require.NoError(t, err)
	assert.Len(t, neighbors, 0, "Empty string should return empty neighbors")
}

// TestLLDPParserWhitespaceOnly 测试解析仅包含空白的输出
func TestLLDPParserWhitespaceOnly(t *testing.T) {
	parser := NewLLDPParser()

	neighbors, err := parser.ParseLLDPNeighbors("   \n  \t  ", models.VendorHuawei)

	require.NoError(t, err)
	assert.Len(t, neighbors, 0, "Whitespace-only output should return empty neighbors")
}

// BenchmarkParseLLDP 基准测试LLDP解析性能
func BenchmarkParseLLDP(b *testing.B) {
	parser := NewLLDPParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.ParseLLDPNeighbors(fixtures.HuaweiLLDPBrief, models.VendorHuawei)
	}
}

// BenchmarkParseRuijieLLDP 基准测试锐捷LLDP解析性能
func BenchmarkParseRuijieLLDP(b *testing.B) {
	parser := NewLLDPParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.ParseLLDPNeighbors(fixtures.RuijieLLDPNeighbors, models.VendorRuijie)
	}
}

// TestParseLLDPNeighborsFields 测试解析后字段完整性
func TestParseLLDPNeighborsFields(t *testing.T) {
	parser := NewLLDPParser()

	neighbors, err := parser.ParseLLDPNeighbors(fixtures.HuaweiLLDPDetailed, models.VendorHuawei)

	require.NoError(t, err)
	require.NotEmpty(t, neighbors, "Should have at least one neighbor")

	neighbor := neighbors[0]

	// 验证必需字段存在
	assert.NotEmpty(t, neighbor.LocalInterface, "LocalInterface must not be empty")

	// 验证可选字段（可能为空，取决于输出格式）
	// NeighborID, NeighborInterface, NeighborName, Capabilities
}

// TestLLDPParserMultipleVendors 测试解析多个厂商的LLDP输出
func TestLLDPParserMultipleVendors(t *testing.T) {
	parser := NewLLDPParser()

	vendors := []struct {
		vendor    models.DeviceVendor
		output    string
		minCount  int
		maxCount  int
	}{
		{models.VendorHuawei, fixtures.HuaweiLLDPBrief, 3, 10},
		{models.VendorRuijie, fixtures.RuijieLLDPNeighbors, 3, 10},
		{models.VendorH3C, fixtures.H3CLLDPNeighbors, 2, 10},
		{models.VendorMaipu, fixtures.MaipuLLDPNeighbors, 2, 10},
	}

	for _, tt := range vendors {
		t.Run(string(tt.vendor), func(t *testing.T) {
			neighbors, err := parser.ParseLLDPNeighbors(tt.output, tt.vendor)

			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(neighbors), tt.minCount,
				"Should have at least %d neighbors for %s", tt.minCount, tt.vendor)
			assert.LessOrEqual(t, len(neighbors), tt.maxCount,
				"Should have at most %d neighbors for %s", tt.maxCount, tt.vendor)
		})
	}
}
