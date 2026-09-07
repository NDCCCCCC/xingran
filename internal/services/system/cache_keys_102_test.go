package system

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCacheKeyEquivalence 等价快照测试：D-102-5① + D-102-1
// 期望列 = 原字面量快照，被测列 = 常量/helper 输出
// 键值漂移（大小写/分隔符/动词）→ 测试失败，守护 Redis 契约
func TestCacheKeyEquivalence(t *testing.T) {
	// ── 快照表：常量值 == 原字面量 ────────────────────────────────────────
	constantCases := []struct {
		name string // 常量名
		want string // 原字面量快照
		got string // 被测常量值
	}{
		// notice
		{"CacheKeyNoticeMyNotices", "notice:my_notices", CacheKeyNoticeMyNotices},
		{"CacheKeyNoticeUnreadCount", "notice:unread_count", CacheKeyNoticeUnreadCount},
		{"CacheKeyNoticeDetail", "notice:detail", CacheKeyNoticeDetail},
		// settings
		{"CacheKeySettingsUser", "settings:user", CacheKeySettingsUser},
		// duty
		{"CacheKeyDutyToday", "duty:today", CacheKeyDutyToday},
		{"CacheKeyDutyMonthly", "duty:monthly", CacheKeyDutyMonthly},
		{"CacheKeyDutyHolidays", "duty:holidays", CacheKeyDutyHolidays},
		// workorder
		{"CacheKeyWorkorderMyPending", "workorder:my_pending", CacheKeyWorkorderMyPending},
		{"CacheKeyWorkorderStatistics", "workorder:statistics", CacheKeyWorkorderStatistics},
		{"CacheKeyWorkorderDetail", "workorder:detail", CacheKeyWorkorderDetail},
		// knowledge
		{"CacheKeyKbArticle", "kb:article", CacheKeyKbArticle},
		{"CacheKeyKbCategoryTree", "kb:category:tree", CacheKeyKbCategoryTree},
		{"CacheKeyKbCategoryParent", "kb:category:parent", CacheKeyKbCategoryParent},
		{"CacheKeyKbTagsAll", "kb:tags:all", CacheKeyKbTagsAll},
		// network
		{"CacheKeyNetworkDeviceStatistics", "network_device:statistics", CacheKeyNetworkDeviceStatistics},
		{"CacheKeyNetworkDeviceDept", "network_device:dept", CacheKeyNetworkDeviceDept},
		{"CacheKeyNetworkDeviceCredential", "network_device:credential", CacheKeyNetworkDeviceCredential},
		{"CacheKeyNetworkDeviceDetail", "network_device:detail", CacheKeyNetworkDeviceDetail},
		// widget
		{"CacheKeyWidgetData", "widget:data", CacheKeyWidgetData},
		// rpa
		{"CacheKeyRpaSelectorBest", "rpa:selector:best", CacheKeyRpaSelectorBest},
	}

	for _, tc := range constantCases {
		t.Run("const/"+tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.got,
				"常量 %s 值漂移：期望 %q，实际 %q", tc.name, tc.want, tc.got)
		})
	}

	// ── helper 输出断言 ─────────────────────────────────────────────────
	helperCases := []struct {
		name string
		want string
		got  string
	}{
		// notice
		{"GetNoticeMyNoticesKey(u1,2,10)", "notice:my_notices:u1:page:2:size:10", GetNoticeMyNoticesKey("u1", 2, 10)},
		{"GetNoticeMyNoticesStatusKey(u1,2,10,0)", "notice:my_notices:u1:page:2:size:10:status:0", GetNoticeMyNoticesStatusKey("u1", 2, 10, "0")},
		{"GetNoticeUnreadCountKey(u1)", "notice:unread_count:u1", GetNoticeUnreadCountKey("u1")},
		{"GetNoticeDetailKey(n1)", "notice:detail:n1", GetNoticeDetailKey("n1")},
		{"GetNoticeAllPattern()", "notice:*", GetNoticeAllPattern()},
		// settings
		{"GetSettingsUserKey(u1)", "settings:user:u1", GetSettingsUserKey("u1")},
		// duty
		{"GetDutyMonthlyKey(2026,9)", "duty:monthly:2026:9", GetDutyMonthlyKey(2026, 9)},
		{"GetDutyHolidaysKey(2026)", "duty:holidays:2026", GetDutyHolidaysKey(2026)},
		{"GetDutyAllPattern()", "duty:*", GetDutyAllPattern()},
		{"GetDutyHolidaysPattern()", "duty:holidays:*", GetDutyHolidaysPattern()},
		// workorder
		{"GetWorkorderMyPendingKey(u1)", "workorder:my_pending:u1", GetWorkorderMyPendingKey("u1")},
		{"GetWorkorderMyPendingLimitKey(u1,20)", "workorder:my_pending:u1:limit:20", GetWorkorderMyPendingLimitKey("u1", 20)},
		{"GetWorkorderDetailKey(w1)", "workorder:detail:w1", GetWorkorderDetailKey("w1")},
		{"GetWorkorderAllPattern()", "workorder:*", GetWorkorderAllPattern()},
		// knowledge
		{"GetKbArticleKey(a1)", "kb:article:a1", GetKbArticleKey("a1")},
		{"GetKbCategoryParentKey(c1)", "kb:category:parent:c1", GetKbCategoryParentKey("c1")},
		{"GetKbCategoryPattern()", "kb:category:*", GetKbCategoryPattern()},
		{"GetKbArticlePattern()", "kb:article:*", GetKbArticlePattern()},
		// knowledge — GetKbCategoryStatusKey: 体内拼接，参数化输出
		{"GetKbCategoryStatusKey(kb:category:tree,1)", "kb:category:tree:status:1", GetKbCategoryStatusKey("kb:category:tree", 1)},
		// network
		{"GetNetworkDeviceDeptKey(d1)", "network_device:dept:d1", GetNetworkDeviceDeptKey("d1")},
		{"GetNetworkDeviceCredentialKey(c1)", "network_device:credential:c1", GetNetworkDeviceCredentialKey("c1")},
		{"GetNetworkDeviceDetailKey(dev1)", "network_device:detail:dev1", GetNetworkDeviceDetailKey("dev1")},
		{"GetNetworkDevicePattern()", "network_device:*", GetNetworkDevicePattern()},
		// widget
		{"GetWidgetDataKey(w1)", "widget:data:w1", GetWidgetDataKey("w1")},
		// widget %x verb preserved
		{"GetWidgetDataParamsKey(w1,1a2b)", "widget:data:w1:1a2b", GetWidgetDataParamsKey("w1", []byte{0x1a, 0x2b})},
		// rpa
		{"GetRpaSelectorBestKey(url,el)", "rpa:selector:best:url:el", GetRpaSelectorBestKey("url", "el")},
	}

	for _, tc := range helperCases {
		t.Run("helper/"+tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.got,
				"helper %s 输出漂移：期望 %q，实际 %q", tc.name, tc.want, tc.got)
		})
	}
}
