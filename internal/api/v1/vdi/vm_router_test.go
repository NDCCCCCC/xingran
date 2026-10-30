package vdi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/xingran-next/xingran-go-backend/internal/core"
	"github.com/xingran-next/xingran-go-backend/internal/core/db"
)

// setupOperateTestEngine 构造挂载 SetupVMRouter 的测试引擎（testCore 照抄
// rpa 包 router_public_test.go 的 smoke 模式：零值 DB，注册期不触 DB）。
func setupOperateTestEngine() (*gin.Engine, *core.Core) {
	gin.SetMode(gin.TestMode)
	testCore := &core.Core{
		CoreInfra:    &core.CoreInfra{DB: &db.Database{}},
		CoreServices: &core.CoreServices{},
	}
	engine := gin.New()
	group := engine.Group("/vdi/vms")
	SetupVMRouter(group, testCore)
	return engine, testCore
}

// TestSetupVMRouter_OperateRegistered 断言 POST /vdi/vms/operate 已注册
// （Phase 100 D-100-10 接线缺陷修复：handler/service 早已存在，仅缺生产注册）。
func TestSetupVMRouter_OperateRegistered(t *testing.T) {
	engine, _ := setupOperateTestEngine()

	var found bool
	for _, route := range engine.Routes() {
		if route.Method == http.MethodPost && route.Path == "/vdi/vms/operate" {
			found = true
			break
		}
	}
	assert.True(t, found, "POST /vdi/vms/operate should be registered, routes: %v", engine.Routes())

	// smoke：现有 18 条路由 + operate = 19（list/resource-groups/resources/
	// create/:id/:id/update/:id/delete/start/stop/restart/operate/:id/bind_user/
	// :id/unbind_user/:id/sync/sync-all/vtp-platforms/run-positions/storages/networks）
	assert.GreaterOrEqual(t, len(engine.Routes()), 19, "expected at least 19 routes, got %d", len(engine.Routes()))
}

// TestSetupVMRouter_OperateRequiresAuth 断言未认证请求被 401 拒绝——证明
// RequirePermissions 在注册链上（裸注册会直接进 handler 返回 binding 400，
// 该断言即可区分两种注册形态；getUserIDAsString 失败先于任何 core/DB 访问）。
func TestSetupVMRouter_OperateRequiresAuth(t *testing.T) {
	engine, _ := setupOperateTestEngine()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/vdi/vms/operate", bytes.NewBufferString(`{}`))
	engine.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "unauthenticated request should be rejected with 401, body: %s", w.Body.String())
}
