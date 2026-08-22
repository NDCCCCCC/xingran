package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xingran-next/xingran-go-backend/internal/config"
	"github.com/xingran-next/xingran-go-backend/internal/core/security"
)

// =====================================================================
// 74-08 Batch C: pkg/middleware auth/cors/gzip/recovery/websocket/
// logger/selector_perms 中间件测试(httptest + 真实 JWTManager)。
// =====================================================================

func newMWContext(method, path string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, nil)
	return c, w
}

func newTestJWTManager(t *testing.T) *security.JWTManager {
	t.Helper()
	m, err := security.NewJWTManager(&config.JWTConfig{
		SecretKey:        "mw-test-secret-key-0123456789abcdef",
		AccessKeyExpire:  7200,
		RefreshKeyExpire: 604800,
		Issuer:           "mw-test",
		UseSM2:           false,
	})
	require.NoError(t, err)
	return m
}

// ---------------- auth.go ----------------

func TestExtractBearerToken(t *testing.T) {
	assert.Equal(t, "abc123", extractBearerToken("Bearer abc123"))
	assert.Equal(t, "", extractBearerToken("bearer abc"), "大小写敏感: 非 Bearer 前缀 → 空")
	assert.Equal(t, "", extractBearerToken("Basic xyz"))
	assert.Equal(t, "", extractBearerToken(""))
}

func TestExtractToken_HeaderAndQuery(t *testing.T) {
	c, _ := newMWContext("GET", "/x")
	c.Request.Header.Set("Authorization", "Bearer tok-header")
	assert.Equal(t, "tok-header", extractToken(c))

	// 无 header → query token(WebSocket 场景)
	c2, _ := newMWContext("GET", "/ws?token=tok-query")
	assert.Equal(t, "tok-query", extractToken(c2))

	// 都没有 → 空
	c3, _ := newMWContext("GET", "/ws")
	assert.Equal(t, "", extractToken(c3))
}

func TestJWTAuth_Middleware(t *testing.T) {
	jm := newTestJWTManager(t)

	// 缺 token → 401 abort
	c, w := newMWContext("GET", "/secure")
	JWTAuth(jm)(c)
	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 坏 token → abort
	c2, _ := newMWContext("GET", "/secure")
	c2.Request.Header.Set("Authorization", "Bearer not-a-jwt")
	JWTAuth(jm)(c2)
	assert.True(t, c2.IsAborted())

	// 好 token → 通过 + user context 写入
	pair, err := jm.GenerateTokenPair("u1", "alice", "Alice", []string{"admin"})
	require.NoError(t, err)
	c3, _ := newMWContext("GET", "/secure")
	c3.Request.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	JWTAuth(jm)(c3)
	assert.False(t, c3.IsAborted())
	uid, _ := c3.Get("user_id")
	assert.Equal(t, "u1", uid)
	uname, _ := c3.Get("username")
	assert.Equal(t, "alice", uname)
}

// mockBlacklist 实现 services.TokenBlacklistService。
type mockBlacklist struct {
	blacklisted bool
	err         error
}

func (m *mockBlacklist) AddToBlacklist(ctx context.Context, token string, expiry time.Time) error {
	return nil
}
func (m *mockBlacklist) IsBlacklisted(ctx context.Context, token string) (bool, error) {
	return m.blacklisted, m.err
}
func (m *mockBlacklist) RemoveFromBlacklist(ctx context.Context, token string) error {
	return nil
}

func TestJWTAuthWithBlacklist(t *testing.T) {
	jm := newTestJWTManager(t)
	pair, err := jm.GenerateTokenPair("u2", "bob", "Bob", nil)
	require.NoError(t, err)

	// 缺 token → 401
	c, w := newMWContext("GET", "/s")
	JWTAuthWithBlacklist(jm, &mockBlacklist{})(c)
	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 黑名单命中 → 401 "令牌已失效"
	c2, _ := newMWContext("GET", "/s")
	c2.Request.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	JWTAuthWithBlacklist(jm, &mockBlacklist{blacklisted: true})(c2)
	assert.True(t, c2.IsAborted())

	// 黑名单服务故障 → fail-open 放行(注释明确: 缓存不可用 ≠ 令牌被拉黑)
	c3, _ := newMWContext("GET", "/s")
	c3.Request.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	JWTAuthWithBlacklist(jm, &mockBlacklist{err: errors.New("redis down")})(c3)
	assert.False(t, c3.IsAborted(), "黑名单检查失败应 fail-open")

	// 正常 → 通过 + token/claims 写入 ctx
	c4, _ := newMWContext("GET", "/s")
	c4.Request.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	JWTAuthWithBlacklist(jm, &mockBlacklist{})(c4)
	assert.False(t, c4.IsAborted())
	tok, _ := c4.Get("token")
	assert.Equal(t, pair.AccessToken, tok)

	// 黑名单通过但 JWT 非法 → abort
	c5, _ := newMWContext("GET", "/s")
	c5.Request.Header.Set("Authorization", "Bearer garbage")
	JWTAuthWithBlacklist(jm, &mockBlacklist{})(c5)
	assert.True(t, c5.IsAborted())
}

