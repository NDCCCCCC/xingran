package portcollection

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// PortAction 端口写操作类型（Phase 50 锁定；Phase 52 operlog action 字段直采此值）。
type PortAction string

const (
	ActionShutdown      PortAction = "shutdown"
	ActionUndoShutdown  PortAction = "undo_shutdown"
	ActionDescription   PortAction = "description"
	ActionDot1xEnable   PortAction = "dot1x_enable"
	ActionDot1xDisable  PortAction = "dot1x_disable"
	// ActionSetAccessVLAN Phase 56 v1.20.1 新增：设置 access 端口 PVID（接口视图下下发）。
	ActionSetAccessVLAN PortAction = "set_access_vlan"
	// ActionPortBinding Phase 56 v1.20.1 新增：IP/MAC 端口绑定 (add / remove 由 PortTemplateParams.BindOp 区分)。
	ActionPortBinding PortAction = "port_binding"
)

// String 返回 PortAction 的字符串值，便于 operlog / 日志记录。
func (a PortAction) String() string { return string(a) }

// PortTemplateParams 模板渲染参数。
//
// InterfaceName  所有 action 必填（RenderCommand 入口校验）。
// Description    仅 ActionDescription 使用；其他 action 忽略。
// VLANID         仅 ActionSetAccessVLAN 使用（1-4094；renderer 做防御性范围校验）。
// BindOp         仅 ActionPortBinding 使用（"add" 或 "remove"；service 层是 400 拦截源）。
// IPAddress      仅 ActionPortBinding 使用。
// MACAddress     仅 ActionPortBinding 使用（可选；Huawei/H3C 用，Ruijie 接受）。
// Dot1xUserLimit 仅 Ruijie.ActionDot1xEnable 使用；从采集缓存读出 (MAX_USER)，
//
//	nil (设备 "unlimited" 或尚未采集) 时兜底 1。保证 disable → enable 往返
//	对称：锐捷 disable 自动清除 default-user-limit，enable 必须显式恢复。
type PortTemplateParams struct {
	InterfaceName  string
	Description    string
	VLANID         int
	BindOp         string
	IPAddress      string
	MACAddress     string
	Dot1xUserLimit *int
}

// 哨兵错误（Phase 51 service 翻译为 HTTP 400 / 404 / 422 等业务码）。
var (
	ErrUnsupportedVendor  = errors.New("portcollection: vendor not supported for write operations")
	ErrUnknownAction      = errors.New("portcollection: unknown port action")
	ErrEmptyInterfaceName = errors.New("portcollection: interface name is required")
	ErrDescriptionEmpty   = errors.New("portcollection: description is required for description action")
	ErrDescriptionTooLong = errors.New("portcollection: description exceeds 80 characters")
)

