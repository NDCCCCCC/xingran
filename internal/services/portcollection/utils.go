package portcollection

import (
	"regexp"
	"strings"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
)

// garbledShortFullPattern 匹配"短名 + 全称残 + 数字"拼接的乱码形式。
//
// 实测数据(2026-07-01 verify-format-unify 发现):
//   - "GEGigabitEthernet5/29"           → 短GE + 全GigabitEthernet + 数字5/29
//   - "GEngigabitethernet5/34"         → 短GE + 全残gigabitethernet + 数字
//   - "XGigabitEthernet0/0/1"          → 短XGE + 全GigabitEthernet + 数字
//   - "XGengigabitethernet0/49"        → 短XGE + 全残gigabitethernet + 数字
//   - "GEgabitetherngigabitethernet4/48" → 短GE + 多次重复gigabitethernet + 数字
//
// 限制: regex 不能完全避免与"合法短名(如 twentyfivegige)含 gige 子串"冲突,
//  真实数据中此重叠不存在(短名没有内嵌 gigabit),本规则安全。
//  不匹配(避免误伤):
//   - "GE0/0/1" (干净短名) — 无 gigabit 残
//   - "GE2/25" (干净短名) — 无 gigabit 残
//   - "GEet5/34" (2 行异常数据) — 中间是 "et" 不是 gigabit 残,留给 migration 187 单独处理
//
// 提取: capture[1] = 短名 (GE/XGE/HGE/FOE/FE), capture[2] = 数字+斜杠部分
// 模式: 短名 + 任意(含 gigabit/igabit/gige 关键词) + 数字
// 注意: 排除 TWE — 短名 `twentyfivegige` 内嵌 "gige" 子串,会与本规则冲突
// 排除 XG — 单独走显式前缀映射(见下)
var garbledShortFullPattern = regexp.MustCompile(`(?i)^(GE|XGE|HGE|FOE|FE)(.*?(?:igabit|gigabit|gige).*?)(\d.*)$`)

// garbledXGPrefix 显式处理 XGigabitEthernet0/0/1 → XGE0/0/1
// 这是 H3C 设备特定乱码(短名 XG 截断了 XGE 末尾 E,全名截断了 Gigabit 开头 G)
var garbledXGPrefix = regexp.MustCompile(`(?i)^XG(.*?(?:igabit|gigabit|gige).*?)(\d.*)$`)

