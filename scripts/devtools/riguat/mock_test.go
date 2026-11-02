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
//
// ★ mock 的边界（2026-11-02 实跑后明确）：mock 只模拟**命令语义**
//
//	（设备是否接受该命令、该命令是否落地配置行），**不模拟设备的自动改写行为** ——
//	真机事实：下发 untagged 3992,3993 后，设备自行把配置改写成
//	  switchport hybrid allowed vlan only tagged 2-3991,3994-4094
//	  switchport hybrid allowed vlan add untagged 1,3992-3993
//	（自动补集 + 区间形式 + 自动并入 PVID）。那是**设备侧行为**，由真机取证覆盖；
//	mock 若去模拟它，反而会让"产品下发形态"与"设备落盘形态"在自测里混为一谈。
//	⇒ 断言一律针对**产品下发的形态**（如 "switchport hybrid allowed vlan untagged ..."）。
type mockRuijie struct {
	lines    []string        // 当前接口配置行
	inIface  bool            // 是否处于接口视图
	failCmds map[string]bool // 指定命令永久失效（模拟设备不接受）
	failOnce map[string]bool // 指定命令首次失效（模拟偶发失败 → 触发自愈重试）
	rejected []string        // 被拒绝的命令记录
	// rejectOutput 命令 → 自定义拒绝回显（复刻真机的业务级错误串）
	rejectOutput map[string]string
}

