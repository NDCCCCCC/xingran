package portwrite

// FIX-04（v1.53 Phase 138）：物理参数「速率 / 双工 / MTU 调整」service 层。
// Phase 143 (PORT-02)：入口 validator 接 (vendor, model, action, params)，
// 与 portcollection.CheckPhysicalSupport 同源（型号级能力矩阵）；审计 Extra
// 的写入抽成公共 applyPhysicalAuditExtra 供单条与批量共用。
//
// 沿用 v1.20.1 / v1.51 端口写基建：sentinel validator（入口真相源）→
// writeSinglePort（pre-state / 互斥 / 预检 / RenderCommand / ExecuteCustom）→
// PortResult.Extra 写审计 after_value。与 PortSecWrite / TrunkWrite 同构。

import (
	"context"
	"errors"
	"fmt"

	"github.com/xingran-next/xingran-go-backend/internal/models"
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

// isPhysicalAction 物理三动作判定（单条 / 批量 / 预览共用，避免三份 switch 漂移）。
func isPhysicalAction(action Action) bool {
	switch action {
	case portcollection.ActionSetSpeed, portcollection.ActionSetDuplex, portcollection.ActionSetMTU:
		return true
	}
	return false
}

// physicalTemplateParams 把 PhysicalParams 转成 portcollection 渲染载体（校验/渲染同源）。
func physicalTemplateParams(p PhysicalParams) portcollection.PortTemplateParams {
	return portcollection.PortTemplateParams{
		Speed:     p.Speed,
		SpeedMode: p.SpeedMode,
		Duplex:    p.Duplex,
		MTU:       p.MTU,
	}
}

// validatePhysicalRequest 物理参数入口校验（真相源；dry-run 与实跑同源）。
//
// Phase 143：接 (vendor, model, action, params)，走 portcollection.CheckPhysicalSupport
// —— 型号级能力矩阵（型号命中 / 厂商回退 / 未知 fail-safe / 枚举范围）是唯一真相源，
// renderer 的 renderPhysical 调同一个函数，保证「校验放行 ⇔ 渲染接受」。
// 返回的画像供 PreviewResult.evidenceLevel / 审计 Extra 取证级别回显。
func validatePhysicalRequest(vendor models.DeviceVendor, model string, action Action, p PhysicalParams) (portcollection.PhysicalEvidence, error) {
	if !isPhysicalAction(action) {
		return portcollection.PhysicalEvidence{}, fmt.Errorf("%w: %s is not a physical action", ErrPhysicalParamsInvalid, action)
	}
	prof, err := portcollection.CheckPhysicalSupport(vendor, model, action, physicalTemplateParams(p))
	if err != nil {
		// 把能力矩阵错误翻译成 portwrite 400 sentinel（handler 已有翻译表），
		// 参数越界按动作细分带上原画像信息，便于既有测试与前端定位。
		return portcollection.PhysicalEvidence{}, translatePhysicalSupportError(action, p, err)
	}
	return portcollection.PhysicalEvidence{
		Level:    portcollection.PhysicalEvidenceLevel(prof),
		Evidence: prof.Evidence,
	}, nil
}

// translatePhysicalSupportError 把 CheckPhysicalSupport 的错误映射到 400 sentinel。
// ErrActionNotSupported（厂商/型号显式拒绝）原样透传（handler 已有 400 分支）；
// ErrPhysicalParamInvalid 按动作细分带上合法值域文案。
func translatePhysicalSupportError(action Action, p PhysicalParams, err error) error {
	if errors.Is(err, portcollection.ErrActionNotSupported) {
		return err
	}
	switch action {
	case portcollection.ActionSetSpeed:
		return fmt.Errorf("%w: %d / speedMode=%q (%v)", ErrSpeedInvalid, p.Speed, p.SpeedMode, err)
	case portcollection.ActionSetDuplex:
		return fmt.Errorf("%w: %q (%v)", ErrDuplexInvalid, p.Duplex, err)
	case portcollection.ActionSetMTU:
		return fmt.Errorf("%w: %d (%v)", ErrMTUOutOfRange, p.MTU, err)
	}
	return fmt.Errorf("%w: %v", ErrPhysicalParamsInvalid, err)
}

// applyPhysicalAuditExtra 把物理参数回显写进 PortResult.Extra（审计 after_value 载体）。
//
// Phase 143：从 FIX-04 单条 PhysicalWrite 的成功后写 Extra 逻辑抽成公共函数，
// 单条与批量共用 —— 修「批量 audit.after_value 恒 {}」缺口，且静态上只有一份实现。
// buildAfterValue 仍从 Extra 读 speed/duplex/mtu；额外写 physicalEvidence（取证依据，
// 空串 = 未取证）与 physicalEvidenceLevel（unverified|model|vendor）。
func applyPhysicalAuditExtra(action Action, params PhysicalParams, evidence portcollection.PhysicalEvidence, result *PortResult) {
	if result == nil || !isPhysicalAction(action) {
		return
	}
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
	result.Extra["physicalEvidence"] = evidence.Evidence
	result.Extra["physicalEvidenceLevel"] = evidence.Level
}

// PhysicalWrite 物理参数写入口（FIX-04；单端点三动作）。
//
// 管线与 PortSecWrite 一致：validator → writeSinglePort（复用 renderExtras.physical
// 透传参数到 executeWrite）→ 成功后把参数回显进 PortResult.Extra 供审计
// after_value（buildAfterValue 从 Extra 读 speed/duplex/mtu）。
func (s *portWriteServiceImpl) PhysicalWrite(ctx context.Context, portID string, action Action, params PhysicalParams, operator string, opts ...WriteOption) (*PortResult, error) {
	if !isPhysicalAction(action) {
		return nil, fmt.Errorf("%w: %s is not a physical action", ErrPhysicalParamsInvalid, action)
	}

	// 入口先取设备型号（能力矩阵判定需要 vendor + model）。与 writeSinglePort 同样的
	// 两步查询：端口行 → 设备行。端口行不存在 / 设备缺失时用零值 vendor/model 走全局
	// 默认画像（不阻塞，保持 D-13 fallback 语义；真正下发时 writeSinglePort 会按既有
	// 逻辑返回 ErrDeviceNotFound）。
	var port models.DevicePortStatus
	var dev models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("id = ?", portID).First(&port).Error; err == nil {
		_ = s.db.WithContext(ctx).Where("id = ?", port.DeviceID).First(&dev).Error
	}

	evidence, err := validatePhysicalRequest(dev.Vendor, dev.Model, action, params)
	if err != nil {
		return nil, err
	}
	result, err := s.writeSinglePort(ctx, portID, action, "", operator, 0, "", "", "", nil, applyWriteOptions(opts), &renderExtras{physical: &params})
	if err != nil {
		return nil, err
	}
	// 审计 after_value：物理参数回显（公共函数，与批量共用）
	applyPhysicalAuditExtra(action, params, evidence, result)
	return result, nil
}
