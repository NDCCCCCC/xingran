package utils

import (
	"strings"
)

// Contains 检查字符串是否包含子串（不区分大小写）
func Contains(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}

// IsEmpty 检查字符串是否为空
func IsEmpty(s *string) bool {
	return s == nil || *s == ""
}

// IsNotEmpty 检查字符串是否非空
func IsNotEmpty(s *string) bool {
	return !IsEmpty(s)
}

// TrimSpace 去除字符串空格
func TrimSpace(s *string) *string {
	if s == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// JoinStringSlice 连接字符串切片
func JoinStringSlice(items []string, separator string) string {
	if len(items) == 0 {
		return ""
	}
	return strings.Join(items, separator)
}

// SplitString 分割字符串
func SplitString(s, separator string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, separator)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