// NormalizeInterfaceName 标准化接口名称
//
// 双向归一化策略:
//   - 输入全称(GigabitEthernet/TenGigE/FortyGigE/HundredGigE/FastEthernet/Loopback/Vlan/Vlanif)
//     → 输出对应短名(GE/XGE/FOE/HGE/FE/Loop/Vlan/Vlanif)
//   - 输入厂商短名(ge/gi/xe/twe/tw/te/fo/hge/fge/fa/vlan/vl/null)→ 输出对应全称
//     用于跨厂商数据 JOIN(sys_device_port_status 与 sys_device_mac_address 可能来自
//     不同厂商的采集命令,名称不一致)
//
// 历史背景:
//   - 2026-05-12 dot1x 路径已确认 OK(只覆盖全称→短名)
//   - 2026-06-29 物理链路 SQL view 失配暴露短名→全称方向缺漏
//   - 2026-07-01 port-mac-format-unify 提升为导出符号 NormalizeInterfaceName,
//     此前为内部小写 normalizeInterfaceName(同包 + lldp/port_classifier 各有一份副本)
func NormalizeInterfaceName(name string) string {
	name = strings.ReplaceAll(name, " ", "")

	// === 阶段 0 (2026-07-01 增强): 清理乱码形式(短名+全称残+数字)===
	// 策略: 显式 regex 匹配已知乱码形式,只保留短名 + 数字部分
	// 短名必须 ToUpper() 输出(原始大小写可能因 (?i) regex 而小写化)
	if matched := garbledXGPrefix.FindStringSubmatch(name); matched != nil {
		// XG 是 XGE 的截断短名,归一为标准短名 XGE
		return "XGE" + matched[2]
	}
	if matched := garbledShortFullPattern.FindStringSubmatch(name); matched != nil {
		shortPrefix := strings.ToUpper(matched[1]) // 短名 GE/XGE/HGE/FOE/FE
		digitsPart := matched[3]                   // 数字 + 斜杠 部分(中间全称残被丢弃)
		return shortPrefix + digitsPart
	}

	// === 阶段 1: 全称转简写:GigabitEthernet0/0/1 → GE0/0/1 ===
	fullToShort := map[string]string{
		"FastEthernet":    "FE",
		"GigabitEthernet": "GE",
		"TenGigE":         "XGE",
		"FortyGigE":       "FOE",
		"HundredGigE":     "HGE",
		"TwentyFiveGigE":  "TWE",
		"Vlanif":          "Vlanif",
		"Loopback":        "Loop",
	}

	for full, short := range fullToShort {
		if strings.HasPrefix(name, full) {
			suffix := name[len(full):]
			return short + suffix
		}
	}

	validPrefixes := []string{"FastEthernet", "GigabitEthernet", "TenGigE", "FortyGigE", "HundredGigE", "TwentyFiveGigE", "Vlan", "Vlanif", "Loopback", "NULL"}
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(name, prefix) {
			return name
		}
	}

	lowerName := strings.ToLower(name)

	type prefixMapping struct {
		short string
		full  string
	}
	// 短名 → 全称(按前缀长度倒序排,避免短前缀误匹配长前缀)
	// 例如 "tw" 与 "twe",先匹配 "twe" 再匹配 "tw"
	// 例如 "ge" 与 "gi",顺序无关(无重叠),但 "ge" 必须放在前面否则会被误匹配
	// 实际顺序敏感点:
	//   * "twe" 必须早于 "tw"  (tw1/1 在 twe 之后会被截取为 TwentyFiveGigE1/1,正确;
	//                            但 twe1/1 必须先匹配 twe 而非 tw)
	//   * "ge" 与 "gi" 无字符重叠,任意顺序都可
	prefixList := []prefixMapping{
		// 全称短写(长度 >= 3,优先级最高)
		{"gigabitethernet", "GigabitEthernet"},
		{"gigabitether", "GigabitEthernet"},
		{"hundredgige", "HundredGigE"},
		{"twentyfivegige", "TwentyFiveGigE"},
		{"fortygige", "FortyGigE"},
		{"fastethernet", "FastEthernet"},
		{"tengige", "TenGigE"},
		{"loopback", "Loopback"},
		// 厂商短名(按"长度倒序"原则:同一根前缀的更长形式在前)
		{"twe", "TwentyFiveGigE"}, // 25G 短名(部分 HP/Huawei 设备) — 必须早于 tw
		{"tw", "TwentyFiveGigE"},  // 25G 短名(部分 HP/Huawei 设备)
		{"et", "TwentyFiveGigE"},  // 25G 短名(部分 Huawei 设备,2026-07-01 加) — 必须早于 te
		{"hge", "HundredGigE"},    // 100G 短名(部分 Cisco/Huawei)
		{"fge", "FortyGigE"},      // 40G 短名(部分 Cisco Nexus)
		{"xe", "TenGigE"},         // 部分厂商 10G 短名(2026-06-29 补) — 与 te 无重叠,但放前保持逻辑顺序
		{"ge", "GigabitEthernet"}, // HP/H3C/Ruijie 短名(2026-06-29 补) — 必须与 gi 都能匹配
		{"gi", "GigabitEthernet"}, // Cisco/Huawei 短名
		{"te", "TenGigE"},         // Cisco/Huawei 短名
		{"fo", "FortyGigE"},       // Cisco 短名
		{"fa", "FastEthernet"},    // 100M 短名
		{"vlanif", "Vlanif"},
		{"vlan", "Vlan"},
		{"vl", "Vlan"},
		{"null", "NULL"},
	}

	for _, mapping := range prefixList {
		if strings.HasPrefix(lowerName, mapping.short) {
			suffix := lowerName[len(mapping.short):]
			// 严格要求 suffix 必须以数字开头才匹配短名,避免乱码(短名+字母碎片+数字)
			// 形式被错误识别为短名。例如 "geet5/34" 误判为 ge + et5/34 会把 et 变成
			// GigabitEthernet;实际是 GE + et (Huawei 25G 短名) 拼接的乱码,留给 migration 187
			if len(suffix) > 0 && suffix[0] >= '0' && suffix[0] <= '9' {
				return mapping.full + suffix
			}
			// 注意: 不再有"len(suffix)>0 不检查数字"的兜底返回(2026-07-01 移除)
			// 原代码有"if len(suffix) > 0 return full+suffix" 会导致
			// ge+et5/34 → GigabitEthernetet5/34 的错误。删除后这类乱码会落入
			// 最终 return name(原样),由 migration 187 显式处理。
		}
	}

	return name
}

