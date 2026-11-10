package portcollection

// Phase 143 (PORT-02)：型号级物理参数（速率/双工/MTU）能力矩阵框架。
//
// 与 device_capability.go 的 vendorCapabilities（真机事实唯一真相源，测试锁定）同构：
// 只用 Go 代码常量表达规则，不引入配置 / DB / 规则库。
// 型号 key：B-3 / v1.56 起支持 SysObjectID 精确/前缀优先 + NetworkDevice.Model
// 子串兜底（见 oid_match.go）；规则本身仍留 Go 常量，不落规则库。
// SysObjectID 作为设备属性落在 sys_network_device，是**识别依据**而非真机取证依据，
// 绝不写进 physicalProfile.Evidence（下方取证诚实性口径不变）。
//
// ⚠️ 取证诚实性（本里程碑不做真机逐型号取证，严禁伪造闭环）：
//   - physicalProfile.Evidence == "" 表示「未取证」，绝不写成「已验证」；
//   - 厂商基线 profile 的 Evidence 明确标注「厂商基线，未逐型号取证」；
//   - 「默认放行 ≠ 有证据」：全局默认 profile 与 FIX-04 现状等价（三家交集），
//     Evidence="" + level="unverified"，不阻断写操作，仅如实登记结论级别。
//
// 回退链（严格对齐既有 lookupSupport 优先级，见 CheckPhysicalSupport）：
//   型号 modelRules.unsupported 命中 → 拒绝（400，Evidence 必含依据）
//   → 型号 modelPhysicalRule 命中 → 用该型号 profile
//   → 厂商 vendorUnsupported 命中 → 拒绝
//   → 厂商 entry.physical 基线 → 用厂商 profile（Evidence 标「厂商基线，未逐型号取证」）
//   → 全局默认（FIX-04 现状，Evidence=""）。

