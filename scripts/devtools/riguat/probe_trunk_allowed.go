package main

// 锐捷 `switchport trunk allowed vlan` 第二层（add / remove / replace）取证探针。
//
// ★ 纯只读：全部命令以 `?` 结尾（参数位帮助查询），外加 `show` 只读采集。
//   工具级硬约束：每条探针命令下发前都要过 isReadOnlyProbeCmd 白名单，
//   不合规的命令**直接跳过并记录**（防止把写命令误塞进探针）。
//
// ★ 零残留核查：探测前后各采集一次「接口配置块（字节数 + 行数）」与「VLAN ID 集合」，
//   前后不一致即告警 —— 万一某条命令真的改到了配置（锐捷 hybrid 的教训：
//   "命令不报错"≠"没生效"，反过来"只读命令"也未必无害），能立刻发现并报告。
//
// 背景：hybrid 那条链已查透，但 trunk 这条链还没查；锐捷 hybrid 的经验是
// 「命令不报错 ≠ 命令生效」⇒ trunk 第二层极可能也藏着「不报错但不落地」的坑。

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/device"
)

// isReadOnlyProbeCmd 判定一条探针命令是否**只读**。
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

// trunkAllowedProbeCmds 锐捷 trunk allowed 第二层的探针命令表（全部只读）。
//
// 顺序按「由外到内」：先看 trunk 子命令全景，再逐层深入 allowed vlan 的关键字。
// 末两条是参数格式提示（带一个示例 VLAN 号 + `?`），用来读设备如何提示参数格式。
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
	// ConfigBytes 接口配置块原始回显的字节数（去 ANSI/提示符后的文本）
	ConfigBytes int
	// ConfigLines 接口配置行数
	ConfigLines int
	// VlanIDs 设备 VLAN 表里的 VLAN ID 集合
	VlanIDs []int
	// Raw 接口配置原始回显（报告里留证）
	Raw string
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
	// 字节数按「接口配置块文本」计（设备自报字节数的口径可能不同，
	// 这里的关键是**探测前后一致**，不是与设备 display 的数字对齐）。
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

// runTrunkAllowedProbe 执行锐捷 trunk allowed 第二层只读取证探针。
//
// 输出三部分：① 探测前基线；② 每条探针命令的**原始回显**（不改写）；③ 零残留核查。
func runTrunkAllowedProbe(ctx context.Context, ch *execChannel, dev *deviceInfo, cliIface string,
	rep *report, timeout time.Duration) {

	rep.section("锐捷 trunk allowed 第二层只读取证（add / remove / replace）")
	rep.line("设备: %s（%s）  接口: %s", dev.DeviceName, dev.IPAddress, cliIface)
	rep.line("边界: 只跑 `?` 帮助查询与 `show` 采集，**一条写命令都不下发**（工具级白名单强制）")

	cmds := trunkAllowedProbeCmds()

	// 白名单预检：不合规的命令直接剔除并记录（双保险：列表是我们写的，仍逐条校验）
	var allowed []string
	for _, c := range cmds {
		if isReadOnlyProbeCmd(c) {
			allowed = append(allowed, c)
			continue
		}
		rep.line("⛔ 跳过（非只读，未下发）: %s", c)
	}

	var before, after probeBaseline
	var results []cmdResult

	err := ch.run(ctx, dev.ID, timeout, func(execCtx context.Context, w *device.ScrapliWrapper) error {
		// ① 探测前基线
		b, berr := collectProbeBaseline(w, cliIface)
		if berr != nil {
			return berr
		}
		before = b

		// ② 逐条探针（每条都在接口视图下）
		sender := newCmdSender(w, cliIface)
		results = sender.sendInterfaceCmds(allowed)

		// ③ 探测后基线
		a, aerr := collectProbeBaseline(w, cliIface)
		if aerr != nil {
			return aerr
		}
		after = a
		return nil
	})
	if err != nil {
		rep.line("❌ 探测执行异常: %v", err)
		rep.line("⚠️ 请人工核查设备状态（配置字节数 / VLAN 表）")
		return
	}

	// —— 基线（探测前）——
	rep.section("1. 探测前基线")
	rep.line("接口配置块: %d 字节 / %d 行", before.ConfigBytes, before.ConfigLines)
	rep.line("VLAN 表: %v", before.VlanIDs)
	rep.raw("show running-config interface "+cliIface, sanitizeOutput(before.Raw))

	// —— 探针回显（原始，不改写）——
	rep.section("2. 探针回显（原始回显，逐条）")
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

	// —— 零残留核查 ——
	rep.section("3. 零残留核查（探测前后对比）")
	sameBytes := before.ConfigBytes == after.ConfigBytes
	sameLines := before.ConfigLines == after.ConfigLines
	sameVlan := fmt.Sprint(before.VlanIDs) == fmt.Sprint(after.VlanIDs)
	rep.line("接口配置块字节数: %d → %d  %s", before.ConfigBytes, after.ConfigBytes, okMark(sameBytes))
	rep.line("接口配置行数    : %d → %d  %s", before.ConfigLines, after.ConfigLines, okMark(sameLines))
	rep.line("VLAN 表          : %v → %v  %s", before.VlanIDs, after.VlanIDs, okMark(sameVlan))
	if sameBytes && sameLines && sameVlan {
		rep.line("✅ 零残留：探测未改变设备任何配置（字节数/行数/VLAN 集合全部一致）")
	} else {
		rep.line("🚨 有变化！探测命令疑似改到了配置 —— 请立即人工核查设备：")
		rep.raw("探测后 show running-config interface "+cliIface, sanitizeOutput(after.Raw))
	}

	// —— 判读表骨架（结论由人填，工具不预设）——
	rep.section("4. 判读表（填「是否被接受 / 参数格式 / 与第一层的关系 / 判读」）")
	rep.line("%-46s | %-12s | %-34s | %-16s | %s", "命令", "是否被接受", "参数格式（? 回显）", "与第一层关系", "判读")
	rep.line("%s", strings.Repeat("-", 140))
	for _, r := range results {
		cmd := strings.TrimSpace(r.Command)
		accepted := "是"
		if r.Failed {
			accepted = "否（设备报错）"
		}
		rep.line("%-46s | %-12s | %-34s | %-16s | %s", cmd, accepted, "（抄 ? 回显）", "", "")
	}

	rep.section("5. 判读要点（沿用锐捷 hybrid 已确立的经验）")
	rep.line("1. 有没有「被接受但语义不同」的坑？锐捷 hybrid 是自动补集语义（untagged 之外全进 tagged）——trunk allowed 是否也自动补集？")
	rep.line("2. `add` 与直接指定的关系：是并集（叠加）还是替换（覆盖）？")
	rep.line("3. `remove` 是否存在？（有些厂商没有 remove）")
	rep.line("4. `replace` 是否存在？（锐捷第一层 `switchport trunk allowed vlan <list>` 是全量覆盖形态）")
	rep.line("5. 参数是否接受 VLAN 区间/列表（3990-3995,3999）？是否接受 all / none？")
	rep.line("6. 关键字大小写与顺序敏感性（`add` 必须在 `allowed vlan` 之后？`switchport trunk allowed vlan add 3990` vs `add` 前置）")
	rep.line("7. ⚠️ 「命令不报错」≠「命令生效」——锐捷 hybrid 的 `allowed vlan tagged <list>` 就是无错但不落地 ⇒ 任何「被接受」的形态都需实跑验证落地（需专用实验口）")
}

// okMark 一致性标记。
func okMark(ok bool) string {
	if ok {
		return "✅ 一致"
	}
	return "🚨 不一致"
}
