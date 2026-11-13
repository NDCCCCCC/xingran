package api

// restore_write_guard_test.go —— 整库恢复期写入阻断中间件单测（A8 / #87）。
//
// 覆盖任务书要求：
//   - IsRestoring=true 时写端点被阻断（503 + 明确文案）；
//   - IsRestoring=false 时放行；
//   - 放行 GET / 只读端点与 dbbackup 自身任务轮询端点；
//   - 标志复位后同一引擎自动放行（动态读取，不缓存状态）；
//   - RestoreWriteGuard 已挂在 SetupRouter 全局入口（防漏挂）。

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// restoreGuardEngine 构造一个挂了待测中间件 + 若干桩路由的最小 gin 引擎。
// restoring 指针可随时翻转，用于验证「逐请求动态读取」。
func restoreGuardEngine(restoring *bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	e := gin.New()
	e.Use(restoreWriteGuardWith(func() bool { return *restoring }))
	// /api/v1 之下注册桩路由，模拟真实路由形态（含 POST /list 读语义、
	// POST /:id/update 写语义、GET 任务轮询）。
	g := e.Group("/api/v1")
	g.POST("/system/users/list", func(c *gin.Context) { c.String(http.StatusOK, "ok-read") })
	g.POST("/system/users/:id/update", func(c *gin.Context) { c.String(http.StatusOK, "ok-write") })
	g.POST("/system/users/:id/delete", func(c *gin.Context) { c.String(http.StatusOK, "ok-delete") })
	g.GET("/system/users/:id", func(c *gin.Context) { c.String(http.StatusOK, "ok-get") })
	g.GET("/system/db-backup/tasks/:id", func(c *gin.Context) { c.String(http.StatusOK, "ok-poll") })
	g.POST("/system/db-backup/restore", func(c *gin.Context) { c.String(http.StatusOK, "ok-restore") })
	g.POST("/system/auth/login", func(c *gin.Context) { c.String(http.StatusOK, "ok-login") })
	// 用户自视图读端点（前缀放行）与角色授权写端点（必须阻断，防白名单豁免写）。
	g.POST("/system/my-menus/permissions", func(c *gin.Context) { c.String(http.StatusOK, "ok-my-perms") })
	g.POST("/system/roles/:id/permissions", func(c *gin.Context) { c.String(http.StatusOK, "ok-role-perms") })
	return e
}

func doGuardRequest(e *gin.Engine, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	e.ServeHTTP(w, req)
	return w
}

// TestRestoreWriteGuard_Off 空闲态：所有方法与路径一律放行（含写端点）。
func TestRestoreWriteGuard_Off(t *testing.T) {
	restoring := false
	e := restoreGuardEngine(&restoring)

	for _, tc := range []struct {
		method, path string
	}{
		{http.MethodPost, "/api/v1/system/users/list"},
		{http.MethodPost, "/api/v1/system/users/u-1/update"},
		{http.MethodPost, "/api/v1/system/users/u-1/delete"},
		{http.MethodGet, "/api/v1/system/users/u-1"},
		{http.MethodGet, "/api/v1/system/db-backup/tasks/t-1"},
		{http.MethodPost, "/api/v1/system/db-backup/restore"},
	} {
		w := doGuardRequest(e, tc.method, tc.path)
		assert.Equal(t, http.StatusOK, w.Code, "空闲态应放行 %s %s", tc.method, tc.path)
	}
}

// TestRestoreWriteGuard_On_BlocksWrites 维护中：写端点被 503 + 明确文案阻断，
// 且后续 handler 不执行（响应体不含桩路由的 ok-* 文本）。
func TestRestoreWriteGuard_On_BlocksWrites(t *testing.T) {
	restoring := true
	e := restoreGuardEngine(&restoring)

	for _, tc := range []struct {
		method, path string
	}{
		{http.MethodPost, "/api/v1/system/users/u-1/update"},
		{http.MethodPost, "/api/v1/system/users/u-1/delete"},
		{http.MethodPost, "/api/v1/system/db-backup/restore"},
		{http.MethodPut, "/api/v1/system/users/u-1/update"},
		{http.MethodPatch, "/api/v1/system/users/u-1/update"},
		{http.MethodDelete, "/api/v1/system/users/u-1"},
		// 角色授权写端点：末段 "permissions" 不得进末段白名单（防未来授权写端点被静默放行）。
		{http.MethodPost, "/api/v1/system/roles/r-1/permissions"},
	} {
		w := doGuardRequest(e, tc.method, tc.path)
		require.Equal(t, http.StatusServiceUnavailable, w.Code, "%s %s 应被阻断", tc.method, tc.path)
		var body map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		assert.Equal(t, float64(http.StatusServiceUnavailable), body["code"])
		assert.Equal(t, restoreGuardMessage, body["message"])
		assert.NotContains(t, w.Body.String(), "ok-write", "被阻断时 handler 不应执行")
	}
}

