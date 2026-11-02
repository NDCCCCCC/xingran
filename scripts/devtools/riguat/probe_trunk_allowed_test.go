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

// TestControlledRunAllowed 受控实跑白名单：只允许那 4 条 trunk allowed 命令。
func TestControlledRunAllowed(t *testing.T) {
	for _, c := range controlledRunCmds {
		assert.True(t, controlledRunAllowed(c), "白名单内的命令必须被允许: %q", c)
	}
	for _, c := range []string{
		"switchport trunk allowed vlan 3990",         // 无关键字形态（真机已确证 Invalid input）
		"switchport trunk allowed vlan add 3990",     // 换 VLAN 号 ⇒ 不在白名单
		"switchport trunk allowed vlan replace 3991", // replace 真机已确证不存在
		"switchport trunk allowed vlan only 3992",    // 换 VLAN 号 ⇒ 不在白名单
		"switchport trunk allowed vlan 3990-3995",
		"no switchport trunk allowed vlan 3990",
		"switchport mode hybrid",
		"switchport hybrid allowed vlan untagged 3992",
	} {
		assert.False(t, controlledRunAllowed(c), "非白名单的命令必须被拒绝: %q", c)
	}
}

// TestControlledRunCoversSecondLayer 受控实跑必须覆盖 add / remove / replace 三形态。
func TestControlledRunCoversSecondLayer(t *testing.T) {
	joined := strings.Join(controlledRunCmds, "\n")
	//   ① only 语义（允许式 vs 排除式）、② add 语义（并集 vs 替换）、③ remove 语义
	for _, need := range []string{
		"switchport trunk allowed vlan only 3990",
		"switchport trunk allowed vlan add 3991",
		"switchport trunk allowed vlan remove 3991",
	} {
		assert.Contains(t, joined, need, "受控实跑命令表缺少 %q", need)
	}
}

// TestJudgeRunOutcome 三态判读：被拒 / 生效 / **无错但不落地**（最关键的那一态）。
func TestJudgeRunOutcome(t *testing.T) {
	mk := func(lines ...string) probeBaseline {
		return probeBaseline{ConfigLinesRaw: lines, VlanIDs: []int{1}}
	}

	// ① 被拒 ⇒ 关键字不存在
	o := judgeRunOutcome("switchport trunk allowed vlan add 3990", true, mk("a"), mk("a"), mk("a"))
	assert.Equal(t, verdictRejected, o.Verdict)
	assert.False(t, o.Changed)
	assert.False(t, o.SideEffect)

	// ② 被接受且**allowed 行**变了 ⇒ 存在且生效
	//    （判据 = 回读的 `switchport trunk allowed vlan` 那一行原文，不是整块 diff）
	o = judgeRunOutcome("switchport trunk allowed vlan only 3990", false,
		mk("switchport trunk allowed vlan only 1-3989,3991-4094"),
		mk("switchport trunk allowed vlan only 3990"), mk("switchport trunk allowed vlan only 1-3989,3991-4094"))
	assert.Equal(t, verdictLanded, o.Verdict)
	assert.True(t, o.Changed)
	assert.Equal(t, "switchport trunk allowed vlan only 3990", o.AllowedAfter)

	// ③ 被接受但回读无变化 ⇒ ⚠️ 无错但不落地（hybrid 的 tagged 就是这样）
	o = judgeRunOutcome("switchport trunk allowed vlan add 3991", false,
		mk("switchport trunk allowed vlan only 3990"),
		mk("switchport trunk allowed vlan only 3990"), mk("switchport trunk allowed vlan only 3990"))
	assert.Equal(t, verdictSilentNoop, o.Verdict)
	assert.False(t, o.Changed)
	assert.False(t, o.Rejected)
}

