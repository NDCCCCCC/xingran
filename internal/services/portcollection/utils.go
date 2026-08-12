package portcollection

import (
	"strconv"
	"strings"
	"time"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/pkg/normalize"
	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// NormalizeInterfaceName 标准化接口名称(转发到 pkg/normalize 单一真实源)。
//
// 实现已于 2026-07-01 port-mac-format-unify 下沉到 pkg/normalize.InterfaceName,
// 供 models/services/collectors 各层共用(解 import cycle)。本符号保留向后兼容,
// 所有历史调用点(portcollection 包内 + collectors + services)无需改动。
//
// 归一化目标: 大写短名 + 数字(GE0/0/1 / XGE1/0/1 / FE0/1 ...)。
// 详细策略、对称化契约、守卫逻辑见 pkg/normalize/iface.go + iface_test.go。
func NormalizeInterfaceName(name string) string {
	return normalize.InterfaceName(name)
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
		// 锐捷 MAX_USER 写入缓存 (enable 路径必读):
		//   - "unlimited" / 空 → nil (不写入或清零,语义等价)
		//   - 合法数字 → *int
		//   - 非数字 (异常设备输出) → nil + warn,enable 路径兜底 1,排障留痕
		// 往返不对称: disable 自动清 limit,enable 必须显式恢复 → 必须先采集写 DB。
		maxUser := strings.TrimSpace(dot1x.MaxUser)
		if maxUser != "" && maxUser != "unlimited" {
			if n, convErr := strconv.Atoi(maxUser); convErr == nil && n > 0 {
				v := n
				portStatus.Dot1xUserLimit = &v
			} else if convErr != nil {
				applogger.Warnf("接口 %s MAX_USER 非数字 %q → enable 兜底 1: %v", interfaceName, maxUser, convErr)
			}
		}
		applogger.Debugf("接口 %s 匹配到dot1x数据: Enabled=%v, Status=%s, MaxUser=%s", interfaceName, dot1x.Enabled, dot1x.PortStatus, maxUser)
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
