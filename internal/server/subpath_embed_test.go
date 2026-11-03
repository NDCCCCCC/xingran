//go:build embed
// +build embed

package server

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// 根路径重定向语义的回归测试(仅 embed 构建)。
//
// 关键不变量:
//  1. 直连 :9000 时 GET / → 302 Location: /xingran/ ,浏览器落到与前端
//     Router basename 匹配的地址上,不再白屏。
//  2. nginx 剥前缀部署时请求带 X-Forwarded-Prefix,此时不得重定向 ——
//     否则浏览器会在 "/" 与 "/xingran/" 之间无限跳转。
//  3. 跟随重定向后的第二次请求(/xingran/)不得再产生任何重定向。

// newSubPathEmbedCtx 构造 embed 变体专用的测试上下文。
// 命名区别于 newTestCtx / newSubPathCtx —— 同包撞名 = 重复符号编译失败。
func newSubPathEmbedCtx(method, target string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, target, nil)
	return c, w
}

// embedDistStaged 报告嵌入的 dist 是否已 staged(构建机未前置 dist 时为 false)。
func embedDistStaged() bool {
	_, err := fs.ReadFile(frontendFS, "xingran-react-frontend/dist/index.html")
	return err == nil
}

// TestDetectedFrontendBase_MatchesEmbeddedIndex 锁定 base 探测与嵌入产物的对应关系。
//
// 背景:探测 marker 是 `"/xingran/assets/`,取 html[idx+1:] 只去掉引号、前导斜杠仍在,
// 比较串必须同样带斜杠。此前比较串漏了斜杠 → detectedFrontendBase() 恒为 "" →
// 直连 :9000 时不做 302 引导 → 浏览器停在 "/" → 与前端 basename 不匹配 → 白屏。
// 而旧测试在 detected 为空时只断言"不是 302",把这个缺陷一并放过了。
//
// 不变量:嵌入的 index.html 带子路径资源前缀时,detectedFrontendBase() 必须等于该前缀;
// 根路径构建时必须为空。
func TestDetectedFrontendBase_MatchesEmbeddedIndex(t *testing.T) {
	if !embedDistStaged() {
		t.Skip("embed dist not staged")
	}
	content, err := fs.ReadFile(frontendFS, "xingran-react-frontend/dist/index.html")
	if err != nil {
		t.Fatalf("read embedded index.html: %v", err)
	}
	html := string(content)
	prefix := strings.TrimPrefix(frontendSubPath, "/")

	if !strings.Contains(html, `"/`+prefix+`/assets/`) {
		// 根路径构建的 dist → 无 basename 概念,探测结果应为空
		if got := detectedFrontendBase(); got != "" {
			t.Fatalf("detectedFrontendBase() = %q, want %q (根路径构建的 dist)", got, "")
		}
		return
	}
	if got := detectedFrontendBase(); got != frontendSubPath {
		t.Fatalf("detectedFrontendBase() = %q, want %q (index.html 带子路径资源前缀)",
			got, frontendSubPath)
	}
}

func TestSubPath_ServeRoot_RedirectsToSubPath(t *testing.T) {
	c, w := newSubPathEmbedCtx(http.MethodGet, "/")
	ServeRoot(c)
	if detectedFrontendBase() == "" {
		// 根路径构建的 dist 没有 basename,必须直接回 index.html,不得重定向
		if w.Code == http.StatusFound {
			t.Fatalf("GET / (base=\"\"): status 302, want 200 (no basename to redirect into)")
		}
		if !embedDistStaged() {
			t.Skip("embed dist not staged")
		}
		if w.Code != http.StatusOK {
			t.Fatalf("GET / (base=\"\"): status %d, want 200", w.Code)
		}
		return
	}
	if w.Code != http.StatusFound {
		t.Fatalf("GET /: status %d, want 302", w.Code)
	}
	if got := w.Header().Get("Location"); got != "/xingran/" {
		t.Errorf("GET /: Location %q, want /xingran/", got)
	}
}

