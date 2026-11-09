package portwrite

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
)

// Phase 124 (v1.51 PWX-01)：VLAN trunk/hybrid 扩展 —— service 层。
//
// 沿用 121/123 全管线：RenderCommand → checkPreState → 互斥 → 预检 → 快照/回滚。
// trunk 动作复用 executeWrite 下发路径（尾部 trunkRenderParams 参数，nil = 非 trunk）。

// 哨兵错误（handler 翻译 400）。
var (
	// ErrHybridOverlap hybrid tagged/untagged 列表重叠。
	// 依据见 vlan_trunk_template.go 头注：华为设备侧会拒绝（有兜底），H3C 设备侧
	// 静默优先 untagged 不报错（无兜底）→ 系统侧校验是两厂商共同真相源。
	ErrHybridOverlap = fmt.Errorf("portwrite: hybrid tagged and untagged vlan lists must not overlap")
	// ErrAllowedModeInvalid trunk allowed 变更模式非法。
	ErrAllowedModeInvalid = fmt.Errorf("portwrite: allowed mode must be add, remove or replace")
	// ErrTrunkParamsInvalid trunk 参数缺失/组合非法。
	ErrTrunkParamsInvalid = fmt.Errorf("portwrite: trunk params invalid")
)

// TrunkParams trunk 动作参数集（PWX-01）。
type TrunkParams struct {
	NativeVLAN     int
	AllowedVLANs   []int
	AllowedMode    string // add | remove | replace（空 = replace）
	HybridTagged   []int
	HybridUntagged []int
}

// PortSecParams 端口安全动作参数集（PWX-02）。
type PortSecParams struct {
	MaxMACCount int
	// StaticBind 复用 MACAddress + VLANID 载体（PortSecStaticBind）
	MACAddress string
	VLANID     int
}

// renderExtras executeWrite 渲染参数扩展载体（nil = 既有 7 动作直通）。
// Phase 124 trunk / Phase 125 portsec / Phase 138 physical（FIX-04）共用，避免签名持续膨胀。
type renderExtras struct {
	trunk    *TrunkParams
	portsec  *PortSecParams
	physical *PhysicalParams // FIX-04 物理参数（速率/双工/MTU）
}

// trunkRenderParams 兼容别名（124 交付符号，writeSinglePort/executeWrite 形参）。
type trunkRenderParams = renderExtras

// validateVlanListRange VLAN 列表逐项 1-4094（复用单值 sentinel）。
func validateVlanListRange(vlans []int) error {
	for _, v := range vlans {
		if err := validateVlanIdRange(v); err != nil {
			return err
		}
	}
	return nil
}

// validateTrunkRequest trunk 动作入口校验（真相源；dry-run 与实跑同源）。
func validateTrunkRequest(action Action, p TrunkParams) error {
	switch action {
	case portcollection.TrunkSetNative:
		return validateVlanIdRange(p.NativeVLAN)
	case portcollection.TrunkSetAllowed:
		if len(p.AllowedVLANs) == 0 {
			return fmt.Errorf("%w: allowedVlans is required", ErrTrunkParamsInvalid)
		}
		if err := validateVlanListRange(p.AllowedVLANs); err != nil {
			return err
		}
		switch p.AllowedMode {
		case "", portcollection.AllowedModeAdd, portcollection.AllowedModeRemove, portcollection.AllowedModeReplace:
			return nil
		default:
			return ErrAllowedModeInvalid
		}
	case portcollection.TrunkSetHybrid:
		if len(p.HybridTagged) == 0 && len(p.HybridUntagged) == 0 {
			return fmt.Errorf("%w: hybridTagged/hybridUntagged at least one required", ErrTrunkParamsInvalid)
		}
		if err := validateVlanListRange(p.HybridTagged); err != nil {
			return err
		}
		if err := validateVlanListRange(p.HybridUntagged); err != nil {
			return err
		}
		if hybridListsOverlap(p.HybridTagged, p.HybridUntagged) {
			return ErrHybridOverlap
		}
		return nil
	default:
		return fmt.Errorf("%w: %s", ErrTrunkParamsInvalid, action)
	}
}

