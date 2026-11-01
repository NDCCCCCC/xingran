package portwrite

import (
	"context"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
)

// Phase 125 (v1.51 PWX-02)：端口安全 —— service 层。
//
// portsec_enable / portsec_disable / portsec_max_mac / portsec_sticky /
// portsec_static_bind。沿用 121/123/124 全管线。

// 哨兵错误。
var (
	// ErrPortSecParamsInvalid 端口安全参数缺失/组合非法。
	ErrPortSecParamsInvalid = fmt.Errorf("portwrite: port-security params invalid")
	// ErrMaxMacOutOfRange max-mac-num 超出 1-4096。
	ErrMaxMacOutOfRange = fmt.Errorf("portwrite: maxMacCount out of range 1-4096")
)

// validatePortSecRequest 端口安全入口校验（真相源；dry-run 与实跑同源）。
func validatePortSecRequest(action Action, p PortSecParams) error {
	switch action {
	case portcollection.PortSecEnable, portcollection.PortSecDisable, portcollection.PortSecSticky:
		return nil
	case portcollection.PortSecMaxMac:
		if p.MaxMACCount < 1 || p.MaxMACCount > 4096 {
			return ErrMaxMacOutOfRange
		}
		return nil
	case portcollection.PortSecStaticBind:
		if err := validateMACAddress(p.MACAddress); err != nil {
			return err
		}
		if p.MACAddress == "" {
			return fmt.Errorf("%w: macAddress is required for static bind", ErrPortSecParamsInvalid)
		}
		// Phase 129 真机修复：华为 S8700 sticky 不接受 vlan 参数（2026-11-01
		// 接口视图探测，渲染层忽略 VLANID）→ VLANID 可选：非零才校验范围。
		if p.VLANID != 0 {
			return validateVlanIdRange(p.VLANID)
		}
		return nil
	default:
		return fmt.Errorf("%w: %s is not a port-security action", ErrPortSecParamsInvalid, action)
	}
}

// PortSecWrite 端口安全写入口（PWX-02；单端点五动作）。
//
// 管线与既有动作一致。portsec_static_bind 风险 critical（影响合法用户接入），
// 其余 medium；快照/回滚无 DB 原值（sys_device_port_status 无绑定表）→
// no_rollback 语义（同 trunk）。MAC 与接口唯一性：sys_device_port_status 不存
// 绑定表（Phase 56 Pitfall 6 同源），由设备端 duplicate detection 兜底 ——
// 系统侧校验 MAC 格式 + VLAN 范围，契约已注明。
func (s *portWriteServiceImpl) PortSecWrite(ctx context.Context, portID string, action Action, params PortSecParams, operator string, opts ...WriteOption) (*PortResult, error) {
	switch action {
	case portcollection.PortSecEnable, portcollection.PortSecDisable, portcollection.PortSecMaxMac, portcollection.PortSecSticky, portcollection.PortSecStaticBind:
	default:
		return nil, fmt.Errorf("%w: %s is not a port-security action", ErrPortSecParamsInvalid, action)
	}
	if err := validatePortSecRequest(action, params); err != nil {
		return nil, err
	}
	result, err := s.writeSinglePort(ctx, portID, action, "", operator, 0, "", "", "", nil, applyWriteOptions(opts), &renderExtras{portsec: &params})
	if err != nil {
		return nil, err
	}
	// 审计 after_value：端口安全参数回显
	if result != nil {
		if result.Extra == nil {
			result.Extra = make(map[string]interface{})
		}
		result.Extra["portsec"] = params
	}
	return result, nil
}

// checkPortSecPreState 端口安全 pre-state NoOp（PORT-06 语义扩展，Phase 125）。
//
// 复用采集缓存（sys_device_port_status.PortSecurityEnabled / MaxMACCount，
// port-collection cron 周期刷新）：
//   - portsec_enable：已启用 → NoOp
//   - portsec_disable：未启用 → NoOp
//   - portsec_max_mac：缓存上限已等于目标 → NoOp
//   - sticky / static_bind：无缓存语义 → 不短路（同 port_binding）
func checkPortSecPreState(port *models.DevicePortStatus, action Action, p PortSecParams) *PortResult {
	noop := func(currentState string) *PortResult {
		return &PortResult{
			PortID:       port.ID,
			Action:       action,
			Status:       "skipped",
			NoOp:         true,
			CurrentState: currentState,
		}
	}
	switch action {
	case portcollection.PortSecEnable:
		if port.PortSecurityEnabled {
			return noop("portsec_enabled")
		}
	case portcollection.PortSecDisable:
		if !port.PortSecurityEnabled {
			return noop("portsec_disabled")
		}
	case portcollection.PortSecMaxMac:
		if port.MaxMACCount != nil && *port.MaxMACCount == p.MaxMACCount {
			return noop("max_mac_match")
		}
	}
	return nil
}

// riskLevelForActionPortSec 端口安全风险等级。
func riskLevelForActionPortSec(action Action) string {
	if action == portcollection.PortSecStaticBind {
		return RiskLevelCritical
	}
	if action == portcollection.PortSecEnable || action == portcollection.PortSecDisable ||
		action == portcollection.PortSecMaxMac || action == portcollection.PortSecSticky {
		return RiskLevelMedium
	}
	return riskLevelForActionTrunk(action)
}

// isCriticalActionPortSec 快照覆盖判断（static_bind / max_mac 纳入快照，虽无回滚命令仍留原状态）。
func isCriticalActionPortSec(action Action) bool {
	return isCriticalActionExt(action) || action == portcollection.PortSecStaticBind
}
