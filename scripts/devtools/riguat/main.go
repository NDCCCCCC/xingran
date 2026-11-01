// 临时工具：锐捷 RS8607E 真机复测执行器（跑完即删）
//
// 目的：
//
//	华为 S8700 侧真机 UAT 已完成；锐捷侧此前用自研 SSH 会话复测，因 RGOS 回显含
//	光标控制序列 + PTY 折行导致逐条判定失真（结果不可信）。本工具改走**产品自身的
//	scrapli 执行链路**（internal/device 的连接池 + 任务调度器 + 执行器 + 锐捷 patched
//	platform），与真实用户从系统下发端口写操作的路径一致。
//
// 四段闭环：基线采集 → 下发 → 验证 → 还原 → 基线 diff 核对（diff 为空才算还原成功）。
//
// 安全约束：
//   - 只操作一个测试口（--port，默认 GE1/1）；
//   - 下发命令一律来自 portcollection.RenderCommand（产品真相源）；
//   - 密码只在内存中由产品连接池解密，绝不输出、绝不落盘；
//   - 无 --apply 时只打印将下发的命令，不连设备。
//
// 用法：
//
//	riguat --device <设备名> [--action <动作>] [--apply] [--probe] [其它参数]
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/device"
	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	"github.com/xingran-next/xingran-go-backend/pkg/crypto"
)

// report 执行记录（终端打印 + 可选落盘）。
type report struct {
	sb  strings.Builder
	out string
}

func (r *report) section(title string) {
	r.sb.WriteString("\n########## " + title + " ##########\n")
	fmt.Printf("\n########## %s ##########\n", title)
}

func (r *report) line(format string, args ...interface{}) {
	s := fmt.Sprintf(format, args...)
	r.sb.WriteString(s + "\n")
	fmt.Print(s + "\n")
}

func (r *report) raw(title, body string) {
	r.section(title)
	r.sb.WriteString(body + "\n")
	fmt.Print(body + "\n")
}

func (r *report) save() {
	if r.out == "" {
		return
	}
	if err := os.WriteFile(r.out, []byte(r.sb.String()), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "写输出文件失败: %v\n", err)
		return
	}
	fmt.Printf("\n[执行记录已保存] %s\n", r.out)
}

// cmdResult 一条命令的下发结果（命令 + 设备回显 + 失败标记）。
type cmdResult struct {
	Command string
	Output  string
	Failed  bool
}

// sendConfigCmds 经产品 scrapli 通道下发配置命令序列，返回逐条回显。
func sendConfigCmds(w *device.ScrapliWrapper, cmds []string) ([]cmdResult, error) {
	if len(cmds) == 0 {
		return nil, nil
	}
	responses, err := w.SendConfigs(cmds)
	if err != nil {
		return nil, err
	}
	results := make([]cmdResult, 0, len(cmds))
	for i, r := range responses {
		cmd := ""
		if i < len(cmds) {
			cmd = cmds[i]
		}
		out := ""
		failed := false
		if r != nil {
			out = r.Result
			failed = r.Failed
		}
		results = append(results, cmdResult{Command: cmd, Output: out, Failed: failed})
	}
	return results, nil
}

// sendReadCmd 下发只读命令（用户视图）。scrapli 会自动切回 privilege-exec。
func sendReadCmd(w *device.ScrapliWrapper, cmd string) (string, error) {
	resp, err := w.SendCommand(cmd, true)
	if err != nil {
		return "", err
	}
	return resp.Result, nil
}

// hasDeviceError 判断回显里是否含设备侧错误标记。
func hasDeviceError(out string) bool {
	clean := sanitizeOutput(out)
	for _, m := range deviceErrorMarkers {
		if strings.Contains(clean, m) {
			return true
		}
	}
	return false
}

// printCmdResults 打印逐条命令回显，并返回失败条数。
func printCmdResults(rep *report, title string, results []cmdResult) int {
	rep.section(title)
	failed := 0
	for _, c := range results {
		out := sanitizeOutput(c.Output)
		trimmed := strings.TrimSpace(out)
		if trimmed == "" {
			trimmed = "(无回显)"
		}
		mark := "OK"
		if c.Failed || hasDeviceError(out) {
			mark = "FAILED"
			failed++
		}
		rep.line("--- [%s] %s", mark, c.Command)
		rep.line("%s", trimmed)
	}
	return failed
}

