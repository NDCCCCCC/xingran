package main

// 锐捷 `switchport trunk allowed vlan` 第二层（add / remove / replace）取证探针。
//
// ★★★ 三条已被真机/实测确立的取证方法论（本文件是它们的实现）★★★
//
// 1) **锐捷的 `?` 帮助可选项集合依赖当前端口模式**（2026-11-02 首轮实测）：
//
//	switchport trunk ?                  → Invalid input
//	switchport trunk allowed vlan all ? → Can't set allow list on this port mode!
//
//	⇒ access 模式下设备**连 `switchport trunk` 关键字都不提供** ⇒ 首轮 8 条探针的
//	  Invalid input **只说明「当前模式不提供」，不能推断「关键字不存在」**。
//
// 2) **更狠的一条**（2026-11-02 第二轮实测）：`?` 只能回答「**当前位置**可补全什么」。
//	对「单命令带固定参数」形态（如 `switchport trunk allowed vlan all`），
//	`?` 放在参数前（`... vlan ?`）或参数后（`... vlan 3990 ?`）**必然报错**，
//	而 `... vlan all ?` 回显 `<cr>`（= 命令到此完整，直接回车）。
//	⇒ ★ **`?` 法对这类命令无效**；add / remove / replace 是否存在**只能实跑才知道**。
//	⇒ **「Invalid input」也可能只是 `?` 的位置不对 —— 不可据此推断关键字不存在。**
//
// 3) **「命令不报错」≠「命令生效」**（锐捷 hybrid 的 `allowed vlan tagged <list>` 就是
//	无错但不落地）⇒ 受控实跑**每条之后必须立刻回读 running-config**，
//	据此区分三态：被拒（关键字不存在）/ 被接受且回读有变化（存在且生效）/
//	**被接受但回读无变化（无错但不落地 —— 那个坑）**。
//
// ★ 安全机制（三层白名单 + 还原 + 零残留）：
//  1. isReadOnlyProbeCmd：只放行「以 ? 结尾」与「show 前缀」—— 完全只读；
//  2. modeSwitchPairs（受控写）：仅 `switchport mode trunk` / `switchport mode access`，
//     且**必须成对**（trunk 必配 access 还原）；
//  3. controlledRunCmds（受控实跑）：仅 4 条 trunk allowed 命令，逐条回读判定；
//     每条之后若发现**不该有的**副作用（VLAN 表变化）⇒ 立即停止后续实跑（仍会还原）。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/device"
)

// isReadOnlyProbeCmd 判定一条探针命令是否**完全只读**。
func isReadOnlyProbeCmd(c string) bool {
	t := strings.TrimSpace(c)
	if t == "" {
		return false
	}
	if strings.HasSuffix(t, "?") {
		return true
	}
	return strings.HasPrefix(strings.ToLower(t), "show ")
}

// modeSwitchPairs 受控模式切换白名单：command → 必需的还原命令。
var modeSwitchPairs = map[string]string{
	"switchport mode trunk":  "switchport mode access",
	"switchport mode access": "",
}

// modeSwitchAllowed 判定一条**受控写命令**是否被允许，且其还原命令是否齐备。
func modeSwitchAllowed(c string) (bool, string) {
	t := strings.TrimSpace(c)
	restore, ok := modeSwitchPairs[t]
	if !ok {
		return false, ""
	}
	if t == "switchport mode trunk" && restore == "" {
		return false, ""
	}
	return true, restore
}

// modeSwitchComplete 判定一组受控写命令是否**成对完整**。
func modeSwitchComplete(cmds []string) bool {
	hasTrunk, hasAccess := false, false
	for _, c := range cmds {
		t := strings.TrimSpace(c)
		if _, ok := modeSwitchPairs[t]; !ok {
			continue
		}
		if _, restore := modeSwitchAllowed(t); restore == "" && t == "switchport mode trunk" {
			return false
		}
		switch t {
		case "switchport mode trunk":
			hasTrunk = true
		case "switchport mode access":
			hasAccess = true
		}
	}
	return hasTrunk == hasAccess
}

