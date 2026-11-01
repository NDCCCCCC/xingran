package portcollection

import (
	"fmt"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// Phase 125 (v1.51 PWX-02)：端口安全渲染模板。
//
// 新动作：
//   - PortSecEnable / PortSecDisable：port-security 启停
//   - PortSecMaxMac：max-mac-num 上限设置
//   - PortSecSticky：sticky MAC 模式（ learn 的 MAC 写入 running-config）
//   - PortSecStaticBind：静态 MAC 1:1 绑定（macAddress + vlan 绑定接口）
//
// 厂商语法依据（与既有三厂商分支同源口径）：
//   - 华为 VRP：port-security enable / undo port-security enable /
//     port-security max-mac-num N / port-security mac-address sticky /
//     port-security mac-address sticky <mac> vlan <vlan>
//   - H3C Comware：port-security enable / undo port-security enable /
//     port-security max-mac-count N / port-security mac-address sticky /
//     port-security mac-address security <mac> vlan <vlan>
//   - 锐捷 RGOS（Cisco 风格）：switchport port-security /
//     no switchport port-security / switchport port-security maximum N /
//     switchport port-security mac-address sticky /
//     switchport port-security mac-address <mac> vlan <vlan>
//
// sticky 行为差异（PWX-02 注明）：
//   - 锐捷/Ruijie 与 Cisco 一致：sticky MAC 立即写入 running-config，save 后持久
//   - 华为 VRP：mac-address sticky 把已 learn 的动态 MAC 转为 sticky 类型，但
//     不自动 save；且 VRP 的 sticky 转换是端口级一次性的
//   - H3C Comware：sticky 与 static 等价存储（security MAC），接口 down/up 后保留
//   三家共同点：sticky 不改变 max-mac 上限语义，均需先 enable port-security
//   （renderer 在 enable 分支前置 enable 命令，规避依赖顺序错误）。

// PortAction Phase 125 新增动作。
const (
	PortSecEnable     PortAction = "portsec_enable"
	PortSecDisable    PortAction = "portsec_disable"
	PortSecMaxMac     PortAction = "portsec_max_mac"
	PortSecSticky     PortAction = "portsec_sticky"
	PortSecStaticBind PortAction = "portsec_static_bind"
)

// renderPortSec 华为/H3C 端口安全命令拼装（命令体按厂商+动作展开）。
func renderPortSec(p PortTemplateParams, vendor models.DeviceVendor, action PortAction) ([]string, error) {
	switch action {
	case PortSecEnable:
		if vendor == models.VendorRuijie {
			return wrapInterface(p, "switchport port-security")
		}
		return wrapInterface(p, "port-security enable")
	case PortSecDisable:
		if vendor == models.VendorRuijie {
			return wrapInterface(p, "no switchport port-security")
		}
		return wrapInterface(p, "undo port-security enable")
	case PortSecMaxMac:
		if p.MaxMACCount < 1 || p.MaxMACCount > 4096 {
			return nil, fmt.Errorf("portcollection: maxMacCount %d out of range 1-4096", p.MaxMACCount)
		}
		switch vendor {
		case models.VendorRuijie:
			return wrapInterface(p, fmt.Sprintf("switchport port-security maximum %d", p.MaxMACCount))
		case models.VendorH3C:
			return wrapInterface(p, fmt.Sprintf("port-security max-mac-count %d", p.MaxMACCount))
		default: // 华为
			return wrapInterface(p, fmt.Sprintf("port-security max-mac-num %d", p.MaxMACCount))
		}
	case PortSecSticky:
		switch vendor {
		case models.VendorRuijie:
			return wrapInterface(p, "switchport port-security mac-address sticky")
		default: // 华为 / H3C
			return wrapInterface(p, "port-security mac-address sticky")
		}
	case PortSecStaticBind:
		if p.MACAddress == "" {
			return nil, fmt.Errorf("portcollection: macAddress is required for static bind")
		}
		if p.VLANID < 1 || p.VLANID > 4094 {
			return nil, fmt.Errorf("portcollection: vlanId %d out of range 1-4094", p.VLANID)
		}
		mac := toHuaweiH3CMACFormat(p.MACAddress)
		if vendor == models.VendorRuijie {
			// 锐捷用点分 Cisco 格式
			mac = toCiscoMACFormat(p.MACAddress)
			return wrapInterface(p, fmt.Sprintf("switchport port-security mac-address %s vlan %d", mac, p.VLANID))
		}
		if vendor == models.VendorH3C {
			return wrapInterface(p, fmt.Sprintf("port-security mac-address security %s vlan %d", mac, p.VLANID))
		}
		return wrapInterface(p, fmt.Sprintf("port-security mac-address sticky %s vlan %d", mac, p.VLANID))
	}
	return nil, fmt.Errorf("portcollection: unknown port-security action %s", action)
}

// toCiscoMACFormat 归一 MAC 为 Cisco/锐捷点分 `aabb.ccdd.eeff` 格式。
func toCiscoMACFormat(mac string) string {
	normalized := localNormalizeMACAddress(mac)
	if normalized == "" {
		return ""
	}
	hex := strings.ReplaceAll(normalized, ":", "")
	if len(hex) != 12 {
		return ""
	}
	return strings.ToLower(hex[0:4] + "." + hex[4:8] + "." + hex[8:12])
}

// RegisterPortSecTemplates 注册 Phase 125 端口安全渲染闭包。
func RegisterPortSecTemplates() {
	vendorPortTemplate[models.VendorHuawei][PortSecEnable] = func(p PortTemplateParams) ([]string, error) { return renderPortSec(p, models.VendorHuawei, PortSecEnable) }
	vendorPortTemplate[models.VendorHuawei][PortSecDisable] = func(p PortTemplateParams) ([]string, error) { return renderPortSec(p, models.VendorHuawei, PortSecDisable) }
	vendorPortTemplate[models.VendorHuawei][PortSecMaxMac] = func(p PortTemplateParams) ([]string, error) { return renderPortSec(p, models.VendorHuawei, PortSecMaxMac) }
	vendorPortTemplate[models.VendorHuawei][PortSecSticky] = func(p PortTemplateParams) ([]string, error) { return renderPortSec(p, models.VendorHuawei, PortSecSticky) }
	vendorPortTemplate[models.VendorHuawei][PortSecStaticBind] = func(p PortTemplateParams) ([]string, error) { return renderPortSec(p, models.VendorHuawei, PortSecStaticBind) }
	vendorPortTemplate[models.VendorH3C][PortSecEnable] = func(p PortTemplateParams) ([]string, error) { return renderPortSec(p, models.VendorH3C, PortSecEnable) }
	vendorPortTemplate[models.VendorH3C][PortSecDisable] = func(p PortTemplateParams) ([]string, error) { return renderPortSec(p, models.VendorH3C, PortSecDisable) }
	vendorPortTemplate[models.VendorH3C][PortSecMaxMac] = func(p PortTemplateParams) ([]string, error) { return renderPortSec(p, models.VendorH3C, PortSecMaxMac) }
	vendorPortTemplate[models.VendorH3C][PortSecSticky] = func(p PortTemplateParams) ([]string, error) { return renderPortSec(p, models.VendorH3C, PortSecSticky) }
	vendorPortTemplate[models.VendorH3C][PortSecStaticBind] = func(p PortTemplateParams) ([]string, error) { return renderPortSec(p, models.VendorH3C, PortSecStaticBind) }
	vendorPortTemplate[models.VendorRuijie][PortSecEnable] = func(p PortTemplateParams) ([]string, error) { return renderPortSec(p, models.VendorRuijie, PortSecEnable) }
	vendorPortTemplate[models.VendorRuijie][PortSecDisable] = func(p PortTemplateParams) ([]string, error) { return renderPortSec(p, models.VendorRuijie, PortSecDisable) }
	vendorPortTemplate[models.VendorRuijie][PortSecMaxMac] = func(p PortTemplateParams) ([]string, error) { return renderPortSec(p, models.VendorRuijie, PortSecMaxMac) }
	vendorPortTemplate[models.VendorRuijie][PortSecSticky] = func(p PortTemplateParams) ([]string, error) { return renderPortSec(p, models.VendorRuijie, PortSecSticky) }
	vendorPortTemplate[models.VendorRuijie][PortSecStaticBind] = func(p PortTemplateParams) ([]string, error) { return renderPortSec(p, models.VendorRuijie, PortSecStaticBind) }
}

func init() {
	RegisterPortSecTemplates()
}
