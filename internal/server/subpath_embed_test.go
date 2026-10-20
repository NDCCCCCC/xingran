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

func TestSubPath_ServeRoot_RedirectsToSubPath(t *testing.T) {
	c, w := newSubPathEmbedCtx(http.MethodGet, "/")
	ServeRoot(c)
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