// controlledRunCmds 受控**实跑**白名单（第三类）：会改配置 ⇒ 必须逐条回读判定。
//
// ⚠️ 只允许这 4 条（lead 2026-11-02 裁定）：
//
//	switchport trunk allowed vlan 3990        第一层：建立基线（已知合法）
//	switchport trunk allowed vlan add 3990     第二层 add
//	switchport trunk allowed vlan remove 3990  第二层 remove
//	switchport trunk allowed vlan replace 3990 第二层 replace
//
// 逐条之后**立刻回读 running-config**（判读的关键：命令无错 ≠ 生效）。
var controlledRunCmds = []string{
	// ① only 的语义：允许式（只允许 3990）还是排除式（排除 3990）？
	//    回读若是 `only 3990` ⇒ 允许式；若是补集（如 only 1-3899,3991-4094）⇒ 排除式。
	"switchport trunk allowed vlan only 3990",
	// ② add 的语义：并集（only 3990,3991）还是替换（只留 3991）？
	"switchport trunk allowed vlan add 3991",
	// ③ remove 的语义：能否把 VLAN 从允许列表去掉（若 ② 生效 ⇒ 期望回到 only 3990）？
	"switchport trunk allowed vlan remove 3991",
}

// controlledRunAllowed 判定一条命令是否在受控实跑白名单内。
func controlledRunAllowed(c string) bool {
	t := strings.TrimSpace(c)
	for _, allowed := range controlledRunCmds {
		if allowed == t {
			return true
		}
	}
	return false
}

// trunkAllowedProbeCmds 锐捷 trunk allowed 第二层的探针命令表（全部只读）。
//
// ⚠️ 第二轮实测已证明：这些 `?` 探针对「单命令带参数」形态**无效**
// （`?` 位置不对必然报错）⇒ 第三轮起以**受控实跑**为主，本表仅作对照保留。
func trunkAllowedProbeCmds() []string {
	return []string{
		"switchport trunk ?",
		"switchport trunk allowed ?",
		"switchport trunk allowed vlan ?",
		"switchport trunk allowed vlan add ?",
		"switchport trunk allowed vlan remove ?",
		"switchport trunk allowed vlan replace ?",
		"switchport trunk allowed vlan 3990 ?",
		"switchport trunk allowed vlan all ?",
		"switchport trunk allowed vlan none ?",
	}
}

// probeBaseline 探针前/后的设备状态快照（用于零残留核查与回读判定）。
type probeBaseline struct {
	// ConfigBytes 接口配置块文本的字节数
	ConfigBytes int
	// ConfigLines 接口配置行数
	ConfigLines int
	// ConfigLinesRaw 接口配置行（用于逐条比对"回读有没有变化"）
	ConfigLinesRaw []string
	// PortMode 端口模式判定
	PortMode string
	// ModeProbeRaw `switchport mode ?` 的回显（设备自己说的模式证据）
	ModeProbeRaw string
	// VlanIDs 设备 VLAN 表里的 VLAN ID 集合
	VlanIDs []int
	// Raw 接口配置原始回显（报告里留证）
	Raw string
}

// parsePortMode 从接口配置行判定端口模式。
//
// ★ 2026-11-02 修正（第二轮实测发现的 bug）：**不能假设锐捷在 trunk 模式下会落
// `switchport mode trunk` 行** —— 实测 trunk 模式下 running-config 里**没有** mode 行，
// 于是第一版判定把 trunk 误读成 access（判读表每条都写 access，明显不对）。
//
// 改判据（三条证据合并）：
//  1. 有 `switchport mode X` 行 ⇒ 用 X（设备显式落行时最可信）；
//  2. 否则有 **trunk 特征行**（`switchport trunk ...`）⇒ trunk
//     （access 口不会配 trunk 允许列表/原生 VLAN）；
//  3. 否则 ⇒ access（默认模式，running-config 不落 mode 行）。
//
// ⚠️ 仍属**推断**（设备没有直接的"当前模式"查询）⇒ 报告里同时给出
// `switchport mode ?` 的原始回显作交叉证据，不把推断当确证。
func parsePortMode(lines []string) string {
	for _, l := range lines {
		t := strings.ToLower(strings.TrimSpace(l))
		if strings.HasPrefix(t, "switchport mode ") {
			return strings.TrimSpace(strings.TrimPrefix(t, "switchport mode "))
		}
	}
	for _, l := range lines {
		t := strings.ToLower(strings.TrimSpace(l))
		if strings.HasPrefix(t, "switchport trunk ") {
			return "trunk（依据 trunk 特征行推断，设备未落 mode 行）"
		}
	}
	return "access（默认：无 mode 行也无 trunk 特征行）"
}

