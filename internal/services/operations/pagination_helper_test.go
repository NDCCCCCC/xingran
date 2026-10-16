package operations

import (
	"testing"
)

// TestExtractPagination / TestClampPageSize / TestClampPageSizeMath 已随
// extractPagination / clampPageSize 于 Phase 99-04 (V130R-09 D-03-3) 删除：
// 生产调用方全部迁移到 pkg/query.NormalizePaginationWithMax 单一权威出口，
// 行为回归由 pkg/query/pagination_99_04_test.go 锁定。

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
