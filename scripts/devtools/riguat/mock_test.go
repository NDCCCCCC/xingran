package main

// 无设备模式自测：用 mock 锐捷设备跑完整「下发 → 还原 → diff」闭环。
//
// 为什么需要（lead 要求 3）：真机上还原因"少了 interface 前缀"静默失效，
// 且失败发生在生产设备上、批次中途才发现。此处的 mock 严格模拟锐捷 RGOS 的两个关键行为：
//
//  1. **接口级命令在 (config)# 顶层下发会被拒绝**（"% Invalid input detected"）——
//     正是真机 bug 的放大器：漏掉 interface 前缀时，命令静默不生效；
//  2. **no dot1x port-control auto 会连带清除 default-user-limit 与 mac-auth-bypass**
//     —— 真机上已观测到的连带行为。
//
// 因此：任何还原组装一旦丢掉 interface 前缀，本自测立刻红。

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
)

// mockRuijie 模拟锐捷 RGOS 的接口配置状态机。
type mockRuijie struct {
	lines    []string        // 当前接口配置行
	inIface  bool            // 是否处于接口视图
	failCmds map[string]bool // 指定命令永久失效（模拟设备不接受）
	failOnce map[string]bool // 指定命令首次失效（模拟偶发失败 → 触发自愈重试）
	rejected []string        // 被拒绝的命令记录
}

func newMockRuijie(baseline []string) *mockRuijie {
	return &mockRuijie{
		lines:    append([]string{}, baseline...),
		failCmds: map[string]bool{},
		failOnce: map[string]bool{},
	}
}

// sendInterfaceCmds 实现 interfaceSender（mock 版）。
func (m *mockRuijie) sendInterfaceCmds(body []string) []cmdResult {
	var out []cmdResult
	// 进入接口视图
	out = append(out, m.apply("interface GigabitEthernet 1/1"))
	for _, c := range body {
		out = append(out, m.apply(c))
	}
	out = append(out, m.apply("exit"))
	return out
}

// apply 应用一条命令到 mock 设备。
func (m *mockRuijie) apply(cmd string) cmdResult {
	res := cmdResult{Command: cmd}
	c := strings.TrimSpace(cmd)
	switch {
	case strings.HasPrefix(strings.ToLower(c), "interface "):
		m.inIface = true
		return res
	case c == "exit":
		m.inIface = false
		return res
	}
	if !m.inIface {
		// ★ 真机关键行为：接口级命令在 (config)# 顶层被设备拒绝
		m.rejected = append(m.rejected, c)
		res.Failed = true
		res.Output = "% Invalid input detected at '^' marker."
		return res
	}
	if m.failCmds[c] {
		m.rejected = append(m.rejected, c)
		res.Failed = true
		res.Output = "% Invalid input detected at '^' marker."
		return res
	}
	if m.failOnce[c] {
		delete(m.failOnce, c)
		m.rejected = append(m.rejected, c)
		res.Failed = true
		res.Output = "% Invalid input detected at '^' marker."
		return res
	}
	m.applyConfig(c)
	return res
}

