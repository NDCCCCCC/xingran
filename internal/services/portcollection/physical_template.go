package portcollection

// FIX-04（v1.53 Phase 138）：物理参数「速率 / 双工 / MTU 调整」命令模板。
//
// 沿用 v1.19/v1.20.1 端口写基建：vendor × action → 命令序列渲染闭包，通过
// init() 注册进 vendorPortTemplate 派发表（与 portsec_template.go / vlan_trunk_template.go
// 同源模式）。命令均在 interface view 下下发，故统一用 wrapInterface 包裹。
//
// 3 厂商语法依据（公开稳定 CLI，未做真机取证 —— 见报告「遗留问题」）：
//   - 华为 VRP：speed {10|100|1000|auto} / duplex {half|full|auto} / mtu <1500-9600>
//     （jumbo frame 用 `jumboframe enable <val>` 在部分型号；本实现用通用 `mtu`）
//   - H3C Comware：speed {10|100|1000|auto} / duplex {half|full|auto} /
//     mtu <1280-9600>（Comware 7 接口下 `mtu <val>`）
//   - 锐捷 RGOS（Cisco 风格）：speed {10|100|1000|auto} /
//     duplex {half|full|auto} / mtu <64-9216>
//
// ⚠️ 真机取证状态：本文件命令未针对三家真机逐一验证（Phase 129 UAT 未覆盖物理
// 参数）。语法按各厂商官方 command reference 的稳定形态；能力矩阵不对新动作
// 设 Unsupported（沿用 vendor baseline 默认支持），故不对既有真机结论产生回归。
// 若现场型号关键字不同，需在真机 UAT 后按型号追加 modelRules / 模板分支。

import (
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// PortAction FIX-04 新增 3 个物理参数动作。
const (
	ActionSetSpeed  PortAction = "set_speed"
	ActionSetDuplex PortAction = "set_duplex"
	ActionSetMTU    PortAction = "set_mtu"
)

// 物理参数合法值集合（service 层 validator 与 renderer 共用同一份真相源，
// 避免「校验放行但渲染拒绝」或反之）。速率单位 Mbps。
var allowedSpeeds = map[int]bool{10: true, 100: true, 1000: true, 10000: true}

// allowedSpeedStrings auto 是独立枚举（不与数字速率混用）：用 SpeedMode 区分。
const (
	SpeedModeAuto = "auto"
	DuplexHalf    = "half"
	DuplexFull    = "full"
	DuplexAuto    = "auto"
)

var allowedDuplexes = map[string]bool{DuplexHalf: true, DuplexFull: true, DuplexAuto: true}

// SpeedAllowed 数字速率是否合法（供 service 层 validator 复用，保持单一真相源）。
func SpeedAllowed(speed int) bool { return allowedSpeeds[speed] }

// DuplexAllowed 双工枚举是否合法（供 service 层 validator 复用）。
func DuplexAllowed(duplex string) bool { return allowedDuplexes[duplex] }

// MTU 范围：取三家交集 [1280, 9216]（华为/Jumbo、H3C、锐捷均覆盖该区间），
// 作为系统侧保守真相源；renderer 不再做二次范围校验。
const (
	MTUMin = 1280
	MTUMax = 9216
)

// renderPhysical 三厂商物理参数命令拼装（命令体按厂商 + 动作展开）。
//
// 参数载体复用 PortTemplateParams 新增字段（见 vendor_port_template.go）：
//   - SpeedMode: "auto" 或 ""（非 auto 时用数字速率 Speed）
//   - Speed: 数字速率 Mbps（SpeedMode=="" 时有效）
//   - Duplex: half|full|auto
//   - MTU: 1280-9216
func renderPhysical(p PortTemplateParams, vendor models.DeviceVendor, action PortAction) ([]string, error) {
	switch action {
	case ActionSetSpeed:
		var val string
		if p.SpeedMode == SpeedModeAuto {
			val = SpeedModeAuto
		} else {
			if !allowedSpeeds[p.Speed] {
				return nil, fmt.Errorf("portcollection: invalid speed %d (must be 10|100|1000|10000|auto)", p.Speed)
			}
			val = fmt.Sprintf("%d", p.Speed)
		}
		return wrapInterface(p, fmt.Sprintf("speed %s", val))
	case ActionSetDuplex:
		if !allowedDuplexes[p.Duplex] {
			return nil, fmt.Errorf("portcollection: invalid duplex %q (must be half|full|auto)", p.Duplex)
		}
		return wrapInterface(p, fmt.Sprintf("duplex %s", p.Duplex))
	case ActionSetMTU:
		if p.MTU < MTUMin || p.MTU > MTUMax {
			return nil, fmt.Errorf("portcollection: mtu %d out of range %d-%d", p.MTU, MTUMin, MTUMax)
		}
		// 三家接口视图下 `mtu <val>` 关键字一致；保留 vendor 分支便于未来
		// 追加型号差异（如华为 CE 系 jumboframe enable）。
		_ = vendor
		return wrapInterface(p, fmt.Sprintf("mtu %d", p.MTU))
	}
	return nil, fmt.Errorf("portcollection: unknown physical action %s", action)
}

// RegisterPhysicalTemplates 注册 FIX-04 物理参数渲染闭包（3 厂商 × 3 动作）。
func RegisterPhysicalTemplates() {
	for _, vendor := range []models.DeviceVendor{models.VendorHuawei, models.VendorH3C, models.VendorRuijie} {
		v := vendor
		vendorPortTemplate[v][ActionSetSpeed] = func(p PortTemplateParams) ([]string, error) { return renderPhysical(p, v, ActionSetSpeed) }
		vendorPortTemplate[v][ActionSetDuplex] = func(p PortTemplateParams) ([]string, error) { return renderPhysical(p, v, ActionSetDuplex) }
		vendorPortTemplate[v][ActionSetMTU] = func(p PortTemplateParams) ([]string, error) { return renderPhysical(p, v, ActionSetMTU) }
	}
}

func init() {
	RegisterPhysicalTemplates()
}
