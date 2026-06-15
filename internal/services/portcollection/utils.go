package portcollection

import (
	"strings"
	"time"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
	applogger "github.com/ruoyi-next/ruoyi-go-backend/pkg/logger"
)

// normalizeInterfaceName 标准化接口名称
func normalizeInterfaceName(name string) string {
	name = strings.ReplaceAll(name, " ", "")

	// 全称转简写：GigabitEthernet0/0/1 → GE0/0/1
	fullToShort := map[string]string{
		"FastEthernet": "FE",
		"GigabitEthernet": "GE",
		"TenGigE": "XGE",
		"FortyGigE": "FOE",
		"HundredGigE": "HGE",
		"Vlanif": "Vlanif",
		"Loopback": "Loop",
	}

	for full, short := range fullToShort {
		if strings.HasPrefix(name, full) {
			suffix := name[len(full):]
			return short + suffix
		}
	}

	validPrefixes := []string{"FastEthernet", "GigabitEthernet", "TenGigE", "FortyGigE", "HundredGigE", "Vlan", "Vlanif", "Loopback", "NULL"}
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
	prefixList := []prefixMapping{
		{"fastethernet", "FastEthernet"},
		{"gigabitethernet", "GigabitEthernet"},
		{"tengige", "TenGigE"},
		{"fortygige", "FortyGigE"},
		{"hundredgige", "HundredGigE"},
		{"vlanif", "Vlanif"},
		{"loopback", "Loopback"},
		{"fa", "FastEthernet"},
		{"gi", "GigabitEthernet"},
		{"te", "TenGigE"},
		{"fo", "FortyGigE"},
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