// applyConfig 应用接口级配置语义。
func (m *mockRuijie) applyConfig(c string) {
	lower := strings.ToLower(c)
	switch {
	case lower == "no description":
		m.removePrefix("description ")
	case strings.HasPrefix(lower, "description "):
		m.set("description " + strings.TrimPrefix(c, "description "))
	case lower == "no switchport access vlan":
		m.removePrefix("switchport access vlan ")
	case strings.HasPrefix(lower, "switchport access vlan "):
		m.set(c)
	case lower == "no switchport mode access":
		m.removePrefix("switchport mode ")
	case strings.HasPrefix(lower, "switchport mode "):
		// ★ 锐捷真实行为：access 是默认模式，running-config 里不落 mode 行
		//   （权威 9 行基线里确实没有 mode 行）→ 这里同样不落行。
		if strings.HasPrefix(lower, "switchport mode access") {
			return
		}
		m.set(c)
	case lower == "no dot1x port-control auto":
		// ★ 真机行为：连带清除 default-user-limit 与 mac-auth-bypass
		m.removePrefix("dot1x port-control auto")
		m.removePrefix("dot1x default-user-limit ")
		m.removePrefix("dot1x mac-auth-bypass ")
	case strings.HasPrefix(lower, "dot1x port-control auto"):
		m.set("dot1x port-control auto")
	case lower == "no dot1x default-user-limit":
		m.removePrefix("dot1x default-user-limit ")
	case strings.HasPrefix(lower, "dot1x default-user-limit "):
		m.set(c)
	case lower == "no dot1x mac-auth-bypass":
		m.removePrefix("dot1x mac-auth-bypass ")
	case strings.HasPrefix(lower, "dot1x mac-auth-bypass "):
		m.set(c)
	case lower == "no switchport port-security maximum":
		m.removePrefix("switchport port-security maximum ")
	case strings.HasPrefix(lower, "switchport port-security maximum "):
		m.set(c)
	case lower == "no switchport port-security mac-address sticky":
		m.removePrefix("switchport port-security mac-address sticky")
	case lower == "switchport port-security mac-address sticky":
		m.set(c)
	case strings.HasPrefix(lower, "no switchport port-security mac-address "):
		m.removePrefix("switchport port-security mac-address ")
	case strings.HasPrefix(lower, "switchport port-security mac-address "):
		m.set(c)
	case strings.HasPrefix(lower, "no switchport port-security binding "):
		m.removePrefix("switchport port-security binding ")
	case strings.HasPrefix(lower, "switchport port-security binding "):
		m.set(c)
	case lower == "no switchport port-security":
		m.removePrefix("switchport port-security")
	case lower == "switchport port-security":
		m.set(c)
	case lower == "no shutdown":
		m.removePrefix("shutdown")
	case lower == "shutdown":
		m.set("shutdown")
	default:
		// 其它命令（spanning-tree / rldp / switchport access vlan 之类）按覆盖写处理
		m.set(c)
	}
}

// set 覆盖写一行（按"命令关键字前缀"匹配已有行，命中则替换，未命中则追加）。
func (m *mockRuijie) set(line string) {
	key := configLineKey(line)
	for i, l := range m.lines {
		if configLineKey(l) == key {
			m.lines[i] = line
			return
		}
	}
	m.lines = append(m.lines, line)
}

// removePrefix 删除以 prefix 开头的行。
func (m *mockRuijie) removePrefix(prefix string) {
	var kept []string
	for _, l := range m.lines {
		if strings.HasPrefix(strings.ToLower(l), strings.ToLower(prefix)) {
			continue
		}
		kept = append(kept, l)
	}
	m.lines = kept
}

// configLineKey 配置行的"同一性"键。
//
// 语义：同一行配置的不同参数取值视为同一行（覆盖写），但**不同子命令**必须是不同键
// —— 例如 "switchport port-security" 与 "switchport port-security binding X Y" 是两行，
// "switchport port-security maximum 5" 又是另一行。
func configLineKey(line string) string {
	f := strings.Fields(strings.ToLower(line))
	switch {
	case len(f) == 0:
		return ""
	case len(f) == 1:
		return f[0]
	}
	switch f[0] {
	case "description", "shutdown":
		return f[0]
	case "switchport":
		switch {
		case f[1] == "port-security" && len(f) >= 3 && (f[2] == "binding" || f[2] == "maximum" || f[2] == "mac-address"):
			return f[0] + " " + f[1] + " " + f[2]
		case f[1] == "mac-address" && len(f) >= 3 && f[2] == "sticky":
			return f[0] + " " + f[1] + " " + f[2]
		default:
			return f[0] + " " + f[1]
		}
	case "dot1x":
		switch {
		case f[1] == "port-control" && len(f) >= 3:
			return f[0] + " " + f[1] + " " + f[2]
		case f[1] == "default-user-limit":
			return f[0] + " " + f[1]
		case f[1] == "mac-auth-bypass" && len(f) >= 3:
			return f[0] + " " + f[1] + " " + f[2]
		default:
			return f[0] + " " + f[1]
		}
	}
	return f[0] + " " + f[1]
}

// actionParamsForTest 测试参数。
func actionParamsForTest() actionParams {
	limit := 1
	return actionParams{
		CliIface:     "GigabitEthernet 1/1",
		VLAN:         399,
		MAC:          "00:11:22:33:44:55",
		IP:           "10.62.255.254",
		Desc:         "RIGUAT-TEST",
		MaxMAC:       5,
		Dot1xLimit:   &limit,
		BaselineVLAN: 318,
	}
}

