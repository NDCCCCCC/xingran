package main

// 12 个动作的定义表（锐捷 RS8607E 真机复测）。
//
// 三条铁律：
//  1. 下发命令一律来自 portcollection.RenderCommand（产品真相源），本文件不手写下发命令；
//  2. undo（还原）命令是"测试工具层"的反向命令，产品只有 shutdown / set_access_vlan
//     两个动作有回滚命令生成（internal/services/portwrite/safety.go），这里补齐其余动作，
//     语义与 safety.go 的 renderRollbackCommands 保持一致（厂商差异 + 基线感知）；
//  3. 每个动作都要给"判读要点"——锐捷专项断言靠它把证据摆到 lead 面前。

import (
	"fmt"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
)

// actionParams 单个动作的渲染参数。
type actionParams struct {
	CliIface     string // 展开后的设备 CLI 接口名（如 GigabitEthernet 1/1）
	VLAN         int
	MAC          string // 归一化大写冒号格式（AA:BB:CC:DD:EE:FF）
	IP           string
	Desc         string
	MaxMAC       int
	Dot1xLimit   *int
	BaselineVLAN int // 基线里的 access vlan（用于 set_access_vlan 还原判断）
}

// actionSpec 动作定义。
type actionSpec struct {
	// Name 动作名（CLI --action 取值）
	Name string
	// PortAction 产品动作标识（决定走哪套渲染模板）
	PortAction portcollection.PortAction
	// Build 构造渲染参数
	Build func(p actionParams) portcollection.PortTemplateParams
	// Undo 生成还原命令（接口视图内命令，不含 "interface X" 前缀）
	// baseline 为动作前采集到的接口配置行集合，用于基线感知还原
	Undo func(baseline []string, p actionParams) []string
	// Judgement 判读要点（锐捷专项断言提示）
	Judgement string
}