func main() {
	var (
		cfgPath    string
		envFile    string
		backendPID int
		procRoot   string
		deviceName string
		ifaceShort string
		actionName string
		outFile    string
		vlan       int
		mac        string
		ip         string
		desc       string
		maxMac     int
		timeout    time.Duration
		apply      bool
		probe      bool
	)
	flag.StringVar(&cfgPath, "config", "/app/szh/configs/config.yaml", "后端 config.yaml 路径")
	flag.StringVar(&envFile, "sm4-env-file", "", "SM4_KEY 环境变量文件路径（如 /proc/<pid>/environ）")
	flag.IntVar(&backendPID, "backend-pid", 0, "运行中后端进程 pid（从 /proc/<pid>/environ 取 SM4_KEY）")
	flag.StringVar(&procRoot, "proc-root", "/proc", "/proc 根目录")
	flag.StringVar(&deviceName, "device", "", "设备名（sys_network_device.device_name）")
	flag.StringVar(&ifaceShort, "port", "GE1/1", "测试口（库中归一化短名，如 GE1/1）")
	flag.StringVar(&actionName, "action", "", "动作名（见下方动作列表）")
	flag.StringVar(&outFile, "out", "", "执行记录输出文件")
	flag.IntVar(&vlan, "vlan", 0, "测试 VLAN（0 = 自动选设备上存在且不同于基线的 VLAN）")
	flag.StringVar(&mac, "mac", "00:11:22:33:44:55", "测试 MAC")
	flag.StringVar(&ip, "ip", "10.62.255.254", "测试 IP（port_binding 用）")
	flag.StringVar(&desc, "desc", "RIGUAT-TEST", "测试描述（description 动作用）")
	flag.IntVar(&maxMac, "max-mac", 5, "最大 MAC 数（portsec_max_mac 用）")
	flag.DurationVar(&timeout, "timeout", 5*time.Minute, "单个动作超时")
	flag.BoolVar(&apply, "apply", false, "真正下发（默认只打印将下发的命令，不连设备）")
	flag.BoolVar(&probe, "probe", false, "只读探测模式（接口视图 ? 帮助查询，零配置变更）")
	flag.Parse()

	if deviceName == "" {
		printUsage()
		os.Exit(2)
	}

	rep := &report{out: outFile}

	// ---------- 环境装配 ----------
	cfg, err := loadDBConfig(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(3)
	}
	db, err := openDB(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(3)
	}

	sm4Key, keySource, err := resolveSM4Key(envFile, backendPID, procRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(3)
	}
	cipher, err := crypto.NewSM4Cipher(sm4Key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "SM4 初始化失败: %v\n", err)
		os.Exit(3)
	}

	dev, err := loadDevice(db, deviceName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(4)
	}
	if dev.Vendor != models.VendorRuijie {
		fmt.Fprintf(os.Stderr, "本工具仅面向锐捷设备，当前设备厂商=%s\n", dev.Vendor)
		os.Exit(4)
	}

	cliIface := expandInterfaceName(ifaceShort, dev.Vendor)

	rep.section("0. 环境与目标")
	rep.line("设备      : %s (%s %s) %s", dev.DeviceName, dev.Vendor, dev.Model, dev.IPAddress)
	rep.line("接口      : %s → CLI 全称 %s", ifaceShort, cliIface)
	rep.line("SM4 密钥  : 来源 %s（密钥本身不打印）", keySource)

	// 端口采集行（可选，提供 dot1x limit 与基线 VLAN 参考）
	var prow *portRow
	if p, perr := loadPort(db, dev.ID, ifaceShort); perr == nil {
		prow = p
		rep.line("端口采集行: admin=%s oper=%s desc=%q vlan=%v dot1x_limit=%v",
			p.AdminStatus, p.OperStatus, p.Description, p.VLAN, p.Dot1xUserLimit)
	} else {
		rep.line("端口采集行: 未采集到（%v），继续执行（dot1x limit 兜底为 1）", perr)
	}

	// ---------- 只读探测模式 ----------
	if probe {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		ch := newExecChannel(db, cipher)
		defer ch.Close()
		runProbe(ctx, ch, dev, cliIface, rep, timeout)
		rep.save()
		return
	}

	if actionName == "" {
		printUsage()
		rep.save()
		return
	}
	spec, ok := actionRegistry[actionName]
	if !ok {
		fmt.Fprintf(os.Stderr, "未知动作: %s\n可用动作: %s\n", actionName, strings.Join(actionNames, ", "))
		os.Exit(2)
	}

	// ---------- 渲染参数（产品真相源） ----------
	baseVLAN := 0
	if prow != nil && prow.VLAN != nil {
		baseVLAN = *prow.VLAN
	}
	limit := 1
	if prow != nil && prow.Dot1xUserLimit != nil && *prow.Dot1xUserLimit > 0 {
		limit = *prow.Dot1xUserLimit
	}
	params := actionParams{
		CliIface:     cliIface,
		VLAN:         vlan,
		MAC:          mac,
		IP:           ip,
		Desc:         desc,
		MaxMAC:       maxMac,
		Dot1xLimit:   &limit,
		BaselineVLAN: baseVLAN,
	}

	if !apply {
		// DRY-RUN：不连设备。VLAN 用估算值渲染（实际执行时会先采集 `show vlan`
		// 校验/自动选择存在的 VLAN）。
		estVLAN := vlan
		if estVLAN == 0 {
			estVLAN = baseVLAN
			if estVLAN == 0 {
				estVLAN = 318
			}
		}
		est := params
		est.VLAN = estVLAN
		tpl := spec.Build(est)
		tpl.InterfaceName = cliIface
		rendered, rerr := portcollection.RenderCommand(dev.Vendor, spec.PortAction, tpl)
		rep.section("1. 动作与命令（来源 portcollection.RenderCommand）")
		rep.line("动作      : %s（产品动作标识 %s）", spec.Name, spec.PortAction)
		rep.line("判读要点  :\n%s", spec.Judgement)
		if rerr != nil {
			rep.line("渲染失败  : %v", rerr)
		} else {
			rep.line("下发命令  :")
			for _, c := range rendered {
				rep.line("  %s", c)
			}
		}
		rep.section("DRY-RUN")
		rep.line("未加 --apply，不连设备、不下发任何命令。")
		rep.line("还原命令依赖真机基线采集，--apply 时按「undo 动作影响项 + 重放基线行」自动生成。")
		if needsVLAN(spec) && vlan == 0 {
			rep.line("注意：本次 VLAN 为自动选择（--vlan 未指定）—— 执行时会先读 `show vlan` 选一个设备上存在且不同于基线的 VLAN。")
		}
		rep.save()
		return
	}

	// ---------- 执行四段闭环 ----------
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	ch := newExecChannel(db, cipher)
	defer ch.Close()

	runAction(ctx, ch, dev, spec, params, rep, timeout)
	rep.save()
}

