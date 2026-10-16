package system

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
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

// =====================================================================
// Phase 102-03 Task 3: 缓存键内联字面量扫描守护（D-102-5② / D-102-7）
//
// 12 文件窄扫：fmt.Sprintf 首参含 ":" 的字符串字面量硬失败。
// 匹配规则（三道判定）：
//   1. 剥离 % 格式动词后字面量必须含至少一个 ":"
//   2. 按 ":" 分段后每一段匹配 ^[a-z0-9_]*$（末段允许独立匹配 ^\*$）
//   3. 非空段数 >= 1
//
// 白名单初始零条目；runtime.Caller 相对推导仓库根（禁本地绝对路径）。
// =====================================================================

// allowedKeyResidues 白名单：文件名 → 允许的内联键字面量残留数。
// 初始全空 = 期望 0；未来确需豁免必须显式登记并附原因（diff 可见）。
var allowedKeyResidues = map[string]int{}

// keyLiteralFiles 窄扫 12 文件清单（D-102-7）
var keyLiteralFiles = []string{
	// captcha 2 文件
	"internal/core/captcha.go",
	"internal/core/captcha_background.go",
	// system 同包 3 文件
	"internal/services/system/notice_cache_impl.go",
	"internal/services/system/settings_cache_impl.go",
	"internal/services/system/widget_data_fetcher.go",
	// 子包 4 cache_impl
	"internal/services/duty/duty_cache_impl.go",
	"internal/services/workorder/workorder_cache_impl.go",
	"internal/services/knowledge/knowledge_cache_impl.go",
	"internal/services/network/cache_impl.go",
	// rpa
	"internal/services/rpa/selector_learner.go",
	// 根包 2 文件
	"internal/services/api_endpoint_service.go",
	"internal/services/mac_history_query_service.go",
}

// cacheKeyInlineResidue 解析单个 Go 文件，返回其中内联缓存键字面量的
// "文件名:行号" 列表。
//
// 匹配口径：fmt.Sprintf 的首参（CallExpr 第 0 实参）为字符串字面量，
// 剥离 % 格式动词（正则 %\w+）后字面量含 ":"，按 ":" 分段后各非空段
// 匹配 ^[a-z0-9_]*$（末段允许独立为 "*"），非空段数 >= 1。
func cacheKeyInlineResidue(t *testing.T, path string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("解析 %s 失败: %v", path, err)
	}
	var hits []string
	verbRe := regexp.MustCompile(`%\w+`)
	segRe := regexp.MustCompile(`^[a-z0-9_]*$`)
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		// 匹配 fmt.Sprintf
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Sprintf" {
			return true
		}
		if len(call.Args) < 1 {
			return true
		}
		// 首参必须为字符串字面量
		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		// 解析字符串值（去引号）
		val := lit.Value
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = val[1 : len(val)-1]
		}
		// 剥离 % 格式动词
		stripped := verbRe.ReplaceAllString(val, "")
		// 判定 1：必须含 ":"
		if !strings.Contains(stripped, ":") {
			return true
		}
		// 按 ":" 分段
		segments := strings.Split(stripped, ":")
		nonEmpty := 0
		valid := true
		for i, seg := range segments {
			if seg == "" {
				continue
			}
			// 末段允许独立为通配 pattern
			if i == len(segments)-1 && seg == "*" {
				nonEmpty++
				continue
			}
			if !segRe.MatchString(seg) {
				valid = false
				break
			}
			nonEmpty++
		}
		if !valid || nonEmpty < 1 {
			return true
		}
		// 命中
		pos := fset.Position(call.Pos())
		hits = append(hits, fmt.Sprintf("%s:%d", filepath.Base(pos.Filename), pos.Line))
		return true
	})
	return hits
}

// repoRoot 以本测试文件位置向上搜索 go.mod 定位仓库根（禁本地绝对路径断言）。
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller 失败——无法定位测试文件位置")
	}
	dir := filepath.Dir(thisFile)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("未找到 go.mod，仓库根定位失败")
		}
		dir = parent
	}
}

// TestCacheKeyInlineResidue 窄扫 12 文件，fmt.Sprintf 缓存键字面量零残留。
// 白名单初始零条目；超出即 Errorf 硬失败。
func TestCacheKeyInlineResidue(t *testing.T) {
	root := repoRoot(t)

	perFile := map[string][]string{}
	for _, rel := range keyLiteralFiles {
		path := filepath.Join(root, rel)
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				t.Logf("[SKIP] %s 不存在（可能已重构）", rel)
				continue
			}
			t.Fatalf("stat %s 失败: %v", path, err)
		}
		if info.IsDir() {
			t.Logf("[SKIP] %s 是目录而非文件", rel)
			continue
		}
		hits := cacheKeyInlineResidue(t, path)
		if len(hits) > 0 {
			perFile[filepath.Base(path)] = append(perFile[filepath.Base(path)], hits...)
		}
	}

	// 逐文件判定：超出白名单部分硬失败
	names := make([]string, 0, len(perFile))
	for name := range perFile {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		hits := perFile[name]
		allowed := allowedKeyResidues[name]
		if len(hits) > allowed {
			for _, h := range hits[allowed:] {
				t.Errorf("内联缓存键字面量残留：%s（D-102-7 硬档——12 文件零残留面不可倒退；确需豁免请显式登记 allowedKeyResidues）", h)
			}
		} else if len(hits) > 0 {
			for _, h := range hits {
				t.Logf("[ALLOWED] %s 处于白名单豁免额度内", h)
			}
		}
	}

	// 白名单卫生提示
	for name, allowed := range allowedKeyResidues {
		if _, ok := perFile[name]; !ok && allowed > 0 {
			t.Logf("[ALLOWED] 白名单条目 %s=%d 当前无对应残留（可清理）", name, allowed)
		}
	}
}
