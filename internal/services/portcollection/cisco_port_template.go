package portcollection

import (
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// Phase 126 (v1.51 PWX-03)：Cisco IOS-XE 第 4 厂商端口写渲染。
//
// 覆盖 15 动作中的 14 个（TrunkSetHybrid 不注册 —— Cisco IOS 无 hybrid 端口
// 模式，那是华为 VRP / H3C Comware 私有概念；Cisco 的语音 VLAN/私有 VLAN 属
// 不同语义，deferred 至真机 UAT 后按需扩展）。
//
// 跨固件差异矩阵（验收 #3 注明依据）：
//   - IOS-XE 17.x / 16.x / IOS 15.x：本文件全部命令语法一致（switchport 系
//     命令自 IOS 12.x 起稳定，17.x 未引入 breaking change）。差异集中在
//     platform 层（scrapli cisco_iosxe driver 覆盖 15.x-17.x）而非命令层，
//     故命令矩阵不设固件分支。
//   - "IOS-XA" 非官方命名（卡面描述疑指 IOS-XR 早期版本）：IOS-XR 命令模型
//     完全不同（unconfigured/commit 两级提交 + 接口命名 Gi0/0/0/0 四段式 +
//     无 switchport port-security），语法差异大到必须独立模板 + 独立
//     scrapli platform（cisco_iosxr）+ commit 确认流程 → **XR deferred**，
//     待 XR 真机到位后在 Phase 129 UAT 中立项。
//   - 真机验证标注：本文件命令未经 Cisco 真机验证（Phase 129 UAT 挂起），
//     语法依据 Cisco IOS-XE 官方 command reference（switchport/port-security
//     系列为公开稳定语法）。

// renderCiscoDescription Cisco description：
//
//	interface <iface>
//	description <desc>
func renderCiscoDescription(p PortTemplateParams) ([]string, error) {
	if p.Description == "" {
		return nil, ErrDescriptionEmpty
	}
	if len(p.Description) > 80 {
		return nil, fmt.Errorf("%w: %d > 80", ErrDescriptionTooLong, len(p.Description))
	}
	return []string{
		fmt.Sprintf("interface %s", p.InterfaceName),
		fmt.Sprintf("description %s", p.Description),
	}, nil
}

// renderCiscoSetAccessVlan Cisco access VLAN（Cisco 原生语法）：
//
//	interface <iface>
//	switchport mode access
//	switchport access vlan <vlanId>
func renderCiscoSetAccessVlan(p PortTemplateParams) ([]string, error) {
	if p.VLANID < 1 || p.VLANID > 4094 {
		return nil, fmt.Errorf("portcollection: vlanId %d out of range 1-4094", p.VLANID)
	}
	return []string{
		fmt.Sprintf("interface %s", p.InterfaceName),
		"switchport mode access",
		fmt.Sprintf("switchport access vlan %d", p.VLANID),
	}, nil
}

// renderCiscoPortBinding Cisco 端口绑定（语义映射，非 Cisco 原生概念）：
//
//	Cisco IOS 无 IP-MAC 端口绑定（那是华为 user-bind / H3C IP-Source Guard
//	私有特性）。语义映射到 port-security 静态 MAC + VLAN：
//	  add:    switchport port-security mac-address <mac> vlan <vlan>
//	  remove: no switchport port-security mac-address <mac>
//	IP 地址部分 Cisco 无法直接绑定（IP Source Guard 需 DHCP snooping 基建），
//	仅落地 MAC+VLAN —— IP 字段记入审计 Extra 供追溯（契约注明）。
func renderCiscoPortBinding(p PortTemplateParams) ([]string, error) {
	if p.MACAddress == "" {
		return nil, fmt.Errorf("portcollection: macAddress is required for cisco port binding")
	}
	if p.VLANID < 1 || p.VLANID > 4094 {
		return nil, fmt.Errorf("portcollection: vlanId %d out of range 1-4094", p.VLANID)
	}
	mac := toCiscoMACFormat(p.MACAddress)
	if mac == "" {
		return nil, fmt.Errorf("portcollection: invalid mac address %q", p.MACAddress)
	}
	if p.BindOp == "add" {
		return []string{
			fmt.Sprintf("interface %s", p.InterfaceName),
			fmt.Sprintf("switchport port-security mac-address %s vlan %d", mac, p.VLANID),
		}, nil
	}
	return []string{
		fmt.Sprintf("interface %s", p.InterfaceName),
		fmt.Sprintf("no switchport port-security mac-address %s", mac),
	}, nil
}

// RegisterCiscoTemplates 注册 Cisco IOS-XE 渲染闭包（14/15 动作；hybrid deferred）。
func RegisterCiscoTemplates() {
	m := vendorPortTemplate[models.VendorCisco]
	if m == nil {
		m = make(map[PortAction]func(PortTemplateParams) ([]string, error))
		vendorPortTemplate[models.VendorCisco] = m
	}
	m[ActionShutdown] = func(p PortTemplateParams) ([]string, error) { return wrapInterface(p, "shutdown") }
	m[ActionUndoShutdown] = func(p PortTemplateParams) ([]string, error) { return wrapInterface(p, "no shutdown") }
	m[ActionDescription] = renderCiscoDescription
	// dot1x：Cisco IOS-XE 经典 802.1X 语法（dot1x pae authenticator +
	// authentication port-control auto，16.x/17.x 稳定；IBNS 2.0 policy-map
	// 高级场景 deferred）。
	m[ActionDot1xEnable] = func(p PortTemplateParams) ([]string, error) {
		return []string{
			fmt.Sprintf("interface %s", p.InterfaceName),
			"dot1x pae authenticator",
			"authentication port-control auto",
		}, nil
	}
	m[ActionDot1xDisable] = func(p PortTemplateParams) ([]string, error) {
		return []string{
			fmt.Sprintf("interface %s", p.InterfaceName),
			"no dot1x pae authenticator",
			"no authentication port-control auto",
		}, nil
	}
	m[ActionSetAccessVLAN] = renderCiscoSetAccessVlan
	m[ActionPortBinding] = renderCiscoPortBinding
	m[TrunkSetNative] = renderRuijieTrunkSetNative         // switchport trunk native vlan（Cisco 原生，锐捷同源）
	m[TrunkSetAllowed] = renderRuijieTrunkSetAllowed        // switchport trunk allowed vlan add|remove|全量（Cisco 原生）
	// TrunkSetHybrid 不注册：RenderCommand → ErrUnknownAction（vendor 差异契约）
	m[PortSecEnable] = func(p PortTemplateParams) ([]string, error) { return renderPortSec(p, models.VendorRuijie, PortSecEnable) }
	m[PortSecDisable] = func(p PortTemplateParams) ([]string, error) { return renderPortSec(p, models.VendorRuijie, PortSecDisable) }
	m[PortSecMaxMac] = func(p PortTemplateParams) ([]string, error) { return renderPortSec(p, models.VendorRuijie, PortSecMaxMac) }
	m[PortSecSticky] = func(p PortTemplateParams) ([]string, error) { return renderPortSec(p, models.VendorRuijie, PortSecSticky) }
	m[PortSecStaticBind] = func(p PortTemplateParams) ([]string, error) { return renderPortSec(p, models.VendorRuijie, PortSecStaticBind) }
}

func init() {
	RegisterCiscoTemplates()
}
