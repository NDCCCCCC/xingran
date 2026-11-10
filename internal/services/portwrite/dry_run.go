package portwrite

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
	"github.com/xingran-next/xingran-go-backend/internal/services/portcollection"
	"gorm.io/gorm"
)

// Phase 121 (v1.51 PORTW-01)：端口写 dry-run 命令预览。
//
// 目标：在「命令已渲染（RenderCommand）、尚未进入连接层（ExecuteCustom）」的位置
// 分叉，纯模板渲染层生成"将执行的命令序列 + 风险标注"返回给前端，不产生任何
// 设备连接 / SSH 流量 / audit 行 / operlog 行。
//
// 验收契约（task-3 PORTW-01）：
//   - 覆盖 7 个既有 action（5 v1.19 + 2 v1.20.1）× 3 厂商（huawei/ruijie/h3c）差异
//   - 风险等级标注（critical/medium/low）
//   - NoOp pre-state 检测与真实写路径共享同一份 checkPreState 语义
//   - 单测证明零网络调用（mock executor 记录调用次数 == 0）

// 风险等级常量（dry-run 预览专用标注，不参与真实写路径判定）。
const (
	RiskLevelCritical = "critical" // 可能中断业务流量（shutdown / 换 VLAN）
	RiskLevelMedium   = "medium"   // 影响认证/绑定语义（dot1x / port_binding / undo_shutdown）
	RiskLevelLow      = "low"      // 仅描述性变更（description）
	RiskLevelNone     = "none"     // 无命令需要执行（NoOp 短路）
)

// PreviewRequest dry-run 预览请求。
//
// 复用批量写请求形状（单端口 = PortIDs 长度 1），前端同一套表单既能预览也能提交。
// DryRun 字段保留为请求侧显式语义标记（独立 /preview 端点时恒为 true）。
type PreviewRequest struct {
	DeviceID    string   `json:"deviceId" binding:"required"`
	Action      Action   `json:"action" binding:"required"`
	PortIDs     []string `json:"portIds" binding:"required,min=1,max=50"`
	Description string   `json:"description,omitempty"`
	VLANID      int      `json:"vlanId,omitempty" binding:"omitempty,min=1,max=4094"`
	BindOp      string   `json:"bindOp,omitempty" binding:"omitempty,oneof=add remove"`
	IPAddress   string   `json:"ipAddress,omitempty"`
	MACAddress  string   `json:"macAddress,omitempty"`
	// Phase 124 (PWX-01) trunk/hybrid 参数（action 为 trunk_* 时使用）
	NativeVLAN     int    `json:"nativeVlan,omitempty" binding:"omitempty,min=1,max=4094"`
	AllowedVLANs   []int  `json:"allowedVlans,omitempty"`
	AllowedMode    string `json:"mode,omitempty" binding:"omitempty,oneof=add remove replace"`
	HybridTagged   []int  `json:"hybridTagged,omitempty"`
	HybridUntagged []int  `json:"hybridUntagged,omitempty"`
	// Phase 125 (PWX-02) 端口安全参数：MaxMacCount 仅 portsec_max_mac；
	// portsec_static_bind 复用 macAddress + vlanId 字段
	MaxMacCount int `json:"maxMacCount,omitempty" binding:"omitempty,min=1,max=4096"`
	// FIX-04 (Phase 138) 物理参数：速率/双工/MTU（仅对应动作使用）
	Speed     int    `json:"speed,omitempty"`
	SpeedMode string `json:"speedMode,omitempty"`
	Duplex    string `json:"duplex,omitempty"`
	MTU       int    `json:"mtu,omitempty"`
	DryRun    bool   `json:"dryRun,omitempty"`
}

// physicalParamsOf 从 preview 请求抽取物理参数（非物理动作返回 nil）。
func (r PreviewRequest) physicalParamsOf() *PhysicalParams {
	switch r.Action {
	case portcollection.ActionSetSpeed, portcollection.ActionSetDuplex, portcollection.ActionSetMTU:
		return &PhysicalParams{Speed: r.Speed, SpeedMode: r.SpeedMode, Duplex: r.Duplex, MTU: r.MTU}
	}
	return nil
}

