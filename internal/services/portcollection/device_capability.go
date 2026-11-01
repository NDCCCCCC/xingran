package portcollection

import (
	"fmt"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// Phase 129 真机 UAT 修复（2026-11-01 华为 S8700 + 锐捷 RS8607E 实测）：
// 厂商/型号能力矩阵 —— 让不支持的动作返回明确错误，而不是下发必失败命令。
//
// 真机事实（华为 S8700 10.62.25.253，接口 GE1/0/0，设备 `?` 探测）：
//   1. user-bind 仅系统视图可用（接口视图 Unrecognized）→ 华为 port_binding
//      命令渲染为无 interface 前缀的系统视图序列（RequiresInterfaceView=false）
//   2. `port-security ?` 子命令仅 aging-time/enable/mac-address/protect-action
//      → S8700 无 max-mac-num → portsec_max_mac 不支持
//   3. `port ?` 子命令无 trunk/hybrid → S8700 无 trunk/hybrid 关键字
//      （port default vlan / port link-type 是正确关键字，既有动作不受影响）
//
// 设计：
//   - 厂商基线（vendor baseline）：该厂商各动作默认支持级别
//   - 型号规则（model rules）：按 NetworkDevice.Model 字符串匹配叠加覆盖
//     （如 "S87" 匹配 S8700 系列）；未匹配型号沿用厂商基线（保守兼容）
//   - 校验入口 CheckActionSupport：写路径与 dry-run 前统一调用，不支持动作
//     返回 ErrActionNotSupported（handler 翻译 400 + reason 含真机依据）

// ActionSupport 动作支持级别。
type ActionSupport int

const (
	// ActionSupported 支持（常规接口视图执行）。
	ActionSupported ActionSupport = iota
	// ActionSystemView 需在系统视图执行（命令序列不带 interface 前缀，
	// executeWrite 不追加 view 退出命令）。
	ActionSystemView
	// ActionUnsupported 该厂商/型号不支持（下发必失败，应显式拒绝）。
	ActionUnsupported
)

// ErrActionNotSupported 能力矩阵拒绝（%w 链携带 vendor/model/action/reason）。
var ErrActionNotSupported = fmt.Errorf("portcollection: action not supported on this device")

// capabilityEntry 单厂商能力描述。
type capabilityEntry struct {
	// baseline 厂商基线：action → 级别（缺省 ActionSupported）
	baseline map[PortAction]ActionSupport
	// modelRules 型号覆盖规则：model 子串（大小写不敏感）→ 覆盖表
	modelRules []modelCapabilityRule
}

// modelCapabilityRule 型号级能力覆盖。
type modelCapabilityRule struct {
	// matchModelSubstr model 字符串包含任一子串即命中（大小写不敏感）
	matchModelSubstr []string
	// unsupported 命中后不支持的动作集合
	unsupported map[PortAction]string // action → reason（真机依据）
}

// vendorCapabilities 厂商能力矩阵（真机事实唯一真相源，测试锁定）。
var vendorCapabilities = map[models.DeviceVendor]capabilityEntry{
	models.VendorHuawei: {
		baseline: map[PortAction]ActionSupport{
			// 真机事实 #1：user-bind 是系统视图命令（接口视图 Unrecognized）
			ActionPortBinding: ActionSystemView,
		},
		modelRules: []modelCapabilityRule{
			{
				// 真机事实 #2/#3：S8700（S87xx）无 port trunk/hybrid 关键字、
				// port-security 无 max-mac-num 子命令
				matchModelSubstr: []string{"s87", "s8700"},
				unsupported: map[PortAction]string{
					TrunkSetNative:  "S8700 接口视图 port ? 无 trunk 子命令（2026-11-01 真机 ? 探测）",
					TrunkSetAllowed: "S8700 接口视图 port ? 无 trunk 子命令（2026-11-01 真机 ? 探测）",
					TrunkSetHybrid:  "S8700 接口视图 port ? 无 hybrid 子命令（2026-11-01 真机 ? 探测）",
					PortSecMaxMac:   "S8700 port-security ? 无 max-mac-num 子命令（2026-11-01 真机 ? 探测）",
				},
			},
		},
	},
	// H3C / Ruijie / Cisco：真机证据不足，全部默认支持（S8700 之外的华为
	// CE/S57 系列按公允语法支持 trunk/hybrid/max-mac-num）。
	// 锐捷 hybrid 在渲染层不注册（RenderCommand → ErrUnknownAction），此处
	// 不重复表达。
}

// RequiresInterfaceView 该动作命令序列是否需要 interface 前缀视图。
//
// false 时 executeWrite 不追加 view 退出命令（命令直接在系统视图执行，
// 追加 exit 会把会话掉出配置视图破坏 scrapli priv 跟踪）。
func RequiresInterfaceView(vendor models.DeviceVendor, action PortAction) bool {
	return lookupSupport(vendor, "", action) != ActionSystemView
}

// lookupSupport 解析 (vendor, model, action) → 支持级别（model 规则优先于基线）。
func lookupSupport(vendor models.DeviceVendor, model string, action PortAction) ActionSupport {
	entry, ok := vendorCapabilities[vendor]
	if !ok {
		return ActionSupported
	}
	if model != "" {
		lower := strings.ToLower(model)
		for _, rule := range entry.modelRules {
			for _, substr := range rule.matchModelSubstr {
				if strings.Contains(lower, substr) {
					if _, banned := rule.unsupported[action]; banned {
						return ActionUnsupported
					}
				}
			}
		}
	}
	if lvl, ok := entry.baseline[action]; ok {
		return lvl
	}
	return ActionSupported
}

// unsupportedReason 返回型号级不支持原因（无则空串）。
func unsupportedReason(vendor models.DeviceVendor, model string, action PortAction) string {
	entry, ok := vendorCapabilities[vendor]
	if !ok || model == "" {
		return ""
	}
	lower := strings.ToLower(model)
	for _, rule := range entry.modelRules {
		for _, substr := range rule.matchModelSubstr {
			if strings.Contains(lower, substr) {
				if reason, banned := rule.unsupported[action]; banned {
					return reason
				}
			}
		}
	}
	return ""
}

// CheckActionSupport 能力矩阵校验入口（写路径与 dry-run 统一调用）。
//
// model 为设备型号（models.NetworkDevice.Model，可空——空则仅厂商基线生效）。
// 不支持返回 ErrActionNotSupported 包装错误（reason 含真机依据）。
func CheckActionSupport(vendor models.DeviceVendor, model string, action PortAction) error {
	switch lookupSupport(vendor, model, action) {
	case ActionUnsupported:
		reason := unsupportedReason(vendor, model, action)
		if reason == "" {
			reason = "该厂商不支持此动作"
		}
		return fmt.Errorf("%w: vendor=%s model=%q action=%s — %s",
			ErrActionNotSupported, vendor, model, action, reason)
	default:
		return nil
	}
}
