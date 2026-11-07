// Package operations: Phase 131 缓存 key scope 维度结构断言（PMODEL-01）。
//
// 命名前缀 ScopeRead* 避免与其它 agent 的测试/helper 冲突。
package operations

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xingran-next/xingran-go-backend/pkg/datascope"
)

// ScopeReadCtx 构造带 ScopeResult 的请求 ctx。
func ScopeReadCtx(r datascope.ScopeResult) context.Context {
	return datascope.WithContext(context.Background(), r)
}

func TestScopeRead_ScopedFloorTreeKey_Segments(t *testing.T) {
	// 未注入 ctx（存量单测 / 内部任务）→ 旧精确 key，零回归
	assert.Equal(t, "floor:tree", scopedFloorTreeKey(context.Background()))

	// 已注入：All 档短路 key 也必须含 scope 段（LOCKED §1 关键不变量 3）
	allKey := scopedFloorTreeKey(ScopeReadCtx(datascope.AllScope("u-super")))
	assert.Equal(t, "floor:tree:sall", allKey)
	assert.Contains(t, allKey, ":sall")

	// FailClosed / Self / 不同 DeptIDs 各自不同
	closedKey := scopedFloorTreeKey(ScopeReadCtx(datascope.FailClosedScope("u")))
	selfKey := scopedFloorTreeKey(ScopeReadCtx(datascope.ScopeResult{
		ResourceHook: datascope.HookSelfUser, UserID: "uA",
	}))
	d1Key := scopedFloorTreeKey(ScopeReadCtx(datascope.ScopeResult{DeptIDs: []string{"d1"}}))
	d2Key := scopedFloorTreeKey(ScopeReadCtx(datascope.ScopeResult{DeptIDs: []string{"d2"}}))

	all := []string{allKey, closedKey, selfKey, d1Key, d2Key}
	seen := map[string]bool{}
	for _, k := range all {
		assert.True(t, strings.HasPrefix(k, "floor:tree:s"), "生产路径 key 必须带 scope 段: "+k)
		assert.False(t, seen[k], "不同 scope 不应碰撞: "+k)
		seen[k] = true
	}
	assert.Contains(t, closedKey, ":sclosed")
	assert.NotEqual(t, d1Key, d2Key, "不同可见集楼层树不共享缓存（防跨范围泄漏）")

	// 相同 scope 稳定
	assert.Equal(t, d1Key, scopedFloorTreeKey(ScopeReadCtx(datascope.ScopeResult{DeptIDs: []string{"d1"}})))
}

func TestScopeRead_ScopedFloorBuildingKey_Segments(t *testing.T) {
	// 未注入 → 旧 key，零回归
	assert.Equal(t, "floor:building:B1", scopedFloorBuildingKey(context.Background(), "B1"))

	allKey := scopedFloorBuildingKey(ScopeReadCtx(datascope.AllScope("u")), "B1")
	assert.Equal(t, "floor:building:sall:B1", allKey)
	d1 := scopedFloorBuildingKey(ScopeReadCtx(datascope.ScopeResult{DeptIDs: []string{"d1"}}), "B1")
	d2 := scopedFloorBuildingKey(ScopeReadCtx(datascope.ScopeResult{DeptIDs: []string{"d2"}}), "B1")
	assert.NotEqual(t, d1, d2, "不同可见集同一楼宇的楼层缓存不共享")
	assert.True(t, strings.HasPrefix(d1, "floor:building:s"))
	// 同一 scope 不同 buildingID 仍不同
	d1B2 := scopedFloorBuildingKey(ScopeReadCtx(datascope.ScopeResult{DeptIDs: []string{"d1"}}), "B2")
	assert.NotEqual(t, d1, d1B2)
}

func TestScopeRead_BuildScopedDropdownCacheKey_StableAndIsolated(t *testing.T) {
	ctxD1 := ScopeReadCtx(datascope.ScopeResult{DeptIDs: []string{"d1"}})
	ctxD2 := ScopeReadCtx(datascope.ScopeResult{DeptIDs: []string{"d2"}})

	k1 := BuildScopedDropdownCacheKey(ctxD1, "floor", map[string]any{"buildingId": "B1"})
	k1Repeat := BuildScopedDropdownCacheKey(
		ScopeReadCtx(datascope.ScopeResult{DeptIDs: []string{"d1"}}),
		"floor", map[string]any{"buildingId": "B1"})
	assert.Equal(t, k1, k1Repeat, "相同 scope + 相同 filters 必须稳定")

	k2 := BuildScopedDropdownCacheKey(ctxD2, "floor", map[string]any{"buildingId": "B1"})
	assert.NotEqual(t, k1, k2, "PMODEL-01：不同可见集 dropdown 缓存 key 必须隔离")

	// filters 仍参与 key（旧行为不破）
	k1OtherFilter := BuildScopedDropdownCacheKey(ctxD1, "floor", map[string]any{"buildingId": "B2"})
	assert.NotEqual(t, k1, k1OtherFilter)

	// All 档短路也含 all 段
	allKey := BuildScopedDropdownCacheKey(ScopeReadCtx(datascope.AllScope("u")), "floor", nil)
	assert.Contains(t, allKey, ":sall:")
	// 未注入（存量测试）也含 scope 段，且 prefix 仍可被 dropdown:floor:* 命中
	bgKey := BuildScopedDropdownCacheKey(context.Background(), "floor", nil)
	assert.Contains(t, bgKey, ":sall:")
	assert.True(t, strings.HasPrefix(bgKey, "dropdown:floor:"))

	// 旧签名逐字行为不变（dropdown_helper_test.go 既有断言守护）
	assert.Equal(t, "dropdown:building:all", BuildDropdownCacheKey("building", nil))
}