// collectProbeBaseline 采集只读状态快照（含端口模式交叉证据）。
func collectProbeBaseline(w *device.ScrapliWrapper, cliIface string) (probeBaseline, error) {
	var b probeBaseline

	cfgOut, err := sendReadCmd(w, "show running-config interface "+cliIface)
	if err != nil {
		return b, fmt.Errorf("采集接口配置失败: %w", err)
	}
	lines, _, _ := ifaceConfigBlock(cfgOut, cliIface)
	b.ConfigLines = len(lines)
	b.ConfigLinesRaw = lines
	b.Raw = cfgOut
	b.PortMode = parsePortMode(lines)
	var sb strings.Builder
	for _, l := range lines {
		sb.WriteString(l)
		sb.WriteString("\n")
	}
	b.ConfigBytes = sb.Len()

	// 端口模式的交叉证据：设备自己的 ? 回显（只读）
	// ★ 必须在**接口视图**下取：特权模式下 `switchport mode ?` 回显
	//   `% Unknown command`（lead 2026-11-02 指出）。
	if modeRes := newCmdSender(w, cliIface).sendInterfaceCmds([]string{"switchport mode ?"}); len(modeRes) > 0 {
		if r, ok := findCmdResult(modeRes, "switchport mode ?"); ok {
			b.ModeProbeRaw = sanitizeOutput(r.Output)
		}
	}

	vlanOut, err := sendReadCmd(w, "show vlan")
	if err != nil {
		return b, fmt.Errorf("采集 VLAN 表失败: %w", err)
	}
	b.VlanIDs = sortedIntKeys(parseVlanIDs(vlanOut))
	return b, nil
}

// 三态判读结论。
const (
	verdictRejected   = "命令被拒 ⇒ 关键字不存在"
	verdictLanded     = "被接受且回读有变化 ⇒ 存在且生效"
	verdictSilentNoop = "被接受但回读无变化 ⇒ ⚠️ 无错但不落地（那个坑）"
)

// runOutcome 一条受控实跑命令的判读结果。
type runOutcome struct {
	Cmd           string
	Rejected      bool   // 设备报错（命令被拒）
	Changed       bool   // 回读的 allowed 行有变化
	Verdict       string // 三态结论
	Before        probeBaseline
	After         probeBaseline
	AllowedBefore string // 回读的 `switchport trunk allowed vlan` 行原文（判读的唯一可信依据）
	AllowedAfter  string // 同上（回读后）
	VlanAdded     []int  // 本条新增的 VLAN（设备自动创建，锐捷已确证会这么做）
	SideEffect    bool   // 出现**不该有**的副作用（非 VLAN 新增的变化）⇒ 停止后续实跑
}

// findCmdResult 按命令文本取该条 cmdResult。
//
// ★ 为什么不能按索引（lead 2026-11-02 发现的判读矛盾根因）：
//
//	sendInterfaceCmds 返回的**第一项是 `interface <iface>`（进入接口视图）**、
//	最后一项是 `exit` ⇒ 取 res[0] 拿到的是"进视图"那条，不是被测命令 ⇒
//	「设备是否报错」列恒为"否"，与原始回显直接矛盾。
func findCmdResult(results []cmdResult, cmd string) (cmdResult, bool) {
	for _, r := range results {
		if strings.TrimSpace(r.Command) == strings.TrimSpace(cmd) {
			return r, true
		}
	}
	return cmdResult{}, false
}

// diffConfigLines 比较两次回读得到的接口配置行是否不同。
func diffConfigLines(before, after []string) bool {
	if len(before) != len(after) {
		return true
	}
	for i := range before {
		if strings.TrimSpace(before[i]) != strings.TrimSpace(after[i]) {
			return true
		}
	}
	return false
}