// vendorPortTemplate 内部派发表：vendor × action → 命令序列渲染闭包。
//
// D-08: Huawei VRP 与 H3C Comware 命令同源（VRP 血统），仅 scrapligo platform 配置不同。
// D-01: 不收录 VendorMaipu（v1.19 OUT-OF-SCOPE）。
// D-07: 锐捷 dot1x / description 走 Cisco 风格显式进 interface view。
var vendorPortTemplate = map[models.DeviceVendor]map[PortAction]func(PortTemplateParams) ([]string, error){
	models.VendorHuawei: {
		ActionShutdown:     func(p PortTemplateParams) ([]string, error) { return wrapInterface(p, "shutdown") },
		ActionUndoShutdown: func(p PortTemplateParams) ([]string, error) { return wrapInterface(p, "undo shutdown") },
		ActionDescription:  renderH3CDescription,
		// 华为 VRP 模板式 802.1X：接口下应用 authentication-profile（profile 名约定为 dot1x）。
		// 不同于早期版本接口下的 dot1x enable，VRP 新版本统一走 authentication-profile 模板，
		// profile 名需与设备上预配置的 authentication-profile 名称一致（本系统约定 dot1x）。
		ActionDot1xEnable:  func(p PortTemplateParams) ([]string, error) { return wrapInterface(p, "authentication-profile dot1x") },
		ActionDot1xDisable: func(p PortTemplateParams) ([]string, error) { return wrapInterface(p, "undo authentication-profile dot1x") },
		ActionSetAccessVLAN: renderHuaweiSetAccessVlan,
		// PortBinding 由 BindOp add/remove 分派；任何非 add 落到 remove（保守默认）。
		ActionPortBinding: func(p PortTemplateParams) ([]string, error) {
			if p.BindOp == "add" {
				return renderHuaweiPortBindingAdd(p)
			}
			return renderHuaweiPortBindingRemove(p)
		},
	},
	models.VendorH3C: {
		ActionShutdown:     func(p PortTemplateParams) ([]string, error) { return wrapInterface(p, "shutdown") },
		ActionUndoShutdown: func(p PortTemplateParams) ([]string, error) { return wrapInterface(p, "undo shutdown") },
		ActionDescription:  renderH3CDescription,
		ActionDot1xEnable:  func(p PortTemplateParams) ([]string, error) { return wrapInterface(p, "dot1x enable") },
		ActionDot1xDisable: func(p PortTemplateParams) ([]string, error) { return wrapInterface(p, "undo dot1x enable") },
		ActionSetAccessVLAN: renderH3CSetAccessVlan,
		ActionPortBinding: func(p PortTemplateParams) ([]string, error) {
			if p.BindOp == "add" {
				return renderH3CPortBindingAdd(p)
			}
			return renderH3CPortBindingRemove(p)
		},
	},
	models.VendorRuijie: {
		ActionShutdown:     func(p PortTemplateParams) ([]string, error) { return wrapInterface(p, "shutdown") },
		ActionUndoShutdown: func(p PortTemplateParams) ([]string, error) { return wrapInterface(p, "no shutdown") },
		ActionDescription:  renderRuijieDescription,
		ActionDot1xEnable:  renderRuijieDot1xEnable,
		ActionDot1xDisable: renderRuijieDot1xDisable,
		ActionSetAccessVLAN: renderRuijieSetAccessVlan,
		ActionPortBinding: func(p PortTemplateParams) ([]string, error) {
			if p.BindOp == "add" {
				return renderRuijiePortBindingAdd(p)
			}
			return renderRuijiePortBindingRemove(p)
		},
	},
}

// wrapInterface 在指定接口视图下下发单条命令。
//
// 华为 VRP / H3C Comware / 锐捷 RGOS 的 shutdown / dot1x 等命令都必须在 interface view
// 下执行；scrapli SendConfig 仅进入系统视图（config mode），必须显式下发 `interface <name>`
// 进入接口视图后命令才会被设备接受。缺少此前缀会导致设备返回 "Unrecognized command" 而
// 端口状态不变 —— 这是端口写命令"必超时 + 端口无变化"的根因（shutdown/undo/dot1x 早期
// 模板漏写 interface 前缀，仅 description/锐捷 dot1x 写了）。
func wrapInterface(p PortTemplateParams, cmd string) ([]string, error) {
	return []string{fmt.Sprintf("interface %s", p.InterfaceName), cmd}, nil
}

// toHuaweiH3CMACFormat 将任意 MAC 输入归一为 Huawei / H3C 期望的 `AA-BB-CC-DD-EE-FF` 格式。
//
// 与 internal/services/mac_normalize.go NormalizeMACAddress 规则一致，避免循环依赖（此包
// 不直接 import parent services）；归一失败（空输入或非法 hex 长度）返回 ""，
// 调用方需自行判断是否跳过 mac 段。
func toHuaweiH3CMACFormat(mac string) string {
	normalized := localNormalizeMACAddress(mac)
	if normalized == "" {
		return ""
	}
	return strings.ReplaceAll(normalized, ":", "-")
}

// localNormalizeMACAddress 将任意 MAC 格式归一为大写冒号格式 `AA:BB:CC:DD:EE:FF`。
//
// 与 internal/services/mac_normalize.go 的 NormalizeMACAddress 规则完全一致——本地副本的
// 唯一目的是规避 import cycle（parent services → portcollection；portcollection 不能反向
// import services）。任何字段含义变更须同步两边。
func localNormalizeMACAddress(input string) string {
	if input == "" {
		return ""
	}
	stripped := strings.NewReplacer(":", "", "-", "", ".", "", " ", "").Replace(input)
	stripped = strings.ToUpper(stripped)
	if !localIsHexOnlyMACPattern.MatchString(stripped) {
		return ""
	}
	var b strings.Builder
	for i := 0; i < 12; i += 2 {
		if i > 0 {
			b.WriteByte(':')
		}
		b.WriteString(stripped[i : i+2])
	}
	return b.String()
}