func newMockRuijie(baseline []string) *mockRuijie {
	return &mockRuijie{
		lines:        append([]string{}, baseline...),
		failCmds:     map[string]bool{},
		failOnce:     map[string]bool{},
		rejectOutput: map[string]string{},
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
	if m.rejectsCommand(c) {
		m.rejected = append(m.rejected, c)
		res.Failed = true
		res.Output = "% Invalid input detected at '^' marker."
		if custom, ok := m.rejectOutput[c]; ok {
			res.Output = custom
		}
		return res
	}
	m.applyConfig(c)
	return res
}

// rejectsCommand 判断该命令是否会被真机设备拒绝（模拟锐捷 RGOS 行为）。
//
// ★ 缺 tagged / untagged 关键字的 hybrid allowed 形态：真机
//
//	`switchport hybrid allowed vlan ?` → add/only/remove/tagged/untagged
//	⇒ 不带关键字的 `switchport hybrid allowed vlan <list>` 会被设备拒绝。
//	mock 必须复现这条，否则渲染层回归（少关键字）不会被自测发现。
func (m *mockRuijie) rejectsCommand(c string) bool {
	lower := strings.ToLower(strings.TrimSpace(c))
	// ★ 真机事实（2026-11-02 第三次取证，事实④）：tagged 与 untagged 形态**不对称** ——
	//   switchport hybrid allowed vlan only untagged 3992 → ❌ % Invalid input detected
	//   switchport hybrid allowed vlan untagged 3992      → ✅ 生效
	// ⇒ only untagged 必须被拒绝；untagged 不带 only 才是合法形态。
	if strings.HasPrefix(lower, "switchport hybrid allowed vlan only untagged ") {
		return true
	}
	// 关键字完全不对（既无 tagged/untagged 也无 only tagged）→ 设备拒绝
	if strings.HasPrefix(lower, "switchport hybrid allowed vlan ") &&
		!strings.HasPrefix(lower, "switchport hybrid allowed vlan tagged ") &&
		!strings.HasPrefix(lower, "switchport hybrid allowed vlan untagged ") &&
		!strings.HasPrefix(lower, "switchport hybrid allowed vlan only tagged ") {
		return true
	}
	// ★ 真机事实（2026-11-02 第三次取证）：锐捷**不接受**逐条否定 hybrid 配置
	//   （no switchport hybrid allowed vlan tagged 3990 → % Invalid input detected）
	//   ⇒ 唯一可靠的还原是模式回退 `switchport mode access`。
	if strings.HasPrefix(lower, "no switchport hybrid ") {
		return true
	}
	// ★ 真机事实（2026-11-02 实跑，RS8607E GE1/25）：端口配了 802.1X 时，
	//   设备不允许把口切到 hybrid 模式（设备级互斥）：
	//     Can not operate on interface Gi1/25. Conflict with IEEE 802.1X.
	if strings.HasPrefix(lower, "switchport mode hybrid") && m.hasDot1x() {
		return true
	}
	return false
}

// hasDot1x 端口当前是否配了 802.1X（dot1x port-control auto）。
func (m *mockRuijie) hasDot1x() bool {
	for _, l := range m.lines {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(l)), "dot1x port-control auto") {
			return true
		}
	}
	return false
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
	case lower == "no switchport mode hybrid":
		// 兜底路径：否定 hybrid 模式 → 清掉 hybrid 配置行与 mode 行
		m.removePrefix("switchport hybrid ")
		m.removePrefix("switchport mode hybrid")
	case lower == "default switchport mode access":
		// ★ 兜底写法（Cisco 风格）：切回 access → 清 hybrid 行，access 是默认模式不落 mode 行
		m.removePrefix("switchport hybrid ")
		m.removePrefix("switchport mode hybrid")
	case strings.HasPrefix(lower, "switchport mode "):
		// ★ 锐捷真实行为：access 是默认模式，running-config 里不落 mode 行
		//   （权威 9 行基线里确实没有 mode 行）→ 这里同样不落行。
		if strings.HasPrefix(lower, "switchport mode access") {
			// 切回 access ⇒ 清掉 hybrid 相关行（真机行为：换模式会清 hybrid 配置）
			m.removePrefix("switchport hybrid ")
			m.removePrefix("switchport mode hybrid")
			return
		}
		m.set(c)
	// ---- hybrid（锐捷 TrunkSetHybrid，PR #44 形态）----
	case strings.HasPrefix(lower, "no switchport hybrid allowed vlan tagged "):
		m.removePrefix("switchport hybrid allowed vlan tagged ")
	case strings.HasPrefix(lower, "no switchport hybrid allowed vlan untagged "):
		m.removePrefix("switchport hybrid allowed vlan untagged ")
	case strings.HasPrefix(lower, "no switchport hybrid native vlan "):
		m.removePrefix("switchport hybrid native vlan ")
	case strings.HasPrefix(lower, "switchport hybrid allowed vlan only tagged "):
		// ★ tagged 的正确形态（带 only）→ 落地
		m.set(c)
	case strings.HasPrefix(lower, "switchport hybrid allowed vlan untagged "):
		// ★ untagged 的正确形态（**不带** only）→ 落地
		m.set(c)
	case strings.HasPrefix(lower, "switchport hybrid allowed vlan tagged "):
		// ★ tagged 缺 only → 设备**无错误但不落地**（配置不变）→ 这里不写配置行
		return
	case strings.HasPrefix(lower, "switchport hybrid native vlan "):
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
		case f[1] == "hybrid" && len(f) >= 5 && f[2] == "allowed" && f[3] == "vlan":
			// switchport hybrid allowed vlan [only] tagged/untagged <list> 各是不同行
			key := f[0] + " " + f[1] + " " + f[2] + " " + f[3] + " " + f[4]
			if f[4] == "only" && len(f) >= 6 {
				key += " " + f[5]
			}
			return key
		case f[1] == "hybrid" && len(f) >= 4 && f[2] == "native":
			return f[0] + " " + f[1] + " " + f[2]
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

// ====================== hybrid 场景（锐捷 TrunkSetHybrid，PR #44 形态） ======================

// idleBaseline 空闲口的基线：锐捷默认 access 模式**不落** mode 行，
// 空闲口也没有 description / access vlan 行（lead 选定的 GE1/25 就是这种口）。
var idleBaseline = []string{}