// portsecParamsOf 从 preview 请求抽取端口安全参数（非 portsec 动作返回 nil）。
func (r PreviewRequest) portsecParamsOf() *PortSecParams {
	switch r.Action {
	case portcollection.PortSecEnable, portcollection.PortSecDisable, portcollection.PortSecMaxMac, portcollection.PortSecSticky, portcollection.PortSecStaticBind:
		return &PortSecParams{MaxMACCount: r.MaxMacCount, MACAddress: r.MACAddress, VLANID: r.VLANID}
	}
	return nil
}

// trunkParamsOf 从 preview 请求抽取 trunk 参数（非 trunk 动作返回 nil）。
func (r PreviewRequest) trunkParamsOf() *TrunkParams {
	switch r.Action {
	case portcollection.TrunkSetNative, portcollection.TrunkSetAllowed, portcollection.TrunkSetHybrid:
		return &TrunkParams{
			NativeVLAN:     r.NativeVLAN,
			AllowedVLANs:   r.AllowedVLANs,
			AllowedMode:    r.AllowedMode,
			HybridTagged:   r.HybridTagged,
			HybridUntagged: r.HybridUntagged,
		}
	}
	return nil
}

// CommandPreview 单条命令的预览行。
type CommandPreview struct {
	Index     int    `json:"index"`
	Command   string `json:"command"`
	RiskLevel string `json:"riskLevel"`
	Critical  bool   `json:"critical"`
	Note      string `json:"note,omitempty"` // 说明用途，如 "进入接口视图" / "view 退出清理命令"
}

// PortPreviewResult 单端口预览结果。
type PortPreviewResult struct {
	PortID        string           `json:"portId"`
	InterfaceName string           `json:"interfaceName"`
	Status        string           `json:"status"` // "preview" | "skipped"
	NoOp          bool             `json:"noOp"`
	NoOpReason    string           `json:"noOpReason,omitempty"`
	CurrentState  string           `json:"currentState,omitempty"`
	Commands      []CommandPreview `json:"commands"`
}

// PreviewResult dry-run 预览整体响应。
type PreviewResult struct {
	DryRun     bool                `json:"dryRun"`
	DeviceID   string              `json:"deviceId"`
	DeviceName string              `json:"deviceName"`
	Vendor     string              `json:"vendor"`
	Model      string              `json:"model,omitempty"`
	Action     Action              `json:"action"`
	RiskLevel  string              `json:"riskLevel"`
	Summary    string              `json:"summary"`
	Ports      []PortPreviewResult `json:"ports"`
	// EvidenceLevel Phase 143 (PORT-02)：物理能力画像取证级别
	// "unverified" | "model" | "vendor"。非物理动作恒为空串。
	// 默认放行 ≠ 有证据：unverified 不阻断写、不冒充真机结论。
	EvidenceLevel string `json:"evidenceLevel,omitempty"`
	// PhysicalEvidence 画像依据文本；空串 = 未取证（严禁写成已验证）。
	PhysicalEvidence string `json:"physicalEvidence,omitempty"`
}

// riskLevelForAction action → 风险等级（PORTW-01 风险标注真相源）。
//
//   - shutdown / set_access_vlan：直接中断或改变业务转发路径 → critical
//   - undo_shutdown / dot1x / port_binding：改变接入控制语义但通常不影响已在线流量 → medium
//   - description：纯描述性 → low
func riskLevelForAction(action Action) string {
	switch action {
	case portcollection.ActionShutdown, portcollection.ActionSetAccessVLAN:
		return RiskLevelCritical
	case portcollection.ActionUndoShutdown,
		portcollection.ActionDot1xEnable,
		portcollection.ActionDot1xDisable,
		portcollection.ActionPortBinding:
		return RiskLevelMedium
	case portcollection.ActionDescription:
		return RiskLevelLow
	default:
		return RiskLevelMedium // 未知 action 保守标注
	}
}

