package utils

import (
	"strings"
	"testing"
)

// TestSanitizeForDB_NULByte 验证 NUL 字节剥离。
//
// 触发场景:go-ldap v3.4.12 在 error.go:223 直接把 AD 服务器的原始字节
// 包装为 error message,某些 Windows AD 诊断消息含 0x00。
// PostgreSQL TEXT 列不允许 NUL,会抛 SQLSTATE 22021。
func TestSanitizeForDB_NULByte(t *testing.T) {
	in := "LDAP Result Code 49\x00\x00 (Invalid Credentials)"
	got := SanitizeForDB(in)
	if strings.ContainsRune(got, 0x00) {
		t.Fatalf("SanitizeForDB should strip NUL bytes, got %q", got)
	}
	if !strings.Contains(got, "LDAP Result Code 49") {
		t.Fatalf("SanitizeForDB should preserve non-NUL content, got %q", got)
	}
}

// TestSanitizeForDB_ControlChars 验证其他控制字符替换为单空格。
func TestSanitizeForDB_ControlChars(t *testing.T) {
	in := "hello\x01\x02\x03world"
	got := SanitizeForDB(in)
	if got != "hello   world" {
		t.Fatalf("expected control chars replaced by spaces, got %q", got)
	}
}

// TestSanitizeForDB_DelChar 验证 DEL (0x7F) 替换。
func TestSanitizeForDB_DelChar(t *testing.T) {
	in := "a\x7Fb"
	got := SanitizeForDB(in)
	if got != "a b" {
		t.Fatalf("expected DEL replaced by space, got %q", got)
	}
}

// TestSanitizeForDB_InvalidUTF8 验证无效 UTF-8 字节序列替换为 '?'。
//
// 触发场景:AD 服务器可能返回 Latin-1 等非 UTF-8 字节序列,PostgreSQL
// UTF8 编码同样拒绝。
func TestSanitizeForDB_InvalidUTF8(t *testing.T) {
	// 0xFF 是无效的 UTF-8 起始字节
	in := "before\xff\xfeafter"
	got := SanitizeForDB(in)
	if strings.Contains(got, "\xff") || strings.Contains(got, "\xfe") {
		t.Fatalf("SanitizeForDB should strip invalid UTF-8 bytes, got %q (bytes=% x)", got, []byte(got))
	}
	if !strings.Contains(got, "before") || !strings.Contains(got, "after") {
		t.Fatalf("SanitizeForDB should preserve valid surrounding content, got %q", got)
	}
}

// TestSanitizeForDB_EmptyAndPlain 验证空串与纯文本不被破坏。
func TestSanitizeForDB_EmptyAndPlain(t *testing.T) {
	if got := SanitizeForDB(""); got != "" {
		t.Fatalf("empty string should pass through, got %q", got)
	}
	if got := SanitizeForDB("hello world"); got != "hello world" {
		t.Fatalf("plain text should pass through, got %q", got)
	}
}

// TestSanitizeForDB_Multibyte 验证 UTF-8 多字节字符不被破坏(中文等)。
func TestSanitizeForDB_Multibyte(t *testing.T) {
	in := "用户同步失败: 80090308 中文"
	got := SanitizeForDB(in)
	if got != in {
		t.Fatalf("Chinese characters should pass through unchanged, got %q", got)
	}
}

// TestSanitizeForDB_RealisticLDAPError 验证真实场景的错误消息能清洗成功。
//
// 直接复制 2026-06-16 17:37:29 真实崩溃时的错误消息体(去掉 NUL 后)。
func TestSanitizeForDB_RealisticLDAPError(t *testing.T) {
	in := "绑定失败: LDAP Result Code 49 \"Invalid Credentials\": 80090308: LdapErr: DSID-0C090451, comment: AcceptSecurityContext error, data 775, v3839 (尝试: UPN, NetBIOS, 直连)"
	got := SanitizeForDB(in)
	if strings.ContainsRune(got, 0x00) {
		t.Fatalf("realistic LDAP error should be NUL-free after sanitize, got %q", got)
	}
	if !strings.Contains(got, "AcceptSecurityContext") {
		t.Fatalf("realistic LDAP error content lost, got %q", got)
	}
}

// TestTruncateForLog 验证日志截断逻辑。
func TestTruncateForLog(t *testing.T) {
	if got := TruncateForLog("hello", 10); got != "hello" {
		t.Fatalf("short string should pass through, got %q", got)
	}
	if got := TruncateForLog("hello world", 5); got != "hell…" {
		t.Fatalf("long string should be truncated with ellipsis, got %q", got)
	}
	if got := TruncateForLog("hello world", 1); got != "h" {
		t.Fatalf("maxLen=1 should keep one rune, got %q", got)
	}
}

// TestSanitizeAndTruncate 验证组合行为。
func TestSanitizeAndTruncate(t *testing.T) {
	in := "a\x00b\x01c\x02d\x03e\x04f" // 6 控制字符
	got := SanitizeAndTruncate(in, 5)
	// 控制字符被替换为空格,然后截断到 5 字符(含省略号)
	// "a b c d e f" 截断到 5 字符应该是 "a b c…"
	if strings.ContainsRune(got, 0x00) {
		t.Fatalf("NUL should be stripped, got %q", got)
	}
	if len([]rune(got)) > 5 {
		t.Fatalf("expected ≤5 runes, got %d in %q", len([]rune(got)), got)
	}
}

// TestSanitizeForDB_RegressionForDBWrite 回归保护:
//
// 此测试模拟真实写入路径:把 sanitize 后的字符串安全地当作 Go string 即可,
// 不需要专门构造 PostgreSQL 错误。语义保证是:任何输出都不含 0x00 字节。
func TestSanitizeForDB_RegressionForDBWrite(t *testing.T) {
	inputs := []string{
		"",
		"normal",
		"中文正常",
		"has\x00NUL",
		"tab\there",
		"newline\nhere",
		"cr\rin",
		"all controls \x00\x01\x02\x03\x04\x05\x06\x07\x08\x0b\x0c\x0e\x0f\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f\x7f end",
		string([]byte{0xC3, 0x28}), // 无效 UTF-8 (0xC3 期望第二个字节是 0x80-0xBF)
		string([]byte{0xFF, 0xFE, 0xFD}),
	}
	for _, in := range inputs {
		got := SanitizeForDB(in)
		for i := 0; i < len(got); i++ {
			if got[i] == 0x00 {
				t.Fatalf("SanitizeForDB(%q) output %q still contains NUL at index %d", in, got, i)
			}
		}
	}
}