// hybridParamsForTest hybrid 动作的测试参数。
//
// ★ untagged 给**两个** VLAN：PR #44 撤销了「多 untagged 拒绝」，
//
//	这里锁定「多 untagged 能被渲染且下发被设备接受」。
func hybridParamsForTest() actionParams {
	p := actionParamsForTest()
	p.BaselineVLAN = 1
	// ★ 锐捷 hybrid 是自动补集语义（2026-11-02 真机 + 补验，产品口径 (i)）：
	//   只下发 untagged，**不下发 tagged**（tagged 在锐捷无设备侧对应物，
	//   提交 tagged 会被产品层以 ErrActionNotSupported 拒绝）。
	p.HybridTagged = nil
	p.HybridUntagged = []int{3991, 3992}
	p.HybridNative = 3993
	return p
}

// TestHybridRestoreNoDevice hybrid 全流程还原（无设备 mock）。
//
// 断言：
//  1. 还原后与基线逐行一致（diff 为空）；
//  2. ★ 基线**无** `switchport access vlan` 行 ⇒ 还原**不得**下发该行
//     （lead 点名的坑：臆断下发会凭空多出一行 → 制造 diff → 误判还原失败）；
//  3. 还原必须含端口模式回退 `switchport mode access`；
//  4. 还原必须含三条 no hybrid（tagged / untagged / native）。
func TestHybridRestoreNoDevice(t *testing.T) {
	params := hybridParamsForTest()
	spec := actionRegistry["hybrid"]

	tpl := spec.Build(params)
	tpl.InterfaceName = params.CliIface
	rendered, err := portcollection.RenderCommand(models.VendorRuijie, spec.PortAction, tpl)
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}

	m := newMockRuijie(idleBaseline)
	for _, seg := range actionSegments(params.CliIface, rendered) {
		m.sendInterfaceCmds(seg)
	}
	if len(m.rejected) > 0 {
		t.Fatalf("下发被设备拒绝: %v", m.rejected)
	}
	if len(m.lines) == 0 {
		t.Fatalf("下发后 mock 没有任何配置行（hybrid 形态未生效）")
	}

	outcome, err := restoreWithVerify(m, params.CliIface, spec, params, idleBaseline,
		func() ([]string, error) { return append([]string{}, m.lines...), nil },
		3, func(format string, args ...interface{}) {})
	if err != nil {
		t.Fatalf("还原引擎报错: %v", err)
	}
	if len(m.rejected) > 0 {
		t.Errorf("还原过程有命令被设备拒绝: %v", m.rejected)
	}
	if !outcome.OK {
		t.Errorf("还原未通过: diff=%v 最终=%v", outcome.Diff, m.lines)
	}
	if d := diffConfig(idleBaseline, m.lines); len(d) != 0 {
		t.Errorf("还原后与基线不一致: %v（最终状态 %v）", d, m.lines)
	}

	var hasModeBack bool
	for _, c := range outcome.Cmds {
		cmd := strings.ToLower(strings.TrimSpace(c.Command))
		switch {
		case cmd == "switchport mode access" || cmd == "default switchport mode access":
			hasModeBack = true
		case strings.HasPrefix(cmd, "no switchport hybrid"):
			// ★ 真机事实：锐捷**不接受**逐条否定 hybrid 配置（% Invalid input）
			//   ⇒ 还原只走模式回退；下发逐条 no = 还原失败 = 口留在 hybrid 模式
			//   = 转发行为改变 = 潜在业务中断（安全级）
			t.Errorf("还原不得下发逐条 `no switchport hybrid ...`（锐捷不认，必须走模式回退）: %q", c.Command)
		case strings.Contains(cmd, "switchport access vlan"):
			// ★ 基线无 access vlan 行时，还原不得下发该行
			t.Errorf("基线无 access vlan 行时还原不得下发该行: %q", c.Command)
		}
	}
	if !hasModeBack {
		t.Errorf("还原必须含端口模式回退（switchport mode access / default ...）: %v", outcome.Cmds)
	}
}