// TestJudgeRunOutcome_SideEffect 副作用判定对「实跑期基线」判定（lead 2026-11-02 第四轮）。
//
// ★ 关键：「端口模式 access → trunk」是**实跑期期望状态**（实跑的前提就是切 trunk）
//
//	⇒ 若对「探测前」基线比就会误报「非预期副作用」⇒ 实跑被中途停掉。
func TestJudgeRunOutcome_SideEffect(t *testing.T) {
	// 实跑期基线：切 trunk 之后、实跑之前
	runBase := probeBaseline{ConfigLinesRaw: []string{"switchport trunk allowed vlan only 1-3989,3991-4094"},
		VlanIDs: []int{1}, PortMode: "trunk"}
	before := probeBaseline{ConfigLinesRaw: []string{"switchport trunk allowed vlan only 3990"},
		VlanIDs: []int{1}, PortMode: "trunk"}

	// ① 设备自动创建 VLAN（锐捷已确证会这么做）→ 预期内，不停止
	after := probeBaseline{ConfigLinesRaw: []string{"switchport trunk allowed vlan only 3990,3991"},
		VlanIDs: []int{1, 3991}, PortMode: "trunk"}
	o := judgeRunOutcome("switchport trunk allowed vlan add 3991", false, before, after, runBase)
	assert.False(t, o.SideEffect, "设备自动创建 VLAN 是预期内行为，不应停止实跑")
	assert.Equal(t, []int{3991}, o.VlanAdded, "新增 VLAN 应被记录")
	assert.Equal(t, verdictLanded, o.Verdict)

	// ② 端口模式被改动（相对实跑期基线）⇒ 非预期 ⇒ 副作用
	after2 := probeBaseline{ConfigLinesRaw: []string{"switchport trunk allowed vlan only 3990"},
		VlanIDs: []int{1}, PortMode: "hybrid"}
	o2 := judgeRunOutcome("switchport trunk allowed vlan only 3990", false, before, after2, runBase)
	assert.True(t, o2.SideEffect, "端口模式被改动必须判为副作用")

	// ③ 【误报回归】实跑期基线已是 trunk ⇒ 不会因「access → trunk」而误报
	probeBefore := probeBaseline{ConfigLinesRaw: []string{"description X"}, VlanIDs: []int{1}, PortMode: "access"}
	after3 := probeBaseline{ConfigLinesRaw: []string{"switchport trunk allowed vlan only 3990"},
		VlanIDs: []int{1}, PortMode: "trunk"}
	o3 := judgeRunOutcome("switchport trunk allowed vlan only 3990", false, probeBefore, after3, runBase)
	assert.False(t, o3.SideEffect,
		"实跑期基线已是 trunk ⇒ 不应因「access → trunk」误报副作用（lead 第四轮误报的根因）")
}

// TestParsePortMode_TrunkFeatureLine 端口模式判定修正：
// trunk 模式下设备**不落** `switchport mode trunk` 行 ⇒ 靠 trunk 特征行判定。
func TestParsePortMode_TrunkFeatureLine(t *testing.T) {
	// trunk 特征行 ⇒ trunk（第二版修的 bug：旧版只看 mode 行 ⇒ 误判 access）
	got := parsePortMode([]string{"switchport trunk allowed vlan 3990"})
	assert.Contains(t, got, "trunk", "有 trunk 特征行必须判为 trunk")
	// 显式 mode 行优先
	assert.Equal(t, "hybrid", parsePortMode([]string{"switchport mode hybrid"}))
	// 什么都没有 ⇒ access
	assert.Contains(t, parsePortMode([]string{"description X"}), "access")
}

// TestFindCmdResult 按命令文本取结果（不能按索引！）。
//
// ★ 这是 lead 2026-11-02 指出的「判读表与原始回显互相矛盾」的根因：
//
//	sendInterfaceCmds 返回的首项是 `interface <iface>`（进接口视图）、末项是 `exit`
//	⇒ 取 res[0] 拿到的是「进视图」那条 ⇒ 「设备是否报错」列恒为「否」。
func TestFindCmdResult(t *testing.T) {
	results := []cmdResult{
		{Command: "interface GigabitEthernet 1/25", Failed: false},
		{Command: "switchport trunk allowed vlan only 3990", Failed: true, Output: "% Invalid input detected"},
		{Command: "exit", Failed: false},
	}
	r, ok := findCmdResult(results, "switchport trunk allowed vlan only 3990")
	assert.True(t, ok)
	assert.True(t, r.Failed, "被测命令报错时必须能被取到")
	assert.Contains(t, r.Output, "Invalid input")
	_, ok2 := findCmdResult(results, "switchport trunk allowed vlan add 3991")
	assert.False(t, ok2)
}

// TestExtractTrunkAllowedLine 取 allowed 行原文（判读的唯一可信依据）。
func TestExtractTrunkAllowedLine(t *testing.T) {
	assert.Equal(t, "switchport trunk allowed vlan only 1-3989,3991-4094",
		extractTrunkAllowedLine([]string{
			"interface GigabitEthernet 1/25",
			" switchport mode trunk",
			" switchport trunk allowed vlan only 1-3989,3991-4094",
		}))
	assert.Equal(t, "", extractTrunkAllowedLine([]string{"description X"}))
}

