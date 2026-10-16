package services

import "testing"

// TestNormalizeMACAddress 锁住 MAC 规范化大写+冒号契约
//
// 2026-07-01 port-mac-format-unify 落地:mac_history_service 删本地实现,
// mac_collector 删本地实现,所有 MAC 写入统一走 services.NormalizeMACAddress
// (mac_normalize.go)。
//
// 契约(参见 memory mac-address-normalize-returns-colon-format.md):
//   - 输入任意格式(Cisco/Huawei/标准/无分隔符)
//   - 输出大写+冒号 AA:BB:CC:DD:EE:FF
//   - 空字符串、格式非法 → 返回原值
func TestNormalizeMACAddress(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Cisco/Huawei 点分格式 aabb.ccdd.eeff",
			input:    "9c7b.ef2f.31b8",
			expected: "9C:7B:EF:2F:31:B8",
		},
		{
			name:     "标准 - 分隔 aa-bb-cc-dd-ee-ff",
			input:    "9c-7b-ef-2f-31-b8",
			expected: "9C:7B:EF:2F:31:B8",
		},
		{
			name:     "已规范冒号格式(全小写)",
			input:    "9c:7b:ef:2f:31:b8",
			expected: "9C:7B:EF:2F:31:B8",
		},
		{
			name:     "已规范冒号格式(全大写)",
			input:    "9C:7B:EF:2F:31:B8",
			expected: "9C:7B:EF:2F:31:B8",
		},
		{
			name:     "无分隔符 AABBCCDDEEFF",
			input:    "9C7BEF2F31B8",
			expected: "9C:7B:EF:2F:31:B8",
		},
		{
			name:     "无分隔符小写",
			input:    "9c7bef2f31b8",
			expected: "9C:7B:EF:2F:31:B8",
		},
		{
			name:     "前导空格剥离",
			input:    "  9c7b.ef2f.31b8",
			expected: "9C:7B:EF:2F:31:B8",
		},
		{
			name:     "尾随空格剥离",
			input:    "9c7b.ef2f.31b8  ",
			expected: "9C:7B:EF:2F:31:B8",
		},
		{
			name:     "大小写混写",
			input:    "9C7b.Ef2f.31b8",
			expected: "9C:7B:EF:2F:31:B8",
		},
		{
			name:     "空字符串原样返回",
			input:    "",
			expected: "",
		},
		{
			name:     "全空格原样返回",
			input:    "   ",
			expected: "",
		},
		{
			// 文档契约(mac_normalize.go:26):不符合 12 hex 字符的输入返回 ""
			// (丢弃语义,与 parseRuijiePortSecurityLine 失败路径对齐)
			name:     "格式不合法(非 12 hex)返回空串",
			input:    "not-a-mac",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeMACAddress(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeMACAddress(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestNormalizeMACAddress_EmptyInputs 边界:空输入不 panic
func TestNormalizeMACAddress_EmptyInputs(t *testing.T) {
	cases := []string{"", " ", "\t", "\n", "  \t  "}
	for _, c := range cases {
		got := NormalizeMACAddress(c)
		if got != "" {
			t.Errorf("NormalizeMACAddress(%q) = %q, want empty", c, got)
		}
	}
}