// TestHybridMultiUntaggedAccepted 多 untagged 被接受（PR #44 撤销拒绝规则的行为验证）。
//
// 渲染层：两个 untagged 必须渲成一条 `... untagged 3991,3992`（不是被拒绝、也不是拆两条）。
// mock 层：该形态被设备接受（不被 reject）。
func TestHybridMultiUntaggedAccepted(t *testing.T) {
	params := hybridParamsForTest()
	spec := actionRegistry["hybrid"]
	tpl := spec.Build(params)
	tpl.InterfaceName = params.CliIface
	rendered, err := portcollection.RenderCommand(models.VendorRuijie, spec.PortAction, tpl)
	if err != nil {
		t.Fatalf("多 untagged 必须被接受（PR #44 已撤销拒绝规则）: %v", err)
	}
	joined := strings.Join(rendered, " | ")
	if !strings.Contains(joined, "switchport hybrid allowed vlan untagged 3991,3992") {
		t.Errorf("多 untagged 渲染形态不对（★ untagged 不得带 only）: %s", joined)
	}
	if strings.Contains(joined, "allowed vlan only untagged") {
		t.Errorf("untagged 不得带 only（真机回显 Invalid input）: %s", joined)
	}
	// ★ 产品口径 (i)：锐捷侧不下发 tagged（自动补集语义，tagged 无设备侧对应物）
	if strings.Contains(joined, "allowed vlan only tagged") || strings.Contains(joined, "allowed vlan tagged") {
		t.Errorf("锐捷侧不得下发 tagged 命令: %s", joined)
	}
	if !strings.Contains(joined, "switchport hybrid native vlan 3993") {
		t.Errorf("native 渲染形态不对: %s", joined)
	}

	m := newMockRuijie(idleBaseline)
	for _, seg := range actionSegments(params.CliIface, rendered) {
		m.sendInterfaceCmds(seg)
	}
	if len(m.rejected) > 0 {
		t.Errorf("多 untagged 形态被 mock 设备拒绝: %v", m.rejected)
	}
}

// TestHybridRejectsMissingKeyword mock 复现真机行为：缺 tagged/untagged 关键字的
// `switchport hybrid allowed vlan <list>` 会被设备拒绝。
//
// 这条自测守住 PR #44 的核心修正：渲染层一旦退回无关键字形态，mock 立刻能发现。
func TestHybridRejectsMissingKeyword(t *testing.T) {
	m := newMockRuijie(idleBaseline)
	m.apply("interface GigabitEthernet 1/1") // 进接口视图（否则顶层命令一律被拒）
	res := m.apply("switchport hybrid allowed vlan 3990")
	if !res.Failed {
		t.Errorf("缺 tagged/untagged 关键字的 hybrid allowed 形态必须被设备拒绝")
	}
	ok := m.apply("switchport hybrid allowed vlan tagged 3990")
	if ok.Failed {
		t.Errorf("带 tagged 关键字的形态必须被接受: %s", ok.Output)
	}
}

// TestHybridModeAccessFallback 端口模式回退兜底：
// `switchport mode access` 被设备拒绝时，改用 `default switchport mode access`，
// 且最终仍能收敛到基线（不留 hybrid 行）。
func TestHybridModeAccessFallback(t *testing.T) {
	params := hybridParamsForTest()
	spec := actionRegistry["hybrid"]
	tpl := spec.Build(params)
	tpl.InterfaceName = params.CliIface
	rendered, err := portcollection.RenderCommand(models.VendorRuijie, spec.PortAction, tpl)
	if err != nil {
		t.Fatalf("渲染失败: %v", err)
	}

	m := newMockRuijie(idleBaseline)
	for _, seg := range actionSegments(params.CliIface, rendered) {
		m.sendInterfaceCmds(seg)
	}
	// 模拟设备不接受 `switchport mode access`（首次即失败 → 触发兜底）
	m.failOnce["switchport mode access"] = true

	outcome, err := restoreWithVerify(m, params.CliIface, spec, params, idleBaseline,
		func() ([]string, error) { return append([]string{}, m.lines...), nil },
		3, func(format string, args ...interface{}) {})
	if err != nil {
		t.Fatalf("还原引擎报错: %v", err)
	}
	if !outcome.OK {
		t.Errorf("兜底后仍未收敛: diff=%v 最终=%v", outcome.Diff, m.lines)
	}
	var usedFallback bool
	for _, c := range outcome.Cmds {
		if strings.EqualFold(strings.TrimSpace(c.Command), "default switchport mode access") {
			usedFallback = true
		}
	}
	if !usedFallback {
		t.Errorf("未触发 `default switchport mode access` 兜底: %v", outcome.Cmds)
	}
}

