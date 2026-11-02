package main

// 锐捷 `switchport trunk allowed vlan` 第二层（add / remove / replace）取证探针。
//
// ★★ 机制性发现（2026-11-02 首轮探针实测，RG8607E GE1/25）：
//
//	 switchport trunk ?                  → Invalid input
//	 switchport trunk allowed vlan all ? → Can't set allow list on this port mode!
//
// ⇒ **锐捷的 `?` 帮助可选项集合依赖当前端口模式**：口处于 access 模式时，
//   设备**连 `switchport trunk` 这个关键字本身都不提供**，于是 8 条探针全被
//   "Invalid input" 挡掉 —— 但那只说明「当前模式下不提供」，**不能推断「关键字不存在」**。
//   （hybrid 那轮没踩到这个坑，是因为它就在 hybrid 模式下查的。）
// ⇒ 所以取证必须**先进入 trunk 模式**再查，且**必须记录探测时的端口模式**。
//
// ★ 只读性质的三道闸：
//  1. 只读白名单 isReadOnlyProbeCmd：只放行「以 ? 结尾」与「show 前缀」；
//  2. 受控写白名单 modeSwitchPairs：**仅** `switchport mode trunk` /
//     `switchport mode access` 两条，且**必须成对**（trunk 必配 access 还原）——
//     没有 access 还原就不允许 trunk（测试锁定）；
//  3. 零残留核查：**还原后 vs 探测前**比对（中间那段 trunk 模式是预期内的），
//     且还原动作在错误路径上也会执行（探针失败也必须还原）。
//
// 背景：hybrid 那条链已查透，教训是「命令不报错」≠「命令生效」
// ⇒ trunk 第二层极可能也藏着「不报错但不落地」的坑。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/device"
)

// isReadOnlyProbeCmd 判定一条探针命令是否**完全只读**。
//
// 白名单只有两类：
//  1. 以 `?` 结尾 —— 参数位帮助查询（Cisco 风格 `cmd <partial> ?`），设备只回显候选；
//  2. 以 `show ` 开头 —— 只读采集。
//
// 其它一律拒绝（写命令、`no ...`、`switchport ...` 的赋值形态都会被挡下）。
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
//
// ★ 只允许两条，且必须成对：没有还原命令就不允许执行切换（见 modeSwitchAllowed）。
var modeSwitchPairs = map[string]string{
	"switchport mode trunk":  "switchport mode access",
	"switchport mode access": "",
}

// modeSwitchAllowed 判定一条**受控写命令**是否被允许，且其还原命令是否齐备。
//
// 返回 (允许, 还原命令)。trunk 切换缺还原 ⇒ **不允许**。
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
//
// 探针入口用这条做最后一道闸：只要 trunk 切换缺它的 access 还原，整组拒绝执行。
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
	// 必须同时出现 trunk 切换与其 access 还原（成对）
	return hasTrunk == hasAccess
}

// trunkAllowedProbeCmds 锐捷 trunk allowed 第二层的探针命令表（全部只读）。
//
// 顺序按「由外到内」：先看 trunk 子命令全景，再逐层深入 allowed vlan 的关键字。
// 末三条是参数格式提示（带一个示例 VLAN 号 + `?`），用来读设备如何提示参数格式。
func trunkAllowedProbeCmds() []string {
	return []string{
		// —— 第一层（已知可用的那层，作为对照）——
		"switchport trunk ?",
		"switchport trunk allowed ?",
		"switchport trunk allowed vlan ?",
		// —— 第二层：本次要查的 add / remove / replace ——
		"switchport trunk allowed vlan add ?",
		"switchport trunk allowed vlan remove ?",
		"switchport trunk allowed vlan replace ?",
		// —— 参数格式提示（看设备怎么提示 VLAN 列表/区间形态）——
		"switchport trunk allowed vlan 3990 ?",
		"switchport trunk allowed vlan all ?",
		"switchport trunk allowed vlan none ?",
	}
}

