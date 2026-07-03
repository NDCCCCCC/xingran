package normalize

import (
	"regexp"
	"strings"
)

// macHexPattern 校验剥分隔符后的 12 位大写十六进制串。
var macHexPattern = regexp.MustCompile(`^[0-9A-F]{12}$`)

// MACAddress 把任意厂商 MAC 格式归一化为"大写 + 冒号"(AA:BB:CC:DD:EE:FF)。
//
// 支持输入:
//   - aabb.ccdd.eeff (Cisco/Huawei 点分隔) → AA:BB:CC:DD:EE:FF
//   - aa-bb-cc-dd-ee-ff (连字符)            → AA:BB:CC:DD:EE:FF
//   - aa:bb:cc:dd:ee:ff (冒号)              → AA:BB:CC:DD:EE:FF
//   - AABBCCDDEEFF     (无分隔符)           → AA:BB:CC:DD:EE:FF
//
// 三步: 剥所有分隔符(. : -) → UPPER → 12 hex 校验 → 插冒号。
// 非法输入(剥完不是 12 hex,如设备输出垃圾 'Flags:'/'Total')原样返回,
// 由调用方或 M184/M189 迁移兜底清理。本包不依赖 logger 以保持纯净。
func MACAddress(mac string) string {
	if mac == "" {
		return ""
	}
	mac = strings.TrimSpace(mac)
	if mac == "" {
		return ""
	}

	normalized := strings.ToUpper(strings.NewReplacer(".", "", ":", "", "-", "").Replace(mac))
	if !macHexPattern.MatchString(normalized) {
		return mac
	}

	var b strings.Builder
	b.Grow(17)
	for i := 0; i < 12; i += 2 {
		if i > 0 {
			b.WriteByte(':')
		}
		b.WriteString(normalized[i : i+2])
	}
	return b.String()
}