// TestRestoreAllActionsNoDevice 无设备模式：12 个动作逐一跑「下发 → 还原 → diff」。
//
// 这是 lead 要求 3 的核心回归：修复前（重放基线行缺 interface 前缀）本用例必然失败。
func TestRestoreAllActionsNoDevice(t *testing.T) {
	vendor := models.VendorRuijie
	params := actionParamsForTest()

	for _, name := range actionNames {
		spec := actionRegistry[name]
		tpl := spec.Build(params)
		tpl.InterfaceName = params.CliIface
		rendered, err := portcollection.RenderCommand(vendor, spec.PortAction, tpl)
		if err != nil {
			t.Fatalf("[%s] 渲染失败: %v", name, err)
		}

		m := newMockRuijie(baselineRuijieGE11)

		// 下发动作（渲染结果自带 interface 前缀 → 走接口视图）
		for _, seg := range actionSegments(params.CliIface, rendered) {
			m.sendInterfaceCmds(seg)
		}

		// 还原：走**生产同一条**还原引擎（undo 影响项 + 基线重放 + 自证重试）
		outcome, err := restoreWithVerify(m, params.CliIface, spec, params, baselineRuijieGE11,
			func() ([]string, error) { return append([]string{}, m.lines...), nil },
			3,
			func(format string, args ...interface{}) {},
		)
		if err != nil {
			t.Fatalf("[%s] 还原引擎报错: %v", name, err)
		}
		if len(m.rejected) > 0 {
			t.Errorf("[%s] 有命令被设备拒绝（还原组装丢了 interface 前缀？）: %v", name, m.rejected)
		}
		if !outcome.OK {
			t.Errorf("[%s] 还原未通过: diff=%v 最终=%v", name, outcome.Diff, m.lines)
		}
		if d := diffConfig(baselineRuijieGE11, m.lines); len(d) != 0 {
			t.Errorf("[%s] 还原后与基线不一致: %v（最终状态 %v）", name, d, m.lines)
		}
	}
}

// actionSegments 把渲染结果拆成"组接口命令"（生产 sendActionCmds 的等价拆分）。
func actionSegments(cliIface string, rendered []string) [][]string {
	if len(rendered) == 0 {
		return nil
	}
	if isInterfaceEntry(rendered[0]) {
		return [][]string{rendered[1:]}
	}
	return [][]string{rendered}
}

// TestRestoreSequenceCarriesInterfacePrefix 锁定还原序列必须带 interface 前缀。
//
// 语义：序列由若干"组"组成，每组 = interface X → 若干命令 → exit。
// 因此：interface 次数 == exit 次数，且序列末条必须是 exit。
func TestRestoreSequenceCarriesInterfacePrefix(t *testing.T) {
	params := actionParamsForTest()
	for _, name := range actionNames {
		spec := actionRegistry[name]
		seq := restoreSequence(params.CliIface, spec, params, baselineRuijieGE11)
		if len(seq) == 0 {
			continue
		}
		if !isInterfaceEntry(seq[0]) {
			t.Errorf("[%s] 还原序列首条必须是 interface 视图切换，实际: %q", name, seq[0])
		}
		ifaces, exits := 0, 0
		for _, c := range seq {
			if isInterfaceEntry(c) {
				ifaces++
			}
			if c == "exit" {
				exits++
			}
		}
		if ifaces != exits {
			t.Errorf("[%s] interface/exit 不配对（%d/%d）: %v", name, ifaces, exits, seq)
		}
		if seq[len(seq)-1] != "exit" {
			t.Errorf("[%s] 还原序列必须以 exit 结尾: %v", name, seq)
		}
	}
}

// TestRestoreSelfHeal 验证自证重试 + 差异驱动修复能收敛。
//
// 场景：还原阶段 undo 命令被设备拒绝一次（模拟偶发失败），
// 第一轮 diff 非空 → 差异驱动修复补回 → 第二轮 diff 为空 → 判定成功。
func TestRestoreSelfHeal(t *testing.T) {
	spec := actionRegistry["portsec_max_mac"]
	params := actionParamsForTest()
	m := newMockRuijie(baselineRuijieGE11)

	// 模拟动作生效：端口安全上限被设置
	m.sendInterfaceCmds([]string{"switchport port-security maximum 5"})

	// 还原阶段：undo 命令首次失败（偶发）
	m.failOnce["no switchport port-security maximum"] = true

	outcome, err := restoreWithVerify(m, params.CliIface, spec, params, baselineRuijieGE11,
		func() ([]string, error) { return append([]string{}, m.lines...), nil },
		3,
		func(format string, args ...interface{}) {},
	)
	if err != nil {
		t.Fatalf("还原引擎报错: %v", err)
	}
	if !outcome.OK {
		t.Fatalf("自愈后仍未收敛: diff=%v 最终=%v", outcome.Diff, m.lines)
	}
	if outcome.Attempts < 2 {
		t.Fatalf("应至少尝试 2 轮，实际 %d", outcome.Attempts)
	}
	if len(outcome.Fixed) == 0 {
		t.Fatalf("应触发差异驱动修复")
	}
}