// actionRegistry 12 个动作。
var actionRegistry = map[string]actionSpec{
	"shutdown": {
		Name:       "shutdown",
		PortAction: portcollection.ActionShutdown,
		Build: func(p actionParams) portcollection.PortTemplateParams {
			return portcollection.PortTemplateParams{InterfaceName: p.CliIface}
		},
		Undo: func(baseline []string, p actionParams) []string {
			if containsLine(baseline, "shutdown") {
				// 基线本来就是 down → 重放基线行即可恢复，无需 undo
				return nil
			}
			return []string{"no shutdown"}
		},
		Judgement: "接口应进入 admin down；还原后 running-config 不应残留 shutdown 行",
	},
	"undo_shutdown": {
		Name:       "undo_shutdown",
		PortAction: portcollection.ActionUndoShutdown,
		Build: func(p actionParams) portcollection.PortTemplateParams {
			return portcollection.PortTemplateParams{InterfaceName: p.CliIface}
		},
		Undo: func(baseline []string, p actionParams) []string {
			if containsLine(baseline, "shutdown") {
				return []string{"shutdown"}
			}
			return nil
		},
		Judgement: "接口应从 admin down 恢复 up；还原后 running-config 与基线一致",
	},
	"description": {
		Name:       "description",
		PortAction: portcollection.ActionDescription,
		Build: func(p actionParams) portcollection.PortTemplateParams {
			return portcollection.PortTemplateParams{InterfaceName: p.CliIface, Description: p.Desc}
		},
		Undo: func(baseline []string, p actionParams) []string {
			if containsLine(baseline, "description "+p.Desc) {
				return nil
			}
			return []string{"no description"}
		},
		Judgement: "接口描述应被改写为测试描述；还原后应恢复基线描述 3D001",
	},
	"dot1x_enable": {
		Name:       "dot1x_enable",
		PortAction: portcollection.ActionDot1xEnable,
		Build: func(p actionParams) portcollection.PortTemplateParams {
			return portcollection.PortTemplateParams{InterfaceName: p.CliIface, Dot1xUserLimit: p.Dot1xLimit}
		},
		Undo: func(baseline []string, p actionParams) []string {
			// 锐捷：no dot1x port-control auto 会连带清除 default-user-limit，
			// 因此必须**无条件**下发 —— 否则动作新增的 default-user-limit 行会残留，
			// 还原后 diff 非空；随后由基线重放恢复 "dot1x port-control auto"。
			return []string{"no dot1x port-control auto"}
		},
		Judgement: "应出现 dot1x port-control auto + dot1x default-user-limit N；\n" +
			"           设备在 disable 时会自动清除 default-user-limit，还原后两者都不应残留",
	},
	"dot1x_disable": {
		Name:       "dot1x_disable",
		PortAction: portcollection.ActionDot1xDisable,
		Build: func(p actionParams) portcollection.PortTemplateParams {
			return portcollection.PortTemplateParams{InterfaceName: p.CliIface}
		},
		Undo: func(baseline []string, p actionParams) []string {
			if !containsLine(baseline, "dot1x port-control auto") {
				return nil
			}
			// 只有基线里本来就有 default-user-limit 行才恢复它；
			// 基线没有 → 恢复 port-control auto 即可（设备默认 unlimited 不落配置）。
			cmds := []string{"dot1x port-control auto"}
			for _, l := range baseline {
				if strings.HasPrefix(strings.ToLower(l), "dot1x default-user-limit") {
					cmds = append(cmds, l)
					break
				}
			}
			return cmds
		},
		Judgement: "应移除 dot1x port-control auto；还原后应恢复基线中的 dot1x 相关行",
	},
	"set_access_vlan": {
		Name:       "set_access_vlan",
		PortAction: portcollection.ActionSetAccessVLAN,
		Build: func(p actionParams) portcollection.PortTemplateParams {
			return portcollection.PortTemplateParams{InterfaceName: p.CliIface, VLANID: p.VLAN}
		},
		Undo: func(baseline []string, p actionParams) []string {
			// 目标 VLAN 与基线一致 → 重放基线行即可，无需 clear；
			// 否则先清 PVID（锐捷：no switchport access vlan），再由基线重放恢复基线 VLAN。
			for _, l := range baseline {
				if strings.HasPrefix(strings.ToLower(l), "switchport access vlan ") {
					if strings.TrimSpace(strings.TrimPrefix(l, "switchport access vlan")) == fmt.Sprint(p.VLAN) {
						return nil
					}
					return []string{"no switchport access vlan"}
				}
			}
			// 基线没有 access vlan 行 → 清掉动作引入的 PVID
			return []string{"no switchport access vlan"}
		},
		Judgement: "access vlan 应被改写为测试 VLAN；还原后应回到基线 VLAN",
	},
	"port_binding": {
		Name:       "port_binding",
		PortAction: portcollection.ActionPortBinding,
		Build: func(p actionParams) portcollection.PortTemplateParams {
			return portcollection.PortTemplateParams{
				InterfaceName: p.CliIface,
				BindOp:        "add",
				IPAddress:     p.IP,
				MACAddress:    p.MAC,
			}
		},
		Undo: func(baseline []string, p actionParams) []string {
			return []string{fmt.Sprintf("no switchport port-security binding %s %s", ruijieMAC(p.MAC), p.IP)}
		},
		Judgement: "锐捷专项①：`switchport port-security binding <mac> <ip>` 参数顺序是否被接受；\n" +
			"           是否触发端口保护性 shutdown（running-config 出现 shutdown / port-security 保护动作）；\n" +
			"           基线已有一条 IP-only binding（switchport port-security binding 10.62.51.173），\n" +
			"           还原后必须仍然存在且不多不少",
	},
	"portsec_enable": {
		Name:       "portsec_enable",
		PortAction: portcollection.PortSecEnable,
		Build: func(p actionParams) portcollection.PortTemplateParams {
			return portcollection.PortTemplateParams{InterfaceName: p.CliIface}
		},
		Undo: func(baseline []string, p actionParams) []string {
			if containsLine(baseline, "switchport port-security") {
				return nil
			}
			return []string{"no switchport port-security"}
		},
		Judgement: "端口安全应被启用；还原后回到基线的 port-security 状态",
	},
	"portsec_disable": {
		Name:       "portsec_disable",
		PortAction: portcollection.PortSecDisable,
		Build: func(p actionParams) portcollection.PortTemplateParams {
			return portcollection.PortTemplateParams{InterfaceName: p.CliIface}
		},
		Undo: func(baseline []string, p actionParams) []string {
			if !containsLine(baseline, "switchport port-security") {
				return nil
			}
			return []string{"switchport port-security"}
		},
		Judgement: "端口安全应被关闭；还原后应恢复基线的 switchport port-security 行",
	},
	"portsec_max_mac": {
		Name:       "portsec_max_mac",
		PortAction: portcollection.PortSecMaxMac,
		Build: func(p actionParams) portcollection.PortTemplateParams {
			return portcollection.PortTemplateParams{InterfaceName: p.CliIface, MaxMACCount: p.MaxMAC}
		},
		Undo: func(baseline []string, p actionParams) []string {
			return []string{"no switchport port-security maximum"}
		},
		Judgement: "应出现 switchport port-security maximum N；还原后该行应消失",
	},
	"portsec_sticky": {
		Name:       "portsec_sticky",
		PortAction: portcollection.PortSecSticky,
		Build: func(p actionParams) portcollection.PortTemplateParams {
			return portcollection.PortTemplateParams{InterfaceName: p.CliIface}
		},
		Undo: func(baseline []string, p actionParams) []string {
			return []string{"no switchport port-security mac-address sticky"}
		},
		Judgement: "锐捷专项②：`switchport port-security mac-address sticky` 是否被 RGOS 支持\n" +
			"           （看设备回显是否 % Invalid input / % Unknown command）",
	},
	"portsec_static_bind": {
		Name:       "portsec_static_bind",
		PortAction: portcollection.PortSecStaticBind,
		Build: func(p actionParams) portcollection.PortTemplateParams {
			return portcollection.PortTemplateParams{
				InterfaceName: p.CliIface,
				MACAddress:    p.MAC,
				VLANID:        p.VLAN,
			}
		},
		Undo: func(baseline []string, p actionParams) []string {
			return []string{fmt.Sprintf("no switchport port-security mac-address %s", ruijieMAC(p.MAC))}
		},
		Judgement: "锐捷专项③：`switchport port-security mac-address <mac> vlan <vlan>` 是否被支持\n" +
			"           （看设备回显是否 % Invalid input / % Unknown command）",
	},
}

// actionNames 有序动作名列表（报告与用法输出用）。
var actionNames = []string{
	"shutdown", "undo_shutdown", "description", "dot1x_enable", "dot1x_disable",
	"set_access_vlan", "port_binding", "portsec_enable", "portsec_disable",
	"portsec_max_mac", "portsec_sticky", "portsec_static_bind",
}

// ruijieMAC 把 MAC 归一为锐捷/Cisco 点分格式 aabb.ccdd.eeff。
func ruijieMAC(mac string) string {
	stripped := strings.NewReplacer(":", "", "-", "", ".", "", " ", "").Replace(mac)
	if len(stripped) != 12 {
		return mac
	}
	lower := strings.ToLower(stripped)
	return lower[0:4] + "." + lower[4:8] + "." + lower[8:12]
}