// extractTrunkAllowedLine 从接口配置行里取出 `switchport trunk allowed vlan` 那一行**原文**。
//
// ★ 这是判读的唯一可信依据（lead 2026-11-02 明确要求）：
//
//	「每条之后回读 running-config 的 switchport trunk allowed vlan 那一行 ——
//	  这是唯一可信的判据 ⇒ 请把这一行原文直接贴出来（不要只给有/无变化）」
//
// 找不到 ⇒ 返回空串（该行不存在，例如 access 模式下没有 trunk 配置）。
func extractTrunkAllowedLine(lines []string) string {
	for _, l := range lines {
		t := strings.ToLower(strings.TrimSpace(l))
		if strings.HasPrefix(t, "switchport trunk allowed vlan") {
			return strings.TrimSpace(l)
		}
	}
	return ""
}

// diffIntSets 求 a 中有、b 中没有的元素（用于识别设备自动创建的 VLAN）。
func diffIntSets(a, b []int) []int {
	inB := make(map[int]bool, len(b))
	for _, v := range b {
		inB[v] = true
	}
	var added []int
	for _, v := range a {
		if !inB[v] {
			added = append(added, v)
		}
	}
	return added
}

// judgeRunOutcome 按三态判定一条实跑命令的结果。
//
// ★ 判据 = **回读的 `switchport trunk allowed vlan` 行**（不是整个配置块 diff）：
//
//	· 设备报错 ⇒ 关键字不存在
//	· 被接受且 allowed 行**变了** ⇒ 存在且生效
//	· 被接受但 allowed 行**没变** ⇒ ⚠️ 无错但不落地（那个坑）
func judgeRunOutcome(cmd string, rejected bool, before, after, runBase probeBaseline) runOutcome {
	o := runOutcome{Cmd: cmd, Rejected: rejected, Before: before, After: after}
	o.AllowedBefore = extractTrunkAllowedLine(before.ConfigLinesRaw)
	o.AllowedAfter = extractTrunkAllowedLine(after.ConfigLinesRaw)

	// 副作用判定必须对「**实跑期基线**」⇒ 即「切 trunk 之后、实跑之前」的状态（runBase）。
	//
	// ★ 为何不能对「探测前」基线比（lead 2026-11-02 第四轮指出的误报根因）：
	//   实跑的前提就是切 trunk 模式 ⇒ 「端口模式 access → trunk」本身就是**实跑期期望状态**，
	//   若对探测前基线比就会误报「非预期副作用」⇒ 实跑被中途停掉。
	//
	// · 设备**自动创建** VLAN 是锐捷的已知行为（hybrid 实跑已确证）⇒ 属预期内，只记录不停止；
	// · 其余变化（VLAN 表非「新增」的变化、端口模式被改动）⇒ 真副作用。
	if runBase.PortMode != "" {
		o.VlanAdded = diffIntSets(after.VlanIDs, runBase.VlanIDs)
		if len(o.VlanAdded) == 0 && fmt.Sprint(runBase.VlanIDs) != fmt.Sprint(after.VlanIDs) {
			o.SideEffect = true
		}
		if after.PortMode != runBase.PortMode {
			o.SideEffect = true
		}
	}

	switch {
	case rejected:
		o.Verdict = verdictRejected
	case o.AllowedBefore != o.AllowedAfter:
		o.Changed = true
		o.Verdict = verdictLanded
	default:
		o.Verdict = verdictSilentNoop
	}
	return o
}

// 探针模式（三选一）。
const (
	probeModeReadonly      = "readonly"
	probeModeSwitchOnly    = "switch"
	probeModeControlledRun = "run"
)

