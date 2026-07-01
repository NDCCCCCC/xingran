package portcollection

import "testing"

// TestNormalizeInterfaceName 锁住跨厂商接口名归一化等价关系
//
// 回归目标:
//   - 同一物理端口跨厂商命名(GE/Gi/GigabitEthernet/te/TenGigE/xe/twe/tw/hge/fge/fa)
//     经 NormalizeInterfaceName 后落到一致的"短名"或"全称"上,确保下游 SQL view
//     (reconciliation_physical_chain) 的 JOIN 能匹配
//   - 2026-06-29 物理链路 view 失配根因之一即 prefixList 缺 ge/xe/twe/tw/hge/fge
//   - 2026-07-01 port-mac-format-unify 提升为导出符号,本测试函数名/调用点同步
//   - Phase 45 R5 修复(GetPhysicalDevices 已用 REGEXP_REPLACE 折叠)与 Go 函数对齐
func TestNormalizeInterfaceName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// === 全称 → 短名 ===
		{
			name:     "全称 GigabitEthernet 转 GE",
			input:    "GigabitEthernet2/25",
			expected: "GE2/25",
		},
		{
			name:     "全称 TenGigE 转 XGE",
			input:    "TenGigE1/0/1",
			expected: "XGE1/0/1",
		},
		{
			name:     "全称 FortyGigE 转 FOE",
			input:    "FortyGigE1/1",
			expected: "FOE1/1",
		},
		{
			name:     "全称 HundredGigE 转 HGE",
			input:    "HundredGigE1/49",
			expected: "HGE1/49",
		},
		{
			name:     "全称 TwentyFiveGigE 转 TWE",
			input:    "TwentyFiveGigE1/1",
			expected: "TWE1/1",
		},
		{
			name:     "全称 FastEthernet 转 FE",
			input:    "FastEthernet0/1",
			expected: "FE0/1",
		},
		{
			name:     "全称 Loopback 不变(短名 Loop)",
			input:    "Loopback0",
			expected: "Loop0",
		},

		// === 短名 → 全称 ===
		{
			name:     "短名 gi (Cisco) → GigabitEthernet",
			input:    "gi2/25",
			expected: "GigabitEthernet2/25",
		},
		{
			name:     "短名 ge (HP/Huawei) → GigabitEthernet [2026-06-29 补]",
			input:    "ge2/25",
			expected: "GigabitEthernet2/25",
		},
		{
			name:     "短名 ge 大写 GE → GigabitEthernet",
			input:    "GE2/25",
			expected: "GigabitEthernet2/25",
		},
		{
			name:     "短名 Gi 大写 → GigabitEthernet",
			input:    "Gi2/25",
			expected: "GigabitEthernet2/25",
		},
		{
			name:     "短名 te (Cisco) → TenGigE",
			input:    "te1/0/1",
			expected: "TenGigE1/0/1",
		},
		{
			name:     "短名 xe (部分厂商 10G) → TenGigE [2026-06-29 补]",
			input:    "xe2/25",
			expected: "TenGigE2/25",
		},
		{
			name:     "短名 fo (Cisco) → FortyGigE",
			input:    "fo1/1",
			expected: "FortyGigE1/1",
		},
		{
			name:     "短名 hge (部分 Cisco/Huawei 100G) → HundredGigE [2026-06-29 补]",
			input:    "hge1/49",
			expected: "HundredGigE1/49",
		},
		{
			name:     "短名 fge (Cisco Nexus 40G) → FortyGigE [2026-06-29 补]",
			input:    "fge1/1",
			expected: "FortyGigE1/1",
		},
		{
			name:     "短名 twe (25G) → TwentyFiveGigE [2026-06-29 补]",
			input:    "twe1/1",
			expected: "TwentyFiveGigE1/1",
		},
		{
			name:     "短名 tw (25G) → TwentyFiveGigE [2026-06-29 补]",
			input:    "tw1/1",
			expected: "TwentyFiveGigE1/1",
		},
		{
			name:     "短名 fa (100M) → FastEthernet",
			input:    "fa0/1",
			expected: "FastEthernet0/1",
		},
		{
			name:     "短名 vl → Vlan",
			input:    "vl100",
			expected: "Vlan100",
		},

		// === 等价关系(关键回归守护:这些必须产生同一个全称才能 JOIN) ===
		{
			name:     "等价: GE/Gi/GigabitEthernet 全部归一为 GigabitEthernet",
			input:    "GE2/25",
			expected: "GigabitEthernet2/25",
		},
		{
			name:     "等价: TE/XE/TenGigE 全部归一为 TenGigE",
			input:    "TE1/0/1",
			expected: "TenGigE1/0/1",
		},

		// === 空格剥离 ===
		{
			name:     "空格剥离:GigabitEthernet 2/25",
			input:    "GigabitEthernet 2/25",
			expected: "GE2/25",
		},
		{
			name:     "空格剥离:ge 2/25",
			input:    "ge 2/25",
			expected: "GigabitEthernet2/25",
		},

		// === 大小写 ===
		// 注:fullToShort 走 strings.HasPrefix(name, full) 是 case-sensitive,
		// 所以全大写 GIGABITETHERNET 会回退到短名路径(走 lowerName → gigabitethernet → GigabitEthernet)
		// 历史行为一致,非本次修复范围
		{
			name:     "大小写:Ge 大小写混写 → GigabitEthernet(走 short→full)",
			input:    "Ge2/25",
			expected: "GigabitEthernet2/25",
		},

		// === 边缘情况 ===
		{
			name:     "空字符串不变",
			input:    "",
			expected: "",
		},
		{
			name:     "NULL 接口保留",
			input:    "NULL",
			expected: "NULL",
		},
		{
			name:     "未知前缀原样返回",
			input:    "Port-Channel1",
			expected: "Port-Channel1",
		},
		{
			name:     "Vlanif 不变(短名=全称)",
			input:    "Vlanif100",
			expected: "Vlanif100",
		},

		// === 2026-07-01 port-mac-format-unify 新增契约 ===
		// 注:全小写 full name(gigabitethernet...)走 short→full 路径,落到 GigabitEthernet
		// 这是当前函数行为,符合"等价类:GE/Gi/GigabitEthernet 全部归一为 GigabitEthernet"
		// 既有断言.真实采集器侧由 collector 调用 NormalizeInterfaceName,
		// 把已是大写 full name 的 GigabitEthernet0/0/1 折叠为短名 GE0/0/1.
		{
			name:     "[新] 全小写 full name gigabitethernet0/0/1 -> 全称 GigabitEthernet0/0/1(走 short->full 路径)",
			input:    "gigabitethernet0/0/1",
			expected: "GigabitEthernet0/0/1",
		},
		{
			name:     "[新] 前导空格 GigabitEthernet0/0/1 -> GE0/0/1",
			input:    " GigabitEthernet0/0/1",
			expected: "GE0/0/1",
		},
		{
			name:     "[新] 尾随空格 GigabitEthernet0/0/1 -> GE0/0/1",
			input:    "GigabitEthernet0/0/1 ",
			expected: "GE0/0/1",
		},
		{
			name:     "[新] TenGigE 全小写 tengige1/0/1 -> TenGigE1/0/1(走 short->full 路径)",
			input:    "tengige1/0/1",
			expected: "TenGigE1/0/1",
		},
		{
			name:     "[新] FastEthernet 全小写 fastethernet0/1 -> FastEthernet0/1(走 short->full 路径)",
			input:    "fastethernet0/1",
			expected: "FastEthernet0/1",
		},
		{
			name:     "[新] HundredGigE 全小写 hundredgige1/49 -> HundredGigE1/49(走 short->full 路径)",
			input:    "hundredgige1/49",
			expected: "HundredGigE1/49",
		},
		{
			name:     "[新] FortyGigE 全小写 fortygige1/1 -> FortyGigE1/1(走 short->full 路径)",
			input:    "fortygige1/1",
			expected: "FortyGigE1/1",
		},
		{
			name:     "[新] TwentyFiveGigE 全小写 twentyfivegige1/1 -> TwentyFiveGigE1/1(走 short->full 路径)",
			input:    "twentyfivegige1/1",
			expected: "TwentyFiveGigE1/1",
		},

		// === 2026-07-01 port-mac-format-unify 阶段 2:Huawei et 短名 + 乱码清理 ===
		{
			name:     "[新] 短名 et (Huawei 25G) -> TwentyFiveGigE",
			input:    "et3/46",
			expected: "TwentyFiveGigE3/46",
		},
		{
			name:     "[新] 短名 et 大写 ET -> TwentyFiveGigE",
			input:    "ET3/46",
			expected: "TwentyFiveGigE3/46",
		},
		// 乱码(短+全+数字)清理
		{
			name:     "[新] 乱码 GEGigabitEthernet5/29 -> GE5/29",
			input:    "GEGigabitEthernet5/29",
			expected: "GE5/29",
		},
		{
			name:     "[新] 乱码 GEgabitetherngigabitethernet4/48 -> GE4/48",
			input:    "GEgabitetherngigabitethernet4/48",
			expected: "GE4/48",
		},
		{
			name:     "[新] 乱码 GEngigabitethernet5/34 -> GE5/34",
			input:    "GEngigabitethernet5/34",
			expected: "GE5/34",
		},
		{
			name:     "[新] 乱码 XGengigabitethernet0/49 -> XGE0/49",
			input:    "XGengigabitethernet0/49",
			expected: "XGE0/49",
		},
		{
			name:     "[新] 乱码 XGigabitEthernet0/0/1 -> XGE0/0/1",
			input:    "XGigabitEthernet0/0/1",
			expected: "XGE0/0/1",
		},
		// 注: GEet5/34 留待 migration 187 单独处理(中间是 'et' 不是 gigabit 残,
		// 阶段 0 不触发;阶段 1+ 也不识别;需要 SQL 显式 case-when)
		{
			name:     "[新] 乱码 GEet5/34 保持原样(留给 migration 187 显式处理)",
			input:    "GEet5/34",
			expected: "GEet5/34",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeInterfaceName(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeInterfaceName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestNormalizeInterfaceName_PrefixOrder 验证前缀匹配顺序正确(防止 tw 误吞 twe)
func TestNormalizeInterfaceName_PrefixOrder(t *testing.T) {
	// "twe1/1" 应该匹配 TwentyFiveGigE 而不是 Tw(未知前缀)
	got := NormalizeInterfaceName("twe1/1")
	if got != "TwentyFiveGigE1/1" {
		t.Errorf("前缀顺序错误:twe1/1 = %q, want TwentyFiveGigE1/1", got)
	}

	// "tw1/1" 应该匹配 TwentyFiveGigE
	got = NormalizeInterfaceName("tw1/1")
	if got != "TwentyFiveGigE1/1" {
		t.Errorf("tw1/1 = %q, want TwentyFiveGigE1/1", got)
	}

	// "te1/0/1" 应该匹配 TenGigE 而非 twe(防止 twe 误吞 te)
	got = NormalizeInterfaceName("te1/0/1")
	if got != "TenGigE1/0/1" {
		t.Errorf("te1/0/1 = %q, want TenGigE1/0/1", got)
	}
}