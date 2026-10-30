package asset

import (
	"net"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// ============================================================================
// R3 / Phase 44 — 例外规则纯函数匹配器
//
// D-R3-A1-03 锁定性能架构：DetectLayer3 批量检测循环**不查 GiST 索引**,
// 入口一次性 `SELECT * FROM sys_reconciliation_exception WHERE is_active=0
// AND deleted_at IS NULL` 预加载所有 active 规则到内存,逐资产用 Go
// `net.ParseCIDR` + `ipNet.Contains` 内存匹配(参考 internal/middleware/apikey.go:110-141)。
//
// GiST inet_ops 索引(migration_174)留给命中测试工具的单点查询(service.MatchTest)。
//
// 参考:
//   - 44-RESEARCH.md §Pattern 1 (Layer 3.5 内存 CIDR 匹配) + §Pattern 2 (多规则合并)
//   - 44-CONTEXT.md D-R3-A1-03 / A2-01 / A2-02 / A3-01 / A3-02
//   - internal/middleware/apikey.go:110-141 (net.ParseCIDR + ipNet.Contains 现成模式)
// ============================================================================

// compiledRule 预编译规则:原始规则 + 已 ParseCIDR 的 *net.IPNet
//
// net.IPNet 已是 Go stdlib 原生结构,Contains(ip) 直接判断 IP ∈ CIDR,
// 原生支持 IPv4/IPv6 双栈(无需自己写位运算)。
type compiledRule struct {
	rule  models.SysReconciliationException
	ipNet *net.IPNet // net.ParseCIDR(rule.IPRange) 预编译结果
}

// severityOrder 严重度→整数序映射(D-R3-A2-02 锁定)
//
// 数字越小严重度越低(low=0 / medium=1 / high=2 / critical=3)。
// 用于:
//   - applySkipSeverity: 当前 severity 降一级(level-1 对应的 key)
//   - mergeActions severity_override: 取最低(min(severityOrder[ov], severityOrder[sev]))
var severityOrder = map[string]int{
	"low":      0,
	"medium":   1,
	"high":     2,
	"critical": 3,
}

// applySkipSeverity skip_severity 降级一级(D-R3-A2-02)
//
// 降级链: critical → high → medium → low,low 不再降。
// 未知 severity 兜底返回 low(防御性,业务侧不会传未知值)。
func applySkipSeverity(s string) string {
	level, ok := severityOrder[s]
	if !ok || level <= 0 {
		return "low"
	}
	// 找 level-1 对应的 key(降一级)
	for k, v := range severityOrder {
		if v == level-1 {
			return k
		}
	}
	return s // 兜底(逻辑上不可达)
}

// preloadActiveRules 一次性预加载所有 active 规则到内存(D-R3-A1-03)
//
// 行为:
//   - WHERE is_active = 0 (Status Convention: 0=启用) AND deleted_at IS NULL
//   - 逐条 net.ParseCIDR(rule.IPRange):
//       * 解析失败 → logrus.Warnf 跳过(不阻塞检测,人工修复)
//       * 解析成功 → append 到结果切片
//   - 返回 []compiledRule,DetectLayer3 循环内零 DB 查询
//
// 注意:此函数是包级函数(非 receiver),DetectLayer3 / MatchTest 都可调用。
// 在 MatchTest 场景也用同一函数预加载,保持一致性。
func preloadActiveRules(db *gorm.DB) []compiledRule {
	var rules []models.SysReconciliationException
	if err := db.Where("is_active = ? AND deleted_at IS NULL", 0).Find(&rules).Error; err != nil {
		logrus.WithError(err).Warn("[reconciliation] preloadActiveRules 查询失败,返回空切片")
		return nil
	}
	out := make([]compiledRule, 0, len(rules))
	for _, r := range rules {
		_, ipNet, err := net.ParseCIDR(r.IPRange)
		if err != nil {
			logrus.Warnf("[reconciliation] 例外规则 %s CIDR 解析失败(%s): %v", r.ID, r.IPRange, err)
			continue
		}
		out = append(out, compiledRule{rule: r, ipNet: ipNet})
	}
	return out
}

// matchException 纯函数:对单条资产 CIDR 匹配 + 多规则合并
//
// 入参:
//   - rules: 预加载的 active 规则切片
//   - assetIP: 资产 IP 文本(若非法直接返回空)
//   - assetUserID: 资产责任人 user_id(空字符串表示未知;dept/user scope 规则不命中)
//   - conflictType: 当前冲突类型(A/B/C/D/E/F;调用方保证非 A,因 A 不入主表)
//
// 返回:
//   - matchedRuleID: 首条命中规则 ID(空表示无命中,审计指向触发规则)
//   - appliedActions: 合并后的 actions 并集(无命中时 nil)
//   - finalSeverity: 合并后 severity(无命中时为空字符串)
//   - isSilence: 合并后 actions 是否含 silence(无命中时 false)
//
// 关键(D-R3-A2-01/02):
//   - matchedRuleID 取首条命中(审计可追溯),但 actions/severity 取**所有命中规则**的合并结果
//   - 合并顺序: step1 skip_severity 降级 → step2 severity_override 取最低 → step3 actions UNION
//
// 注:originalSeverity 默认 "medium"(命中测试工具的基线);DetectLayer3 调用方
// 应在调用前用 ComputeSeverity(conflictType) 算出真实 severity,通过
// matchExceptionWithSeverity 显式传入,本便捷函数保留 medium 兼容现有测试。
func matchException(rules []compiledRule, assetIP string, assetUserID string, conflictType string) (
	matchedRuleID string, appliedActions pq.StringArray, finalSeverity string, isSilence bool,
) {
	return matchExceptionWithSeverity(rules, assetIP, assetUserID, conflictType, "medium")
}

// matchExceptionWithSeverity 显式传入原始 severity 的版本(DetectLayer3 用)
//
// DetectLayer3 调用前已用 ComputeSeverity(conflictType) 算出真实 severity,
// 传入后 mergeActions 据此降级 / 覆盖。
func matchExceptionWithSeverity(rules []compiledRule, assetIP string, assetUserID string, conflictType string, originalSeverity string) (
	matchedRuleID string, appliedActions pq.StringArray, finalSeverity string, isSilence bool,
) {
	ip := net.ParseIP(assetIP)
	if ip == nil {
		return "", nil, "", false
	}

	// 收集所有命中规则(遍历全部,因为多规则合并需要全部)
	var matched []compiledRule
	for _, r := range rules {
		if !r.ipNet.Contains(ip) {
			continue
		}
		// ConflictTypes 空数组匹配全部 B-F(D-R3-A3-02)
		if len(r.rule.ConflictTypes) > 0 {
			found := false
			for _, ct := range r.rule.ConflictTypes {
				if ct == conflictType {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		// ScopeType 双条件(D-R3-A3-01):
		//   - global: 仅 IP CIDR 匹配即生效
		//   - dept/user: 需 IP CIDR 命中 AND assetUserID == rule.ScopeID
		if r.rule.ScopeType == "dept" || r.rule.ScopeType == "user" {
			if r.rule.ScopeID == nil || assetUserID == "" || *r.rule.ScopeID != assetUserID {
				continue
			}
		}
		matched = append(matched, r)
	}

	if len(matched) == 0 {
		return "", nil, "", false
	}

	// 审计指向首条命中规则(规则变更后历史记录仍可回溯到此 ID)
	matchedRuleID = matched[0].rule.ID

	// 合并 actions + severity(D-R3-A2-01/02)
	actions, sev, silence := mergeActions(originalSeverity, matched, conflictType)
	return matchedRuleID, actions, sev, silence
}

// mergeActions 多规则合并算法(D-R3-A2-01/02 verbatim)
//
// 入参:
//   - originalSeverity: 调用方提供的原始 severity(如 ComputeSeverity(conflictType))
//   - matched: 全部命中规则
//   - conflictType: 当前冲突类型(目前未使用,预留 future scope 扩展)
//
// 降级链(D-R3-A2-02): 原始severity --skip_severity--> 降一级 --severity_override--> 取更宽
//
// 返回:
//   - actions: 合并后的 actions 并集(去重)
//   - finalSeverity: 最终 severity
//   - isSilence: actions 是否含 silence
func mergeActions(originalSeverity string, matched []compiledRule, conflictType string) (
	actions pq.StringArray, finalSeverity string, isSilence bool,
) {
	_ = conflictType // 预留扩展(目前未使用,但接口对齐 D-R3-A2-01/02)

	// step 1: 是否触发 skip_severity(任一规则含此 action 即触发)
	skipTriggered := false
	for _, r := range matched {
		for _, a := range r.rule.ExceptionActions {
			if a == "skip_severity" {
				skipTriggered = true
				break
			}
		}
		if skipTriggered {
			break
		}
	}
	sev := originalSeverity
	if skipTriggered {
		sev = applySkipSeverity(sev)
	}

	// step 2: severity_override 取最低(最宽松,D-R3-A2-01)
	for _, r := range matched {
		if r.rule.SeverityOverride != nil {
			ov := *r.rule.SeverityOverride
			ovLevel, ok := severityOrder[ov]
			if !ok {
				continue // 非法 override 跳过(DB CHECK 已兜底,这里双重保险)
			}
			sevLevel, sevOK := severityOrder[sev]
			if !sevOK {
				continue
			}
			if ovLevel < sevLevel {
				sev = ov
			}
		}
	}
	finalSeverity = sev

	// step 3: actions 取并集(去重)
	seen := map[string]struct{}{}
	for _, r := range matched {
		for _, a := range r.rule.ExceptionActions {
			if _, ok := seen[a]; !ok {
				seen[a] = struct{}{}
				actions = append(actions, a)
			}
		}
	}

	// step 4: isSilence 判定
	for _, a := range actions {
		if a == "silence" {
			isSilence = true
			break
		}
	}
	return actions, finalSeverity, isSilence
}
