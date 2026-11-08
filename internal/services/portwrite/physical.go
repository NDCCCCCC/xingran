package portwrite

// FIX-04（v1.53 Phase 138）：物理参数「速率 / 双工 / MTU 调整」service 层。
//
// 沿用 v1.20.1 / v1.51 端口写基建：sentinel validator（入口真相源）→
// writeSinglePort（pre-state / 互斥 / 预检 / RenderCommand / ExecuteCustom）→
// PortResult.Extra 写审计 after_value。与 PortSecWrite / TrunkWrite 同构。

import (
	"context"
	"errors"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
)

// 哨兵错误（handler 翻译 400）。沿用 portwrite: 前缀与既有 sentinel 风格。
var (
	// ErrSpeedInvalid 速率非法（既非 10/100/1000/10000 也非 auto）。
	ErrSpeedInvalid = errors.New("portwrite: speed must be 10|100|1000|10000 or auto")
	// ErrDuplexInvalid 双工非法（非 half|full|auto）。
	ErrDuplexInvalid = errors.New("portwrite: duplex must be half|full|auto")
	// ErrMTUOutOfRange MTU 越界（1280-9216 三家交集）。
	ErrMTUOutOfRange = errors.New("portwrite: mtu out of range 1280-9216")
	// ErrPhysicalParamsInvalid 物理参数动作非法或参数组合缺失。
	ErrPhysicalParamsInvalid = errors.New("portwrite: physical params invalid")
)

// PhysicalParams 物理参数动作参数集（FIX-04）。
//
//   - set_speed：SpeedMode=="auto" 时忽略 Speed；否则 Speed ∈ {10,100,1000,10000}
//   - set_duplex：Duplex ∈ {half, full, auto}
//   - set_mtu：MTU ∈ [1280, 9216]
type PhysicalParams struct {
	Speed     int    `json:"speed,omitempty"`
	SpeedMode string `json:"speedMode,omitempty"`
	Duplex    string `json:"duplex,omitempty"`
	MTU       int    `json:"mtu,omitempty"`
}

// validatePhysicalRequest 物理参数入口校验（真相源；dry-run 与实跑同源）。
//
// 复用 portcollection 的 allowedSpeeds / allowedDuplexes / MTUMin / MTUMax
// 作为唯一枚举定义（避免 service 与 renderer 两份漂移），但返回 portwrite
// sentinel 供 handler 翻译 400。
func validatePhysicalRequest(action Action, p PhysicalParams) error {
	switch action {
	case portcollection.ActionSetSpeed:
		if p.SpeedMode == portcollection.SpeedModeAuto {
			return nil
		}
		if p.SpeedMode != "" {
			return fmt.Errorf("%w: unknown speedMode %q", ErrSpeedInvalid, p.SpeedMode)
		}
		if !portcollection.SpeedAllowed(p.Speed) {
			return fmt.Errorf("%w: %d", ErrSpeedInvalid, p.Speed)
		}
		return nil
	case portcollection.ActionSetDuplex:
		if !portcollection.DuplexAllowed(p.Duplex) {
			return fmt.Errorf("%w: %q", ErrDuplexInvalid, p.Duplex)
		}
		return nil
	case portcollection.ActionSetMTU:
		if p.MTU < portcollection.MTUMin || p.MTU > portcollection.MTUMax {
			return fmt.Errorf("%w: %d (must be %d-%d)", ErrMTUOutOfRange, p.MTU, portcollection.MTUMin, portcollection.MTUMax)
		}
		return nil
	default:
		return fmt.Errorf("%w: %s is not a physical action", ErrPhysicalParamsInvalid, action)
	}
}

// PhysicalWrite 物理参数写入口（FIX-04；单端点三动作）。
//
// 管线与 PortSecWrite 一致：validator → writeSinglePort（复用 renderExtras.physical
// 透传参数到 executeWrite）→ 成功后把参数回显进 PortResult.Extra 供审计
// after_value（buildAfterValue 从 Extra 读 speed/duplex/mtu）。
func (s *portWriteServiceImpl) PhysicalWrite(ctx context.Context, portID string, action Action, params PhysicalParams, operator string, opts ...WriteOption) (*PortResult, error) {
	switch action {
	case portcollection.ActionSetSpeed, portcollection.ActionSetDuplex, portcollection.ActionSetMTU:
	default:
		return nil, fmt.Errorf("%w: %s is not a physical action", ErrPhysicalParamsInvalid, action)
	}
	if err := validatePhysicalRequest(action, params); err != nil {
		return nil, err
	}
	result, err := s.writeSinglePort(ctx, portID, action, "", operator, 0, "", "", "", nil, applyWriteOptions(opts), &renderExtras{physical: &params})
	if err != nil {
		return nil, err
	}
	// 审计 after_value：物理参数回显（handler buildAfterValue 读 Extra）
	if result != nil {
		if result.Extra == nil {
			result.Extra = make(map[string]interface{})
		}
		switch action {
		case portcollection.ActionSetSpeed:
			if params.SpeedMode == portcollection.SpeedModeAuto {
				result.Extra["speed"] = "auto"
			} else {
				result.Extra["speed"] = params.Speed
			}
		case portcollection.ActionSetDuplex:
			result.Extra["duplex"] = params.Duplex
		case portcollection.ActionSetMTU:
			result.Extra["mtu"] = params.MTU
		}
	}
	return result, nil
}