// ====================== 真机事实复刻：802.1X × hybrid 互斥 + 假 OK 缺陷 ======================

// TestHybridBlockedByDot1x 真机事实复刻（2026-11-02 实跑，RS8607E GE1/25）：
//
//	interface GigabitEthernet 1/25
//	 switchport mode hybrid
//	 Can not operate on interface Gi1/25. Conflict with IEEE 802.1X.
//
// ⇒ 端口配了 802.1X（dot1x port-control auto）时，设备不允许切 hybrid 模式。
// 本用例同时锁住 lead 报出的**假 OK 缺陷**：工具的 hasDeviceError 必须把这条
// 自然语言错误判为失败（否则实跑会把失败命令报成 [OK]）。
func TestHybridBlockedByDot1x(t *testing.T) {
	const realErr = "Can not operate on interface Gi1/25. Conflict with IEEE 802.1X."

	m := newMockRuijie([]string{
		"dot1x port-control auto",
		"dot1x mac-auth-bypass multi-user",
	})
	m.rejectOutput["switchport mode hybrid"] = realErr

	m.apply("interface GigabitEthernet 1/1")
	res := m.apply("switchport mode hybrid")
	if !res.Failed {
		t.Fatalf("配了 dot1x 的端口切 hybrid 必须被设备拒绝（真机事实）")
	}
	if !strings.Contains(res.Output, "Conflict with IEEE 802.1X") {
		t.Errorf("拒绝回显应含真机错误串: %q", res.Output)
	}
	// ★ 工具判定必须识别这条业务级错误（lead 报的假 OK 缺陷）
	if !hasDeviceError(res.Output) {
		t.Errorf("hasDeviceError 必须识别锐捷业务级错误（不含百分号前缀）: %q", res.Output)
	}
	// 正常回显不能误判
	if hasDeviceError("switchport mode hybrid\nRuijie(config-if-GigabitEthernet 1/25)#") {
		t.Errorf("正常回显不得判为失败")
	}
}

// TestHasDeviceError 设备侧错误判定的单元测试（2026-11-02 真机实跑修正）。
func TestHasDeviceError(t *testing.T) {
	cases := []struct {
		name string
		out  string
		want bool
	}{
		{"锐捷业务错误：dot1x 与 hybrid 互斥",
			"switchport mode hybrid\nCan not operate on interface Gi1/25. Conflict with IEEE 802.1X.", true},
		{"锐捷百分号错误（行首）",
			"% Invalid input detected at '^' marker.", true},
		{"多行回显中的行首百分号",
			"switchport mode hybrid\n% Unknown command\nRuijie(config-if-GigabitEthernet 1/25)#", true},
		{"思科风格 Unknown",
			"Unrecognized command found at '^' position.", true},
		{"正常回显",
			"switchport mode hybrid\nRuijie(config-if-GigabitEthernet 1/25)#", false},
		{"正常配置行",
			"switchport access vlan 318", false},
	}
	for _, c := range cases {
		if got := hasDeviceError(c.out); got != c.want {
			t.Errorf("[%s] hasDeviceError = %v, want %v（out=%q）", c.name, got, c.want, c.out)
		}
	}
}

