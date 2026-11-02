package main

// 输出清洗与逐行判定。
//
// 为什么需要这一层（锐捷 RGOS 真机复测的核心）：
//
//  1. RGOS 回显含 ANSI 控制序列（光标定位/清行），直接按行切分会把
//     "\x1b[2K\r" 之类的字节粘到配置行首尾，导致基线行与还原行永远不等；
//  2. 设备以 PTY 80 列回显，超长行会被终端折行，续行不以配置关键字开头；
//  3. 命令被拒绝时设备打印 "% Invalid input detected at '^' marker." 等错误行，
//     若混进配置行集合，会把"下发失败"伪装成"配置已变更"。
//
// 因此：先清洗 → 再按接口块切分 → 再分类为 配置行 / 错误行 / 噪声。

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	// CSI 序列：ESC [ 参数 中间字节 最终字节
	ansiCSI = regexp.MustCompile("\x1b\\[[0-9;?]*[ -/]*[@-~]")
	// OSC 序列：ESC ] ... BEL 或 ESC \
	ansiOSC = regexp.MustCompile("\x1b\\][^\x07\x1b]*(?:\x07|\x1b\\\\)")
	// 两字节转义序列
	ansiEsc = regexp.MustCompile("\x1b[@-Z\\\\-_]")
	// 其余控制字符（保留 \n \t）
	ctrlChars = regexp.MustCompile("[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]")
)

// deviceErrorMarkers 设备侧"命令被拒绝"的错误标记（锐捷 % 前缀）。
//
// 只收锐捷/思科风格的**完整**错误串 —— 过宽的标记（如 "not exist"）会与
// `?` 帮助文本里的说明措辞相撞，造成误报 FAILED。
var deviceErrorMarkers = []string{
	"% Invalid input",
	"% Unknown command",
	"% Ambiguous command",
	"% Incomplete command",
	"% Error:",
	"Invalid input detected",
	"Unrecognized command",
	"Wrong parameter",
}

// globalConfigLinePrefixes 全局配置行前缀。
//
// `show running-config interface X` 正常只返回该接口块（以 "!" 收尾）；
// 若设备实现异常没有 "!" 收尾，用这组前缀在遇到全局段时提前截断，
// 避免把全局配置误收进接口配置行（那会让基线 diff 永远不空）。
var globalConfigLinePrefixes = []string{
	"hostname",
	"ip route",
	"ip default-gateway",
	"line vty",
	"snmp-server",
	"service ",
	"end",
	"exit",
	"quit",
}

// baselineRuijieGE11 lead 实测取得的 9 行原始基线（权威快照）。
//
// 用途：工具采集到的基线与本快照比对，不一致时告警 ——
// 说明设备当前状态已偏离 lead 实测时的状态（有人改过、或解析有偏差），
// 此时"还原成功"的判据需要人工确认，不能盲目信任 diff。
var baselineRuijieGE11 = []string{
	"description 3D001",
	"switchport access vlan 318",
	"dot1x port-control auto",
	"dot1x mac-auth-bypass multi-user",
	"spanning-tree bpduguard enable",
	"spanning-tree portfast",
	"rldp port loop-detect shutdown-port",
	"switchport port-security binding 10.62.51.173",
	"switchport port-security",
}

// configLineKeywords 接口级配置行关键字前缀。
// 用于识别"疑似 PTY 折行产生的续行"（续行不会以这些关键字开头）。
var configLineKeywords = []string{
	"description",
	"switchport",
	"dot1x",
	"spanning-tree",
	"rldp",
	"shutdown",
	"no ",
	"ip ",
	"vlan ",
	"port ",
	"speed",
	"duplex",
	"negotiation",
	"mtu",
	"bandwidth",
	"load-interval",
	"storm-control",
	"link-flap",
	"loopback-detect",
	"mac-address",
	"authentication",
	"lldp",
	"power",
	"qos",
	"voice",
	"logging",
	"keepalive",
	"auto-negotiation",
	"media-type",
}

// sanitizeOutput 清洗设备回显：去 ANSI 控制序列、去 \r、去控制字符。
func sanitizeOutput(s string) string {
	s = ansiCSI.ReplaceAllString(s, "")
	s = ansiOSC.ReplaceAllString(s, "")
	s = ansiEsc.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "\r", "")
	s = ctrlChars.ReplaceAllString(s, "")
	return s
}

// normalizeIfaceKey 接口名归一（去空格 + 小写），用于匹配接口块。
func normalizeIfaceKey(name string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), " ", ""))
}

// isErrorLine 判断是否设备侧错误行。
func isErrorLine(l string) bool {
	for _, m := range deviceErrorMarkers {
		if strings.Contains(l, m) {
			return true
		}
	}
	return false
}

// isNoiseLine 判断是否噪声行（回显抬头/分隔符等）。
func isNoiseLine(l string) bool {
	switch {
	case l == "":
		return true
	case l == "!":
		return true
	case strings.HasPrefix(l, "Building configuration"):
		return true
	case strings.HasPrefix(l, "Current configuration"):
		return true
	case strings.HasPrefix(l, "Enter configuration commands"):
		return true
	case strings.HasPrefix(l, "End of configuration"):
		return true
	case l == "end":
		return true
	case l == "exit":
		return true
	case l == "quit":
		return true
	case strings.HasPrefix(l, "%") && !isErrorLine(l):
		return true
	}
	// 设备提示符（如 "Ruijie#"、"Ruijie(config-if-GigabitEthernet 1/1)#"）
	// —— scrapli 的 Result 默认带 prompt（StripPrompt 未生效），必须过滤，
	// 否则提示符会被当成配置行，基线 diff 永远不空。
	return looksLikePrompt(l)
}

