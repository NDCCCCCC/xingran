package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// newRequireAPIKeyResourcePermissionRouter 构造一个独立的 gin.Engine,
// 仅挂载 RequireAPIKeyResourcePermission(resource, action) 中间件,
// 用于精准测试该 helper 的命中 / 未命中 / scope 检查行为。
//
// 不挂载 MultiAuth — 本测试只验证 RequireAPIKeyResourcePermission 单
// 中间件逻辑;MultiAuth 的 InheritPerms 集成测试在 apikey_inherit_integration_test.go。
func newRequireAPIKeyResourcePermissionRouter(t *testing.T, resource, action string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(RequireAPIKeyResourcePermission(resource, action))
	r.GET("/probe", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})
	return r
}

// runProbeWithScopes 构造带 scopes 的请求并执行 — 用 helper middleware 注入 c.Set("scopes")。
func runProbeWithScopes(t *testing.T, resource, action string, scopes []string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		if scopes != nil {
			c.Set("scopes", scopes)
		}
		c.Next()
	})
	r.Use(RequireAPIKeyResourcePermission(resource, action))
	r.GET("/probe", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/probe", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestRequireAPIKeyResourcePermission_Hit 命中 → admin 通配 → 200。
//
// 验证: (system:user, list) → UserList → admin 含所有权限 → c.Next() 200。
func TestRequireAPIKeyResourcePermission_Hit(t *testing.T) {
	w := runProbeWithScopes(t, "system:user", "list", []string{"admin"})
	assert.Equal(t, 200, w.Code, "admin 作用域应通过 (admin 含所有权限)")
}

// TestRequireAPIKeyResourcePermission_HitReadScope view → read, scopes 含 read → 200。
func TestRequireAPIKeyResourcePermission_HitReadScope(t *testing.T) {
	w := runProbeWithScopes(t, "system:user", "view", []string{"read"})
	assert.Equal(t, 200, w.Code, "view → read 命中 scopes=[read] 应通过")
}

// TestRequireAPIKeyResourcePermission_HitExactPermCode 直接持有 PermissionCode
// (D-06 InheritPerms=true 路径下 scopes 含细粒度 system:user:list) → 200。
func TestRequireAPIKeyResourcePermission_HitExactPermCode(t *testing.T) {
	w := runProbeWithScopes(t, "system:user", "list", []string{"system:user:list"})
	assert.Equal(t, 200, w.Code, "直接持有细粒度 PermissionCode 应通过")
}

// TestRequireAPIKeyResourcePermission_MissingScope read scope 但 action=edit
// (需要 write) → 403 fail-closed。
func TestRequireAPIKeyResourcePermission_MissingScope(t *testing.T) {
	w := runProbeWithScopes(t, "system:user", "edit", []string{"read"})
	assert.Equal(t, 403, w.Code, "edit 需要 write, scopes=[read] 应 403")
}

// TestRequireAPIKeyResourcePermission_UnmappedResource monitor:* 不在 map (D-02
// 范围限定),即使 admin 也必须 403 (D-03 fail-closed)。
func TestRequireAPIKeyResourcePermission_UnmappedResource(t *testing.T) {
	w := runProbeWithScopes(t, "monitor:operlog", "list", []string{"admin"})
	assert.Equal(t, 403, w.Code, "monitor:operlog 未注册,即使 admin 也应 403 (D-03 fail-closed)")
	assert.Contains(t, w.Body.String(), "资源权限未定义", "未命中错误信息应明确指出资源权限未定义")
}

// TestRequireAPIKeyResourcePermission_UnmappedAction system:user 已注册但
// flyToMoon 不在 action 词汇集 → 403 fail-closed。
func TestRequireAPIKeyResourcePermission_UnmappedAction(t *testing.T) {
	w := runProbeWithScopes(t, "system:user", "flyToMoon", []string{"admin"})
	assert.Equal(t, 403, w.Code, "system:user.flyToMoon 未注册,即使 admin 也应 403")
	assert.Contains(t, w.Body.String(), "资源权限未定义")
}

// TestRequireAPIKeyResourcePermission_MissingScopes 不设置 c.Set("scopes") → 403。
func TestRequireAPIKeyResourcePermission_MissingScopes(t *testing.T) {
	// 不传 scopes → helper middleware 不会 c.Set("scopes")
	w := runProbeWithScopes(t, "system:user", "list", nil)
	assert.Equal(t, 403, w.Code, "scopes 缺失应 403")
	assert.Contains(t, w.Body.String(), "权限作用域不足", "缺失 scopes 应返回权限作用域不足")
}

// TestRequireAPIKeyResourcePermission_BadScopeType scopes 类型错误(非 []string)
// → 403 (防御性断言)。
func TestRequireAPIKeyResourcePermission_BadScopeType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		// 故意设置错误类型 — 模拟 buggy 调用方或被攻击
		c.Set("scopes", "admin") // string 而非 []string
		c.Next()
	})
	r.Use(RequireAPIKeyResourcePermission("system:user", "list"))
	r.GET("/probe", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/probe", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, 403, w.Code, "scopes 类型错误应 403")
	assert.Contains(t, w.Body.String(), "权限作用域格式错误")
}
