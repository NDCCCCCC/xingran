package main

// 还原引擎：组装 → 逐条下发 → 重新采集 → 与基线 diff → 差异驱动修复 → 重试。
//
// ★ 本文件是 2026-11-02 真机复测失败的修复核心：
//
//	根因：重放基线行时没有带 "interface X" 前缀，命令在 (config)# 顶层下发被设备拒绝，
//	      导致 description / dot1x_disable / set_access_vlan 三个动作还原失败、配置偏离基线。
//	修复：所有接口级命令统一经 buildInterfaceCmds 组装（必带 interface 前缀 + exit），
//	      且**逐条独立下发**（单条失败不再中断整批），还原后再重新采集 + diff 自证。
//
// 还原成功判据（lead 要求）：还原后重新采集，与权威基线 diff 为空；不一致则自动重试
// （默认 3 轮，含差异驱动修复）；仍不一致则判定还原失败，由批次模式立即停止后续动作。

import (
	"fmt"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
)

// buildActionCmds 组装"动作命令"序列。
//
// rendered 来自 portcollection.RenderCommand —— 锐捷/H3C/华为的接口级动作渲染结果自带
// "interface X" 前缀（产品 wrapInterface 语义），系统视图动作（华为 user-bind）不带。
// 因此：
//   - 带前缀 → 视为接口视图动作：原样保留前缀 + 追加厂商退一级命令（锐捷 exit）；
//   - 不带前缀 → 视为系统视图动作：命令原样下发，且**不追加**退一级命令
//     （追加会把会话掉出配置视图，破坏 scrapli 的 priv 跟踪）。
func buildActionCmds(cliIface string, rendered []string) []string {
	if len(rendered) == 0 {
		return nil
	}
	if !isInterfaceEntry(rendered[0]) {
		return append([]string{}, rendered...)
	}
	out := append([]string{}, rendered...)
	return append(out, portcollection.VendorExitViewCmd(models.VendorRuijie))
}

// buildInterfaceCmds 组装"接口视图裸命令"序列（还原/修复用）。
//
// body 是**接口级命令本身**（如 "description 3D001"、"no switchport access vlan"），
// 不含视图切换。统一组装为：
//
//	["interface <全称>"] + body + ["exit"]
//
// ★ 这里必须带 interface 前缀：真机复测证明少了前缀，命令会在 (config)# 顶层被设备拒绝。
func buildInterfaceCmds(cliIface string, body []string) []string {
	if len(body) == 0 {
		return nil
	}
	out := make([]string, 0, len(body)+2)
	out = append(out, "interface "+cliIface)
	out = append(out, body...)
	out = append(out, portcollection.VendorExitViewCmd(models.VendorRuijie))
	return out
}

// isInterfaceEntry 判断命令是否为视图切换命令（"interface X"）。
func isInterfaceEntry(cmd string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(cmd)), "interface ")
}

// cmdResult 一条命令的下发结果。
type cmdResult struct {
	Command string
	Output  string
	Failed  bool
	Err     error
	Elapsed time.Duration
	Retries int
}

// cmdSender 逐条下发器。
//
// 为什么逐条：scrapligo 的 SendConfigs 批量下发时，**任一条命令超时/失败会导致整批返回
// error、后续命令不再下发**（真机上表现为还原只做了一半）。逐条下发 + 单条错误隔离 +
// 单条重试，保证"一条坏命令不会吃掉整批还原"。
type cmdSender struct {
	w         *device.ScrapliWrapper
	cliIface  string
	singleTry int // 单条命令的重试次数（含首次）
}

// newCmdSender 创建下发器（singleTry=2 表示失败重试 1 次）。
func newCmdSender(w *device.ScrapliWrapper, cliIface string) *cmdSender {
	return &cmdSender{w: w, cliIface: cliIface, singleTry: 2}
}