// looksLikePrompt 判断一行是否像设备提示符。
//
// 形如 host# / host(config)# / host(config-if-GigabitEthernet 1/1)#。
// 判据：以 # 或 > 结尾；含括号时括号内只允许 view 名（字母/数字/下划线/连字符/空格）。
func looksLikePrompt(l string) bool {
	trimmed := strings.TrimRight(l, " \t")
	if trimmed == "" {
		return false
	}
	if !strings.HasSuffix(trimmed, "#") && !strings.HasSuffix(trimmed, ">") {
		return false
	}
	open := strings.Index(trimmed, "(")
	if open < 0 {
		// "Ruijie#" 形态
		return !strings.ContainsAny(trimmed, " \t")
	}
	closeIdx := strings.LastIndex(trimmed, ")")
	if closeIdx < open {
		return true
	}
	inner := trimmed[open+1 : closeIdx]
	for _, r := range inner {
		if !(r == ' ' || r == '-' || r == '_' ||
			(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false
		}
	}
	return true
}

// ifaceConfigBlock 从 running-config 接口视图输出中切出目标接口的配置块。
//
// 返回：配置行（已 trim）、错误行、疑似折行告警。
func ifaceConfigBlock(out, cliIface string) (cfgLines []string, errLines []string, warns []string) {
	s := sanitizeOutput(out)
	target := normalizeIfaceKey(cliIface)
	inBlock := false
	seen := false

	for _, raw := range strings.Split(s, "\n") {
		l := strings.TrimSpace(raw)
		if l == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(l), "interface ") {
			key := normalizeIfaceKey(l[len("interface "):])
			if inBlock {
				// 进入下一个接口块 → 目标接口块结束
				break
			}
			inBlock = key == target
			if inBlock {
				seen = true
			}
			continue
		}
		if !inBlock {
			continue
		}
		if l == "!" {
			// 接口块以 "!" 结束
			break
		}
		if isGlobalConfigLine(l) {
			// 设备实现异常未用 "!" 收尾时，遇全局配置段提前截断
			break
		}
		if isErrorLine(l) {
			errLines = append(errLines, l)
			continue
		}
		if isNoiseLine(l) {
			continue
		}
		if !looksLikeConfigLine(l) && len(cfgLines) > 0 {
			// 不以配置关键字开头 → 疑似 PTY 折行续行，合并到上一行
			warns = append(warns, fmt.Sprintf("疑似终端折行续行，已并入上一行: %q", l))
			cfgLines[len(cfgLines)-1] = cfgLines[len(cfgLines)-1] + " " + l
			continue
		}
		cfgLines = append(cfgLines, l)
	}
	if !seen {
		warns = append(warns, fmt.Sprintf("输出中未找到接口块 %q（接口名可能被设备改写或命令未生效）", cliIface))
	}
	return cfgLines, errLines, warns
}

// isGlobalConfigLine 判断是否是全局配置段行（用于接口块提前截断）。
func isGlobalConfigLine(l string) bool {
	lower := strings.ToLower(l)
	for _, p := range globalConfigLinePrefixes {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	return false
}

// looksLikeConfigLine 判断一行是否像接口级配置行。
func looksLikeConfigLine(l string) bool {
	lower := strings.ToLower(l)
	for _, kw := range configLineKeywords {
		if strings.HasPrefix(lower, kw) {
			return true
		}
	}
	return false
}

// normalizeForCompare 比较用归一：压缩连续空白 + trim。
func normalizeForCompare(l string) string {
	return strings.Join(strings.Fields(l), " ")
}

// diffConfig 基线 vs 当前配置行的集合差异（忽略顺序）。
// 返回 "缺失: xxx"（基线有、当前无）与 "多出: xxx"（当前有、基线无）。
func diffConfig(base, after []string) []string {
	baseSet := map[string]bool{}
	for _, l := range base {
		baseSet[normalizeForCompare(l)] = true
	}
	afterSet := map[string]bool{}
	for _, l := range after {
		afterSet[normalizeForCompare(l)] = true
	}

	var diff []string
	for _, l := range base {
		if !afterSet[normalizeForCompare(l)] {
			diff = append(diff, "缺失: "+l)
		}
	}
	for _, l := range after {
		if !baseSet[normalizeForCompare(l)] {
			diff = append(diff, "多出: "+l)
		}
	}
	return diff
}

// containsLine 判断配置行集合中是否含某行（归一比较）。
func containsLine(lines []string, want string) bool {
	w := normalizeForCompare(want)
	for _, l := range lines {
		if normalizeForCompare(l) == w {
			return true
		}
	}
	return false
}

// sortedIntKeys 返回 map[int]bool 的升序键列表（报告用）。
func sortedIntKeys(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

// parseVlanIDs 从 `show vlan` 输出中解析已存在的 VLAN ID 集合。
func parseVlanIDs(out string) map[int]bool {
	s := sanitizeOutput(out)
	res := map[int]bool{}
	for _, raw := range strings.Split(s, "\n") {
		l := strings.TrimSpace(raw)
		if l == "" {
			continue
		}
		fields := strings.Fields(l)
		if len(fields) == 0 {
			continue
		}
		// 形如 "318  318  ...  active" 或 "VLAN Name  Status Ports"
		var id int
		if _, err := fmt.Sscanf(fields[0], "%d", &id); err != nil {
			continue
		}
		if id < 1 || id > 4094 {
			continue
		}
		res[id] = true
	}
	return res
}
