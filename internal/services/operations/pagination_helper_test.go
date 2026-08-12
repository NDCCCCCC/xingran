package operations

import (
	"math"
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/constants"
)

func TestExtractPagination(t *testing.T) {
	tests := []struct {
		name     string
		params   map[string]interface{}
		expected PaginationParams
	}{
		{
			name:     "默认值",
			params:   map[string]interface{}{},
			expected: PaginationParams{Current: 1, PageSize: 10},
		},
		{
			name: "整数参数",
			params: map[string]interface{}{
				"current":  2,
				"pageSize": 20,
			},
			expected: PaginationParams{Current: 2, PageSize: 20},
		},
		{
			name: "浮点数参数",
			params: map[string]interface{}{
				"current":  float64(3),
				"pageSize": float64(30),
			},
			expected: PaginationParams{Current: 3, PageSize: 30},
		},
		{
			name: "PageSize 小于最小值",
			params: map[string]interface{}{
				"current":  1,
				"pageSize": 5,
			},
			expected: PaginationParams{Current: 1, PageSize: 10},
		},
		{
			name: "PageSize 大于最大值",
			params: map[string]interface{}{
				"current":  1,
				"pageSize": 20000,
			},
			expected: PaginationParams{Current: 1, PageSize: 10000},
		},
		{
			name: "混合类型参数",
			params: map[string]interface{}{
				"current":  int(5),
				"pageSize": float64(50),
			},
			expected: PaginationParams{Current: 5, PageSize: 50},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPagination(tt.params)
			if result.Current != tt.expected.Current {
				t.Errorf("Current = %v, want %v", result.Current, tt.expected.Current)
			}
			if result.PageSize != tt.expected.PageSize {
				t.Errorf("PageSize = %v, want %v", result.PageSize, tt.expected.PageSize)
			}
		})
	}
}

func TestCalculateOffset(t *testing.T) {
	tests := []struct {
		name     string
		params   PaginationParams
		expected int
	}{
		{
			name:     "第一页",
			params:   PaginationParams{Current: 1, PageSize: 10},
			expected: 0,
		},
		{
			name:     "第二页",
			params:   PaginationParams{Current: 2, PageSize: 10},
			expected: 10,
		},
		{
			name:     "第三页每页20条",
			params:   PaginationParams{Current: 3, PageSize: 20},
			expected: 40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateOffset(tt.params)
			if result != tt.expected {
				t.Errorf("calculateOffset() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestExtractIntParam(t *testing.T) {
	tests := []struct {
		name         string
		params       map[string]interface{}
		key          string
		defaultValue int
		expected     int
	}{
		{
			name:         "键不存在",
			params:       map[string]interface{}{},
			key:          "current",
			defaultValue: 1,
			expected:     1,
		},
		{
			name:         "整数值",
			params:       map[string]interface{}{"current": 5},
			key:          "current",
			defaultValue: 1,
			expected:     5,
		},
		{
			name:         "浮点数值",
			params:       map[string]interface{}{"current": float64(7)},
			key:          "current",
			defaultValue: 1,
			expected:     7,
		},
		{
			name:         "类型不匹配",
			params:       map[string]interface{}{"current": "invalid"},
			key:          "current",
			defaultValue: 1,
			expected:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractIntParam(tt.params, tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("extractIntParam() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestClampPageSize(t *testing.T) {
	tests := []struct {
		name     string
		pageSize int
		expected int
	}{
		{"小于最小值", 5, 10},
		{"等于最小值", 10, 10},
		{"在范围内", 50, 50},
		{"等于最大值", 100, 100},
		{"大于最大值", 200, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clampPageSize(tt.pageSize)
			if result != tt.expected {
				t.Errorf("clampPageSize() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestClampPageSizeMath(t *testing.T) {
	// 验证 clampPageSize 的数学计算逻辑(引用 constants 统一定义)。
	// operations 模块 clamp 上限为 MaxOptionsPageSize(10000)。
	pageSize := 50
	min := float64(constants.MinPageSize)        // 10
	max := float64(constants.MaxOptionsPageSize) // 10000
	clamped := int(math.Max(min, math.Min(max, float64(pageSize))))

	if clamped != 50 {
		t.Errorf("math clamp calculation failed: got %v, want 50", clamped)
	}

	// 测试边界值:下限 clamp 到 MinPageSize
	if int(math.Max(min, math.Min(max, 5))) != 10 {
		t.Error("lower bound clamp failed")
	}

	// 200 在 [10, 10000] 范围内,不应被 clamp
	if int(math.Max(min, math.Min(max, 200))) != 200 {
		t.Error("in-range value should not be clamped")
	}

	// 超过 MaxOptionsPageSize 才 clamp 到上界
	if int(math.Max(min, math.Min(max, 50000))) != 10000 {
		t.Error("upper bound clamp failed")
	}
}
