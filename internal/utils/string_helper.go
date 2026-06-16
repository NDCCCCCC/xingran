package utils

import (
	"strings"
	"unicode/utf8"
)

// sanitizeControlChars 控制字符替换表。
// PostgreSQL TEXT/VARCHAR 列禁止存储 NUL 字节 (0x00),其他控制字符
// (0x01-0x08, 0x0B, 0x0C, 0x0E-0x1F, 0x7F) 也应在日志/审计字段中被替换,
// 否则可能绕过日志分析、终端渲染或下游 JSON 序列化。
var sanitizeControlChars = strings.NewReplacer(
	"\x00", "", // NUL — PostgreSQL 拒绝
	"\x01", " ",
	"\x02", " ",
	"\x03", " ",
	"\x04", " ",
	"\x05", " ",
	"\x06", " ",
	"\x07", " ",
	"\x08", " ",
	"\x0B", " ",
	"\x0C", " ",
	"\x0E", " ",
	"\x0F", " ",
	"\x10", " ",
	"\x11", " ",
	"\x12", " ",
	"\x13", " ",
	"\x14", " ",
	"\x15", " ",
	"\x16", " ",
	"\x17", " ",
	"\x18", " ",
	"\x19", " ",
	"\x1A", " ",
	"\x1B", " ",
	"\x1C", " ",
	"\x1D", " ",
	"\x1E", " ",
	"\x1F", " ",
	"\x7F", " ", // DEL
)

// SanitizeForDB 把字符串清洗为可安全写入 PostgreSQL TEXT 列的形式。
//
// 行为:
//   - 移除 NUL 字节 (0x00) — 这是 AD/LDAP 错误消息反复触发
//     "invalid byte sequence for encoding UTF8: 0x00 (SQLSTATE 22021)"
//     的根源。go-ldap v3.4.12 在 error.go:223 直接把 AD 服务器原始字节
//     包成 error message,不剔除 0x00。
//   - 替换其他 ASCII/Latin-1 控制字符为单空格,保留可读性。
//   - 用 utf8.RuneError 替换无法解码为合法 UTF-8 的字节序列 — PostgreSQL
//     UTF8 编码同样会拒绝。
//   - 把超长字符串截断到 maxLen,防止攻击者构造巨大错误消息耗尽列宽或日志空间。
//
// 用途:任何写入数据库/日志/审计字段的外部字符串(尤其是 LDAP/AD/REST 响应
// 衍生字段)都应先调用本函数。
//
// 参考触发场景:
//
//	ERRO[2026-06-16 17:37:29] [GORM错误] UPDATE "sys_ad_sync_log" SET
//	"error_message"='绑定失败: LDAP Result Code 49 ...' |
//	错误: ERROR: invalid byte sequence for encoding "UTF8": 0x00 (SQLSTATE 22021)
func SanitizeForDB(s string) string {
	if s == "" {
		return s
	}
	// 1. 控制字符 (含 0x00) → 替换
	s = sanitizeControlChars.Replace(s)
	// 2. 兜底:用 utf8.ValidString 检测残余非法 UTF-8 序列,替换为 '?'
	if !utf8.ValidString(s) {
		var b strings.Builder
		b.Grow(len(s))
		for i := 0; i < len(s); {
			r, size := utf8.DecodeRuneInString(s[i:])
			if r == utf8.RuneError && size <= 1 {
				b.WriteByte('?')
				i++
				continue
			}
			b.WriteRune(r)
			i += size
		}
		s = b.String()
	}
	return s
}

// TruncateForLog 把字符串截断到 maxLen 字符数,超出部分用省略号收尾。
// 与 SanitizeForDB 配合使用可同时控制控制字符与长度,适合日志字段。
func TruncateForLog(s string, maxLen int) string {
	if maxLen <= 0 || s == "" {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-1]) + "…"
}

// SanitizeAndTruncate 是 SanitizeForDB + TruncateForLog 的组合,
// 推荐用于写入数据库 + 日志双重场景的字符串。
func SanitizeAndTruncate(s string, maxLen int) string {
	return TruncateForLog(SanitizeForDB(s), maxLen)
}

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