// TestRestoreWriteGuard_On_AllowsReads 维护中：GET / 只读端点 / dbbackup
// 任务轮询端点 / 认证登录仍放行（任务书明确要求的放行面）。
func TestRestoreWriteGuard_On_AllowsReads(t *testing.T) {
	restoring := true
	e := restoreGuardEngine(&restoring)

	for _, tc := range []struct {
		method, path string
	}{
		// 安全方法
		{http.MethodGet, "/api/v1/system/users/u-1"},
		{http.MethodHead, "/api/v1/system/users/u-1"},
		{http.MethodOptions, "/api/v1/system/users/u-1"},
		// POST /list 读语义（本项目 API 约定）
		{http.MethodPost, "/api/v1/system/users/list"},
		// dbbackup 自身任务轮询
		{http.MethodGet, "/api/v1/system/db-backup/tasks/t-1"},
		// 认证会话维持
		{http.MethodPost, "/api/v1/system/auth/login"},
		// 用户自视图读端点（精确前缀放行）
		{http.MethodPost, "/api/v1/system/my-menus/permissions"},
	} {
		w := doGuardRequest(e, tc.method, tc.path)
		assert.NotEqual(t, http.StatusServiceUnavailable, w.Code,
			"恢复期应放行 %s %s", tc.method, tc.path)
	}
}

// TestRestoreWriteGuard_FlagReset_AutoAllows 标志复位后自动放行：同一引擎
// 先 on（写被挡），再 off（写放行），验证每次请求动态读取、无状态缓存。
func TestRestoreWriteGuard_FlagReset_AutoAllows(t *testing.T) {
	restoring := true
	e := restoreGuardEngine(&restoring)

	blocked := doGuardRequest(e, http.MethodPost, "/api/v1/system/users/u-1/update")
	require.Equal(t, http.StatusServiceUnavailable, blocked.Code, "恢复标志置位时写端点必须阻断")

	// 模拟 runRestore defer restoring.Store(false) 复位。
	restoring = false
	allowed := doGuardRequest(e, http.MethodPost, "/api/v1/system/users/u-1/update")
	assert.Equal(t, http.StatusOK, allowed.Code, "标志复位后必须自动放行")
	assert.Contains(t, allowed.Body.String(), "ok-write")
}

// TestRestoreWriteGuard_MountedInSetupRouter 防漏挂守卫：SetupRouter 入口
// 必须全局挂载 RestoreWriteGuard()（源码文本扫描，不启动 gin、不连 DB）。
// 与 datascope_mount_test.go / strict_mount_count_test.go 同风格。
func TestRestoreWriteGuard_MountedInSetupRouter(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok, "无法定位测试文件路径")
	root := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	data, err := os.ReadFile(filepath.Join(root, "internal", "api", "router.go"))
	require.NoError(t, err)
	src := string(data)

	// 挂载必须在 SetupRouter 内、且位于 setupEncryptionMiddlewares 之前（早于解密）。
	setupIdx := strings.Index(src, "func SetupRouter(")
	require.GreaterOrEqual(t, setupIdx, 0, "router.go 中找不到 SetupRouter")
	guardIdx := strings.Index(src, "r.Use(RestoreWriteGuard())")
	encIdx := strings.Index(src, "setupEncryptionMiddlewares(r, core)")
	require.GreaterOrEqual(t, guardIdx, 0, "SetupRouter 未挂载 RestoreWriteGuard()")
	assert.Greater(t, guardIdx, setupIdx, "RestoreWriteGuard() 必须写在 SetupRouter 函数体内")
	assert.Less(t, guardIdx, encIdx, "恢复期阻断必须早于请求加解密中间件")
	assert.Equal(t, 1, strings.Count(src, "r.Use(RestoreWriteGuard())"), "全局挂载应恰好 1 处")
}
