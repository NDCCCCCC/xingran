package asset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComputeByWorkstation_MatcherOverrideReachesResponse 锁住 v1.55 收档登记的
// computeByWorkstation append 顺序缺陷（quick 261109-wdu 清偿）。
//
// 缺陷形态：循环内曾把 `assetItems = append(assetItems, item)` 放在 per-asset
// MatchException 覆盖 item.ExceptionRuleID / item.AppliedActions **之前**，切片里
// 存的是未被覆盖的副本 → matcher 结果不进 resp.Assets。修复 = append 后移到
// matcher 块之后。
//
// 红绿判据：fixture 里 rec-b 的 MV 值 exception_rule_id='rule-1'、
// applied_actions 为空；matcher 命中 seedOldHitRule 的 'rule-old'（actions=no_alert）
// 后应覆盖。修复前断言拿到 rule-1 / 空 actions（红），修复后拿到 rule-old /
// no_alert（绿）。复用 newContractFixture（reconciliation_cache_contract_test.go
// 同包，含真实 matcher + spy 缓存），零额外基建。
func TestComputeByWorkstation_MatcherOverrideReachesResponse(t *testing.T) {
	f := newContractFixture(t, true) // seedOldHitRule=true → 10.1.0.5 命中 rule-old

	resp, err := f.reconSvc.GetByWorkstation(f.ctx, "ws-1", "7d")
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Assets, 2)

	byCode := make(map[string]AssetHealthItem, len(resp.Assets))
	for _, it := range resp.Assets {
		byCode[it.AssetCode] = it
	}
	a1, ok := byCode["SN-A"]
	require.True(t, ok, "fixture 应含资产 SN-A")
	require.Equal(t, "10.1.0.5", a1.IP)
	require.Equal(t, "B", a1.ConflictType, "per-asset matcher 的前置条件")

	// matcher 覆盖后的值必须真正进入响应（append 在覆盖之后）。
	require.NotNil(t, a1.ExceptionRuleID, "命中规则后 ExceptionRuleID 不应为空")
	assert.Equal(t, "rule-old", *a1.ExceptionRuleID,
		"append 早于 matcher 覆盖会残留 MV 的 'rule-1'，matcher 结果必须进响应")
	assert.Contains(t, a1.AppliedActions, "no_alert",
		"matcher 的 AppliedActions 覆盖必须进响应（修复前为空）")
}
