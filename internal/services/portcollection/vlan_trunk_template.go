package portcollection

import (
	"fmt"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// Phase 124 (v1.51 PWX-01)：VLAN trunk/hybrid 扩展渲染模板。
//
// 新动作（PWX-01）：
//   - TrunkSetNative：trunk native/PVID VLAN
//   - TrunkSetAllowed：allowed VLAN 列表，AllowedMode ∈ {add, remove, replace}
//   - TrunkSetHybrid：hybrid 模式 tagged/untagged VLAN 列表（华为/H3C）
//
// 依据（命令语法，2026 真机口径与既有 set_access_vlan 三厂商分支同源）：
//   - 华为 VRP：port trunk pvid vlan X / port trunk allow-pass vlan <list> /
//     undo port trunk allow-pass vlan <list>；hybrid: port link-type hybrid +
//     port hybrid tagged vlan <list> / port hybrid untagged vlan <list>
//   - H3C Comware：port trunk pvid vlan X / port trunk permit vlan <list> /
//     undo port trunk permit vlan <list>；hybrid: port link-type hybrid +
//     port hybrid vlan <list>（tagged）/ port hybrid vlan untagged <list>（untagged）
//   - 锐捷 RGOS（Cisco 风格）：switchport trunk native vlan X /
//     switchport trunk allowed vlan {add|remove} <list> / replace 无关键字全量。
//     锐捷无 hybrid 端口模式 → TrunkSetHybrid 返回 ErrUnsupportedVendor。
//
// 厂商语义差异（PWX-01 验收 #3 注明）：
//   tagged/untagged 不可重叠 —— 华为 VRP 设备侧会拒绝重叠配置（两命令互斥，
//   设备报 Config error 兜底）；H3C Comware 设备侧不报错，untagged 表静默
//   优先于 tagged 表（port hybrid vlan 与 untagged 写入不同硬件表项），重叠时
//   实际转发表项与用户意图不一致且无告警。因此 H3C 的重叠校验必须在系统侧
//   前置（service 层 400 拦截为真相源），设备不兜底。

// PortAction Phase 124 新增动作。
const (
	TrunkSetNative  PortAction = "trunk_set_native"
	TrunkSetAllowed PortAction = "trunk_set_allowed"
	TrunkSetHybrid  PortAction = "trunk_set_hybrid"
)

// AllowedMode trunk_set_allowed 的列表变更模式。
const (
	AllowedModeAdd     = "add"     // 增量加入 allowed 列表
	AllowedModeRemove  = "remove"  // 从 allowed 列表移除
	AllowedModeReplace = "replace" // 全量替换 allowed 列表
)

// vlanListFormat 把 VLAN ID 列表格式化为命令参数串（空格分隔，华为/H3C 风格）。
func vlanListFormat(vlans []int) string {
	parts := make([]string, 0, len(vlans))
	for _, v := range vlans {
		parts = append(parts, fmt.Sprintf("%d", v))
	}
	return strings.Join(parts, " ")
}

// vlanListFormatCisco 逗号分隔（锐捷 Cisco 风格）。
func vlanListFormatCisco(vlans []int) string {
	parts := make([]string, 0, len(vlans))
	for _, v := range vlans {
		parts = append(parts, fmt.Sprintf("%d", v))
	}
	return strings.Join(parts, ",")
}

// validateVlanListForRender 渲染层防御性范围校验（service 层是 400 拦截真相源）。
func validateVlanListForRender(vlans []int) error {
	if len(vlans) == 0 {
		return fmt.Errorf("portcollection: vlan list is required")
	}
	for _, v := range vlans {
		if v < 1 || v > 4094 {
			return fmt.Errorf("portcollection: vlanId %d out of range 1-4094", v)
		}
	}
	return nil
}

// ====================== 华为 VRP ======================

// renderHuaweiTrunkSetNative 华为 trunk native VLAN：
//
//	interface <iface>
//	port trunk pvid vlan <nativeVlan>
func renderHuaweiTrunkSetNative(p PortTemplateParams) ([]string, error) {
	if p.NativeVLAN < 1 || p.NativeVLAN > 4094 {
		return nil, fmt.Errorf("portcollection: nativeVlan %d out of range 1-4094", p.NativeVLAN)
	}
	return []string{
		fmt.Sprintf("interface %s", p.InterfaceName),
		fmt.Sprintf("port trunk pvid vlan %d", p.NativeVLAN),
	}, nil
}

// renderHuaweiTrunkSetAllowed 华为 allowed 列表：
//
//	add:     port trunk allow-pass vlan <list>          （VRP allow-pass 是叠加语义）
//	remove:  undo port trunk allow-pass vlan <list>
//	replace: undo port trunk allow-pass vlan 全量不可行（无法枚举旧列表）→
//	         直接 allow-pass 新列表覆盖增量项；VRP 无 "全量替换" 关键字，
//	         replace 模式按叠加下发新列表并附注（见 service 层契约说明）。
func renderHuaweiTrunkSetAllowed(p PortTemplateParams) ([]string, error) {
	if err := validateVlanListForRender(p.AllowedVLANs); err != nil {
		return nil, err
	}
	switch p.AllowedMode {
	case AllowedModeRemove:
		return []string{
			fmt.Sprintf("interface %s", p.InterfaceName),
			fmt.Sprintf("undo port trunk allow-pass vlan %s", vlanListFormat(p.AllowedVLANs)),
		}, nil
	default: // add / replace（VRP allow-pass 天然叠加）
		return []string{
			fmt.Sprintf("interface %s", p.InterfaceName),
			fmt.Sprintf("port trunk allow-pass vlan %s", vlanListFormat(p.AllowedVLANs)),
		}, nil
	}
}

// renderHuaweiTrunkSetHybrid 华为 hybrid tagged/untagged：
//
//	interface <iface>
//	port link-type hybrid
//	port hybrid tagged vlan <taggedList>      （tagged 列表非空时）
//	port hybrid untagged vlan <untaggedList>  （untagged 列表非空时）
func renderHuaweiTrunkSetHybrid(p PortTemplateParams) ([]string, error) {
	if len(p.HybridTagged) == 0 && len(p.HybridUntagged) == 0 {
		return nil, fmt.Errorf("portcollection: hybrid tagged/untagged list required")
	}
	cmds := []string{fmt.Sprintf("interface %s", p.InterfaceName), "port link-type hybrid"}
	if len(p.HybridTagged) > 0 {
		if err := validateVlanListForRender(p.HybridTagged); err != nil {
			return nil, err
		}
		cmds = append(cmds, fmt.Sprintf("port hybrid tagged vlan %s", vlanListFormat(p.HybridTagged)))
	}
	if len(p.HybridUntagged) > 0 {
		if err := validateVlanListForRender(p.HybridUntagged); err != nil {
			return nil, err
		}
		cmds = append(cmds, fmt.Sprintf("port hybrid untagged vlan %s", vlanListFormat(p.HybridUntagged)))
	}
	return cmds, nil
}

// ====================== H3C Comware ======================

// renderH3CTrunkSetNative H3C trunk native VLAN：
//
//	interface <iface>
//	port trunk pvid vlan <nativeVlan>
func renderH3CTrunkSetNative(p PortTemplateParams) ([]string, error) {
	if p.NativeVLAN < 1 || p.NativeVLAN > 4094 {
		return nil, fmt.Errorf("portcollection: nativeVlan %d out of range 1-4094", p.NativeVLAN)
	}
	return []string{
		fmt.Sprintf("interface %s", p.InterfaceName),
		fmt.Sprintf("port trunk pvid vlan %d", p.NativeVLAN),
	}, nil
}

// renderH3CTrunkSetAllowed H3C allowed 列表（permit 关键字，华为是 allow-pass）：
//
//	add:     port trunk permit vlan <list>
//	remove:  undo port trunk permit vlan <list>
//	replace: 同华为，permit 叠加语义，无全量替换关键字
func renderH3CTrunkSetAllowed(p PortTemplateParams) ([]string, error) {
	if err := validateVlanListForRender(p.AllowedVLANs); err != nil {
		return nil, err
	}
	switch p.AllowedMode {
	case AllowedModeRemove:
		return []string{
			fmt.Sprintf("interface %s", p.InterfaceName),
			fmt.Sprintf("undo port trunk permit vlan %s", vlanListFormat(p.AllowedVLANs)),
		}, nil
	default:
		return []string{
			fmt.Sprintf("interface %s", p.InterfaceName),
			fmt.Sprintf("port trunk permit vlan %s", vlanListFormat(p.AllowedVLANs)),
		}, nil
	}
}

// renderH3CTrunkSetHybrid H3C hybrid：
//
//	interface <iface>
//	port link-type hybrid
//	port hybrid vlan <taggedList>            （tagged：Comware 默认带 tag）
//	port hybrid vlan untagged <untaggedList> （untagged：显式 untagged 关键字）
//
// ⚠ H3C untagged/tagged 两表项重叠时设备静默优先 untagged 不报错 —— 系统侧
// service 层重叠校验（hybridListsOverlap）是唯一防线，见文件头差异说明。
func renderH3CTrunkSetHybrid(p PortTemplateParams) ([]string, error) {
	if len(p.HybridTagged) == 0 && len(p.HybridUntagged) == 0 {
		return nil, fmt.Errorf("portcollection: hybrid tagged/untagged list required")
	}
	cmds := []string{fmt.Sprintf("interface %s", p.InterfaceName), "port link-type hybrid"}
	if len(p.HybridTagged) > 0 {
		if err := validateVlanListForRender(p.HybridTagged); err != nil {
			return nil, err
		}
		cmds = append(cmds, fmt.Sprintf("port hybrid vlan %s", vlanListFormat(p.HybridTagged)))
	}
	if len(p.HybridUntagged) > 0 {
		if err := validateVlanListForRender(p.HybridUntagged); err != nil {
			return nil, err
		}
		cmds = append(cmds, fmt.Sprintf("port hybrid vlan untagged %s", vlanListFormat(p.HybridUntagged)))
	}
	return cmds, nil
}

// ====================== 锐捷 RGOS（Cisco 风格） ======================

// renderRuijieTrunkSetNative 锐捷 trunk native VLAN：
//
//	interface <iface>
//	switchport trunk native vlan <nativeVlan>
func renderRuijieTrunkSetNative(p PortTemplateParams) ([]string, error) {
	if p.NativeVLAN < 1 || p.NativeVLAN > 4094 {
		return nil, fmt.Errorf("portcollection: nativeVlan %d out of range 1-4094", p.NativeVLAN)
	}
	return []string{
		fmt.Sprintf("interface %s", p.InterfaceName),
		fmt.Sprintf("switchport trunk native vlan %d", p.NativeVLAN),
	}, nil
}

// renderRuijieTrunkSetAllowed 锐捷 allowed 列表（Cisco IOS 关键字 add/remove/all）：
//
//	add:     switchport trunk allowed vlan add <list>
//	remove:  switchport trunk allowed vlan remove <list>
//	replace: switchport trunk allowed vlan <list>  （无关键字 = 全量替换）
//
// 锐捷是三家中唯一原生支持 replace 全量替换关键字的（华为/H3C 的 allow-pass /
// permit 均为叠加语义，replace 退化为叠加——契约中已注明）。
func renderRuijieTrunkSetAllowed(p PortTemplateParams) ([]string, error) {
	if err := validateVlanListForRender(p.AllowedVLANs); err != nil {
		return nil, err
	}
	keyword := ""
	switch p.AllowedMode {
	case AllowedModeAdd:
		keyword = " add "
	case AllowedModeRemove:
		keyword = " remove "
	default: // replace
		keyword = " "
	}
	return []string{
		fmt.Sprintf("interface %s", p.InterfaceName),
		fmt.Sprintf("switchport trunk allowed vlan%s%s", keyword, vlanListFormatCisco(p.AllowedVLANs)),
	}, nil
}

// RegisterTrunkTemplates 把 Phase 124 trunk/hybrid 渲染闭包注册进
// vendorPortTemplate 派发表（init 时机，避免改动既有表字面量）。
func RegisterTrunkTemplates() {
	vendorPortTemplate[models.VendorHuawei][TrunkSetNative] = renderHuaweiTrunkSetNative
	vendorPortTemplate[models.VendorHuawei][TrunkSetAllowed] = renderHuaweiTrunkSetAllowed
	vendorPortTemplate[models.VendorHuawei][TrunkSetHybrid] = renderHuaweiTrunkSetHybrid
	vendorPortTemplate[models.VendorH3C][TrunkSetNative] = renderH3CTrunkSetNative
	vendorPortTemplate[models.VendorH3C][TrunkSetAllowed] = renderH3CTrunkSetAllowed
	vendorPortTemplate[models.VendorH3C][TrunkSetHybrid] = renderH3CTrunkSetHybrid
	vendorPortTemplate[models.VendorRuijie][TrunkSetNative] = renderRuijieTrunkSetNative
	vendorPortTemplate[models.VendorRuijie][TrunkSetAllowed] = renderRuijieTrunkSetAllowed
	// 锐捷无 hybrid 端口模式：不注册 TrunkSetHybrid → RenderCommand 返回
	// ErrUnsupportedVendor（service 层翻译为 400）。
}

func init() {
	RegisterTrunkTemplates()
}