// sendOne 独立下发一条命令（错误隔离 + 重试）。
func (s *cmdSender) sendOne(cmd string) cmdResult {
	res := cmdResult{Command: cmd}
	start := time.Now()
	for attempt := 1; attempt <= s.singleTry; attempt++ {
		res.Retries = attempt - 1
		responses, err := s.w.SendConfigs([]string{cmd})
		res.Elapsed = time.Since(start)
		if err != nil {
			res.Err = err
			res.Failed = true
			res.Output = err.Error()
			if attempt < s.singleTry {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			return res
		}
		res.Err = nil
		out := ""
		failed := false
		if len(responses) > 0 && responses[0] != nil {
			out = responses[0].Result
			failed = responses[0].Failed
		}
		res.Output = out
		res.Failed = failed || hasDeviceError(out)
		return res
	}
	return res
}

// sendActionCmds 下发"动作命令"序列（来自 RenderCommand）。
//
// 渲染结果自带 "interface X" 前缀 → 走接口视图路径（逐条下发，末尾自动 exit）；
// 不带前缀 → 系统视图动作（如华为 user-bind），逐条原样下发且不追加 exit
// （追加会把会话掉出配置视图，破坏 scrapli 的 priv 跟踪）。
func (s *cmdSender) sendActionCmds(rendered []string) []cmdResult {
	if len(rendered) == 0 {
		return nil
	}
	if isInterfaceEntry(rendered[0]) {
		return s.sendInterfaceCmds(rendered[1:])
	}
	results := make([]cmdResult, 0, len(rendered))
	for _, c := range rendered {
		results = append(results, s.sendOne(c))
	}
	return results
}

// sendInterfaceCmds 在接口视图下逐条下发 body 命令。
//
// 流程：interface X → body 逐条 → exit。
// 若 "interface X" 下发失败（重试后仍失败），则跳过 body（避免在错误视图下发），
// 全部标记 FAILED 并继续，保证调用方能看到完整问题清单。
func (s *cmdSender) sendInterfaceCmds(body []string) []cmdResult {
	results := make([]cmdResult, 0, len(body)+2)

	enter := s.sendOne("interface " + s.cliIface)
	results = append(results, enter)
	if enter.Failed {
		for _, c := range body {
			results = append(results, cmdResult{Command: c, Failed: true,
				Output: "未下发：进入接口视图失败（" + enter.Command + "）"})
		}
		results = append(results, cmdResult{Command: portcollection.VendorExitViewCmd(models.VendorRuijie), Failed: true,
			Output: "未下发：进入接口视图失败"})
		return results
	}

	for _, c := range body {
		results = append(results, s.sendOne(c))
	}

	results = append(results, s.sendOne(portcollection.VendorExitViewCmd(models.VendorRuijie)))
	return results
}

// interfaceSender 抽象"接口视图下发器"，让还原逻辑可在**无设备模式**下自测
// （生产实现 = *cmdSender；测试实现 = mock 锐捷设备）。
type interfaceSender interface {
	sendInterfaceCmds(body []string) []cmdResult
}

// restoreSequence 组装还原命令序列（纯函数）。
//
//   - undo 影响项（删除动作引入的配置行）→ 独立一次组装；
//   - 基线行重放（恢复被动作删除的配置行）→ 独立一次组装。
//
// ★ 两次组装都必须带 "interface X" 前缀 —— 真机复测证明少了前缀，命令会在
// (config)# 顶层被设备拒绝，还原静默失效。
func restoreSequence(cliIface string, spec actionSpec, params actionParams, baseline []string) []string {
	var seq []string
	for _, seg := range restoreSegments(cliIface, spec, params, baseline) {
		seq = append(seq, buildInterfaceCmds(cliIface, seg)...)
	}
	return seq
}

// restoreSegments 把还原拆成若干"组接口命令"（每组独立组装 + 独立下发）。
//
//   - 第 1 组：undo 影响项（删除动作引入的配置行）
//   - 第 2 组：基线行重放（恢复被动作删除的配置行）
//
// 每组下发时都会独立带 "interface X" 前缀 + exit。
func restoreSegments(cliIface string, spec actionSpec, params actionParams, baseline []string) [][]string {
	var segs [][]string
	if undoCmds := spec.Undo(baseline, params); len(undoCmds) > 0 {
		segs = append(segs, undoCmds)
	}
	if len(baseline) > 0 {
		segs = append(segs, baseline)
	}
	return segs
}

// restoreOutcome 还原结果。
type restoreOutcome struct {
	Attempts   int         // 实际尝试轮数
	FinalLines []string    // 还原后重新采集到的配置行
	Diff       []string    // 与基线的差异（空 = 还原成功）
	OK         bool        // diff 为空 = 成功
	Cmds       []cmdResult // 全部下发记录
	Fixed      []string    // 差异驱动修复下发过的命令
}

// restoreWithVerify 还原 + 自证 + 重试。
//
// 参数：
//   - baseline 动作前采集到的配置行（权威判据）
//   - collect  重新采集当前配置行的函数
//   - maxAttempt 最大尝试轮数（默认 3；lead 要求 ≥2 次重试）
//
// 每一轮：
//  1. 下发 undo 影响项（删除动作引入的配置行）；
//  2. 重放基线行（恢复被动作删除的配置行）—— 独立一次组装，必带 interface 前缀；
//  3. 重新采集 + 与基线 diff；
//  4. diff 非空 → 差异驱动修复：缺失行直接重放；多出行用 no 形式否定；
//     无法安全否定的行只报告不修。
func restoreWithVerify(
	sender interfaceSender,
	cliIface string,
	spec actionSpec,
	params actionParams,
	baseline []string,
	collect func() ([]string, error),
	maxAttempt int,
	log func(format string, args ...interface{}),
) (*restoreOutcome, error) {
	if maxAttempt < 1 {
		maxAttempt = 3
	}
	out := &restoreOutcome{}

	for attempt := 1; attempt <= maxAttempt; attempt++ {
		out.Attempts = attempt
		log("还原第 %d/%d 轮：下发 undo 影响项 + 重放基线行（%d 行）", attempt, maxAttempt, len(baseline))

		// ① undo 影响项 + ② 基线行重放（两组各自独立组装，均带 interface 前缀）
		for _, seg := range restoreSegments(cliIface, spec, params, baseline) {
			results := sender.sendInterfaceCmds(seg)
			out.Cmds = append(out.Cmds, results...)
			// 端口模式回退兜底：`switchport mode access` 被设备拒绝（Invalid input）时
			// 改用 Cisco 风格的 `default switchport mode access`（lead 批准的兜底）。
			// 若两者都不生效，running-config 会残留 `switchport mode hybrid` 行 →
			// 由下面的差异驱动修复走 `no switchport mode hybrid`（negateConfigLine 已映射）。
			for _, r := range results {
				if strings.EqualFold(strings.TrimSpace(r.Command), "switchport mode access") && r.Failed {
					log("  ⚠️ `switchport mode access` 被设备拒绝 → 兜底改用 `default switchport mode access`")
					out.Cmds = append(out.Cmds, sender.sendInterfaceCmds([]string{"default switchport mode access"})...)
					break
				}
			}
		}

		// ③ 重新采集 + diff
		lines, err := collect()
		if err != nil {
			return out, fmt.Errorf("还原后采集失败: %w", err)
		}
		out.FinalLines = lines
		out.Diff = diffConfig(baseline, lines)
		if len(out.Diff) == 0 {
			out.OK = true
			log("  ✅ 第 %d 轮还原后与基线一致（diff 为空）", attempt)
			return out, nil
		}
		log("  ⚠️ 第 %d 轮还原后仍有 %d 处差异：%s", attempt, len(out.Diff), strings.Join(out.Diff, " ; "))

		// ④ 差异驱动修复
		fix := diffFixCommands(baseline, lines)
		if len(fix) == 0 {
			log("  差异无法自动修复（无可安全否定的多余行），提前结束重试")
			break
		}
		log("  差异驱动修复: %s", strings.Join(fix, " ; "))
		out.Fixed = append(out.Fixed, fix...)
		out.Cmds = append(out.Cmds, sender.sendInterfaceCmds(fix)...)
	}

	return out, nil
}

// diffFixCommands 由基线与当前的差异推导修复命令。
//
//   - 基线有、当前无 → 直接重放该行（覆盖写）。
//   - 当前有、基线无 → 用 no 形式否定（negateConfigLine）；无法安全否定则跳过并由上层报告。
func diffFixCommands(baseline, current []string) []string {
	var fix []string
	curSet := map[string]bool{}
	for _, l := range current {
		curSet[normalizeForCompare(l)] = true
	}
	baseSet := map[string]bool{}
	for _, l := range baseline {
		baseSet[normalizeForCompare(l)] = true
	}
	for _, l := range baseline {
		if !curSet[normalizeForCompare(l)] {
			fix = append(fix, l)
		}
	}
	for _, l := range current {
		if !baseSet[normalizeForCompare(l)] {
			if neg, ok := negateConfigLine(l); ok {
				fix = append(fix, neg)
			}
		}
	}
	return fix
}