// needsVLAN 该动作是否需要 VLAN 参数。
func needsVLAN(spec actionSpec) bool {
	switch spec.PortAction {
	case portcollection.ActionSetAccessVLAN, portcollection.PortSecStaticBind:
		return true
	}
	return false
}

// pickTargetVLAN 选一个设备上存在、且不同于基线的 VLAN（避免与基线同值导致幂等无验证价值）。
func pickTargetVLAN(vlans map[int]bool, baseVLAN int) int {
	for v := 2; v <= 4094; v++ {
		if vlans[v] && v != baseVLAN {
			return v
		}
	}
	if baseVLAN > 0 {
		return baseVLAN
	}
	return 0
}

// verifyCmdsFor 验证命令（真机实测可用）。
//
// 注意：锐捷 `show interfaces status <port>` **不接受接口参数**（Invalid input），
// 故不纳入；端口状态改用 `show port-security interface <port>` 与 binding 表判断。
func verifyCmdsFor(cliIface string) []string {
	return []string{
		"show running-config interface " + cliIface,
		"show port-security interface " + cliIface,
		"show port-security binding",
	}
}

// runAction 执行「基线 → 下发 → 验证 → 还原 → 核对」闭环。
func runAction(ctx context.Context, ch *execChannel, dev *deviceInfo, spec actionSpec,
	params actionParams, rep *report, timeout time.Duration) {

	cliIface := params.CliIface

	// ① VLAN 决策（只读采集 `show vlan`，确保下发的 VLAN 在设备上真实存在）
	var vlans map[int]bool
	var vlanRaw string
	if needsVLAN(spec) {
		verr := ch.run(ctx, dev.ID, timeout, func(execCtx context.Context, w *device.ScrapliWrapper) error {
			out, err := sendReadCmd(w, "show vlan")
			if err != nil {
				return fmt.Errorf("show vlan 采集失败: %w", err)
			}
			vlanRaw = out
			vlans = parseVlanIDs(out)
			return nil
		})
		if verr != nil {
			rep.section("执行异常")
			rep.line("⚠️ %v", verr)
			return
		}
		rep.section("1. VLAN 决策（只读 `show vlan`）")
		rep.raw("show vlan", sanitizeOutput(vlanRaw))
		rep.line("解析到的 VLAN ID: %v", sortedIntKeys(vlans))
		if params.VLAN == 0 {
			picked := pickTargetVLAN(vlans, params.BaselineVLAN)
			if picked == 0 {
				rep.line("⚠️ 无法确定可用 VLAN（设备 VLAN 列表解析为空且基线无 VLAN），中止。")
				return
			}
			rep.line("自动选择 VLAN: %d（基线 VLAN=%d）", picked, params.BaselineVLAN)
			params.VLAN = picked
		} else {
			if len(vlans) > 0 && !vlans[params.VLAN] {
				rep.line("⚠️ 指定的 VLAN %d 在设备上不存在，中止（不下发）。", params.VLAN)
				return
			}
			rep.line("使用指定 VLAN: %d", params.VLAN)
		}
	}

	// ② 渲染（产品真相源）
	tpl := spec.Build(params)
	tpl.InterfaceName = cliIface
	rendered, rerr := portcollection.RenderCommand(dev.Vendor, spec.PortAction, tpl)
	if rerr != nil {
		rep.section("渲染失败")
		rep.line("⚠️ %v", rerr)
		return
	}
	rep.section("2. 动作与命令（来源 portcollection.RenderCommand）")
	rep.line("动作      : %s（产品动作标识 %s）", spec.Name, spec.PortAction)
	rep.line("判读要点  :\n%s", spec.Judgement)
	rep.line("下发命令  :")
	for _, c := range rendered {
		rep.line("  %s", c)
	}

	var (
		baseLines  []string
		baseErrs   []string
		baseWarns  []string
		baseOut    map[string]string
		actionOut  []cmdResult
		afterLines []string
		afterErrs  []string
		afterWarns []string
		afterOut   map[string]string
		restoreOut []cmdResult
		finalLines []string
		finalErrs  []string
		finalWarns []string
		finalOut   map[string]string
	)

	err := ch.run(ctx, dev.ID, timeout, func(execCtx context.Context, w *device.ScrapliWrapper) error {
		// ① 基线采集
		baseOut = map[string]string{}
		for _, c := range verifyCmdsFor(cliIface) {
			out, err := sendReadCmd(w, c)
			if err != nil {
				return fmt.Errorf("基线采集失败 [%s]: %w", c, err)
			}
			baseOut[c] = out
		}
		baseLines, baseErrs, baseWarns = ifaceConfigBlock(baseOut["show running-config interface "+cliIface], cliIface)

		// ② 下发动作
		sent, err := sendConfigCmds(w, buildWriteCmds(cliIface, rendered))
		if err != nil {
			return fmt.Errorf("下发失败: %w", err)
		}
		actionOut = sent

		// ③ 验证
		afterOut = map[string]string{}
		for _, c := range verifyCmdsFor(cliIface) {
			out, err := sendReadCmd(w, c)
			if err != nil {
				return fmt.Errorf("验证采集失败 [%s]: %w", c, err)
			}
			afterOut[c] = out
		}
		afterLines, afterErrs, afterWarns = ifaceConfigBlock(afterOut["show running-config interface "+cliIface], cliIface)

		// ④ 还原：undo 动作影响项 → 重放基线行
		undoCmds := spec.Undo(baseLines, params)
		restoreSeq := buildWriteCmds(cliIface, undoCmds)
		if len(baseLines) > 0 {
			restoreSeq = append(restoreSeq, buildWriteCmds(cliIface, baseLines)...)
		}
		restored, err := sendConfigCmds(w, restoreSeq)
		if err != nil {
			return fmt.Errorf("还原失败: %w", err)
		}
		restoreOut = restored
		// ⑤ 还原后基线核对
		finalOut = map[string]string{}
		for _, c := range verifyCmdsFor(cliIface) {
			out, err := sendReadCmd(w, c)
			if err != nil {
				return fmt.Errorf("还原后采集失败 [%s]: %w", c, err)
			}
			finalOut[c] = out
		}
		finalLines, finalErrs, finalWarns = ifaceConfigBlock(finalOut["show running-config interface "+cliIface], cliIface)
		return nil
	})

	if err != nil {
		rep.section("执行异常")
		rep.line("⚠️ %v", err)
		rep.line("已采集到的中间结果如下（请人工核对设备当前配置）：")
		if len(baseLines) > 0 {
			rep.line("动作前基线行: %s", strings.Join(baseLines, " | "))
		}
		if len(afterLines) > 0 {
			rep.line("动作后配置行: %s", strings.Join(afterLines, " | "))
		}
		return
	}

	// ---------- 报告 ----------
	rep.section("3. 动作前基线")
	for _, c := range verifyCmdsFor(cliIface) {
		rep.raw(c, sanitizeOutput(baseOut[c]))
	}
	rep.line("--- 解析出的接口配置行（%d 行）---", len(baseLines))
	for i, l := range baseLines {
		rep.line("  %2d. %s", i+1, l)
	}
	// 与 lead 实测的 9 行权威基线自检
	if d := diffConfig(baselineRuijieGE11, baseLines); len(d) > 0 {
		rep.line("⚠️ 采集到的基线与 lead 实测的 9 行权威基线不一致（%d 处）：", len(d))
		for _, x := range d {
			rep.line("    %s", x)
		}
		rep.line("    → 设备当前状态已偏离 lead 实测时点，或本次解析有偏差；请人工判读后再继续。")
	} else {
		rep.line("✅ 采集基线与 lead 实测的 9 行权威基线逐行一致")
	}
	if len(baseErrs) > 0 {
		rep.line("设备错误行: %s", strings.Join(baseErrs, " | "))
	}
	for _, w := range baseWarns {
		rep.line("⚠️ %s", w)
	}

	failed := printCmdResults(rep, "4. 下发动作", actionOut)
	rep.line("下发失败条数: %d", failed)

	rep.section("5. 验证（动作后）")
	for _, c := range verifyCmdsFor(cliIface) {
		rep.raw(c, sanitizeOutput(afterOut[c]))
	}
	rep.line("--- 动作后接口配置行（%d 行）---", len(afterLines))
	for i, l := range afterLines {
		rep.line("  %2d. %s", i+1, l)
	}
	if len(afterErrs) > 0 {
		rep.line("设备错误行: %s", strings.Join(afterErrs, " | "))
	}
	for _, w := range afterWarns {
		rep.line("⚠️ %s", w)
	}
	rep.line("--- 动作前后配置行差异 ---")
	actDiff := diffConfig(baseLines, afterLines)
	if len(actDiff) == 0 {
		rep.line("（无差异：本次动作未改变接口配置 —— 可能命令未生效，或与基线同值）")
	}
	for _, d := range actDiff {
		rep.line("  %s", d)
	}

	// 锐捷专项断言
	rep.section("6. 锐捷专项判读")
	judgeRuijie(spec, baseLines, afterLines, actionOut, rep)

	// 还原
	restoreFailed := printCmdResults(rep, "7. 还原（undo 动作影响项 + 重放基线行）", restoreOut)
	rep.line("还原阶段失败条数: %d（基线重放是幂等操作，「已存在/重复」类报错不影响最终 diff，以第 8 段 diff 为准）", restoreFailed)

	// ---------- 最终核对 ----------
	rep.section("8. 最终配置 vs 原始基线（逐行核对）")
	for _, c := range verifyCmdsFor(cliIface) {
		if c == "show running-config interface "+cliIface {
			continue
		}
		rep.raw("还原后 "+c, sanitizeOutput(finalOut[c]))
	}
	rep.line("%-4s %-52s %s", "序号", "基线行", "最终核对")
	ok := true
	for i, l := range baseLines {
		mark := "✗ 缺失"
		if containsLine(finalLines, l) {
			mark = "✓ 一致"
		} else {
			ok = false
		}
		rep.line("%-4d %-52s %s", i+1, l, mark)
	}
	extra := diffConfig(baseLines, finalLines)
	extraOnly := false
	rep.line("--- 最终配置多出的行 ---")
	for _, e := range extra {
		if strings.HasPrefix(e, "多出:") {
			ok = false
			extraOnly = true
			rep.line("  %s", e)
		}
	}
	if !extraOnly {
		rep.line("  （无）")
	}
	if len(finalErrs) > 0 {
		rep.line("最终采集到的设备错误行: %s", strings.Join(finalErrs, " | "))
	}
	for _, w := range finalWarns {
		rep.line("⚠️ %s", w)
	}
	if len(extra) == 0 && ok {
		rep.line("✅ 还原核对通过：最终配置与原始基线完全一致（diff 为空）")
	} else {
		rep.line("⚠️ 还原核对失败：与基线存在 %d 处差异，需人工处理", len(extra))
	}
}

