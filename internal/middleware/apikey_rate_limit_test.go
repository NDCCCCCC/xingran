package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/xingran-next/xingran-go-backend/internal/services"
)

// scopeProbe 记录 handler 内 SelectScope 纯函数推导结果,
// 使中间件单元测试可直接断言 scope 选择语义(无需额外的 context 中转键)
type scopeProbe struct {
	called  bool
	scope   string
	allowed bool
}

// newRateLimitTestRouter 构造 gin.Engine:
// setter 中间件写入测试 context 键 → RateLimitByScope(rl, action) → handler
// handler 内直接调用 SelectScope 验证推导结果(D-20 测试可观测性)
func newRateLimitTestRouter(action string, setters map[string]interface{}, probe *scopeProbe) *gin.Engine {
	gin.SetMode(gin.TestMode)
	rl := services.NewRateLimiter(nil) // nil → staticRateLimitProvider 兜底(默认值与既有硬编码一致)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		for k, v := range setters {
			c.Set(k, v)
		}
		c.Next()
	})
	router.Use(RateLimitByScope(rl, action))
	router.GET("/ping", func(c *gin.Context) {
		scopesRaw, _ := c.Get("scopes")
		scopes, _ := scopesRaw.([]string)
		inheritRaw, inheritExists := c.Get("inherit_perms")
		inheritPerms, _ := inheritRaw.(bool)
		probe.scope, probe.allowed = SelectScope(scopes, inheritPerms && inheritExists, action)
		probe.called = true
		c.JSON(200, gin.H{"ok": true})
	})
	return router
}

func servePing(router *gin.Engine) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// TestRateLimitByScope_ActionListSelectsRead D-11/D-12: action=list → required=read,
// scopes=["read"] → 选 read 档限流
func TestRateLimitByScope_ActionListSelectsRead(t *testing.T) {
	probe := &scopeProbe{}
	router := newRateLimitTestRouter("list", map[string]interface{}{
		"scopes":    []string{"read"},
		"auth_type": "api_key",
	}, probe)

	w := servePing(router)
	assert.Equal(t, 200, w.Code)
	assert.True(t, probe.called)
	assert.Equal(t, "read", probe.scope)
	assert.True(t, probe.allowed)
}

// TestRateLimitByScope_ActionEditSelectsWrite D-12: action=edit → required=write,
// scopes=["write"] → 选 write 档限流
func TestRateLimitByScope_ActionEditSelectsWrite(t *testing.T) {
	probe := &scopeProbe{}
	router := newRateLimitTestRouter("edit", map[string]interface{}{
		"scopes":    []string{"write"},
		"auth_type": "api_key",
	}, probe)

	w := servePing(router)
	assert.Equal(t, 200, w.Code)
	assert.True(t, probe.called)
	assert.Equal(t, "write", probe.scope)
	assert.True(t, probe.allowed)
}

// TestRateLimitByScope_ActionMissingScope_403 D-12 fail-closed: action=edit 需要 write,
// scopes=["read"] 不含 → 403 + code != 0,不进入限流检查(handler 不执行)
func TestRateLimitByScope_ActionMissingScope_403(t *testing.T) {
	probe := &scopeProbe{}
	router := newRateLimitTestRouter("edit", map[string]interface{}{
		"scopes":    []string{"read"},
		"auth_type": "api_key",
	}, probe)

	w := servePing(router)
	assert.Equal(t, 403, w.Code, "scopes 不含 required scope 应 403 (fail-closed)")
	assert.False(t, probe.called, "403 拒绝路径不应进入下游 handler")
	assert.Contains(t, w.Body.String(), `"code":403`, "响应体 code 应非 0")
}

// TestRateLimitByScope_AdminOverriddes D-12: scopes=["read","admin"],action=edit
// 需要 write,read 不含,但 admin 最高限额覆盖
func TestRateLimitByScope_AdminOverriddes(t *testing.T) {
	probe := &scopeProbe{}
	router := newRateLimitTestRouter("edit", map[string]interface{}{
		"scopes":    []string{"read", "admin"},
		"auth_type": "api_key",
	}, probe)

	w := servePing(router)
	assert.Equal(t, 200, w.Code)
	assert.True(t, probe.called)
	assert.Equal(t, "admin", probe.scope)
	assert.True(t, probe.allowed)
}

// TestRateLimitByScope_InheritPermsDefaultLimit D-13: InheritPerms=true 时 scopes
// 仅含细粒度 permission code(system:user:list),短路走 default 限额
func TestRateLimitByScope_InheritPermsDefaultLimit(t *testing.T) {
	probe := &scopeProbe{}
	router := newRateLimitTestRouter("list", map[string]interface{}{
		"scopes":        []string{"system:user:list"},
		"auth_type":     "api_key",
		"inherit_perms": true,
	}, probe)

	w := servePing(router)
	assert.Equal(t, 200, w.Code)
	assert.True(t, probe.called)
	assert.Equal(t, "default", probe.scope)
	assert.True(t, probe.allowed)
}

// TestRateLimitByScope_MultiScope_SelectsMatchingNotFirst D-12 核心: 多 scope
// ["admin","write","read"] + action=list → 选 read(action-aware),
// 而非既有代码错误地任意取 scopes[0]=admin
func TestRateLimitByScope_MultiScope_SelectsMatchingNotFirst(t *testing.T) {
	probe := &scopeProbe{}
	router := newRateLimitTestRouter("list", map[string]interface{}{
		"scopes":    []string{"admin", "write", "read"},
		"auth_type": "api_key",
	}, probe)

	w := servePing(router)
	assert.Equal(t, 200, w.Code)
	assert.True(t, probe.called)
	assert.Equal(t, "read", probe.scope)
	assert.True(t, probe.allowed)
}

// TestRateLimitByScope_ActionRemoveRequiresWrite WR-04/BL-01 回归锚:
// action=remove(写 action)需要 write,scopes=["read"] → 403 fail-closed。
// BL-01 修复前 remove 经 getRequiredScope 兜底 read,只读 key 的写请求会被
// 按 read 档限流放行;修复后按 write 档校验,read scope 直接 403。
func TestRateLimitByScope_ActionRemoveRequiresWrite(t *testing.T) {
	probe := &scopeProbe{}
	router := newRateLimitTestRouter("remove", map[string]interface{}{
		"scopes":    []string{"read"},
		"auth_type": "api_key",
	}, probe)

	w := servePing(router)
	assert.Equal(t, 403, w.Code, "remove 是写 action, scopes=[read] 应 403 (BL-01 回归锚)")
	assert.False(t, probe.called, "403 拒绝路径不应进入下游 handler")
}

// TestRateLimitByScope_NotAPIKeyAuth_Skip 既有行为保留: 非 API Key 认证(jwt)
// 跳过限流,handler 继续执行,不写 X-RateLimit-* 响应头
func TestRateLimitByScope_NotAPIKeyAuth_Skip(t *testing.T) {
	probe := &scopeProbe{}
	router := newRateLimitTestRouter("list", map[string]interface{}{
		"auth_type": "jwt",
	}, probe)

	w := servePing(router)
	assert.Equal(t, 200, w.Code, "非 API Key 认证应跳过限流直达 handler")
	assert.True(t, probe.called)
	assert.Empty(t, w.Header().Get("X-RateLimit-Limit"), "跳过限流不应写 X-RateLimit-Limit 头")
}