// TestZeroResidualVerdict 零残留判定必须以「探测前」为基准。
//
// ★ lead 2026-11-02 第五轮误报的根因：基准用了循环内的 before（实跑结束时的 trunk 态）
// ⇒ 把「正确的还原结果（trunk→access）」判成不一致 ⇒ 安全网误报。
func TestZeroResidualVerdict(t *testing.T) {
	// 探测前：access、0 字节、0 行
	initial := probeBaseline{ConfigBytes: 0, ConfigLines: 0, PortMode: "access（默认）", VlanIDs: []int{1}}
	// 还原后：应与探测前完全一致
	after := probeBaseline{ConfigBytes: 0, ConfigLines: 0, PortMode: "access（默认）", VlanIDs: []int{1}}
	sb, sl, sm, sv := zeroResidualVerdict(initial, after)
	assert.True(t, sb && sl && sm && sv, "还原后与探测前一致 ⇒ 零残留应通过")

	// 真残留（配置块变了）⇒ 必须报不一致
	dirty := probeBaseline{ConfigBytes: 40, ConfigLines: 2, PortMode: "trunk", VlanIDs: []int{1}}
	sb, sl, sm, _ = zeroResidualVerdict(initial, dirty)
	assert.False(t, sb, "配置块字节变了 ⇒ 零残留必须报不一致")
	assert.False(t, sl)
	assert.False(t, sm)

	// VLAN 表变了 ⇒ 也必须报不一致
	vlanDirty := probeBaseline{ConfigBytes: 0, ConfigLines: 0, PortMode: "access（默认）", VlanIDs: []int{1, 3990}}
	_, _, _, sv = zeroResidualVerdict(initial, vlanDirty)
	assert.False(t, sv, "VLAN 表多了 VLAN ⇒ 零残留必须报不一致")
}

// TestExtractTrunkNativeLine native 行原文提取（native 取证的判据）。
func TestExtractTrunkNativeLine(t *testing.T) {
	assert.Equal(t, "switchport trunk native vlan 3990",
		extractTrunkNativeLine([]string{
			"interface GigabitEthernet 1/25",
			" switchport mode trunk",
			" switchport trunk native vlan 3990",
		}))
	// 没有 native 行 ⇒ 空串（access 模式下无此行）
	assert.Equal(t, "", extractTrunkNativeLine([]string{"description X"}))
	// 不得误把 allowed 行当成 native 行
	assert.Equal(t, "", extractTrunkNativeLine([]string{"switchport trunk allowed vlan only 3990"}))
}

// TestControlledRunCoversNative native 必须在受控实跑白名单里（第二个 P0 候选）。
func TestControlledRunCoversNative(t *testing.T) {
	joined := strings.Join(controlledRunCmds, "\n")
	assert.Contains(t, joined, "switchport trunk native vlan 3990",
		"受控实跑必须包含 native 形态（trunk allowed 已被证伪 ⇒ native 同族也可能需要 only）")
	// native 必须放第一条（allowed 列表还是设备默认态 ⇒ 语义判读更干净）
	assert.Equal(t, "switchport trunk native vlan 3990", controlledRunCmds[0],
		"native 实跑应放第一条")
}

// TestJudgeRunOutcome_NativeUsesNativeLine native 命令必须比 native 行判读（lead 2026-11-02 第六轮定位的 bug）。
//
// ★ 原 bug：三态判定对**所有命令**都拿 allowed 行作判据 ⇒ native 命令执行后
//
//	allowed 行本来就不变（native 与 allowed 是独立维度）⇒ 误判「无错但不落地」。
func TestJudgeRunOutcome_NativeUsesNativeLine(t *testing.T) {
	// 执行前无 native 行、无 allowed 行；执行后 native 行落盘
	before := probeBaseline{ConfigLinesRaw: []string{}, PortMode: "trunk"}
	after := probeBaseline{ConfigLinesRaw: []string{"switchport trunk native vlan 3990"}, PortMode: "trunk"}
	o := judgeRunOutcome("switchport trunk native vlan 3990", false, before, after, before)
	assert.Equal(t, verdictLanded, o.Verdict, "native 行落盘 ⇒ 必须判「存在且生效」（不得误判「无错但不落地」）")
	assert.True(t, o.Changed)
	assert.Equal(t, "switchport trunk native vlan 3990", o.VerdictAfter, "判读行应为 native 行而非 allowed 行")

	// 反向：native 落盘但 **allowed 行没动** ⇒ 仍必须判「生效」（判读行跟 native 命令走）
	before2 := probeBaseline{ConfigLinesRaw: []string{}, PortMode: "trunk"}
	after2 := probeBaseline{ConfigLinesRaw: []string{"switchport trunk native vlan 3990"}, PortMode: "trunk"}
	o2 := judgeRunOutcome("switchport trunk native vlan 3990", false, before2, after2, before2)
	assert.Equal(t, verdictLanded, o2.Verdict)

	// allowed 命令仍比 allowed 行
	before3 := probeBaseline{ConfigLinesRaw: []string{"switchport trunk allowed vlan only 3990"}, PortMode: "trunk"}
	after3 := probeBaseline{ConfigLinesRaw: []string{"switchport trunk allowed vlan only 3990,3991"}, PortMode: "trunk"}
	o3 := judgeRunOutcome("switchport trunk allowed vlan add 3991", false, before3, after3, before3)
	assert.Equal(t, verdictLanded, o3.Verdict)
	assert.Contains(t, o3.VerdictAfter, "allowed vlan only 3990,3991")
}