// buildPortStatus 从接口信息构建端口状态
func buildPortStatus(deviceID, interfaceName string, iface InterfaceInfo, adminStatus, operStatus, description string, dot1xMap map[string]Dot1xInfo, securityMap map[string]PortSecurityInfo, collectionTime time.Time) *models.DevicePortStatus {
	portStatus := &models.DevicePortStatus{
		DeviceID:      deviceID,
		InterfaceName: interfaceName,
		AdminStatus:   adminStatus,
		OperStatus:    operStatus,
		Description:   description,
		VLAN:          iface.VLAN,
		Duplex:        iface.Duplex,
		Speed:         iface.Speed,
		PortType:      iface.PortType,
		CollectedAt:   collectionTime,
	}

	// 添加802.1X状态
	if dot1x, ok := dot1xMap[interfaceName]; ok {
		portStatus.Dot1xEnabled = dot1x.Enabled
		switch strings.ToUpper(dot1x.PortStatus) {
		case "AUTHORIZED":
			portStatus.Dot1xPortStatus = models.Dot1xStatusAuthorized
		case "UNAUTHORIZED":
			portStatus.Dot1xPortStatus = models.Dot1xStatusUnauthorized
		default:
			portStatus.Dot1xPortStatus = models.Dot1xStatusUnknown
		}
		applogger.Debugf("接口 %s 匹配到dot1x数据: Enabled=%v, Status=%s", interfaceName, dot1x.Enabled, dot1x.PortStatus)
	} else {
		applogger.Debugf("接口 %s 没有匹配到dot1x数据", interfaceName)
	}

	// 添加端口安全配置
	if security, ok := securityMap[interfaceName]; ok {
		portStatus.PortSecurityEnabled = security.Enabled

		var securityMode models.PortSecurityMode
		switch strings.ToUpper(security.SecurityMode) {
		case "RESTRICT":
			securityMode = models.PortSecurityModeRestrict
		case "PROTECT":
			securityMode = models.PortSecurityModeProtect
		case "SHUTDOWN":
			securityMode = models.PortSecurityModeShutdown
		default:
			securityMode = models.PortSecurityModeNone
		}
		portStatus.PortSecurityMode = securityMode

		if security.MaxMAC > 0 {
			portStatus.MaxMACCount = &security.MaxMAC
		}
		if security.CurrentMAC > 0 {
			portStatus.CurrentMACCount = &security.CurrentMAC
		}
	}

	return portStatus
}

// buildPortStatusFromDesc 从描述信息构建端口状态（降级方案）
func buildPortStatusFromDesc(deviceID, ifaceName string, desc InterfaceDescription, dot1xMap map[string]Dot1xInfo, securityMap map[string]PortSecurityInfo, collectionTime time.Time) *models.DevicePortStatus {
	portStatus := &models.DevicePortStatus{
		DeviceID:      deviceID,
		InterfaceName: ifaceName,
		AdminStatus:   desc.AdminStatus,
		OperStatus:    desc.OperStatus,
		Description:   desc.Description,
		CollectedAt:   collectionTime,
	}

	// 添加802.1X状态
	if dot1x, ok := dot1xMap[ifaceName]; ok {
		portStatus.Dot1xEnabled = dot1x.Enabled
		switch strings.ToUpper(dot1x.PortStatus) {
		case "AUTHORIZED":
			portStatus.Dot1xPortStatus = models.Dot1xStatusAuthorized
		case "UNAUTHORIZED":
			portStatus.Dot1xPortStatus = models.Dot1xStatusUnauthorized
		default:
			portStatus.Dot1xPortStatus = models.Dot1xStatusUnknown
		}
	}

	// 添加端口安全配置
	if security, ok := securityMap[ifaceName]; ok {
		portStatus.PortSecurityEnabled = security.Enabled

		var securityMode models.PortSecurityMode
		switch strings.ToUpper(security.SecurityMode) {
		case "RESTRICT":
			securityMode = models.PortSecurityModeRestrict
		case "PROTECT":
			securityMode = models.PortSecurityModeProtect
		case "SHUTDOWN":
			securityMode = models.PortSecurityModeShutdown
		default:
			securityMode = models.PortSecurityModeNone
		}
		portStatus.PortSecurityMode = securityMode

		if security.MaxMAC > 0 {
			portStatus.MaxMACCount = &security.MaxMAC
		}
		if security.CurrentMAC > 0 {
			portStatus.CurrentMACCount = &security.CurrentMAC
		}
	}

	return portStatus
}
