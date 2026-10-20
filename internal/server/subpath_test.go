package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// 子路径剥离与根路径重定向的回归测试(默认构建即可运行,不带 build tag)。
//
// 关键不变量:
//  1. 直连 :9000 时浏览器按 Vite base 发 /xingran/... ,后端必须把
//     /xingran/api/... 与 /xingran/uploads/... 剥掉前缀后交给真实路由。
//  2. /xingran/dashboard(SPA 路由)、/xingran/assets/... 不属于后端路由前缀,
//     不得被剥掉(否则会改变 NoRoute/ServeSPA 的既有行为面)。
//  3. /xingranfoo/... 这类相似前缀不得被误剥。
//  4. 根路径在没有 X-Forwarded-Prefix 时 302 到 /xingran/ ;带该头时直接返回
//     index.html(nginx 剥前缀部署场景,避免重定向死循环)。

func init() {
	gin.SetMode(gin.TestMode)
}

// newSubPathCtx 构造一个带指定请求的 gin 测试上下文。
// 命名区别于 embed_frontend_prod_test.go 的 newTestCtx —— embed 构建下两个
// 文件同包,撞名 = 重复符号编译失败。
func newSubPathCtx(method, target string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, target, nil)
	return c, w
}

// newSubPathRouter 构造一个带子路径剥离规则的最小 gin 引擎:
//   - /api/v1/probe  模拟 engine.Group("/api/v1")
//   - /uploads/*     模拟 engine.Static("/uploads", "./uploads")
//   - /dashboard     模拟 SPA 客户端路由(不应被剥前缀命中)
func newSubPathRouter() *gin.Engine {
	r := gin.New()
	r.Use(StripSubPathMiddleware())
	r.GET("/api/v1/probe", func(c *gin.Context) {
		c.String(http.StatusOK, c.Request.URL.Path)
	})
	r.GET("/uploads/*filepath", func(c *gin.Context) {
		c.String(http.StatusOK, c.Request.URL.Path)
	})
	r.GET("/dashboard", func(c *gin.Context) {
		c.String(http.StatusOK, c.Request.URL.Path)
	})
	return r
}

// TestSubPath_Handler_RewritesBackendPrefixes 锁定"带前缀的后端请求必须真的
// 落到后端路由"这一行为 —— gin 在中间件之前完成路由匹配,所以生效点必须
// 在进入 gin 之前(StripSubPathHandler)。
func TestSubPath_Handler_RewritesBackendPrefixes(t *testing.T) {
	h := StripSubPathHandler(newSubPathRouter())

	cases := []struct {
		in, want string
	}{
		{"/xingran/api/v1/probe", "/api/v1/probe"},
		{"/xingran/uploads/a.png", "/uploads/a.png"},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.in, nil))
		if w.Code != http.StatusOK {
			t.Fatalf("GET %s: status %d, want 200 (prefix must be stripped before routing)", tc.in, w.Code)
		}
		if got := w.Body.String(); got != tc.want {
			t.Errorf("GET %s: body %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestSubPath_Middleware_RewritesBackendPrefixes 锁定中间件对 handler 可见
// 路径的改写:后端前缀被剥掉,RequestURI 保持客户端原始值(供访问日志排障)。
func TestSubPath_Middleware_RewritesBackendPrefixes(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"/xingran/api/v1/probe", "/api/v1/probe"},
		{"/xingran/uploads/a.png", "/uploads/a.png"},
	}
	for _, tc := range cases {
		c, _ := newSubPathCtx(http.MethodGet, tc.in)
		StripSubPathMiddleware()(c)
		if got := c.Request.URL.Path; got != tc.want {
			t.Errorf("path %q rewritten to %q, want %q", tc.in, got, tc.want)
		}
		if got := c.Request.RequestURI; got != tc.in {
			t.Errorf("RequestURI = %q, want original %q", got, tc.in)
		}
	}
}

// TestSubPath_Middleware_LeavesOtherPathsUnchanged 逐条锁死"不得重写"的路径:
// 无前缀请求、SPA 客户端路由、相似前缀 /xingranfoo、根路径。
func TestSubPath_Middleware_LeavesOtherPathsUnchanged(t *testing.T) {
	unchanged := []string{
		"/api/v1/probe",            // 无前缀
		"/uploads/a.png",           // 无前缀
		"/xingran/dashboard",       // SPA 客户端路由
		"/xingran/",                // 子路径根
		"/xingran",                 // 子路径根(无尾斜杠)
		"/xingranfoo/api/v1/probe", // 相似前缀,不得误剥
		"/",                        // 根路径
	}
	for _, p := range unchanged {
		c, _ := newSubPathCtx(http.MethodGet, p)
		StripSubPathMiddleware()(c)
		if got := c.Request.URL.Path; got != p {
			t.Errorf("path %q rewritten to %q, want unchanged", p, got)
		}
	}

	// /xingran/dashboard 不得命中 /dashboard 路由(否则会改变 SPA 行为面)
	h := StripSubPathHandler(newSubPathRouter())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/xingran/dashboard", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("GET /xingran/dashboard: status %d, want 404 (SPA route must not be rewritten)", w.Code)
	}
}

// TestSubPath_RootRedirectPath 锁定根路径重定向的唯一豁免信号。
func TestSubPath_RootRedirectPath(t *testing.T) {
	c, _ := newSubPathCtx(http.MethodGet, "/")
	if got := rootRedirectPath(c); got != "/xingran/" {
		t.Errorf("rootRedirectPath without X-Forwarded-Prefix = %q, want /xingran/", got)
	}

	c2, _ := newSubPathCtx(http.MethodGet, "/")
	c2.Request.Header.Set("X-Forwarded-Prefix", "/xingran")
	if got := rootRedirectPath(c2); got != "" {
		t.Errorf("rootRedirectPath with X-Forwarded-Prefix = %q, want empty (no redirect)", got)
	}
}

// TestSubPath_StripSubPathTable 让默认构建也覆盖共享的 stripSubPath
// (该函数的既有表驱动用例在 embed_frontend_prod_test.go 里,只跑 embed 构建)。
func TestSubPath_StripSubPathTable(t *testing.T) {
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