// localIsHexOnlyMACPattern 12 位 hex 字符中间态校验，与 parent services 保持一致。
var localIsHexOnlyMACPattern = regexp.MustCompile(`^[0-9A-F]{12}$`)

// toRuijieMACFormat 把已归一为大写冒号格式的 MAC 转成 Ruijie / Cisco `H.H.H` 形式，
// 即 `aabb.ccdd.eeff`（3 组 4 位 hex 字符 + 点分隔，全小写）。
//
// 输入必须是 localNormalizeMACAddress 已成功归一的 17 字符串 `AA:BB:CC:DD:EE:FF`。
func toRuijieMACFormat(normalized string) string {
	// 去掉冒号，截 4-4-4 拼为 aabb.ccdd.eeff
	hex := strings.ToLower(strings.ReplaceAll(normalized, ":", ""))
	return hex[0:4] + "." + hex[4:8] + "." + hex[8:12]
}

// VendorExitViewCmd 返回厂商"退一级 view"命令，用于批量端口写每端口前回 (config)# 顶层。
//
// 2026-07-08 dump 实测：锐捷 RGOS 不支持在 (config-if-X)# 下直接 interface Y
// （返回 `% Unknown command.`），与 Cisco IOS 行为不同。批量场景复用 SSH 连接时，
// 设备停在上端口的 (config-if-X)# 视图，本端口 interface 命令被拒、action 落到错接口。
// 每端口前发 exit/quit 退一级回 (config)#，再 interface Y 进新视图。
//
// exit/quit 只退一级（end 会退到 privileged EXEC 破坏 scrapli priv 跟踪）。
// scrapli SendConfig 发 exit/quit 时 AcquirePriv 总先确保在 configuration priv，
// 故首端口即使在 privileged exec 也不会断连（先 escalate 进 config 再退一级）。
func VendorExitViewCmd(vendor models.DeviceVendor) string {
	switch vendor {
	case models.VendorHuawei, models.VendorH3C:
		return "quit"
	default: // VendorRuijie / 思科风格
		return "exit"
	}
}

// RenderCommand 公共入口：vendor × action → 命令序列。
//
// Phase 51 PortWriteService 直接消费返回 []string；多命令时通过 scrapli.SendConfigs
// 顺序执行，索引 i 即失败命令定位锚点。
func RenderCommand(vendor models.DeviceVendor, action PortAction, params PortTemplateParams) ([]string, error) {
	if params.InterfaceName == "" {
		return nil, ErrEmptyInterfaceName
	}

	vendorMap, ok := vendorPortTemplate[vendor]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedVendor, vendor)
	}

	render, ok := vendorMap[action]
	if !ok {
		return nil, fmt.Errorf("%w: %s (vendor: %s)", ErrUnknownAction, action, vendor)
	}

	return render(params)
}