// probeBaseline 探针前/后的设备状态快照（用于零残留核查）。
type probeBaseline struct {
	// ConfigBytes 接口配置块文本的字节数
	ConfigBytes int
	// ConfigLines 接口配置行数
	ConfigLines int
	// PortMode 探测时的端口模式（access / trunk / hybrid / 其它）
	PortMode string
	// VlanIDs 设备 VLAN 表里的 VLAN ID 集合
	VlanIDs []int
	// Raw 接口配置原始回显（报告里留证）
	Raw string
}

// parsePortMode 从接口配置行判定端口模式。
//
// 锐捷 access 是**默认模式** ⇒ running-config 里**不落** mode 行
// （Phase 129 的 9 行权威基线里就没有 mode 行）⇒ 无 mode 行即 access。
func parsePortMode(lines []string) string {
	for _, l := range lines {
		t := strings.ToLower(strings.TrimSpace(l))
		if strings.HasPrefix(t, "switchport mode ") {
			return strings.TrimSpace(strings.TrimPrefix(t, "switchport mode "))
		}
	}
	return "access（默认，running-config 不落 mode 行）"
}

// collectProbeBaseline 采集只读状态快照。
func collectProbeBaseline(w *device.ScrapliWrapper, cliIface string) (probeBaseline, error) {
	var b probeBaseline

	cfgOut, err := sendReadCmd(w, "show running-config interface "+cliIface)
	if err != nil {
		return b, fmt.Errorf("采集接口配置失败: %w", err)
	}
	lines, _, _ := ifaceConfigBlock(cfgOut, cliIface)
	b.ConfigLines = len(lines)
	b.Raw = cfgOut
	b.PortMode = parsePortMode(lines)
	var sb strings.Builder
	for _, l := range lines {
		sb.WriteString(l)
		sb.WriteString("\n")
	}
	b.ConfigBytes = sb.Len()

	vlanOut, err := sendReadCmd(w, "show vlan")
	if err != nil {
		return b, fmt.Errorf("采集 VLAN 表失败: %w", err)
	}
	b.VlanIDs = sortedIntKeys(parseVlanIDs(vlanOut))
	return b, nil
}

