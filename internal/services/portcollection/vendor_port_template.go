package portcollection

import (
	"errors"
	"fmt"

	"github.com/ruoyi-next/ruoyi-go-backend/internal/models"
)

// PortAction 端口写操作类型（Phase 50 锁定；Phase 52 operlog action 字段直采此值）。
type PortAction string

const (
	ActionShutdown     PortAction = "shutdown"
	ActionUndoShutdown PortAction = "undo_shutdown"
	ActionDescription  PortAction = "description"
	ActionDot1xEnable  PortAction = "dot1x_enable"
	ActionDot1xDisable PortAction = "dot1x_disable"
)

// String 返回 PortAction 的字符串值，便于 operlog / 日志记录。
func (a PortAction) String() string { return string(a) }

// PortTemplateParams 模板渲染参数。
//
// InterfaceName 所有 action 必填（RenderCommand 入口校验）。
// Description   仅 ActionDescription 使用；其他 action 忽略。
type PortTemplateParams struct {
	InterfaceName string
	Description   string
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
	},
	models.VendorH3C: {
		ActionShutdown:     func(p PortTemplateParams) ([]string, error) { return wrapInterface(p, "shutdown") },
		ActionUndoShutdown: func(p PortTemplateParams) ([]string, error) { return wrapInterface(p, "undo shutdown") },
		ActionDescription:  renderH3CDescription,
		ActionDot1xEnable:  func(p PortTemplateParams) ([]string, error) { return wrapInterface(p, "dot1x enable") },
		ActionDot1xDisable: func(p PortTemplateParams) ([]string, error) { return wrapInterface(p, "undo dot1x enable") },
	},
	models.VendorRuijie: {
		ActionShutdown:     func(p PortTemplateParams) ([]string, error) { return wrapInterface(p, "shutdown") },
		ActionUndoShutdown: func(p PortTemplateParams) ([]string, error) { return wrapInterface(p, "no shutdown") },
		ActionDescription:  renderRuijieDescription,
		ActionDot1xEnable:  renderRuijieDot1xEnable,
		ActionDot1xDisable: renderRuijieDot1xDisable,
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
func renderRuijieDot1xEnable(p PortTemplateParams) ([]string, error) {
	return []string{
		fmt.Sprintf("interface %s", p.InterfaceName),
		"dot1x port-control auto",
	}, nil
}

// renderRuijieDot1xDisable 锐捷 dot1x disable：Cisco 风格显式进 interface view 后下发。
func renderRuijieDot1xDisable(p PortTemplateParams) ([]string, error) {
	return []string{
		fmt.Sprintf("interface %s", p.InterfaceName),
		"no dot1x port-control",
	}, nil
}