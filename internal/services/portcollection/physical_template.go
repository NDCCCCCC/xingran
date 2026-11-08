package portcollection

// FIX-04（v1.53 Phase 138）：物理参数「速率 / 双工 / MTU 调整」命令模板。
// Phase 143 (PORT-02)：接入型号级能力矩阵（physical_capability.go），
// 校验与渲染同源，命令形态按 ResolvePhysicalProfile 的画像分叉。
//
// 沿用 v1.19/v1.20.1 端口写基建：vendor × action → 命令序列渲染闭包，通过
// init() 注册进 vendorPortTemplate 派发表（与 portsec_template.go / vlan_trunk_template.go
// 同源模式）。命令均在 interface view 下下发，故默认统一用 wrapInterface 包裹；
// 型号画像 RequiresView=false 时改为系统视图序列（不加 interface 前缀）。
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
// 参数）。语法按各厂商官方 command reference 的稳定形态；全局默认画像 Evidence=""
// 表示「未取证」，不冒充真机结论。若现场型号关键字不同，需在真机 UAT 后按型号
// 追加 modelPhysical / modelRules 分支（Phase 143 只提供可代码化框架）。

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

// 物理参数合法值集合（全局默认画像真相源；service 层 validator 与 renderer 共用同一份，
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

// SpeedAllowed 数字速率是否合法（FIX-04 既有导出 helper，保持向后兼容）。
// 型号级值域以 CheckPhysicalSupport / ResolvePhysicalProfile 为准，本 helper 只表示
// 全局默认画像允许集合（供既有测试与简单调用方使用）。
func SpeedAllowed(speed int) bool { return allowedSpeeds[speed] }

// DuplexAllowed 双工枚举是否合法（FIX-04 既有导出 helper，语义同上）。
func DuplexAllowed(duplex string) bool { return allowedDuplexes[duplex] }

// MTU 范围：取三家交集 [1280, 9216]（华为/Jumbo、H3C、锐捷均覆盖该区间），
// 作为全局默认画像的保守真相源；型号级上下界以 physicalProfile.MTUMin/Max 为准。
const (
	MTUMin = 1280
	MTUMax = 9216
)

// renderPhysical 三厂商物理参数命令拼装（命令体按画像 + 动作展开）。
//
// Phase 143：p.Model 传给 CheckPhysicalSupport（与 service 层 validatePhysicalRequest
// 同源），取回 physicalProfile 后用其 SpeedCommand / DuplexCommand / MTUCommand
// 拼装；三家默认画像命令体与 FIX-04 完全一致，保证单条物理写零回归。
//
// 参数载体复用 PortTemplateParams 字段：
//   - SpeedMode: "auto" 或 ""（非 auto 时用数字速率 Speed）
//   - Speed: 数字速率 Mbps（SpeedMode=="" 时有效）
//   - Duplex: half|full|auto
//   - MTU: 画像上下界内的值
//   - Model: 设备型号（可空；空则厂商基线/全局默认）
func renderPhysical(p PortTemplateParams, vendor models.DeviceVendor, action PortAction) ([]string, error) {
	// 校验/渲染同源：用与 service 入口完全相同的 CheckPhysicalSupport 取画像，
	// 既做能力拒绝（Cisco/Maipu/型号 unsupported），也做参数值域/命令形态解析。
	prof, err := CheckPhysicalSupport(vendor, p.Model, action, p)
	if err != nil {
		return nil, err
	}

	var cmd string
	switch action {
	case ActionSetSpeed:
		val := SpeedModeAuto
		if p.SpeedMode != SpeedModeAuto {
			val = fmt.Sprintf("%d", p.Speed)
		}
		cmd = fmt.Sprintf(prof.SpeedCommand, val)
	case ActionSetDuplex:
		cmd = fmt.Sprintf(prof.DuplexCommand, p.Duplex)
	case ActionSetMTU:
		cmd = fmt.Sprintf(prof.MTUCommand, p.MTU)
	default:
		// CheckPhysicalSupport 已对非物理动作返回 ErrPhysicalParamInvalid，
		// 这里保留兜底防御（renderer 不信任调用方）。
		return nil, fmt.Errorf("portcollection: unknown physical action %s", action)
	}

	if !prof.RequiresView {
		// 系统视图命令序列（型号画像差异）：不加 interface 前缀。
		return []string{cmd}, nil
	}
	return wrapInterface(p, cmd)
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
