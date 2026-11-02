package main

import (
	"strings"
	"testing"
)

// TestIsReadOnlyProbeCmd 只读白名单判定（任务 A 的安全护栏）。
//
// 探针模式的唯一安全保证就是这条白名单：只放行「以 ? 结尾的帮助查询」
// 与「show 前缀的只读采集」，其余（尤其 switchport/no 赋值类）一律不得下发。
func TestIsReadOnlyProbeCmd(t *testing.T) {
	cases := []struct {
		cmd  string
		want bool
	}{
		// 允许：帮助查询（参数位）
		{"switchport trunk allowed vlan ?", true},
		{"switchport trunk allowed vlan add ?", true},
		{"switchport trunk allowed vlan remove ?", true},
		{"switchport trunk allowed vlan replace ?", true},
		{"switchport trunk allowed vlan 3990 ?", true},
		{"switchport trunk ?", true},
		// 允许：只读采集
		{"show vlan", true},
		{"show running-config interface GigabitEthernet 1/25", true},
		{"SHOW VLAN", true},
		// 拒绝：会改配置的一切形态
		{"switchport trunk allowed vlan 3990", false},
		{"switchport trunk allowed vlan add 3990", false},
		{"switchport trunk allowed vlan remove 3990", false},
		{"switchport trunk allowed vlan replace 3990", false},
		{"no switchport trunk allowed vlan 3990", false},
		{"switchport mode hybrid", false},
		{"switchport hybrid allowed vlan untagged 3992", false},
		{"", false},
		{"   ", false},
	}
	for _, c := range cases {
		if got := isReadOnlyProbeCmd(c.cmd); got != c.want {
			t.Errorf("isReadOnlyProbeCmd(%q) = %v, want %v", c.cmd, got, c.want)
		}
	}
}

// TestTrunkAllowedProbeCmdsAreReadOnly 探针命令表**每一条**都必须过只读白名单。
//
// 这是任务 A 的边界保证（lead 明确要求「只允许 ? 帮助命令，任何改配置的一律不跑」）：
// 将来若有人往表里加命令，这条测试会立刻挡住写命令。
func TestTrunkAllowedProbeCmdsAreReadOnly(t *testing.T) {
	cmds := trunkAllowedProbeCmds()
	if len(cmds) == 0 {
		t.Fatal("探针命令表为空")
	}
	for _, c := range cmds {
		if !isReadOnlyProbeCmd(c) {
			t.Errorf("探针命令 %q 不是只读命令 —— 禁止下发", c)
		}
		if !strings.HasSuffix(strings.TrimSpace(c), "?") {
			t.Errorf("探针命令 %q 必须以 ? 结尾（帮助查询）", c)
		}
	}
}

// TestTrunkAllowedProbeCoversSecondLayer 探针表必须覆盖任务 A 要查的四个形态。
func TestTrunkAllowedProbeCoversSecondLayer(t *testing.T) {
	joined := strings.Join(trunkAllowedProbeCmds(), "\n")
	for _, need := range []string{
		"switchport trunk allowed vlan ?",
		"switchport trunk allowed vlan add ?",
		"switchport trunk allowed vlan remove ?",
		"switchport trunk allowed vlan replace ?",
	} {
		if !strings.Contains(joined, need) {
			t.Errorf("探针命令表缺少 %q", need)
		}
	}
}