// TestHybridOnlyKeywordRequired 真机事实（2026-11-02 第三次取证）：
//
//	switchport hybrid allowed vlan tagged 3990     → 无错误但【不落地】
//	switchport hybrid allowed vlan add tagged 3990 → 无错误但【不落地】
//	switchport hybrid allowed vlan only tagged 3990 → ✅【落地】
//
// ⇒ 缺 only 是**静默失败**（命令成功、配置不变、用户以为成功）—— 最阴险的一类缺陷。
// 本用例锁住：带 only 才落地；缺 only 时设备不报错但配置不变。
func TestHybridOnlyKeywordRequired(t *testing.T) {
	// 缺 only → 无错误、但不落地
	m := newMockRuijie(idleBaseline)
	m.apply("interface GigabitEthernet 1/1")
	res := m.apply("switchport hybrid allowed vlan tagged 3990")
	if res.Failed {
		t.Errorf("缺 only 的形态在真机上不报错（只是不落地）: %s", res.Output)
	}
	if len(m.lines) != 0 {
		t.Errorf("缺 only 的形态不得落地配置行: %v", m.lines)
	}

	// 带 only → 落地
	m2 := newMockRuijie(idleBaseline)
	m2.apply("interface GigabitEthernet 1/1")
	res2 := m2.apply("switchport hybrid allowed vlan only tagged 3990")
	if res2.Failed {
		t.Fatalf("带 only 的形态必须被接受: %s", res2.Output)
	}
	if len(m2.lines) != 1 || !strings.Contains(m2.lines[0], "switchport hybrid allowed vlan only tagged 3990") {
		t.Errorf("带 only 的形态必须落地配置行: %v", m2.lines)
	}

	// ★ untagged 形态**不带** only（真机事实④：only untagged 被设备拒绝）
	m3 := newMockRuijie(idleBaseline)
	m3.apply("interface GigabitEthernet 1/1")
	if r := m3.apply("switchport hybrid allowed vlan only untagged 3992"); !r.Failed {
		t.Errorf("only untagged 必须被设备拒绝（真机事实④）")
	}
	m4 := newMockRuijie(idleBaseline)
	m4.apply("interface GigabitEthernet 1/1")
	r4 := m4.apply("switchport hybrid allowed vlan untagged 3992")
	if r4.Failed {
		t.Fatalf("untagged（不带 only）必须被接受: %s", r4.Output)
	}
	// ★ 精确整行断言（lead 2026-11-02 建议）：子串 "untagged 3992" 会同时匹配
	//   "only untagged 3992" ⇒ 用完整命令形态断言，渲染层改动才能精确反映到断言上。
	if len(m4.lines) != 1 || !strings.Contains(m4.lines[0], "switchport hybrid allowed vlan untagged 3992") {
		t.Errorf("untagged（不带 only）必须落地配置行: %v", m4.lines)
	}
}

// TestHybridPerItemNoRejected 真机事实：锐捷**不接受**逐条否定 hybrid 配置。
//
//	no switchport hybrid allowed vlan tagged 3990 → % Invalid input detected
//	no switchport hybrid native vlan 3993        → % Invalid input detected
//
// ⇒ 还原必须走模式回退 `switchport mode access`（一次性清空全部 hybrid 配置）。
func TestHybridPerItemNoRejected(t *testing.T) {
	m := newMockRuijie([]string{"switchport mode hybrid", "switchport hybrid allowed vlan only tagged 3990"})
	m.apply("interface GigabitEthernet 1/1")
	for _, c := range []string{
		"no switchport hybrid allowed vlan tagged 3990",
		"no switchport hybrid native vlan 3993",
	} {
		res := m.apply(c)
		if !res.Failed {
			t.Errorf("逐条否定 hybrid 配置必须被设备拒绝（真机事实）: %q", c)
		}
	}
	// 模式回退可用：一次性清空全部 hybrid 配置
	res := m.apply("switchport mode access")
	if res.Failed {
		t.Fatalf("模式回退必须被设备接受: %s", res.Output)
	}
	if len(m.lines) != 0 {
		t.Errorf("模式回退必须一次性清空全部 hybrid 配置: %v", m.lines)
	}
}