// judgeRuijie 输出锐捷专项断言（把判读证据摆出来，由 lead 判读）。
func judgeRuijie(spec actionSpec, baseLines, afterLines []string, results []cmdResult, rep *report) {
	switch spec.Name {
	case "port_binding":
		rep.line("① `switchport port-security binding <mac> <ip>` 参数顺序：")
		for _, c := range results {
			out := sanitizeOutput(c.Output)
			rep.line("   下发: %s", c.Command)
			rep.line("   回显: %s", strings.TrimSpace(out))
			rep.line("   判定: %s", verdict(out, c.Failed))
		}
		rep.line("   保护性 shutdown 检查：动作后配置含 shutdown 行 = %v", containsLine(afterLines, "shutdown"))
		rep.line("   基线已有 binding 行：%s", joinMatching(baseLines, "switchport port-security binding"))
		rep.line("   动作后 binding 行：%s", joinMatching(afterLines, "switchport port-security binding"))
	case "portsec_sticky":
		rep.line("② `switchport port-security mac-address sticky` 支持性：")
		for _, c := range results {
			out := sanitizeOutput(c.Output)
			rep.line("   下发: %s", c.Command)
			rep.line("   回显: %s", strings.TrimSpace(out))
			rep.line("   判定: %s", verdict(out, c.Failed))
		}
	case "portsec_static_bind":
		rep.line("③ `switchport port-security mac-address <mac> vlan <vlan>` 支持性：")
		for _, c := range results {
			out := sanitizeOutput(c.Output)
			rep.line("   下发: %s", c.Command)
			rep.line("   回显: %s", strings.TrimSpace(out))
			rep.line("   判定: %s", verdict(out, c.Failed))
		}
	default:
		rep.line("（本动作无锐捷专项断言）")
	}
}