// renderH3CDescription Huawei VRP / H3C Comware 共享的 description 模板（D-08 同源）。
// 进入 interface view 后下发 description 命令；Description 必填且不超过 80 字符。
func renderH3CDescription(p PortTemplateParams) ([]string, error) {
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

// renderRuijieDescription Ruijie RGOS description 模板（与 H3C 命令字面一致，
// 拆为独立函数以保留日志 / 调试时可分辨调用路径）。
func renderRuijieDescription(p PortTemplateParams) ([]string, error) {
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

// renderRuijieDot1xEnable 锐捷 dot1x enable：Cisco 风格显式进 interface view 后下发。
//
// 往返不对称 (Disable/Enable 对 default-user-limit 的处理非对称):
//   - disable (`no dot1x port-control auto`): 设备自动清除同视图下的
//     `dot1x default-user-limit N`,无需额外命令;
//   - enable  (`dot1x port-control auto`):   设备**不会**自动恢复 limit,
//     必须显式下发,否则 limit 漂成设备默认 (unlimited) 或被错置为 1。
//
// 因此 enable 命令序列固定为 2 条:
//   - `dot1x port-control auto`
//   - `dot1x default-user-limit N`    ← N 来自采集缓存 (p.Dot1xUserLimit),nil/0 时兜底 1
//
// 历史端口 dot1x_user_limit=0 时启用行为 = 硬编码 1,完全向后兼容。
func renderRuijieDot1xEnable(p PortTemplateParams) ([]string, error) {
	limit := 1
	if p.Dot1xUserLimit != nil && *p.Dot1xUserLimit > 0 {
		limit = *p.Dot1xUserLimit
	}
	return []string{
		fmt.Sprintf("interface %s", p.InterfaceName),
		"dot1x port-control auto",
		fmt.Sprintf("dot1x default-user-limit %d", limit),
	}, nil
}

// renderRuijieDot1xDisable 锐捷 dot1x disable：Cisco 风格显式进 interface view 后下发。
// `no dot1x port-control auto` 执行后,设备自动清除同视图下的 `dot1x default-user-limit N`,
// 无需额外下发清除命令。disable 路径因此固定 2 条,与 enable 往返对称 (清 + 不恢复 = 期望)。
func renderRuijieDot1xDisable(p PortTemplateParams) ([]string, error) {
	return []string{
		fmt.Sprintf("interface %s", p.InterfaceName),
		"no dot1x port-control auto",
	}, nil
}

// renderHuaweiSetAccessVlan 华为 VRP set_access_vlan：
//
//	interface <iface>
//	port link-type access
//	port default vlan <vlanId>
//
// RISK-03 关键字：华为/锐捷/H3C 共享 link-type access 前缀，但 vlan 关键字三选一：
// 华为 = `port default vlan`，H3C = `port access vlan`，锐捷 = `switchport access vlan`。
// vlanId 范围防御性检查在 renderer 内（service 层是 400 拦截源）。
func renderHuaweiSetAccessVlan(p PortTemplateParams) ([]string, error) {
	if p.VLANID < 1 || p.VLANID > 4094 {
		return nil, fmt.Errorf("portcollection: vlanId %d out of range 1-4094", p.VLANID)
	}
	return []string{
		fmt.Sprintf("interface %s", p.InterfaceName),
		"port link-type access",
		fmt.Sprintf("port default vlan %d", p.VLANID),
	}, nil
}

// renderH3CSetAccessVlan H3C Comware set_access_vlan：
//
//	interface <iface>
//	port link-type access
//	port access vlan <vlanId>     ← H3C 关键字差异（RISK-01/03）
func renderH3CSetAccessVlan(p PortTemplateParams) ([]string, error) {
	if p.VLANID < 1 || p.VLANID > 4094 {
		return nil, fmt.Errorf("portcollection: vlanId %d out of range 1-4094", p.VLANID)
	}
	return []string{
		fmt.Sprintf("interface %s", p.InterfaceName),
		"port link-type access",
		fmt.Sprintf("port access vlan %d", p.VLANID),
	}, nil
}

// renderRuijieSetAccessVlan Ruijie RGOS set_access_vlan（Cisco 风格）：
//
//	interface <iface>
//	switchport mode access
//	switchport access vlan <vlanId>
func renderRuijieSetAccessVlan(p PortTemplateParams) ([]string, error) {
	if p.VLANID < 1 || p.VLANID > 4094 {
		return nil, fmt.Errorf("portcollection: vlanId %d out of range 1-4094", p.VLANID)
	}
	return []string{
		fmt.Sprintf("interface %s", p.InterfaceName),
		"switchport mode access",
		fmt.Sprintf("switchport access vlan %d", p.VLANID),
	}, nil
}

// renderHuaweiPortBindingAdd 华为 user-bind 添加：
//
//	user-bind static ip-address <IP> [mac-address <AABB-CCDD-EEFF>]
//
// `static` 关键字是 Huawei 专属（H3C 省略）。
func renderHuaweiPortBindingAdd(p PortTemplateParams) ([]string, error) {
	if p.IPAddress == "" {
		return nil, fmt.Errorf("portcollection: ip-address required for huawei port_binding add")
	}
	args := fmt.Sprintf("ip-address %s", p.IPAddress)
	if p.MACAddress != "" {
		mac := toHuaweiH3CMACFormat(p.MACAddress)
		if mac != "" {
			args = fmt.Sprintf("%s mac-address %s", args, mac)
		}
	}
	return []string{
		fmt.Sprintf("interface %s", p.InterfaceName),
		fmt.Sprintf("user-bind static %s", args),
	}, nil
}

// renderH3CPortBindingAdd H3C user-bind 添加（RISK-01：无 `static` 关键字）：
//
//	user-bind ip-address <IP> [mac-address <AABB-CCDD-EEFF>]
func renderH3CPortBindingAdd(p PortTemplateParams) ([]string, error) {
	if p.IPAddress == "" {
		return nil, fmt.Errorf("portcollection: ip-address required for h3c port_binding add")
	}
	args := fmt.Sprintf("ip-address %s", p.IPAddress)
	if p.MACAddress != "" {
		mac := toHuaweiH3CMACFormat(p.MACAddress)
		if mac != "" {
			args = fmt.Sprintf("%s mac-address %s", args, mac)
		}
	}
	return []string{
		fmt.Sprintf("interface %s", p.InterfaceName),
		fmt.Sprintf("user-bind %s", args),
	}, nil
}

// renderRuijiePortBindingAdd Ruijie switchport port-security binding 添加（RISK-02）：
//
//	switchport port-security binding <aabb.ccdd.eeff> <IP>
//	或 IP-only 形式（无 MAC）：switchport port-security binding <IP>
func renderRuijiePortBindingAdd(p PortTemplateParams) ([]string, error) {
	if p.IPAddress == "" {
		return nil, fmt.Errorf("portcollection: ip-address required for ruijie port_binding add")
	}
	if p.MACAddress == "" {
		return []string{
			fmt.Sprintf("interface %s", p.InterfaceName),
			fmt.Sprintf("switchport port-security binding %s", p.IPAddress),
		}, nil
	}
	normalized := localNormalizeMACAddress(p.MACAddress)
	if normalized == "" {
		return nil, fmt.Errorf("portcollection: invalid mac for ruijie binding: %q", p.MACAddress)
	}
	mac := toRuijieMACFormat(normalized)
	return []string{
		fmt.Sprintf("interface %s", p.InterfaceName),
		fmt.Sprintf("switchport port-security binding %s %s", mac, p.IPAddress),
	}, nil
}

// renderHuaweiPortBindingRemove 华为 user-bind 解绑（带 `undo` 与 `static`）：
//
//	undo user-bind static ip-address <IP> [mac-address <AABB-CCDD-EEFF>]
func renderHuaweiPortBindingRemove(p PortTemplateParams) ([]string, error) {
	if p.IPAddress == "" {
		return nil, fmt.Errorf("portcollection: ip-address required for huawei port_binding remove")
	}
	args := fmt.Sprintf("ip-address %s", p.IPAddress)
	if p.MACAddress != "" {
		mac := toHuaweiH3CMACFormat(p.MACAddress)
		if mac != "" {
			args = fmt.Sprintf("%s mac-address %s", args, mac)
		}
	}
	return []string{
		fmt.Sprintf("interface %s", p.InterfaceName),
		fmt.Sprintf("undo user-bind static %s", args),
	}, nil
}

// renderH3CPortBindingRemove H3C user-bind 解绑（无 `static`）：
//
//	undo user-bind ip-address <IP> [mac-address <AABB-CCDD-EEFF>]
func renderH3CPortBindingRemove(p PortTemplateParams) ([]string, error) {
	if p.IPAddress == "" {
		return nil, fmt.Errorf("portcollection: ip-address required for h3c port_binding remove")
	}
	args := fmt.Sprintf("ip-address %s", p.IPAddress)
	if p.MACAddress != "" {
		mac := toHuaweiH3CMACFormat(p.MACAddress)
		if mac != "" {
			args = fmt.Sprintf("%s mac-address %s", args, mac)
		}
	}
	return []string{
		fmt.Sprintf("interface %s", p.InterfaceName),
		fmt.Sprintf("undo user-bind %s", args),
	}, nil
}

// renderRuijiePortBindingRemove Ruijie switchport port-security binding 解绑（Cisco 风格 `no` 前缀）：
//
//	no switchport port-security binding <aabb.ccdd.eeff> <IP>
//	或 IP-only 形式：no switchport port-security binding <IP>
func renderRuijiePortBindingRemove(p PortTemplateParams) ([]string, error) {
	if p.IPAddress == "" {
		return nil, fmt.Errorf("portcollection: ip-address required for ruijie port_binding remove")
	}
	if p.MACAddress == "" {
		return []string{
			fmt.Sprintf("interface %s", p.InterfaceName),
			fmt.Sprintf("no switchport port-security binding %s", p.IPAddress),
		}, nil
	}
	normalized := localNormalizeMACAddress(p.MACAddress)
	if normalized == "" {
		return nil, fmt.Errorf("portcollection: invalid mac for ruijie binding remove: %q", p.MACAddress)
	}
	mac := toRuijieMACFormat(normalized)
	return []string{
		fmt.Sprintf("interface %s", p.InterfaceName),
		fmt.Sprintf("no switchport port-security binding %s %s", mac, p.IPAddress),
	}, nil
}