// ---------------- cors.go ----------------

func TestContains(t *testing.T) {
	assert.True(t, contains([]string{"a", "b"}, "b"))
	assert.False(t, contains([]string{"a"}, "z"))
	assert.False(t, contains(nil, "x"))
}

func TestMatchDomainPattern(t *testing.T) {
	// 精确匹配
	assert.True(t, matchDomainPattern("example.com", "example.com"))
	assert.True(t, matchDomainPattern("https://example.com", "example.com"), "去协议")
	assert.True(t, matchDomainPattern("http://example.com:8080", "example.com"), "去端口")
	assert.True(t, matchDomainPattern("www.example.com", "example.com"), "www. 前缀反向匹配")
	assert.False(t, matchDomainPattern("other.com", "example.com"))

	// 通配符
	assert.True(t, matchDomainPattern("api.example.com", "*.example.com"))
	assert.True(t, matchDomainPattern("example.com", "*.example.com"), "通配符含裸域")
	assert.True(t, matchDomainPattern("https://a.b.example.com:8443", "*.example.com"))
	assert.False(t, matchDomainPattern("example.com.evil.com", "*.example.com"))

	// 不匹配模式前缀
	assert.False(t, matchDomainPattern("other.org", "*.com"))
}

func TestCors_AllowAll(t *testing.T) {
	// 空列表 → 允许所有(开发模式),OPTIONS 预检 204
	r := gin.New()
	r.Use(Cors(nil))
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/x", nil)
	req.Header.Set("Origin", "http://anywhere.example")
	req.Header.Set("Access-Control-Request-Method", "GET")
	r.ServeHTTP(w, req)
	assert.Equal(t, 204, w.Code)
	// gin-contrib/cors 在 AllowOriginFunc=true 时回显 Origin 而非 "*"
	assert.Equal(t, "http://anywhere.example", w.Header().Get("Access-Control-Allow-Origin"))

	// 带 "*" 也允许所有
	r2 := gin.New()
	r2.Use(Cors([]string{"*"}))
	r2.GET("/x", func(c *gin.Context) { c.String(200, "ok") })
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/x", nil)
	req2.Header.Set("Origin", "http://foo.bar")
	r2.ServeHTTP(w2, req2)
	assert.Equal(t, "http://foo.bar", w2.Header().Get("Access-Control-Allow-Origin"))
}

func TestCors_Whitelist(t *testing.T) {
	r := gin.New()
	r.Use(Cors([]string{"https://ok.example"}))
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

	// 白名单内
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Origin", "https://ok.example")
	r.ServeHTTP(w, req)
	assert.Equal(t, "https://ok.example", w.Header().Get("Access-Control-Allow-Origin"))

	// 白名单外 → 无 CORS 头
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/x", nil)
	req2.Header.Set("Origin", "https://evil.example")
	r.ServeHTTP(w2, req2)
	assert.Empty(t, w2.Header().Get("Access-Control-Allow-Origin"))
}

func TestCorsByPattern(t *testing.T) {
	r := gin.New()
	r.Use(CorsByPattern([]string{"*.example.com"}))
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Origin", "https://api.example.com")
	r.ServeHTTP(w, req)
	assert.Equal(t, "https://api.example.com", w.Header().Get("Access-Control-Allow-Origin"))

	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/x", nil)
	req2.Header.Set("Origin", "https://nope.other.com")
	r.ServeHTTP(w2, req2)
	assert.Empty(t, w2.Header().Get("Access-Control-Allow-Origin"))
}

// ---------------- gzip.go ----------------