// verdict 依据设备回显给出结论。
func verdict(out string, failed bool) string {
	out = sanitizeOutput(out)
	switch {
	case failed:
		return "设备拒绝（scrapli failed-when-contains 命中）"
	case hasDeviceError(out):
		return "设备回显含错误标记 —— 命令未被 RGOS 接受"
	case strings.TrimSpace(out) == "":
		return "无回显（需人工确认是否生效）"
	default:
		return "无错误标记（命令被接受）"
	}
}

// joinMatching 拼接以 prefix 开头的行（不存在时返回 "（无）"）。
func joinMatching(lines []string, prefix string) string {
	var got []string
	for _, l := range lines {
		if strings.HasPrefix(strings.ToLower(l), strings.ToLower(prefix)) {
			got = append(got, l)
		}
	}
	if len(got) == 0 {
		return "（无）"
	}
	return strings.Join(got, " | ")
}

// buildWriteCmds 组装写命令序列（与产品写路径一致）。
//
// rendered 若首元素是 "interface X" → 视为接口视图动作：
// 输出 ["interface X"] + 视图内命令 + ["exit"]（锐捷退一级用 exit）；
// 否则视为系统视图动作：命令原样下发（不追加退出命令，否则会话掉出配置视图）。
func buildWriteCmds(cliIface string, cmds []string) []string {
	if len(cmds) == 0 {
		return nil
	}
	first := strings.TrimSpace(cmds[0])
	if strings.HasPrefix(strings.ToLower(first), "interface ") {
		body := cmds[1:]
		out := make([]string, 0, len(body)+2)
		out = append(out, "interface "+cliIface)
		out = append(out, body...)
		out = append(out, portcollection.VendorExitViewCmd(models.VendorRuijie))
		return out
	}
	return append([]string{}, cmds...)
}

