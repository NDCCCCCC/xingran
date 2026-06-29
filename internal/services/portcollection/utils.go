package portcollection

import (
	"strings"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
)

// normalizeInterfaceName 标准化接口名称
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
//   - 2026-06-29 物理链路 SQL view 失配暴露短名→全称方向缺漏:
//     * HP / H3C 短名 "ge2/25" 未识别
//     * 部分厂商 10G 短名 "xe2/25" 未识别
//     * 25G 短名 "tw"/"twe" 未识别(实测个别 HP/Huawei 设备使用)
//     * 100G 短名 "hge" 未识别(短名已是 HGE 但被 prefixList 漏)
//     * 全称→短名 HGE 已支持,但反向 HundredGigE 走 fastethernet 之外的全称分支返回原样
//     * Cisco 短名 "gi" 已支持,但 Cisco/Nexus 短名 "ge" 缺
func normalizeInterfaceName(name string) string {
	name = strings.ReplaceAll(name, " ", "")

	// 全称转简写：GigabitEthernet0/0/1 → GE0/0/1
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
			if len(suffix) > 0 && (suffix[0] >= '0' && suffix[0] <= '9') {
				return mapping.full + suffix
			}
			if len(suffix) > 0 {
				return mapping.full + suffix
			}
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