// actionSummary 生成人可读摘要行。
func actionSummary(action Action, portCount int, vendor models.DeviceVendor) string {
	names := map[Action]string{
		portcollection.ActionShutdown:      "关闭端口",
		portcollection.ActionUndoShutdown:  "取消关闭端口",
		portcollection.ActionDescription:   "设置端口描述",
		portcollection.ActionDot1xEnable:   "启用 802.1X",
		portcollection.ActionDot1xDisable:  "停用 802.1X",
		portcollection.ActionSetAccessVLAN: "修改 access VLAN",
		portcollection.ActionPortBinding:   "端口 IP/MAC 绑定",
		portcollection.TrunkSetNative:      "设置 trunk native VLAN",
		portcollection.TrunkSetAllowed:     "修改 trunk allowed VLAN 列表",
		portcollection.TrunkSetHybrid:      "设置 hybrid VLAN（锐捷会先把端口切到 hybrid 模式）",
		portcollection.PortSecEnable:       "启用端口安全",
		portcollection.PortSecDisable:      "停用端口安全",
		portcollection.PortSecMaxMac:       "设置端口安全 MAC 上限",
		portcollection.PortSecSticky:       "启用 sticky MAC",
		portcollection.PortSecStaticBind:   "静态 MAC 1:1 绑定",
		// FIX-04 物理参数
		portcollection.ActionSetSpeed:  "调整端口速率",
		portcollection.ActionSetDuplex: "调整端口双工模式",
		portcollection.ActionSetMTU:    "调整端口 MTU",
	}
	name, ok := names[action]
	if !ok {
		name = string(action)
	}
	return fmt.Sprintf("%s（dry-run 预览，%d 个端口，厂商 %s，未下发任何设备命令）", name, portCount, vendor)
}

