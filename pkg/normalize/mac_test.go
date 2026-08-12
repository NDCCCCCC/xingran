package normalize

import "testing"

// TestMACAddress 锁住 MAC 归一化为"大写 + 冒号"的契约。
func TestMACAddress(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"Cisco 点分隔", "aabb.ccdd.eeff", "AA:BB:CC:DD:EE:FF"},
		{"连字符小写", "aa-bb-cc-dd-ee-ff", "AA:BB:CC:DD:EE:FF"},
		{"冒号小写", "aa:bb:cc:dd:ee:ff", "AA:BB:CC:DD:EE:FF"},
		{"已规范大写", "AA:BB:CC:DD:EE:FF", "AA:BB:CC:DD:EE:FF"},
		{"无分隔符大写", "AABBCCDDEEFF", "AA:BB:CC:DD:EE:FF"},
		{"无分隔符小写", "aabbccddeeff", "AA:BB:CC:DD:EE:FF"},
		{"华为 4-4-4 连字符", "00e0-fc12-3456", "00:E0:FC:12:34:56"},
		{"前导空格", "  aa:bb:cc:dd:ee:ff", "AA:BB:CC:DD:EE:FF"},
		{"空串", "", ""},
		{"仅空格", "   ", ""},
		// 非法输入(设备输出垃圾)原样返回,由迁移兜底清理
		{"非法 Flags: 原样返回", "Flags:", "Flags:"},
		{"非法 Total 原样返回", "Total", "Total"},
		{"11 位 hex 原样返回", "AABBCCDDEEF", "AABBCCDDEEF"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MACAddress(tt.input); got != tt.want {
				t.Errorf("MACAddress(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