// TestRestoreFailsWhenUnfixable 差异无法自动修复时必须判定失败（不谎报成功）。
func TestRestoreFailsWhenUnfixable(t *testing.T) {
	spec := actionRegistry["description"]
	params := actionParamsForTest()
	m := newMockRuijie(baselineRuijieGE11)

	// 制造一个"多出且无法安全否定"的行：spanning-tree 类没有 no 形式
	m.lines = append(m.lines, "spanning-tree loopguard default")

	outcome, err := restoreWithVerify(m, params.CliIface, spec, params, baselineRuijieGE11,
		func() ([]string, error) { return append([]string{}, m.lines...), nil },
		2,
		func(format string, args ...interface{}) {},
	)
	if err != nil {
		t.Fatalf("还原引擎报错: %v", err)
	}
	if outcome.OK {
		t.Fatalf("存在无法修复的差异时不应判定成功")
	}
	if len(outcome.Diff) == 0 {
		t.Fatalf("应报告差异")
	}
}

// TestDiffFixCommands 差异驱动修复命令生成。
func TestDiffFixCommands(t *testing.T) {
	base := []string{"description 3D001", "switchport access vlan 318"}
	cur := []string{"switchport access vlan 318", "switchport port-security maximum 5"}
	fix := diffFixCommands(base, cur)
	if !reflect.DeepEqual(fix, []string{"description 3D001", "no switchport port-security maximum"}) {
		t.Fatalf("修复命令不符: %v", fix)
	}
}

// TestNegateConfigLine no 形式推导。
func TestNegateConfigLine(t *testing.T) {
	cases := map[string]string{
		"description RIGUAT-TEST":                                 "no description",
		"switchport access vlan 100":                              "no switchport access vlan",
		"switchport port-security maximum 5":                      "no switchport port-security maximum",
		"switchport port-security mac-address sticky":             "no switchport port-security mac-address sticky",
		"dot1x port-control auto":                                 "no dot1x port-control auto",
		"switchport port-security binding 0011.2233.4455 1.1.1.1": "no switchport port-security binding 0011.2233.4455 1.1.1.1",
	}
	for in, want := range cases {
		got, ok := negateConfigLine(in)
		if !ok || got != want {
			t.Errorf("negateConfigLine(%q)=(%q,%v) want (%q,true)", in, got, ok, want)
		}
	}
	if _, ok := negateConfigLine("spanning-tree bpduguard enable"); ok {
		t.Errorf("spanning-tree 不应被安全否定")
	}
}

// TestMockRejectsTopLevelInterfaceCommand 锁定 mock 的关键行为（接口命令在顶层被拒）。
func TestMockRejectsTopLevelInterfaceCommand(t *testing.T) {
	m := newMockRuijie(baselineRuijieGE11)
	res := m.apply("description X")
	if !res.Failed {
		t.Fatalf("顶层下发的接口命令应被拒绝")
	}
}

// TestPromptFilteredFromConfigLines prompt 行不得进入配置行集合。
func TestPromptFilteredFromConfigLines(t *testing.T) {
	raw := "interface GigabitEthernet 1/1\n description 3D001\n!\nRuijie#\n"
	lines, _, _ := ifaceConfigBlock(raw, "GigabitEthernet 1/1")
	if !reflect.DeepEqual(lines, []string{"description 3D001"}) {
		t.Fatalf("prompt 行未被过滤: %v", lines)
	}
}

// TestBuildActionCmdsKeepsInterfacePrefix 动作命令组装必须保留 interface 前缀并追加 exit。
func TestBuildActionCmdsKeepsInterfacePrefix(t *testing.T) {
	got := buildActionCmds("GigabitEthernet 1/1", []string{
		"interface GigabitEthernet 1/1",
		"switchport port-security",
	})
	want := []string{
		"interface GigabitEthernet 1/1",
		"switchport port-security",
		"exit",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("动作命令组装不符:\n got=%v\nwant=%v", got, want)
	}
	fmt.Fprint(io_Discard, got)
}

// io_Discard 避免引入 io 包的小工具（仅用于上面的输出抑制）。
type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

var io_Discard = discardWriter{}
