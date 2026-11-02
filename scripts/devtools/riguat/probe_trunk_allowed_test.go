package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

// TestModeSwitchAllowed 受控写白名单：只有两条模式切换命令被允许，其余写命令一律拒绝。
func TestModeSwitchAllowed(t *testing.T) {
	// 允许的两条
	ok1, restore := modeSwitchAllowed("switchport mode trunk")
	assert.True(t, ok1, "切 trunk 必须被允许")
	assert.Equal(t, "switchport mode access", restore, "切 trunk 必须带出它的还原命令")
	ok2, _ := modeSwitchAllowed("switchport mode access")
	assert.True(t, ok2, "还原 access 必须被允许")

	// 其它写命令一律拒绝
	for _, c := range []string{
		"switchport trunk allowed vlan 3990",
		"switchport mode hybrid",
		"no switchport mode trunk",
		"switchport mode access vlan 3990",
	} {
		ok, _ := modeSwitchAllowed(c)
		assert.False(t, ok, "非白名单的写命令必须被拒绝: %q", c)
	}
}

// TestModeSwitchCompleteRequiresPair 模式切换**必须成对**：有 trunk 没有 access 还原 ⇒ 拒绝。
//
// 这是 lead 明确要求的护栏：「trunk 必配 access 还原 —— 测试要断言没有 access
// 还原就不允许 trunk」。没有它，一次模式切换失败就可能把口留在 trunk 模式
// （= 转发行为改变 = 潜在业务中断）。
func TestModeSwitchCompleteRequiresPair(t *testing.T) {
	// 只有 trunk ⇒ 不完整 ⇒ 拒绝
	assert.False(t, modeSwitchComplete([]string{"switchport mode trunk"}),
		"只有 trunk 没有 access 还原 ⇒ 必须拒绝执行")
	// 成对 ⇒ 完整
	assert.True(t, modeSwitchComplete([]string{"switchport mode trunk", "switchport mode access"}),
		"trunk + access 成对 ⇒ 允许")
	// 只有 access ⇒ 不完整（无切换却有还原，逻辑上不成对）
	assert.False(t, modeSwitchComplete([]string{"switchport mode access"}),
		"只有 access 没有 trunk ⇒ 不成对")
	// 空集合 ⇒ 视为不需要切换，允许（纯只读路径）
	assert.True(t, modeSwitchComplete(nil), "空集合（纯只读路径）⇒ 允许")
}

// TestParsePortMode 端口模式判定（锐捷 access 是默认模式 ⇒ running-config 不落 mode 行）。
func TestParsePortMode(t *testing.T) {
	assert.Contains(t, parsePortMode(nil), "access", "无 mode 行 ⇒ 默认 access")
	assert.Contains(t, parsePortMode([]string{"description X"}), "access")
	assert.Equal(t, "trunk", parsePortMode([]string{"switchport mode trunk"}))
	assert.Equal(t, "hybrid", parsePortMode([]string{"switchport mode hybrid"}))
}
