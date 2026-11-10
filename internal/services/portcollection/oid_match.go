package portcollection

// B-3 / v1.56：SysObjectID 作为设备身份匹配键（精确/前缀优先、Model 子串兜底）。
//
// 设计取舍（与 physical_capability.go 的取证诚实性口径严格隔离）：
//   - SysObjectID 只用于「选择哪一型号规则/画像」，是设备识别依据，
//     **不是**物理命令在真机上被验证过的依据；绝不写进 physicalProfile.Evidence，
//     不改变 level（unverified / vendor / model）。
//   - 规则继续留 Go 常量（不落规则库 / 不建配置表），符合 Phase 143 架构裁定。
//   - 优先级：OID 精确 > OID 前缀（最长前缀） > Model 子串 > 厂商基线 > 全局默认。
//     OID 为空 / 未命中时完整回退既有 Model 语义，存量设备零回归。
//   - 本轮**不预置任何具体设备的 OID→型号结论**（真机逐型号取证属 Known Gap）；
//     机制与优先级由 *_test.go 用临时规则注入验证，生产表为空即等价纯 Model 兜底。
//
// 兼容策略：既有 matchModelRules / CheckPhysicalSupport / ResolvePhysicalProfile 等
// 函数签名保持不变（旧测试零改动）；新增 ...WithOID 变体，由 portwrite 调用点
// 传入 dev.SysObjectID。旧函数内部等价「OID 为空」路径。

import (
	"strings"
)

// MatchKey 一次能力匹配的输入（B-3 新增）。
// SysObjectID 为空表示设备未采集 OID，走 Model 子串兜底。
type MatchKey struct {
	Vendor      string
	Model       string
	SysObjectID string
}

// MatchedBy 描述本次匹配命中的依据（仅用于诊断/审计，不进 Evidence）。
type MatchedBy string

const (
	MatchedByOIDExact    MatchedBy = "sysoid_exact"
	MatchedByOIDPrefix   MatchedBy = "sysoid_prefix"
	MatchedByModelSubstr MatchedBy = "model_substr"
	MatchedByNone        MatchedBy = "none"
)

// normalizeOID 去首尾空白与前导点，统一点分数字串形。
func normalizeOID(oid string) string {
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(oid), "."))
}

// oidMatches 判断某条规则的 OID 列表是否命中给定 sysObjectID。
// 返回 (是否命中, 精确/前缀, 命中的前缀长度)。
// 规则项形如 "1.3.6.1.4.1.2011.2.23"（精确）或 "1.3.6.1.4.1.2011."（前缀，以点结尾）。
func oidMatches(ruleOIDs []string, sysObjectID string) (bool, MatchedBy, int) {
	oid := normalizeOID(sysObjectID)
	if oid == "" {
		return false, MatchedByNone, 0
	}
	bestPrefixLen := 0
	for _, raw := range ruleOIDs {
		r := normalizeOID(raw)
		if r == "" {
			continue
		}
		if strings.HasSuffix(r, ".") {
			// 前缀项（以点结尾）：最长前缀优先
			p := strings.TrimSuffix(r, ".")
			if strings.HasPrefix(oid, p) && len(p) > bestPrefixLen {
				bestPrefixLen = len(p)
			}
			continue
		}
		if oid == r {
			return true, MatchedByOIDExact, len(r) // 精确匹配立即胜出
		}
	}
	if bestPrefixLen > 0 {
		return true, MatchedByOIDPrefix, bestPrefixLen
	}
	return false, MatchedByNone, 0
}

// modelKeyMatches 判断 (model, sysObjectID) 是否命中某条规则的键集合。
// 优先级：OID 精确 > OID 前缀 > Model 子串。OID 命中时不再看 Model。
// 返回 (命中, 命中方式)。供 matchModelRulesWithOID / resolveModelPhysicalProfileWithOID 共用。
func modelKeyMatches(modelSubstrs, sysObjectIDs []string, model, sysObjectID string) (bool, MatchedBy) {
	if ok, by, _ := oidMatches(sysObjectIDs, sysObjectID); ok {
		return true, by
	}
	if model == "" {
		return false, MatchedByNone
	}
	lower := strings.ToLower(model)
	for _, substr := range modelSubstrs {
		if strings.Contains(lower, substr) {
			return true, MatchedByModelSubstr
		}
	}
	return false, MatchedByNone
}