// PreviewPortWrite dry-run 入口：DB 预查询 + pre-state 检测 + RenderCommand，零 ExecuteCustom。
//
// 与真实写路径（writeSinglePort）共享：
//   - 入参 validator（validateVlanIdRange / validateBindOp / validateIPAddress / validateMACAddress）
//   - pre-state NoOp 语义（checkPreState 同一函数）
//   - 命令渲染（RenderCommand + expandInterfaceName + VendorExitViewCmd）
//
// 与真实写路径的差异（dry-run 不变量，单测锁定）：
//   - 不调 s.deviceExecutor.ExecuteCustom（零网络调用）
//   - 不写 audit 行 / 不调 operlog（handler 层也不写）
//   - 不触发 refreshPortStatus 后台采集
//
// 返回 sentinel error 语义与真实写路径一致：ErrPortNotFound / ErrDeviceNotFound /
// ErrDeviceUnsupportedVendor / 各 validator sentinel —— handler 层共享同一翻译表。
func (s *portWriteServiceImpl) PreviewPortWrite(ctx context.Context, req PreviewRequest, operator string) (*PreviewResult, error) {
	// Phase 143：设备查询提前到入口 validator 之前 —— 物理参数 validator 需要
	// (vendor, model) 走型号级能力矩阵（与实跑同源）。非物理动作行为不变。
	var dev models.NetworkDevice
	if err := s.db.WithContext(ctx).Where("id = ?", req.DeviceID).First(&dev).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDeviceNotFound
		}
		return nil, fmt.Errorf("query device: %w", err)
	}

	// 与真实写路径相同的入口 validator（真相源同源，防 dry-run 和实跑行为漂移）
	var physicalEvidence portcollection.PhysicalEvidence
	switch req.Action {
	case portcollection.ActionSetAccessVLAN:
		if err := validateVlanIdRange(req.VLANID); err != nil {
			return nil, err
		}
	case portcollection.ActionPortBinding:
		if err := validateBindOp(req.BindOp); err != nil {
			return nil, err
		}
		if err := validateIPAddress(req.IPAddress); err != nil {
			return nil, err
		}
		if err := validateMACAddress(req.MACAddress); err != nil {
			return nil, err
		}
	case portcollection.ActionDescription:
		if req.Description == "" {
			return nil, portcollection.ErrDescriptionEmpty
		}
		if len([]rune(req.Description)) > 80 {
			return nil, portcollection.ErrDescriptionTooLong
		}
	default:
		// Phase 124 trunk / Phase 125 portsec 入口校验（与实跑同源）
		if tp := req.trunkParamsOf(); tp != nil {
			if err := validateTrunkRequest(req.Action, *tp); err != nil {
				return nil, err
			}
		}
		if pp := req.portsecParamsOf(); pp != nil {
			if err := validatePortSecRequest(req.Action, *pp); err != nil {
				return nil, err
			}
		}
		// FIX-04 / Phase 143 物理参数入口校验（与实跑同源，接 vendor/model）
		if ph := req.physicalParamsOf(); ph != nil {
			ev, err := validatePhysicalRequest(dev.Vendor, dev.Model, req.Action, *ph)
			if err != nil {
				return nil, err
			}
			physicalEvidence = ev
		}
	}

	// Phase 129 真机修复：能力矩阵校验（S8700 等型号受限动作 dry-run 即报
	// 明确错误，不下发必失败命令）
	if err := portcollection.CheckActionSupportParams(dev.Vendor, dev.Model, req.Action, req.MACAddress); err != nil {
		return nil, err
	}

	// 厂商/(action, 参数) 支持性探测：用完整合法参数试渲染一次，把
	// ErrUnsupportedVendor / ErrUnknownAction / 参数类校验提前到 4xx。
	// 真正的逐端口渲染在 previewSinglePort 中重做（interfaceName 不同）。
	probe, err := portcollection.RenderCommand(dev.Vendor, req.Action, portcollection.PortTemplateParams{
		InterfaceName:  expandInterfaceName("GE0/0/1", dev.Vendor),
		Description:    req.Description,
		VLANID:         req.VLANID,
		BindOp:         req.BindOp,
		IPAddress:      req.IPAddress,
		MACAddress:     req.MACAddress,
		Dot1xUserLimit: nil,
		NativeVLAN:     req.NativeVLAN,
		AllowedVLANs:   req.AllowedVLANs,
		AllowedMode:    req.AllowedMode,
		HybridTagged:   req.HybridTagged,
		HybridUntagged: req.HybridUntagged,
		MaxMACCount:    req.MaxMacCount,
		// FIX-04 物理参数
		Speed:     req.Speed,
		SpeedMode: req.SpeedMode,
		Duplex:    req.Duplex,
		MTU:       req.MTU,
		// Phase 143：型号透传（校验/渲染同源）
		Model: dev.Model,
	})
	if err != nil {
		return nil, err
	}
	_ = probe

	// dot1x_enable 与真实路径同源：读采集缓存的 limit（nil 兜底由 renderer 承担）
	var dot1xLimit *int
	if req.Action == portcollection.ActionDot1xEnable {
		// 预览只需一个端口样本读 limit；取第一个端口即可
		limit, err := s.loadPortDot1xUserLimit(ctx, req.PortIDs[0])
		if err != nil {
			limit = nil // 与 EnableDot1x 相同兜底语义
		}
		dot1xLimit = limit
	}

	result := &PreviewResult{
		DryRun:     true,
		DeviceID:   dev.ID,
		DeviceName: dev.DeviceName,
		Vendor:     string(dev.Vendor),
		Model:      dev.Model,
		Action:     req.Action,
		RiskLevel:  riskLevelForActionPortSec(req.Action),
		Ports:      make([]PortPreviewResult, 0, len(req.PortIDs)),
		// Phase 143：物理动作回显取证级别（unverified|model|vendor）与依据文本。
		// 默认放行 ≠ 有证据：unverified 不阻断写、不冒充真机结论。
		EvidenceLevel:    physicalEvidence.Level,
		PhysicalEvidence: physicalEvidence.Evidence,
	}
	result.Summary = actionSummary(req.Action, len(req.PortIDs), dev.Vendor)
	// Phase 143 裁决：批量物理参数不纳入 critical 自动回滚，预览与审计显式警示，
	// 不伪造闭环。B-4 / quick 261110-161 诚实性修正：原文案说「按 audit.before_value
	// 恢复」，但 buildBeforeValue（port_write_handler.go:497-509）只含端口状态字段
	// （admin_status/dot1x_enabled/description/interface_name/vlan），不含
	// speed/duplex/mtu 物理原值，人工按它恢复不了。改为指向真实可恢复依据。
	if isPhysicalAction(req.Action) {
		result.Summary += "；⚠️ 批量物理参数无自动回滚，失败需人工按设备 running-config / 采集快照与 audit.command_sent 恢复（audit.before_value 仅含端口状态，不含物理原值）"
	}

	for _, portID := range req.PortIDs {
		pr, perr := s.previewSinglePort(ctx, &dev, portID, req, dot1xLimit)
		if perr != nil {
			return nil, perr
		}
		result.Ports = append(result.Ports, *pr)
	}
	return result, nil
}

