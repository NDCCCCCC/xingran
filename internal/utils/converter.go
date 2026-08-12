package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// ToInt 从 interface{} 转换为 int
func ToInt(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case string:
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return 0
}

// ToInt64 从 interface{} 转换为 int64
func ToInt64(v interface{}) int64 {
	switch val := v.(type) {
	case float64:
		return int64(val)
	case int:
		return int64(val)
	case int64:
		return val
	case string:
		if i, err := strconv.ParseInt(val, 10, 64); err == nil {
			return i
		}
	}
	return 0
}

// ToBoolPtr 从 interface{} 转换为 *bool
func ToBoolPtr(v interface{}) *bool {
	switch val := v.(type) {
	case bool:
		return &val
	case string:
		if b, err := strconv.ParseBool(val); err == nil {
			return &b
		}
	}
	return nil
}

// ToIntPtr 从 interface{} 转换为 *int
func ToIntPtr(v interface{}) *int {
	i := ToInt(v)
	if i != 0 {
		return &i
	}
	return nil
}

// ToStringPtr 从 interface{} 转换为 *string
func ToStringPtr(v interface{}) *string {
	if v == nil {
		return nil
	}
	s := fmt.Sprintf("%v", v)
	if s != "" {
		return &s
	}
	return nil
}

// ToStringSlice 从 interface{} 转换为 []string
func ToStringSlice(v interface{}) []string {
	var result []string
	switch val := v.(type) {
	case []interface{}:
		for _, item := range val {
			result = append(result, fmt.Sprintf("%v", item))
		}
	case []string:
		return val
	case string:
		if val != "" {
			parts := strings.Split(val, ",")
			for _, part := range parts {
				if trimmed := strings.TrimSpace(part); trimmed != "" {
					result = append(result, trimmed)
				}
			}
		}
	}
	return result
}

// ParseStatusToInt 从 interface{} 解析状态值
// 支持: float64, int, string, bool (true=0启用, false=1禁用)
func ParseStatusToInt(v interface{}) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case string:
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	case bool:
		if val {
			return 0 // 启用
		}
		return 1 // 禁用
	}
	return 0
}

// DerefString 安全地解引用字符串指针
func DerefString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

// DerefInt 安全地解引用整数指针
func DerefInt(i *int) int {
	if i != nil {
		return *i
	}
	return 0
}

// DerefBool 安全地解引用布尔指针
func DerefBool(b *bool) bool {
	if b != nil {
		return *b
	}
	return false
}

// ToSlicePtr 将值切片转换为指针切片
func ToSlicePtr[T any](slice []T) []*T {
	result := make([]*T, len(slice))
	for i := range slice {
		result[i] = &slice[i]
	}
	return result
}

// ToSlice 将指针切片转换为值切片
func ToSlice[T any](ptrs []*T) []T {
	result := make([]T, 0, len(ptrs))
	for _, ptr := range ptrs {
		if ptr != nil {
			result = append(result, *ptr)
		}
	}
	return result
}
