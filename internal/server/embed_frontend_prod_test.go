//go:build embed
// +build embed

package server

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// 前缀剥离与静态资源服务的回归测试。
//
// 关键不变量:
//  1. nginx 子路径部署下,浏览器以 /xingran/assets/xxx 访问;nginx 剥前缀
//     后端看到 /assets/xxx,正常服务。
//  2. 本地直连 :9000 调试时浏览器绕过 nginx,直接发 /xingran/assets/xxx 到后端;
//     后端必须也接受这个形式(剥前缀后等价于根路径访问)。
//  3. API/upload 等路径在 NoRoute fallback 中必须被剥离前缀后再判断,
//     否则 /xingran/api/x 会被误判为 SPA 路由返回 index.html。
//
// 文件名带 hash,测试不硬编码:从 staged dist/assets/ 取首个 .js 文件名。

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestCtx(method, target string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, target, nil)
	return c, w
}

// firstAsset 选取 dist/assets 下的第一个 .js 文件,返回相对路径
// ("assets/<name>");dist 未 staged 时返回空串。
func firstAsset() string {
	entries, err := fs.ReadDir(frontendFS, "xingran-react-frontend/dist/assets")
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) == ".js" {
			return "assets/" + e.Name()
		}
	}
	return ""
}

func TestServeFrontend_RootReturnsIndex(t *testing.T) {
	c, w := newTestCtx(http.MethodGet, "/")
	ServeFrontend(c)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /: status %d, want 200", w.Code)
	}
	if got := w.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("GET /: content-type %q, want text/html", got)
	}
}

func TestServeFrontend_SubPathRootReturnsIndex(t *testing.T) {
	for _, p := range []string{"/xingran", "/xingran/"} {
		c, w := newTestCtx(http.MethodGet, p)
		ServeFrontend(c)
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s: status %d, want 200 (prefix strip should map to /)", p, w.Code)
		}
		if got := w.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
			t.Errorf("GET %s: content-type %q, want text/html", p, got)
		}
	}
}

func TestServeFrontend_AssetRootAndSubPath(t *testing.T) {
	rel := firstAsset()
	if rel == "" {
		t.Skip("no dist/assets/*.js in staged embed — skip asset serving tests")
	}

	// 根路径
	c, w := newTestCtx(http.MethodGet, "/"+rel)
	ServeFrontend(c)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /%s: status %d, want 200", rel, w.Code)
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "javascript") {
		t.Errorf("GET /%s: content-type %q, want javascript", rel, w.Header().Get("Content-Type"))
	}

	// /xingran 前缀(直连 :9000 调试场景)— 必须剥前缀后命中同一资源
	c, w = newTestCtx(http.MethodGet, "/xingran/"+rel)
	ServeFrontend(c)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /xingran/%s: status %d, want 200 (prefix strip broken?)", rel, w.Code)
	}
}

func TestServeFrontend_MissingAssetReturns404(t *testing.T) {
	c, w := newTestCtx(http.MethodGet, "/assets/does-not-exist-abc123.js")
	ServeFrontend(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404", w.Code)
	}
}

func TestServeSPA_AssetSubPathServesFromEmbed(t *testing.T) {
	rel := firstAsset()
	if rel == "" {
		t.Skip("no dist/assets/*.js in staged embed")
	}
	// NoRoute 落到 ServeSPA;路径带 /xingran/ 时必须剥前缀后命中嵌入资源。
	c, w := newTestCtx(http.MethodGet, "/xingran/"+rel)
	ServeSPA(c)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /xingran/%s: status %d, want 200", rel, w.Code)
	}
}

func TestServeSPA_SPARouteReturnsIndex(t *testing.T) {
	// /xingran/dashboard → 剥前缀 → /dashboard → 无扩展名 → index.html
	c, w := newTestCtx(http.MethodGet, "/xingran/dashboard")
	ServeSPA(c)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", w.Code)
	}
	if got := w.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("content-type %q, want text/html", got)
	}
}

func TestServeSPA_APIPathAfterStripReturnsNotFound(t *testing.T) {
	// /xingran/api/x 剥前缀后是 /api/x — 必须在 API 排除列表里返回 404,
	// 否则会被错误地当作 SPA 路由返回 index.html。
	c, w := newTestCtx(http.MethodGet, "/xingran/api/v1/foo")
	ServeSPA(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404 (API path must not fall through to index.html)", w.Code)
	}
}

func TestServeSPA_NonGETReturnsNotFound(t *testing.T) {
	c, w := newTestCtx(http.MethodPost, "/xingran/dashboard")
	ServeSPA(c)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404 (POST to SPA route)", w.Code)
	}
}

func TestStripSubPath_EdgeCases(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"/", "/"},
		{"/xingran", "/"},
		{"/xingran/", "/"},
		{"/xingran/assets/index.js", "/assets/index.js"},
		{"/assets/index.js", "/assets/index.js"}, // 无前缀时不动
		{"/other", "/other"},                     // 不匹配时不动
	}
	for _, tc := range cases {
		if got := stripSubPath(tc.in); got != tc.want {
			t.Errorf("stripSubPath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}