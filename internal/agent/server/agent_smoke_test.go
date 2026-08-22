package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =====================================================================
// account_manager.go: AccountManager / Account 结构 / 平台策略选择
// =====================================================================

func TestNewAccountManager_PlatformStrategy(t *testing.T) {
	m := NewAccountManager()
	require.NotNil(t, m)
	require.NotNil(t, m.strategy)

	// 平台选择
	switch runtime.GOOS {
	case "windows":
		assert.IsType(t, &windowsPlatformStrategy{}, m.strategy)
	case "linux":
		assert.IsType(t, &linuxPlatformStrategy{}, m.strategy)
	default:
		assert.IsType(t, &linuxPlatformStrategy{}, m.strategy)
	}
}

func TestAccount_Struct(t *testing.T) {
	a := Account{
		Username:  "alice",
		Password:  "x",
		IsAdmin:   true,
		IsEnabled: true,
	}
	assert.Equal(t, "alice", a.Username)
	assert.True(t, a.IsAdmin)
}

// =====================================================================
// logger.go: InitLogger / WithContext / WithRequestID / WithFields / Debug...
// =====================================================================

func TestInitLogger(t *testing.T) {
	// 空 path → 默认
	require.NoError(t, InitLogger("info", ""))
	// QUIRK: InitLogger 内部 level parse 失败降级 info 但**不返回 err**
	// (ParseLevel 后丢弃 err,直接 return nil)。invalid level → nil err。
	err := InitLogger("bogus", "")
	require.NoError(t, err, "QUIRK: InitLogger 应返回 nil even with bogus level")
}

func TestLoggerShortcuts(t *testing.T) {
	require.NoError(t, InitLogger("info", ""))
	Debug("debug")
	Info("info")
	Warn("warn")
	Error("error")
	// 不 panic 即可
}

func TestWithRequestIDAndFields(t *testing.T) {
	require.NoError(t, InitLogger("info", ""))
	e := WithRequestID("rid-1")
	assert.NotNil(t, e)
	f := WithFields(map[string]interface{}{"k": "v"})
	assert.NotNil(t, f)
}

func TestWithContext_NilCtx(t *testing.T) {
	require.NoError(t, InitLogger("info", ""))
	e := WithContext(context.TODO())
	assert.NotNil(t, e)
}

// =====================================================================
// middleware.go: CORS / SecurityHeaders / Logging / Recovery / JWTAuth
// =====================================================================

func init() { gin.SetMode(gin.TestMode) }

func TestCORSMiddleware_OptionsAndGet(t *testing.T) {
	r := gin.New()
	r.Use(CORSMiddleware())
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))

	// OPTIONS → 204
	w2 := httptest.NewRecorder()
	r.OPTIONS("/x", func(c *gin.Context) {})
	req2 := httptest.NewRequest(http.MethodOptions, "/x", nil)
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusNoContent, w2.Code)
}

func TestSecurityHeaders(t *testing.T) {
	r := gin.New()
	r.Use(SecurityHeaders())
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Contains(t, w.Header().Get("Content-Security-Policy"), "self")
}

func TestLoggingAndRecovery(t *testing.T) {
	require.NotNil(t, LoggingMiddleware())
	require.NotNil(t, RecoveryMiddleware())
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	require.NoError(t, InitLogger("info", ""))
	auth := NewJWTAuthenticator("secret", "http://x", "a1", "v1", nil)
	r := gin.New()
	r.Use(JWTAuth(auth))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_InvalidFormat(t *testing.T) {
	require.NoError(t, InitLogger("info", ""))
	auth := NewJWTAuthenticator("secret", "http://x", "a1", "v1", nil)
	r := gin.New()
	r.Use(JWTAuth(auth))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Basic xyz")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// =====================================================================
// pty_manager.go / connection_manager.go: 仅构造覆盖
// =====================================================================

func TestNewPTYManager(t *testing.T) {
	m := NewPTYManager()
	require.NotNil(t, m)
}

func TestNewConnectionManager(t *testing.T) {
	auth := NewJWTAuthenticator("secret", "http://x", "a1", "v1", nil)
	m := NewConnectionManager(auth)
	require.NotNil(t, m)
}

// =====================================================================
// jwt_auth.go: NewJWTAuthenticator / ParseTokenClaims / NewTLSConfigFromConfig
// =====================================================================

func TestNewJWTAuthenticator(t *testing.T) {
	a := NewJWTAuthenticator("secret", "http://x", "a1", "v1", nil)
	require.NotNil(t, a)
}

func TestParseTokenClaims_Invalid(t *testing.T) {
	_, err := ParseTokenClaims("not-a-jwt")
	require.Error(t, err)
}

func TestNewTLSConfigFromConfig_Errors(t *testing.T) {
	// QUIRK: 全部空字符串参数不报错,仅 verifyCertificates=true 时构造 empty TLS config
	cfg, err := NewTLSConfigFromConfig("", "", "", true)
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// 提供无效 CA 文件 → err
	_, err = NewTLSConfigFromConfig("", "", "/nonexistent/ca.pem", true)
	require.Error(t, err)
}

// =====================================================================
// config.go: LoadConfig / SystemFingerprint
// =====================================================================

func TestLoadConfig_NotExist(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path.yaml")
	require.Error(t, err)
}

func TestCollectSystemFingerprint_Smoke(t *testing.T) {
	fp, err := CollectSystemFingerprint()
	if err != nil {
		t.Skipf("CollectSystemFingerprint 失败(平台限制): %v", err)
	}
	assert.NotNil(t, fp)
	assert.NotEmpty(t, fp.Hostname)
}
