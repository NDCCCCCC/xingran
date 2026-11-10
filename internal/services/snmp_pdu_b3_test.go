package services

// B-3 / v1.56：snmpPDUString 统一转换分支覆盖（GOV-03 diff coverage 补测）。
// 纯函数，零网络依赖。

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestB3_SnmpPDUString(t *testing.T) {
	cases := []struct {
		name string
		in   interface{}
		want string
	}{
		{"普通字符串", "1.3.6.1.4.1.2011.2.23", "1.3.6.1.4.1.2011.2.23"},
		{"带空白", "  1.3.6  ", "1.3.6"},
		{"前导点去掉", ".1.3.6.1.2.1.1.2.0", "1.3.6.1.2.1.1.2.0"},
		{"前导点+空白", "  .1.3.6  ", "1.3.6"},
		{"字节切片", []byte("1.3.6.1.2.1.1.2.0"), "1.3.6.1.2.1.1.2.0"},
		{"字节切片带前导点空白", []byte("  .1.3.6  "), "1.3.6"},
		{"整数走 fmt.Sprint", 42, "42"},
		{"nil 走 fmt.Sprint", nil, "<nil>"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, snmpPDUString(tc.in))
		})
	}
}
