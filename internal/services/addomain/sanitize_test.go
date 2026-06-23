package addomain

import (
	"strings"
	"testing"
)

// TestSanitizeFailureReason 锁定 MarkFailure 写入 PostgreSQL text 列前的清洗行为。
// 背景：AD/LDAP 错误消息可能含 null 字节（0x00）或无效 UTF-8，PostgreSQL text 列
// 拒绝 0x00（SQLSTATE 22021），未清洗会使 failure_count 不累加、坏账号永不熔断。
func TestSanitizeFailureReason(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "正常字符串原样通过",
			in:   "dial:绑定失败: LDAP Result Code 49 data 52e",
			want: "dial:绑定失败: LDAP Result Code 49 data 52e",
		},
		{
			name: "移除 null 字节 0x00（PG text 列拒绝）",
			in:   "dial:绑定失败\x00data 52e",
			want: "dial:绑定失败data 52e",
		},
		{
			name: "移除多个 null 字节",
			in:   "a\x00b\x00c",
			want: "abc",
		},
		{
			name: "移除无效 UTF-8 序列",
			in:   "err\xff\xfe\x00end",
			want: "errend", // \xff\xfe 非法 UTF-8 被移除，\x00 也被移除：err+end
		},
		{
			name: "空字符串",
			in:   "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeFailureReason(tt.in)
			if got != tt.want {
				t.Errorf("sanitizeFailureReason(%q) = %q, want %q", tt.in, got, tt.want)
			}
			// 硬性断言：结果绝不含 0x00
			if strings.ContainsRune(got, 0) {
				t.Errorf("结果含 null 字节: %q", got)
			}
		})
	}

	// 截断测试：超长输入截断到 1000
	t.Run("超长输入截断到1000", func(t *testing.T) {
		long := strings.Repeat("x", 2500)
		got := sanitizeFailureReason(long)
		if len(got) != 1000 {
			t.Errorf("超长输入截断后长度 = %d, want 1000", len(got))
		}
	})

	// 回归铁证：模拟日志中的真实 AD 错误（含可能嵌入的 null 字节）必须可安全写库
	t.Run("真实AD错误含null字节可安全写库", func(t *testing.T) {
		realErr := "dial:绑定失败: LDAP Result Code 49 \"Invalid Credentials\": 80090308: LdapErr: DSID-0C090451, comment: AcceptSecurityContext error, data 52e, v3839\x00(尝试: UPN, NetBIOS, 直连)"
		got := sanitizeFailureReason(realErr)
		if strings.ContainsRune(got, 0) {
			t.Errorf("清洗后仍含 null 字节，写库会 SQLSTATE 22021: %q", got)
		}
	})
}
