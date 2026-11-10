package device

// B-3 / v1.56：snmpValueToString 与 parseSNMPValue 的统一转换分支覆盖
// （GOV-03 diff coverage 补测）。全部为纯函数 / 纯 PDU，零网络依赖。

import (
	"testing"

	"github.com/gosnmp/gosnmp"
	"github.com/stretchr/testify/assert"
)

func TestB3_SnmpValueToString(t *testing.T) {
	cases := []struct {
		name string
		in   interface{}
		want string
	}{
		{"普通字符串", "core-sw-1", "core-sw-1"},
		{"带空白", "  core-sw-1  ", "core-sw-1"},
		{"OID 前导点去掉", ".1.3.6.1.4.1.2011.2.23", "1.3.6.1.4.1.2011.2.23"},
		{"OID 前导点+空白", "  .1.3.6.1  ", "1.3.6.1"},
		{"字节切片", []byte("1.3.6.1.2.1.1.2.0"), "1.3.6.1.2.1.1.2.0"},
		{"字节切片带前导点空白", []byte("  .1.3.6  "), "1.3.6"},
		{"整数走 fmt.Sprint", 42, "42"},
		{"nil 走 fmt.Sprint", nil, "<nil>"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, snmpValueToString(tc.in))
		})
	}
}

func TestB3_ParseSNMPValue_StringAndOID(t *testing.T) {
	// OctetString：string 与 []byte 两种形态都收敛成字符串（不再硬断言 panic）
	s := parseSNMPValue(gosnmp.SnmpPDU{Type: gosnmp.OctetString, Value: "Huawei S8700"})
	assert.Equal(t, "Huawei S8700", s)
	b := parseSNMPValue(gosnmp.SnmpPDU{Type: gosnmp.OctetString, Value: []byte("  .1.3.6  ")})
	assert.Equal(t, "1.3.6", b)

	// ObjectIdentifier：string / []byte / 其它类型
	oidStr := parseSNMPValue(gosnmp.SnmpPDU{Type: gosnmp.ObjectIdentifier, Value: ".1.3.6.1.4.1.2011.2.23"})
	assert.Equal(t, "1.3.6.1.4.1.2011.2.23", oidStr)
	oidBytes := parseSNMPValue(gosnmp.SnmpPDU{Type: gosnmp.ObjectIdentifier, Value: []byte("1.3.6.1.4.1.2011")})
	assert.Equal(t, "1.3.6.1.4.1.2011", oidBytes)
	oidOther := parseSNMPValue(gosnmp.SnmpPDU{Type: gosnmp.ObjectIdentifier, Value: 99})
	assert.Equal(t, "99", oidOther)

	// 其它类型（Integer）行为不变
	n := parseSNMPValue(gosnmp.SnmpPDU{Type: gosnmp.Integer, Value: 7})
	assert.Equal(t, int64(7), n)
}