// runTrunkAllowedProbe 执行锐捷 trunk allowed 第二层取证探针。
//
// switchMode=true 时：先受控切到 trunk 模式 → 跑 9 条 `?` 探针 → **必定还原**回
// access 模式（错误路径也会还原）⇒ 零残留核查比对的是「**还原后 vs 探测前**」。
//
// switchMode=false 时：纯只读（不切模式）⇒ 零残留核查比对「探测前后」。
func runTrunkAllowedProbe(ctx context.Context, ch *execChannel, dev *deviceInfo, cliIface string,
	rep *report, timeout time.Duration, switchMode bool) {

	rep.section("锐捷 trunk allowed 第二层取证（add / remove / replace）")
	rep.line("设备: %s（%s）  接口: %s", dev.DeviceName, dev.IPAddress, cliIface)
	if switchMode {
		rep.line("模式: **受控切换** —— 先切 trunk 模式查关键字，查完必定还原为 access")
		rep.line("      （锐捷的 ? 帮助可选项集合依赖端口模式：access 模式下连 `switchport trunk` 都不提供）")
	} else {
		rep.line("模式: **纯只读**（不切端口模式）—— 注意：access 模式下设备不提供 trunk 关键字，")
		rep.line("      探针报 Invalid input 只说明「当前模式不提供」，不能推断「关键字不存在」")
	}

	cmds := trunkAllowedProbeCmds()

	// 只读白名单预检
	var allowed []string
	for _, c := range cmds {
		if isReadOnlyProbeCmd(c) {
			allowed = append(allowed, c)
			continue
		}
		rep.line("⛔ 跳过（非只读，未下发）: %s", c)
	}

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

	var before, after probeBaseline
	var results []cmdResult
	var modeSwitchOut []cmdResult
	var restoreOut []cmdResult

	err := ch.run(ctx, dev.ID, timeout, func(execCtx context.Context, w *device.ScrapliWrapper) error {
		sender := newCmdSender(w, cliIface)

		// ① 探测前基线
		b, berr := collectProbeBaseline(w, cliIface)
		if berr != nil {
			return berr
		}
		before = b

		// ② 受控切 trunk 模式（若启用）
		if switchMode {
			modeSwitchOut = sender.sendInterfaceCmds([]string{"switchport mode trunk"})
		}

		// ③ 逐条探针（全部在接口视图下）
		results = sender.sendInterfaceCmds(allowed)

		// ④ 必定还原为 access（无论探针是否出错）—— 零残留的前提
		if switchMode {
			restoreOut = sender.sendInterfaceCmds([]string{"switchport mode access"})
		}

		// ⑤ 还原后基线（零残留核查的比对对象）
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

	// —— 基线（探测前）——
	rep.section("1. 探测前基线")
	rep.line("端口模式      : %s", before.PortMode)
	rep.line("接口配置块    : %d 字节 / %d 行", before.ConfigBytes, before.ConfigLines)
	rep.line("VLAN 表       : %v", before.VlanIDs)
	rep.raw("show running-config interface "+cliIface, sanitizeOutput(before.Raw))

	// —— 模式切换回显（受控写）——
	if switchMode {
		rep.section("2. 受控模式切换回显")
		printCmdResults(rep, "切 trunk 模式", modeSwitchOut)
		printCmdResults(rep, "还原 access 模式", restoreOut)
	}

	// —— 探针回显（原始，不改写）——
	rep.section("3. 探针回显（原始回显，逐条）")
	for i, r := range results {
		cmd := strings.TrimSpace(r.Command)
		rep.line("──── [%d/%d] %s  （探测时端口模式: %s）", i+1, len(results), cmd, before.PortMode)
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

	// —— 零残留核查 ——
	rep.section("4. 零残留核查（还原后 vs 探测前）")
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

	// —— 判读表骨架（结论由人填，工具不预设）——
	rep.section("5. 判读表（填「是否被接受 / 参数格式 / 与第一层的关系 / 判读」）")
	rep.line("探测时端口模式: %s", before.PortMode)
	rep.line("%-46s | %-14s | %-34s | %-16s | %s", "命令", "是否被接受", "参数格式（? 回显）", "与第一层关系", "判读")
	rep.line("%s", strings.Repeat("-", 140))
	for _, r := range results {
		cmd := strings.TrimSpace(r.Command)
		accepted := "是"
		if r.Failed {
			accepted = "否（设备报错）"
		}
		rep.line("%-46s | %-14s | %-34s | %-16s | %s", cmd, accepted, "（抄 ? 回显）", "", "")
	}

	rep.section("6. 判读要点（沿用锐捷 hybrid 已确立的经验）")
	rep.line("1. 有没有「被接受但语义不同」的坑？锐捷 hybrid 是自动补集语义（untagged 之外全进 tagged）——trunk allowed 是否也自动补集？")
	rep.line("2. `add` 与直接指定的关系：是并集（叠加）还是替换（覆盖）？")
	rep.line("3. `remove` 是否存在？（有些厂商没有 remove）")
	rep.line("4. `replace` 是否存在？（锐捷第一层 `switchport trunk allowed vlan <list>` 是全量覆盖形态）")
	rep.line("5. 参数是否接受 VLAN 区间/列表（3990-3995,3999）？是否接受 all / none？")
	rep.line("6. 关键字大小写与顺序敏感性（`add` 必须在 `allowed vlan` 之后？）")
	rep.line("7. ⚠️ 「命令不报错」≠「命令生效」——锐捷 hybrid 的 `allowed vlan tagged <list>` 就是无错但不落地 ⇒ 任何「被接受」的形态都需实跑验证落地（需专用实验口）")
	rep.line("8. ⚠️ **「Invalid input」也可能是模式问题**：锐捷 access 模式下 `switchport trunk ?` 直接 Invalid input ⇒ 判读前先确认端口模式")
}

// okMark 一致性标记。
func okMark(ok bool) string {
	if ok {
		return "✅ 一致"
	}
	return "🚨 不一致"
}
