package main

// 12 个动作的定义表（锐捷 RS8607E 真机复测）。
//
// 三条铁律：
//  1. 下发命令一律来自 portcollection.RenderCommand（产品真相源），本文件不手写下发命令；
//  2. 还原语义统一为：
//     - Undo 只负责**删除动作引入的配置行**（用锐捷/Cisco 风格的 no 形式）；
//     - **恢复动作删除的配置行由基线重放统一负责**（还原引擎在接口视图逐行重放基线）。
//     这样"改写型"动作（description / set_access_vlan）与"删除型"动作（dot1x_disable）
//     的还原对称性天然正确，不需要各写一套反向模板；
//  3. 每个动作都要给"判读要点"——锐捷专项断言靠它把证据摆到 lead 面前。
//
// ★ 2026-11-02 真机复测修复（lead 反馈）：原先重放基线行时没有带 "interface X" 前缀，
//   命令在 (config)# 顶层下发被设备全部拒绝 → description / dot1x_disable /
//   set_access_vlan 三个动作还原失败、配置偏离基线。现由还原引擎统一组装
//   （buildInterfaceCmds 必带 interface 前缀 + exit），动作表只负责语义。

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
	// hybrid 动作参数（锐捷 TrunkSetHybrid，PR #44 新渲染形态）。
	HybridTagged   []int // → switchport hybrid allowed vlan tagged <list>
	HybridUntagged []int // → switchport hybrid allowed vlan untagged <list>
	HybridNative   int   // → switchport hybrid native vlan <id>（单值）
}

