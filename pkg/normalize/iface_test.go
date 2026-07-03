package normalize

import "testing"

// TestInterfaceName 锁住接口名归一化的对称化契约(2026-07-01 port-mac-format-unify)。
// 目标: 所有输入统一为"大写短名 + 数字"(GE0/0/1 / XGE1/0/1 / FE0/1 ...)。
func TestInterfaceName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// 全称 → 短名
		{"GigabitEthernet2/25", "GigabitEthernet2/25", "GE2/25"},
		{"TenGigE1/0/1", "TenGigE1/0/1", "XGE1/0/1"},
		{"TenGigabitEthernet0/49(华为/Cisco 10G 全称)", "TenGigabitEthernet0/49", "XGE0/49"},
		{"FortyGigE1/1", "FortyGigE1/1", "FOE1/1"},
		{"HundredGigE1/49", "HundredGigE1/49", "HGE1/49"},
		{"TwentyFiveGigE1/1", "TwentyFiveGigE1/1", "TWE1/1"},
		{"FastEthernet0/1", "FastEthernet0/1", "FE0/1"},
		{"Loopback0 → Loop", "Loopback0", "Loop0"},

		// 守卫: 已是标准大写短名不反向展开
		{"GE2/25 保持(不反向展开)", "GE2/25", "GE2/25"},
		{"GE0/0/1 保持", "GE0/0/1", "GE0/0/1"},
		{"XGE1/0/1 保持", "XGE1/0/1", "XGE1/0/1"},
		{"HGE1/49 保持(不反向展开)", "HGE1/49", "HGE1/49"},
		{"FE0/1 保持", "FE0/1", "FE0/1"},

		// 对称化: cisco 小写短名 → 大写短名
		{"gi2/25 (Cisco) → GE", "gi2/25", "GE2/25"},
		{"Gi2/25 → GE", "Gi2/25", "GE2/25"},
		{"te1/0/1 (Cisco) → XGE", "te1/0/1", "XGE1/0/1"},
		{"xe2/25 → XGE", "xe2/25", "XGE2/25"},
		{"fo1/1 → FOE", "fo1/1", "FOE1/1"},
		{"fge1/1 (Cisco Nexus) → FOE", "fge1/1", "FOE1/1"},
		{"hge1/49 → HGE", "hge1/49", "HGE1/49"},
		{"twe1/1 → TWE", "twe1/1", "TWE1/1"},
		{"tw1/1 → TWE", "tw1/1", "TWE1/1"},
		{"fa0/1 → FE", "fa0/1", "FE0/1"},
		{"TE1/0/1 → XGE", "TE1/0/1", "XGE1/0/1"},
		{"ge2/25 小写 → GE", "ge2/25", "GE2/25"},

		// 全小写 full name → 大写短名
		{"gigabitethernet0/0/1 → GE", "gigabitethernet0/0/1", "GE0/0/1"},
		{"tengige1/0/1 → XGE", "tengige1/0/1", "XGE1/0/1"},
		{"fastethernet0/1 → FE", "fastethernet0/1", "FE0/1"},
		{"hundredgige1/49 → HGE", "hundredgige1/49", "HGE1/49"},
		{"fortygige1/1 → FOE", "fortygige1/1", "FOE1/1"},
		{"twentyfivegige1/1 → TWE", "twentyfivegige1/1", "TWE1/1"},

		// et 短名 (Huawei 25G)
		{"et3/46 → ET", "et3/46", "ET3/46"},
		{"ET3/46 保持", "ET3/46", "ET3/46"},

		// 空格剥离
		{"GigabitEthernet 2/25 去空格", "GigabitEthernet 2/25", "GE2/25"},
		{"ge 2/25 去空格", "ge 2/25", "GE2/25"},
		{"前导空格", " GigabitEthernet0/0/1", "GE0/0/1"},
		{"尾随空格", "GigabitEthernet0/0/1 ", "GE0/0/1"},
		{"大小写混写 Ge2/25 → GE2/25", "Ge2/25", "GE2/25"},

		// 尾部字母垃圾剥离(2026-07-02 mac-iface-security-suffix)
		// 华为 security 类型 MAC 接口名粘连 security 标记,必须剥离尾部
		{"GE1/0/4SECURITY → 剥离尾部", "GE1/0/4SECURITY", "GE1/0/4"},
		{"GE1/0/4security → 剥离尾部", "GE1/0/4security", "GE1/0/4"},
		{"GE1/0/4 security 去空格+剥离", "GE1/0/4 security", "GE1/0/4"},
		{"GE0/0/1.5 子接口点保留", "GE0/0/1.5", "GE0/0/1.5"},
		{"XGE1/0/1dynamic → 剥离", "XGE1/0/1dynamic", "XGE1/0/1"},

		// 乱码清理
		{"GEGigabitEthernet5/29 → GE5/29", "GEGigabitEthernet5/29", "GE5/29"},
		{"GEgabitetherngigabitethernet4/48 → GE4/48", "GEgabitetherngigabitethernet4/48", "GE4/48"},
		{"GEngigabitethernet5/34 → GE5/34", "GEngigabitethernet5/34", "GE5/34"},
		{"XGengigabitethernet0/49 → XGE0/49", "XGengigabitethernet0/49", "XGE0/49"},
		{"XGigabitEthernet0/0/1 → XGE0/0/1", "XGigabitEthernet0/0/1", "XGE0/0/1"},
		// GEet5/34: GE 后是 'e' 非数字,守卫不触发;阶段2 不识别;原样返回(留给迁移显式处理)
		{"GEet5/34 保持原样", "GEet5/34", "GEet5/34"},

		// 边缘
		{"空串", "", ""},
		{"NULL", "NULL", "NULL"},
		{"Vlanif100", "Vlanif100", "Vlanif100"},
		{"Vlan100", "Vlan100", "Vlan100"},
		{"未知前缀原样", "Port-Channel1", "Port-Channel1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InterfaceName(tt.input); got != tt.expected {
				t.Errorf("InterfaceName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestInterfaceName_PrefixOrder 验证前缀匹配顺序(twe 早于 tw,et 早于 te)。
func TestInterfaceName_PrefixOrder(t *testing.T) {
	cases := map[string]string{
		"twe1/1": "TWE1/1",
		"tw1/1":  "TWE1/1",
		"te1/0/1": "XGE1/0/1",
	}
	for in, want := range cases {
		if got := InterfaceName(in); got != want {
			t.Errorf("InterfaceName(%q) = %q, want %q", in, got, want)
		}
	}
}