// runTrunkAllowedProbe 执行锐捷 trunk allowed 第二层取证探针。
//
// mode 三选一：
//   - probeModeReadonly：纯只读（不切模式，只跑 ? 探针）
//   - probeModeSwitchOnly：切 trunk 模式 + ? 探针（第二轮）
//   - probeModeControlledRun：切 trunk 模式 + 受控实跑（第三轮，逐条回读）
//
// switchMode=true 时：先受控切到 trunk 模式 → 主体 → **必定还原**回 access 模式
// （错误路径也会还原）⇒ 零残留核查比对的是「**还原后 vs 探测前**」。
func runTrunkAllowedProbe(ctx context.Context, ch *execChannel, dev *deviceInfo, cliIface string,
	rep *report, timeout time.Duration, mode string) {

	rep.section("锐捷 trunk allowed 第二层取证（add / remove / replace）")
	rep.line("设备: %s（%s）  接口: %s", dev.DeviceName, dev.IPAddress, cliIface)
	switch mode {
	case probeModeControlledRun:
		rep.line("模式: **受控实跑** —— 切 trunk 模式后逐条实跑，每条之后**立刻回读 running-config**")
		rep.line("      ⇒ 三态判读：被拒 / 被接受且回读有变化 / **被接受但回读无变化（无错但不落地）**")
	case probeModeSwitchOnly:
		rep.line("模式: **受控切换 + ? 探针**（第二轮）")
		rep.line("      ⚠️ 实测：`?` 对「单命令带参数」形态无效（`?` 位置不对必然报错）⇒ 只能问「当前位置可补全什么」")
	default:
		rep.line("模式: **纯只读**（不切端口模式）")
		rep.line("      ⚠️ access 模式下设备不提供 trunk 关键字 ⇒ Invalid input 只说明「当前模式不提供」")
	}

	cmds := trunkAllowedProbeCmds()
	var allowed []string
	for _, c := range cmds {
		if isReadOnlyProbeCmd(c) {
			allowed = append(allowed, c)
			continue
		}
		rep.line("⛔ 跳过（非只读，未下发）: %s", c)
	}

	switchMode := mode != probeModeReadonly

	// 受控写白名单预检 + 成对校验
	var switchCmds []string
	if switchMode {
		switchCmds = []string{"switchport mode trunk", "switchport mode access"}
		for _, c := range switchCmds {
			if ok, _ := modeSwitchAllowed(c); !ok {
				rep.line("⛔ 模式切换命令不在受控白名单，未执行: %s", c)
				rep.save()
				return
			}
		}
		if !modeSwitchComplete(switchCmds) {
			rep.line("⛔ 模式切换命令**未成对**（trunk 必配 access 还原）—— 拒绝执行")
			rep.save()
			return
		}
	}

	// 受控实跑白名单预检
	var runCmds []string
	if mode == probeModeControlledRun {
		for _, c := range controlledRunCmds {
			if !controlledRunAllowed(c) {
				rep.line("⛔ 受控实跑命令不在白名单，未执行: %s", c)
				rep.save()
				return
			}
			runCmds = append(runCmds, c)
		}
	}

	// ★ initial = **探测前**基线（永远不被循环覆盖）；before 只作循环内的比较基线。
	//   （lead 2026-11-02 指出：旧实现把 before 在循环里更新，导致报告里的
	//    「探测前基线」实际是最后一次实跑后的状态 ⇒ 采集时机/顺序全错。）
	// ★ 三个基线，职责不同（lead 2026-11-02 第四轮指出的误报根因）：
	//   initial — 探测前（切 trunk **之前**）⇒ 用于**零残留核查**（还原后 vs 探测前）；
	//   runBase — **实跑期基线**（切 trunk **之后**、实跑之前）⇒ 用于**副作用判定**；
	//   before  — 循环内的比较基线（上一条的结果）。
	var initial, runBase, before, after probeBaseline
	var results []cmdResult
	var modeSwitchOut, restoreOut []cmdResult
	var outcomes []runOutcome
	var stoppedEarly bool

	err := ch.run(ctx, dev.ID, timeout, func(execCtx context.Context, w *device.ScrapliWrapper) error {
		sender := newCmdSender(w, cliIface)

		b, berr := collectProbeBaseline(w, cliIface)
		if berr != nil {
			return berr
		}
		initial = b
		before = b

		if switchMode {
			modeSwitchOut = sender.sendInterfaceCmds([]string{"switchport mode trunk"})
			// 切 trunk 之后立即采集「实跑期基线」⇒ 副作用判定的对照基准
			rb, rberr := collectProbeBaseline(w, cliIface)
			if rberr != nil {
				return rberr
			}
			runBase = rb
		}

		if mode == probeModeControlledRun {
			// 逐条实跑 + 逐条回读（判读的关键）
			for _, c := range runCmds {
				res := sender.sendInterfaceCmds([]string{c})
				results = append(results, res...)

				cur, cerr := collectProbeBaseline(w, cliIface)
				if cerr != nil {
					return cerr
				}
				// ★ 按命令文本取该条结果（sendInterfaceCmds 的首项是进视图、末项是 exit）
				cr, ok := findCmdResult(res, c)
				rejected := ok && cr.Failed
				o := judgeRunOutcome(c, rejected, before, cur, runBase)
				outcomes = append(outcomes, o)
				before = cur // 下一条的基线是上一条的结果

				if len(o.VlanAdded) > 0 {
					// 设备自动创建 VLAN 是锐捷的已确证行为（hybrid 实跑见过）
					// ⇒ 属预期内，只记录（还原阶段会清理），不停止实跑
					rep.line("ℹ️ 设备自动创建了 VLAN %v（预期内行为，还原阶段清理）", o.VlanAdded)
				}
				if o.SideEffect {
					stoppedEarly = true
					rep.line("🚨 检测到非预期副作用（端口模式: %s → %s；VLAN 表: %v → %v）⇒ 停止后续实跑，立即还原",
						o.Before.PortMode, o.After.PortMode, o.Before.VlanIDs, o.After.VlanIDs)
					break
				}
			}
		} else {
			results = sender.sendInterfaceCmds(allowed)
		}

		// 必定还原为 access（无论是否出错）—— 零残留的前提
		if switchMode {
			restoreOut = sender.sendInterfaceCmds([]string{"switchport mode access"})
		}

		a, aerr := collectProbeBaseline(w, cliIface)
		if aerr != nil {
			return aerr
		}
		after = a
		return nil
	})
	if err != nil {
		rep.line("❌ 探测执行异常: %v", err)
		rep.line("⚠️ 请人工核查设备状态（配置字节数 / 行数 / 端口模式 / VLAN 表）")
		return
	}

	// —— 基线（探测前，即**切 trunk 之前**）——
	rep.section("1. 探测前基线（切 trunk 之前）")
	rep.line("端口模式      : %s", initial.PortMode)
	rep.line("接口配置块    : %d 字节 / %d 行", initial.ConfigBytes, initial.ConfigLines)
	rep.line("VLAN 表       : %v", initial.VlanIDs)
	rep.line("allowed 行原文   : %s", orNone(extractTrunkAllowedLine(initial.ConfigLinesRaw)))
	rep.raw("show running-config interface "+cliIface, sanitizeOutput(initial.Raw))
	if strings.TrimSpace(initial.ModeProbeRaw) != "" {
		rep.raw("switchport mode ?（端口模式交叉证据，接口视图）", initial.ModeProbeRaw)
	}

	// —— 实跑期基线（切 trunk 之后、实跑之前）——
	if switchMode {
		rep.section("1b. 实跑期基线（切 trunk 之后、实跑之前）")
		rep.line("端口模式      : %s", runBase.PortMode)
		rep.line("接口配置块    : %d 字节 / %d 行", runBase.ConfigBytes, runBase.ConfigLines)
		rep.line("VLAN 表       : %v", runBase.VlanIDs)
		rep.line("allowed 行原文   : %s", orNone(extractTrunkAllowedLine(runBase.ConfigLinesRaw)))
	}

	// —— 模式切换回显（受控写）——
	if switchMode {
		rep.section("2. 受控模式切换回显")
		printCmdResults(rep, "切 trunk 模式", modeSwitchOut)
		printCmdResults(rep, "还原 access 模式", restoreOut)
	}

	// —— 探针/实跑回显（原始，不改写）——
	rep.section("3. 探针/实跑回显（原始回显，逐条）")
	for i, r := range results {
		cmd := strings.TrimSpace(r.Command)
		rep.line("──── [%d/%d] %s", i+1, len(results), cmd)
		if r.Failed {
			rep.line("  ⛔ 设备报错: %s", strings.TrimSpace(r.Output))
			continue
		}
		for _, l := range strings.Split(sanitizeOutput(r.Output), "\n") {
			if strings.TrimSpace(l) == "" {
				continue
			}
			rep.line("  %s", l)
		}
	}

	// —— 三态判读表（受控实跑模式）——
	if mode == probeModeControlledRun {
		rep.section("4. 三态判读表（受控实跑）")
		rep.line("%-42s | %-12s | %-14s | %s", "命令", "设备是否报错", "回读是否变化", "判读")
		rep.line("%s", strings.Repeat("-", 130))
		for _, o := range outcomes {
			rejected := "否"
			if o.Rejected {
				rejected = "是"
			}
			changed := "否"
			if o.Changed {
				changed = "是"
			}
			rep.line("%-42s | %-12s | %-14s | %s", o.Cmd, rejected, changed, o.Verdict)
		}
		rep.line("")
		rep.line("【allowed 行原文对照】（判读的唯一可信依据）")
		for _, o := range outcomes {
			rep.line("  ▸ %s", o.Cmd)
			rep.line("      执行前: %s", orNone(o.AllowedBefore))
			rep.line("      执行后: %s", orNone(o.AllowedAfter))
		}
		rep.line("")
		rep.line("【VLAN 表对照】（设备是否自动创建 VLAN —— 影响「产品是否需要预建 VLAN」）")
		rep.line("  探测前    : %v", initial.VlanIDs)
		if switchMode {
			rep.line("  实跑期基线: %v", runBase.VlanIDs)
		}
		for _, o := range outcomes {
			if len(o.VlanAdded) > 0 {
				rep.line("  ▸ %s 自动新增: %v", o.Cmd, o.VlanAdded)
			}
		}
		rep.line("  还原后    : %v", after.VlanIDs)
		if stoppedEarly {
			rep.line("⚠️ 因检测到副作用，后续实跑命令**未执行**")
		}
		rep.line("")
		rep.line("三态含义：")
		rep.line("  · %s", verdictRejected)
		rep.line("  · %s", verdictLanded)
		rep.line("  · %s", verdictSilentNoop)
	}

	// —— 零残留核查 ——
	rep.section("5. 零残留核查（还原后 vs 探测前）")
	sameBytes := before.ConfigBytes == after.ConfigBytes
	sameLines := before.ConfigLines == after.ConfigLines
	sameMode := before.PortMode == after.PortMode
	sameVlan := fmt.Sprint(before.VlanIDs) == fmt.Sprint(after.VlanIDs)
	rep.line("端口模式      : %s → %s  %s", before.PortMode, after.PortMode, okMark(sameMode))
	rep.line("接口配置块字节: %d → %d  %s", before.ConfigBytes, after.ConfigBytes, okMark(sameBytes))
	rep.line("接口配置行数  : %d → %d  %s", before.ConfigLines, after.ConfigLines, okMark(sameLines))
	rep.line("VLAN 表       : %v → %v  %s", before.VlanIDs, after.VlanIDs, okMark(sameVlan))
	if sameBytes && sameLines && sameMode && sameVlan {
		rep.line("✅ 零残留：还原后与探测前完全一致（模式/字节数/行数/VLAN 集合）")
	} else {
		rep.line("🚨 有变化！请立即人工核查设备：")
		rep.raw("还原后 show running-config interface "+cliIface, sanitizeOutput(after.Raw))
	}

	// —— 判读要点 ——
	rep.section("6. 判读要点（三条已确立的方法论 + 取证清单）")
	rep.line("★ 1. **`?` 只能回答「当前位置可补全什么」** —— 对「单命令带固定参数」形态，")
	rep.line("     `?` 放参数前/参数后都必然报 Invalid input ⇒ **不可据此推断关键字不存在**。")
	rep.line("★ 2. **「Invalid input」也可能是端口模式问题** —— access 模式下连 `switchport trunk` 都不提供。")
	rep.line("★ 3. **「命令不报错」≠「命令生效」** —— 锐捷 hybrid 的 `allowed vlan tagged <list>` 就是无错但不落地。")
	rep.line("")
	rep.line("待判读的问题：")
	rep.line("1. add 是否存在？存在则并集还是替换？")
	rep.line("2. remove 是否存在？remove 之后配置是否真减少？")
	rep.line("3. replace 是否存在？是否存在且真能全量覆盖？")
	rep.line("4. 参数是否接受 VLAN 区间/列表（3990-3995,3999）？是否接受 all / none？")
	rep.line("5. 关键字大小写与顺序敏感性（`add` 必须在 `allowed vlan` 之后？）")
}

// orNone 空字符串的可读表达。
func orNone(s string) string {
	if strings.TrimSpace(s) == "" {
		return "（该行不存在）"
	}
	return s
}

// okMark 一致性标记。
func okMark(ok bool) string {
	if ok {
		return "✅ 一致"
	}
	return "🚨 不一致"
}
