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
		ActionShutdown:     func(p PortTemplateParams) ([]string, error) { return []string{"shutdown"}, nil },
		ActionUndoShutdown: func(p PortTemplateParams) ([]string, error) { return []string{"undo shutdown"}, nil },
		ActionDescription:  renderH3CDescription,
		ActionDot1xEnable:  func(p PortTemplateParams) ([]string, error) { return []string{"dot1x enable"}, nil },
		ActionDot1xDisable: func(p PortTemplateParams) ([]string, error) { return []string{"undo dot1x enable"}, nil },
	},
	models.VendorH3C: {
		ActionShutdown:     func(p PortTemplateParams) ([]string, error) { return []string{"shutdown"}, nil },
		ActionUndoShutdown: func(p PortTemplateParams) ([]string, error) { return []string{"undo shutdown"}, nil },
		ActionDescription:  renderH3CDescription,
		ActionDot1xEnable:  func(p PortTemplateParams) ([]string, error) { return []string{"dot1x enable"}, nil },
		ActionDot1xDisable: func(p PortTemplateParams) ([]string, error) { return []string{"undo dot1x enable"}, nil },
	},
	models.VendorRuijie: {
		ActionShutdown:     func(p PortTemplateParams) ([]string, error) { return []string{"shutdown"}, nil },
		ActionUndoShutdown: func(p PortTemplateParams) ([]string, error) { return []string{"no shutdown"}, nil },
		ActionDescription:  renderRuijieDescription,
		ActionDot1xEnable:  renderRuijieDot1xEnable,
		ActionDot1xDisable: renderRuijieDot1xDisable,
	},
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