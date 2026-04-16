package services

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCacheKeyBuilder_Build 测试缓存键构建
func TestCacheKeyBuilder_Build(t *testing.T) {
	tests := []struct {
		name          string
		prefix        string
		params        []interface{}
		expected      string
		shouldContain string
	}{
		{
			name:     "无参数",
			prefix:   "test",
			params:   []interface{}{},
			expected: "test",
		},
		{
			name:     "单个字符串参数",
			prefix:   "user",
			params:   []interface{}{"123"},
			expected: "user:123",
		},
		{
			name:     "多个字符串参数",
			prefix:   "cache",
			params:   []interface{}{"module", "key1", "key2"},
			expected: "cache:module:key1:key2",
		},
		{
			name:     "整数参数",
			prefix:   "user",
			params:   []interface{}{"id", 12345},
			expected: "user:id:12345",
		},
		{
			name:     "混合类型参数",
			prefix:   "data",
			params:   []interface{}{"type", 100, "subkey"},
			expected: "data:type:100:subkey",
		},
		{
			name:     "负数参数",
			prefix:   "test",
			params:   []interface{}{"value", -100},
			expected: "test:value:-100",
		},
		{
			name:          "包含空字符串",
			prefix:        "test",
			params:        []interface{}{"", "value"},
			shouldContain: "test::value",
		},
		{
			name:     "浮点数参数",
			prefix:   "metric",
			params:   []interface{}{"cpu", 95.5},
			expected: "metric:cpu:95.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewCacheKeyBuilder(tt.prefix)
			result := builder.Build(tt.params...)

			if tt.expected != "" {
				assert.Equal(t, tt.expected, result)
			}
			if tt.shouldContain != "" {
				assert.Contains(t, result, tt.shouldContain)
			}
		})
	}
}

// TestCacheKeyBuilder_Build_Performance 性能对比测试
func TestCacheKeyBuilder_Build_Performance(t *testing.T) {
	params := []interface{}{"module", "key1", "key2", 12345, true}

	// 测试新方法
	t.Run("OptimizedMethod", func(t *testing.T) {
		builder := NewCacheKeyBuilder("test")
		for i := 0; i < 1000; i++ {
			_ = builder.Build(params...)
		}
	})

	// 测试旧方法（字符串拼接）
	t.Run("OldStringConcat", func(t *testing.T) {
		for i := 0; i < 1000; i++ {
			key := "test"
			for _, p := range params {
				key += ":" + toString(p)
			}
			_ = key
		}
	})
}

// BenchmarkCacheKeyBuilder_Build 基准测试
func BenchmarkCacheKeyBuilder_Build(b *testing.B) {
	builder := NewCacheKeyBuilder("cache")
	params := []interface{}{"module", "key1", "key2", 12345, "subkey"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = builder.Build(params...)
	}
}

// BenchmarkCacheKeyBuilder_Build_SingleParam 单参数基准测试
func BenchmarkCacheKeyBuilder_Build_SingleParam(b *testing.B) {
	builder := NewCacheKeyBuilder("user")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = builder.Build("12345")
	}
}

// BenchmarkCacheKeyBuilder_Build_ManyParams 多参数基准测试
func BenchmarkCacheKeyBuilder_Build_ManyParams(b *testing.B) {
	builder := NewCacheKeyBuilder("complex")
	params := []interface{}{"a", "b", "c", "d", "e", 1, 2, 3, 4, 5}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = builder.Build(params...)
	}
}

// BenchmarkOldStringConcat 旧方法基准测试（对比）
func BenchmarkOldStringConcat(b *testing.B) {
	params := []interface{}{"module", "key1", "key2", 12345, "subkey"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := "cache"
		for _, p := range params {
			key += ":" + toString(p)
		}
		_ = key
	}
}

// BenchmarkStringsBuilder 纯 strings.Builder 基准测试
func BenchmarkStringsBuilder(b *testing.B) {
	params := []interface{}{"module", "key1", "key2", 12345, "subkey"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var sb strings.Builder
		sb.Grow(len("cache") + len(params)*10)
		sb.WriteString("cache")
		for _, p := range params {
			sb.WriteByte(':')
			sb.WriteString(toString(p))
		}
		_ = sb.String()
	}
}

// toString 辅助函数 - 模拟旧的类型转换
func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int:
		return formatInt(val)
	case int64:
		return formatInt(int(val))
	default:
		return ""
	}
}

func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	if neg {
		digits = append(digits, '-')
	}
	// 反转
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}