// probeCmds 接口视图只读探测命令（? 帮助查询，零配置变更）。
func probeCmds() []string {
	return []string{
		"switchport ?",
		"switchport hybrid ?",
		"switchport port-security ?",
		"switchport port-security mac-address ?",
		"switchport port-security maximum ?",
		"dot1x ?",
	}
}

// runProbe 只读探测：接口视图 ? 帮助查询（零配置变更）。
func runProbe(ctx context.Context, ch *execChannel, dev *deviceInfo, cliIface string, rep *report, timeout time.Duration) {
	rep.section("只读探测（接口视图 ? 帮助查询，零配置变更）")
	rep.line("说明：switchport hybrid 的支持性结论以设备帮助回显为准。")

	var results []cmdResult
	err := ch.run(ctx, dev.ID, timeout, func(execCtx context.Context, w *device.ScrapliWrapper) error {
		seq := append([]string{"interface " + cliIface}, probeCmds()...)
		seq = append(seq, portcollection.VendorExitViewCmd(models.VendorRuijie))
		sent, err := sendConfigCmds(w, seq)
		if err != nil {
			return err
		}
		results = sent
		// 顺带采集只读状态
		for _, c := range verifyCmdsFor(cliIface) {
			out, err := sendReadCmd(w, c)
			if err != nil {
				rep.line("采集 %s 失败: %v", c, err)
				continue
			}
			rep.raw(c, sanitizeOutput(out))
		}
		return nil
	})
	if err != nil {
		rep.line("探测执行异常: %v", err)
	}
	printCmdResults(rep, "探测回显", results)
	rep.line("结论提示：")
	rep.line("  - 若 `switchport hybrid ?` 回显列出 hybrid 子命令 → 设备支持 hybrid，")
	rep.line("    而产品当前锐捷渲染层未注册 TrunkSetHybrid（RenderCommand 返回 ErrUnknownAction），")
	rep.line("    建议在 portcollection 补锐捷 hybrid 模板后再开放该动作；")
	rep.line("  - 若回显为 Invalid input / Unknown command → 设备不支持，产品现状正确，无需补。")
}

// printUsage 打印用法。
func printUsage() {
	fmt.Println("用法: riguat --device <设备名> [选项]")
	fmt.Println()
	fmt.Println("动作（--action）:")
	for _, n := range actionNames {
		spec := actionRegistry[n]
		fmt.Printf("  %-20s %s\n", n, strings.ReplaceAll(spec.Judgement, "\n", " "))
	}
	fmt.Println()
	fmt.Println("选项:")
	fmt.Println("  --device <设备名>       必填，sys_network_device.device_name")
	fmt.Println("  --port <短名>           测试口，默认 GE1/1（库中归一化短名）")
	fmt.Println("  --action <动作>         动作名")
	fmt.Println("  --apply                 真正下发（默认只打印命令）")
	fmt.Println("  --probe                 只读探测模式")
	fmt.Println("  --config <path>         后端 config.yaml，默认 /app/szh/configs/config.yaml")
	fmt.Println("  --backend-pid <pid>     从 /proc/<pid>/environ 取 SM4_KEY")
	fmt.Println("  --sm4-env-file <path>   直接指定环境变量文件")
	fmt.Println("  --vlan/--mac/--ip/--desc/--max-mac  测试参数")
	fmt.Println("  --out <path>            执行记录输出文件")
}