func TestSubPath_ServeRoot_WithForwardedPrefixServesIndex(t *testing.T) {
	c, w := newSubPathEmbedCtx(http.MethodGet, "/")
	c.Request.Header.Set("X-Forwarded-Prefix", "/xingran")
	ServeRoot(c)
	if w.Code == http.StatusFound {
		t.Fatalf("GET / with X-Forwarded-Prefix: status 302, want no redirect (redirect loop)")
	}
	if !embedDistStaged() {
		t.Skip("embed dist not staged")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("GET / with X-Forwarded-Prefix: status %d, want 200", w.Code)
	}
	if got := w.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Errorf("content-type %q, want text/html", got)
	}
}

func TestSubPath_NoRedirectLoop(t *testing.T) {
	// 模拟浏览器跟随 302 后的第二次请求
	c, w := newSubPathEmbedCtx(http.MethodGet, "/xingran/")
	ServeSPA(c)
	if w.Code == http.StatusFound {
		t.Fatalf("GET /xingran/: status 302, want no redirect (redirect loop)")
	}
	if !embedDistStaged() {
		t.Skip("embed dist not staged")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("GET /xingran/: status %d, want 200", w.Code)
	}
	if got := w.Header().Get("Content-Type"); !strings.Contains(got, "text/html") {
		t.Errorf("content-type %q, want text/html", got)
	}
}

// TestSubPath_StaleUnprefixedSPARouteRedirects 锁定陈旧无前缀 SPA URL 的兜底:
// window.location.href = "/login" 这类根绝对路径跳转让浏览器停在 basename 之外
// (控制台报 "<Router basename=/xingran/> is not able to match"),此时 302 补
// 前缀让地址落回 basename 之下,而不是回一个必然白屏的 index.html。
func TestSubPath_StaleUnprefixedSPARouteRedirects(t *testing.T) {
	c, w := newSubPathEmbedCtx(http.MethodGet, "/login")
	ServeSPA(c)
	if detectedFrontendBase() == "" {
		// 根路径构建的 dist:无 basename,无前缀路径本就是合法地址,必须回 index.html
		if w.Code == http.StatusFound {
			t.Fatalf("GET /login (base=\"\"): status 302, want 200 (no basename to redirect into)")
		}
		if !embedDistStaged() {
			t.Skip("embed dist not staged")
		}
		if w.Code != http.StatusOK {
			t.Fatalf("GET /login (base=\"\"): status %d, want 200", w.Code)
		}
		return
	}
	if w.Code != http.StatusFound {
		t.Fatalf("GET /login: status %d, want 302", w.Code)
	}
	if got := w.Header().Get("Location"); got != "/xingran/login" {
		t.Errorf("GET /login: Location %q, want /xingran/login", got)
	}
}

// TestSubPath_StaleUnprefixedWithForwardedPrefixServesIndex nginx 剥前缀部署下
// 请求带 X-Forwarded-Prefix,无前缀 SPA 路径必须照旧返回 index.html,不得重定向
// (否则 /login → /xingran/login → nginx 剥回 /login → 死循环)。
func TestSubPath_StaleUnprefixedWithForwardedPrefixServesIndex(t *testing.T) {
	c, w := newSubPathEmbedCtx(http.MethodGet, "/login")
	c.Request.Header.Set("X-Forwarded-Prefix", "/xingran")
	ServeSPA(c)
	if w.Code == http.StatusFound {
		t.Fatalf("GET /login with X-Forwarded-Prefix: status 302, want no redirect")
	}
	if !embedDistStaged() {
		t.Skip("embed dist not staged")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("GET /login with X-Forwarded-Prefix: status %d, want 200", w.Code)
	}
}

// TestSubPath_StaleUnprefixedExcludedPathsStay404 后端自有前缀与带扩展名的
// 陈旧资源请求不得重定向 —— 404 语义保持,重定向只会掩盖真实问题。
func TestSubPath_StaleUnprefixedExcludedPathsStay404(t *testing.T) {
	cases := []string{
		"/api/v1/nonexistent",   // API 前缀
		"/uploads/missing.png",  // 上传前缀
		"/swagger/old-path",     // swagger 前缀
		"/debug/pprof/x",        // debug 前缀
		"/assets/stale-hash.js", // 带扩展名的陈旧资源
		"/favicon.ico",          // 带扩展名
	}
	for _, p := range cases {
		c, w := newSubPathEmbedCtx(http.MethodGet, p)
		ServeSPA(c)
		if w.Code == http.StatusFound {
			t.Errorf("GET %s: status 302 → %q, want 404 (must not redirect)", p, w.Header().Get("Location"))
		}
	}
}
