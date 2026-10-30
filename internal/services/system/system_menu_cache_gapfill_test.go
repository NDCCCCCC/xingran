package system

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// =====================================================================
// Phase 74-07 收尾:menu_service BatchDelete + appendAncestorMenuIDs +
// stringPtrValue;cache_adapter adaptCacheStats/extractExtendedStats。
// 复用 menu_service_test.go 的 setupMenuServiceDB/seedMenuDirect。
// =====================================================================

func TestMenuService_BatchDelete(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	ctx := context.Background()

	root := seedMenuDirect(t, db, &models.Menu{MenuName: "root", MenuType: "M"})
	child := seedMenuDirect(t, db, &models.Menu{MenuName: "child", MenuType: "M", ParentID: &root})
	seedMenuDirect(t, db, &models.Menu{MenuName: "leaf", MenuType: "M", ParentID: &child})
	other := seedMenuDirect(t, db, &models.Menu{MenuName: "other", MenuType: "M"})

	// 空 ids
	require.ErrorContains(t, svc.BatchDelete(ctx, nil, false), "不能为空")

	// 有子菜单非级联 → 拦截
	require.ErrorContains(t, svc.BatchDelete(ctx, []string{root}, false), "存在子菜单")

	// 级联删除:root/child/leaf 全清
	require.NoError(t, svc.BatchDelete(ctx, []string{root}, true))
	for _, id := range []string{root, child} {
		_, err := svc.GetByID(ctx, id)
		require.Error(t, err, "级联后"+id+"应被删除")
	}

	// 无子菜单 → 直接删
	require.NoError(t, svc.BatchDelete(ctx, []string{other}, false))
	_, err := svc.GetByID(ctx, other)
	require.Error(t, err)
}

func TestMenuService_AppendAncestorMenuIDs(t *testing.T) {
	db := setupMenuServiceDB(t)
	svc := NewMenuService(db).(*menuService)
	ctx := context.Background()

	// 空入参 → 返空
	assert.Empty(t, svc.appendAncestorMenuIDs(ctx, nil))

	// 三层菜单 a → b → c,求 c 的祖先链 = [a, b, c]
	a := seedMenuDirect(t, db, &models.Menu{MenuName: "a", MenuType: "M"})
	b := seedMenuDirect(t, db, &models.Menu{MenuName: "b", MenuType: "M", ParentID: &a})
	c := seedMenuDirect(t, db, &models.Menu{MenuName: "c", MenuType: "M", ParentID: &b})

	// 根 → 仅自身
	got := svc.appendAncestorMenuIDs(ctx, []string{a})
	assert.Equal(t, []string{a}, got)

	// QUIRK(D-12 不修复): 实现用 map[string]bool 收集结果(menu_service.go:306),
	// 返回顺序非确定(Go map 遍历序随机;CI Linux 上复现反序,Windows 偶绿)。
	// 调用方不应依赖顺序 — 测试用 ElementsMatch 只锁成员集合。
	// 中层 b → 成员 {b, a}
	got = svc.appendAncestorMenuIDs(ctx, []string{b})
	assert.ElementsMatch(t, []string{b, a}, got)

	// 叶 c → 成员 {c, b, a}
	got = svc.appendAncestorMenuIDs(ctx, []string{c})
	assert.ElementsMatch(t, []string{c, b, a}, got)

	// 不存在的 id → 原样返回(容错)
	got = svc.appendAncestorMenuIDs(ctx, []string{"ghost-id"})
	assert.Equal(t, []string{"ghost-id"}, got)
}

func TestMenuService_StringPtrValue(t *testing.T) {
	assert.Equal(t, "", stringPtrValue(nil))
	assert.Equal(t, "v", stringPtrValue(strPtr("v")))
}

func TestCacheAdapter_AdaptAndExtract(t *testing.T) {
	// adaptCacheStats:nil → 空 stats
	got := adaptCacheStats(nil)
	require.NotNil(t, got)
	assert.Zero(t, got.Hits)

	// 全字段(int64 / int / float64 / 异常类型)
	got = adaptCacheStats(map[string]interface{}{
		"hits":      int64(10),
		"misses":    5, // int
		"count":     int64(20),
		"key_count": 15, // int
		"noise":     "ignored",
	})
	assert.Equal(t, int64(10), got.Hits)
	assert.Equal(t, int64(5), got.Misses)
	assert.Equal(t, int64(20), got.Count)
	assert.Equal(t, 15, got.KeyCount)
	assert.InDelta(t, 10.0/15.0, got.HitRate, 1e-9, "命中率由 hits/(hits+misses) 计算")

	// extractExtendedStats:只透传已知子表
	ext := extractExtendedStats(map[string]interface{}{
		"l2_writer":    map[string]interface{}{"pending": 1},
		"retry_worker": map[string]interface{}{"retries": 2},
		"l1":           map[string]interface{}{"hit": 3},
		"l2":           map[string]interface{}{"miss": 4},
		"junk":         "ignored",
	})
	assert.Equal(t, 1, ext["l2_writer"].(map[string]interface{})["pending"])
	assert.Equal(t, 2, ext["retry_worker"].(map[string]interface{})["retries"])
	assert.Equal(t, 3, ext["l1"].(map[string]interface{})["hit"])
	assert.Equal(t, 4, ext["l2"].(map[string]interface{})["miss"])
	_, hasJunk := ext["junk"]
	assert.False(t, hasJunk)
}

// strPtr 同包已在 user_service_test.go 定义,直接复用。

// 编译期辅助:确保 gorm 被引用。
var _ = func(_ *gorm.DB) {}