// previewSinglePort 单端口预览：查端口行 → checkPreState → RenderCommand → 组装命令行。
func (s *portWriteServiceImpl) previewSinglePort(
	ctx context.Context, dev *models.NetworkDevice, portID string, req PreviewRequest, dot1xLimit *int,
) (*PortPreviewResult, error) {
	out := &PortPreviewResult{PortID: portID, Status: "preview", Commands: []CommandPreview{}}

	var port models.DevicePortStatus
	err := s.db.WithContext(ctx).Where("id = ?", portID).First(&port).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPortNotFound
		}
		return nil, fmt.Errorf("query port: %w", err)
	}
	out.InterfaceName = port.InterfaceName

	noop := s.checkPreState(&port, req.Action, req.Description, req.VLANID, req.BindOp, req.IPAddress, req.MACAddress)
	// Phase 125：端口安全 pre-state 与实跑同源
	if noop == nil {
		if pp := req.portsecParamsOf(); pp != nil {
			noop = checkPortSecPreState(&port, req.Action, *pp)
		}
	}
	if noop != nil {
		out.Status = "skipped"
		out.NoOp = true
		out.NoOpReason = "当前状态已匹配目标状态，无需下发"
		out.CurrentState = noop.CurrentState
		return out, nil
	}

	cmds, err := portcollection.RenderCommand(dev.Vendor, req.Action, portcollection.PortTemplateParams{
		InterfaceName:  expandInterfaceName(port.InterfaceName, dev.Vendor),
		Description:    req.Description,
		VLANID:         req.VLANID,
		BindOp:         req.BindOp,
		IPAddress:      req.IPAddress,
		MACAddress:     req.MACAddress,
		Dot1xUserLimit: dot1xLimit,
		NativeVLAN:     req.NativeVLAN,
		AllowedVLANs:   req.AllowedVLANs,
		AllowedMode:    req.AllowedMode,
		HybridTagged:   req.HybridTagged,
		HybridUntagged: req.HybridUntagged,
		MaxMACCount:    req.MaxMacCount,
		// FIX-04 物理参数
		Speed:     req.Speed,
		SpeedMode: req.SpeedMode,
		Duplex:    req.Duplex,
		MTU:       req.MTU,
		// Phase 143：型号透传（校验/渲染同源）
		Model: dev.Model,
	})
	if err != nil {
		return nil, err
	}

	risk := riskLevelForActionPortSec(req.Action)
	for i, cmd := range cmds {
		cp := CommandPreview{
			Index:     i,
			Command:   cmd,
			RiskLevel: risk,
			Critical:  risk == RiskLevelCritical,
		}
		if i == 0 {
			cp.Note = "进入接口视图"
		}
		if cmd == "switchport mode hybrid" {
			// ★ 端口模式变更必须显式出现在预览里（不能藏在命令列表中）
			// —— 锐捷 hybrid 动作会先把端口从 access/trunk 切到 hybrid 模式，
			// 这会改变转发行为，dry-run 必须让操作者看见。
			cp.Note = "★ 端口模式变更为 hybrid（会改变转发模式，锐捷 hybrid 动作的前置命令）"
			cp.RiskLevel = RiskLevelCritical
			cp.Critical = true
		}
		out.Commands = append(out.Commands, cp)
	}
	// 真实写路径末尾会追加厂商 view 退出命令（executeWrite V7 后置 exit）——预览如实呈现
	if exitCmd := portcollection.VendorExitViewCmd(dev.Vendor); exitCmd != "" {
		out.Commands = append(out.Commands, CommandPreview{
			Index:     len(out.Commands),
			Command:   exitCmd,
			RiskLevel: RiskLevelLow,
			Note:      "视图退出清理命令（真实执行时自动追加）",
		})
	}
	return out, nil
}

// sanitizePreviewCommands 供测试/调试：把命令列表压成单行字符串（与 CommandSent 同形）。
func sanitizePreviewCommands(cmds []CommandPreview) string {
	parts := make([]string, 0, len(cmds))
	for _, c := range cmds {
		parts = append(parts, c.Command)
	}
	return strings.Join(parts, " | ")
}