// checkRuijieHybridParams 锐捷 hybrid 的**厂商语义**校验（产品口径 (i)+(i-a)）。
//
// 真机依据（2026-11-02 第三次取证 + 补验，lead 拍板 (i)）：
// 锐捷 hybrid 是「自动补集」语义 —— 下发 untagged 后，设备把其余全部 VLAN 自动划入
// tagged（实测 untagged 3992 ⇒ 设备落成 `only tagged 2-3991,3993-4094` +
// `add untagged 1,3992`）⇒ 锐捷 hybrid **不做 VLAN 收敛**，产品的 tagged 列表在
// 锐捷上**没有任何设备侧对应物**。
//
// ⇒ 提交 tagged 必须**明确报错**（不静默丢弃：静默丢弃 = 静默改变用户意图，
//
//	比直接失败危险得多）；untagged 至少一个。
//
// 与通用 validator（validateTrunkRequest）的分工：通用校验管"列表非空/范围/重叠"，
// 本函数管**厂商语义**（同一份参数在华为合法、在锐捷不合法）。
// 渲染层（portcollection 锐捷渲染器）也做同样拦截 —— 那里是产品真相源，
// 直接调 RenderCommand 的场景同样拦得住。
func checkRuijieHybridParams(action Action, p TrunkParams, vendor models.DeviceVendor) error {
	if action != portcollection.TrunkSetHybrid || vendor != models.VendorRuijie {
		return nil
	}
	if len(p.HybridTagged) > 0 {
		return fmt.Errorf("%w: 锐捷 hybrid 为自动补集语义：untagged 之外的 VLAN 会被设备自动划入 tagged，无法显式指定 tagged 列表；请只提交 untagged 列表（华为设备的 tagged 为显式语义，可正常使用）",
			portcollection.ErrActionNotSupported)
	}
	if len(p.HybridUntagged) == 0 {
		return fmt.Errorf("%w: 锐捷 hybrid 至少需要一个 untagged VLAN（锐捷不支持显式 tagged 列表；untagged 之外的 VLAN 由设备自动划入 tagged）",
			portcollection.ErrActionNotSupported)
	}
	return nil
}

// hybridListsOverlap tagged/untagged 交集判断。
func hybridListsOverlap(tagged, untagged []int) bool {
	set := make(map[int]struct{}, len(tagged))
	for _, v := range tagged {
		set[v] = struct{}{}
	}
	for _, v := range untagged {
		if _, ok := set[v]; ok {
			return true
		}
	}
	return false
}

// TrunkWrite trunk/hybrid 写入口（PWX-01；单端点三动作）。
//
// 管线与既有 7 动作完全一致：validator → writeSinglePort（互斥/预检/快照/回滚）
// → executeWrite 渲染下发。trunk 动作无 DB 缓存（sys_device_port_status 不存
// native/allowed/hybrid 列表），pre-state 恒不短路（同 port_binding 语义）。
func (s *portWriteServiceImpl) TrunkWrite(ctx context.Context, portID string, action Action, params TrunkParams, operator string, opts ...WriteOption) (*PortResult, error) {
	if action != portcollection.TrunkSetNative && action != portcollection.TrunkSetAllowed && action != portcollection.TrunkSetHybrid {
		return nil, fmt.Errorf("%w: %s is not a trunk action", ErrTrunkParamsInvalid, action)
	}
	if err := validateTrunkRequest(action, params); err != nil {
		return nil, err
	}
	result, err := s.writeSinglePort(ctx, portID, action, "", operator, 0, "", "", "", nil, applyWriteOptions(opts), &renderExtras{trunk: &params})
	if err != nil {
		return nil, err
	}
	// 审计 after_value：把 trunk 参数完整写入 Extra（handler buildAfterValue 直采）
	if result != nil {
		if result.Extra == nil {
			result.Extra = make(map[string]interface{})
		}
		result.Extra["trunk"] = params
	}
	return result, nil
}

// riskLevelForActionTrunk 扩展版风险标注（trunk 影响面大：跨 VLAN 转发路径变更 → critical）。
func riskLevelForActionTrunk(action Action) string {
	if action == portcollection.TrunkSetNative || action == portcollection.TrunkSetAllowed || action == portcollection.TrunkSetHybrid {
		return RiskLevelCritical
	}
	return riskLevelForAction(action)
}

// isCriticalActionExt trunk 动作同样纳入快照覆盖。
func isCriticalActionExt(action Action) bool {
	return isCriticalAction(action) || action == portcollection.TrunkSetNative ||
		action == portcollection.TrunkSetAllowed || action == portcollection.TrunkSetHybrid
}