func TestGzip_CompressesResponse(t *testing.T) {
	r := gin.New()
	r.Use(Gzip())
	r.GET("/x", func(c *gin.Context) { c.String(200, strings.Repeat("a", 2048)) })

	// 不接受 gzip → 原文
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/x", nil))
	assert.Equal(t, 200, w.Code)
	assert.Empty(t, w.Header().Get("Content-Encoding"))

	// 接受 gzip → 压缩
	w2 := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	r.ServeHTTP(w2, req)
	assert.Equal(t, "gzip", w2.Header().Get("Content-Encoding"))
}

// ---------------- recovery.go ----------------

func TestRecovery_PanicTo500(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Recovery())
	r.GET("/panic", func(c *gin.Context) { panic("boom") })
	r.GET("/ok", func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/panic", nil))
	assert.Equal(t, 500, w.Code)

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, httptest.NewRequest("GET", "/ok", nil))
	assert.Equal(t, 200, w2.Code)
	assert.Equal(t, "ok", w2.Body.String())
}

// ---------------- websocket.go ----------------

func TestWebSocketAuth(t *testing.T) {
	jm := newTestJWTManager(t)

	// 无 token → 401
	c, w := newMWContext("GET", "/ws")
	WebSocketAuth(nil)(c)
	assert.True(t, c.IsAborted())
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// 有 token 但 jwtManager 不是 *core.Core → 500
	c2, w2 := newMWContext("GET", "/ws?token=abc")
	WebSocketAuth("not-a-core")(c2)
	assert.True(t, c2.IsAborted())
	assert.Equal(t, http.StatusInternalServerError, w2.Code)

	// Bearer header 提取路径(token 非空后仍因类型断言失败 → 500)
	c3, w3 := newMWContext("GET", "/ws")
	c3.Request.Header.Set("Authorization", "bearer xyz")
	WebSocketAuth(nil)(c3)
	assert.True(t, c3.IsAborted())
	assert.Equal(t, http.StatusInternalServerError, w3.Code)

	// 错误 scheme 的 Authorization → token 仍为空 → 401
	c4, w4 := newMWContext("GET", "/ws")
	c4.Request.Header.Set("Authorization", "Basic xyz")
	WebSocketAuth(nil)(c4)
	assert.Equal(t, http.StatusUnauthorized, w4.Code)

	_ = jm // ValidateToken 路径需要完整 core.Core(重依赖),由集成环境覆盖
}

// ---------------- logger.go ----------------

func TestLoggerMiddleware_AndRequestID(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.Use(Logger())
	r.GET("/ok", func(c *gin.Context) { c.String(200, "ok") })
	r.POST("/err", func(c *gin.Context) { c.String(500, "bad") })

	// GET(无 body)+ 200 → info 路径
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/ok", nil))
	assert.Equal(t, 200, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"), "RequestID 中间件生成 ID")

	// 自带 X-Request-ID → 透传
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/ok", nil)
	req2.Header.Set("X-Request-ID", "fixed-id-1")
	r.ServeHTTP(w2, req2)
	assert.Equal(t, "fixed-id-1", w2.Header().Get("X-Request-ID"))

	// POST 带 body + 500 → error 路径(body 读取+恢复不破坏 handler)
	w3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("POST", "/err", strings.NewReader("payload"))
	r.ServeHTTP(w3, req3)
	assert.Equal(t, 500, w3.Code)
}

func TestFormatBody_Truncation(t *testing.T) {
	short := formatBody([]byte("abc"))
	assert.Equal(t, "abc", short)

	long := formatBody([]byte(strings.Repeat("x", maxBodyLogSize+100)))
	assert.Len(t, long, maxBodyLogSize+3, "截断到 1000 + '...'")
	assert.True(t, strings.HasSuffix(long, "..."))
}

// ---------------- selector_perms.go ----------------

func TestOpsSelectorReadPerms_Consistent(t *testing.T) {
	require.NotEmpty(t, OpsSelectorReadPerms)
	seen := map[string]bool{}
	for _, p := range OpsSelectorReadPerms {
		assert.True(t, strings.HasPrefix(p, "ops:"), "全部 ops: 前缀: %s", p)
		assert.True(t, strings.HasSuffix(p, ":list"), "全部 :list 后缀: %s", p)
		assert.False(t, seen[p], "无重复: %s", p)
		seen[p] = true
	}
}