// actionSpec 动作定义。
type actionSpec struct {
	// Name 动作名（CLI --action 取值）
	Name string
	// PortAction 产品动作标识（决定走哪套渲染模板）
	PortAction portcollection.PortAction
	// Build 构造渲染参数
	Build func(p actionParams) portcollection.PortTemplateParams
	// Undo 生成"删除动作引入的配置行"的命令（接口视图内命令，不含 interface 前缀）。
	// baseline 为动作前采集到的接口配置行集合，用于基线感知还原。
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
				// 基线本来就是 down → 由基线重放恢复 shutdown 行
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
			// 动作只是把端口置 up（不引入配置行）；基线若本来是 down，由基线重放恢复。
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
			for _, l := range baseline {
				if strings.HasPrefix(strings.ToLower(l), "description ") {
					// 基线本来就有描述行 → 基线重放会覆盖回去，无需 no description
					return nil
				}
			}
			// 基线无描述行 → 动作引入的描述需要清掉
			return []string{"no description"}
		},
		Judgement: "接口描述应被改写为测试描述；还原后应恢复基线描述 3D001\n" +
			"           （真机复测重点：还原必须真的把 description 3D001 写回设备）",
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
			// 动作删掉 dot1x port-control auto（并连带清 default-user-limit /
			// mac-auth-bypass multi-user）→ 三行全部由基线重放恢复，Undo 无需动作。
			return nil
		},
		Judgement: "应移除 dot1x port-control auto；还原后必须补回基线里的 dot1x 三行\n" +
			"           （port-control auto / default-user-limit / mac-auth-bypass multi-user）",
	},
	"set_access_vlan": {
		Name:       "set_access_vlan",
		PortAction: portcollection.ActionSetAccessVLAN,
		Build: func(p actionParams) portcollection.PortTemplateParams {
			return portcollection.PortTemplateParams{InterfaceName: p.CliIface, VLANID: p.VLAN}
		},
		Undo: func(baseline []string, p actionParams) []string {
			// 基线的 access vlan 行会被基线重放直接覆盖 → 无需 clear；
			// 但若基线没有 access vlan 行，动作引入的 PVID 必须清掉。
			for _, l := range baseline {
				if strings.HasPrefix(strings.ToLower(l), "switchport access vlan ") {
					return nil
				}
			}
			return []string{"no switchport access vlan"}
		},
		Judgement: "access vlan 应被改写为测试 VLAN；还原后必须回到基线 VLAN\n" +
			"           （真机复测重点：还原必须真的把 switchport access vlan 318 写回设备）",
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
			// 动作删掉 switchport port-security 行 → 由基线重放恢复。
			return nil
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
		Judgement: "应出现 switchport port-security maximum N；还原后该行应消失\n" +
			"           （探针已确认设备支持：<1-128> Maximum addrs）",
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

	// hybrid（锐捷 TrunkSetHybrid，PR #44 新渲染形态）。
	//
	// 注意：**不加入 actionNames** —— 批次模式（--batch）不跑 hybrid：
	//   hybrid 会把端口从 access 切到 hybrid 模式（改变转发模式），且需要显式
	//   指定 VLAN 参数，与「12 个固定动作」的批次语义不同，必须由人显式指定口与
	//   VLAN 后单独跑。
	"hybrid": {
		Name:       "hybrid",
		PortAction: portcollection.TrunkSetHybrid,
		Build: func(p actionParams) portcollection.PortTemplateParams {
			return portcollection.PortTemplateParams{
				InterfaceName:  p.CliIface,
				HybridTagged:   p.HybridTagged,
				HybridUntagged: p.HybridUntagged,
				NativeVLAN:     p.HybridNative,
			}
		},
		Undo: func(baseline []string, p actionParams) []string {
			// 还原语义：undo 只删「动作引入的配置行」；「恢复动作删除的配置行」由
			// 基线重放负责（基线若有 `switchport access vlan N` 行，重放会补回；
			// 基线若没有这一行 ⇒ **不下发** —— 否则凭空多出一行制造 diff）。
			var cmds []string
			if len(p.HybridTagged) > 0 {
				cmds = append(cmds, fmt.Sprintf("no switchport hybrid allowed vlan tagged %s", ruijieVlanList(p.HybridTagged)))
			}
			if len(p.HybridUntagged) > 0 {
				cmds = append(cmds, fmt.Sprintf("no switchport hybrid allowed vlan untagged %s", ruijieVlanList(p.HybridUntagged)))
			}
			if p.HybridNative > 0 {
				cmds = append(cmds, fmt.Sprintf("no switchport hybrid native vlan %d", p.HybridNative))
			}
			// ★ 端口模式回退：渲染层 undo 不覆盖端口模式（lead 裁定：不在本动作范围，
			//   safety.go 未扩写），但真机实跑必须把口切回 access ⇒ 还原显式下发。
			//   设备回 Invalid input 时兜底 `default switchport mode access`；
			//   若 running-config 仍残留 `switchport mode hybrid` 行，由差异驱动修复
			//   走 `no switchport mode hybrid`（negateConfigLine 已映射该行）。
			cmds = append(cmds, "switchport mode access")
			return cmds
		},
		Judgement: "锐捷专项④：hybrid 三种形态是否被设备接受 ——\n" +
			"           `switchport mode hybrid`（前置切模式）+\n" +
			"           `switchport hybrid allowed vlan tagged <list>` / `... untagged <list>` /\n" +
			"           `switchport hybrid native vlan <id>`（单值）\n" +
			"           ★ 多 untagged 是否被接受 = PR #44 撤销「多 untagged 拒绝」后的行为验证\n" +
			"           ★ 还原后端口模式必须回到 access（running-config 不应残留 hybrid 行）",
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

// ruijieVlanList 把 VLAN ID 列表格式化为锐捷/Cisco 逗号分隔形态（30,40）。
func ruijieVlanList(vlans []int) string {
	parts := make([]string, 0, len(vlans))
	for _, v := range vlans {
		parts = append(parts, fmt.Sprintf("%d", v))
	}
	return strings.Join(parts, ",")
}

// negateConfigLine 由"多出的配置行"推导对应的否定命令（差异驱动修复用）。
//
// 返回 ok=false 表示该行无法安全否定（例如 spanning-tree / rldp 这类无 no 形式的行）。
func negateConfigLine(line string) (string, bool) {
	l := strings.TrimSpace(line)
	lower := strings.ToLower(l)
	switch {
	case lower == "shutdown":
		return "no shutdown", true
	case strings.HasPrefix(lower, "description "):
		return "no description", true
	case strings.HasPrefix(lower, "switchport access vlan "):
		return "no switchport access vlan", true
	case strings.HasPrefix(lower, "switchport port-security maximum"):
		return "no switchport port-security maximum", true
	case strings.HasPrefix(lower, "switchport port-security mac-address sticky"):
		return "no switchport port-security mac-address sticky", true
	case strings.HasPrefix(lower, "switchport port-security mac-address "):
		return "no " + l, true
	case strings.HasPrefix(lower, "switchport port-security binding "):
		return "no " + l, true
	case strings.HasPrefix(lower, "switchport port-security"):
		return "no switchport port-security", true
	case strings.HasPrefix(lower, "dot1x port-control auto"):
		return "no dot1x port-control auto", true
	case strings.HasPrefix(lower, "dot1x default-user-limit"):
		return "no dot1x default-user-limit", true
	case strings.HasPrefix(lower, "dot1x mac-auth-bypass"):
		return "no dot1x mac-auth-bypass", true
	case strings.HasPrefix(lower, "switchport mode "):
		return "no " + l, true
	}
	return "", false
}