import (
	"fmt"
	"sort"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ErrPhysicalParamInvalid 物理参数超出命中 profile 允许的范围，或动作不是物理动作。
// 写路径 validator 把它翻译成 portwrite 的 400 sentinel；renderer 是同源的第二道防线。
var ErrPhysicalParamInvalid = fmt.Errorf("portcollection: physical param not allowed by capability profile")

// physicalProfile 单条物理能力画像（型号级 / 厂商级 / 全局默认共用同一结构）。
//
// 字段语义：
//   - AllowedSpeeds / AllowedDuplexes / MTUMin / MTUMax：校验与渲染共用的合法值域
//   - SpeedCommand / DuplexCommand / MTUCommand：命令形态（fmt.Sprintf 模板），
//     型号命中时可改变命令体（如华为 CE 系 jumboframe enable），默认三家一致
//   - RequiresView：true = interface view（wrapInterface 加 `interface <name>` 前缀）；
//     false = 系统视图命令，不加前缀也不追加 view 退出命令
//   - Evidence：真机依据文本；**空串 = 未取证**，严禁写成「已验证」
//   - level（私有）："unverified" | "model" | "vendor"，供 PreviewResult.evidenceLevel 回显；
//     不作为导出字段，避免与裁决给定的结构清单混淆，用 PhysicalEvidenceLevel 读取
type physicalProfile struct {
	AllowedSpeeds   []int
	AllowedDuplexes []string
	MTUMin          int
	MTUMax          int
	SpeedCommand    string
	DuplexCommand   string
	MTUCommand      string
	RequiresView    bool
	Evidence        string
	level           string
}

// modelPhysicalRule 型号级物理画像覆盖（与 modelCapabilityRule 同构，子串匹配）。
// profile 由真机取证后经 code review + *_test.go 锁定写入；本里程碑不预置任何型号结论。
type modelPhysicalRule struct {
	matchModelSubstr []string
	// matchSysObjectIDs B-3 / v1.56：OID 精确/前缀匹配（优先于 Model 子串）。
	// 设备识别依据，不是真机取证依据，不进 profile.Evidence。
	matchSysObjectIDs []string
	profile           physicalProfile
}

// physicalVendorUnsupportedReason Cisco / 迈普 物理动作显式拒绝依据。
//
// ⚠️ 依据是「渲染层未实现 + 无真机取证依据」，不是「真机证明不支持」—— 不伪造真机结论。
// 复用 device_capability.go ciscoUnsupportedReason 的诚实标注口径。
const physicalVendorUnsupportedReason = "该厂商物理参数端口写不支持：渲染层未实现 speed/duplex/mtu 渲染，" +
	"且无真机取证依据（Phase 143 裁定：矩阵显式标记 vendorUnsupported，不以推测冒充真机结论）"

// defaultPhysicalProfile 全局默认画像 = FIX-04 现状（三家交集），保证单条物理写零回归。
//
//   - 速率 {10,100,1000,10000} / 双工 half|full|auto / MTU [1280,9216]
//   - 命令体 speed %s / duplex %s / mtu %d（接口视图）
//   - Evidence="" → level="unverified"（未取证，不冒充真机结论）
func defaultPhysicalProfile() physicalProfile {
	speeds := make([]int, 0, len(allowedSpeeds))
	for s := range allowedSpeeds {
		speeds = append(speeds, s)
	}
	sort.Ints(speeds)
	return physicalProfile{
		AllowedSpeeds:   speeds,
		AllowedDuplexes: []string{DuplexHalf, DuplexFull, DuplexAuto},
		MTUMin:          MTUMin,
		MTUMax:          MTUMax,
		SpeedCommand:    "speed %s",
		DuplexCommand:   "duplex %s",
		MTUCommand:      "mtu %d",
		RequiresView:    true,
		Evidence:        "",
		level:           "unverified",
	}
}

// PhysicalEvidence 物理能力画像的取证结论（供 service 层 validator → dry-run 预览 /
// 审计 Extra 透传，不引入第二套判定）。
//
// Evidence 空串 = 未取证（严禁写成「已验证」）；Level 为 "unverified" | "model" | "vendor"。
type PhysicalEvidence struct {
	Level    string
	Evidence string
}

// PhysicalEvidenceLevel 返回画像的取证级别（"unverified" | "model" | "vendor"）。
// 供 dry_run PreviewResult.evidenceLevel 回显；零值兜底 "unverified"。
func PhysicalEvidenceLevel(p physicalProfile) string {
	if p.level == "" {
		return "unverified"
	}
	return p.level
}

// resolveModelPhysicalProfile 型号子串命中 → 型号画像（大小写不敏感）。
// 与 matchModelRules 语义一致：model 为空或厂商不在矩阵内 → 不命中。
func resolveModelPhysicalProfile(vendor models.DeviceVendor, model string) (physicalProfile, bool) {
	return resolveModelPhysicalProfileWithOID(vendor, model, "")
}

// resolveModelPhysicalProfileWithOID B-3 / v1.56：型号画像匹配（OID 优先、Model 兜底）。
func resolveModelPhysicalProfileWithOID(vendor models.DeviceVendor, model, sysObjectID string) (physicalProfile, bool) {
	entry, ok := vendorCapabilities[vendor]
	if !ok || (model == "" && normalizeOID(sysObjectID) == "") {
		return physicalProfile{}, false
	}
	for _, rule := range entry.modelPhysical {
		if ok, _ := modelKeyMatches(rule.matchModelSubstr, rule.matchSysObjectIDs, model, sysObjectID); ok {
			p := rule.profile
			p.level = "model"
			return p, true
		}
	}
	return physicalProfile{}, false
}

// ResolvePhysicalProfile 解析 (vendor, model) → 物理画像（只做画像选择，不做拒绝）。
//
// 优先级：型号 modelPhysicalRule > 厂商 entry.physical 基线 > 全局默认。
// 型号/厂商级「不支持」的拒绝在 CheckPhysicalSupport 里按 lookupSupport 顺序前置处理
// （ResolvePhysicalProfile 签名不含 action，无法表达 action 级拒绝）。
//
//   - 命中型号画像：level="model"（Evidence 由配置者填真机依据）
//   - 命中厂商基线：level="vendor"，Evidence 兜底标「厂商基线，未逐型号取证」
//   - 全局默认：level="unverified"，Evidence=""（未取证，与 FIX-04 等价）
func ResolvePhysicalProfile(vendor models.DeviceVendor, model string) physicalProfile {
	return ResolvePhysicalProfileWithOID(vendor, model, "")
}

// ResolvePhysicalProfileWithOID B-3 / v1.56：解析 (vendor, model, sysObjectID) → 物理画像。
// OID 优先命中型号画像；未命中完整回退 Model 子串 / 厂商基线 / 全局默认（零回归）。
// 注意：返回画像的 Evidence / level 语义不变，OID 命中不构成真机取证。
func ResolvePhysicalProfileWithOID(vendor models.DeviceVendor, model, sysObjectID string) physicalProfile {
	if p, ok := resolveModelPhysicalProfileWithOID(vendor, model, sysObjectID); ok {
		return p
	}
	if entry, ok := vendorCapabilities[vendor]; ok && entry.physical != nil {
		p := *entry.physical
		p.level = "vendor"
		if p.Evidence == "" {
			p.Evidence = "厂商基线，未逐型号取证"
		}
		return p
	}
	return defaultPhysicalProfile()
}

// CheckPhysicalSupport 物理动作校验入口（能力 + 参数范围 + 画像解析，唯一真相源）。
//
// 写路径入口 validator 与 renderer（renderPhysical）共用本函数，保证
// 「校验放行 ⇔ 渲染接受」：两处拿到的是同一份 physicalProfile 的命令形态与值域。
//
// 回退链严格对齐 lookupSupport（型号 unsupported > 型号 supported > 厂商 unsupported >
// 基线 > 默认）：
//  1. 型号 modelRules.unsupported[action] 命中 → ErrActionNotSupported（Evidence 必含依据）
//  2. 型号 modelPhysicalRule 命中 → 用型号画像（其优先级高于厂商级 unsupported）
//  3. 厂商 vendorUnsupported[action] 命中 → ErrActionNotSupported
//  4/5. 厂商 entry.physical 基线 / 全局默认 → ResolvePhysicalProfile
//
// params 用 PortTemplateParams 作为载体（portcollection 自有类型，避免 import portwrite 循环）；
// 仅消费 Speed / SpeedMode / Duplex / MTU 四个字段。
func CheckPhysicalSupport(vendor models.DeviceVendor, model string, action PortAction, params PortTemplateParams) (physicalProfile, error) {
	return CheckPhysicalSupportWithOID(vendor, model, action, params, "")
}

// CheckPhysicalSupportWithOID B-3 / v1.56：物理动作校验（OID 优先、Model 兜底）。
// 旧签名 CheckPhysicalSupport 委托到本函数（OID 为空），存量零回归。
// 校验/渲染同源：renderPhysical 也走本变体（传 PortTemplateParams.SysObjectID）。
func CheckPhysicalSupportWithOID(vendor models.DeviceVendor, model string, action PortAction, params PortTemplateParams, sysObjectID string) (physicalProfile, error) {
	// 1. 既有型号级「不支持」优先于一切（含型号级物理画像）
	for _, rule := range matchModelRulesWithOID(vendor, model, sysObjectID) {
		if reason, banned := rule.unsupported[action]; banned {
			return physicalProfile{}, fmt.Errorf("%w: vendor=%s model=%q action=%s — %s",
				ErrActionNotSupported, vendor, model, action, reason)
		}
	}
	// 2. 型号物理画像命中 → 型号级证据优先（对齐 lookupSupport：型号 supported 在厂商 unsupported 之前）
	if prof, matched := resolveModelPhysicalProfileWithOID(vendor, model, sysObjectID); matched {
		return validatePhysicalParams(prof, action, params)
	}
	// 3. 厂商级「不支持」（与型号无关）：Cisco / 迈普 物理动作显式标 Unsupported
	if entry, ok := vendorCapabilities[vendor]; ok {
		if reason, banned := entry.vendorUnsupported[action]; banned {
			return physicalProfile{}, fmt.Errorf("%w: vendor=%s model=%q action=%s — %s",
				ErrActionNotSupported, vendor, model, action, reason)
		}
	}
	// 4/5. 厂商基线 / 全局默认
	return validatePhysicalParams(ResolvePhysicalProfileWithOID(vendor, model, sysObjectID), action, params)
}

// validatePhysicalParams 按 physicalProfile 的值域校验参数（校验/渲染同源的核心）。
func validatePhysicalParams(prof physicalProfile, action PortAction, params PortTemplateParams) (physicalProfile, error) {
	switch action {
	case ActionSetSpeed:
		if params.SpeedMode == SpeedModeAuto {
			return prof, nil
		}
		if params.SpeedMode != "" {
			return physicalProfile{}, fmt.Errorf("%w: unknown speedMode %q", ErrPhysicalParamInvalid, params.SpeedMode)
		}
		if !prof.allowsSpeed(params.Speed) {
			return physicalProfile{}, fmt.Errorf("%w: speed %d not in %v", ErrPhysicalParamInvalid, params.Speed, prof.AllowedSpeeds)
		}
		return prof, nil
	case ActionSetDuplex:
		if !prof.allowsDuplex(params.Duplex) {
			return physicalProfile{}, fmt.Errorf("%w: duplex %q not in %v", ErrPhysicalParamInvalid, params.Duplex, prof.AllowedDuplexes)
		}
		return prof, nil
	case ActionSetMTU:
		if params.MTU < prof.MTUMin || params.MTU > prof.MTUMax {
			return physicalProfile{}, fmt.Errorf("%w: mtu %d out of range %d-%d",
				ErrPhysicalParamInvalid, params.MTU, prof.MTUMin, prof.MTUMax)
		}
		return prof, nil
	default:
		return physicalProfile{}, fmt.Errorf("%w: %s is not a physical action", ErrPhysicalParamInvalid, action)
	}
}

func (p physicalProfile) allowsSpeed(speed int) bool {
	for _, v := range p.AllowedSpeeds {
		if v == speed {
			return true
		}
	}
	return false
}

func (p physicalProfile) allowsDuplex(duplex string) bool {
	for _, v := range p.AllowedDuplexes {
		if v == duplex {
			return true
		}
	}
	return false
}
